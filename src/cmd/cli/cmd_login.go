package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spechtlabs/tka/pkg/api"
	"github.com/spechtlabs/tka/pkg/models"
	"github.com/spf13/cobra"
)

var (
	quiet  bool
	setEnv bool
)

func init() {
	cmdSignIn.Flags().BoolVarP(&quiet, "quiet", "q", false, "Do not print login information")
	cmdSignIn.Flags().BoolVarP(&setEnv, "set-env", "e", false, "Set KUBECONFIG environment variable")
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

func signIn(cmd *cobra.Command, args []string) error {
	if setEnv {
		_ = cmd.Flags().Set("quiet", "true")
	}

	loginInfo, _, err := doRequestAndDecode[models.UserLoginResponse](http.MethodPost, api.LoginApiRoute, nil, http.StatusCreated, http.StatusAccepted)
	if err != nil {
		if err.Cause() != nil {
			pretty_print.PrintError(err.Cause())
		} else {
			pretty_print.PrintError(err)
		}

		os.Exit(1)
	}

	if !quiet {
		pretty_print.PrintOk("sign-in successful!")
		pretty_print.PrintLoginInformation(loginInfo)
	}

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

	useStatement := fmt.Sprintf("export KUBECONFIG=%s", file)

	if setEnv {
		fmt.Printf("eval $(%s --quiet)", cmd.CommandPath())
	} else if quiet {
		fmt.Println(useStatement)
	} else {
		pretty_print.PrintOk("kubeconfig written to:", file)
		pretty_print.PrintInfoIcon("→", "To use this session, run:", useStatement)
	}

	return nil
}
