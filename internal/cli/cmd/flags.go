package cmd

import (
	"time"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/client/k8s"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configFileName string

func addCommonFlags(cmd *cobra.Command) {
	viper.SetDefault("otel.endpoint", "")
	viper.SetDefault("otel.insecure", true)
	viper.SetDefault("operator.namespace", k8s.DefaultNamespace)
	viper.SetDefault("operator.clusterName", k8s.DefaultClusterName)
	viper.SetDefault("operator.contextPrefix", k8s.DefaultContextPrefix)
	viper.SetDefault("operator.userPrefix", k8s.DefaultUserEntryPrefix)
	viper.SetDefault("api.retryAfterSeconds", 1)

	cmd.PersistentFlags().StringVarP(&configFileName, "config", "c", "", "Name of the config file")
	_ = cmd.RegisterFlagCompletionFunc("config", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"json", "yaml", "yaml"}, cobra.ShellCompDirectiveDefault
	})

	cmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	viper.SetDefault("debug", false)
	if err := viper.BindPFlag("debug", cmd.PersistentFlags().Lookup("debug")); err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}

	cmd.PersistentFlags().StringP("server", "s", "tka", "The Server Name on the Tailscale Network")
	viper.SetDefault("server.host", "")
	if err := viper.BindPFlag("tailscale.hostname", cmd.PersistentFlags().Lookup("server")); err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}

	cmd.PersistentFlags().IntP("port", "p", 443, "Port of the gRPC API of the Server")
	viper.SetDefault("tailscale.port", 443)
	if err := viper.BindPFlag("tailscale.port", cmd.PersistentFlags().Lookup("port")); err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}

	cmd.PersistentFlags().BoolP("long", "l", false, "Show long output (where available)")
	viper.SetDefault("output.long", false)
	if err := viper.BindPFlag("output.long", cmd.PersistentFlags().Lookup("long")); err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}

	cmd.PersistentFlags().BoolP("quiet", "q", false, "Show no output (where available)")
	viper.SetDefault("output.quiet", false)
	if err := viper.BindPFlag("output.quiet", cmd.PersistentFlags().Lookup("quiet")); err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
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
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}

	cmd.PersistentFlags().StringP("cap-name", "n", "specht-labs.de/cap/tka", "name of the capability to request from api")
	viper.SetDefault("tailscale.capName", "specht-labs.de/cap/tka")
	err = viper.BindPFlag("tailscale.capName", cmd.PersistentFlags().Lookup("cap-name"))
	if err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}
}

func addClientFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)

	cmd.PersistentFlags().StringP("theme", "t", "tokyo-night", "theme to use for the CLI")
	viper.SetDefault("output.theme", "tokyo-night")
	err := viper.BindPFlag("output.theme", cmd.PersistentFlags().Lookup("theme"))
	if err != nil {
		panic(humane.Wrap(err, "fatal binding flag", "check that the flag name matches the viper key")) //nolint:nopanic // flag binding errors are programming errors
	}
	_ = cmd.RegisterFlagCompletionFunc("theme", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return pretty_print.AllThemeNames(), cobra.ShellCompDirectiveDefault
	})

	cmd.PersistentFlags().BoolP("no-eval", "e", false, "Do not evaluate the command")
}
