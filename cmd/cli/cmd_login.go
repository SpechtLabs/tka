package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/service/api"
	"github.com/spechtlabs/tka/pkg/service/models"
	"github.com/spf13/cobra"
)

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
		// Unwrap to get the original cause for cleaner error messages
		if err.Cause() != nil {
			return "", fmt.Errorf("sign-in failed: %w", err.Cause())
		}
		return "", fmt.Errorf("sign-in failed: %w", err)
	}

	if !quiet {
		pretty_print.PrintOk("sign-in successful!")
		pretty_print.PrintLoginInformation(loginInfo)
	}

	time.Sleep(100 * time.Millisecond)

	kubecfg, err := fetchKubeConfig(quiet)
	if err != nil {
		return "", fmt.Errorf("failed to fetch kubeconfig: %w", err)
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		return "", fmt.Errorf("failed to serialize kubeconfig: %w", err)
	}

	return file, nil
}
