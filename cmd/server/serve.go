package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"encoding/base64"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/internal/utils"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/spechtlabs/tka/pkg/cluster"
	authMw "github.com/spechtlabs/tka/pkg/middleware/auth"
	koperator "github.com/spechtlabs/tka/pkg/operator"
	"github.com/spechtlabs/tka/pkg/service"
	"github.com/spechtlabs/tka/pkg/service/api"
	"github.com/spechtlabs/tka/pkg/service/capability"
	"github.com/spechtlabs/tka/pkg/service/models"
	ts "github.com/spechtlabs/tka/pkg/tshttp"
	"github.com/spechtlabs/tka/pkg/tsnet"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"tailscale.com/tailcfg"
)

func init() {
	serveCmd.PersistentFlags().Int("health-port", 8080, "Port for the local metrics and health check server")
	viper.SetDefault("health.port", 8080)
	err := viper.BindPFlag("health.port", serveCmd.PersistentFlags().Lookup("health-port"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	serveCmd.PersistentFlags().String("api-endpoint", "", "API endpoint for the Kubernetes cluster")
	viper.SetDefault("clusterInfo.apiEndpoint", "")
	serveCmd.PersistentFlags().String("ca-data", "", "CA data for the Kubernetes cluster")
	viper.SetDefault("clusterInfo.caData", "")
	serveCmd.PersistentFlags().Bool("insecure-skip-tls-verify", false, "Skip TLS verification for the Kubernetes cluster")
	viper.SetDefault("clusterInfo.insecureSkipTLSVerify", false)
	serveCmd.PersistentFlags().StringToString("labels", nil, "Labels for the Kubernetes cluster")
	viper.SetDefault("clusterInfo.labels", map[string]string{})

	// Defaults for optional ConfigMap reference-based configuration (nested under clusterInfo)
	viper.SetDefault("clusterInfo.configMapRef.enabled", false)
	viper.SetDefault("clusterInfo.configMapRef.name", "cluster-info")
	viper.SetDefault("clusterInfo.configMapRef.namespace", "kube-public")
	viper.SetDefault("clusterInfo.configMapRef.keys.apiEndpoint", "apiEndpoint")
	viper.SetDefault("clusterInfo.configMapRef.keys.caData", "caData")
	viper.SetDefault("clusterInfo.configMapRef.keys.insecure", "insecure")
	viper.SetDefault("clusterInfo.configMapRef.keys.kubeconfig", "kubeconfig")
}

var (
	serveCmd = &cobra.Command{
		Use:   "serve [--server|-s <string>] [--port|-p <int>] [--dir|-d <string>] [--cap-name|-n <string>] [--health-port <int>] [--api-endpoint <string>] [--ca-data <string>] [--insecure-skip-tls-verify] [--labels <key=value>]",
		Short: "Run the TKA API and Kubernetes operator services",
		Long: `Start the Tailscale-embedded HTTP API and the Kubernetes operator.

This command:

- Starts a tailscale tsnet server for inbound connections
- Serves the TKA HTTP API with authentication and capability checks
- Runs the Kubernetes operator to manage kubeconfigs and user resources
- Starts a local HTTP server for metrics and health checks

Configuration is provided via flags and environment variables (see --help).`,
		Example: `# Start the server with defaults from config and environment
tka serve

# Override the capability name and health port
tka serve --cap-name specht-labs.de/cap/custom --health-port 9090

# Start with custom cluster information
tka serve --api-endpoint https://api.cluster.example.com:6443 --labels environment=prod,region=us-west-2

# Start with insecure TLS for development
tka serve --api-endpoint https://localhost:6443 --insecure-skip-tls-verify --labels environment=dev`,
		Args:      cobra.ExactArgs(0),
		ValidArgs: []string{},
		Run: func(cmd *cobra.Command, args []string) {
			if err := runE(cmd, args); err != nil {
				otelzap.L().WithError(err).Fatal("Exiting")
			}

			otelzap.L().Info("Exiting")
		},
	}
)

func configureGinMode(debug bool) {
	if debug {
		configFileName := viper.GetViper().ConfigFileUsed()
		if file, err := os.ReadFile(configFileName); err == nil && len(file) > 0 {
			otelzap.L().Sugar().With(
				"config_file", configFileName,
				string(file), "config", string(file),
			).Debug("Config file used")
		}
		gin.SetMode(gin.DebugMode)
		return
	}

	configFileName := viper.GetViper().ConfigFileUsed()
	otelzap.L().Sugar().With("config_file", configFileName).Debug("Config file used")
	gin.SetMode(gin.ReleaseMode)
}

func getClientOptions() k8s.ClientOptions {
	return k8s.ClientOptions{
		Namespace:     viper.GetString("operator.namespace"),
		ClusterName:   viper.GetString("operator.clusterName"),
		ContextPrefix: viper.GetString("operator.contextPrefix"),
		UserPrefix:    viper.GetString("operator.userPrefix"),
	}
}

func parseBoolish(in string) bool {
	switch in {
	case "true", "1", "TRUE", "True":
		return true
	default:
		return false
	}
}

func loadClusterInfoFromConfigMap(ctx context.Context) (*models.TkaClusterInfo, humane.Error) {
	name := viper.GetString("clusterInfo.configMapRef.name")
	namespace := viper.GetString("clusterInfo.configMapRef.namespace")
	if name == "" || namespace == "" {
		return nil, humane.New("configMapRef.name and configMapRef.namespace are required when configMapRef.enabled is true")
	}

	keyAPI := viper.GetString("clusterInfo.configMapRef.keys.apiEndpoint")
	keyCA := viper.GetString("clusterInfo.configMapRef.keys.caData")
	keyInsecure := viper.GetString("clusterInfo.configMapRef.keys.insecure")
	kubeconfigKey := viper.GetString("clusterInfo.configMapRef.keys.kubeconfig")

	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return nil, humane.Wrap(err, "failed to get Kubernetes rest config")
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, humane.Wrap(err, "failed to create Kubernetes clientset")
	}

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, humane.Wrap(err, "failed to read configMapRef ConfigMap")
	}

	serverURL := cm.Data[keyAPI]
	caData := cm.Data[keyCA]
	insecure := parseBoolish(cm.Data[keyInsecure])

	// kubeadm cluster-info configmap supports embedding a kubeconfig in a single key
	if serverURL == "" && caData == "" && kubeconfigKey != "" {
		if yamlKubeconfig, ok := cm.Data[kubeconfigKey]; ok && yamlKubeconfig != "" {
			cfg, err := clientcmd.Load([]byte(yamlKubeconfig))
			if err != nil {
				return nil, humane.Wrap(err, "failed to parse kubeconfig from ConfigMap")
			}
			// Use the first cluster entry
			for _, cluster := range cfg.Clusters {
				serverURL = cluster.Server
				if len(cluster.CertificateAuthorityData) > 0 {
					caData = base64.StdEncoding.EncodeToString(cluster.CertificateAuthorityData)
				}
				break
			}
		}
	}

	if serverURL == "" {
		return nil, humane.New("configMapRef missing api endpoint", fmt.Sprintf("ConfigMap %s/%s missing key '%s'", namespace, name, keyAPI))
	}

	return &models.TkaClusterInfo{
		ServerURL:             serverURL,
		CAData:                caData,
		InsecureSkipTLSVerify: insecure,
		Labels:                viper.GetStringMapString("clusterInfo.labels"),
	}, nil
}

