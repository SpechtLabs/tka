package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Shows version information",
	Example: "meetingepd version",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Date:    %s\n", Date)
		fmt.Printf("Commit:  %s\n", Commit)
		fmt.Printf("BuiltBy: %s\n", BuiltBy)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
