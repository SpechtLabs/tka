package main

import (
	"github.com/spf13/cobra"
)

func init() {
	cmdReauth.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not print login information")
	cmdReauth.Flags().BoolVarP(&setEnv, "set-env", "e", false, "Set KUBECONFIG environment variable")
}

var cmdReauth = &cobra.Command{
	Use:     "reauthenticate",
	Aliases: []string{"reauth", "refresh"},
	Short:   "Reauthenticate and configure kubectl with temporary access",
	Example: "tka reauthenticate",
	Long:    `Reauthenticate and configure kubectl with temporary access`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if setEnv {
			_ = cmd.Flags().Set("quiet", "true")
		}

		if err := signOut(cmd, args); err != nil {
			return err
		}

		return signIn(cmd, args)
	},
}
