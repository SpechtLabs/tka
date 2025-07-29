package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	cmdRoot.AddCommand(cmdVersion)
}

var cmdVersion = &cobra.Command{
	Use:     "version",
	Short:   "Shows version information",
	Example: "tka version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Date:    %s\n", Date)
		fmt.Printf("Commit:  %s\n", Commit)
	},
}
