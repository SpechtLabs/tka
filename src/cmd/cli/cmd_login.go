package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tailscale-k8s-auth/cmd/cli/tui"
	"github.com/spechtlabs/tailscale-k8s-auth/pkg/tailscale"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func init() {
	cmdRoot.AddCommand(cmdSignIn)
}

var cmdSignIn = &cobra.Command{
	Use:     "login",
	Short:   "Sign in and configure kubectl with temporary access",
	Example: "tka login",
	Long: `Authenticate using your Tailscale identity and retrieve a temporary 
Kubernetes access token. This command automatically fetches your kubeconfig,
writes it to a temporary file, sets the KUBECONFIG environment variable, and 
verifies kubectl connectivity.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		loginInfo, err := loginUser()
		if err != nil {
			tui.Error(err)
			os.Exit(1)
		}

		fmt.Printf("%s %s\n", green.Render("✓"), bold.Render("successfully signed in"))
		time.Sleep(100 * time.Millisecond)

		kubecfg, err := fetchKubeConfig()
		if err != nil {
			tui.Error(err)
			os.Exit(1)
		}

		file, err := serializeKubeconfig(kubecfg)
		if err != nil {
			tui.Error(err)
			os.Exit(1)
		}

		fmt.Printf("%s %s %s\n", green.Render("✓"), bold.Render("kubeconfig saved to"), gray.Render(file))

		// TODO(cedi): fix
		//if err := checkKubectlContext(); err != nil {
		//	tui.Error(err)
		//	os.Exit(1)
		//}

		fmt.Printf("%s %s\n", blue.Render("•"), bold.Render("Login Information:"))
		tui.PrintLoginInformation(loginInfo)

		return nil
	},
}

func loginUser() (*tailscale.UserLoginResponse, humane.Error) {
	return doRequestAndDecode[tailscale.UserLoginResponse](http.MethodPost, tailscale.LoginApiRoute, nil, http.StatusCreated, http.StatusAccepted)
}

func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func fancyFetchKubeconfig() (*api.Config, humane.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	model := pollModel{
		ctx:          ctx,
		cancel:       cancel,
		spinner:      spinner.New(),
		delay:        500 * time.Millisecond,
		attempt:      1,
		message:      "Waiting for kubeconfig to be ready...",
		expectedCode: http.StatusOK,
		maxAttempts:  10,
		startedAt:    time.Now(),
		pollFunc: func() (any, humane.Error) {
			return doRequestAndDecode[api.Config](http.MethodGet, tailscale.KubeconfigApiRoute, nil, http.StatusOK)
		},
	}

	model.spinner.Spinner = spinner.Dot

	prog := tea.NewProgram(model)
	finalModel, err := prog.Run()
	cancel()

	if err != nil {
		return nil, humane.Wrap(err, "UI error while polling")
	}

	m := finalModel.(pollModel)
	if m.err != nil {
		return nil, humane.Wrap(m.err, "failed to fetch kubeconfig")
	}

	return m.result.(*api.Config), nil
}

func fetchKubeConfig() (*api.Config, humane.Error) {
	if isTerminal() {
		return fancyFetchKubeconfig()
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return pollUntilSuccess[api.Config](ctx, tailscale.KubeconfigApiRoute, http.StatusOK, 10)
	}
}

func serializeKubeconfig(kubecfg *api.Config) (string, humane.Error) {
	out, err := clientcmd.Write(*kubecfg)
	if err != nil {
		return "", humane.Wrap(err, "failed to serialize kubeconfig")
	}

	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return "", humane.Wrap(err, "failed to create temp kubeconfig")
	}
	defer func() { _ = tempFile.Close() }()

	_, err = io.WriteString(tempFile, string(out))
	if err != nil {
		return "", humane.Wrap(err, "failed to write temp kubeconfig")
	}

	if err := os.Setenv("KUBECONFIG", tempFile.Name()); err != nil {
		return "", humane.Wrap(err, "failed to set KUBECONFIG")
	}

	return tempFile.Name(), nil
}
