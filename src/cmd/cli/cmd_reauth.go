package main

import (
	"github.com/spf13/cobra"
)

func init() {
	cmdReauth.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not print login information")
}

var cmdReauth = &cobra.Command{
	Use:     "reauthenticate",
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := signOut(cmd, args); err != nil {
			return err
		}

		return signIn(cmd, args)
	},
}