func loadClusterInfo(ctx context.Context) (*models.TkaClusterInfo, humane.Error) {
	useCMRef := viper.GetBool("clusterInfo.configMapRef.enabled")
	explicitEndpoint := viper.GetString("clusterInfo.apiEndpoint")
	if useCMRef && explicitEndpoint != "" {
		return nil, humane.New("invalid configuration: both clusterInfo and configMapRef provided", "Use either explicit clusterInfo.* settings or enable configMapRef, not both")
	}

	if useCMRef {
		return loadClusterInfoFromConfigMap(ctx)
	}

	clusterInfo := &models.TkaClusterInfo{
		ServerURL:             explicitEndpoint,
		CAData:                viper.GetString("clusterInfo.caData"),
		InsecureSkipTLSVerify: viper.GetBool("clusterInfo.insecureSkipTLSVerify"),
		Labels:                viper.GetStringMapString("clusterInfo.labels"),
	}

	if clusterInfo.ServerURL == "" {
		return nil, humane.New("clusterInfo.apiEndpoint is required", "Please provide the API endpoint for the Kubernetes cluster in the config file or via the --api-endpoint flag")
	}

	return clusterInfo, nil
}

func getHealthPort() int {
	localPort := viper.GetInt("health.port")
	if localPort == 0 {
		localPort = 8080
	}
	return localPort
}

