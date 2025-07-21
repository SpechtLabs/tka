package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func formatSigninObjectName(userName string) string {
	return fmt.Sprintf("tka-user-%s", userName)
}

// SignInUser creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *KubeOperator) SignInUser(ctx context.Context, userName, role string, validUntil time.Time) humane.Error {
	ctx, span := t.tracer.Start(ctx, "KubeOperator.SignInUser")
	defer span.End()

	now := time.Now()
	period := validUntil.Sub(now)

	signin := &v1alpha1.TkaSignin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(userName),
			Namespace: "tka-dev", // TODO(cedi): make this dynamic...
		},
		Spec: v1alpha1.TkaSigninSpec{
			Username:   userName,
			Role:       role,
			ValidUntil: validUntil.Format(time.RFC3339),
		},
	}

	c := t.mgr.GetClient()
	if err := c.Create(ctx, signin); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, "User already signed in",
				"please log out before attempting to sign in again",
				"if you have not yet signed in, or you credentials have expired but you are unable to sign in again, contact your kubernetes administrator")
		}

		return humane.Wrap(err, "Error signing in user", "see underlying error for more details")
	}

	signin.Status = v1alpha1.TkaSigninStatus{
		Provisioned:    false,
		ValidityPeriod: period.String(),
		SignedInAt:     now.Format(time.RFC3339),
	}

	if err := c.Status().Update(ctx, signin); err != nil {
		return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
	}

	return nil
}

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
			return nil, humane.New("User not signed in", "Please sign in before requesting kubeconfig")
		}
		return nil, humane.Wrap(err, "Failed to load sign-in request")
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
	kubeCfg := &api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: contextName,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   restCfg.Host,
				CertificateAuthorityData: restCfg.CAData,
				InsecureSkipTLSVerify:    restCfg.Insecure,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userEntry: {
				Token: token,
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userEntry,
			},
		},
	}

	return kubeCfg, nil
}
