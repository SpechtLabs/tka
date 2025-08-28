package cmd

import (
	"fmt"
	"time"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/operator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configFileName string
)

func addCommonFlags(cmd *cobra.Command) {
	viper.SetDefault("otel.endpoint", "")
	viper.SetDefault("otel.insecure", true)
	viper.SetDefault("operator.namespace", operator.DefaultNamespace)
	viper.SetDefault("operator.clusterName", operator.DefaultClusterName)
	viper.SetDefault("operator.contextPrefix", operator.DefaultContextPrefix)
	viper.SetDefault("operator.userPrefix", operator.DefaultUserEntryPrefix)
	viper.SetDefault("api.retryAfterSeconds", 1)

	cmd.PersistentFlags().StringVarP(&configFileName, "config", "c", "", "Name of the config file")

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	viper.SetDefault("debug", false)
	err := viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	cmd.PersistentFlags().StringP("server", "s", "tka", "The Server Name on the Tailscale Network")
	viper.SetDefault("server.host", "")
	err = viper.BindPFlag("tailscale.hostname", cmd.PersistentFlags().Lookup("server"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	cmd.PersistentFlags().IntP("port", "p", 443, "Port of the gRPC API of the Server")
	viper.SetDefault("tailscale.port", 443)
	err = viper.BindPFlag("tailscale.port", cmd.PersistentFlags().Lookup("port"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	cmd.PersistentFlags().BoolP("long", "l", false, "Show long output (where available)")
	viper.SetDefault("output.long", false)
	err = viper.BindPFlag("output.long", cmd.PersistentFlags().Lookup("long"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}
}

func addServerFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)

	viper.SetDefault("server.readTimeout", 10*time.Second)
	viper.SetDefault("server.readHeaderTimeout", 5*time.Second)
	viper.SetDefault("server.writeTimeout", 20*time.Second)
	viper.SetDefault("server.idleTimeout", 120*time.Second)

	cmd.PersistentFlags().StringP("dir", "d", "", "tsnet state directory; a default one will be created if not provided")
	viper.SetDefault("tailscale.stateDir", "")
	err := viper.BindPFlag("tailscale.stateDir", cmd.PersistentFlags().Lookup("dir"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	cmd.PersistentFlags().StringP("cap-name", "n", "specht-labs.de/cap/tka", "name of the capability to request from api")
	viper.SetDefault("tailscale.capName", "specht-labs.de/cap/tka")
	err = viper.BindPFlag("tailscale.capName", cmd.PersistentFlags().Lookup("cap-name"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}
}

func addClientFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)

	cmd.PersistentFlags().StringP("theme", "t", "tokyo-night", "theme to use for the CLI")
	viper.SetDefault("theme", "tokyo-night")
	err := viper.BindPFlag("theme", cmd.PersistentFlags().Lookup("theme"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}
	_ = cmd.RegisterFlagCompletionFunc("theme", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return pretty_print.AllThemeNames(), cobra.ShellCompDirectiveDefault
	})

	cmd.PersistentFlags().BoolP("no-eval", "e", false, "Do not evaluate the command")
}
