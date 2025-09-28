---
title: CLI Reference
permalink: /reference/cli
createTime: 2025/09/08 04:10:13
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
- Config: `output.theme` (in config file)
- Environment: `TKA_THEME`

**Accepted themes**: ascii, dark, dracula, *tokyo-night*, light

#### Notes

- Global flags like `--theme` are available to subcommands

### Examples

```bash
# generic dark theme
$ tka --theme dark login

# light theme
$ TKA_OUTPUT_THEME=light tka kubeconfig

# no theme (useful in non-interactive contexts)
$ tka --theme notty login

```

### Available Commands

> [!TIP]
> Use `tka [command] --help` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|
| **`cluster-info`** | View cluster information |
| **`completion`** | Generate the autocompletion script for the specified shell |
| **`config`** | Get or set configuration values |
| **`generate`** | Generate resources in TKA. |
| **`get`** | Retrieve read-only resources from TKA. |
| **`kubeconfig`** | Fetch your temporary kubeconfig |
| **`login`** | Sign in and configure kubectl with temporary access |
| **`reauthenticate`** | Reauthenticate and configure kubectl with temporary access |
| **`set`** | Set resources in TKA. |
| **`shell`** | Start a subshell with temporary Kubernetes access via Tailscale identity |
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

## Usage `cluster-info`

```bash
tka get cluster-info
```

### Description

View cluster information.
This command returns the cluster information that TKA exposes to understand the cluster you're connecting to.

### Examples

```bash
# View cluster information
tka get cluster-info

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

## Usage `config`

```bash
tka config [key] [value] [--force] [flags]
```

### Description

Get or set configuration values in the TKA configuration file.

This command works similarly to `git config --global`:

- When called with no arguments, shows all current configuration
- When called with just a key, it shows the current value
- When called with key and value, it sets the configuration
- Configuration is written to the file that was used to load the current config
- If no config file exists and `--force` is used, creates `~/.config/tka/config.yaml`

### Examples

```bash
# Show all current configuration
tka config

# Show the current debug setting
tka config debug

# Set output.markdownlint-fix to true
tka config output.markdownlint-fix true

# Create a config file and set a value (when no config exists)
tka config output.long true --force
```

### Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `-f, --force` | `bool` | Create config file at the lowest tier if no config file exists |

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

## Usage `generate`

```bash
tka generate [command]
```

### Description

The generate command generates resources in your Tailscale Kubernetes Auth service

### Examples

```bash
# Generate a kubeconfig
tka generate kubeconfig
```

### Available Commands

> [!TIP]
> Use `tka generate [command] --help` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|
| **`integration`** | Generate shell integration for tka wrapper |

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

### Usage `integration`

```bash
tka generate integration <bash|zsh|fish|powershell>
```

#### Description

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

#### Examples

```bash
# For bash or zsh, add this line to your ~/.bashrc or ~/.zshrc:
eval "$(ts-k8s-auth shell bash)"

# For fish, add this line to your ~/.config/fish/config.fish:
ts-k8s-auth shell fish | source

# For PowerShell, add this line to your profile (e.g. $PROFILE):
ts-k8s-auth shell powershell | Out-String | Invoke-Expression

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
| **`cluster-info`** | View cluster information |
| **`config`** | Get configuration values |
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

### Usage `cluster-info`

```bash
tka get cluster-info [flags]
```

#### Description

View cluster information.
This command returns the cluster information that TKA exposes to understand the cluster you're connecting to.

#### Examples

```bash
# View cluster information
tka get cluster-info

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

### Usage `config`

```bash
tka get config [key] [flags]
```

#### Description

Get configuration values in the TKA configuration file.

This command works similarly to `git config --global`:

- When called with no arguments, shows all current configuration
- When called with just a key, it shows the current value

#### Examples

```bash
# Show all current configuration
$ tka get config
api:
    retryafterseconds: 1
debug: false
output:
    long: true
    markdownlint-fix: false
    quiet: false
    theme: dracula
...snip...

# Show the current theme setting
$ tka get config output.theme
tokyo-night

# Show the current theme setting and the filename of the config file used
$ go run ./cmd/cli get config output.theme
→ output.theme: dracula

# Show the current theme setting and only the value
$ go run ./cmd/cli get config output.theme --quiet
dracula

# Show the current theme setting and the filename of the config file used
$ go run ./cmd/cli get config output.theme --filename
→ output.theme: dracula
    Config file used: /Users/cedi/.config/tka/config.yaml

# Show the all configuration and the filename of the config file used
$ tka get config --filename
ℹ Config file used:
    /Users/cedi/.config/tka/config.yaml

api:
    retryafterseconds: 1
debug: false
output:
    long: true
    markdownlint-fix: false
    quiet: false
    theme: dracula
..snip...

# Show only the filepath of the config file used
$ tka get config --filename --quiet
/Users/cedi/.config/tka/config.yaml

# Invalid: combine --filename and --quiet when using  key
$ tka get config output.theme --filename --quiet
✗ Invalid: cannot combine --filename and --quiet when also specifying a [key]

```

#### Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `    --filename` | `bool` | Show the filename of the config file used |

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
tka login [--quiet|-q] [--long|-l|--no-eval|-e] [--shell]
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

### Flags

| **Flag** | **Type** | **Usage** |
|:---------|:--------:|:----------|
| `    --shell` | `bool` | Start a subshell with temporary Kubernetes access |

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

## Usage `set`

```bash
tka set [command]
```

### Description

The set command sets resources in your Tailscale Kubernetes Auth service

### Examples

```bash
# Set the debug setting to true
tka set output.theme dark
```

### Available Commands

> [!TIP]
> Use `tka set [command] --help` for more information about a command.

| **Command** | **Description** |
|:------------|:----------------|
| **`config`** | Set configuration values |

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

### Usage `config`

```bash
tka set config <key> <value> [--force]
```

#### Description

Set configuration values in the TKA configuration file.

This command works similarly to `git config --global`:

- When called with key and value, it sets the configuration
- Configuration is written to the file that was used to load the current config
- If no config file exists and `--force` is used, creates `~/.config/tka/config.yaml`

#### Examples

```bash
# Set the debug setting to true
tka config output.theme dark

# Set output.markdownlint-fix to true
tka config output.markdownlint-fix true

# Create a config file and set a value (when no config exists)
tka config output.long true --force
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

## Usage `shell`

```bash
tka shell
```

### Description

## Shell Command

The **shell** command authenticates you using your Tailscale identity and
retrieves a short-lived Kubernetes access token. It then spawns an interactive
subshell (using your login shell, e.g. `bash` or `zsh`") with the
`KUBECONFIG` environment variable set to a temporary kubeconfig file.

This provides a clean and secure workflow:

- Your existing shell environment remains untouched.
- All Kubernetes operations inside the subshell use the temporary credentials.
- When you exit the subshell, the credentials are automatically revoked and
  the temporary kubeconfig file is deleted.

This is useful for administrators and developers who need ephemeral access to
a cluster without persisting credentials on disk or leaking them into their
long-lived shell environment.

### Examples

```bash
# Start a subshell with temporary Kubernetes access
tka shell

# Inside the subshell, run kubectl commands as usual
kubectl get pods -n default

# When finished, exit the subshell
exit

# At this point, the temporary credentials are revoked automatically
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
