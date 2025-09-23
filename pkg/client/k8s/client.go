package k8s

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/internal/utils"
	"github.com/spechtlabs/tka/pkg/service/auth/models"
	"go.uber.org/zap"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tkaClient struct {
	client      client.Client
	tracer      trace.Tracer
	opts        ClientOptions
	clusterInfo *models.TkaClusterInfo
}

func NewTkaClient(client client.Client, clusterInfo *models.TkaClusterInfo, opts ClientOptions) TkaClient {
	return &tkaClient{
		client:      client,
		clusterInfo: clusterInfo,
		tracer:      otel.Tracer("tka_k8s_client"),
		opts:        opts,
	}
}

// NewSignIn creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *tkaClient) NewSignIn(ctx context.Context, userName, role string, validPeriod time.Duration) humane.Error {
	ctx, span := t.tracer.Start(ctx, "TkaClient.NewUser")
	defer span.End()

	if validPeriod < MinSigninValidity {
		return humane.New("`period` may not specify a duration less than 10 minutes",
			fmt.Sprintf("Specify a period greater than 10 minutes in your api ACL for user %s", userName),
		)
	}

	signin := NewSignin(userName, role, validPeriod, t.opts.Namespace)
	if err := t.client.Create(ctx, signin); err != nil && k8serrors.IsAlreadyExists(err) {
		otelzap.L().DebugContext(ctx, "User already signed in",
			zap.String("user", userName),
			zap.String("validity", validPeriod.String()),
			zap.String("role", role))

		existing, err := t.GetSignIn(ctx, userName)
		if err != nil {
			return humane.Wrap(err, "Failed to load existing sign-in request")
		}

		existing.Spec.ValidityPeriod = signin.Spec.ValidityPeriod
		existing.Spec.Role = signin.Spec.Role
		existing.Annotations = signin.Annotations
		if err := t.client.Update(ctx, existing); err != nil {
			return humane.Wrap(err, "Failed to update existing sign-in request")
		}
	} else if err != nil {
		return humane.Wrap(err, "Error signing in user", "see underlying error for more details")
	} else {
		if err := t.client.Status().Update(ctx, signin); err != nil {
			return humane.Wrap(err, "Error updating signin status", "see underlying error for more details")
		}
	}

	return nil
}

// GetSignIn creates necessary Kubernetes resources to grant a user temporary access with a specific role
func (t *tkaClient) GetSignIn(ctx context.Context, userName string) (*v1alpha1.TkaSignin, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "TkaClient.GetSignIn")
	defer span.End()

	resName := client.ObjectKey{
		Name:      FormatSigninObjectName(userName),
		Namespace: t.opts.Namespace,
	}

	var signIn v1alpha1.TkaSignin
	if err := t.client.Get(ctx, resName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, humane.Wrap(err, "User not signed in", "Please sign in before requesting")
		}
		return nil, humane.Wrap(err, "Failed to load sign-in request")
	}

	return &signIn, nil
}

func (t *tkaClient) GetKubeconfig(ctx context.Context, userName string) (*api.Config, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "TkaClient.GetKubeconfig")
	defer span.End()

	resName := client.ObjectKey{
		Name:      FormatSigninObjectName(userName),
		Namespace: t.opts.Namespace,
	}

	var signIn v1alpha1.TkaSignin
	if err := t.client.Get(ctx, resName, &signIn); err != nil {
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

	clusterName := t.opts.ClusterName
	contextName := t.opts.ContextPrefix + userName
	userEntry := t.opts.UserPrefix + userName

	// Use discovered external cluster information for clients
	return NewKubeconfig(
		contextName,
		t.clusterInfo,
		token,
		clusterName,
		userEntry,
	), nil
}

func (t *tkaClient) DeleteSignIn(ctx context.Context, userName string) humane.Error {
	ctx, span := t.tracer.Start(ctx, "TkaClient.DeleteSignIn")
	defer span.End()

	var signIn v1alpha1.TkaSignin

	signinName := types.NamespacedName{Name: FormatSigninObjectName(userName), Namespace: t.opts.Namespace}
	if err := t.client.Get(ctx, signinName, &signIn); err != nil {
		if k8serrors.IsNotFound(err) {
			return humane.New("User not signed in", "Please sign in before requesting kubeconfig")
		}
		return humane.Wrap(err, "Failed to load sign-in request")
	}

	if err := t.client.Delete(ctx, &signIn); err != nil {
		return humane.Wrap(err, "Failed to remove sign-in request")
	}

	return nil
}

func (t *tkaClient) GetStatus(ctx context.Context, username string) (*SignInInfo, humane.Error) {
	signIn, err := t.GetSignIn(ctx, username)
	if err != nil {
		return nil, err
	}

	return &SignInInfo{
		Username:       signIn.Spec.Username,
		Role:           signIn.Spec.Role,
		ValidityPeriod: signIn.Spec.ValidityPeriod,
		ValidUntil:     signIn.Status.ValidUntil,
		Provisioned:    signIn.Status.Provisioned,
	}, nil
}

// generateToken creates a token for the service account in Kubernetes versions >= 1.30 do no longer
// automatically include a token for new ServiceAccounts, thus we have to manually create one,
// so we can use it when assembling the kubeconfig for the user
func (t *tkaClient) generateToken(ctx context.Context, signIn *v1alpha1.TkaSignin) (string, humane.Error) {
	// Check if Kubernetes version is at least 1.30
	isSupported, herr := utils.IsK8sVerAtLeast(1, 30)
	if herr != nil {
		return "", herr
	}

	if !isSupported {
		// Token generation not supported in this Kubernetes version
		return "", nil
	}

	config, err := ctrl.GetConfig()
	if err != nil {
		return "", humane.Wrap(err, "Failed to get Kubernetes config")
	}

	// For Kubernetes >= 1.30, we need to create a token request
	clientset, err := kubernetes.NewForConfig(config)
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
	tokenRequest := NewTokenRequest(expirationSeconds)

	tokenResponse, err := clientset.CoreV1().ServiceAccounts(signIn.Namespace).CreateToken(ctx, FormatSigninObjectName(signIn.Spec.Username), tokenRequest, metav1.CreateOptions{})
	if err != nil {
		return "", humane.Wrap(err, "Failed to create token for service account")
	}

	return tokenResponse.Status.Token, nil
}
