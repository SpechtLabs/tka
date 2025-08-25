package main

import (
	"net/http"
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spf13/cobra"
)

func init() {
	cmdSignout.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not print signout information")
}

var cmdSignout = &cobra.Command{
	Use:     "signout",
	Aliases: []string{"logout"},
	Short:   "Sign out and remove access from the cluster",
	Long: `Sign out of the TKA service and revoke your current session.

This command requests the server to invalidate your active credentials. It does
not modify your shell environment automatically. If you previously exported
KUBECONFIG to point at an ephemeral file, consider unsetting or updating it.`,
	Example: `# Sign out and revoke your access
tka signout

# Alias form
tka logout

# Quiet mode (no output)
tka signout --quiet`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	RunE:      signOut,
}

func signOut(_ *cobra.Command, _ []string) error {
	_, _, err := doRequestAndDecode[models.UserLoginResponse](http.MethodPost, api.LogoutApiRoute, nil, http.StatusOK, http.StatusProcessing)
	if err != nil {
		pretty_print.PrintError(err.Cause())
		os.Exit(1)
	}

	if !quiet {
		pretty_print.PrintOk("You have been signed out")
	}

	return nil
}
