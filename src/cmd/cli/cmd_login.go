package main

import (
	"net/http"
	"os"
	"time"

	"github.com/spechtlabs/tailscale-k8s-auth/cmd/cli/tui"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/api"
	"github.com/spf13/cobra"
)

func init() {
	cmdRoot.AddCommand(cmdSignIn)
	cmdGet.AddCommand(cmdGetSignIn)
	cmdNew.AddCommand(cmNewSignIn)
}

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

var cmNewSignIn = &cobra.Command{
	Use:     "login",
	Aliases: []string{"signin"},
	Short:   "Sign in and configure kubectl with temporary access",
	Example: "tka new login",
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
		loginInfo, code, err := doRequestAndDecode[api.UserLoginResponse](http.MethodGet, api.LoginApiRoute, nil, http.StatusOK, http.StatusProcessing)
		if err != nil {
			tui.PrintError(err.Cause())
			os.Exit(1)
		}

		tui.PrintInfo("Login Information:")
		tui.PrintLoginInfoWithProvisioning(loginInfo, code)

		return nil
	},
}

func signIn(_ *cobra.Command, _ []string) error {
	loginInfo, _, err := doRequestAndDecode[api.UserLoginResponse](http.MethodPost, api.LoginApiRoute, nil, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		tui.PrintError(err.Cause())
		os.Exit(1)
	}

	tui.PrintOk("sign-in successful!")
	time.Sleep(100 * time.Millisecond)

	kubecfg, err := fetchKubeConfig()
	if err != nil {
		tui.PrintError(err)
		os.Exit(1)
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		tui.PrintError(err)
		os.Exit(1)
	}

	tui.PrintOk("kubeconfig saved to", file)

	tui.PrintInfo("Login Information:")
	tui.PrintLoginInformation(loginInfo)

	return nil
}
