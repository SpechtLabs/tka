---
title: Configuration Reference
permalink: /reference/configuration
createTime: 2025/08/25 06:31:44
---

This page lists all configuration keys, their defaults, and effects. Sources: flags, environment (`TKA_` prefix), or config files (`config.yaml`).

### Conventions

- Env var mapping replaces `.` with `_` and uppercases keys, prefixed with `TKA_` (e.g., `tailscale.hostname` → `TKA_TAILSCALE_HOSTNAME`).
- Default config search paths: `.`, `$HOME/.config/tka/`, `/etc/tka/`.

### Common

- `debug` (bool, default `false`)
  - Enable debug logging.
- `otel.endpoint` (string, default `""`)
  - OTLP gRPC endpoint; if empty, OTel spans/metrics exporters may be disabled.
- `otel.insecure` (bool, default `true`)
  - Whether to use insecure transport to the OTLP endpoint.

### Tailscale

- `tailscale.hostname` (string, default `tka`)
  - Hostname for the tsnet node; also used by the CLI to build `https://{hostname}.{tailnet}`.
- `tailscale.port` (int, default `443`)
  - API port. Scheme is `https` if `443`, otherwise `http`.
- `tailscale.stateDir` (string, default `""`)
  - Directory for tsnet state (keys, control data).
  - If empty, a directory is selected automatically under [`os.UserConfigDir`](https://golang.org/pkg/os/#UserConfigDir) based on the name of the binary.
- `tailscale.tailnet` (string, no default)
  - Tailnet domain, e.g., `example.ts.net`; used by CLI to compose the base URL.
- `tailscale.capName` (string, default `specht-labs.de/cap/tka`)
  - Capability name the server requires from Tailscale ACLs.

#### Tailscale Environment variables

- `TS_AUTHKEY`
  - Auth key used to register/login the node to your tailnet without an interactive browser flow.
  - Recommended to use an ephemeral/reusable auth key generated from the Tailscale admin console.
  - Example: `export TS_AUTHKEY=tskey-auth-XXXXXXXXXXXXXXXX`
  - Security: treat like a secret. Prefer short-lived keys.

- `TSNET_FORCE_LOGIN`
  - If set to `1`, `true`, or `TRUE`, forces an interactive/browser login even if an auth key is present.
  - Useful for developer setups where you don’t want to depend on an auth key.
  - Example: `export TSNET_FORCE_LOGIN=1`

Notes:

- TKA embeds Tailscale using `tsnet`. These variables are read by Tailscale when bringing the embedded node up; TKA itself does not parse them.
- Variables commonly used with the standalone `tailscaled` daemon (e.g., `TS_STATE_DIR`, `TAILSCALE_*`) do not apply to the embedded `tsnet` client in this binary. Use `tailscale.stateDir` in TKA config or `--dir` to control state location.

### Server HTTP timeouts

- `server.readTimeout` (duration, default `10s`)
- `server.readHeaderTimeout` (duration, default `5s`)
- `server.writeTimeout` (duration, default `20s`)
- `server.idleTimeout` (duration, default `120s`)

These tune the embedded HTTP server created for `tsnet`.

### Operator

Defaults from code: `namespace=tka-dev`, `clusterName=tka-cluster`, `contextPrefix=tka-context-`, `userPrefix=tka-user-`.

- `operator.namespace` (string)
  - Namespace where SignIn and ServiceAccounts/Bindings are created.
- `operator.clusterName` (string)
  - Name written into kubeconfig contexts.
- `operator.contextPrefix` (string)
  - Prefix for per-user kubeconfig context name.
- `operator.userPrefix` (string)
  - Prefix for kubeconfig user entry.

### API behavior

- `api.retryAfterSeconds` (int, default `1`)
  - Hint for clients polling async operations (e.g., kubeconfig provisioning).

### CLI Output Settings

These settings control how the TKA CLI displays information and are used by client commands:

- `output.theme` (string, default `tokyo-night`)
  - Color theme for CLI output. Available themes: `ascii`, `dark`, `dracula`, `light`, `markdown`, `notty`, `tokyo-night`
- `output.long` (bool, default `true`)
  - Show detailed output by default when available
- `output.quiet` (bool, default `false`)
  - Suppress non-essential output when available
- `output.markdownlint-fix` (bool, default `false`)
  - Apply markdown formatting fixes to generated documentation

### Flags

Server and CLI share some flags via the root command:

```text
--config, -c            Path to config file
--debug                 Enable debug logging (maps to debug)
--server, -s            Tailscale hostname (maps to tailscale.hostname)
--port, -p              API port (HTTP) (maps to tailscale.port)
```

Server-only flags:

```text
--dir, -d               tsnet state directory (maps to tailscale.stateDir)
--cap-name, -n          Capability name to require (maps to tailscale.capName)
```

### Example config.yaml

```yaml
# Enable debug logging (top-level, not under server)
debug: false

# CLI output settings
output:
  theme: tokyo-night
  long: true
  quiet: false
  markdownlint-fix: false

tailscale:
  hostname: tka
  port: 443
  stateDir: /var/lib/tka/tsnet-state
  tailnet: your-tailnet.ts.net
  capName: specht-labs.de/cap/tka

server:
  readTimeout: 10s
  readHeaderTimeout: 5s
  writeTimeout: 20s
  idleTimeout: 120s

otel:
  endpoint: ""
  insecure: true

operator:
  namespace: tka-system
  clusterName: tka-cluster
  contextPrefix: tka-context-
  userPrefix: tka-user-

api:
  retryAfterSeconds: 1
```
