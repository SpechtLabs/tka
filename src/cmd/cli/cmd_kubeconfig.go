package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spechtlabs/tailscale-k8s-auth/cmd/cli/tui"
	"github.com/spf13/cobra"
)

func init() {
	cmdRoot.AddCommand(cmdKubeconfig)
}

var cmdKubeconfig = &cobra.Command{
	Use:     "kubeconfig",
	Short:   "Fetch your temporary kubeconfig",
	Example: "tka get kubeconfig",
	Args:    cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
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

		fmt.Printf("✅ KUBECONFIG saved to: %s\n", file)

		// TODO(cedi): fix
		//if err := checkKubectlContext(); err != nil {
		//	tui.Error(err)
		//	os.Exit(1)
		//}

		return nil
	},
}

func checkKubectlContext() error {
	cmd := exec.Command("kubectl", "config", "current-context")
	cmd.Stderr = os.Stderr
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl config check failed: %w", err)
	}
	fmt.Printf("➡️  kubectl is now configured for: %s\n", strings.TrimSpace(string(output)))
	return nil
}
