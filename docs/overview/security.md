---
title: Security Model
permalink: /overview/security
createTime: 2025/08/25 06:31:28
---

:::caution Experimental Security Model
This project is **experimental**. While I’ve done my best to design it with
security in mind, I’m not a professional security auditor or pentester.

There are **no guarantees** that this setup is bullet‑proof.

If you need **high‑assurance security**, you should use a battle‑tested,
professionally reviewed solution instead.

**Disclaimer:** I am **not liable** for any damages, breaches, or issues that
may arise from using this code. Use it at your own risk.
:::

TKA builds on top of Tailscale’s identity and transport to authenticate users and
control access to Kubernetes. Instead of long-lived credentials that linger
around forever, TKA hands out **short-lived, scoped credentials** that are
provisioned on demand and automatically cleaned up when no longer needed.

Think of it as: *you show up, you get a temporary badge, you do your work, and
the badge self-destructs when you’re done.*

## Core Principles

- **Network identity first** → requests must come in over the tailnet
- **Capability-based authorization** → Tailscale ACLs define *who* can do *what*
- **Ephemeral credentials** → per-user ServiceAccounts with short-lived tokens
- **Kubernetes-native RBAC** → roles are bound using standard RBAC

## Components and Trust

Here’s who does what in the system:

- **TKA Server** → terminates HTTP(s) over `tsnet`, enforces identity via `WhoIs`, and checks for the right ACL capability
- **Auth Middleware** → blocks Funnel traffic, extracts username + capability
- **Auth Service** → delegates provisioning to the operator
- **K8s Operator** → creates/deletes ServiceAccounts & RoleBindings, generates
  tokens, and *never* stores secrets outside the cluster

> [!TIP]
> For a deeper dive, check out the
> [Developer Documentation: Request Flows](../reference/developer/request-flows.md).

## Why ServiceAccounts + Tokens (and not Client Certs)?

Kubernetes gives you two main ways to authenticate:

1. Client certificate auth
2. ServiceAccount + token auth

Normally, client certs are the go-to for human-to-API interactions, while tokens are more common for service-to-API. So why did we pick tokens here?

::: info **Spoiler**
!!Because they’re simpler, safer, and easier to audit.!!{.blur .hover}
:::

- Tokens expire automatically (built-in safety net)
- No need to manage CSRs, approvals, revocations, or CRLs
- ServiceAccounts are first-class citizens in Kubernetes, making auditing straightforward

For example, to see who’s currently active in a cluster:

```bash
kubectl get ServiceAccount -n tka
NAME             SECRETS   AGE
default          0         1d
tka-controller   0         1d
tka-user-cedi    0         47m
```

That’s it - no digging through cert lifecycles.

## Why This Design?

- Leverages Tailscale for mutual auth/user identity (no extra ingress or OIDC setup)
- Keeps Kubernetes as the **source of truth** for authorization (RBAC)
- Uses short-lived credentials → reduces blast radius & credential sprawl
- Explicit capability names tie ACL policy directly to the service, preventing accidental privilege bleed

## Threat Considerations

- Funnel traffic is rejected (no off-tailnet access)
- Tagged nodes are denied (to avoid ambiguous identity semantics)
- Capability JSON is validated (multiple rules for a user = rejected)
- Tokens are generated on demand and never persisted by the server
- Logs include trace IDs; metrics are exposed separately under `/metrics/controller`

## Security-Related Config Knobs

- `tailscale.capName` → capability required from ACLs
- `operator.namespace` → where ServiceAccounts and SignIn resources live
- `api.retryAfterSeconds` → polling guidance (not a security control)
- HTTP timeouts (read/write/idle) → apply to server behavior

In short: TKA keeps things simple, auditable, and secure by leaning on
Tailscale for identity and Kubernetes for authorization — with ephemeral
credentials as the glue.
