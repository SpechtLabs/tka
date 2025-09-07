---
title: Getting Started with TKA (Comprehensive Guide)
permalink: /tutorials/comprehensive
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> There is also a [Getting Started with TKA (Quick Start)](./quick.md) tutorial available

<!-- @include: prerequisites.md -->

## Comprehensive Guide

For more detailed setup with explanations and alternatives:

:::: steps

1. ### Step 1: Prepare Your Environment

   #### Option A: Use Existing Cluster

   If you have a Kubernetes cluster, ensure you can connect:

   ```bash
   kubectl cluster-info
   ```

   #### Option B: Create Test Cluster with kind

   If you need a test cluster:

   ```bash
   # Install kind
   curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-$(uname)-$(uname -m)
   chmod +x ./kind
   sudo mv ./kind /usr/local/bin/kind

   # Create cluster
   kind create cluster --name tka-demo

   # Verify connection
   kubectl get nodes
   ```

2. ### Step 2: Install TKA

   #### Option A: Download Release (Recommended)

   ```bash
   # Download for your platform
   curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/tka-$(uname)-$(uname -m).tar.gz | tar -xz

   # Make executable
   chmod +x tka tka-server

   # Install globally (optional)
   sudo mv tka tka-server /usr/local/bin/
   ```

   #### Option B: Build from Source

   ```bash
   # Clone repository
   git clone https://github.com/SpechtLabs/tka
   cd tka/src

   # Build binaries
   go build -o bin/tka-server ./cmd/server
   go build -o bin/tka ./cmd/cli

   # Test build
   ./bin/tka version
   ./bin/tka-server --help
   ```

3. ### Step 3: Install Kubernetes Resources

   TKA requires Custom Resource Definitions and RBAC:

   ```bash
   # Install from release
   kubectl apply -f https://github.com/spechtlabs/tka/releases/latest/download/tka-k8s.yaml

   # Or from source
   cd tka/src
   make generate
   kubectl apply -k config
   ```

   Verify installation:

   ```bash
   # Check CRDs
   kubectl get crd tkasignins.tka.specht-labs.de

   # Check namespace and RBAC
   kubectl get all -n tka-system
   ```

4. ### Step 4: Configure TKA

   #### Using Environment Variables (Simple)

   ```bash
   export TKA_TAILSCALE_HOSTNAME=tka
   export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
   export TKA_TAILSCALE_PORT=443
   ```

   #### Using Configuration File (Flexible)

   Create `~/.config/tka/config.yaml`:

   ```yaml
   tailscale:
     hostname: tka
     port: 443
     tailnet: your-tailnet.ts.net
     stateDir: /var/lib/tka/tsnet-state

   server:
     readTimeout: 10s
     readHeaderTimeout: 5s
     writeTimeout: 20s
     idleTimeout: 120s

   operator:
     namespace: tka-system
     clusterName: tka-cluster
     contextPrefix: tka-context-
     userPrefix: tka-user-

   api:
     retryAfterSeconds: 1
   ```

   #### Configuration Search Order

   TKA uses this precedence:

   1. Command flags (highest priority)
   2. Environment variables (`TKA_` prefix)
   3. Config files (current dir, `$HOME`, `$HOME/.config/tka/`, `/data`)
   4. Default values (lowest priority)

