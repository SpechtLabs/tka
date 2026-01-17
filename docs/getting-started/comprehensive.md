---
title: Comprehensive Guide to TKA
permalink: /getting-started/comprehensive
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> For a minimal setup, see the [Quick Start Guide](./quick.md)

This guide covers both development and production deployments with detailed explanations, alternatives, and best practices.

:::: steps

1. ## Step 1: Prepare Your Environment

   ### Option A: Use Existing Cluster

   If you have a Kubernetes cluster, ensure you can connect:

   ::: terminal Check cluster connection

   ```bash
   kubectl cluster-info
   ```

   :::

   ### Option B: Create Test Cluster with kind

   If you need a test cluster:

   ::: terminal Create test cluster with kind

   ```bash
   # Install kind
   $ curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-$(uname)-$(uname -m)
   $ chmod +x ./kind
   $ sudo mv ./kind /usr/local/bin/kind

   # Create cluster
   $ kind create cluster --name tka-demo

   # Verify connection
   $ kubectl get nodes
   ```

   :::

2. ## Step 2: Install TKA CLI

   ### Option A: Download Release (Recommended)

   ::: terminal Download and install TKA CLI

   ```bash
   # Download CLI for your platform
   $ curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/ts-k8s-auth-$(uname)-$(uname -m) -o ts-k8s-auth

   # Make executable and install
   $ chmod +x ts-k8s-auth
   $ sudo mv ts-k8s-auth /usr/local/bin/

   # Test installation
   $ ts-k8s-auth version
   ```

   :::

   ### Option B: Build from Source

   ::: terminal Build TKA CLI from source

   ```bash
   # Clone repository
   $ git clone https://github.com/SpechtLabs/tka
   $ cd tka

   # Build CLI
   $ go build -o bin/ts-k8s-auth ./cmd/cli

   # Test build
   $ ./bin/ts-k8s-auth version
   ```

   :::

3. ## Step 3: Install TKA Server with Helm

   ### Add Helm Repository

   ::: terminal Add Helm repository

   ```bash
   # Add the SpechtLabs Helm repository
   $ helm repo add spechtlabs https://charts.specht-labs.de
   $ helm repo update

   # Verify repository
   $ helm search repo spechtlabs/tka
   ```

   :::

   ### Create Namespace

   ::: terminal Create namespace

   ```bash
   # Create namespace for TKA
   $ kubectl create namespace tka-system
   ```

   :::

   ### Verify Helm Chart

   ::: terminal Verify Helm chart

   ```bash
   # Check CRDs included in chart
   $ helm template tka spechtlabs/tka | grep -A 5 "kind: CustomResourceDefinition"

   # View all resources that will be created
   $ helm template tka spechtlabs/tka | kubectl apply --dry-run=client -f -
   ```

   :::

4. ## Step 4: Configure TKA Server

   ### Gather Cluster Information

   First, collect the necessary cluster connection details:

   ::: terminal Gather cluster information

   ```bash
   # Get your cluster's API endpoint
   $ kubectl cluster-info
   # Example output: "Kubernetes control plane is running at https://127.0.0.1:6443"

   # For kind clusters, the endpoint might be different for external access
   # Get the actual external endpoint:
   $ kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}'

   # Get the CA certificate data (base64-encoded)
   $ kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'
   ```

   :::

   ### Create Helm Values File

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

   ### Create Tailscale Secret

   ::: terminal Create Tailscale secret

   ```bash
   # Create secret with your Tailscale auth key
   $ kubectl create secret generic tka-tailscale --from-literal=TS_AUTHKEY=tskey-auth-your-key-here -n tka-system
   ```

   :::

5. ## Step 5: Configure Tailscale ACLs

   ### Understanding Capability Grants

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

   ### Key Points

   - **`tag:tka`**: Tag your TKA servers with this
   - **`src`**: Users/groups who can access TKA
   - **`dst`**: Must target `tag:tka`
   - **`ip`**: Must match the port TKA runs on
   - **`role`**: Must be a valid Kubernetes ClusterRole
   - **`period`**: How long tokens remain valid

6. ## Step 6: Deploy TKA Server

   ### Install with Helm

   ::: terminal Deploy TKA with Helm

   ```bash
   # Deploy TKA using your values file
   $ helm install tka spechtlabs/tka --namespace tka-system --values values.yaml

   # Or use inline values for quick setup
   $ helm install tka spechtlabs/tka --namespace tka-system --set tka.tailscale.tailnet=your-tailnet.ts.net
   ```

   :::

   ### Verify Deployment

   ::: terminal Verify deployment

   ```bash
   # Check deployment status
   $ kubectl get pods -n tka-system -l app.kubernetes.io/name=tka

   # Wait for TKA to be ready
   $ kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tka -n tka-system

   # Check logs
   $ kubectl logs -n tka-system -l app.kubernetes.io/name=tka -f
   ```

   :::

   Expected log output:

   ```text
   INFO[0000] Starting TKA server...
   INFO[0001] Tailscale connection established
   INFO[0002] Server listening on https://tka.your-tailnet.ts.net:443
   INFO[0003] Kubernetes operator started
   ```

   ### Production Deployment Considerations

   For production environments, consider these additional configurations:

   - **Resource limits**: Increase memory/CPU requests and limits
   - **Security context**: Enable non-root user and read-only filesystem
   - **Persistence**: Use production-grade storage classes
   - **Monitoring**: Enable ServiceMonitor for Prometheus
   - **Network policies**: Restrict ingress/egress traffic
   - **Node placement**: Use node selectors and tolerations

   See the production configuration section below for detailed setup.

