---
title: CLI Reference
permalink: /reference/cli
createTime: 2025/08/25 06:31:50
---

The `tka` CLI communicates with the TKA server over your tailnet to authenticate and fetch temporary kubeconfigs.

### Global options

```text
--config, -c            Path to config file
--debug                 Enable debug logging
--server, -s            Tailscale hostname (default: tka)
--port, -p              API port (default: 443)
```

Environment variables (Viper): `TKA_TAILSCALE_HOSTNAME`, `TKA_TAILSCALE_TAILNET`, `TKA_TAILSCALE_PORT`.

The CLI determines the server address as:

```text
{scheme}://{hostname}.{tailnet}:{port}
```

Scheme is `https` if port is 443, otherwise `http`.

### Commands

#### login | signin

```bash
tka login
```

Authenticates using your Tailscale identity and triggers Kubernetes credential provisioning. On success, prints status and proceeds to fetch kubeconfig if used in a workflow.

Behavior:

- POST `/api/v1alpha1/login`
- Requires capability (default `specht-labs.de/cap/tka`) defined in Tailscale ACLs with fields `{role, period}`
- Returns 202 Accepted with login info while provisioning

#### get login

```bash
tka get login
```

Retrieves the current login status (GET `/api/v1alpha1/login`). Useful for checking provisioning state.

#### kubeconfig

```bash
tka kubeconfig
```

Polls the server until credentials are ready and writes a temporary kubeconfig file to a secure temp location. Prints the file path and sets `KUBECONFIG` for the process while running.

Behavior:

- GET `/api/v1alpha1/kubeconfig`
- On not-ready, server replies 202 and `Retry-After`; CLI shows a spinner and retries
- On success, writes to a temp file and prints its path

#### get kubeconfig

```bash
tka get kubeconfig
```

Same as `kubeconfig`; provided under the `get` namespace.

#### signout | logout

```bash
tka signout
```

Revokes your current access by deleting the SignIn and bindings in the cluster.

Behavior:

- POST `/api/v1alpha1/logout`
- Returns current login info and confirmation

### Examples

```bash
# Set environment for server discovery
export TKA_TAILSCALE_HOSTNAME=tka
export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
export TKA_TAILSCALE_PORT=8123

# Authenticate and get kubeconfig
tka login
tka kubeconfig

# Use kubectl
export KUBECONFIG=$(tka kubeconfig | awk '{print $NF}')
kubectl get ns

# Revoke access
tka signout
```
