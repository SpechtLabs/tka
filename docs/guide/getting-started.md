---
title: Getting Started
permalink: /guide/getting-started
createTime: 2025/08/25 06:31:08
---

### Goal

Set up Tailscale Kubernetes Auth in a Kubernetes Cluster, join it to the tailnet, and use the CLI to obtain an ephemeral kubeconfig.

> [!TIP]
> What you’ll achieve
>
> - tka server running on your tailnet
> - ephemeral kubeconfig issued for your user
> - RBAC bound to the role defined in your ACL capability
> - logout revokes access and cleans up ephemeral resources

### Prerequisites

- A Kubernetes cluster you can reach (_[kind](https://kubernetes.io/docs/tasks/tools/#kind), dev cluster, etc._)
- Tailscale account[^tailscale_install]
  - Your Laptop/PC joined to the tailnet
  - Tailscale authentication key[^tailscale_docs]
  - If you plan to serve the tka API over https (which you should): have https enabled in your tailnet[^tailscale_enable_https]
- [kubectl]
- (_Optional_) Go 1.24.4+[^self_compile]

[kubectl]: https://kubernetes.io/docs/tasks/tools/

[^tailscale_docs]: [Setting up a server on your Tailscale network](https://tailscale.com/kb/1245/set-up-servers)

[^self_compile]: If you want to compile tka yourself, instead of using a [release](https://github.com/SpechtLabs/tka)

[^tailscale_install]: [Tailscale quickstart](https://tailscale.com/kb/1017/install)

[^tailscale_enable_https]: [Enabling HTTPS](https://tailscale.com/kb/1153/enabling-https)

### Clone and build (_optional_)

> [!TIP]
> This step is optional.
> You can use the latest [release](https://github.com/SpechtLabs/tka) instead

```bash
git clone https://github.com/SpechtLabs/tka
cd tka/src
go build -o bin/tka-server ./cmd/server
go build -o bin/tka ./cmd/cli
```

Expected: two binaries created in `src/bin/`: `tka-server` and `tka`.

### Prepare our Kubernetes cluster

Install the Kubernetes Custom Resource Definition. The CRD is generated and installed using:

```bash
make generate
kubectl apply -k config
```

### Configure TKA

TKA uses [viper] for configuration with this precedence:

[viper]: <https://github.com/spf13/viper>

1. flags
1. env-vars (`TKA_`)
1. config files

Default search paths for `config.yaml`:

1. current directory
1. `$HOME`
1. `$HOME/.config/tka/`
1. `/data`

Start from the example at [`src/config.yaml`](https://github.com/SpechtLabs/tka/blob/main/src/config.yaml):

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

Instead of using a config file, you can also use environment variables.
Environment variables are prefixed with `TKA_`, for example:

```bash
export TKA_TAILSCALE_HOSTNAME=tka
export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
export TKA_TAILSCALE_PORT=8123
```

> [!INFO]
> The default server name is `tka`, and it is good practice to keep your primary login-server on this domain.
> You will use `tka.<your-tailnet>.ts.net` as the primary authentication API endpoint
> to authenticate to all your clusters, and all other clusters will attempt to connect
> to `tka.<your-tailnet>.ts.net` too to join tka.
>
>To change the server-name, use the `--server, -s` flag or the `tailscale.hostname` configuration parameter. See [configuration reference] for more details.

[configuration reference]: ../reference/configuration.md#Tailscale

### Run the server (local debugging)

This is for debugging only at the moment.
Todo: Reference the helm-chart or kustomize deployment in the future

> [!WARNING]
> The server exposes an HTTP API **only** over your tailnet.
>
> You likely won't be able to connect to the tka API just yet!
>
> In order to talk to the tka server,
> you **must**[^rfc_2119] be a member of the same tailnet
> and the Tailscale ACL **must** allow communication between your client and tka on the specified port[^tailscale_grant_example].
>
> **We will set up access in the next step.**

[^rfc_2119]: [RFC 2119: Key words for use in RFCs to Indicate Requirement Levels](https://www.rfc-editor.org/rfc/rfc2119.html)

[^tailscale_grant_example]: [Grant examples: Allow based on purpose using tags](https://tailscale.com/kb/1458/grant-examples#allow-based-on-purpose-using-tags)

Ensure you have your Tailscale auth key ready

```shell
$ cd tka/src
$ env TS_AUTHKEY="tskey-auth-XXXX-XXXXXXXXXX" \
    ./bin/tka-server serve --server tka --port 443

> [!NOTE]
> TLS and ports
>
> - Port 443 implies HTTPS in the CLI’s URL construction. Use 443 unless you have a reason to run HTTP.
> - If you pick a non-443 port, the CLI will use HTTP unless you include an explicit scheme in `tailscale.hostname`.
```

Expected:

- Logs show Tailscale coming up and a URL like `https://tka.your-tailnet.ts.net:8123`
- The Kubernetes operator starts; metrics available at `/metrics/controller`
- Swagger UI at `/swagger/`

### Prepare Tailscale capability grant

In your Tailscale ACLs, assign a capability with a role and period. Example snippet:

```jsonc
{
    "groups": {
        "group:admins":    ["alice@example.com",],
        "group:developer": ["bob@example.com",],
    },
     "tagOwners": {
        "tag:tka": ["tag:k8s-operator", "group:admins"], // Devices tagged with tag:tka are tailscale-k8s-auth services exposed by tka
    },
    "grants": [
        // Allow admins to use tka & connect via admin role
        {
            "src": ["group:admins"],
            "dst": ["tag:tka"],
            "ip":  ["443"], // Match the port used by the tka-server
            "app": {
                "specht-labs.de/cap/tka": [
                    {
                        "role":   "cluster-admin",
                        "period": "8h",
                    },
                ],
            },
        },
        // Allow developers to use tka & connect via read-only role for debugging purposes
        {
            "src": ["group:developer"],
            "dst": ["tag:tka"],
            "ip":  ["443"], // Match the port used by the tka-server
            "app": {
                "specht-labs.de/cap/tka": [
                    {
                        "role":   "cluster-reader",
                        "period": "4h",
                    },
                ],
            },
        },
    ]
}
```

> [!TIP]
> Capability name must match the server’s `--cap-name`. See [configuration reference] for more details.

### Configure the CLI to reach the server

The CLI constructs the server address as:

```text
{scheme}://{hostname}.{tailnet}:{port}
```

Where scheme is `https` if port is `443`, otherwise `http`.

Set via config or env for the CLI:

```bash
export TKA_TAILSCALE_HOSTNAME=tka
export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
export TKA_TAILSCALE_PORT=8123
```

> [!NOTE]
> If you set port 443, the CLI will use HTTPS automatically. For other ports, it uses HTTP unless you prefix `tailscale.hostname` with `https://`.

### Sign in and fetch kubeconfig

```shell
# Authenticate and obtain a kubeconfig
$ ./bin/tka login
✓ sign-in successful!
✓ kubeconfig saved to
    /var/folders/tn/s4s0wwrx7mgch6939pp4qp1h0000gn/T/kubeconfig-2528675676.yaml
    export KUBECONFIG="/var/folders/tn/s4s0wwrx7mgch6939pp4qp1h0000gn/T/kubeconfig-2528675676.yaml"
• Login Information:
  ╭─────────────────────────────────────────────╮
  │ User:        alice                          │
  │ Role:        cluster-admin                  │
  │ Until:       Mon, 25 Aug 2025 14:50:10 CEST │
  ╰─────────────────────────────────────────────╯

# Use kubeconfig
$ export KUBECONFIG="/var/folders/tn/s4s0wwrx7mgch6939pp4qp1h0000gn/T/kubeconfig-2528675676.yaml"

# We can now access the Cluster
$ kubectl get -n tka-dev TkaSignin
NAME            PROVISIONED
tka-user-alice  true

$ kubectl get -n tka-dev ServiceAccount
NAME             SECRETS   AGE
default          0         15m
tka-controller   0         10m
tka-user-alice   0         5m

$ kubectl get ClusterRoleBinding tka-user-alice-binding
NAME                    ROLE                        AGE
tka-user-alice-binding  ClusterRole/cluster-admin   5m9s
```

Expected:

- `tka login` returns Accepted and shows login info
- `tka kubeconfig` prints a file path and sets KUBECONFIG in that process; use the shown path in your shell
- `kubectl` commands succeed within the granted role

### Sign out

```bash
# Logout
$ ./bin/tka logout
✓ You have been signed out

# Verify Logout
$ kubectl get ns
error: You must be logged in to the server (Unauthorized)
```

Expected: your ephemeral resources are removed; access is revoked.

### Troubleshooting

- If kubeconfig is "not ready yet," the CLI retries; server sets `Retry-After` hint
- Ensure your client is on the tailnet; Funnel requests are rejected
- Verify the capability mapping and period string in your ACLs
- Check server logs for WhoIs or capability parsing errors
