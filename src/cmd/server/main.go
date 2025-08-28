package main

import (
	"fmt"
	"os"

	"github.com/spechtlabs/tka/internal/cli/cmd"
)

func main() {
	cmdRoot := cmd.NewServerRootCmd()

	cmdRoot.AddCommand(serveCmd)

	err := cmdRoot.Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
