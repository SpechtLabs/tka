package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (t *KubeOperator) signInUser(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	// 1. Create Service Account
	_, err := t.createOrUpdateServiceAccount(ctx, signIn)
	if err != nil {
		return err
	}

	// 4. Create ClusterRoleBinding
	if err := t.createOrUpdateClusterRoleBinding(ctx, signIn); err != nil {
		return err
	}

	time.Sleep(4 * time.Second)

	signIn.Status.Provisioned = true
	client := t.mgr.GetClient()
	if err := client.Status().Update(ctx, signIn); err != nil {
		return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
	}

	otelzap.L().InfoContext(ctx, "Successfully signed in user",
		zap.String("user", signIn.Spec.Username),
		zap.String("valid_until", signIn.Spec.ValidUntil),
		zap.String("role", signIn.Spec.Role),
	)
	return nil
}

// createOrUpdateServiceAccount creates a new service account or updates an existing one with the given parameters
func (t *KubeOperator) createOrUpdateServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) (*corev1.ServiceAccount, humane.Error) {
	client := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	serviceAccount := newServiceAccount(signIn)

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, serviceAccount, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := client.Create(ctx, serviceAccount); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to create service account for user %s", signIn.Spec.Username))
		}

		// If the service account already exists, we'll just update it
		saName := types.NamespacedName{
			Name:      formatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		}
		if err := client.Get(ctx, saName, serviceAccount); err != nil {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to get existing service account for user %s", signIn.Spec.Username))
		}

		// Update the validUntil annotation
		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = make(map[string]string)
		}
		serviceAccount.Annotations[ValidUntilAnnotation] = signIn.Spec.ValidUntil

		if err := client.Update(ctx, serviceAccount); err != nil {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to update service account for user %s", signIn.Spec.Username))
		}
	}

	return serviceAccount, nil
}

// generateToken creates a token for the service account in Kubernetes versions >= 1.30 do no longer
// automatically include a token for new ServiceAccounts, thus we have to manually create one,
// so we can use it when assembling the kubeconfig for the user
func (t *KubeOperator) generateToken(ctx context.Context, signIn *v1alpha1.TkaSignin) (string, humane.Error) {
	// Check if Kubernetes version is at least 1.30
	isSupported, herr := t.isK8sVerAtLeast(1, 30)
	if herr != nil {
		return "", herr
	}

	if !isSupported {
		// Token generation not supported in this Kubernetes version
		return "", nil
	}

	// For Kubernetes >= 1.30, we need to create a token request
	clientset, err := kubernetes.NewForConfig(t.mgr.GetConfig())
	if err != nil {
		return "", humane.Wrap(err, "Failed to create Kubernetes clientset")
	}

	// Create a token request with expiration time
	validUntil, err := time.Parse(time.RFC3339, signIn.Spec.ValidUntil)
	if err != nil {
		return "", humane.Wrap(err, "Failed to parse validUntil")
	}

	expirationSeconds := int64(time.Until(validUntil).Seconds())
	tokenRequest := newTokenRequest(expirationSeconds)

	tokenResponse, err := clientset.CoreV1().ServiceAccounts(signIn.Namespace).CreateToken(ctx, formatSigninObjectName(signIn.Spec.Username), tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", humane.Wrap(err, "Failed to create token for service account")
	}

	return tokenResponse.Status.Token, nil
}

// createOrUpdateClusterRoleBinding creates or updates a ClusterRoleBinding for the specified user and role
func (t *KubeOperator) createOrUpdateClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	client := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	clusterRoleBinding := newClusterRoleBinding(signIn)

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, clusterRoleBinding, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := client.Create(ctx, clusterRoleBinding); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, fmt.Sprintf("Failed to create cluster role binding for user %s", signIn.Spec.Username))
		}

		// If the cluster role binding already exists, we'll just update it
		existingCRB := &rbacv1.ClusterRoleBinding{}
		crbName := types.NamespacedName{
			Name: getClusterRoleBindingName(signIn),
		}
		if err := client.Get(ctx, crbName, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to get existing cluster role binding for user %s", signIn.Spec.Username))
		}

		// Update the validUntil annotation and role reference
		if existingCRB.Annotations == nil {
			existingCRB.Annotations = make(map[string]string)
		}
		existingCRB.Annotations[ValidUntilAnnotation] = signIn.Spec.ValidUntil
		existingCRB.RoleRef = newRoleRef(signIn)

		if err := client.Update(ctx, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to update cluster role binding for user %s", signIn.Spec.Username))
		}
	}

	return nil
}
