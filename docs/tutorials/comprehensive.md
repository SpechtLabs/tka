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

2. ### Step 2: Install TKA CLI

   #### Option A: Download Release (Recommended)

   ```bash
   # Download CLI for your platform
   curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/ts-k8s-auth-$(uname)-$(uname -m) -o ts-k8s-auth

   # Make executable and install
   chmod +x ts-k8s-auth
   sudo mv ts-k8s-auth /usr/local/bin/

   # Test installation
   ts-k8s-auth version
   ```

   #### Option B: Build from Source

   ```bash
   # Clone repository
   git clone https://github.com/SpechtLabs/tka
   cd tka

   # Build CLI
   go build -o bin/ts-k8s-auth ./cmd/cli

   # Test build
   ./bin/ts-k8s-auth version
   ```

3. ### Step 3: Install TKA Server with Helm

   #### Add Helm Repository

   ```bash
   # Add the SpechtLabs Helm repository
   helm repo add spechtlabs https://charts.specht-labs.de
   helm repo update

   # Verify repository
   helm search repo spechtlabs/tka
   ```

   #### Create Namespace

   ```bash
   # Create namespace for TKA
   kubectl create namespace tka-system
   ```

   #### Verify Helm Chart

   ```bash
   # Check CRDs included in chart
   helm template tka spechtlabs/tka | grep -A 5 "kind: CustomResourceDefinition"

   # View all resources that will be created
   helm template tka spechtlabs/tka | kubectl apply --dry-run=client -f -
   ```

4. ### Step 4: Configure TKA Server

   #### Gather Cluster Information

   First, collect the necessary cluster connection details:

   ```bash
   # Get your cluster's API endpoint
   kubectl cluster-info
   # Example output: "Kubernetes control plane is running at https://127.0.0.1:6443"

   # For kind clusters, the endpoint might be different for external access
   # Get the actual external endpoint:
   kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'

   # Get the CA certificate data (base64-encoded)
   kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'
   ```

   #### Create Helm Values File

   Create a values file for your TKA deployment:

   ```yaml
   # values.yaml
   tka:
     tailscale:
       hostname: tka
       port: 443
       tailnet: your-tailnet.ts.net
       capName: specht-labs.de/cap/tka

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

     # Cluster information that will be exposed to authenticated users
     clusterInfo:
       apiEndpoint: "https://127.0.0.1:6443"  # Replace with your cluster's endpoint
       caData: ""  # Add your base64-encoded CA data here, or set insecureSkipTLSVerify: true for testing
       insecureSkipTLSVerify: true  # Set to false and provide caData for production
       labels:
         environment: development
         cluster: kind-tka-demo
         type: demo

     api:
       retryAfterSeconds: 1

   # Resource configuration
   resources:
     requests:
       memory: "256Mi"
       cpu: "100m"
     limits:
       memory: "512Mi"
       cpu: "500m"

   # Enable persistence for Tailscale state
   persistence:
     enabled: true
     size: "1Gi"

   # Secret configuration
   secrets:
     tailscale:
       create: true
       authKey: ""  # Set this or create secret manually
   ```

   #### Create Tailscale Secret

   ```bash
   # Create secret with your Tailscale auth key
   kubectl create secret generic tka-tailscale \
     --from-literal=TS_AUTHKEY=tskey-auth-your-key-here \
     -n tka-system
   ```

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
               "period": "8h",
               "priority": 200
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
               "period": "4h",
               "priority": 100
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

6. ### Step 6: Deploy TKA Server

   #### Install with Helm

   ```bash
   # Deploy TKA using your values file
   helm install tka spechtlabs/tka \
     --namespace tka-system \
     --values values.yaml

   # Or use inline values for quick setup
   helm install tka spechtlabs/tka \
     --namespace tka-system \
     --set tka.tailscale.tailnet=your-tailnet.ts.net
   ```

   #### Verify Deployment

   ```bash
   # Check deployment status
   kubectl get pods -n tka-system -l app.kubernetes.io/name=tka

   # Wait for TKA to be ready
   kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tka -n tka-system

   # Check logs
   kubectl logs -n tka-system -l app.kubernetes.io/name=tka -f
   ```

   Expected log output:

   ```text
   INFO[0000] Starting TKA server...
   INFO[0001] Tailscale connection established
   INFO[0002] Server listening on https://tka.your-tailnet.ts.net:443
   INFO[0003] Kubernetes operator started
   ```

   #### For Production Deployments

   See the [Production Deployment Guide](../how-to/deploy-production.md) for production-ready configurations with monitoring, security hardening, and high availability considerations.

7. ### Step 7: Configure the CLI

   Create a CLI configuration file and install shell integration:

   ```bash
   # Create config directory
   mkdir -p ~/.config/tka

   # Create configuration file
   cat > ~/.config/tka/config.yaml << EOF
   tailscale:
     hostname: tka  # matches Helm chart default
     tailnet: your-tailnet.ts.net
     port: 443
   EOF

   # Install shell integration for the tka wrapper functions
   eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish
   ```

   #### Alternative: Environment Variables

   You can also configure via environment variables (but still need shell integration):

   ```bash
   export TKA_TAILSCALE_HOSTNAME=tka
   export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
   export TKA_TAILSCALE_PORT=443

   # Still install shell integration for tka wrapper functions
   eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish
   ```

   #### URL Construction

   The CLI builds the server URL as:

   - **HTTPS** if port is 443: `https://tka.your-tailnet.ts.net:443`
   - **HTTP** for other ports: `http://tka.your-tailnet.ts.net:8080`
   - **Custom scheme**: prefix hostname with `https://` to force HTTPS

8. ### Step 8: Authenticate and Test

   #### Basic Authentication Flow

   ```bash
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
