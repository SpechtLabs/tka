# Tailscale Kubernetes Auth (TKA)

[![Go Build & Docker Build](https://github.com/spechtlabs/tka/actions/workflows/build.yaml/badge.svg)](https://github.com/spechtlabs/tka/actions/workflows/build.yaml)
[![Documentation](https://github.com/spechtlabs/tka/actions/workflows/docs-website.yaml/badge.svg)](https://github.com/spechtlabs/tka/actions/workflows/docs-website.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/spechtlabs/tka)](https://goreportcard.com/report/github.com/spechtlabs/tka)

> Secure, ephemeral Kubernetes access powered by your Tailscale network and identity.

TKA eliminates the complexity of traditional Kubernetes access control by leveraging your existing Tailscale infrastructure. No auth proxies, no OIDC headaches, no kubeconfig sprawl - just clean, auditable access with short-lived credentials.

**[Full Documentation & Getting Started Guide →](https://tka.specht-labs.de)**

## What is TKA?

TKA (Tailscale Kubernetes Auth) is a zero-trust authentication system that issues short-lived Kubernetes credentials based on your Tailscale identity and ACL grants. It runs entirely within your private Tailscale network with no public endpoints.

### Key Benefits

- **Zero-trust by design** - No public endpoints, access gated by Tailscale ACLs and device attestation
- **Ephemeral credentials** - Short-lived tokens that auto-expire, reducing blast radius
- **Kubernetes-native** - Built on ServiceAccounts, ClusterRoles, and standard APIs
- **Developer-friendly** - `tsh login`-like UX that just works
- **GitOps-ready** - Declarative grant-to-role mapping via CRDs

## Quick Start

```bash
# Install TKA CLI
curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/ts-k8s-auth-linux-amd64 -o ts-k8s-auth
chmod +x ts-k8s-auth && sudo mv ts-k8s-auth /usr/local/bin/

# Deploy TKA server with Helm
helm repo add spechtlabs https://charts.specht-labs.de
helm install tka spechtlabs/tka -n tka-system --create-namespace \
  --set tka.tailscale.tailnet=your-tailnet.ts.net \
  --set secrets.tailscale.authKey=tskey-auth-your-key-here

# Install shell integration for tka wrapper functions
eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish

# Start an ephemeral session (perfect for debugging)
tka shell
(tka) $ kubectl get pods
(tka) $ exit  # Access automatically revoked

# Or login for persistent access (KUBECONFIG auto-managed)
tka login
$ kubectl get namespaces
$ tka logout
```

## How It Works

```mermaid
graph TB
    A[Developer] -->|1 - tka login| B[TKA Server]
    B -->|2 - Validate identity| C[Tailscale API]
    B -->|3 - Check grants| D[Tailscale ACLs]
    B -->|4 - Create ServiceAccount| E[Kubernetes API]
    B -->|5 - Return kubeconfig| A
    F[TKA Operator] -->|6. Cleanup expired tokens| E
```

1. **Identity verification** - TKA validates your Tailscale identity and device
2. **Authorization** - Checks your ACL grants for Kubernetes access permissions
3. **Credential provisioning** - Creates ephemeral ServiceAccount and token
4. **Access granted** - Returns time-limited kubeconfig for direct cluster access
5. **Automatic cleanup** - Expired credentials are automatically revoked

## Architecture

TKA consists of two main components:

- **CLI** (`tka`) - User-facing command for authentication and kubeconfig management
- **Server** - In-cluster service handling authentication and credential issuance
- **Operator** - Kubernetes controller managing ServiceAccount lifecycle

The server runs inside your cluster and is only accessible via your Tailscale network. No ingress controllers, load balancers, or public endpoints required.

## Repository Structure

```text
.
├── cmd/
│   ├── cli/          # TKA CLI implementation
│   └── server/       # TKA server and operator
├── pkg/
│   ├── api/          # HTTP API handlers
│   ├── auth/         # Authentication middleware
│   ├── operator/     # Kubernetes operator logic
│   └── tailscale/    # Tailscale integration
└── internal/
    └── cli/          # CLI-specific utilities
```

## Development

### Prerequisites

- Go 1.21+
- Kubernetes cluster (for testing)
- Tailscale account and tailnet

### Building

```bash
# Build CLI
make build-cli

# Build server
make build-server

# Run tests
make test

# Generate documentation
make docs
```

### Running Locally

```bash
# Start development server
make dev-server

# Run CLI against local server
export TKA_TAILSCALE_HOSTNAME=localhost
export TKA_TAILSCALE_PORT=8080
./bin/tka-cli login
```

## Security Model

TKA's security model is built on several key principles:

- **Network isolation** - All communication happens within your private Tailscale network
- **Device attestation** - Access requires authenticated Tailscale device
- **Ephemeral credentials** - Short-lived tokens minimize exposure window
- **Principle of least privilege** - Explicit grant mapping to Kubernetes roles
- **Audit trail** - All access is logged and attributable to specific users/devices

> [!WARNING]
> TKA's security model is thoughtfully designed but still evolving.
> While suitable for many use cases, it hasn't undergone professional security auditing.
>
> See the [Security Documentation](https://tka.specht-labs.de/explanation/security) for details.

## Comparison with Alternatives

| Feature | TKA | Teleport | Tailscale K8s Operator |
|---------|-----|----------|-------------------------|
| **Zero public endpoints** | ✅ | ❌ | ✅ |
| **Ephemeral credentials** | ✅ | ✅ | ❌ |
| **Tailscale native** | ✅ | ❌ | ✅ |
| **Multi-protocol support** | ❌ | ✅ | ❌ |
| **Session recording** | ❌ | ✅ | ❌ |
| **Lightweight deployment** | ✅ | ❌ | ✅ |

## Documentation

- **[Full Documentation](https://tka.specht-labs.de)** - Comprehensive guides and reference
- **[Quick Start Tutorial](https://tka.specht-labs.de/tutorials/quick)** - Get up and running in 5 minutes
- **[Production Deployment](https://tka.specht-labs.de/how-to/deploy-production)** - Production-ready setup guide
- **[Configuration Reference](https://tka.specht-labs.de/reference/configuration)** - All configuration options
- **[Troubleshooting](https://tka.specht-labs.de/how-to/troubleshooting)** - Common issues and solutions

### Reporting Issues

Found a bug or have a feature request? Please check existing [issues](https://github.com/spechtlabs/tka/issues) first, then [open a new issue](https://github.com/spechtlabs/tka/issues/new/choose) with details.

## Acknowledgments

- **[Tailscale](https://tailscale.com)** - For the amazing zero-trust networking platform
- **[Teleport](https://goteleport.com)** - Inspiration for the UX and security model

---

**Built by SREs, for SREs.** TKA is designed for real-world production operations with security, reliability, and developer experience as top priorities.
