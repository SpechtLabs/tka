package operator

import (
	"fmt"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/api/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
)

func formatSigninObjectName(userName string) string {
	return fmt.Sprintf("%s%s", DefaultUserEntryPrefix, userName)
}

func newSignin(userName, role string, validPeriod time.Duration, namespace string) *v1alpha1.TkaSignin {
	now := time.Now()
	return &v1alpha1.TkaSignin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(userName),
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

func newServiceAccount(signIn *v1alpha1.TkaSignin) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      formatSigninObjectName(signIn.Spec.Username),
			Namespace: signIn.Namespace,
		},
	}
}

func newKubeconfig(contextName string, restCfg *rest.Config, token string, clusterName string, userEntry string) *api.Config {
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

func newTokenRequest(expirationSeconds int64) *authenticationv1.TokenRequest {
	return &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{
			ExpirationSeconds: &expirationSeconds,
			// Audiences:         []string{"https://kubernetes.default.svc.cluster.local"}, // TODO(cedi): implement properly
		},
	}
}

func newRoleRef(signIn *v1alpha1.TkaSignin) rbacv1.RoleRef {
	return rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     signIn.Spec.Role,
	}
}

func getClusterRoleBindingName(signIn *v1alpha1.TkaSignin) string {
	username := formatSigninObjectName(signIn.Spec.Username)
	return fmt.Sprintf("%s-binding", username)
}

func newClusterRoleBinding(signIn *v1alpha1.TkaSignin) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getClusterRoleBindingName(signIn),
			Namespace: signIn.Namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      formatSigninObjectName(signIn.Spec.Username),
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
