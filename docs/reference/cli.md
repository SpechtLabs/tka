---
title: CLI Reference
permalink: /reference/cli
createTime: 2025/08/28 23:12:22
---


## Usage `tka`

```bash
tka [command]
```

### Description

tka is the client for Tailscale Kubernetes Auth. It lets you authenticate to clusters over Tailscale, manage kubeconfig entries, and inspect status with readable, themed output.

#### Theming

Control the CLI's look and feel using one of the following:

- Flag: `--theme` or `-t`
- Config: `theme` (in config file)
- Environment: `TKA_THEME`

**Accepted themes**: ascii, dark, dracula, *tokyo-night*, light

#### Notes

- Global flags like `--theme` are available to subcommands

### Examples

```bash
# generic dark theme
$ tka --theme dark login

# light theme
TKA_THEME=light tka kubeconfig

# no theme (usefull in non-interactive contexts)
$ tka --theme notty login

```

### Available Commands

> [!TIP]
> Use `tka [command] --help` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|
| **`completion`** | Generate the autocompletion script for the specified shell |
| **`get`** | Retrieve read-only resources from TKA. |
| **`kubeconfig`** | Fetch your temporary kubeconfig |
| **`login`** | Sign in and configure kubectl with temporary access |
| **`reauthenticate`** | Reauthenticate and configure kubectl with temporary access |
| **`shell`** | Generate shell integration for tka wrapper |
| **`signout`** | Sign out and remove access from the cluster |
| **`version`** | Shows version information |

### Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `get`

```bash
tka get [command]
```

### Description

The get command retrieves resources from your Tailscale Kubernetes Auth service

### Examples

```bash
# Fetch your current kubeconfig
tka get kubeconfig

# Show current login information
tka get login
```

### Available Commands

> [!TIP]
> Use `tka get [command] --help` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|
| **`kubeconfig`** | Fetch your temporary kubeconfig |
| **`login`** | Show current login information and provisioning status. |

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

### Usage `kubeconfig`

```bash
tka get kubeconfig [--quiet|-q] [--no-eval|-e]
```

#### Description

Retrieve an ephemeral kubeconfig for your current session and save it to a temporary file.
This command downloads the kubeconfig from the TKA server and writes it to a temp file.
It also sets KUBECONFIG for this process so that subsequent kubectl calls from this process
use the new file.
To update your interactive shell, export KUBECONFIG yourself

#### Examples

```bash
# Fetch and save your current ephemeral kubeconfig
tka kubeconfig
tka get kubeconfig

```

#### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

### Usage `login`

```bash
tka get login
```

#### Aliases

- `login`, `signin`

#### Description

Display details about your current login state, including whether provisioning was successful.
This does not modify your session.

#### Examples

```bash
# Display current login status information
tka get login
```

#### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `kubeconfig`

```bash
tka get kubeconfig [--quiet|-q] [--no-eval|-e] [flags]
```

### Description

Retrieve an ephemeral kubeconfig for your current session and save it to a temporary file.
This command downloads the kubeconfig from the TKA server and writes it to a temp file.
It also sets KUBECONFIG for this process so that subsequent kubectl calls from this process
use the new file.
To update your interactive shell, export KUBECONFIG yourself

### Examples

```bash
# Fetch and save your current ephemeral kubeconfig
tka kubeconfig
tka get kubeconfig

```

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `login`

```bash
tka login [--quiet|-q] [--long|-l|--no-eval|-e]
```

### Aliases

- `login`, `signin`, `auth`

### Description

Authenticate using your Tailscale identity and retrieve a temporary
Kubernetes access token. This command automatically fetches your kubeconfig,
writes it to a temporary file, sets the KUBECONFIG environment variable.

### Examples

```bash
# Sign in with user friendly output
tka login --no-eval

# Login and start using your session
tka login
kubectl get pods
```

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `reauthenticate`

```bash
tka reauthenticate [--quiet|-q] [--long|-l|--no-eval|-e]
```

### Aliases

- `reauthenticate`, `reauth`, `refresh`

### Description

Reauthenticate by signing out and then signing in again to refresh your temporary access.
This command is a convenience wrapper which:

1. Calls signout to revoke your current session
2. Calls login to obtain a fresh ephemeral kubeconfig

### Examples

```bash
# Reauthenticate and see human-friendly output
tka reauthenticate --no-eval

# Reauthenticate and update your current shell's KUBECONFIG
tka reauthenticate
```

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `shell`

```bash
tka shell <bash|zsh|fish|powershell>
```

### Description

The "shell" command generates shell integration code for the tka wrapper.

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

```bash
tka login        # signs in and updates your environment
tka refresh      # refreshes credentials and updates your environment
tka logout       # signs out
```

If you want to bypass the automatic environment updates and see the full
human-friendly output, you can pass the "--no-eval" flag:

```bash
tka login --no-eval
```

This command only prints the integration code. You must eval or source it
in your shell for it to take effect.

### Examples

```bash
# For bash or zsh, add this line to your ~/.bashrc or ~/.zshrc:
eval "$(ts-k8s-auth shell bash)"

# For fish, add this line to your ~/.config/fish/config.fish:
ts-k8s-auth shell fish | source

# For PowerShell, add this line to your profile (e.g. $PROFILE):
ts-k8s-auth shell powershell | Out-String | Invoke-Expression

```

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `signout`

```bash
tka signout [--quiet|-q]
```

### Aliases

- `signout`, `logout`

### Description

Sign out of the TKA service and revoke your current session.

This command requests the server to invalidate your active credentials. It does
not modify your shell environment automatically. If you previously exported
KUBECONFIG to point at an ephemeral file, consider unsetting or updating it.

### Examples

```bash
# Sign out and revoke your access
tka signout

# Alias form
tka logout

# Quiet mode (no output)
tka signout --quiet
```

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |

## Usage `version`

```bash
tka version
```

### Description

Shows version information

### Global Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-c, --config` | `string` | Name of the config file |
| `    --debug` | `bool` | enable debug logging |
| `-l, --long` | `bool` | Show long output (where available) |
| `-e, --no-eval` | `bool` | Do not evaluate the command |
| `-p, --port` | `int` | Port of the gRPC API of the Server (*default: 443*) |
| `-q, --quiet` | `bool` | Show no output (where available) |
| `-s, --server` | `string` | The Server Name on the Tailscale Network (*default: "tka"*) |
| `-t, --theme` | `string` | theme to use for the CLI (*default: "tokyo-night"*) |
