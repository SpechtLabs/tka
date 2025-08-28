package cmd

import (
	"fmt"
	"slices"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

	cmdRoot.AddCommand(newVersionCmd())
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
$ tka --theme dark login

# light theme
TKA_THEME=light tka kubeconfig

# no theme (usefull in non-interactive contexts)
$ tka --theme notty login
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
