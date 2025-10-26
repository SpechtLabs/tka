package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Experimental cli for cluster gossip",
	Long: `The cluster command is an experimental cli for cluster gossip.
It allows you to play around with gossip protocols and see how they work.
It is not meant to be used in production.`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
