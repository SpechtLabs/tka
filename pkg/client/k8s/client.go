package k8s

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/api/v1alpha1"
	"github.com/spechtlabs/tka/internal/utils"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type tkaClient struct {
	client client.Client
	tracer trace.Tracer
	opts   ClientOptions
	config *rest.Config
}

func NewTkaClient(client client.Client, config *rest.Config, opts ClientOptions) TkaClient {
	return &tkaClient{
		client: client,
		config: config,
		tracer: otel.Tracer("tka_k8s_client"),
		opts:   opts,
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

	// Try to discover external cluster information automatically
	externalInfo, err := t.discoverExternalClusterInfo(ctx)
	if err != nil {
		otelzap.L().WarnContext(ctx, "Failed to discover external cluster info, using internal config", zap.Error(err.Cause()))
		// Fallback to internal cluster configuration
		return NewKubeconfig(contextName, t.config, token, clusterName, userEntry), nil
	}

	// Use discovered external cluster information for clients
	return NewKubeconfigWithExternalCluster(
		contextName,
		token,
		clusterName,
		userEntry,
		externalInfo.ServerURL,
		externalInfo.CAData,
		t.config.Insecure,
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

// ExternalClusterInfo contains the public API server endpoint and CA certificate
// for external clients to access the cluster.
type ExternalClusterInfo struct {
	ServerURL string
	CAData    []byte
}

// discoverExternalClusterInfo attempts to discover the public API server endpoint
// and CA certificate that external clients should use to connect to the cluster.
func (t *tkaClient) discoverExternalClusterInfo(ctx context.Context) (*ExternalClusterInfo, humane.Error) {
	ctx, span := t.tracer.Start(ctx, "TkaClient.discoverExternalClusterInfo")
	defer span.End()

	// Strategy 1: cluster-info ConfigMap
	serverURL, err := t.getPublicEndpointFromClusterInfo(ctx)
	if err != nil || serverURL == "" {
		if err != nil {
			otelzap.L().DebugContext(ctx, "cluster-info endpoint discovery failed", zap.Error(err.Cause()))
		}
		// Strategy 2: Infer from API server cert SANs
		if inferred, ierr := t.inferPublicEndpointFromCert(ctx); ierr == nil && inferred != "" {
			serverURL = inferred
		} else {
			// Strategy 3: Fall back to internal host
			serverURL = t.config.Host
		}
	}

	// Get the CA certificate - try multiple sources
	caData, err := t.getExternalCAData(ctx)
	if err != nil {
		otelzap.L().WarnContext(ctx, "Failed to get external CA data, falling back to internal CA", zap.Error(err.Cause()))
		// Fallback to the internal config CA
		caData = t.config.CAData
	}

	return &ExternalClusterInfo{
		ServerURL: serverURL,
		CAData:    caData,
	}, nil
}

// getPublicEndpointFromClusterInfo retrieves the public API server endpoint
// from the cluster-info ConfigMap in the kube-public namespace.
func (t *tkaClient) getPublicEndpointFromClusterInfo(ctx context.Context) (string, humane.Error) {
	var configMap corev1.ConfigMap

	// The cluster-info ConfigMap is typically in the kube-public namespace
	if err := t.client.Get(ctx, client.ObjectKey{
		Name:      "cluster-info",
		Namespace: "kube-public",
	}, &configMap); err != nil {
		if k8serrors.IsNotFound(err) {
			return "", humane.New("cluster-info ConfigMap not found", "The cluster may not expose public endpoint information via cluster-info ConfigMap")
		}
		return "", humane.Wrap(err, "Failed to get cluster-info ConfigMap")
	}

	// The kubeconfig is typically stored in the "kubeconfig" key
	kubeconfigData, exists := configMap.Data["kubeconfig"]
	if !exists {
		return "", humane.New("kubeconfig key not found in cluster-info ConfigMap", "The cluster-info ConfigMap does not contain the expected kubeconfig data")
	}

	// Parse the kubeconfig to extract the server URL
	config, err := clientcmd.Load([]byte(kubeconfigData))
	if err != nil {
		return "", humane.Wrap(err, "Failed to parse kubeconfig from cluster-info")
	}

	// Get the first cluster's server URL
	for _, cluster := range config.Clusters {
		if cluster.Server != "" {
			return cluster.Server, nil
		}
	}

	return "", humane.New("No server URL found in cluster-info kubeconfig", "The kubeconfig in cluster-info does not contain a valid server URL")
}

// getExternalCAData attempts to get the CA certificate data for external access.
// It tries multiple sources in order of preference.
func (t *tkaClient) getExternalCAData(ctx context.Context) ([]byte, humane.Error) {
	// Method 1: Try to get from cluster-info ConfigMap
	if caData, err := t.getCAFromClusterInfo(ctx); err == nil {
		return caData, nil
	}

	// Method 2: Try to read from the service account mount point (if running in-cluster)
	if caData, err := t.getCAFromServiceAccount(); err == nil {
		return caData, nil
	}

	// Method 3: Fall back to the internal config CA
	return t.config.CAData, nil
}

// getCAFromClusterInfo gets the CA certificate from the cluster-info ConfigMap.
func (t *tkaClient) getCAFromClusterInfo(ctx context.Context) ([]byte, humane.Error) {
	var configMap corev1.ConfigMap

	if err := t.client.Get(ctx, client.ObjectKey{
		Name:      "cluster-info",
		Namespace: "kube-public",
	}, &configMap); err != nil {
		return nil, humane.Wrap(err, "Failed to get cluster-info ConfigMap for CA")
	}

	kubeconfigData, exists := configMap.Data["kubeconfig"]
	if !exists {
		return nil, humane.New("kubeconfig key not found in cluster-info ConfigMap")
	}

	config, err := clientcmd.Load([]byte(kubeconfigData))
	if err != nil {
		return nil, humane.Wrap(err, "Failed to parse kubeconfig from cluster-info for CA")
	}

	// Get the first cluster's CA data
	for _, cluster := range config.Clusters {
		if len(cluster.CertificateAuthorityData) > 0 {
			return cluster.CertificateAuthorityData, nil
		}
	}

	return nil, humane.New("No CA data found in cluster-info kubeconfig")
}

// getCAFromServiceAccount reads the CA certificate from the service account mount.
// This works when running inside a pod in the cluster.
func (t *tkaClient) getCAFromServiceAccount() ([]byte, humane.Error) {
	caPath := "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"

	// Check if the file exists
	if _, err := os.Stat(caPath); os.IsNotExist(err) {
		return nil, humane.New("Service account CA file not found", "Not running in a pod or service account not mounted")
	}

	// Read the CA certificate file
	file, err := os.Open(caPath)
	if err != nil {
		return nil, humane.Wrap(err, "Failed to open service account CA file")
	}
	defer func() {
		if err := file.Close(); err != nil {
			otelzap.L().WithError(err).Warn("Failed to close service account CA file")
		}
	}()

	caBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, humane.Wrap(err, "Failed to read service account CA file")
	}

	// Return raw PEM bytes; clientcmd serializes CertificateAuthorityData as base64 later
	return caBytes, nil
}

// inferPublicEndpointFromCert tries to infer a public-facing API server URL by
// performing a TLS handshake against the configured host and inspecting the
// certificate SubjectAltNames for externally routable DNS names or IPs.
func (t *tkaClient) inferPublicEndpointFromCert(ctx context.Context) (string, humane.Error) {
	parsed, err := url.Parse(t.config.Host)
	if err != nil {
		return "", humane.Wrap(err, "Failed to parse API server host URL")
	}

	host := parsed.Hostname()
	port := parsed.Port()
	if port == "" {
		port = "443"
	}

	// Build a cert pool from our known CA (if available) to validate the server cert
	rootCAs := x509.NewCertPool()
	if len(t.config.CAData) > 0 {
		if ok := rootCAs.AppendCertsFromPEM(t.config.CAData); !ok {
			// Continue, but we might need to skip verification if we truly can't add CA
			otelzap.L().Warn("Failed to append CA data to cert pool")
		}
	}

	// Prepare TLS config. Prefer to verify using provided CA when available.
	// If no CA is available, we still perform the handshake but skip verify to read certs.
	skipVerify := len(t.config.CAData) == 0
	tlsCfg := &tls.Config{
		InsecureSkipVerify: skipVerify, //nolint:gosec // used only to read cert SANs when CAData missing
		RootCAs:            rootCAs,
	}

	conn, err := tls.Dial("tcp", net.JoinHostPort(host, port), tlsCfg)
	if err != nil {
		return "", humane.Wrap(err, "Failed TLS handshake to API server for cert inspection")
	}
	defer func() {
		if err := conn.Close(); err != nil {
			otelzap.L().WithError(err).Warn("Failed to close TLS connection")
		}
	}()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return "", humane.New("No peer certificates from API server")
	}

	leaf := state.PeerCertificates[0]

	// Prefer DNS names that look public; otherwise try IPs that are not private
	for _, dns := range leaf.DNSNames {
		if isPublicDNSName(dns) {
			return buildHTTPSURL(dns, port), nil
		}
	}

	for _, ip := range leaf.IPAddresses {
		if isPublicIP(ip) {
			return buildHTTPSURL(ip.String(), port), nil
		}
	}

	return "", humane.New("No suitable public SAN found on API server certificate")
}

func isPublicDNSName(name string) bool {
	// Exclude typical internal cluster domains
	if hasSuffixFold(name, ".cluster.local") || hasSuffixFold(name, ".svc") || hasSuffixFold(name, ".svc.cluster.local") {
		return false
	}
	// Must contain a dot to be a FQDN-like name
	if !containsDot(name) {
		return false
	}
	return true
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	// Filter RFC1918 and link-local addresses
	if ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}
	return true
}

func buildHTTPSURL(host, port string) string {
	if port == "443" || port == "" {
		return fmt.Sprintf("https://%s", host)
	}
	return fmt.Sprintf("https://%s:%s", host, port)
}

func hasSuffixFold(s, suffix string) bool {
	// Case-insensitive suffix check without importing strings for a single use
	if len(suffix) > len(s) {
		return false
	}
	ls := len(s)
	lsuf := len(suffix)
	for i := 0; i < lsuf; i++ {
		cs := s[ls-lsuf+i]
		cf := suffix[i]
		// fold ASCII letters
		if cs >= 'A' && cs <= 'Z' {
			cs += 'a' - 'A'
		}
		if cf >= 'A' && cf <= 'Z' {
			cf += 'a' - 'A'
		}
		if cs != cf {
			return false
		}
	}
	return true
}

func containsDot(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return true
		}
	}
	return false
}