7. ## Step 7: Configure the CLI

   Create a CLI configuration file and install shell integration:

   ::: terminal Configure TKA CLI

   ```bash
   # Create config directory
   $ mkdir -p ~/.config/tka

   # Create configuration file
   $ cat > ~/.config/tka/config.yaml << EOF
   tailscale:
     hostname: tka  # matches Helm chart default
     tailnet: your-tailnet.ts.net
     port: 443
   EOF

   # Install shell integration for the tka wrapper functions
   $ eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish
   ```

   :::

   ### Alternative: Environment Variables

   You can also configure via environment variables (but still need shell integration):

   ::: terminal Alternative: Environment variables

   ```bash
   $ export TKA_TAILSCALE_HOSTNAME=tka
   $ export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
   $ export TKA_TAILSCALE_PORT=443

   # Still install shell integration for tka wrapper functions
   $ eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish
   ```

   :::

   ### URL Construction

   The CLI builds the server URL as:

   - **HTTPS** if port is 443: `https://tka.your-tailnet.ts.net:443`
   - **HTTP** for other ports: `http://tka.your-tailnet.ts.net:8080`
   - **Custom scheme**: prefix hostname with `https://` to force HTTPS

8. ## Step 8: Authenticate and Test

   ### Basic Authentication Flow

   ::: terminal Basic authentication flow

   ```bash
   $ tka login
   ✓ sign-in successful!
       ╭───────────────────────────────────────╮
       │ User:  alice@example.com              │
       │ Role:  cluster-admin                  │
       │ Until: Mon, 27 Jan 2025 18:30:00 CET  │
       ╰───────────────────────────────────────╯
   ✓ kubeconfig written to: /tmp/kubeconfig-123456.yaml
   → To use this session, run: export KUBECONFIG=/tmp/kubeconfig-123456.yaml

   $ export KUBECONFIG=/tmp/kubeconfig-123456.yaml

   $ kubectl get pods -A
   $ kubectl get nodes

   $ tka get login

   $ tka logout
   ✓ You have been signed out
   ```

   :::

   ### Alternative: Use Subshell

   For isolated access that automatically cleans up:

   ::: terminal Use subshell for isolated access

   ```bash
   # Start subshell with temporary access
   $ tka shell

   # Inside the subshell, KUBECONFIG is automatically set
   (tka) $ kubectl get pods
   (tka) $ kubectl get nodes

   # Exit cleans up automatically
   (tka) $ exit
   ✓ You have been signed out
   ```

   :::

9. ## Step 9: Verify Everything Works

   ### Check TKA Resources

     ::: terminal Check TKA resources

     ```bash
     # View your signin
     $ kubectl get tkasignins -n tka-system

     # Check service account
     $ kubectl get serviceaccounts -n tka-system

     # View role binding
     $ kubectl get clusterrolebindings | grep tka-user
     ```

     :::

   ### Test RBAC

     ::: terminal Test RBAC permissions

     ```bash
     # Test based on your assigned role
     $ kubectl auth can-i get pods
     $ kubectl auth can-i create deployments
     $ kubectl auth can-i "*" "*"  # Only true for cluster-admin
     ```

     :::

::::

## Production Deployment

For production environments, TKA requires additional configuration for security, monitoring, and reliability.

### Architecture Overview

Production TKA deployments typically use:

- **Primary Login Cluster**: Single cluster hosting the TKA API (usually `tka.your-tailnet.ts.net`)
- **Federated Clusters**: Additional clusters that register with the primary for authentication
- **External Secrets**: Secure storage for Tailscale auth keys
- **Monitoring Stack**: Observability for the TKA components

### Production Configuration

Create a production values file with enhanced security and monitoring:

```yaml
# values-production.yaml
tka:
  tailscale:
    hostname: prod-cluster
    tailnet: your-tailnet.ts.net

  server:
    readTimeout: 30s
    readHeaderTimeout: 10s
    writeTimeout: 60s
    idleTimeout: 300s

  operator:
    namespace: tka-system

  # Cluster information exposed to authenticated users
  clusterInfo:
    apiEndpoint: "https://api.prod-cluster.example.com:6443"
    caData: "LS0tLS1CRUdJTi... (base64-encoded CA certificate)"
    insecureSkipTLSVerify: false
    labels:
      environment: production
      region: us-west-2
      cluster: prod-cluster
      team: platform

  otel:
    endpoint: jaeger-collector.monitoring.svc.cluster.local:14250
    insecure: false

# Production resource limits
resources:
  requests:
    memory: "512Mi"
    cpu: "200m"
  limits:
    memory: "1Gi"
    cpu: "1000m"

# Persistence for Tailscale state
persistence:
  enabled: true
  size: "2Gi"
  storageClass: "fast-ssd"  # Adjust for your cluster

# Security configuration
podSecurityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 2000

securityContext:
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000
  capabilities:
    drop:
    - ALL

# Node placement for production
nodeSelector:
  node-role.kubernetes.io/control-plane: ""

tolerations:
- key: "node-role.kubernetes.io/control-plane"
  operator: "Exists"
  effect: "NoSchedule"

# Monitoring
serviceMonitor:
  enabled: true

# Network policies for security
networkPolicy:
  enabled: true
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # Tailscale control plane
    - protocol: TCP
      port: 6443  # Kubernetes API
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080  # Metrics

# Secret configuration
secrets:
  tailscale:
    # Use existing secret for production
    create: false
    secretName: tka-tailscale-prod
```

