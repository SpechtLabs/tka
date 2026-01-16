package k8s

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/pkg/service/models"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
)

// FormatSigninObjectName generates the Kubernetes object name for a user's sign-in resource.
func FormatSigninObjectName(userName string) string {
	return fmt.Sprintf("%s%s", DefaultUserEntryPrefix, userName)
}

// NewSignin creates a new TkaSignin custom resource for the given user, role, and validity period.
func NewSignin(userName, role string, validPeriod time.Duration, namespace string) *v1alpha1.TkaSignin {
	now := time.Now()
	return &v1alpha1.TkaSignin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatSigninObjectName(userName),
			Namespace: namespace,
			Annotations: map[string]string{
				LastAttemptedSignIn: now.Format(time.RFC3339),
				SignInValidUntil:    now.Add(validPeriod).Format(time.RFC3339),
			},
		},
		Spec: v1alpha1.TkaSigninSpec{
			Username:       userName,
			Role:           role,
			ValidityPeriod: validPeriod.String(),
		},
		Status: v1alpha1.TkaSigninStatus{
			Provisioned: false,
			ValidUntil:  "",
			SignedInAt:  "",
		},
	}
}

// NewServiceAccount creates a new Kubernetes ServiceAccount for the given TkaSignin resource.
func NewServiceAccount(signIn *v1alpha1.TkaSignin) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		},
	}
}

// NewKubeconfig creates a kubeconfig for accessing the cluster with the given credentials.
func NewKubeconfig(contextName string, clusterInfo *models.TkaClusterInfo, token string, clusterName string, userEntry string) *api.Config {
	caData, herr := base64.StdEncoding.DecodeString(clusterInfo.CAData)
	if herr != nil {
		otelzap.L().WithError(herr).Fatal("failed to decode CA data")
	}

	return &api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: contextName,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   clusterInfo.ServerURL,
				CertificateAuthorityData: caData,
				InsecureSkipTLSVerify:    clusterInfo.InsecureSkipTLSVerify,
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
}

// NewTokenRequest creates a Kubernetes TokenRequest with the specified expiration time.
func NewTokenRequest(expirationSeconds int64) *authenticationv1.TokenRequest {
	return &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
		},
	}
}

// NewRoleRef creates a RoleRef pointing to the ClusterRole specified in the TkaSignin.
func NewRoleRef(signIn *v1alpha1.TkaSignin) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     signIn.Spec.Role,
	}
}

// GetClusterRoleBindingName returns the name of the ClusterRoleBinding for a TkaSignin.
func GetClusterRoleBindingName(signIn *v1alpha1.TkaSignin) string {
	username := FormatSigninObjectName(signIn.Spec.Username)
	return fmt.Sprintf("%s-binding", username)
}

// NewClusterRoleBinding creates a ClusterRoleBinding that grants the user the specified role.
func NewClusterRoleBinding(signIn *v1alpha1.TkaSignin) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetClusterRoleBindingName(signIn),
			Namespace: signIn.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      FormatSigninObjectName(signIn.Spec.Username),
				Namespace: signIn.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     signIn.Spec.Role,
		},
	}
}
