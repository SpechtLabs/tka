package main

import (
	"net/http"
	"os"
	"time"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	api "github.com/spechtlabs/tka/pkg/api/tka_api"
	"github.com/spechtlabs/tka/pkg/models"
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

var cmdGetSignIn = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Show current login information and provisioning status.",
	Long: `Display details about your current login state, including whether provisioning was successful.
This does not modify your session.`,
	Example: `# Display current login status information
tka get login`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	RunE: func(cmd *cobra.Command, args []string) error {
		loginInfo, code, err := doRequestAndDecode[models.UserLoginResponse](http.MethodGet, api.LoginApiRoute, nil, http.StatusOK, http.StatusProcessing)
		if err != nil {
			pretty_print.PrintError(err.Cause())
			os.Exit(1)
		}

		pretty_print.PrintInfo("Login Information:")
		pretty_print.PrintLoginInfoWithProvisioning(loginInfo, code)

		return nil
	},
}

func signIn(quiet bool) (string, error) {
	loginInfo, _, err := doRequestAndDecode[models.UserLoginResponse](http.MethodPost, api.LoginApiRoute, nil, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		if err.Cause() != nil {
			return "", err.Cause()
		}

		return "", err
	}

	if !quiet {
		pretty_print.PrintOk("sign-in successful!")
		pretty_print.PrintLoginInformation(loginInfo)
	}

	time.Sleep(100 * time.Millisecond)

	kubecfg, err := fetchKubeConfig(quiet)
	if err != nil {
		return "", err
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		return "", err
	}

	return file, nil
}
