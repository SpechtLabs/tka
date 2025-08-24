package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SignInUser creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *KubeOperator) SignInUser(ctx context.Context, userName, role string, validPeriod time.Duration) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.SignInUser")
	defer span.End()

	if validPeriod < 10*time.Minute {
		return humane.New("`period` may not specify a duration less than 10 minutesD",
			fmt.Sprintf("Specify a period greater than 10 minutes in your api ACL for user %s", userName),
		)
	}

	c := t.mgr.GetClient()

	signin := newSignin(userName, role, validPeriod, t.opts.Namespace)
	if err := c.Create(ctx, signin); err != nil && k8serrors.IsAlreadyExists(err) {
		otelzap.L().DebugContext(ctx, "User already signed in",
			zap.String("user", userName),
			zap.String("validity", validPeriod.String()),
			zap.String("role", role))

		existing, err := t.GetSignInUser(ctx, userName)
		if err != nil {
			return humane.Wrap(err, "Failed to load existing sign-in request")
		}

		existing.Spec.ValidityPeriod = signin.Spec.ValidityPeriod
		existing.Spec.Role = signin.Spec.Role
		existing.Annotations = signin.Annotations
		if err := c.Update(ctx, existing); err != nil {
			return humane.Wrap(err, "Failed to update existing sign-in request")
		}
	} else if err != nil {
		return humane.Wrap(err, "Error signing in user", "see underlying error for more details")
	} else {
		if err := c.Status().Update(ctx, signin); err != nil {
			return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
		}
	}

	return nil
}

var NotReadyYetError = humane.New("Not ready yet", "Please wait for the TKA signin to be ready")

// GetSignInUser creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *KubeOperator) GetSignInUser(ctx context.Context, userName string) (*v1alpha1.TkaSignin, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.GetSignInUser")
	defer span.End()

	c := t.mgr.GetClient()

	resName := client.ObjectKey{
		Name:      formatSigninObjectName(userName),
		Namespace: t.opts.Namespace,
	}

	var signIn v1alpha1.TkaSignin
	if err := c.Get(ctx, resName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, humane.Wrap(err, "User not signed in", "Please sign in before requesting")
		}
		return nil, humane.Wrap(err, "Failed to load sign-in request")
	}

	return &signIn, nil
}

func (t *KubeOperator) GetKubeconfig(ctx context.Context, userName string) (*api.Config, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.GetKubeconfig")
	defer span.End()

	c := t.mgr.GetClient()

	resName := client.ObjectKey{
		Name:      formatSigninObjectName(userName),
		Namespace: t.opts.Namespace,
	}

	var signIn v1alpha1.TkaSignin
	if err := c.Get(ctx, resName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, humane.Wrap(err, "User not signed in", "Please sign in before requesting kubeconfig")
		}
		return nil, humane.Wrap(err, "Failed to load sign-in request")
	}

	if !signIn.Status.Provisioned {
		return nil, NotReadyYetError
	}

	// Generate token for ServiceAccount
	token, err := t.generateToken(ctx, &signIn)
	if err != nil {
		return nil, humane.Wrap(err, "Failed to generate token")
	}

	// Extract Kubernetes API tailscale host from controller config
	restCfg := t.mgr.GetConfig()
	clusterName := t.opts.ClusterName
	contextName := t.opts.ContextPrefix + userName
	userEntry := t.opts.UserPrefix + userName

	// Build kubeconfig
	return newKubeconfig(contextName, restCfg, token, clusterName, userEntry), nil
}

func (t *KubeOperator) LogOutUser(ctx context.Context, userName string) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.GetKubeconfig")
	defer span.End()

	c := t.mgr.GetClient()
	var signIn v1alpha1.TkaSignin

	signinName := types.NamespacedName{Name: formatSigninObjectName(userName), Namespace: t.opts.Namespace}
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
