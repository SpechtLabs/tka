---
title: Configuration Reference
permalink: /reference/configuration
createTime: 2025/08/25 06:31:44
---

This page lists all configuration keys, their defaults, and effects. Sources: flags, environment (`TKA_` prefix), or config files (`config.yaml`).

## Conventions

- Env var mapping replaces `.` with `_` and uppercases keys, prefixed with `TKA_` (e.g., `tailscale.hostname` → `TKA_TAILSCALE_HOSTNAME`).
- Default config search paths: `.`, `$HOME/.config/tka/`, `/etc/tka/`.

## Common

- `debug` (bool, default `false`)
  - Enable debug logging.
- `otel.endpoint` (string, default `""`)
  - OTLP gRPC endpoint; if empty, OTel spans/metrics exporters may be disabled.
- `otel.insecure` (bool, default `true`)
  - Whether to use insecure transport to the OTLP endpoint.

## Tailscale

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

### Tailscale Environment variables

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

## Server HTTP timeouts

- `server.readTimeout` (duration, default `10s`)
- `server.readHeaderTimeout` (duration, default `5s`)
- `server.writeTimeout` (duration, default `20s`)
- `server.idleTimeout` (duration, default `120s`)

These tune the embedded HTTP server created for `tsnet`.

## Operator

Defaults from code: `namespace=tka-dev`, `clusterName=tka-cluster`, `contextPrefix=tka-context-`, `userPrefix=tka-user-`.

- `operator.namespace` (string)
  - Namespace where SignIn and ServiceAccounts/Bindings are created.
- `operator.clusterName` (string)
  - Name written into kubeconfig contexts.
- `operator.contextPrefix` (string)
  - Prefix for per-user kubeconfig context name.
- `operator.userPrefix` (string)
  - Prefix for kubeconfig user entry.

## API behavior

- `api.retryAfterSeconds` (int, default `1`)
  - Hint for clients polling async operations (e.g., kubeconfig provisioning).

## CLI Output Settings

These settings control how the TKA CLI displays information and are used by client commands:

- `output.theme` (string, default `tokyo-night`)
  - Color theme for CLI output. Available themes: `ascii`, `dark`, `dracula`, `light`, `markdown`, `notty`, `tokyo-night`
- `output.long` (bool, default `true`)
  - Show detailed output by default when available
- `output.quiet` (bool, default `false`)
  - Suppress non-essential output when available
- `output.markdownlint-fix` (bool, default `false`)
  - Apply markdown formatting fixes to generated documentation

## Cluster Information

The `clusterInfo` section configures the cluster connection details that TKA exposes to authenticated users through the cluster-info API endpoint. This information is used by users to configure their kubeconfig files and understand the cluster they're connecting to.

- `clusterInfo.apiEndpoint` (string, **required unless `configMapRef.enabled` is true**)
  - The Kubernetes API server URL or IP address that users should connect to
  - This should be the externally accessible endpoint of the cluster's API server
  - Examples: `"https://api.cluster.example.com:6443"`, `"https://192.168.1.100:6443"`
  - **Note**: Must be set for TKA to start when not using `configMapRef`.

- `clusterInfo.caData` (string, base64-encoded, default `""`)
  - The base64-encoded Certificate Authority (CA) data for the Kubernetes cluster
  - Used to verify the TLS certificate presented by the API server
  - Should be the PEM-encoded CA certificate, encoded as base64
  - If empty and `insecureSkipTLSVerify` is false, the system's root CA bundle will be used
  - **Note**: CA data is public information (not sensitive) and can be stored in ConfigMaps
  - **Source**: Available from `cluster-info` ConfigMap in `kube-public` namespace or kubeconfig
  - **Purpose**: Required for production clusters to verify server identity

- `clusterInfo.insecureSkipTLSVerify` (bool, default `false`)
  - Controls whether TLS certificate verification should be skipped when connecting to the cluster
  - When `true`, clients will accept any certificate and ignore hostname mismatches
  - **Security**: Should only be set to `true` for development/testing with self-signed certificates
  - Production clusters should use valid certificates and keep this `false`

- `clusterInfo.labels` (map[string]string, default `{}`)
  - Key-value pairs used to identify and categorize the cluster
  - Helps users distinguish between different clusters and can be used for automation
  - Common examples: `environment: production`, `region: us-west-2`, `project: webapp`, `team: platform`
  - These labels are exposed in the cluster-info API response and can be used by client tools

### ConfigMap Reference

Use a Kubernetes ConfigMap to supply cluster connection details. This is useful for kubeadm clusters which expose a `cluster-info` ConfigMap in the `kube-public` namespace.

- `clusterInfo.configMapRef.enabled` (bool, default `false`)
  - When `true`, TKA reads connection details from the specified ConfigMap
  - Mutually exclusive with explicit `clusterInfo.*` connection settings
- `clusterInfo.configMapRef.name` (string, default `""`)
  - Name of the ConfigMap containing connection details
- `clusterInfo.configMapRef.namespace` (string, default `""`)
  - Namespace where the ConfigMap is located
- `clusterInfo.configMapRef.keys.apiEndpoint` (string, default `"apiEndpoint"`)
  - Key within the ConfigMap data which contains the API endpoint
- `clusterInfo.configMapRef.keys.caData` (string, default `"caData"`)
  - Key within the ConfigMap data which contains base64-encoded CA certificate data
- `clusterInfo.configMapRef.keys.insecure` (string, default `"insecure"`)
  - Key within the ConfigMap data which contains a boolean-like string for insecure TLS (e.g., `"true"`, `"1"`)
- `clusterInfo.configMapRef.keys.kubeconfig` (string, default `"kubeconfig"`)
  - Optional. If set and present, TKA will parse the embedded kubeconfig and extract the first cluster's `server` and `certificate-authority-data` (kubeadm `cluster-info` style).
- Use `clusterInfo.labels` to add labels; they apply in both explicit and ConfigMap modes

> [!NOTE]
> You must provide exactly one of: explicit `clusterInfo.apiEndpoint` or `configMapRef.enabled: true` with a valid reference. If both are provided, TKA will refuse to start.

## Flags

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
--health-port           Port for local metrics and health server (maps to health.port)
--api-endpoint          Kubernetes API endpoint URL (maps to clusterInfo.apiEndpoint)
--ca-data               Base64-encoded cluster CA data (maps to clusterInfo.caData)
--insecure-skip-tls-verify  Skip TLS verification (maps to clusterInfo.insecureSkipTLSVerify)
--labels                Cluster labels key=value list (maps to clusterInfo.labels)
```

## Example config.yaml

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

clusterInfo:
  labels:
    environment: production
    region: us-west-2
    project: webapp
    team: platform

  # Alternative: use a ConfigMap as the source of cluster connection details
  # Enable and omit apiEndpoint above.
  configMapRef:
    enabled: false
    name: "cluster-info"
    namespace: "kube-public"
    keys:
      apiEndpoint: "apiEndpoint"
      caData: "caData"
      insecure: "insecure"
```
