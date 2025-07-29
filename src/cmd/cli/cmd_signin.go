package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdRoot.AddCommand(cmdSignIn)
}

var cmdSignIn = &cobra.Command{
	Use:     "signin",
	Short:   "Sign in to the cluster via Tailscale identity",
	Example: "tka signin",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := viper.GetString("server")
		resp, err := http.Post(fmt.Sprintf("%s/kubeconfig", server), "application/json", nil)
		if err != nil {
			return fmt.Errorf("failed to contact server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
			return renderError(resp)
		}

		fmt.Println("✅ Sign-in request accepted. Please run 'tka get kubeconfig' once your access is provisioned.")
		return nil
	},
}
