package main

import (
	"fmt"
	"os"

	"github.com/spechtlabs/tailscale-k8s-auth/pkg/cmd"
	"github.com/spf13/viper"
)

var (
	hostname      string
	port          int
	tsNetStateDir string
	capName       string
	debug         bool
)

func main() {
	cmdRoot := cmd.NewServerRootCmd(initConfig)

	cmdRoot.AddCommand(serveCmd)

	err := cmdRoot.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	hostname = viper.GetString("tailscale.hostname")
	port = viper.GetInt("tailscale.port")
	tsNetStateDir = viper.GetString("tailscale.stateDir")
	capName = viper.GetString("tailscale.capName")
	debug = viper.GetBool("debug")
}
