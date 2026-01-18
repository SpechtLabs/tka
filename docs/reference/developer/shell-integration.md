---
title: Shell Integration Details
createTime: 2025/09/07 23:34:59
permalink: /reference/developer/shell-integration
---

This documentation will go more in detail about the inner workings of the [Shell Integration](../../guides/shell-integration.md)

## Integration Details

### What the Wrapper Does

1. **Intercepts Commands**: Catches `tka login`, `tka refresh`, and `tka logout`
2. **Executes TKA**: Runs the actual TKA binary
3. **Parses Output**: Extracts environment variable exports
4. **Updates Environment**: Sets variables in your current shell
5. **Shows Status**: Displays success/failure messages

### Environment Variables Managed

- `KUBECONFIG` - Path to the temporary kubeconfig file
- Custom variables (if added in future versions)

### Temporary File Management

The wrapper tracks temporary kubeconfig files and cleans them up during:

- `tka logout` - Explicit cleanup
- Shell exit - Automatic cleanup (if supported by shell)
