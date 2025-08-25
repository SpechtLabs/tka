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
	serverAddr string
)

func main() {
	cmdRoot := cmd.NewCliRootCmd(initConfig)

	var cmdGet = &cobra.Command{
		Use:   "get",
		Short: "Retrieve read-only resources from TKA.",
		Long:  `The get command retrieves resources from your Tailscale Kubernetes Auth service`,
		Example: `# Fetch your current kubeconfig
tka get kubeconfig

# Show current login information
tka get login`,
	}

	cmdRoot.AddCommand(cmdShell)
	cmdRoot.AddCommand(cmdSignIn)
	cmdRoot.AddCommand(cmdKubeconfig)
	cmdRoot.AddCommand(cmdSignout)
	cmdRoot.AddCommand(cmdReauth)

	cmdRoot.AddCommand(cmdGet)
	cmdGet.AddCommand(cmdGetSignIn)
	cmdGet.AddCommand(cmdKubeconfig)

	if err := cmdRoot.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
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

	serverAddr = fmt.Sprintf("%s%s.%s:%d", prefix, hostname, tailnet, apiPort)
}
