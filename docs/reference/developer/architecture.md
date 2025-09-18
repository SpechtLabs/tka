---
title: Developer Architecture Reference
permalink: /reference/developer/architecture
createTime: 2025/01/27 10:00:00
---

This page provides a **developer‑oriented view** of TKA’s architecture.

It describes the components, APIs, resources, and control flows in detail.

## System Components

### Package Architecture

TKA follows a clean architecture with well-defined layers:

```text
┌─────────────────────────────────────────────────────────────┐
│ CLI Layer (cmd/cli)                                         │
│ ├── HTTP client requests                                    │
│ └── Shell integration                                       │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ HTTP API Layer (pkg/api)                                    │
│ ├── Gin router and handlers                                 │
│ ├── Request/response models (pkg/models)                    │
│ └── Authentication middleware (pkg/middleware/auth)         │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ Business Logic Layer (pkg/service)                          │
│ ├── Service interface                                       │
│ ├── Operator implementation (pkg/service/operator)          │
│ └── Mock implementation (pkg/service/mock)                  │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ Infrastructure Layer                                        │
│ ├── Kubernetes operator (pkg/operator)                      │
│ ├── Tailscale networking (pkg/tailscale)                    │
│ └── Utility functions (pkg/utils)                           │
└─────────────────────────────────────────────────────────────┘
```

### 1. TKA CLI

**Responsibilities**:

- Provide user‑facing commands (`login`, `logout`, `shell`, etc.)
- Construct API requests to TKA Server
- Manage kubeconfig files
- Integrate with shell environments
- Display session information

**Command Tree**:

```text
tka
├── login          # Authenticate and get kubeconfig
├── logout         # Revoke access and cleanup
├── shell          # Start subshell with temp access
├── kubeconfig     # Fetch current kubeconfig
├── reauthenticate # Refresh credentials
├── get            # Status and information commands
│   ├── login      # Show current login status
│   └── kubeconfig # Alias for kubeconfig command
└── integration    # Generate shell integration code
```

> [!TIP]
> Check out the full [CLI Reference](../cli.md) that is auto generated using the Cobra documentation of each command

### 2. TKA Server

**Responsibilities**:

- Expose REST API over `tsnet` (tailnet‑only)
- Authenticate users via a dedicated Gin middleware that integrates with Tailscale `WhoIs`. The middleware extracts the username and capability from the request context before passing control to the handlers. (See [Request Flows: Tailscale Auth Middleware])
- Validate ACL capabilities (role + validity period)
- Delegate business logic to service layer (`pkg/service`)
- Write `TkaSignin` CRDs into the Kubernetes cluster via operator service
- Return kubeconfigs with ephemeral ServiceAccount tokens

[Request Flows: Tailscale Auth Middleware]: ./request-flows.md#tailscale-auth-middleware

**Implementation**:

- **Networking**: [`pkg/tailscale`](./tailscale-server.md) - Tailscale-only HTTP server
- **HTTP**: Gin router + authentication middleware (`pkg/middleware/auth`)
- **Business Logic**: Service layer (`pkg/service`) with operator implementation
- **Data Models**: Structured API models (`pkg/models`)
- **Kubernetes API**: `client-go` via operator service
- **Observability**: OpenTelemetry (traces, metrics, logs)

**Deployment**:

- Runs as a Kubernetes Deployment
- Stateless (no DB; persistence via Kubernetes API)

**API Endpoints**:

**Authentication API (`/api/v1alpha1`)**:

- `POST /login` → authenticate user, create `TkaSignin`
- `GET /login` → check current authentication status
- `GET /kubeconfig` → return kubeconfig for active session
- `POST /logout` → revoke session

**Orchestrator API (`/orchestrator/v1alpha1`)**:

- `GET /clusters` → list available clusters for user
- `POST /clusters` → register new cluster (future)

### 3. Middleware Layer (`pkg/middleware`)

**Responsibilities**:

