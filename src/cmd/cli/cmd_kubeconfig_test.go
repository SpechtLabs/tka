package main

import (
	"runtime"
	"testing"
)

func TestDetectShell(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		setup       func(t *testing.T)
		cleanup     func(t *testing.T)
		expected    shellType
		description string
	}{
		{
			name: "detect_bash_from_shell_env",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "/bin/bash")
				clearShellSpecificEnvs(t)
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellBash,
			description: "should detect bash when SHELL env is set to bash",
		},
		{
			name: "detect_zsh_from_shell_env",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "/usr/bin/zsh")
				clearShellSpecificEnvs(t)
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellZsh,
			description: "should detect zsh when SHELL env is set to zsh",
		},
		{
			name: "detect_fish_from_shell_env",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "/usr/local/bin/fish")
				clearShellSpecificEnvs(t)
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellFish,
			description: "should detect fish when SHELL env is set to fish",
		},
		{
			name: "detect_fish_from_fish_version",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "")
				t.Setenv("FISH_VERSION", "3.6.1")
				clearOtherShellEnvs(t, "FISH_VERSION")
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellFish,
			description: "should detect fish when FISH_VERSION is set",
		},
		{
			name: "detect_zsh_from_zsh_version",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "")
				t.Setenv("ZSH_VERSION", "5.8")
				clearOtherShellEnvs(t, "ZSH_VERSION")
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellZsh,
			description: "should detect zsh when ZSH_VERSION is set",
		},
		{
			name: "detect_bash_from_bash_version",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "")
				t.Setenv("BASH_VERSION", "5.1.16")
				clearOtherShellEnvs(t, "BASH_VERSION")
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellBash,
			description: "should detect bash when BASH_VERSION is set",
		},
		{
			name: "detect_powershell_from_distribution_channel",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "")
				t.Setenv("POWERSHELL_DISTRIBUTION_CHANNEL", "MSI:Windows 10")
				clearOtherShellEnvs(t, "POWERSHELL_DISTRIBUTION_CHANNEL")
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellPowerShell,
			description: "should detect PowerShell when POWERSHELL_DISTRIBUTION_CHANNEL is set",
		},
		{
			name: "fallback_to_bash_when_unknown",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "")
				clearShellSpecificEnvs(t)
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellBash,
			description: "should fallback to bash when no shell is detected",
		},
		{
			name: "shell_env_takes_precedence",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("SHELL", "/bin/bash")
				t.Setenv("ZSH_VERSION", "5.8")
				t.Setenv("FISH_VERSION", "3.6.1")
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellBash,
			description: "should prefer SHELL env variable over version-specific env vars",
		},
	}

	// Skip Windows-specific test on non-Windows platforms
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name        string
			setup       func(t *testing.T)
			cleanup     func(t *testing.T)
			expected    shellType
			description string
		}{
			name: "detect_powershell_on_windows",
			setup: func(t *testing.T) {
				t.Helper()
				// On Windows, we always assume PowerShell regardless of env vars
			},
			cleanup:     func(t *testing.T) { t.Helper() },
			expected:    shellPowerShell,
			description: "should detect PowerShell on Windows platform",
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			// Setup test environment
			tt.setup(t)
			defer tt.cleanup(t)

			// Run test
			got := detectShell()
			if got != tt.expected {
				t.Errorf("detectShell() = %v, want %v. %s", got, tt.expected, tt.description)
			}
		})
	}
}

func TestGenerateExportStatement(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		fileName string
		shell    shellType
		expected string
	}{
		{
			name:     "bash_export",
			fileName: "/tmp/kubeconfig-123.yaml",
			shell:    shellBash,
			expected: "export KUBECONFIG=/tmp/kubeconfig-123.yaml",
		},
		{
			name:     "zsh_export",
			fileName: "/tmp/kubeconfig-456.yaml",
			shell:    shellZsh,
			expected: "export KUBECONFIG=/tmp/kubeconfig-456.yaml",
		},
		{
			name:     "fish_export",
			fileName: "/tmp/kubeconfig-789.yaml",
			shell:    shellFish,
			expected: "set -gx KUBECONFIG /tmp/kubeconfig-789.yaml",
		},
		{
			name:     "powershell_export",
			fileName: "C:\\temp\\kubeconfig-abc.yaml",
			shell:    shellPowerShell,
			expected: "$env:KUBECONFIG = \"C:\\temp\\kubeconfig-abc.yaml\"",
		},
		{
			name:     "unknown_shell_fallback",
			fileName: "/tmp/kubeconfig-def.yaml",
			shell:    shellUnknown,
			expected: "export KUBECONFIG=/tmp/kubeconfig-def.yaml",
		},
		{
			name:     "path_with_spaces",
			fileName: "/tmp/my config/kubeconfig-spaces.yaml",
			shell:    shellBash,
			expected: "export KUBECONFIG=/tmp/my config/kubeconfig-spaces.yaml",
		},
		{
			name:     "fish_path_with_spaces",
			fileName: "/tmp/my config/kubeconfig-spaces.yaml",
			shell:    shellFish,
			expected: "set -gx KUBECONFIG /tmp/my config/kubeconfig-spaces.yaml",
		},
		{
			name:     "powershell_path_with_spaces",
			fileName: "C:\\temp\\my config\\kubeconfig-spaces.yaml",
			shell:    shellPowerShell,
			expected: "$env:KUBECONFIG = \"C:\\temp\\my config\\kubeconfig-spaces.yaml\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			got := generateExportStatement(tt.fileName, tt.shell)
			if got != tt.expected {
				t.Errorf("generateExportStatement(%q, %v) = %q, want %q",
					tt.fileName, tt.shell, got, tt.expected)
			}
		})
	}
}

// clearShellSpecificEnvs clears all shell-specific environment variables
func clearShellSpecificEnvs(t *testing.T) {
	t.Helper()
	t.Setenv("FISH_VERSION", "")
	t.Setenv("ZSH_VERSION", "")
	t.Setenv("BASH_VERSION", "")
	t.Setenv("POWERSHELL_DISTRIBUTION_CHANNEL", "")
}

// clearOtherShellEnvs clears all shell-specific environment variables except the specified one
func clearOtherShellEnvs(t *testing.T, keep string) {
	t.Helper()
	envVars := []string{"FISH_VERSION", "ZSH_VERSION", "BASH_VERSION", "POWERSHELL_DISTRIBUTION_CHANNEL"}
	for _, env := range envVars {
		if env != keep {
			t.Setenv(env, "")
		}
	}
}
