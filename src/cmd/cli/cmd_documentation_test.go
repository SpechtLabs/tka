package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixHeadingLevels(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		depth    int
	}{
		{
			name:  "root level",
			depth: 0,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
## Usage tka
### Description
#### Theming
#### Notes
`,
		},
		{
			name:  "first level",
			depth: 1,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
## Usage tka
### Description
#### Theming
#### Notes
`,
		},
		{
			name:  "third level",
			depth: 3,
			input: `
# Usage tka
## Description
### Theming
### Notes
`,
			expected: `
#### Usage tka
##### Description
###### Theming
###### Notes
`,
		},
		{
			name: "hashtag in the middle",
			input: `
# A title with #hashtag in the middle
Bla bla bla`,
			depth: 2,
			expected: `
### A title with #hashtag in the middle
Bla bla bla`,
		},
		{
			name: "do nothing within codeblock",
			input: `
# A title with #hashtag in the middle
Bla bla bla

` + "```md" + `
Code block
` + "```" + `

blafoo

` + "```shell" + `
# example
tka login

# different example
tka login --quiet

` + "```" + `

yay
`,
			depth: 2,
			expected: `
### A title with #hashtag in the middle
Bla bla bla

` + "```md" + `
Code block
` + "```" + `

blafoo

` + "```shell" + `
# example
tka login

# different example
tka login --quiet

` + "```" + `

yay
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := fixHeadingLevels(test.input, test.depth)
			require.Equal(t, test.expected, result)
		})
	}
}

// createTestCommand creates a test command with a specific structure for testing
func createTestCommand(t *testing.T, name, use, short string) *cobra.Command {
	t.Helper()
	return &cobra.Command{
		Use:   use,
		Short: short,
		Long:  "Long description for " + name,
		Run: func(cmd *cobra.Command, args []string) {
			// Test command - no implementation needed
		},
	}
}

// createTestCommandTree creates a test command tree structure for testing tree traversal
func createTestCommandTree(t *testing.T) *cobra.Command {
	t.Helper()

	// Create root command
	rootCmd := createTestCommand(t, "root", "testapp", "Test application")

	// Create first level commands
	level1Cmd1 := createTestCommand(t, "level1-1", "command1", "First level command 1")
	level1Cmd2 := createTestCommand(t, "level1-2", "command2", "First level command 2")

	// Create second level commands under level1Cmd1
	level2Cmd1 := createTestCommand(t, "level2-1", "subcmd1", "Second level command 1")
	level2Cmd2 := createTestCommand(t, "level2-2", "subcmd2", "Second level command 2")

	// Create third level command under level2Cmd1
	level3Cmd := createTestCommand(t, "level3", "deepcmd", "Third level command")

	// Build the tree structure
	rootCmd.AddCommand(level1Cmd1)
	rootCmd.AddCommand(level1Cmd2)
	level1Cmd1.AddCommand(level2Cmd1)
	level1Cmd1.AddCommand(level2Cmd2)
	level2Cmd1.AddCommand(level3Cmd)

	return rootCmd
}

func TestRenderReferenceHelp(t *testing.T) {
	tests := []struct {
		name           string
		setupTree      func(t *testing.T) *cobra.Command
		startingCmd    func(rootCmd *cobra.Command) *cobra.Command
		depth          int
		expectedOrder  []string // Expected command names in traversal order
		expectedDepths []int    // Expected depths for each command
	}{
		{
			name: "single command at root level",
			setupTree: func(t *testing.T) *cobra.Command {
				t.Helper()
				return createTestCommand(t, "single", "single", "Single command")
			},
			startingCmd: func(rootCmd *cobra.Command) *cobra.Command {
				return rootCmd
			},
			depth:          0,
			expectedOrder:  []string{"single"},
			expectedDepths: []int{0},
		},
		{
			name: "tree traversal from root",
			setupTree: func(t *testing.T) *cobra.Command {
				t.Helper()
				return createTestCommandTree(t)
			},
			startingCmd: func(rootCmd *cobra.Command) *cobra.Command {
				return rootCmd
			},
			depth:          0,
			expectedOrder:  []string{"testapp", "command1", "subcmd1", "deepcmd", "subcmd2", "command2"},
			expectedDepths: []int{0, 1, 2, 3, 2, 1},
		},
		{
			name: "tree traversal from first level command",
			setupTree: func(t *testing.T) *cobra.Command {
				t.Helper()
				return createTestCommandTree(t)
			},
			startingCmd: func(rootCmd *cobra.Command) *cobra.Command {
				// Find the "command1" command
				for _, cmd := range rootCmd.Commands() {
					if cmd.Name() == "command1" {
						return cmd
					}
				}
				t.Fatal("command1 not found")
				return nil
			},
			depth:          1,
			expectedOrder:  []string{"command1", "subcmd1", "deepcmd", "subcmd2"},
			expectedDepths: []int{1, 2, 3, 2},
		},
		{
			name: "tree traversal from second level command",
			setupTree: func(t *testing.T) *cobra.Command {
				t.Helper()
				return createTestCommandTree(t)
			},
			startingCmd: func(rootCmd *cobra.Command) *cobra.Command {
				// Find the "subcmd1" command (level 2)
				for _, level1Cmd := range rootCmd.Commands() {
					if level1Cmd.Name() == "command1" {
						for _, level2Cmd := range level1Cmd.Commands() {
							if level2Cmd.Name() == "subcmd1" {
								return level2Cmd
							}
						}
					}
				}
				t.Fatal("subcmd1 not found")
				return nil
			},
			depth:          2,
			expectedOrder:  []string{"subcmd1", "deepcmd"},
			expectedDepths: []int{2, 3},
		},
		{
			name: "commands with hidden commands filtered out",
			setupTree: func(t *testing.T) *cobra.Command {
				t.Helper()
				rootCmd := createTestCommand(t, "root", "testapp", "Test application")

				visibleCmd := createTestCommand(t, "visible", "visible", "Visible command")
				hiddenCmd := createTestCommand(t, "hidden", "hidden", "Hidden command")
				hiddenCmd.Hidden = true

				helpCmd := createTestCommand(t, "help", "help", "Help command")
				completionCmd := createTestCommand(t, "completion", "completion", "Completion command")

				rootCmd.AddCommand(visibleCmd)
				rootCmd.AddCommand(hiddenCmd)
				rootCmd.AddCommand(helpCmd)
				rootCmd.AddCommand(completionCmd)

				return rootCmd
			},
			startingCmd: func(rootCmd *cobra.Command) *cobra.Command {
				return rootCmd
			},
			depth:          0,
			expectedOrder:  []string{"testapp", "visible"}, // hidden, help, and completion should be filtered out
			expectedDepths: []int{0, 1},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup
			rootCmd := test.setupTree(t)
			startingCmd := test.startingCmd(rootCmd)

			// Execute
			results := renderReferenceHelp(startingCmd, test.depth)

			// Verify the correct number of commands were processed
			assert.Len(t, results, len(test.expectedOrder), "Number of rendered help sections should match expected commands")

			// Verify the order and depth of commands by checking the help text content
			for i, expectedCmd := range test.expectedOrder {
				// The help text should contain the command name
				assert.Contains(t, results[i], expectedCmd, "Help text %d should contain command name %s", i, expectedCmd)

				// Verify the heading depth by counting # characters at the start of Usage lines
				lines := strings.Split(results[i], "\n")
				for _, line := range lines {
					if strings.Contains(line, "Usage "+expectedCmd) || strings.Contains(line, "Usage testapp") {
						// Count leading # characters
						hashCount := 0
						for j := 0; j < len(line) && line[j] == '#'; j++ {
							hashCount++
						}
						expectedHashCount := test.expectedDepths[i] + 2 // +2 because fixHeadingLevels adds 1 and we expect ## for first level
						assert.Equal(t, expectedHashCount, hashCount, "Command %s at index %d should have %d hash characters but has %d", expectedCmd, i, expectedHashCount, hashCount)
						break
					}
				}
			}
		})
	}
}

func TestCmdDocumentationRun(t *testing.T) {
	tests := []struct {
		name                  string
		setupDocumentationCmd func(t *testing.T, rootCmd *cobra.Command) *cobra.Command
		args                  []string
		expectError           bool
		expectedFileExists    bool
		validateContent       func(t *testing.T, content string)
	}{
		{
			name: "documentation command at root level",
			setupDocumentationCmd: func(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
				t.Helper()
				rootCmd.AddCommand(cmdDocumentation)
				return cmdDocumentation
			},
			args:               []string{"test_output.md"},
			expectError:        false,
			expectedFileExists: true,
			validateContent: func(t *testing.T, content string) {
				t.Helper()
				assert.Contains(t, content, frontMatter, "Content should contain front matter")
				assert.Contains(t, content, "testapp", "Content should contain root command")
				assert.Contains(t, content, "visible", "Content should contain visible subcommand")
				assert.NotContains(t, content, "documentation", "Content should not contain hidden documentation command")
			},
		},
		{
			name: "documentation command two levels deep",
			setupDocumentationCmd: func(t *testing.T, rootCmd *cobra.Command) *cobra.Command {
				t.Helper()
				// Create intermediate command
				adminCmd := createTestCommand(t, "admin", "admin", "Admin commands")
				rootCmd.AddCommand(adminCmd)
				adminCmd.AddCommand(cmdDocumentation)
				return cmdDocumentation
			},
			args:               []string{"test_output_nested.md"},
			expectError:        false,
			expectedFileExists: true,
			validateContent: func(t *testing.T, content string) {
				t.Helper()
				assert.Contains(t, content, frontMatter, "Content should contain front matter")
				assert.Contains(t, content, "testapp", "Content should contain root command")
				assert.Contains(t, content, "admin", "Content should contain admin command")
				assert.Contains(t, content, "visible", "Content should contain visible subcommand")
				assert.NotContains(t, content, "documentation", "Content should not contain hidden documentation command")
			},
		},
	}

	// Create a temporary directory for test files
	tempDir := t.TempDir()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Setup test command tree
			rootCmd := createTestCommand(t, "root", "testapp", "Test application")
			visibleCmd := createTestCommand(t, "visible", "visible", "Visible command")
			rootCmd.AddCommand(visibleCmd)

			// Setup documentation command
			docCmd := test.setupDocumentationCmd(t, rootCmd)

			// Prepare file path in temp directory
			outputPath := filepath.Join(tempDir, test.args[0])
			args := []string{outputPath}

			// Execute the command
			docCmd.Run(docCmd, args)

			// Verify file exists if expected
			if test.expectedFileExists {
				_, err := os.Stat(outputPath)
				assert.NoError(t, err, "Output file should exist")

				// Read and validate content
				content, err := os.ReadFile(outputPath)
				require.NoError(t, err, "Should be able to read output file")

				if test.validateContent != nil {
					test.validateContent(t, string(content))
				}

				// Clean up the file
				if err := os.Remove(outputPath); err != nil {
					t.Errorf("Failed to remove output file: %v", err)
				}
			}
		})
	}
}
