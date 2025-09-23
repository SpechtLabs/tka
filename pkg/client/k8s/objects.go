package k8s

import (
	"fmt"
	"time"

	"github.com/spechtlabs/tka/api/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

func FormatSigninObjectName(userName string) string {
	return fmt.Sprintf("%s%s", DefaultUserEntryPrefix, userName)
}

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

func NewServiceAccount(signIn *v1alpha1.TkaSignin) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      FormatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		},
	}
}

func NewKubeconfig(contextName string, restCfg *rest.Config, token string, clusterName string, userEntry string) *api.Config {
	return &api.Config{
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
}

// NewKubeconfigWithExternalCluster creates a kubeconfig with external cluster information.
// This is preferred when generating kubeconfigs for external clients from within a cluster.
func NewKubeconfigWithExternalCluster(contextName, token, clusterName, userEntry, serverURL string, caData []byte, insecure bool) *api.Config {
	return &api.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: contextName,
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   serverURL,
				CertificateAuthorityData: caData,
				InsecureSkipTLSVerify:    insecure,
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

func NewTokenRequest(expirationSeconds int64) *authenticationv1.TokenRequest {
	return &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
			// Audiences:         []string{"https://kubernetes.default.svc.cluster.local"}, // TODO(cedi): implement properly
		},
	}
}

func NewRoleRef(signIn *v1alpha1.TkaSignin) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     signIn.Spec.Role,
	}
}

func GetClusterRoleBindingName(signIn *v1alpha1.TkaSignin) string {
	username := FormatSigninObjectName(signIn.Spec.Username)
	return fmt.Sprintf("%s-binding", username)
}

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
