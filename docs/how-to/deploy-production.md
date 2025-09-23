---
title: Production Deployment
permalink: /how-to/deploy-production
createTime: 2025/01/27 10:00:00
---

Deploy TKA in a production environment with high availability, monitoring, and security best practices using the official Helm chart.

## Architecture Overview

Production TKA deployments typically use:

- **Primary Login Cluster**: Single cluster hosting the TKA API (usually `tka.your-tailnet.ts.net`)
- **Federated Clusters**: Additional clusters that register with the primary for authentication
- **External Secrets**: Secure storage for Tailscale auth keys
- **Monitoring Stack**: Observability for the TKA components

:::: steps

1. ## Helm Repository Setup

   ### Add TKA Helm Repository

   ```bash
   # Add the SpechtLabs Helm repository
   helm repo add spechtlabs https://charts.specht-labs.de
   helm repo update
   ```

   ### Create Namespace

   ```bash
   # Create namespace for TKA
   kubectl create namespace tka-system
   ```

2. ## Production Configuration

   ### Gather Cluster Information

   Before creating the values file, you'll need to gather the cluster connection details:

   ```bash
   # Get your cluster's API endpoint
   kubectl cluster-info
   # Look for: "Kubernetes control plane is running at https://..."

   # Get the CA certificate data (base64-encoded)
   kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'

   # Alternative: Get CA from secret if using service account
   kubectl get secret -n kube-system -o jsonpath='{.data.ca\.crt}' <service-account-secret>
   ```

   ### Create Values File

   Create a production values file:

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

   ### Secure Secrets Management

   Store the Tailscale auth key securely:

   ```bash
   # Create secret for auth key
   kubectl create secret generic tka-tailscale-prod \
     --from-literal=TS_AUTHKEY=tskey-auth-your-production-key \
     -n tka-system
   ```

3. ## Deploy TKA with Helm

   ### Install TKA

   ```bash
   # Install TKA using Helm
   helm install tka spechtlabs/tka \
     --namespace tka-system \
     --values values-production.yaml

   # Verify deployment
   kubectl get pods -n tka-system
   kubectl get services -n tka-system
   ```

   ### Verify Installation

   ```bash
   # Check TKA pods
   kubectl get pods -n tka-system -l app.kubernetes.io/name=tka

   # Check logs
   kubectl logs -n tka-system -l app.kubernetes.io/name=tka -f

   # Verify CRDs
   kubectl get crd tkasignins.tka.specht-labs.de
   ```

4. ## Monitoring and Observability

   ### Prometheus Monitoring

   The Helm chart automatically creates a ServiceMonitor when enabled in your values:

   ```yaml
   # Already included in values-production.yaml above
   serviceMonitor:
     enabled: true
     labels:
       release: prometheus  # Must match your Prometheus selector
   ```

   ### Grafana Dashboard

   Key metrics to monitor:

   - **Request latency and error rates**: Standard HTTP metrics
   - **User authentication metrics**:
     - `tka_login_attempts_total`: Login attempts by cluster role and outcome (universal metric for any role)
     - `tka_active_user_sessions`: Current active sessions by cluster role
     - `tka_user_signins_total`: Total successful sign-ins by cluster role and username
   - **ServiceAccount creation/deletion rates**: Kubernetes resource metrics
   - **Controller reconciliation metrics**: `tka_reconciler_duration`
   - **Resource consumption**: Memory, CPU, and storage metrics

   ### Alerting Rules

   TKA includes built-in Prometheus alerting rules for security monitoring. Enable them in your Helm values:

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

   - **TKAPrivilegedRoleLogin**: Triggers when someone logs in with the configured privileged role (uses `tka_login_attempts_total` metric)
   - **TKAMultiplePrivilegedSessions**: Alerts when multiple sessions of the privileged role are active
   - **TKAServerDown**: Critical alert when TKA server is unavailable
   - **TKAHighErrorRate**: Warning when login error rate exceeds threshold
   - **TKAHighForbiddenRate**: Security alert for potential unauthorized access attempts

   **Individual Alert Control**: Each alert type can be individually enabled or disabled using the `enabled` flag. This allows you to:
   - Enable only security-critical alerts in production
   - Enable all alerts in development/lab environments
   - Customize alerting per environment needs

   The privileged role alerts are universally configurable - you can monitor any cluster role (cluster-admin, admin, edit, etc.) by changing the `clusterRole` parameter. This is particularly useful for break-glass account monitoring, automatically triggering security alerts when privileged access is used.

