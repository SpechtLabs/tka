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
		Short: "Get resources from the tka",
		Long:  "Get command retrieves resources from your Tailscale Kubernetes Auth tailscale.\nIt can be used to fetch various resources like kubeconfigs or clusters.",
	}

	cmdRoot.AddCommand(cmdSignIn)
	cmdRoot.AddCommand(cmdKubeconfig)
	cmdRoot.AddCommand(cmdSignout)

	cmdRoot.AddCommand(cmdGet)
	cmdGet.AddCommand(cmdGetSignIn)
	cmdGet.AddCommand(cmdGetKubeconfig)

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
