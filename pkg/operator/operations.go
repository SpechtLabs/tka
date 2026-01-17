package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/client/k8s"
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

	// 2. Create ClusterRoleBinding
	if err := t.createOrUpdateClusterRoleBinding(ctx, signIn); err != nil {
		return err
	}

	c := t.mgr.GetClient()
	resName := client.ObjectKey{ //nolint:golint-sl // used in Get call and error message
		Name:      signIn.Name,
		Namespace: signIn.Namespace,
	}
	if err := c.Get(ctx, resName, signIn); err != nil {
		return humane.Wrap(err, "Failed to load sign-in request",
			"name: "+resName.Name,
			"namespace: "+resName.Namespace)
	}

	if signedInAt, ok := signIn.Annotations[k8s.LastAttemptedSignIn]; ok {
		signIn.Status.SignedInAt = signedInAt
	} else {
		signIn.Status.SignedInAt = time.Now().Format(time.RFC3339)
	}

	signedIn, e := time.Parse(time.RFC3339, signIn.Status.SignedInAt) //nolint:golint-sl // signedIn is used after this if block
	if e != nil {
		return humane.Wrap(e, "Failed to parse signedInAt", "ensure the timestamp is in RFC3339 format")
	}

	duration, e := time.ParseDuration(signIn.Spec.ValidityPeriod) //nolint:golint-sl // duration is used after this if block
	if e != nil {
		return humane.Wrap(e, "Failed to parse validityPeriod", "use a valid duration format like '1h', '30m', or '24h'")
	}

	validUntil := signedIn.Add(duration)
	signIn.Status.ValidUntil = validUntil.Format(time.RFC3339)

	signIn.Status.Provisioned = true
	if err := c.Status().Update(ctx, signIn); err != nil {
		return humane.Wrap(err, "Error updating signin status", "check Kubernetes API connectivity and RBAC permissions")
	}

	// Update Prometheus metrics for user sign-in
	userSignInsTotal.WithLabelValues(signIn.Spec.Role, signIn.Spec.Username).Inc()
	activeUserSessions.WithLabelValues(signIn.Spec.Role).Inc()

	return nil
}

func (t *KubeOperator) signOutUser(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	if err := t.deleteClusterRoleBinding(ctx, signIn); err != nil {
		return humane.Wrap(err, "failed to delete cluster role binding", "check Kubernetes RBAC permissions and cluster connectivity")
	}

	if err := t.deleteServiceAccount(ctx, signIn); err != nil {
		return humane.Wrap(err, "failed to delete service account", "check Kubernetes permissions and that the service account exists")
	}

	if err := t.client.DeleteSignIn(ctx, signIn.Spec.Username); err != nil {
		return humane.Wrap(err, "failed to delete user", "verify the user exists and the operator has delete permissions")
	}

	// Update Prometheus metrics for user sign-out
	activeUserSessions.WithLabelValues(signIn.Spec.Role).Dec()

	return nil
}

// createOrUpdateServiceAccount creates a new service account or updates an existing one with the given parameters
func (t *KubeOperator) createOrUpdateServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) (*corev1.ServiceAccount, humane.Error) {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	serviceAccount := k8s.NewServiceAccount(signIn)

	// Set SignIn as the owner of the ServiceAccount
	_ = ctrl.SetControllerReference(signIn, serviceAccount, scheme)

	if err := c.Create(ctx, serviceAccount); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to create service account for user %s", signIn.Spec.Username), "check Kubernetes permissions for creating service accounts in namespace "+signIn.Namespace)
		}

		// If the service account already exists, we'll just update it
		saName := types.NamespacedName{
			Name:      k8s.FormatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		}
		if err := c.Get(ctx, saName, serviceAccount); err != nil {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to get existing service account for user %s", signIn.Spec.Username), "verify the service account exists and you have read permissions")
		}

		if err := c.Update(ctx, serviceAccount); err != nil {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to update service account for user %s", signIn.Spec.Username), "check Kubernetes permissions for updating service accounts")
		}
	}

	return serviceAccount, nil
}

// createOrUpdateClusterRoleBinding creates or updates a ClusterRoleBinding for the specified user and role
func (t *KubeOperator) createOrUpdateClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()
	scheme := t.mgr.GetScheme()

	clusterRoleBinding := k8s.NewClusterRoleBinding(signIn)

	// Set SignIn as the owner of the ClusterRoleBinding
	_ = ctrl.SetControllerReference(signIn, clusterRoleBinding, scheme)

	if err := c.Create(ctx, clusterRoleBinding); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, fmt.Sprintf("Failed to create cluster role binding for user %s", signIn.Spec.Username), "check Kubernetes RBAC permissions for creating cluster role bindings")
		}

		// If the cluster role binding already exists, we'll just update it
		existingCRB := &rbacv1.ClusterRoleBinding{}
		crbName := types.NamespacedName{
			Name: k8s.GetClusterRoleBindingName(signIn),
		}
		if err := c.Get(ctx, crbName, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to get existing cluster role binding for user %s", signIn.Spec.Username), "verify the cluster role binding exists and you have read permissions")
		}

		// Update the validUntil annotation and role reference
		existingCRB.RoleRef = k8s.NewRoleRef(signIn)

		if err := c.Update(ctx, existingCRB); err != nil {
			return humane.Wrap(err, fmt.Sprintf("Failed to update cluster role binding for user %s", signIn.Spec.Username), "check Kubernetes permissions for updating cluster role bindings")
		}
	}

	return nil
}

func (t *KubeOperator) deleteClusterRoleBinding(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()

	var crb rbacv1.ClusterRoleBinding

	crbName := types.NamespacedName{Name: k8s.GetClusterRoleBindingName(signIn), Namespace: signIn.Namespace} //nolint:golint-sl // used in Get call
	if err := c.Get(ctx, crbName, &crb); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("Cluster role binding not found", "the cluster role binding may have been already deleted")
		}
		return humane.Wrap(err, "Failed to load cluster role binding", "check Kubernetes connectivity and RBAC read permissions")
	}

	if err := c.Delete(ctx, &crb); err != nil {
		return humane.Wrap(err, "Failed to remove cluster role binding", "check Kubernetes permissions for deleting cluster role bindings")
	}

	return nil
}

func (t *KubeOperator) deleteServiceAccount(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	c := t.mgr.GetClient()

	var sa corev1.ServiceAccount

	saName := types.NamespacedName{Name: k8s.FormatSigninObjectName(signIn.Spec.Username), Namespace: signIn.Namespace} //nolint:golint-sl // used in Get call
	if err := c.Get(ctx, saName, &sa); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("Service account not found", "the service account may have been already deleted")
		}
		return humane.Wrap(err, "Failed to load service account", "check Kubernetes connectivity and read permissions in namespace "+signIn.Namespace)
	}

	if err := c.Delete(ctx, &sa); err != nil {
		return humane.Wrap(err, "Failed to remove service account", "check Kubernetes permissions for deleting service accounts")
	}

	return nil
}
