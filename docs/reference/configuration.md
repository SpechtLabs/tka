---
title: Configuration Reference
permalink: /reference/configuration
createTime: 2025/08/25 06:31:44
---

This page lists all configuration keys, their defaults, and effects. Sources: flags, environment (`TKA_` prefix), or config files (`config.yaml`).

### Conventions

- Env var mapping replaces `.` with `_` and uppercases keys, prefixed with `TKA_` (e.g., `tailscale.hostname` → `TKA_TAILSCALE_HOSTNAME`).
- Default config search paths: `.`, `$HOME`, `$HOME/.config/tka/`, `/data`.

### Common

- `debug` (bool, default `false`)
  - Enable debug logging.
- `otel.endpoint` (string, default `""`)
  - OTLP gRPC endpoint; if empty, OTel spans/metrics exporters may be disabled.
- `otel.insecure` (bool, default `true`)
  - Whether to use insecure transport to the OTLP endpoint.

### Tailscale

- `tailscale.hostname` (string, default empty via flag default `tka` for CLI/server)
  - Hostname for the tsnet node; also used by the CLI to build `https://{hostname}.{tailnet}`.
- `tailscale.port` (int, default `443`)
  - API port. Scheme is `https` if `443`, otherwise `http`.
- `tailscale.stateDir` (string, default `""`)
  - Directory for tsnet state (keys, control data). If empty, tsnet uses its default.
- `tailscale.tailnet` (string, no default)
  - Tailnet domain, e.g., `example.ts.net`; used by CLI to compose the base URL.
- `tailscale.capName` (string, default `specht-labs.de/cap/tka`)
  - Capability name the server requires from Tailscale ACLs.

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

### Flags

Server and CLI share some flags via the root command:

```text
--config, -c            Path to config file
--debug                 Enable debug logging (maps to debug)
--server, -s            Tailscale hostname (maps to tailscale.hostname)
--port, -p              API port (maps to tailscale.port)
```

Server-only flags:

```text
--dir, -d               tsnet state directory (maps to tailscale.stateDir)
--cap-name, -n          Capability name to require (maps to tailscale.capName)
```

### Example config.yaml

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