func runE(cmd *cobra.Command, _ []string) humane.Error {
	debug := viper.GetBool("debug")
	configureGinMode(debug)

	ctx, cancelFn := context.WithCancelCause(cmd.Context())
	utils.InterruptHandler(ctx, cancelFn)

	clientOpts := getClientOptions()

	clusterInfo, err := loadClusterInfo(ctx)
	if err != nil {
		cancelFn(err)
		return err
	}

	k8sOperator, err := koperator.NewK8sOperator(clusterInfo, clientOpts)
	if err != nil {
		cancelFn(err)
		return err
	}

	// Create Gossip Client
	var gossipClient *cluster.GossipClient[service.NodeMetadata]
	var gossipStore cluster.GossipStore[service.NodeMetadata]

	// Create Tailscale server
	tailscaleServer := tsnet.NewServer(viper.GetString("tailscale.hostname"))

	// Connect to tailscale network
	if _, err := tailscaleServer.Up(ctx); err != nil {
		cancelFn(err)
		return humane.Wrap(err, "failed to start api tailscale", "check (debug) logs for more details")
	}

	gossipClient, gossipStore, err = newGossip(tailscaleServer, clusterInfo)
	if err != nil {
		cancelFn(err)
		return err
	}

	// Create shared Prometheus instance for all servers
	sharedPrometheus := ginprometheus.NewPrometheus("tka")

	srv, tkaApiServer, err := newTkaServer(
		debug,
		tailscaleServer,
		sharedPrometheus,
		clusterInfo,
		gossipStore,
		k8sOperator.GetClient(),
	)

	if err != nil {
		cancelFn(err)
		return err
	}

	// Create local metrics server
	healthSrv := newHealthServer(tailscaleServer, sharedPrometheus)
	healthSrv.Addr = fmt.Sprintf(":%d", getHealthPort())

	// Start the gossip client
	if viper.GetBool("gossip.enabled") {
		go gossipClient.Start(ctx)
	}

	// Start TKA server (Tailscale)
	go func() {
		// Start the tailscale http server connection
		if err := srv.Start(ctx); err != nil {
			herr := humane.Wrap(err, "failed to connect to tailscale", "ensure your TS_AUTH_KEY is set and valid")
			cancelFn(herr)
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start TKA tailscale")
		}

		network := "tcp"
		if viper.GetInt("tailscale.port") == 443 {
			network = "tls"
		}
		if err := srv.Serve(ctx, tkaApiServer.Engine(), network); err != nil {
			if err.Cause() != nil {
				cancelFn(err.Cause())
			} else {
				cancelFn(err)
			}
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to serve TKA tailscale")
		}
	}()

	// Start metrics server (Local)
	go func() {
		otelzap.L().InfoContext(ctx, "Starting local metrics server", zap.String("addr", healthSrv.Addr))

		if err := healthSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			cancelFn(fmt.Errorf("local metrics server failed: %w", err))
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start local metrics server")
		}
	}()

	// Start Kubernetes operator
	go func() {
		if err := k8sOperator.Start(ctx); err != nil {
			if err.Cause() != nil {
				cancelFn(err.Cause())
			} else {
				cancelFn(err)
			}
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start k8s operator")
		}
	}()

	// Wait for context done
	<-ctx.Done()
	// No more logging to ctx from here onwards

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	otelzap.L().Info("Shutting down servers...")

	// Shutdown Gossip
	if gossipClient != nil {
		gossipClient.Stop()
	}

	// Shutdown local metrics server first
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		otelzap.L().WithError(err).Error("Failed to shutdown local metrics server gracefully")
		// Continue with other shutdowns even if local server shutdown failed
	}

	// Shutdown Tailscale server
	if err := srv.Stop(shutdownCtx); err != nil {
		otelzap.L().WithError(err).Error("Failed to shutdown Tailscale server gracefully")
		return err
	}

	otelzap.L().Info("Servers shut down successfully")

	// Check termination cause
	cause := context.Cause(ctx)
	if cause != nil && !errors.Is(cause, context.Canceled) {
		return humane.Wrap(cause, "server terminated due to error")
	}

	return nil
}

