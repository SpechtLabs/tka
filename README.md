# Tailscale Kubernetes Auth

[![Go Build & Docker Build](https://github.com/spechtlabs/tka/actions/workflows/build.yaml/badge.svg)](https://github.com/spechtlabs/tka/actions/workflows/build.yaml)
[![Documentation](https://github.com/spechtlabs/tka/actions/workflows/docs-website.yaml/badge.svg)](https://github.com/spechtlabs/tka/actions/workflows/docs-website.yaml)

Forget complex auth proxies, VPNs, or OIDC setups. `tka` gives you secure, identity-aware access to your Kubernetes clusters using just your Tailscale identity and network — with short-lived, auto-cleaned credentials.

## Why?

Traditional Kubernetes access control is either:

- Painful to manage (e.g., OIDC integrations, kubeconfig sprawl), or
- Overly centralized and complex (e.g., auth proxies and bastion-style gateways).

We believe Kubernetes access should be:

- **Secure by default** via ephemeral, scoped credentials
- **Simple to use** via `tsh login`-like UX
- **Network-gated** via your existing [Tailscale ACLs and Grants]
- **Kubernetes-native** using built-in ServiceAccounts and RBAC

### What about [Teleport]?

We love [Teleport][gh-teleport] dearly, and it was a major inspiration for this project.
It's a robust, production-proven system that handles multi-protocol access with powerful SSO, audit, and session recording features.

That said, we needed something just for Kubernetes, something much lighter weight, and most importantly, something that works with our existing Tailscale setup. Why have two ZTNA systems that provide _almost_ the same features, when you can go out of your way to waste time building your own thing, learn a ton in the process, and make it integrate better into your existing setup?

### What about [Tailscale's API server proxy]

Tailscale’s Kubernetes Operator is a fantastic way to access your Kubernetes cluster over the tailnet.
It can proxy requests to the Kubernetes API and impersonate users or groups based on tailnet identity, allowing you to define fine-grained access via standard Kubernetes RBAC.
It’s a great fit for many use cases.

However, we wanted a different model of access.
Our idea around access is about dynamically provisioning ephemeral Service Accounts for users, with the Cluster Role Bindings configured via the tailscale ACL file.
With `tka`, we can define ephemeral access with zero-permission-by-default but still tie in to a kube-native experience.

## How It Works

1. **Login API**: A deployment running in your cluster, reachable only via Tailscale, exposing a login API
2. **Tailscale Identity Validation**: Requests to the login API are authenticated using the Tailscale API and grant syntax (e.g., `user@example.com can access k8s with role read-only`)
3. **Credential Issuance**: The API dynamically provisions a short-lived Kubernetes `ServiceAccount` and a scoped `ClusterRoleBinding`
4. **Kubeconfig Returned**: A time-limited `kubeconfig` is assembled and returned to the user
5. **Automatic Cleanup**: A controller reconciles active logins, cleaning up expired SAs and bindings, keeping your cluster's RBAC config tidy and auditable

## Features

- **Tailnet-only, zero-ingress**: No public endpoints or reverse proxies. All traffic stays on your tailnet via `tsnet`.
- **Capability-driven authorization**: Enforce access with a specific Tailscale ACL capability (role + validity period) for least-privilege by default.
- **Ephemeral credentials**: Per-user ServiceAccounts with short-lived tokens, provisioned on demand and cleaned up on logout/expiry.
- **Kubernetes-native RBAC**: Map capabilities to standard ClusterRoles; no custom auth layer to learn or operate.
- **Simple CLI UX**: `tka login` and `tka kubeconfig` produce a ready-to-use kubeconfig; no manual token wrangling.
- **Built-in observability**: Structured logs, OpenTelemetry tracing, and Prometheus metrics (controller metrics at `/metrics/controller`).
- **Multi-cluster friendly**: Operator options for `clusterName`, `contextPrefix`, and `userPrefix` make per-user contexts and federation patterns straightforward.
- **Small, hackable core**: Minimal moving parts, clear extension points (auth middleware, operator service), and a clean Go codebase.

## Example Flow

```shell
# Step 1: list available clusters
$ tka list
...

# Step 1: login to the cluster
$ tka login aws-us-central-prod

# Step 2: Use Kubernetes as usual
$ kubectl get ns
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

<!-- Links -->

[Tailscale ACLs and Grants]: https://tailscale.com/kb/1393/access-control
[Teleport]: https://goteleport.com
[gh-teleport]: https://github.com/gravitational/teleport
[Tailscale's API server proxy]: https://tailscale.com/kb/1437/kubernetes-operator-api-server-proxy