### Security Hardening

#### Pod Security Standards

Apply Pod Security Standards to the namespace:

::: terminal Apply Pod Security Standards

```bash
# Apply PSS labels to namespace
$ kubectl label namespace tka-system pod-security.kubernetes.io/enforce=restricted pod-security.kubernetes.io/audit=restricted pod-security.kubernetes.io/warn=restricted
```

:::

#### Secure Secrets Management

Store the Tailscale auth key securely:

::: terminal Create production Tailscale secret

```bash
# Create secret for auth key
$ kubectl create secret generic tka-tailscale-prod --from-literal=TS_AUTHKEY=tskey-auth-your-production-key -n tka-system
```

:::

### Monitoring and Observability

#### Prometheus Monitoring

The Helm chart automatically creates a ServiceMonitor when enabled:

```yaml
serviceMonitor:
  enabled: true
  labels:
    release: prometheus  # Must match your Prometheus selector
```

#### Key Metrics to Monitor

- **Request latency and error rates**: Standard HTTP metrics
- **User authentication metrics**:
  - `tka_login_attempts_total`: Login attempts by cluster role and outcome
  - `tka_active_user_sessions`: Current active sessions by cluster role
  - `tka_user_signins_total`: Total successful sign-ins by cluster role and username
- **ServiceAccount creation/deletion rates**: Kubernetes resource metrics
- **Controller reconciliation metrics**: `tka_reconciler_duration`
- **Resource consumption**: Memory, CPU, and storage metrics

#### Alerting Rules

TKA includes built-in Prometheus alerting rules for security monitoring:

```yaml
# values-production.yaml
prometheusRule:
  enabled: true
  labels:
    release: prometheus  # Must match your Prometheus selector
  privilegedRoleAlert:
    enabled: true       # Enable/disable privileged role monitoring
    clusterRole: cluster-admin  # Configure any role to monitor
    severity: critical
    duration: 0s        # Immediate alert for privileged role usage
    maxActiveSessions: 1
    runbookUrl: "https://your-company.com/runbooks/tka-privileged-role-login"
  serverDownAlert:
    enabled: true       # Enable/disable server health monitoring
  errorRateAlert:
    enabled: false      # Disable if not needed in production
  forbiddenRateAlert:
    enabled: true       # Enable security monitoring
```

Key alerts included:

- **TKAPrivilegedRoleLogin**: Triggers when someone logs in with the configured privileged role
- **TKAMultiplePrivilegedSessions**: Alerts when multiple sessions of the privileged role are active
- **TKAServerDown**: Critical alert when TKA server is unavailable
- **TKAHighErrorRate**: Warning when login error rate exceeds threshold
- **TKAHighForbiddenRate**: Security alert for potential unauthorized access attempts

### Production Deployment

::: terminal Deploy TKA for production

```bash
# Install TKA using production values
$ helm install tka spechtlabs/tka --namespace tka-system --values values-production.yaml

# Verify deployment
$ kubectl get pods -n tka-system -l app.kubernetes.io/name=tka
$ kubectl logs -n tka-system -l app.kubernetes.io/name=tka -f
```

:::

### High Availability Considerations

> [!IMPORTANT]
> **Scaling Limitations**: TKA cannot scale beyond 1 replica due to Tailscale node identity conflicts. Each Tailscale node requires a unique identity.

#### Backup and Recovery

::: terminal Backup TKA configuration

```bash
# Backup Helm values and TKA secrets
$ helm get values tka -n tka-system > tka-values-backup.yaml
$ kubectl get secret tka-tailscale-prod -n tka-system -o yaml > tka-secret-backup.yaml

# Backup CRDs and custom resources
$ kubectl get crd tkasignins.tka.specht-labs.de -o yaml > tka-crd-backup.yaml
$ kubectl get tkasignins -A -o yaml > tka-signins-backup.yaml
```

:::

### Maintenance and Updates

#### Helm-based Updates

::: terminal Update TKA with Helm

```bash
# Update Helm repository
$ helm repo update

# Upgrade TKA release
$ helm upgrade tka spechtlabs/tka --namespace tka-system --values values-production.yaml

# Monitor rollout
$ kubectl rollout status deployment/tka -n tka-system
```

:::
