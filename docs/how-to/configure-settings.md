---
title: Configure TKA Settings
permalink: /how-to/configure-settings
createTime: 2025/01/27 15:00:00
---

Configure TKA CLI settings such as output themes, debug logging, and behavior preferences using the configuration system.

## Overview

TKA provides a configuration system similar to `git config` that allows you to:

- Set persistent preferences for output formatting
- Control debug and logging behavior
- Customize API retry settings
- Configure default values for common flags

Configuration values are stored in YAML format and can be managed through the CLI.

## Configuration File Location

TKA searches for configuration files in this order:

1. **Home config**: `~/.config/tka/config.yaml`
2. **System config**: `/etc/tka/config.yaml` (Linux/macOS)

The first file found is used. If no configuration file exists, TKA uses built-in defaults.

## Basic Usage

:::: steps

1. ### View Current Configuration

   Display all current settings:

   ```bash
   tka config
   ```

   Example output:

   ```yaml
   api:
       retryafterseconds: 1
   debug: false
   output:
       long: true
       markdownlint-fix: false
       quiet: false
       theme: tokyo-night
   ```

2. ### Get a Specific Setting

   Check the value of a specific configuration key:

   ```bash
   # Check current theme
   tka config output.theme

   # Check debug setting
   tka config debug
   ```

3. ### Set Configuration Values

   Change configuration settings:

   ```bash
   # Set output theme
   tka config output.theme dracula

   # Enable debug logging
   tka config debug true

   # Disable long output by default
   tka config output.long false
   ```

::::

## Common Configuration Tasks

### Change Output Theme

TKA supports multiple color themes for different terminal environments:

```bash
# Dark themes (good for dark terminals)
tka config output.theme tokyo-night  # Default, purple/blue accents
tka config output.theme dracula      # Vibrant colors
tka config output.theme dark         # Simple dark
tka config output.theme light        # Light theme (good for light terminals)
tka config output.theme ascii        # Plaintext theme (no colors, good for scripts/CI)

# Check current theme
tka config output.theme
```

### Control Output Verbosity

Adjust how much information TKA shows:

```bash
tka config output.long true   # Show detailed output by default
tka config output.long false  # Use concise output by default
tka config output.quiet true  # Suppress non-essential messages
tka config output.quiet false # Re-enable all output
```

### Enable Debug Logging

Useful for troubleshooting:

```bash
tka config debug true # Enable debug mode

# Test with debug output
tka login --no-eval --long # Shows detailed debug information

# Disable debug mode
tka config debug false
```

### Adjust API Behavior

Configure retry behavior for slow networks:

```bash
tka config api.retryafterseconds 3 # Increase retry delay for slow networks
tka config api.retryafterseconds 1 # Reset to default
```

## Configuration File Management

### Create Configuration File

If no configuration file exists, use `--force` to create one:

```bash
# Create config file and set a value
tka config output.theme dracula --force
```

This creates `~/.config/tka/config.yaml` with your setting.

### View Configuration File Location

See which configuration file is being used:

```bash
# Show config with filename
tka get config --filename

# Show only the config file path
tka get config --filename --quiet
```

## Troubleshooting

### Configuration Not Found

**Problem**: TKA doesn't seem to use your configuration

**Solutions**:

```bash
# Check which config file is being used
tka get config --filename

# Verify file exists
ls -la ~/.config/tka/config.yaml

# Check file permissions
cat ~/.config/tka/config.yaml
```

### Settings Not Persisting

**Problem**: Configuration changes don't persist

**Solutions**:

```bash
# Ensure you have write permissions
ls -la ~/.config/tka/

# Try creating config explicitly
tka config output.theme dracula --force

# Check if config file was created
tka get config --filename
```

### Multiple Configuration Files

**Problem**: Unsure which configuration file is active

**Solutions**:

```bash
# Show active config file
tka get config --filename

# List all possible config locations
ls -la ./config.yaml ~/.config/tka/config.yaml /etc/tka/config.yaml 2>/dev/null
```

## Related Documentation

- [Configuration Reference](../reference/configuration.md) - Complete list of all available settings and defaults
- [`tka config`](../reference/cli.md#usage-config) - View and set configuration command reference
- [`tka get config`](../reference/cli.md#usage-config-1) - View configuration with additional options
- [`tka set config`](../reference/cli.md#usage-config-2) - Alternative syntax for setting values

## Next Steps

- [Configure ACLs](./configure-acl.md) for user access control
- [Shell Integration Setup](./shell-integration.md) for seamless environment updates
- [Troubleshooting Guide](./troubleshooting.md) for common issues
