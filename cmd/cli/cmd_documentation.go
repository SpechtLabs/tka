package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const frontMatterTmpl = `---
title: {{ .FrontMatterTitle }}
permalink: {{ .FrontMatterPermalink }}
---`

var (
	useFrontMatter       bool
	frontMatterTitle     string
	frontMatterPermalink string
)

func init() {
	cmdDocumentation.Flags().BoolP("markdownlint-fix", "m", false, "Fix markdownlint errors")
	viper.SetDefault("output.markdownlint-fix", false)
	err := viper.BindPFlag("output.markdownlint-fix", cmdDocumentation.Flags().Lookup("markdownlint-fix"))
	if err != nil {
		panic(fmt.Errorf("fatal binding flag: %w", err))
	}

	cmdDocumentation.Flags().BoolVar(&useFrontMatter, "front-matter", false, "Use front matter")
	cmdDocumentation.Flags().StringVar(&frontMatterTitle, "title", "CLI Reference", "Title of the front matter")
	cmdDocumentation.Flags().StringVar(&frontMatterPermalink, "permalink", "/reference/cli", "Permalink of the front matter")
}

var cmdDocumentation = &cobra.Command{
	Use:    "documentation <path> [--markdownlint-fix] [--front-matter-title <title>] [--front-matter-permalink <permalink>]",
	Short:  "Generate the reference documentation for the tka CLI commands",
	Long:   `The documentation command generates the reference markdown documentation for the tka CLI commands.`,
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Example: `
	# Refresh the documentation into docs/reference/cli.md
	tka documentation docs/reference/cli.md

	# Refresh the documentation into docs/reference/cli.md and fix markdownlint errors
	tka documentation docs/reference/cli.md --markdownlint-fix

	# Refresh the documentation into docs/reference/cli.md and set the title and permalink
	tka documentation docs/reference/cli.md --front-matter-title "CLI Reference" --front-matter-permalink "/reference/cli"
	`,
	Run: func(cmd *cobra.Command, args []string) {
		rootCmd := getRootCmd(cmd)

		renderedHelp := renderReferenceHelp(rootCmd, 0)
		markdown := strings.Join(renderedHelp, "")

		if useFrontMatter {
			frontMatter := strings.ReplaceAll(frontMatterTmpl, "{{ .FrontMatterTitle }}", frontMatterTitle)
			frontMatter = strings.ReplaceAll(frontMatter, "{{ .FrontMatterPermalink }}", frontMatterPermalink)
			markdown = frontMatter + "\n\n" + markdown
		}

		// write the markdown to the file
		filePath := args[0]
		if err := os.WriteFile(filePath, []byte(markdown), 0644); err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		if viper.GetBool("output.markdownlint-fix") {
			proc := exec.Command("markdownlint-cli2", "--fix", filePath)
			if err := proc.Run(); err != nil {
				pretty_print.PrintError(err)
				os.Exit(1)
			}
		}
	},
}

func getRootCmd(cmd *cobra.Command) *cobra.Command {
	if cmd.Parent() == nil {
		return cmd
	}
	return getRootCmd(cmd.Parent())
}

// Since all cobra commands are structured in a tree, we can perform a simple in-order DFS to render the help text
func renderReferenceHelp(cmd *cobra.Command, depth int) []string {
	renderedHelp := make([]string, 0)

	// Hide hidden commands
	if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == "completion" {
		return renderedHelp
	}

	// In-Order traversal.
	// first print the current command
	helpText := pretty_print.FormatHelpText(cmd, []string{}, pretty_print.WithTheme(pretty_print.MarkdownStyle), pretty_print.WithoutNewline())
	helpText = fixHeadingLevels(helpText, depth)
	renderedHelp = append(renderedHelp, helpText)

	// then recurse on all the children
	for _, childCommand := range cmd.Commands() {
		renderedHelp = append(renderedHelp, renderReferenceHelp(childCommand, depth+1)...)
	}

	return renderedHelp
}

// fixHeadingLevels increases the heading level of markdown text by `depth`.
// Example: "# Title" with depth=1 -> "## Title"
func fixHeadingLevels(helpText string, depth int) string {
	if depth == 0 {
		depth = 1
	}

	lines := strings.Split(helpText, "\n")
	withinCodeBlock := false
	for i, line := range lines {
		if strings.HasPrefix(line, "```") {
			withinCodeBlock = !withinCodeBlock
		}

		// skip within code blocks
		if withinCodeBlock {
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Count how many '#' are at the start
			j := 0
			for j < len(line) && line[j] == '#' {
				j++
			}
			// Insert extra '#' only at the start
			lines[i] = strings.Repeat("#", j+depth) + line[j:]
		}
	}
	return strings.Join(lines, "\n")
}
