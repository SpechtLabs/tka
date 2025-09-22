package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	c := t.mgr.GetClient()
	resName := client.ObjectKey{
		Name:      signIn.Name,
		Namespace: signIn.Namespace,
	}
	if err := c.Get(ctx, resName, signIn); err != nil {
		otelzap.L().WithError(err).Error("failed to get tka signin", zap.String("name", resName.Name), zap.String("namespace", resName.Namespace))
		return humane.Wrap(err, "Failed to load sign-in request")
	}

	if signedInAt, ok := signIn.Annotations[k8s.LastAttemptedSignIn]; ok {
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

	// Update Prometheus metrics for user sign-in
	userSignInsTotal.WithLabelValues(signIn.Spec.Role, signIn.Spec.Username).Inc()
	activeUserSessions.WithLabelValues(signIn.Spec.Role).Inc()

	return nil
}

func (t *KubeOperator) signOutUser(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	if err := t.deleteClusterRoleBinding(ctx, signIn); err != nil {
		return humane.Wrap(err, "failed to delete cluster role binding")
	}

	if err := t.deleteServiceAccount(ctx, signIn); err != nil {
		return humane.Wrap(err, "failed to delete service account")
	}

	if err := t.client.DeleteSignIn(ctx, signIn.Spec.Username); err != nil {
		return humane.Wrap(err, "failed to delete user")
	}

	otelzap.L().InfoContext(ctx, "Successfully signed out user",
		zap.String("user", signIn.Spec.Username),
		zap.String("role", signIn.Spec.Role),
	)

	// Update Prometheus metrics for user sign-out
	activeUserSessions.WithLabelValues(signIn.Spec.Role).Dec()

	return nil
}

// createOrUpdateServiceAccount creates a new service account or updates an existing one with the given parameters
func (t *KubeOperator) createOrUpdateServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) (*corev1.ServiceAccount, humane.Error) {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	serviceAccount := k8s.NewServiceAccount(signIn)

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
			Name:      k8s.FormatSigninObjectName(signIn.Spec.Username),
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

// createOrUpdateClusterRoleBinding creates or updates a ClusterRoleBinding for the specified user and role
func (t *KubeOperator) createOrUpdateClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	clusterRoleBinding := k8s.NewClusterRoleBinding(signIn)

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
			Name: k8s.GetClusterRoleBindingName(signIn),
		}
		if err := c.Get(ctx, crbName, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to get existing cluster role binding for user %s", signIn.Spec.Username))
		}

		// Update the validUntil annotation and role reference
		existingCRB.RoleRef = k8s.NewRoleRef(signIn)

		if err := c.Update(ctx, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to update cluster role binding for user %s", signIn.Spec.Username))
		}
	}

	return nil
}

func (t *KubeOperator) deleteClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()

	var crb rbacv1.ClusterRoleBinding

	crbName := types.NamespacedName{Name: k8s.GetClusterRoleBindingName(signIn), Namespace: signIn.Namespace}
	if err := c.Get(ctx, crbName, &crb); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("Cluster role binding not found", "Please make sure the cluster role binding exists and remove it manually")
		}
		return humane.Wrap(err, "Failed to load cluster role binding")
	}

	if err := c.Delete(ctx, &crb); err != nil {
		return humane.Wrap(err, "Failed to remove cluster role binding")
	}

	return nil
}

func (t *KubeOperator) deleteServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()

	var sa corev1.ServiceAccount

	saName := types.NamespacedName{Name: k8s.FormatSigninObjectName(signIn.Spec.Username), Namespace: signIn.Namespace}
	if err := c.Get(ctx, saName, &sa); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("Service account not found", "Please make sure the service account exists and remove it manually")
		}
		return humane.Wrap(err, "Failed to load service account")
	}

	if err := c.Delete(ctx, &sa); err != nil {
		return humane.Wrap(err, "Failed to remove service account")
	}

	return nil
}