5. ## Security Hardening

   ### Network Policies

   Network policies are automatically configured when enabled in your values:

   ```yaml
   # Already included in values-production.yaml above
   networkPolicy:
     enabled: true
     # Egress and ingress rules are pre-configured for security
   ```

   ### Pod Security Standards

   Apply Pod Security Standards to the namespace:

   ```bash
   # Apply PSS labels to namespace
   kubectl label namespace tka-system \
     pod-security.kubernetes.io/enforce=restricted \
     pod-security.kubernetes.io/audit=restricted \
     pod-security.kubernetes.io/warn=restricted
   ```

   ### Security Context

   The Helm chart includes production-ready security contexts:

   ```yaml
   # Already included in values-production.yaml above
   podSecurityContext:
     runAsNonRoot: true
     runAsUser: 1000
     fsGroup: 2000

   securityContext:
     allowPrivilegeEscalation: false
     readOnlyRootFilesystem: true
     # ... additional hardening
   ```

6. ## High Availability Considerations

   > [!IMPORTANT]
   > **Scaling Limitations**: TKA cannot scale beyond 1 replica due to Tailscale node identity conflicts. Each Tailscale node requires a unique identity.

   ### Persistence and State Management

   The Helm chart manages persistence automatically:

   ```yaml
   # Already included in values-production.yaml above
   persistence:
     enabled: true
     size: "2Gi"
     storageClass: "fast-ssd"
   ```

   ### Backup and Recovery

   ```bash
   # Backup Helm values and TKA secrets
   helm get values tka -n tka-system > tka-values-backup.yaml
   kubectl get secret tka-tailscale-prod -n tka-system -o yaml > tka-secret-backup.yaml

   # Backup CRDs and custom resources
   kubectl get crd tkasignins.tka.specht-labs.de -o yaml > tka-crd-backup.yaml
   kubectl get tkasignins -A -o yaml > tka-signins-backup.yaml
   ```

7. ## Maintenance and Updates

   ### Helm-based Updates

   ```bash
   # Update Helm repository
   helm repo update

   # Upgrade TKA release
   helm upgrade tka spechtlabs/tka \
     --namespace tka-system \
     --values values-production.yaml

   # Monitor rollout
   kubectl rollout status deployment/tka -n tka-system
   ```

   ### Configuration Updates

   ```bash
   # Update values file and upgrade
   helm upgrade tka spechtlabs/tka \
     --namespace tka-system \
     --values values-production.yaml

   # Check deployment status
   helm status tka -n tka-system
   ```

::::

## Troubleshooting Production Issues

### Common Issues

1. **Tailscale connectivity**: Check auth key validity and network policies
2. **Certificate issues**: Verify HTTPS is enabled in your tailnet
3. **RBAC issues**: Ensure service account has proper permissions
4. **Resource limits**: Monitor memory and CPU usage

### Debug Commands

```bash
# Check server logs (using Helm labels)
kubectl logs -n tka-system -l app.kubernetes.io/name=tka -f

# Check Helm release status
helm status tka -n tka-system

# List TKA resources
helm get all tka -n tka-system

# Verify connectivity
kubectl exec -it deployment/tka -n tka-system -- curl -k https://tka.your-tailnet.ts.net/metrics
```

## Next Steps

- [Multi-cluster Setup](./multi-cluster-setup.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [Security Best Practices](../explanation/security.md)
