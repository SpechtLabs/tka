package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdRoot.AddCommand(cmdSignout)
}

var cmdSignout = &cobra.Command{
	Use:     "signout",
	Short:   "Sign out and remove access from the cluster",
	Example: "tka signout",
	RunE: func(cmd *cobra.Command, args []string) error {
		server := viper.GetString("server")
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/kubeconfig", server), nil)
		if err != nil {
			return err
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("failed to contact server: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			renderError(resp)
			return nil
		}

		fmt.Println("👋 You have been signed out.")
		return nil
	},
}