- Handle cross-cutting concerns for HTTP requests
- Provide authentication and authorization
- Extract user identity from Tailscale network
- Validate capability rules from Tailscale ACL

**Implementation**:

- **Base Interface**: `middleware.Middleware` - Generic middleware contract
- **Auth Middleware**: `middleware/auth.ginAuthMiddleware` - Tailscale authentication
- **Context Helpers**: `middleware/auth.GetUsername()`, `GetCapability()` - Access user data
- **Mock Support**: `middleware/auth/mock.AuthMiddleware` - Testing middleware

### 4. Service Layer (`pkg/service`)

**Responsibilities**:

- Abstract business logic from HTTP handlers
- Provide stable interface for different implementations
- Handle credential lifecycle management
- Validate business rules and constraints

**Implementation**:

- **Interface**: `service.Service` - Core business operations
- **Production**: `service/operator.Service` - Kubernetes operator integration
- **Testing**: `service/mock.MockAuthService` - Configurable mock for tests
- **Models**: `service.SignInInfo` - Router-agnostic authentication status

### 5. TKA Operator (Controller)

**Responsibilities**:

- Watch `TkaSignin` CRDs
- Create/delete ServiceAccounts
- Bind ClusterRoles/Roles via RoleBindings
- Generate ServiceAccount tokens
- Clean up expired resources

**Implementation**:

- **Framework**: `controller-runtime`
- **Custom Resource**: `TkaSignin`
- **RBAC**: ServiceAccount + RoleBinding management
- **Metrics**: exposed at `/metrics/controller`
- **Integration**: Used by `service/operator.Service`

**Reconciliation Flow**:

```mermaid
stateDiagram-v2
    [*] --> Check: Watch TkaSignin CRD

    Check: Check Status
    Update: Update Status
    Delete: Delete Resources

    state Delete {
        [*] --> RemoveTkaSignin
        RemoveTkaSignin --> RemoveClusterRoleBinding
        RemoveClusterRoleBinding --> [*]
    }

    state Create {
        [*] --> CreateServiceAccount
        CreateServiceAccount --> CreateRoleBinding
        CreateRoleBinding --> Update
        Update --> [*]
    }

    Check --> Create: New/Changed
    Check --> Delete: Expired
```

### 6. TKA Orchestrator

**Responsibilities**:

- Provide cluster discoverability to TKA
- List all clusters available to a user

**Implementation**:

- TBD

## Resource Model

### Custom Resource: `TkaSignin`

Represents a user session request.

**Spec fields**:

- `username`: Tailscale user identity
- `capability`: ACL capability name
- `ttl`: requested session duration

**Status fields**:

- `phase`: Pending | Active | Expired
- `serviceAccount`: name of provisioned SA
- `expiry`: timestamp

### Resource Naming Conventions

- **ServiceAccount**: `tka-user-{sanitized-username}`
  Example: `tka-user-alice`

- **ClusterRoleBinding**: `{serviceAccount}-binding`
  Example: `tka-user-alice-binding`

- **TkaSignin**: `tka-user-{sanitized-username}`
  Example: `tka-user-alice`

## Configuration

### Hierarchy

1. Command flags (highest priority)
2. Environment variables (`TKA_` prefix)
3. Config file (`config.yaml`)
4. Defaults (hardcoded)

### Config File Search Order

1. `--config` flag
2. `./config.yaml`
3. `$HOME/.config/tka/config.yaml`
4. `/etc/tka/config.yaml`

## Observability

### Metrics

**Server (`/metrics`)**:

- HTTP request rate, latency, error counts
- Authentication success/failure
- Token generation counts

**Operator (`/metrics/controller`)**:

- Reconciliation rate and latency
- Resource creation/deletion counters
- Error rates by operation type

### Logging

- Structured JSON logs
- Trace IDs for correlation
- Levels: ERROR, WARN, INFO, DEBUG

### Tracing

- OpenTelemetry spans across server + operator
- Request flow visibility
- Performance bottleneck identification
