---
title: Architecture Overview
permalink: /understanding/architecture
createTime: 2025/01/27 10:00:00
---

TKA is designed to bridge **Tailscale identity** with **Kubernetes RBAC** using
**ephemeral ServiceAccounts** and **short‑lived tokens**.

This page explains the *why* and *how* of the architecture, so you can
understand the moving parts and their responsibilities.

## The Big Picture

At a high level, TKA has four main components:

<!-- markdownlint-disable MD051 -->
- **[TKA CLI](#tka-cli)** → the user‑facing tool that makes authentication feel seamless
- **[TKA API Server](#tka-api-server)** → the entrypoint for users, running inside your tailnet
- **[TKA Operator](#tka-operator)** → a Kubernetes controller that provisions and cleans up ephemeral credentials
- **[TKA Orchestrator](#tka-orchestrator)** → cluster discovery
<!-- markdownlint-enable MD051 -->

Together, they form a loop:

1. A user authenticates via the CLI.
2. The server validates identity and writes a `TkaSignin` resource.
3. The operator reconciles that resource into a ServiceAccount + RBAC binding.
4. The CLI fetches a kubeconfig with a short‑lived token.

```mermaid
sequenceDiagram
    participant cli as User (tka cli)
    participant api as TKA API Server (tailnet only)
    participant acl as Tailscale
    participant k8s as Kubernetes Cluster
    participant operator as TKA K8s Operator

    cli->>api: POST /api/v1alpha1/login
    api->>acl: Validate ACL policy
    api->>k8s: Write TkaSignin CRD
    api-->>cli: Accepted

    operator->>k8s: Observe new signin
    operator->>k8s: Create ServiceAccount + RoleBinding

    cli->>api: GET /api/v1alpha1/kubeconfig
    api->>operator: Request kubeconfig
    operator->>k8s: Generate token
    api-->>cli: Return kubeconfig
```

## Why This Design?

- **Ephemeral by default** → credentials expire automatically, reducing risk
- **Network‑gated** → only accessible inside your tailnet, no public ingress
- **Kubernetes‑native** → uses ServiceAccounts and RBAC, no custom auth layer
- **Separation of concerns** → server handles identity, operator handles Kubernetes resources

This separation keeps the server stateless and auditable, while the operator
owns the lifecycle of in‑cluster resources.

## Component Roles

> [!NOTE]
> See [Developer Documentation: Architecture](../reference/developer/architecture.md) for implementation details

### TKA CLI [^dev-cli]

- Provides a simple UX (`tka login`, `tka shell`)
- Talks to the server, manages kubeconfigs
- Makes ephemeral access feel like a normal `kubectl` workflow

### TKA API Server [^dev-api-srv]

- Runs inside the tailnet, exposes an HTTP API
- Authenticates users via Tailscale WhoIs + ACLs
- Writes `TkaSignin` resources into the cluster
- Returns kubeconfigs with ephemeral tokens

### TKA Operator [^dev-k8s-oper]

- Watches for `TkaSignin` resources
- Creates/deletes ServiceAccounts and RoleBindings
- Generates tokens and cleans up expired sessions

### TKA Orchestrator [^dev-orchestrator]

- Provides cluster discovery

[^dev-cli]: [Developer Architecture Reference | System Components | 1. TKA CLI](../reference/developer/architecture.md#1-tka-cli)
[^dev-api-srv]: [Developer Architecture Reference | System Components | 2. TKA Server](../reference/developer/architecture.md#2-tka-server)
[^dev-k8s-oper]: [Developer Architecture Reference | System Components | 3. TKA Operator (Controller)](../reference/developer/architecture.md#3-tka-operator-controller)
[^dev-orchestrator]: [Developer Architecture Reference | System Components | 4. TKA Orchestrator](../reference/developer/architecture.md#4-tka-orchestrator)

## How It Fits Together

Think of TKA as a bridge:

```mermaid
flowchart LR
  subgraph Left["Tailscale"]
    Tailscale["*who you are*<br/>device + user identity"]
  end

  subgraph Middle["TKA"]
    TKA["*glue logic*<br/>issues short‑lived credentials"]
  end

  subgraph Right["Kubernetes"]
    Kubernetes["*what you can do*<br/>RBAC / Policy Enforcement"]
  end

  Tailscale --> TKA --> Kubernetes
```

- On one side: Tailscale provides *who you are* (device + user identity).
- On the other: Kubernetes enforces *what you can do* (RBAC).
- In the middle: TKA glues them together with short‑lived credentials.

## Where to Go Next

- For **implementation details** (API endpoints, CLI commands, config knobs), see the [Developer Reference](../reference/).
- For **security considerations**, see the [Security Model](./security.md).
- For **deployment guidance**, see the [Comprehensive Guide](../tutorials/comprehensive.md) (includes production deployment).
