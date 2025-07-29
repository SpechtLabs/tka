package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdRoot.AddCommand(cmdKubeconfig)
}

var cmdKubeconfig = &cobra.Command{
	Use:     "get kubeconfig",
	Short:   "Fetch your temporary kubeconfig",
	Example: "tka get kubeconfig",
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		server := viper.GetString("server")
		resp, err := http.Get(fmt.Sprintf("%s/kubeconfig", server))
		if err != nil {
			return fmt.Errorf("failed to fetch kubeconfig: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return renderError(resp)
		}

		tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to create temp kubeconfig: %w", err)
		}
		defer func() { _ = tempFile.Close() }()

		_, err = io.Copy(tempFile, resp.Body)
		if err != nil {
			return fmt.Errorf("failed to write kubeconfig: %w", err)
		}

		if err := os.Setenv("KUBECONFIG", tempFile.Name()); err != nil {
			return fmt.Errorf("failed to set KUBECONFIG: %w", err)
		}

		fmt.Printf("✅ KUBECONFIG set to: %s\n", tempFile.Name())
		return checkKubectlContext()
	},
}

func checkKubectlContext() error {
	cmd := exec.Command("kubectl", "config", "current-context")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl config check failed: %w", err)
	}
	fmt.Printf("➡️  kubectl is now configured for: %s", strings.TrimSpace(string(output)))
	return nil
}
