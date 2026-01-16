package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/sierrasoftworks/humane-errors-go"
	async_op2 "github.com/spechtlabs/tka/internal/cli/async_operation"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	tkaApi "github.com/spechtlabs/tka/pkg/service/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

var cmdKubeconfig = &cobra.Command{
	Use:   "kubeconfig [--quiet|-q] [--no-eval|-e]",
	Short: "Fetch your temporary kubeconfig",
	Long: `Retrieve an ephemeral kubeconfig for your current session and save it to a temporary file.
This command downloads the kubeconfig from the TKA server and writes it to a temp file.
It also sets KUBECONFIG for this process so that subsequent kubectl calls from this process
use the new file.
To update your interactive shell, export KUBECONFIG yourself`,
	Example: `# Fetch and save your current ephemeral kubeconfig
tka kubeconfig
tka get kubeconfig
`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	RunE:      getKubeconfig,
}

func getKubeconfig(_ *cobra.Command, _ []string) error {
	quiet := viper.GetBool("output.quiet")
	kubecfg, err := fetchKubeConfig(quiet)
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	file, err := serializeKubeconfig(kubecfg)
	if err != nil {
		pretty_print.PrintError(err)
		os.Exit(1)
	}

	printUseStatement(file, quiet)

	return nil
}

func fetchKubeConfig(quiet bool) (*api.Config, humane.Error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollFunc := func() (api.Config, humane.Error) {
		if cfg, _, err := doRequestAndDecode[api.Config](http.MethodGet, tkaApi.KubeconfigApiRoute, nil, http.StatusOK); err == nil {
			return *cfg, nil
		} else {
			return api.Config{}, err
		}
	}

	operation := async_op2.NewSpinner[api.Config](pollFunc,
		async_op2.WithInProgressMessage("Waiting for kubeconfig to be ready..."),
		async_op2.WithDoneMessage("Kubeconfig is ready."),
		async_op2.WithFailedMessage("Fetching kubeconfig failed."),
		async_op2.WithQuiet(quiet),
	)

	result, err := operation.Run(ctx)
	if err != nil {
		return nil, humane.Wrap(err, "failed to fetch kubeconfig", "ensure you are signed in and the TKA server is reachable")
	}
	return result, nil
}

func serializeKubeconfig(kubecfg *api.Config) (string, humane.Error) {
	out, err := clientcmd.Write(*kubecfg)
	if err != nil {
		return "", humane.Wrap(err, "failed to serialize kubeconfig", "this is likely a bug; please report it")
	}

	tempFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return "", humane.Wrap(err, "failed to create temp kubeconfig", "check you have write permissions to the temp directory")
	}
	defer func() { _ = tempFile.Close() }()

	_, err = tempFile.Write(out)
	if err != nil {
		return "", humane.Wrap(err, "failed to write temp kubeconfig", "check disk space and permissions")
	}

	if err := os.Setenv("KUBECONFIG", tempFile.Name()); err != nil {
		return "", humane.Wrap(err, "failed to set KUBECONFIG", "check environment variable permissions")
	}

	return tempFile.Name(), nil
}

// shellType represents supported shell types
type shellType string

const (
	shellBash       shellType = "bash"
	shellZsh        shellType = "zsh"
	shellFish       shellType = "fish"
	shellPowerShell shellType = "powershell"
	shellUnknown    shellType = "unknown"
)

// detectShell attempts to detect the current shell type
func detectShell() shellType {
	// Check if we're on Windows - assume PowerShell
	if runtime.GOOS == "windows" {
		return shellPowerShell
	}

	// Check SHELL environment variable first
	shell := os.Getenv("SHELL")
	if shell != "" {
		shellName := filepath.Base(shell)
		switch shellName {
		case "bash":
			return shellBash
		case "zsh":
			return shellZsh
		case "fish":
			return shellFish
		}
	}

	// Check if we're in PowerShell on non-Windows (e.g., PowerShell Core)
	if psHome := os.Getenv("POWERSHELL_DISTRIBUTION_CHANNEL"); psHome != "" {
		return shellPowerShell
	}

	// Check for Fish-specific environment variables
	if fishVersion := os.Getenv("FISH_VERSION"); fishVersion != "" {
		return shellFish
	}

	// Check for Zsh-specific environment variables
	if zshVersion := os.Getenv("ZSH_VERSION"); zshVersion != "" {
		return shellZsh
	}

	// Check for Bash-specific environment variables
	if bashVersion := os.Getenv("BASH_VERSION"); bashVersion != "" {
		return shellBash
	}

	// Fallback: assume bash/zsh (POSIX-compatible)
	return shellBash
}

// generateExportStatement creates the appropriate export statement for the detected shell
func generateExportStatement(fileName string, shell shellType) string {
	switch shell {
	case shellFish:
		return fmt.Sprintf("set -gx KUBECONFIG %s", fileName)
	case shellPowerShell:
		return fmt.Sprintf("$env:KUBECONFIG = \"%s\"", fileName)
	case shellBash, shellZsh, shellUnknown:
		fallthrough
	default:
		// Default to POSIX-compatible export for bash, zsh, and unknown shells
		return fmt.Sprintf("export KUBECONFIG=%s", fileName)
	}
}

//nolint:golint-sl // CLI user output
func printUseStatement(fileName string, quiet bool) {
	shell := detectShell()
	useStatement := generateExportStatement(fileName, shell)

	if quiet {
		fmt.Println(useStatement)
	} else {
		pretty_print.PrintOk("kubeconfig written to:", fileName)
		pretty_print.PrintInfoIcon("→", "To use this session, run:", useStatement)
	}
}
