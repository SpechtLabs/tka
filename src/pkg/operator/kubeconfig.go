package operator

import (
	"context"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SignInUser creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *KubeOperator) SignInUser(ctx context.Context, userName, role string, validUntil time.Time) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.SignInUser")
	defer span.End()

	c := t.mgr.GetClient()

	signin := newSignin(userName, role, validUntil)
	if err := c.Create(ctx, signin); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, "User already signed in",
				"please log out before attempting to sign in again",
				"if you have not yet signed in, or you credentials have expired but you are unable to sign in again, contact your kubernetes administrator")
		}

		return humane.Wrap(err, "Error signing in user", "see underlying error for more details")
	}

	signin.Status = *newSigninStatus(validUntil)
	if err := c.Status().Update(ctx, signin); err != nil {
		return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
	}

	return nil
}

var NotReadyYetError = humane.New("Not ready yet", "Please wait for the TKA signin to be ready")

func (t *KubeOperator) GetKubeconfig(ctx context.Context, userName string) (*api.Config, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.GetKubeconfig")
	defer span.End()

	c := t.mgr.GetClient()

	resName := client.ObjectKey{
		Name:      formatSigninObjectName(userName),
		Namespace: "tka-dev", // TODO: make this dynamic
	}

	var signIn v1alpha1.TkaSignin
	if err := c.Get(ctx, resName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, humane.Wrap(err, "User not signed in", "Please sign in before requesting kubeconfig")
		}
		return nil, humane.Wrap(err, "Failed to load sign-in request")
	}

	if signIn.Status.Provisioned == false {
		return nil, NotReadyYetError
	}

	// Generate token for ServiceAccount
	token, err := t.generateToken(ctx, &signIn)
	if err != nil {
		return nil, humane.Wrap(err, "Failed to generate token")
	}

	// Extract Kubernetes API server host from controller config
	restCfg := t.mgr.GetConfig()
	clusterName := "tka-cluster"
	contextName := "tka-context-" + userName
	userEntry := "tka-user-" + userName

	// Build kubeconfig
	return newKubeconfig(contextName, restCfg, token, clusterName, userEntry), nil
}

func (t *KubeOperator) LogOutUser(ctx context.Context, userName string) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.GetKubeconfig")
	defer span.End()

	c := t.mgr.GetClient()
	var signIn v1alpha1.TkaSignin

	// TODO(cedi): Make namespace dynamic
	signinName := types.NamespacedName{Name: formatSigninObjectName(userName), Namespace: "tka-dev"}
	if err := c.Get(ctx, signinName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("User not signed in", "Please sign in before requesting kubeconfig")
		}
		return humane.Wrap(err, "Failed to load sign-in request")
	}

	if err := c.Delete(ctx, &signIn); err != nil {
		return humane.Wrap(err, "Failed to remove sign-in request")
	}

	return nil
}
