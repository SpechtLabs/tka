package main

import (
	"net/http"
	"os"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/internal/cli/pretty_print"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/api"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/models"
	"github.com/spf13/cobra"
)

var cmdSignIn = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Sign in and configure kubectl with temporary access",
	Example: "tka login",
	Long: `Authenticate using your Tailscale identity and retrieve a temporary
Kubernetes access token. This command automatically fetches your kubeconfig,
writes it to a temporary file, sets the KUBECONFIG environment variable, and
verifies kubectl connectivity.`,
	RunE: signIn,
}

var cmdGetSignIn = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "TODO",
	Example: "tka get login",
	Long:    `TODO`,
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

func signIn(_ *cobra.Command, _ []string) error {
	loginInfo, _, err := doRequestAndDecode[models.UserLoginResponse](http.MethodPost, api.LoginApiRoute, nil, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		if err.Cause() != nil {
			pretty_print.PrintError(err.Cause())
		} else {
			pretty_print.PrintError(err)
		}

		os.Exit(1)
	}

	pretty_print.PrintOk("sign-in successful!")
	time.Sleep(100 * time.Millisecond)

	kubecfg, err := fetchKubeConfig()
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	pretty_print.PrintOk("kubeconfig saved to", file)

	pretty_print.PrintInfo("Login Information:")
	pretty_print.PrintLoginInformation(loginInfo)

	return nil
}
