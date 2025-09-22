//go:build unix

package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"

	"github.com/sierrasoftworks/humane-errors-go"
	"github.com/spechtlabs/tka/internal/cli/pretty_print"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	cmdRoot.AddCommand(cmdShell)
}

var cmdShell = &cobra.Command{
	Use:   "shell",
	Short: "Start a subshell with temporary Kubernetes access via Tailscale identity",
	Long: `# Shell Command

The **shell** command authenticates you using your Tailscale identity and
retrieves a short-lived Kubernetes access token. It then spawns an interactive
subshell (using your login shell, e.g. ` + "`bash` or `zsh`" + `") with the
` + "`KUBECONFIG`" + ` environment variable set to a temporary kubeconfig file.

This provides a clean and secure workflow:

- Your existing shell environment remains untouched.
- All Kubernetes operations inside the subshell use the temporary credentials.
- When you exit the subshell, the credentials are automatically revoked and
  the temporary kubeconfig file is deleted.

This is useful for administrators and developers who need ephemeral access to
a cluster without persisting credentials on disk or leaking them into their
long-lived shell environment.`,

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
	err = runShellWithContext(cmd.Context(), kubeCfgPath)

	// 3. Do cleanup
	cleanup(quiet, kubeCfgPath)
	return err
}

func cleanup(quiet bool, kubeCfgPath string) {
	var wg sync.WaitGroup

	// sign out (revoke credentials)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := signOut(nil, nil); err != nil {
			if !quiet {
				pretty_print.PrintError(humane.Wrap(err, "failed to sign out cleanly"))
			}
		}
	}()

	// remove temporary kubeconfig file
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := os.Remove(kubeCfgPath); err != nil && !quiet {
			pretty_print.PrintError(humane.Wrap(err, "failed to remove temporary kubeconfig"))
		}
	}()

	wg.Wait()
}

// runShellWithContext runs a subshell with the given kubeconfig, handling context cancellation
func runShellWithContext(ctx context.Context, kubeconfig string) error {
	// 1. Set up context for signal handling
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 2. Get default shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}

	// 3. Create the subshell
	cmd := exec.CommandContext(ctx, shell)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"KUBECONFIG="+kubeconfig,
		"PS1=(tka) "+os.Getenv("PS1"),
	)

	// 4. Forward signals to the child shell (for terminal control)
	setupShellSignalHandler(ctx, cancel, cmd)

	// 5. Run the subshell
	return cmd.Run()
}

