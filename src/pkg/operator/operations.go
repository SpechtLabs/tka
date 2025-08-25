package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MinSigninValidity is the minimum validity period for a token in Kubernetes. This minimum period is enforced by the Kubernetes API.
const MinSigninValidity = 10 * time.Minute

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

	c := t.mgr.GetClient()
	resName := client.ObjectKey{
		Name:      signIn.Name,
		Namespace: signIn.Namespace,
	}
	if err := c.Get(ctx, resName, signIn); err != nil {
		otelzap.L().WithError(err).Error("failed to get tka signin", zap.String("name", resName.Name), zap.String("namespace", resName.Namespace))
		return humane.Wrap(err, "Failed to load sign-in request")
	}

	if signedInAt, ok := signIn.Annotations[LastAttemptedSignIn]; ok {
		signIn.Status.SignedInAt = signedInAt
	} else {
		signIn.Status.SignedInAt = time.Now().Format(time.RFC3339)
	}

	signedIn, e := time.Parse(time.RFC3339, signIn.Status.SignedInAt)
	if e != nil {
		return humane.Wrap(err, "Failed to parse signedInAt")
	}

	duration, e := time.ParseDuration(signIn.Spec.ValidityPeriod)
	if e != nil {
		return humane.Wrap(err, "Failed to parse validityPeriod")
	}

	validUntil := signedIn.Add(duration)
	signIn.Status.ValidUntil = validUntil.Format(time.RFC3339)

	signIn.Status.Provisioned = true
	if err := c.Status().Update(ctx, signIn); err != nil {
		return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
	}

	otelzap.L().InfoContext(ctx, "Successfully signed in user",
		zap.String("user", signIn.Spec.Username),
		zap.String("validity", signIn.Spec.ValidityPeriod),
		zap.String("role", signIn.Spec.Role),
	)
	return nil
}

// createOrUpdateServiceAccount creates a new service account or updates an existing one with the given parameters
func (t *KubeOperator) createOrUpdateServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) (*corev1.ServiceAccount, humane.Error) {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	serviceAccount := newServiceAccount(signIn)

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, serviceAccount, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := c.Create(ctx, serviceAccount); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to create service account for user %s", signIn.Spec.Username))
		}

		// If the service account already exists, we'll just update it
		saName := types.NamespacedName{
			Name:      formatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		}
		if err := c.Get(ctx, saName, serviceAccount); err != nil {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to get existing service account for user %s", signIn.Spec.Username))
		}

		if err := c.Update(ctx, serviceAccount); err != nil {
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
	validUntil, err := time.Parse(time.RFC3339, signIn.Status.ValidUntil)
	if err != nil {
		return "", humane.Wrap(err, "Failed to parse validUntil")
	}

	expirationSeconds := int64(time.Until(validUntil).Seconds())
	if expirationSeconds < int64(MinSigninValidity.Seconds()) {
		expirationSeconds = int64(MinSigninValidity.Seconds())
	}
	tokenRequest := newTokenRequest(expirationSeconds)

	tokenResponse, err := clientset.CoreV1().ServiceAccounts(signIn.Namespace).CreateToken(ctx, formatSigninObjectName(signIn.Spec.Username), tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", humane.Wrap(err, "Failed to create token for service account")
	}

	return tokenResponse.Status.Token, nil
}

// createOrUpdateClusterRoleBinding creates or updates a ClusterRoleBinding for the specified user and role
func (t *KubeOperator) createOrUpdateClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	clusterRoleBinding := newClusterRoleBinding(signIn)

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, clusterRoleBinding, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := c.Create(ctx, clusterRoleBinding); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, fmt.Sprintf("Failed to create cluster role binding for user %s", signIn.Spec.Username))
		}

		// If the cluster role binding already exists, we'll just update it
		existingCRB := &rbacv1.ClusterRoleBinding{}
		crbName := types.NamespacedName{
			Name: getClusterRoleBindingName(signIn),
		}
		if err := c.Get(ctx, crbName, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to get existing cluster role binding for user %s", signIn.Spec.Username))
		}

		// Update the validUntil annotation and role reference
		existingCRB.RoleRef = newRoleRef(signIn)

		if err := c.Update(ctx, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to update cluster role binding for user %s", signIn.Spec.Username))
		}
	}

	return nil
}