func newTkaServer(
	debug bool,
	tailscaleServer tsnet.TSNet,
	prom *ginprometheus.Prometheus,
	clusterInfo *models.TkaClusterInfo,
	gossipStore cluster.GossipStore[service.NodeMetadata],
	k8sClient k8s.TkaClient) (*ts.Server, *api.TKAServer, humane.Error) {
	tsServer := ts.NewServer(tailscaleServer,
		ts.WithDebug(debug),
		ts.WithPort(viper.GetInt("tailscale.port")),
		ts.WithStateDir(viper.GetString("tailscale.stateDir")),
		ts.WithReadTimeout(viper.GetDuration("server.readTimeout")),
		ts.WithReadHeaderTimeout(viper.GetDuration("server.readHeaderTimeout")),
		ts.WithWriteTimeout(viper.GetDuration("server.writeTimeout")),
		ts.WithIdleTimeout(viper.GetDuration("server.idleTimeout")),
	)

	authMiddleware := authMw.NewGinAuthMiddleware[capability.Rule](tsServer, tailcfg.PeerCapability(viper.GetString("tailscale.capName")))

	apiServer := api.NewTKAServer(
		api.WithRetryAfterSeconds(viper.GetInt("api.retryAfterSeconds")),
		api.WithPrometheusMiddleware(prom),
		api.WithClusterInfo(clusterInfo),
		api.WithAuthMiddleware(authMiddleware),
		api.WithGossipStore(gossipStore),
	)

	if err := apiServer.LoadApiRoutes(k8sClient); err != nil {
		return nil, nil, err
	}

	return tsServer, apiServer, nil
}

func newHealthServer(tsServer tsnet.TSNet, prom *ginprometheus.Prometheus) *http.Server {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(ginzap.GinzapWithConfig(otelzap.L(), &ginzap.Config{
		UTC:        true,
		TimeFormat: time.RFC3339,
	}))

	// Metrics endpoint - expose all Prometheus metrics
	// Since we're using a shared Prometheus instance, all metrics will be available via the default handler
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Controller metrics endpoint - expose controller-runtime metrics for backwards compatibility
	router.GET("/metrics/controller", gin.WrapH(promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{})))

	// Ready endpoint - checks if Tailscale server is connected
	router.GET("/ready", func(c *gin.Context) {
		status := "not ready"
		httpStatus := http.StatusServiceUnavailable
		reason := "tailscale server not initialized"
		if tsServer != nil && tsServer.IsConnected() {
			status = "ready"
			httpStatus = http.StatusOK
			reason = fmt.Sprintf("tailscale server is %s state", tsServer.BackendState())
		}

		// Check if Tailscale server is started
		c.JSON(httpStatus, gin.H{
			"status": status,
			"reason": reason,
		})
	})

	return &http.Server{
		Handler:           router,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func newGossip(tailscaleServer tsnet.TSNet, clusterInfo *models.TkaClusterInfo) (*cluster.GossipClient[service.NodeMetadata], cluster.GossipStore[service.NodeMetadata], humane.Error) {
	gossipPort := viper.GetInt("gossip.port")
	// We listen on all interfaces for gossip
	listenAddr := fmt.Sprintf(":%d", gossipPort)

	meta := service.NodeMetadata{
		APIEndpoint: clusterInfo.ServerURL,
		Labels:      clusterInfo.Labels,
	}

	// Parse port from URL
	if u, _ := url.Parse(clusterInfo.ServerURL); u != nil {
		p := u.Port()
		if p != "" {
			meta.APIPort, _ = strconv.Atoi(p)
		} else {
			if u.Scheme == "https" {
				meta.APIPort = 443
			} else {
				meta.APIPort = 80
			}
		}
	}

	gossipStore := cluster.NewInMemoryGossipStore[service.NodeMetadata](
		fmt.Sprintf("%s:%d", viper.GetString("tailscale.hostname"), gossipPort),
		cluster.WithLocalState(meta),
	)

	listener, err := tailscaleServer.Listen("tcp", listenAddr)
	if err != nil {
		otelzap.L().WithError(err).Error("Failed to listen for gossip")
		return nil, nil, humane.Wrap(err, "failed to listen for gossip")
	}

	gossipClient := cluster.NewGossipClient[service.NodeMetadata](
		gossipStore,
		listener,
		cluster.WithGossipFactor[service.NodeMetadata](viper.GetInt("gossip.factor")),
		cluster.WithGossipInterval[service.NodeMetadata](viper.GetDuration("gossip.interval")),
		cluster.WithBootstrapPeer[service.NodeMetadata](viper.GetStringSlice("gossip.bootstrapPeers")...),
	)

	return gossipClient, gossipStore, nil
}
