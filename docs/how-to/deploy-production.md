---
title: Production Deployment
permalink: /how-to/deploy-production
createTime: 2025/01/27 10:00:00
---

Deploy TKA in a production environment with high availability, monitoring, and security best practices.

## Architecture Overview

Production TKA deployments typically use:

- **Primary Login Cluster**: Single cluster hosting the TKA API (usually `tka.your-tailnet.ts.net`)
- **Federated Clusters**: Additional clusters that register with the primary for authentication
- **External Secrets**: Secure storage for Tailscale auth keys
- **Monitoring Stack**: Observability for the TKA components

:::: steps

1. ## Primary Login Cluster Setup

   ### Install CRDs and Operator

   ```bash
   # Create namespace
   kubectl create namespace tka-system

   # Install TKA resources
   kubectl apply -f https://github.com/spechtlabs/tka/releases/latest/download/tka-k8s.yaml
   ```

   ### Prepare Configuration

   Create a production configuration:

   ```yaml
   # tka-config.yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: tka-config
     namespace: tka-system
   data:
     config.yaml: |
       tailscale:
         hostname: tka
         port: 443
         tailnet: your-tailnet.ts.net
         capName: specht-labs.de/cap/tka

       server:
         readTimeout: 30s
         readHeaderTimeout: 10s
         writeTimeout: 60s
         idleTimeout: 300s

       operator:
         namespace: tka-system
         clusterName: prod-login-cluster
         contextPrefix: tka-
         userPrefix: tka-user-

       otel:
         endpoint: jaeger-collector.monitoring.svc.cluster.local:14250
         insecure: false
   ```

   ### Secure Secrets Management

   Store the Tailscale auth key securely:

   ```bash
   # Create secret for auth key
   kubectl create secret generic tka-tailscale \
     --from-literal=TS_AUTHKEY=tskey-auth-your-production-key \
     -n tka-system
   ```

   ### Deploy TKA Server

   ```yaml
   # tka-deployment.yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: tka-server
     namespace: tka-system
     labels:
       app: tka-server
   spec:
     replicas: 2
     selector:
       matchLabels:
         app: tka-server
     template:
       metadata:
         labels:
           app: tka-server
       spec:
         serviceAccountName: tka-controller
         containers:
         - name: tka-server
           image: ghcr.io/spechtlabs/tka:latest
           command: ["/tka-server"]
           args: ["serve", "--config", "/etc/tka/config.yaml"]
           ports:
           - containerPort: 443
             name: https
           - containerPort: 8080
             name: metrics
           env:
           - name: TS_AUTHKEY
             valueFrom:
               secretKeyRef:
                 name: tka-tailscale
                 key: TS_AUTHKEY
           volumeMounts:
           - name: config
             mountPath: /etc/tka
           - name: tsnet-state
             mountPath: /var/lib/tsnet
           resources:
             requests:
               memory: "256Mi"
               cpu: "100m"
             limits:
               memory: "512Mi"
               cpu: "500m"
           readinessProbe:
             httpGet:
               path: /metrics
               port: 8080
             initialDelaySeconds: 10
             periodSeconds: 5
           livenessProbe:
             httpGet:
               path: /metrics
               port: 8080
             initialDelaySeconds: 30
             periodSeconds: 10
         volumes:
         - name: config
           configMap:
             name: tka-config
         - name: tsnet-state
           persistentVolumeClaim:
             claimName: tka-tsnet-state
   ---
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: tka-tsnet-state
     namespace: tka-system
   spec:
     accessModes:
       - ReadWriteOnce
     resources:
       requests:
         storage: 1Gi
   ```

2. ## Service and Ingress

   ```yaml
   # tka-service.yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: tka-server
     namespace: tka-system
     labels:
       app: tka-server
   spec:
     ports:
     - port: 443
       targetPort: 443
       name: https
     - port: 8080
       targetPort: 8080
       name: metrics
     selector:
       app: tka-server
   ```

