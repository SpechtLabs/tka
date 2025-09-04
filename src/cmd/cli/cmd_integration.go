package main

import (
	"fmt"
	"strings"

	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
)

var cmdIntegration = &cobra.Command{
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Use:       "integration <bash|zsh|fish|powershell>",
	Short:     "Generate shell integration for tka wrapper",
	Example: `# For bash or zsh, add this line to your ~/.bashrc or ~/.zshrc:
eval "$(ts-k8s-auth shell bash)"

# For fish, add this line to your ~/.config/fish/config.fish:
ts-k8s-auth shell fish | source

# For PowerShell, add this line to your profile (e.g. $PROFILE):
ts-k8s-auth shell powershell | Out-String | Invoke-Expression
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

` + "```bash" + `
tka login        # signs in and updates your environment
tka refresh      # refreshes credentials and updates your environment
tka logout       # signs out
` + "```" + `

If you want to bypass the automatic environment updates and see the full
human-friendly output, you can pass the "--no-eval" flag:

` + "```bash" + `
tka login --no-eval
` + "```" + `

This command only prints the integration code. You must eval or source it
in your shell for it to take effect.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := strings.ToLower(args[0])
		switch shell {
		case "bash":
			fmt.Println("# Add the following line to your ~/.bashrc:")
			fmt.Println("#    eval \"$(ts-k8s-auth shell bash)\"")
			fmt.Println(getBashShell())
		case "zsh":
			fmt.Println("# Add the following line to your ~/.zshrc:")
			fmt.Println("#    eval \"$(ts-k8s-auth shell zsh)\"")
			fmt.Println(getZshShell())
		case "fish":
			fmt.Println("# Add the following line to your ~/.config/fish/config.fish:")
			fmt.Println("#   ts-k8s-auth shell fish | source")
			fmt.Println(getFishShell())
		case "powershell":
			fmt.Println("# Add the following line to your PowerShell profile:")
			fmt.Println("#   ts-k8s-auth shell powershell | Out-String | Invoke-Expression")
			fmt.Println(getPowerShell())
		default:
			pretty_print.PrintErrorMessage("Unsupported shell: " + shell)
		}

		return nil
	},
}

func getBashShell() string {
	return `tka() {
    local eval_cmds=("login" "refresh" "kubeconfig")

    local cmd="$1"
    shift || true

    local should_eval=false
    for ec in "${eval_cmds[@]}"; do
        if [[ "$cmd" == "$ec" ]]; then
            should_eval=true
            break
        fi
    done

    if [[ "$should_eval" == false ]]; then
        command ts-k8s-auth "$cmd" "$@"
        return
    fi

    local disable_flags=("--no-eval" "--help" "--long")
    local no_eval=false
    for arg in "$@"; do
        for df in "${disable_flags[@]}"; do
            if [[ "$arg" == "$df" ]]; then
                no_eval=true
                break 2
            fi
        done
    done

    if $no_eval; then
        command ts-k8s-auth "$cmd" "$@"
    else
        eval "$(command ts-k8s-auth "$cmd" --quiet "$@")"
    fi
}
	`
}

func getFishShell() string {
	return `function tka
    set eval_cmds login refresh "get kubeconfig"
    set disable_flags --no-eval --help --long

    if test (count $argv) -eq 0
        command ts-k8s-auth
        return
    end

    set cmd $argv[1]
    set args $argv[2..-1]

    set should_eval false
    for ec in $eval_cmds
        if test "$cmd" = "$ec"
            set should_eval true
            break
        end
    end

    if test "$cmd" = "get"
        if test (count $args) -ge 1 -a "$args[1]" = "kubeconfig"
            set should_eval true
        end
    end

    if test "$should_eval" = false
        command ts-k8s-auth $cmd $args
        return
    end

    set no_eval false
    for arg in $args
        for df in $disable_flags
            if test "$arg" = "$df"
                set no_eval true
                break
            end
        end
        if test "$no_eval" = true
            break
        end
    end

    if test "$no_eval" = true
        command ts-k8s-auth $cmd $args
    else
        eval (command ts-k8s-auth $cmd --quiet $args)
    end
end
	`
}

func getZshShell() string {
	return getBashShell()
}

func getPowerShell() string {
	return `function tka {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Args
    )

    $evalCmds = @("login", "refresh")
    $disableFlags = @("--no-eval", "--help", "--long")

    if ($Args.Count -eq 0) {
        & ts-k8s-auth
        return
    }

    $cmd = $Args[0]
    $rest = $Args[1..($Args.Count - 1)]

    $shouldEval = $false
    if ($evalCmds -contains $cmd) {
        $shouldEval = $true
    }

    if ($cmd -eq "get" -and $rest.Count -ge 1 -and $rest[0] -eq "kubeconfig") {
        $shouldEval = $true
    }

    if (-not $shouldEval) {
        & ts-k8s-auth @Args
        return
    }

    $noEval = $false
    foreach ($arg in $rest) {
        if ($disableFlags -contains $arg) {
            $noEval = $true
            break
        }
    }

    if ($noEval) {
        & ts-k8s-auth @Args
    }
    else {
        $output = & ts-k8s-auth $Args[0] --quiet @($rest)
        Invoke-Expression $output
    }
}
	`
}
