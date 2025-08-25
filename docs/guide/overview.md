---
title: Overview
createTime: 2025/07/05 20:46:00
permalink: /guide/overview
---

## Background

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

That said, we needed something just for Kubernetes, something much lighter weight, and most importantly, something that works with our existing Tailscale setup.
Why have two ZTNA systems that provide _almost_ the same features, when you can go out of your way to waste time building your own thing, learn a ton in the process, and make it integrate better into your existing setup?

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

- Access gated via your **Tailscale ACLs + Grants**
- No ingress needed – everything runs **inside your tailnet**
- Short-lived **ephemeral credentials**
- Kubernetes-native RBAC
- Declarative **grant-to-role mappings** via CRDs
- Access your cluster's API even if it's hidden behind a NAT, thanks to Tailscale & a small proxy
- Support for **multi-cluster** federation (future roadmap)

## Components

| Component            | Description                                                            |
|----------------------|------------------------------------------------------------------------|
| `tka` | Login API pod running in the cluster, reachable via Tailscale          |
| `GrantMapping CRD`   | Maps Tailscale identities (user/group/tag) to Kubernetes ClusterRoles  |
| `tka` CLI            | CLI tool to fetch kubeconfigs, list clusters, etc.                     |

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

<!-- Links -->
[Tailscale ACLs and Grants]: https://tailscale.com/kb/1393/access-control
[Teleport]: https://goteleport.com
[gh-teleport]: https://github.com/gravitational/teleport
