---
title: CLI Autocompletion Setup
permalink: /guides/autocompletion
createTime: 2025/01/27 13:00:00

---

Set up tab autocompletion for the TKA CLI to get command suggestions, flag completion, and faster command line workflow.

## Overview

TKA includes built-in autocompletion support for all major shells. Once configured, you can use `Tab` to:

- Complete command names (`tka log<Tab>` → `tka login`)
- Complete subcommands (`tka get <Tab>` → shows `kubeconfig`, `login`)
- Complete flag names (`tka --<Tab>` → shows all available flags)
- Complete flag values (theme names, shell types, etc.)

## Supported Shells

- **Bash** (4.0+)
- **Zsh**
- **Fish**
- **PowerShell**

## Installation

:::: steps

1. ### Generate Completion Script

   TKA can generate completion scripts for your shell:

   ```bash
   tka completion [bash|zsh|fish|powershell]
   ```

2. ### Install Completion

   ::: tabs

   @tab bash

   Add to your `~/.bashrc`:

   ```bash
   # TKA completion
   source <(tka completion bash)
   ```

   @tab zsh

   Add to your `~/.zshrc`:

   ```zsh
   # TKA completion
   source <(tka completion zsh)

   # If you get compdef errors, add this before the source line:
   autoload -U compinit && compinit
   ```

   @tab fish

   Fish completions are automatically loaded from the right location:

   ```fish
   # Install completion
   $ tka completion fish > ~/.config/fish/completions/tka.fish

   # Reload Fish configuration
   $ source ~/.config/fish/config.fish
   ```

   @tab PowerShell

   ```powershell
   # Create profile directory if it doesn't exist
   PS> New-Item -Type Directory -Path (Split-Path $PROFILE) -Force

   # Add completion to your profile
   PS> tka completion powershell >> $PROFILE

   # Reload profile
   PS> . $PROFILE
   ```

   :::

::::

## Verification

Test that autocompletion is working:

```bash
# Type this and press Tab
$ tka <Tab>
generate completion get integration kubeconfig login reauthenticate shell signout version

# Test flag completion
$ tka --<Tab>
--config --debug --help --long --no-eval --port --server --theme --version

# Test subcommand completion
$ tka get <Tab>
kubeconfig login
```

## Troubleshooting

### Bash Completion Not Working

::: collapse expand

- **Problem**: Tab completion doesn't work after installation.

   **Solutions**:

   1. :- **1. Reload shell completely**

      ::: terminal Reload shell

      ```bash
      # Close and reopen terminal, or
      $ exec $SHELL
      ```

      :::

   2. :- **2. Check bash-completion package**

      ::: terminal Install bash-completion package

      ```bash
      # Ubuntu/Debian
      $ sudo apt install bash-completion

      # macOS with Homebrew
      $ brew install bash-completion

      # RHEL/CentOS
      $ sudo yum install bash-completion
      ```

      :::

:::

### Zsh Completion Issues

::: collapse

- **Problem**: `compdef: function definition file not found` errors.

   **Solution**: Ensure completion system is initialized:

   ```zsh
   # Add to ~/.zshrc before the completion source line
   autoload -U compinit && compinit
   ```

- **Problem**: Completions not loading in Oh My Zsh.

   **Solution**: Place completion in Oh My Zsh custom directory:

   ```zsh
   mkdir -p ~/.oh-my-zsh/custom/plugins/tka
   tka completion zsh > ~/.oh-my-zsh/custom/plugins/tka/_tka

   # Add 'tka' to plugins in ~/.zshrc
   plugins=(... tka)
   ```

:::

### Fish Completion Issues

::: collapse expand

- **Problem**: Completions not appearing.

   **Solutions**:

   1. :- **1. Check Fish version**:

      ```fish
      fish --version  # Requires Fish 3.0+
      ```

   2. :- **2. Verify completion file location**:

      ```fish
      ls ~/.config/fish/completions/tka.fish
      ```

   3. :- **3. Regenerate completions**:

      ```fish
      rm ~/.config/fish/completions/tka.fish
      tka completion fish > ~/.config/fish/completions/tka.fish
      ```

:::

### PowerShell Completion Issues

::: collapse

- **Problem**: Completion functions not available.

   **Solution**: Check execution policy:

   ```powershell
   # Check current policy
   Get-ExecutionPolicy

   # If restricted, allow scripts for current user
   Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
   ```

:::

## Advanced Configuration

### Custom Completion Behavior

You can customize completion behavior by modifying the generated script or using shell-specific configuration.

#### Bash: Case-Insensitive Completion

Add to `~/.inputrc`:

```bash
# Case-insensitive completion
set completion-ignore-case on

# Show all matches immediately
set show-all-if-ambiguous on

# Color completion matches
set colored-stats on
```

#### Zsh: Enhanced Completion Styling

Add to `~/.zshrc`:

```bash
# Better completion styling
zstyle ':completion:*' menu select
zstyle ':completion:*' list-colors "${(s.:.)LS_COLORS}"
zstyle ':completion:*' matcher-list 'm:{a-zA-Z}={A-Za-z}'

# Group completions by type
zstyle ':completion:*' group-name ''
zstyle ':completion:*:descriptions' format '%B%d%b'
```

#### Fish: Custom Completion Priorities

Fish automatically handles intelligent completion ordering, but you can customize behavior in `~/.config/fish/config.fish`:

```fish
# Customize completion paging
set -g fish_pager_color_completion cyan
set -g fish_pager_color_description yellow
set -g fish_pager_color_prefix blue
```

### Integration with Shell Wrapper

If you're using [shell integration](./shell-integration.md), autocompletion will work seamlessly with the `tka` wrapper function. The completion system understands both the wrapper and the underlying `ts-k8s-auth` binary.

## Updating Completions

When you update TKA, you may want to regenerate completion scripts to get the latest commands and flags:

```bash
# Bash
tka completion bash > ~/.local/share/bash-completion/completions/tka

# Zsh
tka completion zsh > ~/.zsh/completions/_tka

# Fish
tka completion fish > ~/.config/fish/completions/tka.fish

# PowerShell
tka completion powershell | Out-File -FilePath $PROFILE -Force
```

## What Gets Completed

TKA's autocompletion provides intelligent suggestions for:

### Commands

- `login` - Sign in and configure kubectl
- `get` - Retrieve resources
- `shell` - Start temporary subshell
- `generate integration` - Generate shell integration
- `completion` - Generate completion scripts
- `kubeconfig` - Fetch kubeconfig
- `reauthenticate` - Refresh authentication
- `signout` - Sign out and cleanup
- `version` - Show version information

### Subcommands

- `get kubeconfig` - Get current kubeconfig
- `get login` - Show login status

### Flags

- `--config, -c` - Configuration file
- `--debug` - Enable debug logging
- `--help, -h` - Show help
- `--long, -l` - Verbose output
- `--no-eval, -e` - Disable environment evaluation
- `--port, -p` - Server port
- `--server, -s` - Server address
- `--theme, -t` - Output theme
- `--version, -v` - Show version

### Values

- **Themes**: `ascii`, `dark`, `dracula`, `tokyo-night`, `light`, `notty`
- **Shells**: `bash`, `zsh`, `fish`, `powershell`
- **File paths**: Intelligent file/directory completion for config files

## Related Documentation

- [Shell Integration Setup](./shell-integration.md) - Automatic environment variable updates
- [CLI Reference](../reference/cli.md) - Complete command documentation
- [Getting Started](../getting-started/quick.md) - Basic TKA usage
