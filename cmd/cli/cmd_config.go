package main

import (
	"bytes"
	"fmt"
	"os"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdConfig.Flags().BoolP("force", "f", false, "Create config file at the lowest tier if no config file exists")
	cmdGetConfig.Flags().Bool("filename", false, "Show the filename of the config file used")
}

var cmdConfig = &cobra.Command{
	Use:   "config [key] [value] [--force]",
	Short: "Get or set configuration values",
	Long: `Get or set configuration values in the TKA configuration file.

This command works similarly to ` + "`git config --global`" + `:

- When called with no arguments, shows all current configuration
- When called with just a key, it shows the current value
- When called with key and value, it sets the configuration
- Configuration is written to the file that was used to load the current config
- If no config file exists and ` + "`--force`" + ` is used, creates ` + "`~/.config/tka/config.yaml`",

	Example: `# Show all current configuration
tka config

# Show the current debug setting
tka config debug

# Set output.markdownlint-fix to true
tka config output.markdownlint-fix true

# Create a config file and set a value (when no config exists)
tka config output.long true --force`,
	Args: cobra.RangeArgs(0, 2),
	Run:  runConfig,
}

var cmdSetConfig = &cobra.Command{
	Use:   "config <key> <value> [--force]",
	Short: "Set configuration values",
	Long: `Set configuration values in the TKA configuration file.

This command works similarly to ` + "`git config --global`" + `:

- When called with key and value, it sets the configuration
- Configuration is written to the file that was used to load the current config
- If no config file exists and ` + "`--force`" + ` is used, creates ` + "`~/.config/tka/config.yaml`",

	Example: `# Set the debug setting to true
tka config output.theme dark

# Set output.markdownlint-fix to true
tka config output.markdownlint-fix true

# Create a config file and set a value (when no config exists)
tka config output.long true --force`,
	Args: cobra.ExactArgs(2),
	Run:  runConfig,
}

var cmdGetConfig = &cobra.Command{
	Use:   "config [key]",
	Short: "Get configuration values",
	Long: `Get configuration values in the TKA configuration file.

This command works similarly to ` + "`git config --global`" + `:

- When called with no arguments, shows all current configuration
- When called with just a key, it shows the current value`,

	Example: `# Show all current configuration
$ tka get config
api:
    retryafterseconds: 1
debug: false
output:
    long: true
    markdownlint-fix: false
    quiet: false
    theme: dracula
...snip...

# Show the current theme setting
$ tka get config output.theme
tokyo-night

# Show the current theme setting and the filename of the config file used
$ go run ./cmd/cli get config output.theme
→ output.theme: dracula

# Show the current theme setting and only the value
$ go run ./cmd/cli get config output.theme --quiet
dracula

# Show the current theme setting and the filename of the config file used
$ go run ./cmd/cli get config output.theme --filename
→ output.theme: dracula
    Config file used: /Users/cedi/.config/tka/config.yaml

# Show the all configuration and the filename of the config file used
$ tka get config --filename
ℹ Config file used:
    /Users/cedi/.config/tka/config.yaml

api:
    retryafterseconds: 1
debug: false
output:
    long: true
    markdownlint-fix: false
    quiet: false
    theme: dracula
..snip...

# Show only the filepath of the config file used
$ tka get config --filename --quiet
/Users/cedi/.config/tka/config.yaml

# Invalid: combine --filename and --quiet when using  key
$ tka get config output.theme --filename --quiet
✗ Invalid: cannot combine --filename and --quiet when also specifying a [key]
`,
	Args: cobra.RangeArgs(0, 1),
	Run:  runConfig,
}

func runConfig(cmd *cobra.Command, args []string) {
	quiet := viper.GetBool("output.quiet")
	showFilename, err := cmd.Flags().GetBool("filename")
	if err != nil {
		showFilename = false
	}

	switch len(args) {
	// If no arguments provided, show all configuration
	case 0:
		showAllConfig(showFilename, quiet)
		return

	// If only one argument provided, show the current value
	case 1:
		if showFilename && quiet {
			pretty_print.PrintErrorMessage("Invalid: cannot combine --filename and --quiet when also specifying a [key]")
			os.Exit(1)
		}
		printConfigValue(args[0], showFilename, quiet)
		return

	// Two arguments provided, set the value
	case 2:
		key := args[0]
		value := args[1]
		forceCreate, err := cmd.Flags().GetBool("force")
		if err != nil {
			forceCreate = false
		}
		if err := setConfigValue(key, value, forceCreate); err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		} else if !quiet {
			pretty_print.PrintOk("Configuration updated", fmt.Sprintf("%s = %v", key, value))
		}
		return

	// Invalid number of arguments
	default:
		pretty_print.PrintErrorMessage("Invalid number of arguments")
		os.Exit(1)
	}
}

//nolint:golint-sl // CLI user output
func printConfigValue(key string, showFilename, quiet bool) {
	value := viper.Get(key)

	if quiet {
		if value != nil {
			fmt.Printf("%v\n", value)
		}
	} else {
		if value != nil {
			if showFilename {
				pretty_print.PrintInfoIcon("→", key+": "+fmt.Sprintf("%v", value), "Config file used: "+viper.ConfigFileUsed())
			} else {
				pretty_print.PrintInfoIcon("→", key+": "+fmt.Sprintf("%v", value))
			}
		} else {
			pretty_print.PrintInfo(fmt.Sprintf("Configuration key not set: %s", key))
		}

	}
}

func setConfigValue(key, value string, forceCreate bool) humane.Error {
	// Parse the value appropriately
	parsedValue := parseValue(value) //nolint:golint-sl // parsed early, used after validation

	// Get the config file that was used
	configFileUsed := viper.ConfigFileUsed()

	// If no config file was used and force is not set, show error
	if configFileUsed == "" && !forceCreate {
		return humane.New("No configuration file is used. Use --force to create one at ~/.config/tka/config.yaml", "run 'tka config set --force <key> <value>' to create a new config file")
	}

	// If forcing creation and no config file in use, create default path
	if configFileUsed == "" && forceCreate {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return humane.Wrap(err, "failed to determine home directory", "ensure $HOME is set")
		}
		configDir := fmt.Sprintf("%s/.config/tka", homeDir)
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return humane.Wrap(err, "failed to create config directory", "check permissions for ~/.config/")
		}
		configPath := fmt.Sprintf("%s/config.yaml", configDir)
		viper.SetConfigFile(configPath)
		// Create empty file if not exists
		if _, err := os.Stat(configPath); err != nil {
			if f, cErr := os.Create(configPath); cErr == nil {
				_ = f.Close()
			}
		}
	}

	// Set the value in viper
	viper.Set(key, parsedValue)

	// Write the configuration back to the file
	// viper will automatically create the file if it doesn't exist
	if err := viper.WriteConfig(); err != nil {
		return humane.Wrap(err, "failed to write config file", "check file permissions and disk space")
	}

	return nil
}

func parseValue(value string) any {
	// Try to parse as boolean
	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	// Return as string for everything else
	return value
}

//nolint:golint-sl // CLI user output
func showAllConfig(showFilename, quiet bool) {
	if showFilename {
		if quiet {
			fmt.Println(viper.ConfigFileUsed())
		} else {
			pretty_print.PrintInfo("Config file used:", viper.ConfigFileUsed())
		}
	}

	if !quiet {
		fmt.Println()
		// Get all configuration as a YAML string
		buf := new(bytes.Buffer)
		if err := viper.WriteConfigTo(buf); err != nil {
			pretty_print.PrintError(err)
			os.Exit(1)
		}

		yamlString := buf.String()
		fmt.Print(yamlString)
	}
}
