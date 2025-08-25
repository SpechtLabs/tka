---
title: System Overview
permalink: /architecture/overview
createTime: 2025/08/25 06:33:41
---

### Background

Kubernetes access is often either cumbersome (OIDC wiring, kubeconfig sprawl) or centralized and complex (auth proxies, bastions). TKA aims to provide a simpler, network-gated approach using Tailscale identity, issuing ephemeral, scoped credentials on demand.

### How it works (high level)

1. A TKA server runs inside your tailnet and exposes a minimal HTTP API over `tsnet`.
2. Incoming requests are authenticated via Tailscale `WhoIs` and authorized by a capability in your ACLs naming a role and validity period.
3. The operator provisions a short-lived ServiceAccount and binds appropriate RBAC.
4. The server returns a kubeconfig that uses the ephemeral token; the operator cleans up resources on logout or expiry.

### Why this model

- Reuses Tailscale for network identity and transport; no public ingress required
- Keeps Kubernetes as the source of truth for authorization via standard RBAC
- Uses short-lived credentials to reduce blast radius and simplify revocation

### Components

- TKA Server (Gin over `tsnet`): HTTP API, Swagger docs, metrics, tracing
- Auth Middleware: validates tailnet identity, enforces capability presence
- Auth Service: bridges API calls to the operator
- K8s Operator: creates/deletes ServiceAccounts and RoleBindings; generates tokens; builds kubeconfigs

### Notable features

- Access gated by Tailscale ACLs and an explicit capability name
- Ephemeral, scoped ServiceAccount tokens
- No ingress; tailnet-only connectivity
- Kubernetes-native RBAC