5. ### Step 5: Configure Tailscale ACLs

   #### Understanding Capability Grants

   TKA uses Tailscale's [capability grants](https://tailscale.com/kb/1324/grants) to map users to Kubernetes roles:

   ```jsonc
   {
     "tagOwners": {
       "tag:tka": ["group:admins", "alice@example.com"]
     },
     "grants": [
       // Admin access for 8 hours
       {
         "src": ["group:admins"],
         "dst": ["tag:tka"],
         "ip": ["443"],
         "app": {
           "specht-labs.de/cap/tka": [
             {
               "role": "cluster-admin",
               "period": "8h"
             }
           ]
         }
       },
       // Developer access for 4 hours
       {
         "src": ["group:developers"],
         "dst": ["tag:tka"],
         "ip": ["443"],
         "app": {
           "specht-labs.de/cap/tka": [
             {
               "role": "view",
               "period": "4h"
             }
           ]
         }
       }
     ]
   }
   ```

   #### Key Points

   - **`tag:tka`**: Tag your TKA servers with this
   - **`src`**: Users/groups who can access TKA
   - **`dst`**: Must target `tag:tka`
   - **`ip`**: Must match the port TKA runs on
   - **`role`**: Must be a valid Kubernetes ClusterRole
   - **`period`**: How long tokens remain valid

6. ### Step 6: Run the TKA Server

   #### For Development (Local Process)

   ```bash
   # Get auth key from Tailscale admin console
   export TS_AUTHKEY=tskey-auth-XXXXXXXXXXXXXXXX

   # Start server
   tka-server serve --server tka --port 443

   # Or with config file
   tka-server serve --config ~/.config/tka/config.yaml
   ```

   #### For Production (Kubernetes Deployment)

   See the [Production Deployment Guide](../how-to/deploy-production.md) for production-ready deployments.

   Expected output:

   ```text
   INFO[0000] Starting TKA server...
   INFO[0001] Tailscale connection established
   INFO[0002] Server listening on https://tka.your-tailnet.ts.net:443
   INFO[0003] Kubernetes operator started
   ```

7. ### Step 7: Configure the CLI

   The CLI needs to know how to reach your TKA server:

   ```bash
   # Via environment variables
   export TKA_TAILSCALE_HOSTNAME=tka
   export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
   export TKA_TAILSCALE_PORT=443

   # Or create ~/.config/tka/config.yaml with the client settings
   ```

   #### URL Construction

   The CLI builds the server URL as:

   - **HTTPS** if port is 443: `https://tka.your-tailnet.ts.net:443`
   - **HTTP** for other ports: `http://tka.your-tailnet.ts.net:8080`
   - **Custom scheme**: prefix hostname with `https://` to force HTTPS

8. ### Step 8: Authenticate and Test

   #### Basic Authentication Flow

   ```bash
   # Authenticate
   tka login
   # ✓ sign-in successful!
   #     ╭───────────────────────────────────────╮
   #     │ User:  alice@example.com              │
   #     │ Role:  cluster-admin                  │
   #     │ Until: Mon, 27 Jan 2025 18:30:00 CET  │
   #     ╰───────────────────────────────────────╯
   # ✓ kubeconfig written to: /tmp/kubeconfig-123456.yaml
   # → To use this session, run: export KUBECONFIG=/tmp/kubeconfig-123456.yaml

   # Use the kubeconfig
   export KUBECONFIG=/tmp/kubeconfig-123456.yaml

   # Test access
   kubectl get pods -A
   kubectl get nodes

   # Check your session
   tka get login

   # Clean up
   tka logout
   # ✓ You have been signed out
   ```

   #### Alternative: Use Subshell

   For isolated access that automatically cleans up:

   ```bash
   # Start subshell with temporary access
   tka shell

   # Inside the subshell, KUBECONFIG is automatically set
   kubectl get pods
   kubectl get nodes

   # Exit cleans up automatically
   exit
   # ✓ You have been signed out
   ```

9. ### Step 9: Verify Everything Works

   #### Check TKA Resources

     ```bash
     # View your signin
     kubectl get tkasignins -n tka-system

     # Check service account
     kubectl get serviceaccounts -n tka-system

     # View role binding
     kubectl get clusterrolebindings | grep tka-user
     ```

   #### Test RBAC

     ```bash
     # Test based on your assigned role
     kubectl auth can-i get pods
     kubectl auth can-i create deployments
     kubectl auth can-i "*" "*"  # Only true for cluster-admin
     ```

::::

<!-- @include: troubleshooting_and_next_steps.md -->
