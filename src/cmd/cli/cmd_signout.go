package main

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdRoot.AddCommand(cmdSignout)
	cmdDelete.AddCommand(cmdDeleteSignout)
}

var cmdSignout = &cobra.Command{
	Use:     "signout",
	Short:   "Sign out and remove access from the cluster",
	Example: "tka signout",
	RunE:    signOut,
}

var cmdDeleteSignout = &cobra.Command{
	Use:     "signin",
	Short:   "Sign out and remove access from the cluster",
	Example: "tka delete signin",
	RunE:    signOut,
}

func signOut(_ *cobra.Command, args []string) error {
	server := viper.GetString("tailscale")
	req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/kubeconfig", server), nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to contact tailscale: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		renderError(resp)
		return nil
	}

	fmt.Println("👋 You have been signed out.")
	return nil
}
