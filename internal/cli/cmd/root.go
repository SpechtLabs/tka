package cmd

import (
	"fmt"
	"slices"

	humane "github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewRootCmd creates the base root command for the TKA CLI without any subcommands.
// It sets up common configuration initialization and error handling.
func NewRootCmd() *cobra.Command {
	cobra.OnInitialize(initConfig)

	// rootCmd represents the base command when called without any subcommands
	cmdRoot := cobra.Command{
		Use:   "tka",
		Short: "tka is the CLI for Tailscale Kubernetes Auth",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			utils.InitObservability()
		},
	}

	cmdRoot.AddCommand(newVersionCmd())
	errPrefix := pretty_print.FormatWithOptions(pretty_print.ErrLvl, "Error:", []string{}, pretty_print.WithoutNewline())
	cmdRoot.SetErrPrefix(errPrefix)

	cmdRoot.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		initConfig()
		pretty_print.PrintHelpText(cmd, args)
	})
	cmdRoot.SetUsageFunc(func(cmd *cobra.Command) error {
		initConfig()
		fmt.Println("")
		pretty_print.PrintUsageText(cmd, []string{})
		return nil
	})
	cmdRoot.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		initConfig()
		pretty_print.PrintErrorMessage(err.Error())
		fmt.Println("")
		pretty_print.PrintHelpText(cmd, []string{})
		return nil
	})

	return &cmdRoot
}

// NewCliRootCmd creates the root command configured for the TKA client CLI
// with client-specific flags and theming support.
func NewCliRootCmd() *cobra.Command {
	cmdRoot := NewRootCmd()
	addClientFlags(cmdRoot)
	cmdRoot.Use = "tka [--config|-c <string>] [--debug] [--server|-s <string>] [--port|-p <int>] [--long|-l] [--theme|-t <string>] [--no-eval|-e]"

	cmdRoot.Long = `tka is the client for Tailscale Kubernetes Auth. It lets you authenticate to clusters over Tailscale, manage kubeconfig entries, and inspect status with readable, themed output.

### Theming

Control the CLI's look and feel using one of the following:

- Flag: ` + "`--theme`" + ` or ` + "`-t`" + `
- Config: ` + "`output.theme`" + ` (in config file)
- Environment: ` + "`TKA_THEME`" + `

**Accepted themes**: ascii, dark, dracula, *tokyo-night*, light

### Notes

- Global flags like ` + "`--theme`" + ` are available to subcommands`

	cmdRoot.Example = `# generic dark theme
$ tka --theme dark login

# light theme
$ TKA_OUTPUT_THEME=light tka kubeconfig

# no theme (useful in non-interactive contexts)
$ tka --theme notty login
`

	cmdRoot.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		theme := viper.GetString("output.theme")
		if theme == "" {
			theme = "tokyo-night"
		}
		if !slices.Contains(pretty_print.AllThemeNames(), theme) {
			viper.Set("output.theme", pretty_print.TokyoNightStyle)
			return humane.New("invalid theme: "+theme, "use one of the supported themes: "+fmt.Sprintf("%v", pretty_print.AllThemeNames()))
		}
		return nil
	}

	return cmdRoot
}

// NewServerRootCmd creates the root command configured for the TKA server
// with server-specific flags for running the operator and API.
func NewServerRootCmd() *cobra.Command {
	cmdRoot := NewRootCmd()
	addServerFlags(cmdRoot)
	cmdRoot.Use = "tka [--config|-c <string>] [--debug]"
	return cmdRoot
}
