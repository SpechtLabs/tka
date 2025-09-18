package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tka/pkg/api"
	koperator "github.com/spechtlabs/tka/pkg/operator"
	authoperator "github.com/spechtlabs/tka/pkg/service/operator"
	ts "github.com/spechtlabs/tka/pkg/tailscale"
	"github.com/spechtlabs/tka/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	serveCmd = &cobra.Command{
		Use:   "serve [--server|-s <string>] [--port|-p <int>] [--dir|-d <string>] [--cap-name|-n <string>]",
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
	} else {
		configFileName := viper.GetViper().ConfigFileUsed()
		otelzap.L().Sugar().With("config_file", configFileName).Debug("Config file used")
	}

	ctx, cancelFn := context.WithCancelCause(cmd.Context())
	utils.InterruptHandler(ctx, cancelFn)

	opOpts := koperator.OperatorOptions{
		Namespace:     viper.GetString("operator.namespace"),
		ClusterName:   viper.GetString("operator.clusterName"),
		ContextPrefix: viper.GetString("operator.contextPrefix"),
		UserPrefix:    viper.GetString("operator.userPrefix"),
	}

	k8sOperator, err := koperator.NewK8sOperatorWithOptions(opOpts)
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
		herr := humane.Wrap(err, "failed to connect to tailscale", "ensure your TS_AUTH_KEY is set", "ensure your TS_AUTH_KEY is valid")
		cancelFn(herr)
		return herr
	}

	tkaServer, err := api.NewTKAServer(srv, nil,
		api.WithDebug(debug),
		api.WithRetryAfterSeconds(viper.GetInt("api.retryAfterSeconds")),
	)
	if err != nil {
		cancelFn(err)
		return err
	}

	authSvc := authoperator.New(k8sOperator)
	if err := tkaServer.LoadApiRoutes(authSvc); err != nil {
		cancelFn(err)
		return err
	}

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

	// Shutdown TKA server first (stops accepting new requests)
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
