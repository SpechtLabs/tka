package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	"go.uber.org/zap"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	rbacv1 "k8s.io/api/rbac/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/spechtlabs/go-otel-utils/otelzap"
)

// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tka.specht-labs.de,resources=TkaSignin/finalizers,verbs=update

func (t *KubeOperator) Reconcile(ctx context.Context, req ctrl.Request) (reconcile.Result, error) {
	startTime := time.Now()
	defer func() {
		reconcilerDuration.WithLabelValues("user", req.Name, req.Namespace).Observe(float64(time.Since(startTime).Microseconds()))
	}()

	ctx, span := t.tracer.Start(ctx, "KubeOperator.Reconcile")
	defer span.End()

	client := t.mgr.GetClient()

	signIn := &v1alpha1.TkaSignin{}
	if err := client.Get(ctx, req.NamespacedName, signIn); err != nil || signIn == nil {
		if k8serrors.IsNotFound(err) {
			otelzap.L().Info("signin deleted")
			return reconcile.Result{}, nil
		}

		otelzap.L().WithError(err).Error("failed to get tka signin")
		return reconcile.Result{}, err
	}

	if signIn.Status.Provisioned == false {
		if err := t.signInUser(ctx, signIn); err != nil {
			otelzap.L().WithError(err).Error("failed to sign in user")
			return reconcile.Result{}, fmt.Errorf("%s", err.Display())
		}
	}

	return reconcile.Result{}, nil
}

func (t *KubeOperator) signInUser(ctx context.Context, signIn *v1alpha1.TkaSignin) humane.Error {
	// 1. Create Service Account
	_, err := t.createOrUpdateServiceAccount(ctx, signIn)
	if err != nil {
		return err
	}

	// TODO(cedi): later in GET /kubeconfig re-use this snippet
	//	// 2 & 3. Check Kubernetes version and generate token if needed
	//	token, err := t.generateToken(ctx, signIn)
	//	if err != nil {
	//		return err
	//	}

	//	// Store or return token as needed
	//	if token != "" {
	//		otelzap.L().DebugContext(ctx, "Generated token",
	//			zap.String("user", signIn.Spec.Username),
	//			zap.String("valid_until", signIn.Spec.ValidUntil))
	//	}

	// 4. Create ClusterRoleBinding
	if err := t.createOrUpdateClusterRoleBinding(ctx, signIn); err != nil {
		return err
	}

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

	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
			Annotations: map[string]string{
				"validUntil": signIn.Spec.ValidUntil,
			},
		},
	}

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, serviceAccount, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := client.Create(ctx, serviceAccount); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, humane.Wrap(err, fmt.Sprintf("Failed to create service account for user %s", signIn.Spec.Username))
		}

		// If the service account already exists, we'll just update it
		if err := client.Get(ctx, types.NamespacedName{Name: signIn.Spec.Username, Namespace: signIn.Namespace}, serviceAccount); err != nil {
			return nil, humane.Wrap(err, "Failed to get existing service account for user %s", signIn.Spec.Username)
		}

		// Update the validUntil annotation
		if serviceAccount.Annotations == nil {
			serviceAccount.Annotations = make(map[string]string)
		}
		serviceAccount.Annotations["validUntil"] = signIn.Spec.ValidUntil

		if err := client.Update(ctx, serviceAccount); err != nil {
			return nil, humane.Wrap(err, "Failed to update service account for user %s", signIn.Spec.Username)
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
	tokenRequest := &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
			// Audiences:         []string{"https://kubernetes.default.svc.cluster.local"}, // TODO(cedi): implement properly
		},
	}

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

	username := formatSigninObjectName(signIn.Spec.Username)

	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-binding", username),
			Namespace: signIn.Namespace,
			Annotations: map[string]string{
				"validUntil": signIn.Spec.ValidUntil,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      username,
				Namespace: "tka-dev", // TODO(cedi): make dynamic
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     signIn.Spec.Role,
		},
	}

	// Set Redirect instance as the owner and api
	if err := ctrl.SetControllerReference(signIn, clusterRoleBinding, scheme); err != nil {
		otelzap.L().WithError(err).Error("Failed to set controller reference")
	}

	if err := client.Create(ctx, clusterRoleBinding); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return humane.Wrap(err, "Failed to create cluster role binding for user %s", signIn.Spec.Username)
		}

		// If the cluster role binding already exists, we'll just update it
		existingCRB := &rbacv1.ClusterRoleBinding{}
		if err := client.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-binding", signIn.Spec.Username)}, existingCRB); err != nil {
			return humane.Wrap(err, "Failed to get existing cluster role binding for user %s", signIn.Spec.Username)
		}

		// Update the validUntil annotation and role reference
		if existingCRB.Annotations == nil {
			existingCRB.Annotations = make(map[string]string)
		}
		existingCRB.Annotations["validUntil"] = signIn.Spec.ValidUntil
		existingCRB.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     signIn.Spec.Role,
		}

		if err := client.Update(ctx, existingCRB); err != nil {
			return humane.Wrap(err, "Failed to update cluster role binding for user %s", signIn.Spec.Username)
		}
	}

	return nil
}
