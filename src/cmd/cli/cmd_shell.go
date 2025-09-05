package main

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cmdShell = &cobra.Command{
	Use:   "shell",
	Short: "Start a subshell with temporary Kubernetes access via Tailscale identity",
	Long: `# Shell Command

The **shell** command authenticates you using your Tailscale identity and
retrieves a short‑lived Kubernetes access token. It then spawns an interactive
subshell (using your login shell, e.g. ` + "`bash` or `zsh`" + `") with the
` + "`KUBECONFIG`" + ` environment variable set to a temporary kubeconfig file.

This provides a clean and secure workflow:

- Your existing shell environment remains untouched.
- All Kubernetes operations inside the subshell use the temporary credentials.
- When you exit the subshell, the credentials are automatically revoked and
  the temporary kubeconfig file is deleted.

This is useful for administrators and developers who need ephemeral access to
a cluster without persisting credentials on disk or leaking them into their
long‑lived shell environment.`,

	Example: `# Start a subshell with temporary Kubernetes access
tka shell

# Inside the subshell, run kubectl commands as usual
kubectl get pods -n default

# When finished, exit the subshell
exit

# At this point, the temporary credentials are revoked automatically`,
	Args:      cobra.ExactArgs(0),
	ValidArgs: []string{},
	RunE:      forkShell,
}

func forkShell(cmd *cobra.Command, args []string) error {
	quiet := viper.GetBool("output.quiet")

	// 1. Login and get kubeconfig path
	kubeCfgPath, err := signIn(quiet)
	if err != nil {
		return err
	}

	// 2. Run subshell
	if err := runShell(kubeCfgPath); err != nil {
		return err
	}

	// 3. Cleanup after shell exits
	defer func() {
		_ = os.Remove(kubeCfgPath)
	}()

	return signOut(cmd, args)
}

func runShell(kubeconfig string) error {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	cmd := exec.Command(shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"KUBECONFIG="+kubeconfig,
		"PS1=(tka) "+os.Getenv("PS1"),
	)

	return cmd.Run()
}
