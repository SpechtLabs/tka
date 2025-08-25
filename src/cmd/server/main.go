package main

import (
	"fmt"
	"os"

	"github.com/spechtlabs/tka/internal/cli/cmd"
	"github.com/spf13/viper"
)

var (
	debug bool
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
	debug = viper.GetBool("debug")
}
