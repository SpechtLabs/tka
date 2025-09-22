//go:build windows

package main

import (
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdSignIn = &cobra.Command{
	Use:     "login [--quiet|-q] [--long|-l|--no-eval|-e]",
	Aliases: []string{"signin", "auth"},
	Short:   "Sign in and configure kubectl with temporary access",
	Long: `Authenticate using your Tailscale identity and retrieve a temporary
Kubernetes access token. This command automatically fetches your kubeconfig,
writes it to a temporary file, sets the KUBECONFIG environment variable.`,
	Example: `# Sign in with user friendly output
tka login --no-eval

# Login and start using your session
tka login
kubectl get pods`,

	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	Run: func(cmd *cobra.Command, args []string) {
		quiet := viper.GetBool("output.quiet")

		file, err := signIn(quiet)
		if err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		printUseStatement(file, quiet)
	},
}
