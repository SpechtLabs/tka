---
title: Shell Integration Setup
permalink: /how-to/shell-integration
createTime: 2025/01/27 10:00:00
---

Set up seamless shell integration so `tka login` automatically updates your shell environment without manual `export` commands.

## Overview

By default, TKA cannot modify your shell's environment variables because subprocesses cannot change the parent shell's state. Shell integration solves this by installing a wrapper function that intercepts TKA commands and automatically evaluates environment exports.

## Supported Shells

- **Bash**
- **Zsh**
- **Fish**
- **PowerShell**

## Installation

::: steps

1. ### Generate Integration Code

   First, generate the shell integration code for your shell:

   ```bash
   tka integration [bash|fish|powershell]
   ```

2. ### Install Integration

   ::: tabs

   @tab bash

   Add to your shell's configuration file:

   ```bash
   # add to ~/.bashrc or ~/.bash_profile
   eval "$(tka integration bash)"
   ```

   Then reload your shell:

   ```bash
   source ~/.bashrc
   ```

   @tab zsh

   Add to your shell's configuration file:

   ```zsh
   # add to ~/.zshrc
   eval "$(tka integration zsh)"
   ```

   Then reload your shell:

   ```zsh
   source ~/.zshrc
   ```

   @tab Fish

   Add to your Fish configuration:

   ```fish
   # Add to ~/.config/fish/config.fish
   tka integration fish | source
   ```

   Reload Fish:

   ```fish
   source ~/.config/fish/config.fish
   ```

   @tab PowerShell

   Add to your PowerShell profile:

   ```powershell
   # Find your profile location
   $PROFILE

   # Add this line to your profile
   tka integration powershell | Out-String | Invoke-Expression
   ```

   Reload PowerShell or run:

   ```powershell
   . $PROFILE
   ```

::::

## Usage After Integration

Once installed, TKA commands work seamlessly:

### Automatic Environment Updates

#### Before

```shell
# Before integration (manual)
$ tka login
 ✓ sign-in successful!
     ╭───────────────────────────────────────╮
     │ User:  alice@example.com              │
     │ Role:  cluster-admin                  │
     │ Until: Mon, 27 Jan 2025 18:30:00 CET  │
     ╰───────────────────────────────────────╯
 ✓ kubeconfig written to: /tmp/kubeconfig-123456.yaml
 → To use this session, run: export KUBECONFIG=/tmp/kubeconfig-123456.yaml

# Use the kubeconfig
$ export KUBECONFIG=/tmp/kubeconfig-123456.yaml

# Works only after manually exporting the KUBECONFIG
$ kubectl get pods
```

#### After

```shell
# After integration (automatic)
$ tka login
# Your KUBECONFIG is automatically set!

$ kubectl get pods  # Works immediately
```

::: note
The shell integration is roughly equal to

```bash
eval "$(command ts-k8s-auth login --quiet)"
```

To learn more, I encourage you to checkout the [source code](https://github.com/SpechtLabs/tka/blob/main/src/cmd/cli/cmd_integration.go#L82) or check out the [Developer Documentation: Shell Integration Details]

:::

### Supported Commands

The wrapper automatically handles these commands:

- `tka login` - Sets `$KUBECONFIG` after successful authentication
- `tka refresh` / `tka reauthenticate` - Updates `$KUBECONFIG` with refreshed credentials
- `tka logout` - Unsets `$KUBECONFIG` and cleans up temp files

### Bypass Integration

To see full output without environment updates:

```bash
tka login --no-eval
```

This shows the traditional output with manual export instructions.

## Troubleshooting

### Integration Not Working

1. **Verify Installation**:

   ```bash
   type tka  # Should show it's a function, not a binary
   ```

2. **Check Path**:

   ```bash
   which tka  # Should show the binary path
   ```

3. **Reload Shell**:

   ```bash
   source ~/.bashrc  # or your shell's config file
   ```

### Multiple TKA Installations

If you have multiple TKA binaries:

```bash
# Specify full path in integration
eval "$(/usr/local/bin/tka integration bash)"
```

### Permission Issues

Ensure the TKA binary is executable:

```bash
chmod +x /path/to/tka
```

### Environment Not Updating

1. **Check Integration Code**: Re-run the integration command to see if there are syntax errors
2. **Manual Test**: Try `tka login --no-eval` and manually export the KUBECONFIG
3. **Shell Compatibility**: Ensure you're using a supported shell version

### Fish-Specific Issues

Fish uses different syntax. If you see errors:

```fish
# Check Fish version
fish --version

# Verify integration code works
tka integration fish | fish_indent
```

### PowerShell-Specific Issues

For PowerShell execution policy issues:

```powershell
# Check execution policy
Get-ExecutionPolicy

# If needed, set policy (run as administrator)
Set-ExecutionPolicy RemoteSigned
```

## Advanced Configuration

### Custom Wrapper Location

Store the wrapper in a custom location:

```bash
# Generate and save wrapper
tka integration bash > ~/.local/share/tka/wrapper.sh

# Source from your shell config
source ~/.local/share/tka/wrapper.sh
```

### Conditional Loading

Load integration only when TKA is available:

```bash
# In ~/.bashrc or ~/.zshrc
if command -v tka >/dev/null 2>&1; then
    eval "$(tka integration bash)"  # or zsh
fi
```

### Multiple Environments

For different TKA configurations per project:

```bash
# Project-specific wrapper
alias tka-prod='TKA_TAILSCALE_TAILNET=prod.ts.net tka'
alias tka-dev='TKA_TAILSCALE_TAILNET=dev.ts.net tka'
```

## Uninstalling Integration

### Remove from Shell Config

1. **Edit your shell configuration file**
2. **Remove or comment out the integration line**:

   ```bash
   # eval "$(tka integration bash)"
   ```

3. **Reload your shell**

### Clean Up Temporary Files

```bash
# Remove any remaining temp kubeconfig files
rm -f /tmp/kubeconfig-*.yaml

# Unset environment variables
unset KUBECONFIG
```

## Security Considerations

- The wrapper only evaluates specific TKA output patterns
- Temporary kubeconfig files are created with restricted permissions (600)
- Integration code can be reviewed before installation
- No sensitive data is stored in shell history

## Next Steps

- [Configure ACLs](./configure-acl.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [Production Deployment](./deploy-production.md)

## Related Documentation

- [Developer Documentation: Shell Integration Details]
- [Use a subshell with ephemeral access](./use-subshell.md)

[Developer Documentation: Shell Integration Details]: ../reference/developer/shell-integration.md
