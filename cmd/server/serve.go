package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/internal/utils"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	koperator "github.com/spechtlabs/tka/pkg/operator"
	"github.com/spechtlabs/tka/pkg/service/auth/api"
	"github.com/spechtlabs/tka/pkg/service/auth/models"
	ts "github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	ginprometheus "github.com/zsais/go-gin-prometheus"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
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

func runE(cmd *cobra.Command, _ []string) humane.Error {
	debug := viper.GetBool("debug")
	if debug {
		configFileName := viper.GetViper().ConfigFileUsed()
		if file, err := os.ReadFile(configFileName); err == nil && len(file) > 0 {
			otelzap.L().Sugar().With(
				"config_file", configFileName,
				string(file), "config", string(file),
			).Debug("Config file used")
		}
		gin.SetMode(gin.DebugMode)
	} else {
		configFileName := viper.GetViper().ConfigFileUsed()
		otelzap.L().Sugar().With("config_file", configFileName).Debug("Config file used")
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, cancelFn := context.WithCancelCause(cmd.Context())
	utils.InterruptHandler(ctx, cancelFn)

	clientOpts := k8s.ClientOptions{
		Namespace:     viper.GetString("operator.namespace"),
		ClusterName:   viper.GetString("operator.clusterName"),
		ContextPrefix: viper.GetString("operator.contextPrefix"),
		UserPrefix:    viper.GetString("operator.userPrefix"),
	}

	clusterInfo := &models.TkaClusterInfo{
		ServerURL:             viper.GetString("clusterInfo.apiEndpoint"),
		CAData:                viper.GetString("clusterInfo.caData"),
		InsecureSkipTLSVerify: viper.GetBool("clusterInfo.insecureSkipTLSVerify"),
		Labels:                viper.GetStringMapString("clusterInfo.labels"),
	}

	if clusterInfo.ServerURL == "" {
		return humane.New("clusterInfo.apiEndpoint is required", "Please provide the API endpoint for the Kubernetes cluster in the config file or via the --api-endpoint flag")
	}

	k8sOperator, err := koperator.NewK8sOperator(clusterInfo, clientOpts)
	if err != nil {
		cancelFn(err)
		return err
	}

	// Create Tailscale server
	srv := ts.NewServer(viper.GetString("tailscale.hostname"),
		ts.WithDebug(debug),
		ts.WithPort(viper.GetInt("tailscale.port")),
		ts.WithStateDir(viper.GetString("tailscale.stateDir")),
		ts.WithReadTimeout(viper.GetDuration("server.readTimeout")),
		ts.WithReadHeaderTimeout(viper.GetDuration("server.readHeaderTimeout")),
		ts.WithWriteTimeout(viper.GetDuration("server.writeTimeout")),
		ts.WithIdleTimeout(viper.GetDuration("server.idleTimeout")),
	)

	// Start the Tailscale connection
	if err := srv.Start(ctx); err != nil {
		herr := humane.Wrap(err, "failed to connect to tailscale", "ensure your TS_AUTH_KEY is set and valid")
		cancelFn(herr)
		return herr
	}

	// Create shared Prometheus instance for all servers
	sharedPrometheus := ginprometheus.NewPrometheus("tka")

	tkaServer := api.NewTKAServer(srv,
		api.WithRetryAfterSeconds(viper.GetInt("api.retryAfterSeconds")),
		api.WithPrometheusMiddleware(sharedPrometheus),
		api.WithClusterInfo(clusterInfo),
	)

	if err := tkaServer.LoadApiRoutes(k8sOperator.GetClient()); err != nil {
		cancelFn(err)
		return err
	}

	// Create local metrics server
	healthSrv := newHealthServer(srv, sharedPrometheus)
	localPort := viper.GetInt("health.port")
	if localPort == 0 {
		localPort = 8080 // Default local metrics port
	}
	healthSrv.Addr = fmt.Sprintf(":%d", localPort)

	// Start TKA server (Tailscale)
	go func() {
		if err := tkaServer.Serve(ctx); err != nil {
			if err.Cause() != nil {
				cancelFn(err.Cause())
			} else {
				cancelFn(err)
			}
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start TKA tailscale")
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

	// Shutdown local metrics server first
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		otelzap.L().WithError(err).Error("Failed to shutdown local metrics server gracefully")
		// Continue with other shutdowns even if local server shutdown failed
	}

	// Shutdown TKA server (stops accepting new requests)
	if err := tkaServer.Shutdown(shutdownCtx); err != nil {
		otelzap.L().WithError(err).Error("Failed to shutdown TKA server gracefully")
		// Continue with Tailscale shutdown even if TKA shutdown failed
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

// newHealthServer creates a local HTTP server for metrics and health checks
func newHealthServer(tsServer ts.TailscaleServer, prom *ginprometheus.Prometheus) *http.Server {
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