3. ## Monitoring and Observability

   ### Prometheus ServiceMonitor

   ```yaml
   # tka-servicemonitor.yaml
   apiVersion: monitoring.coreos.com/v1
   kind: ServiceMonitor
   metadata:
     name: tka-server
     namespace: tka-system
   spec:
     selector:
       matchLabels:
         app: tka-server
     endpoints:
     - port: metrics
       interval: 30s
       path: /metrics
     - port: metrics
       interval: 30s
       path: /metrics/controller
   ```

   ### Grafana Dashboard

   Key metrics to monitor:

   - Request latency and error rates
   - Active user sessions
   - ServiceAccount creation/deletion rates
   - Controller reconciliation metrics
   - Resource consumption

   ### Alerting Rules

   ```yaml
   # tka-alerts.yaml
   apiVersion: monitoring.coreos.com/v1
   kind: PrometheusRule
   metadata:
     name: tka-alerts
     namespace: tka-system
   spec:
     groups:
     - name: tka
       rules:
       - alert: TKAServerDown
         expr: up{job="tka-server"} == 0
         for: 1m
         labels:
           severity: critical
         annotations:
           summary: "TKA server is down"
           description: "TKA server has been down for more than 1 minute"

       - alert: TKAHighErrorRate
         expr: rate(tka_server_requests_total{status=~"5.."}[5m]) > 0.1
         for: 2m
         labels:
           severity: warning
         annotations:
           summary: "High error rate in TKA server"
           description: "TKA server error rate is {{ $value }} errors per second"
   ```

4. ## Security Hardening

   ### Network Policies

   ```yaml
   # tka-networkpolicy.yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: tka-server
     namespace: tka-system
   spec:
     podSelector:
       matchLabels:
         app: tka-server
     policyTypes:
     - Ingress
     - Egress
     ingress:
     - from:
       - namespaceSelector:
           matchLabels:
             name: monitoring
       ports:
       - protocol: TCP
         port: 8080
     egress:
     - to: []
       ports:
       - protocol: TCP
         port: 443
       - protocol: TCP
         port: 6443
   ```

   ### Pod Security Standards

   ```yaml
   # tka-pss.yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: tka-system
     labels:
       pod-security.kubernetes.io/enforce: restricted
       pod-security.kubernetes.io/audit: restricted
       pod-security.kubernetes.io/warn: restricted
   ```

   ### RBAC Hardening

   Review and minimize the RBAC permissions granted to the TKA controller service account.

5. ## High Availability Considerations

   ### Database (if using external state)

   For multi-replica deployments, consider:

   - Shared storage for tsnet state
   - External etcd for Kubernetes state
   - Load balancing across replicas

   ### Backup and Recovery

   ```bash
   # Backup TKA configuration and secrets
   kubectl get configmap tka-config -n tka-system -o yaml > tka-config-backup.yaml
   kubectl get secret tka-tailscale -n tka-system -o yaml > tka-secret-backup.yaml

   # Backup CRDs
   kubectl get crd tkasignins.tka.specht-labs.de -o yaml > tka-crd-backup.yaml
   ```

6. ## Multi-cluster Federation

   For additional clusters that authenticate against the primary:

   ```yaml
   # federated-cluster-config.yaml
   apiVersion: v1
   kind: ConfigMap
   metadata:
     name: tka-config
     namespace: tka-system
   data:
     config.yaml: |
       tailscale:
         hostname: tka-west  # Unique per cluster
         port: 443
         tailnet: your-tailnet.ts.net
         capName: specht-labs.de/cap/tka

       operator:
         namespace: tka-system
         clusterName: prod-west-cluster
         contextPrefix: west-
         userPrefix: west-user-

       # Point to primary login server
       federation:
         loginServer: https://tka.your-tailnet.ts.net
   ```

7. ## Maintenance and Updates

   ### Rolling Updates

   ```bash
   # Update TKA server image
   kubectl set image deployment/tka-server tka-server=ghcr.io/spechtlabs/tka:v1.2.3 -n tka-system

   # Monitor rollout
   kubectl rollout status deployment/tka-server -n tka-system
   ```

   ### Configuration Updates

   ```bash
   # Update configuration
   kubectl apply -f tka-config.yaml

   # Restart pods to pick up new config
   kubectl rollout restart deployment/tka-server -n tka-system
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
# Check server logs
kubectl logs -l app=tka-server -n tka-system -f

# Check controller logs
kubectl logs -l control-plane=controller-manager -n tka-system -f

# Verify connectivity
kubectl exec -it deployment/tka-server -n tka-system -- curl -k https://tka.your-tailnet.ts.net/metrics
```

## Next Steps

- [Multi-cluster Setup](./multi-cluster-setup.md)
- [Troubleshooting Guide](./troubleshooting.md)
- [Security Best Practices](../explanation/security.md)
