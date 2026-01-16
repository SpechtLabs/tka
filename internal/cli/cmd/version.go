package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// Version represents the Version of the tka binary, should be set via ldflags -X
	Version string

	// Date represents the Date of when the tka binary was build, should be set via ldflags -X
	Date string

	// Commit represents the Commit-hash from which the tka binary was build, should be set via ldflags -X
	Commit string
)

//nolint:golint-sl // CLI user output
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Shows version information",
		Args:  cobra.ExactArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Version: %s\n", Version)
			fmt.Printf("Date:    %s\n", Date)
			fmt.Printf("Commit:  %s\n", Commit)
		},
	}
}
