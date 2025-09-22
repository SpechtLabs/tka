//go:build unix

package main

import (
	"os"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdSignIn.PersistentFlags().Bool("shell", false, "Start a subshell with temporary Kubernetes access")
}

var cmdSignIn = &cobra.Command{
	Use:     "login [--quiet|-q] [--long|-l|--no-eval|-e] [--shell]",
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

		useShell, err := cmd.Flags().GetBool("shell")
		if err != nil {
			useShell = false
		}

		if useShell {
			err := forkShell(cmd, args)
			if err != nil {
				pretty_print.PrintError(err)
				os.Exit(1)
			} else {
				return
			}
		}

		file, err := signIn(quiet)
		if err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		printUseStatement(file, quiet)
	},
}
