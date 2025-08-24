package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spechtlabs/tailscale-k8s-auth/pkg/operator"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version represents the Version of the tka binary, should be set via ldflags -X
	Version string

	// Date represents the Date of when the tka binary was build, should be set via ldflags -X
	Date string

	// Commit represents the Commit-hash from which the tka binary was build, should be set via ldflags -X
	Commit string

	undoFunc       func()
	configFileName string
)

func NewRootCmd(initConfigFunc func()) *cobra.Command {
	cobra.OnInitialize(func() {
		initConfig()
		initConfigFunc()
	})

	// rootCmd represents the base command when called without any subcommands
	cmdRoot := cobra.Command{
		Use:   "tka",
		Short: "tka is the CLI for Tailscale Kubernetes Auth",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			undoFunc = utils.InitObservability()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			undoFunc()
		},
	}

	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Shows version information",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\n", Version)
			fmt.Printf("Date:    %s\n", Date)
			fmt.Printf("Commit:  %s\n", Commit)
		},
	}

	cmdRoot.AddCommand(cmdVersion)

	return &cmdRoot
}

func NewCliRootCmd(initConfigFunc func()) *cobra.Command {
	cmdRoot := NewRootCmd(initConfigFunc)
	cmdRoot.Long = `tka is a small CLI to sign in to a Kubernetes cluster using Tailscale identity.
It talks to a tka-api instance and helps you fetch ephemeral kubeconfigs.`
	addClientFlags(cmdRoot)
	return cmdRoot
}

func NewServerRootCmd(initConfigFunc func()) *cobra.Command {
	cmdRoot := NewRootCmd(initConfigFunc)
	cmdRoot.Long = `tka serves the gRPC API for Tailscale Kubernetes Auth`
	addServerFlags(cmdRoot)
	return cmdRoot
}

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

}

func addServerFlags(cmd *cobra.Command) {
	addCommonFlags(cmd)

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
}

func initConfig() {
	if configFileName != "" {
		viper.SetConfigFile(configFileName)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.AddConfigPath("$HOME/.config/tka/")
		viper.AddConfigPath("/data")
	}

	viper.SetEnvPrefix("TKA")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Find and read the config file
	if err := viper.ReadInConfig(); err != nil {
		// Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
}