// setupShellSignalHandler sets up signal handling for the shell command to ensure cleanup
func setupShellSignalHandler(ctx context.Context, cancel context.CancelFunc, cmd *exec.Cmd) {
	sigs := make(chan os.Signal, 1)
	// Hook ALL catchable signals for comprehensive cleanup
	// Note: SIGKILL (9) and SIGSTOP (19) cannot be caught by any process
	signal.Notify(sigs,
		// Standard termination signals
		syscall.SIGINT,  // Interrupt (Ctrl+C)
		syscall.SIGTERM, // Termination request
		syscall.SIGQUIT, // Quit (Ctrl+\)

		// Process control signals
		syscall.SIGHUP, // Hangup (terminal disconnect)

		// Error/fault signals
		syscall.SIGABRT, // Abort signal
		syscall.SIGALRM, // Alarm clock
		syscall.SIGFPE,  // Floating point exception
		syscall.SIGILL,  // Illegal instruction
		syscall.SIGPIPE, // Broken pipe
		syscall.SIGSEGV, // Segmentation violation
		syscall.SIGBUS,  // Bus error

		// Job control signals
		syscall.SIGTSTP, // Terminal stop (Ctrl+Z)
		syscall.SIGTTIN, // Background read from tty
		syscall.SIGTTOU, // Background write to tty

		// Misc signals
		syscall.SIGWINCH,  // Window resize
		syscall.SIGURG,    // Urgent condition on socket
		syscall.SIGXCPU,   // CPU time limit exceeded
		syscall.SIGXFSZ,   // File size limit exceeded
		syscall.SIGVTALRM, // Virtual alarm clock
		syscall.SIGPROF,   // Profiling alarm clock
	)

	go func() {
		defer signal.Stop(sigs)

		select {
		case <-ctx.Done():
			return

		case sig := <-sigs:
			switch sig {
			// === CLEANUP SIGNALS: Always trigger cleanup ===
			case syscall.SIGTERM:
				pretty_print.PrintErrorMessage("Received SIGTERM, cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)
			case syscall.SIGINT:
				pretty_print.PrintErrorMessage("Received interrupt signal (Ctrl+C), cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)
			case syscall.SIGQUIT:
				pretty_print.PrintErrorMessage("Received quit signal (Ctrl+\\), cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)
			case syscall.SIGHUP:
				pretty_print.PrintErrorMessage("Received hangup signal (terminal disconnect), cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)

			// === FATAL ERROR SIGNALS: Cleanup and exit immediately ===
			case syscall.SIGABRT:
				pretty_print.PrintErrorMessage("Received abort signal, emergency cleanup...")
				terminateChildAndCancel(cmd, cancel)
			case syscall.SIGFPE, syscall.SIGILL, syscall.SIGSEGV, syscall.SIGBUS:
				pretty_print.PrintErrorMessage("Received fatal error signal, emergency cleanup...")
				terminateChildAndCancel(cmd, cancel)
			case syscall.SIGALRM, syscall.SIGXCPU, syscall.SIGXFSZ, syscall.SIGVTALRM, syscall.SIGPROF:
				pretty_print.PrintErrorMessage("Received resource limit signal, cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)

			// === SPECIAL HANDLING SIGNALS ===
			case syscall.SIGWINCH:
				// Terminal resize - forward to child for proper display
				resizeChildTerm(cmd)

			case syscall.SIGTSTP:
				// Ctrl+Z - Handle stop signal specially
				if isProcessRunning(cmd) {
					pretty_print.PrintErrorMessage("Received stop signal (Ctrl+Z), suspending shell...")
					_ = cmd.Process.Signal(sig)
				}

			case syscall.SIGPIPE:
				// Broken pipe - usually means parent process died
				pretty_print.PrintErrorMessage("Received broken pipe signal, cleaning up TKA session...")
				terminateChildAndCancel(cmd, cancel)

			// === FORWARD TO CHILD SIGNALS ===
			case syscall.SIGTTIN, syscall.SIGTTOU, syscall.SIGURG:
				// TTY and socket signals - forward to child shell
				if isProcessRunning(cmd) {
					_ = cmd.Process.Signal(sig)
				}

			default:
				// Unknown signal
				pretty_print.PrintErrorMessage("Received unknown signal, ignoring...")
			}
		}
	}()
}

func resizeChildTerm(cmd *exec.Cmd) {
	if !isProcessRunning(cmd) {
		return
	}

	if fd := int(os.Stdin.Fd()); term.IsTerminal(fd) {
		if w, h, err := term.GetSize(fd); err == nil {
			// TIOCSWINSZ ioctl to set window size
			_ = unix.IoctlSetWinsize(int(cmd.Process.Pid), syscall.TIOCSWINSZ, &unix.Winsize{
				Row: uint16(h),
				Col: uint16(w),
			})
		}
	}
}

func isProcessRunning(cmd *exec.Cmd) bool {
	return cmd != nil && cmd.Process != nil && (cmd.ProcessState == nil || !cmd.ProcessState.Exited())
}

// terminateChildAndCancel properly terminates the child shell process and cancels the context
func terminateChildAndCancel(cmd *exec.Cmd, cancel context.CancelFunc) {
	// If no running process, just cancel
	if !isProcessRunning(cmd) {
		cancel()
		return
	}

	// Try graceful termination first
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		// SIGTERM failed, force kill immediately and cancel
		_ = cmd.Process.Kill()
		cancel()
		return
	}

	// Wait briefly for graceful termination
	gracefulTimeout := time.NewTimer(250 * time.Millisecond)
	defer gracefulTimeout.Stop()

	checkInterval := time.NewTicker(50 * time.Millisecond)
	defer checkInterval.Stop()

	for {
		select {
		case <-gracefulTimeout.C:
			// Timeout reached, force kill
			_ = cmd.Process.Kill()
			cancel()
			return

		case <-checkInterval.C:
			// Check if process has exited
			if !isProcessRunning(cmd) {
				// Process exited gracefully
				cancel()
				return
			}
		}
	}
}
