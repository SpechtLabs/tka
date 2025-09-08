package main

import (
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdReauth = &cobra.Command{
	Use:     "reauthenticate [--quiet|-q] [--long|-l|--no-eval|-e]",
	Aliases: []string{"reauth", "refresh"},
	Short:   "Reauthenticate and configure kubectl with temporary access",
	Long: `Reauthenticate by signing out and then signing in again to refresh your temporary access.
This command is a convenience wrapper which:

1. Calls signout to revoke your current session
2. Calls login to obtain a fresh ephemeral kubeconfig`,
	Example: `# Reauthenticate and see human-friendly output
tka reauthenticate --no-eval

# Reauthenticate and update your current shell's KUBECONFIG
tka reauthenticate`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	Run: func(cmd *cobra.Command, args []string) {
		if err := signOut(cmd, args); err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		quiet := viper.GetBool("output.quiet")

		file, err := signIn(quiet)
		if err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		printUseStatement(file, quiet)
	},
}
