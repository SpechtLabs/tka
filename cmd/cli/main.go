package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spechtlabs/tka/internal/cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cmdRoot = cmd.NewCliRootCmd()
)

func main() {
	cmdGet := &cobra.Command{
		Use:   "get <command>",
		Short: "Retrieve read-only resources from TKA.",
		Long:  `The get command retrieves resources from your Tailscale Kubernetes Auth service`,
		Args:  cobra.ExactArgs(0),
		Example: `# Fetch your current kubeconfig
tka get kubeconfig

# Show current login information
tka get login`,
	}

	cmdSet := &cobra.Command{
		Use:   "set <command>",
		Short: "Set resources in TKA.",
		Long:  `The set command sets resources in your Tailscale Kubernetes Auth service`,
		Args:  cobra.ExactArgs(0),
		Example: `# Set the debug setting to true
tka set output.theme dark`,
	}

	cmdGenerate := &cobra.Command{
		Use:   "generate <command>",
		Short: "Generate resources in TKA.",
		Long:  `The generate command generates resources in your Tailscale Kubernetes Auth service`,
		Args:  cobra.ExactArgs(0),
		Example: `# Generate a kubeconfig
tka generate kubeconfig`,
	}

	// Add the verbs
	cmdRoot.AddCommand(cmdGenerate)
	cmdRoot.AddCommand(cmdGet)
	cmdRoot.AddCommand(cmdSet)

	// Config
	cmdRoot.AddCommand(cmdConfig)
	cmdGet.AddCommand(cmdGetConfig)
	cmdSet.AddCommand(cmdSetConfig)

	// Integration
	cmdGenerate.AddCommand(cmdIntegration)
	cmdGenerate.AddCommand(cmdDocumentation)

	// Sign in
	cmdRoot.AddCommand(cmdSignIn)
	cmdGet.AddCommand(cmdGetSignIn)

	// Kubeconfig
	cmdRoot.AddCommand(cmdKubeconfig)
	cmdGet.AddCommand(cmdKubeconfig)

	// Sign out
	cmdRoot.AddCommand(cmdSignout)
	cmdRoot.AddCommand(cmdReauth)

	// Cluster info
	cmdRoot.AddCommand(cmdClusterInfo)
	cmdGet.AddCommand(cmdClusterInfo)

	if err := cmdRoot.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func getServerAddr() string {
	hostname := viper.GetString("tailscale.hostname")
	tailnet := viper.GetString("tailscale.tailnet")
	apiPort := viper.GetInt("tailscale.port")
	prefix := ""

	if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") {
		if apiPort == 443 {
			prefix = "https://"
		} else {
			prefix = "http://"
		}
	}

	if len(tailnet) > 0 && !strings.HasPrefix(tailnet, ".") {
		tailnet = fmt.Sprintf(".%s", tailnet)
	}

	return fmt.Sprintf("%s%s%s:%d", prefix, hostname, tailnet, apiPort)
}
