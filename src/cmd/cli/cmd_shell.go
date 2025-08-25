package main

import (
	"fmt"
	"strings"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
)

var cmdShell = &cobra.Command{
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Use:       "shell [bash|zsh|fish|powershell]",
	Short:     "Generate shell integration for tka wrapper",
	Example: `# For bash or zsh, add this line to your ~/.bashrc or ~/.zshrc:
$ eval "$(ts-k8s-auth shell bash)"

# For fish, add this line to your ~/.config/fish/config.fish:
$ ts-k8s-auth shell fish | source

# For PowerShell, add this line to your profile (e.g. $PROFILE):
$ ts-k8s-auth shell powershell | Out-String | Invoke-Expression
`,
	Long: `The "shell" command generates shell integration code for the tka wrapper.

By default, the ts-k8s-auth binary cannot directly modify your shell's
environment variables (such as "${KUBECONFIG}"), because a subprocess cannot
change the parent shell's state. To work around this, tka provides a
wrapper function that you can install into your shell. This wrapper
intercepts certain commands (like "login" and "refresh") and automatically
evaluates the environment variable exports in your current shell session.

This makes commands like "tka login" feel seamless: your session is
authenticated and your "${KUBECONFIG}" is updated without needing to manually
copy and paste an "export" command.

Once installed, you can use "tka" as your entrypoint:
  $ tka login        # signs in and updates your environment
  $ tka refresh      # refreshes credentials and updates your environment
  $ tka logout       # signs out

If you want to bypass the automatic environment updates and see the full
human-friendly output, you can pass the "--no-eval" flag:
  $ tka login --no-eval

This command only prints the integration code. You must eval or source it
in your shell for it to take effect.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := strings.ToLower(args[0])
		switch shell {
		case "bash":
			fmt.Println(getBashShell())
		case "zsh":
			fmt.Println(getZshShell())
		case "fish":
			fmt.Println(getFishShell())
		case "powershell":
			fmt.Println(getPowerShell())
		default:
			pretty_print.PrintErrorMessage("Unsupported shell: " + shell)
		}

		return nil
	},
}

func getBashShell() string {
	return `
# Add the following line to your ~/.bashrc or ~/.zshrc:
#   eval "$(ts-k8s-auth shell bash)"
#
# This will install the tka wrapper function that automatically
# evals login/refresh commands into your current shell.
tka() {
    case "$1" in
        login|refresh)
            shift
            if [[ " $* " == *" --no-eval "* ]]; then
                command ts-k8s-auth "$1" "$@"
            else
                eval "$($(command -v ts-k8s-auth) "$1" --quiet "$@")"
            fi
            ;;
        *)
            command ts-k8s-auth "$@"
            ;;
    esac
}
	`
}

func getFishShell() string {
	return `
# Add the following line to your ~/.config/fish/config.fish:
#   ts-k8s-auth shell fish | source
function tka
    set cmd $argv[1]
    switch $cmd
        case login refresh
            set -e argv[1]
            if contains -- --no-eval $argv
                command ts-k8s-auth $cmd $argv
            else
                eval (ts-k8s-auth $cmd --quiet $argv)
            end
        case '*'
            command ts-k8s-auth $argv
    end
end
	`
}

func getZshShell() string {
	return getBashShell()
}

func getPowerShell() string {
	return `
# Add the following line to your PowerShell profile:
#   ts-k8s-auth shell powershell | Out-String | Invoke-Expression
function tka {
    param([string]$cmd, [Parameter(ValueFromRemainingArguments=$true)]$args)
    switch ($cmd) {
        "login" { & ts-k8s-auth $cmd --quiet @args | Invoke-Expression }
        "refresh" { & ts-k8s-auth $cmd --quiet @args | Invoke-Expression }
        default { & ts-k8s-auth $cmd @args }
    }
}
	`
}
