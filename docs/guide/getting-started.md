---
title: Getting Started
permalink: /guide/getting-started
createTime: 2025/08/25 06:31:08
---

### Goal

Set up TKA (Tailscale Kubernetes Auth), run the server, and use the CLI to obtain an ephemeral kubeconfig.

### Prerequisites

- Go 1.22+
- Docker (optional, for containerized runs)
- A Kubernetes cluster you can reach (kind, dev cluster, etc.)
- Tailscale account and a tailnet
- Basic familiarity with `kubectl` and `cobra`-style CLIs

### 1) Clone and build

```bash
git clone https://github.com/SpechtLabs/tka
cd tka/src
go build -o bin/tka-server ./cmd/server
go build -o bin/tka ./cmd/cli
```

Expected: two binaries created in `src/bin/`: `tka-server` and `tka`.

### 2) Configure TKA

TKA uses Viper for configuration with this precedence: flags → env (`TKA_`) → config files.

Default search paths for `config.yaml`:

- current directory
- `$HOME`
- `$HOME/.config/tka/`
- `/data`

Start from the example at `src/config.yaml`:

```yaml
tailscale:
  hostname: tka
  port: 8123
  stateDir: /tmp/tka-ts-state
  tailnet: your-tailnet.ts.net

server:
  readTimeout: 10s
  readHeaderTimeout: 5s
  writeTimeout: 20s
  idleTimeout: 120s

otel:
  endpoint: ""
  insecure: true

operator:
  namespace: tka-dev
  clusterName: tka-cluster
  contextPrefix: tka-context-
  userPrefix: tka-user-

api:
  retryAfterSeconds: 1
```

Environment variables are supported with the `TKA_` prefix, for example:

```bash
export TKA_TAILSCALE_HOSTNAME=tka
export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
export TKA_TAILSCALE_PORT=8123
```

### 3) Run the server

The server exposes an HTTP API over your tailnet via `tsnet`.

```bash
cd tka/src
./bin/tka-server serve --server tka --port 8123 --dir /tmp/tka-ts-state
```

Flags of interest:

- `--server, -s`: Tailscale device hostname to register (default `tka`)
- `--port, -p`: API port (default `443`)
- `--dir, -d`: tsnet state directory
- `--cap-name, -n`: capability name to check in Tailscale ACLs (default `specht-labs.de/cap/tka`)

Expected:

- Logs show Tailscale coming up and a URL like `https://tka.your-tailnet.ts.net:8123`
- The Kubernetes operator starts; metrics available at `/metrics/controller`
- Swagger UI at `/swagger/`

### 4) Prepare Tailscale capability grant

In your Tailscale ACLs, assign a capability with a role and period. Example snippet:

```json
{
  "capabilities": {
    "specht-labs.de/cap/tka": {
      "user@example.com": { "role": "read-only", "period": "1h" }
    }
  }
}
```

The server enforces the capability name given by `--cap-name`.

### 5) Configure the CLI to reach the server

The CLI constructs the server address as:

```text
{scheme}://{hostname}.{tailnet}:{port}
```

Where scheme is `https` if port is 443, otherwise `http`.

Set config or env for the CLI:

```bash
export TKA_TAILSCALE_HOSTNAME=tka
export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
export TKA_TAILSCALE_PORT=8123
```

### 6) Sign in and fetch kubeconfig

```bash
# Authenticate and trigger provisioning
./bin/tka login

# Poll until credentials are ready and write a temp kubeconfig; prints its path
./bin/tka kubeconfig

# Use kubectl with the temporary credentials
export KUBECONFIG=$(./bin/tka kubeconfig | awk '{print $NF}')
kubectl get ns
```

Expected:

- `tka login` returns Accepted and shows login info
- `tka kubeconfig` prints a file path and sets KUBECONFIG in that process; use the shown path in your shell
- `kubectl` commands succeed within the granted role

### 7) Sign out

```bash
./bin/tka signout
```

Expected: your ephemeral resources are removed; access is revoked.

### Troubleshooting

- If kubeconfig is "not ready yet," the CLI retries; server sets `Retry-After` hint
- Ensure your client is on the tailnet; Funnel requests are rejected
- Verify the capability mapping and period string in your ACLs
- Check server logs for WhoIs or capability parsing errors
