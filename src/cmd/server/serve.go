package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spechtlabs/go-otel-utils/otelzap"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"tailscale.com/tailcfg"
)

var serveCmd = &cobra.Command{
	Use:     "serve",
	Short:   "Shows version information",
	Example: "meetingepd version",
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			file, err := os.ReadFile(viper.GetViper().ConfigFileUsed())
			if err != nil {
				return fmt.Errorf("fatal error reading config file: %w", err)
			}
			otelzap.L().Sugar().With("config_file", string(file)).Debug("Config file used")
		}

		ctx := context.Background()

		tkaServer, err := tailscale.NewTKAServer(ctx, hostname,
			tailscale.WithDebug(debug),
			tailscale.WithPort(port),
			tailscale.WithStateDir(tsNetStateDir),
			tailscale.WithPeerCapName(tailcfg.PeerCapability(capName)),
		)
		if err != nil {
			return fmt.Errorf(err.Display())

		}

		tkaServer.ServeAsync(cmd.Context())

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		if err := tkaServer.Shutdown(); err != nil {
			return fmt.Errorf(err.Display())
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
