package cmd

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/operator"
	"github.com/spechtlabs/tka/pkg/utils"
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
			utils.InitObservability()
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
	errPrefix := pretty_print.FormatWithOptions(pretty_print.ErrLvl, "Error:", []string{}, pretty_print.WithoutNewline())
	cmdRoot.SetErrPrefix(errPrefix)

	cmdRoot.SetHelpFunc(pretty_print.PrintHelpText)
	cmdRoot.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Println("")
		pretty_print.PrintUsageText(cmd, []string{})
		return nil
	})
	cmdRoot.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		pretty_print.PrintErrorMessage(err.Error())
		fmt.Println("")
		pretty_print.PrintHelpText(cmd, []string{})
		return nil
	})

	return &cmdRoot
}

func NewCliRootCmd(initConfigFunc func()) *cobra.Command {
	cmdRoot := NewRootCmd(initConfigFunc)
	addClientFlags(cmdRoot)

	cmdRoot.Long = `tka is the client for Tailscale Kubernetes Auth. It lets you authenticate to clusters over Tailscale, manage kubeconfig entries, and inspect status with readable, themed output.

### Theming

Control the CLI's look and feel using one of the following:
- Flag: ` + "`--theme`" + ` or ` + "`-t`" + `
- Config: ` + "`theme`" + ` (in config file)
- Environment: ` + "`TKA_THEME`" + `

**Accepted themes**: ascii, dark, dracula, *tokyo-night*, light

### Notes:
- Global flags like ` + "`--theme`" + ` are available to subcommands`

	cmdRoot.Example = `# generic dark theme
tka --theme dark login

# light theme
TKA_THEME=light tka kubeconfig

# no theme (usefull in non-interactive contexts)
tka --theme notty login
`

	cmdRoot.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		theme := viper.GetString("theme")
		if !slices.Contains(pretty_print.AllThemeNames(), theme) {
			viper.Set("theme", "tokyo-night")
			return fmt.Errorf("invalid theme: %s", theme)
		}
		return nil
	}

	return cmdRoot
}

func NewServerRootCmd(initConfigFunc func()) *cobra.Command {
	cmdRoot := NewRootCmd(initConfigFunc)
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

	// Find and read the config file (optional). If not found, continue; if malformed, print and exit.
	if err := viper.ReadInConfig(); err != nil {
		if _, notFound := err.(viper.ConfigFileNotFoundError); notFound {
			return
		}
		fmt.Fprintf(os.Stderr, "error reading config file: %v\n", err)
		os.Exit(2)
	}
}
