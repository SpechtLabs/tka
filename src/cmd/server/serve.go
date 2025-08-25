package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/auth/capability"
	authoperator "github.com/spechtlabs/tka/pkg/auth/operator"
	mwtailscale "github.com/spechtlabs/tka/pkg/middleware/auth/tailscale"
	koperator "github.com/spechtlabs/tka/pkg/operator"
	ts "github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"tailscale.com/tailcfg"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Run the TKA API and Kubernetes operator services",
		Long: `Start the Tailscale-embedded HTTP API and the Kubernetes operator.

This command:

- Starts a tailscale tsnet server for inbound connections
- Serves the TKA HTTP API with authentication and capability checks
- Runs the Kubernetes operator to manage kubeconfigs and user resources

Configuration is provided via flags and environment variables (see --help).`,
		Example: `# Start the server with defaults from config and environment
tka serve

# Override the capability name
tka serve --cap-name specht-labs.de/cap/custom`,
		Args:      cobra.ExactArgs(0),
		ValidArgs: []string{},
		RunE:      runE,
	}
)

func runE(cmd *cobra.Command, _ []string) error {
	if debug {
		if file, err := os.ReadFile(viper.GetViper().ConfigFileUsed()); err == nil && len(file) > 0 {
			otelzap.L().Sugar().With("config_file", string(file)).Debug("Config file used")
		}
	}

	ctx, cancelFn := context.WithCancelCause(cmd.Context())
	interruptHandler(ctx, cancelFn)

	opOpts := koperator.OperatorOptions{
		Namespace:     viper.GetString("operator.namespace"),
		ClusterName:   viper.GetString("operator.clusterName"),
		ContextPrefix: viper.GetString("operator.contextPrefix"),
		UserPrefix:    viper.GetString("operator.userPrefix"),
	}

	k8sOperator, err := koperator.NewK8sOperatorWithOptions(opOpts)
	if err != nil {
		cancelFn(err)
		return fmt.Errorf("%s", err.Display())
	}

	srv := ts.NewServer(viper.GetString("tailscale.hostname"),
		ts.WithDebug(debug),
		ts.WithPort(viper.GetInt("tailscale.port")),
		ts.WithStateDir(viper.GetString("tailscale.stateDir")),
		ts.WithReadTimeout(viper.GetDuration("server.readTimeout")),
		ts.WithReadHeaderTimeout(viper.GetDuration("server.readHeaderTimeout")),
		ts.WithWriteTimeout(viper.GetDuration("server.writeTimeout")),
		ts.WithIdleTimeout(viper.GetDuration("server.idleTimeout")),
	)

	capName := tailcfg.PeerCapability(viper.GetString("tailscale.capName"))
	mw := mwtailscale.NewGinAuthMiddlewareFromServer[capability.Rule](srv, capName)
	authSvc := authoperator.New(k8sOperator)

	tkaServer, err := api.NewTKAServer(nil, nil,
		api.WithDebug(debug),
		api.WithRetryAfterSeconds(viper.GetInt("api.retryAfterSeconds")),
		api.WithAuthMiddleware(mw),
		api.WithAuthService(authSvc),
		api.WithTailnetServer(srv),
	)
	if err != nil {
		cancelFn(err)
		return fmt.Errorf("%s", err.Display())
	}

	go func() {
		if err := tkaServer.Serve(ctx); err != nil {
			cancelFn(err.Cause())
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start TKA tailscale")
		}
	}()

	go func() {
		if err := k8sOperator.Start(ctx); err != nil {
			cancelFn(err.Cause())
			otelzap.L().WithError(err).FatalContext(ctx, "Failed to start k8s operator")
		}
	}()

	// Wait for context done
	<-ctx.Done()
	// No more logging to ctx from here onwards

	ctx = context.Background()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("%s", err.Display())
	}

	// Terminate accordingly
	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		otelzap.L().WithError(err).Fatal("Exiting")
	} else {
		otelzap.L().Info("Exiting")
	}

	return nil
}

func interruptHandler(ctx context.Context, cancelCtx context.CancelCauseFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		select {
		// Wait for context cancel
		case <-ctx.Done():

		// Wait for signal
		case sig := <-sigs:
			switch sig {
			case syscall.SIGTERM:
				fallthrough
			case syscall.SIGINT:
				fallthrough
			case syscall.SIGQUIT:
				// On terminate signal, cancel context causing the program to terminate
				cancelCtx(context.Canceled)

			default:
				otelzap.L().WarnContext(ctx, "Received unknown signal", zap.String("signal", sig.String()))
			}
		}
	}()
}
