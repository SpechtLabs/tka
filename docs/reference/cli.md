---
title: CLI Reference
permalink: /reference/cli
createTime: 2025/08/25 06:31:50
---

The `tka` CLI communicates with the TKA server over your tailnet to authenticate and fetch temporary kubeconfigs.

---

### Command: `tka`

**Usage:**
`tka`

**Description:**
tka is the CLI for Tailscale Kubernetes Auth
tka is the client for Tailscale Kubernetes Auth. It lets you authenticate to clusters over Tailscale, manage kubeconfig entries, and inspect status with readable, themed output.

**Arguments:**

- None

**Aliases:**

- signin, auth

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka shell`

**Usage:**
`shell <bash|zsh|fish|powershell>`

**Description:**
Generate shell integration for tka wrapper
The "shell" command generates shell integration code for the tka wrapper.

**Arguments:**

- `<shell>` — one of: bash | zsh | fish | powershell

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka login`

**Usage:**
`login`

**Description:**
Sign in and configure kubectl with temporary access
Authenticate using your Tailscale identity and retrieve a temporary Kubernetes access token. This command automatically fetches your kubeconfig, writes it to a temporary file, sets the KUBECONFIG environment variable.

**Arguments:**

- None

**Aliases:**

- reauth, refresh

**Flags:**

- `--quiet, -q <bool>` (default: false) — Do not print login information
- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka reauthenticate`

**Usage:**
`reauthenticate`

**Description:**
Reauthenticate and configure kubectl with temporary access
Reauthenticate by signing out and then signing in again to refresh your temporary access. This command is a convenience wrapper which:

1. Calls signout to revoke your current session
2. Calls login to obtain a fresh ephemeral kubeconfig

**Arguments:**

- None

**Aliases:**

- logout

**Flags:**

- `--quiet, -q <bool>` (default: false) — Do not print login information
- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool)` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka signout`

**Usage:**
`signout`

**Description:**
Sign out and remove access from the cluster
Sign out of the TKA service and revoke your current session. This does not modify your shell environment automatically. If you previously exported KUBECONFIG to point at an ephemeral file, consider unsetting or updating it.

**Arguments:**

- None (exactly zero)

**Flags:**

- `--quiet, -q <bool>` (default: false) — Do not print signout information
- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka kubeconfig`

**Usage:**
`kubeconfig`

**Description:**
Fetch your temporary kubeconfig
Retrieve an ephemeral kubeconfig for your current session and save it to a temporary file. This command downloads the kubeconfig from the TKA server and writes it to a temp file. It also sets KUBECONFIG for this process so that subsequent kubectl calls from this process use the new file. To update your interactive shell, export KUBECONFIG yourself

**Arguments:**

- None

**Aliases:**

- signin

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka get`

**Usage:**
`get`

**Description:**
Retrieve read-only resources from TKA.
The get command retrieves resources from your Tailscale Kubernetes Auth service

**Arguments:**

- None

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka get login`

**Usage:**
`login`

**Description:**
Show current login information and provisioning status.
Display details about your current login state, including whether provisioning was successful. This does not modify your session.

**Arguments:**

- None

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Command: `tka get kubeconfig`

**Usage:**
`kubeconfig`

**Description:**
Fetch your temporary kubeconfig
Retrieve an ephemeral kubeconfig for your current session and save it to a temporary file. This command downloads the kubeconfig from the TKA server and writes it to a temp file. It also sets KUBECONFIG for this process so that subsequent kubectl calls from this process use the new file. To update your interactive shell, export KUBECONFIG yourself

**Arguments:**

- None

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--theme, -t <string>` (default: tokyo-night) — theme to use for the CLI
- `--no-eval, -e <bool>` (default: false) — Do not evaluate the command

---

### Server binary

The server uses the same root command with server-specific flags.

### Command: `tka serve`

**Usage:**
`serve`

**Description:**
Run the TKA API and Kubernetes operator services
Start the Tailscale-embedded HTTP API and the Kubernetes operator.

This command:

- Starts a tailscale tsnet server for inbound connections
- Serves the TKA HTTP API with authentication and capability checks
- Runs the Kubernetes operator to manage kubeconfigs and user resources

Configuration is provided via flags and environment variables (see --help).

**Arguments:**

- None

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
- `--dir, -d <string>` (default: "") — tsnet state directory; a default one will be created if not provided
- `--cap-name, -n <string>` (default: specht-labs.de/cap/tka) — name of the capability to request from api

---

### Command: `tka version`

**Usage:**
`version`

**Description:**
Shows version information

**Arguments:**

- None

**Flags:**

- `--config, -c <string>` (default: "") — Name of the config file
- `--debug <bool>` (default: false) — enable debug logging
- `--server, -s <string>` (default: tka) — The Server Name on the Tailscale Network
- `--port, -p <int>` (default: 443) — Port of the gRPC API of the Server
- `--long, -l <bool>` (default: false) — Show long output (where available)
