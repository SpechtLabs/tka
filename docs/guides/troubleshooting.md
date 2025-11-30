---
title: Troubleshooting Guide
permalink: /guides/troubleshooting
createTime: 2025/01/27 10:00:00
---

Common issues and solutions when using TKA.

## Authentication Issues

### 403 Forbidden from Funnel

**Problem**: Getting 403 errors when trying to access TKA.

**Cause**: Requests are coming through Tailscale Funnel instead of direct tailnet access.

**Solution**:

- Ensure you're connected to the tailnet
- Disable Funnel for the TKA server if enabled
- Check that your client device is authenticated to Tailscale

::: terminal Verify connectivity

```bash
# Verify Tailscale status
$ tailscale status

# Check if you can reach the TKA metrics server
$ kubectl port-forward -n tka-system svc/tka 8080:8080
$ curl -s http://localhost:8080/metrics

# Check if you can reach the TKA server
$ curl -s {tka}.{your-tailnet}.ts.net:8123/cluster-info
```

:::

### 400 Bad Request on Capability

**Problem**: Authentication fails with 400 error mentioning capabilities.

**Possible Causes**:

1. **Multiple rules with same priority**: "Multiple capability rules with the same priority found"
2. **Malformed capability JSON**: Invalid JSON syntax in ACL grants
3. **Multiple rules without priority differentiation**

**Solutions**:

1. **Priority Conflicts**: Ensure each rule has a unique priority value

   ```jsonc
   // BAD: Same priority values
   {
     "grants": [
       {"src": ["alice"], "app": {"tka": [{"role": "admin", "priority": 100}]}},
       {"src": ["group:admins"], "app": {"tka": [{"role": "edit", "priority": 100}]}}
     ]
   }

   // GOOD: Different priority values
   {
     "grants": [
       {"src": ["alice"], "app": {"tka": [{"role": "admin", "priority": 200}]}},
       {"src": ["group:admins"], "app": {"tka": [{"role": "edit", "priority": 100}]}}
     ]
   }
   ```

2. **Validate ACL JSON syntax** in Tailscale admin console
3. **Check capability name** matches server configuration (`--cap-name`)

### 401 Unauthorized on Kubeconfig

**Problem**: Getting 401 when fetching kubeconfig.

**Cause**: Not signed in or session expired.

**Solution**:

::: terminal Sign in and fetch kubeconfig

```bash
# Sign in first
$ tka login

# Then fetch kubeconfig
$ tka kubeconfig
```

:::

## Provisioning Issues

### 202 Accepted - Kubeconfig Not Ready

**Problem**: `tka kubeconfig` returns 202 and says provisioning in progress.

**Cause**: Kubernetes operator is still creating ServiceAccount and RBAC.

**Solution**:

- Wait and retry (CLI does this automatically)
- Check controller logs for errors
- Verify RBAC permissions for TKA controller

::: terminal Check provisioning status

```bash
# Check controller logs
$ kubectl logs -l control-plane=controller-manager -n tka-system

# Check TkaSignin resources
$ kubectl get tkasignins -n tka-system

# Check ServiceAccounts
$ kubectl get serviceaccounts -n tka-system
```

:::

### Long Provisioning Times

**Problem**: Taking too long to provision credentials.

**Cause**: Controller performance or resource constraints.

**Solution**:

::: terminal Check controller performance

```bash
# Check controller resource usage
$ kubectl top pods -n tka-system

# Check for pending resources
$ kubectl get events -n tka-system --sort-by='.lastTimestamp'

# Verify controller is running
$ kubectl get pods -l control-plane=controller-manager -n tka-system
```

:::

## Configuration Issues

### Environment Variables Not Applied

**Problem**: Setting `TKA_TAILSCALE_HOSTNAME` doesn't work.

**Cause**: Nested configuration keys need specific format or config file.

**Solution**:

::: terminal Fix environment variables

```bash
# Use underscores for nested keys
$ export TKA_TAILSCALE_HOSTNAME=tka
$ export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net

# Or use config file
$ cat > ~/.config/tka/config.yaml << EOF
tailscale:
  hostname: tka
  tailnet: your-tailnet.ts.net
EOF

# Or use command flags
$ tka --server tka --port 443 login
```

:::

### Server Connection Issues

**Problem**: Cannot connect to TKA server.

**Diagnostics**:

::: terminal Diagnose server connection

```bash
# Check server address construction
$ tka --debug login

# Test connectivity
$ curl -s http://localhost:8080/metrics

# Check DNS resolution
$ nslookup tka.your-tailnet.ts.net

# Verify Tailscale connectivity
$ tailscale ping tka.your-tailnet.ts.net
```

:::

## Server Issues

### Server Won't Start

**Problem**: TKA server fails to start.

**Common Causes and Solutions**:

1. **Invalid Auth Key**:

   ::: terminal Fix invalid auth key

   ```bash
   # For local debugging only (production should use Helm)
   $ export TS_AUTHKEY=tskey-auth-your-new-key
   $ tka-server serve
   ```

   :::

2. **Port Already in Use**:

   ::: terminal Fix port conflict

   ```bash
   # Check what's using the port
   $ sudo lsof -i :443

   # For local debugging - use different port
   $ tka-server serve --port 8443
   ```

   :::

3. **Permission Issues**:

   ::: terminal Fix permission issues

   ```bash
   # Ensure binary is executable
   $ chmod +x tka-server

   # For local debugging - check if port requires privileges
   # Port 443 needs root or CAP_NET_BIND_SERVICE
   $ sudo tka-server serve --port 443
   ```

   :::

### Tailscale Connection Issues

**Problem**: Server can't connect to Tailscale.

**Solution**:

::: terminal Fix Tailscale connection

```bash
# Check auth key permissions
# Key should allow device registration

# Verify network connectivity
$ ping login.tailscale.com

# Check for corporate firewall issues
# Tailscale needs outbound HTTPS (443) and UDP (41641)

# For local debugging - use different state directory
$ tka-server serve --dir /tmp/tka-state
```

:::

## Client Issues

### Shell Integration Not Working

**Problem**: `tka login` doesn't update KUBECONFIG automatically.

**Solution**:

::: terminal Fix shell integration

```bash
# Verify integration is installed
$ type tka  # Should show function, not binary

# Reinstall integration
$ eval "$(tka generate integration bash)"  # or your shell

# Manual verification
$ tka login --no-eval
# Then manually export the shown KUBECONFIG
```

:::

### kubectl Commands Fail After Login

**Problem**: kubectl still shows unauthorized after `tka login`.

**Diagnostics**:

::: terminal Diagnose kubectl issues

```bash
# Check KUBECONFIG is set
$ echo $KUBECONFIG

# Verify kubeconfig content
$ kubectl config view

# Test with explicit kubeconfig
$ kubectl --kubeconfig=/path/to/tka-kubeconfig get pods

# Check token validity
$ kubectl auth whoami
```

:::

## Kubernetes Integration Issues

### RBAC Permission Denied

**Problem**: kubectl commands fail with permission errors.

**Cause**: Role specified in ACL doesn't exist or lacks permissions.

**Solution**:

::: terminal Fix RBAC permissions

```bash
# Check what role you're assigned
$ tka get login

# Verify role exists
$ kubectl get clusterrole cluster-admin  # or your role

# Check role permissions
$ kubectl describe clusterrole cluster-admin

# Create custom role if needed
$ kubectl create clusterrole tka-developer --verb=get,list --resource=pods,services
```

:::

### ServiceAccount Issues

**Problem**: ServiceAccount creation fails.

**Solution**:

::: terminal Fix ServiceAccount issues

```bash
# Check TKA controller permissions
$ kubectl auth can-i create serviceaccounts --as=system:serviceaccount:tka-system:tka-controller

# Check namespace exists
$ kubectl get namespace tka-system

# Check for resource quotas
$ kubectl describe namespace tka-system
```

:::

## Network and Connectivity

### DNS Resolution Issues

**Problem**: Cannot resolve `tka.your-tailnet.ts.net`.

**Solution**:

::: terminal Fix DNS resolution

```bash
# Check MagicDNS is enabled
$ tailscale status

# Try IP address directly
$ tailscale ip tka

# Use full hostname
$ ping tka.your-tailnet.ts.net
```

:::

### Certificate Issues

**Problem**: SSL/TLS certificate errors.

**Cause**: HTTPS not enabled in Tailscale or wrong port.

**Solution**:

::: terminal Fix certificate issues

```bash
# Enable HTTPS in Tailscale admin console
# Go to DNS settings and enable HTTPS certificates

# For local debugging - use HTTP for non-443 ports
$ tka-server serve --port 8080  # Uses HTTP automatically

# Or force HTTPS scheme
$ export TKA_TAILSCALE_HOSTNAME=https://tka
```

:::

## Performance Issues

### Slow Response Times

**Problem**: TKA operations are slow.

**Diagnostics**:

::: terminal Diagnose performance issues

```bash
# Check server logs for bottlenecks
$ kubectl logs -l app=tka-server -n tka-system

# Monitor resource usage
$ kubectl top pods -n tka-system

# Check network latency
$ tailscale ping tka.your-tailnet.ts.net
```

:::

### High Memory Usage

**Problem**: TKA server consuming too much memory.

**Solution**:

::: terminal Fix high memory usage

```bash
# Update resource limits using Helm values
$ cat > resources-values.yaml << EOF
resources:
  limits:
    memory: "512Mi"
  requests:
    memory: "256Mi"
EOF

# Upgrade Helm release with new resource limits
$ helm upgrade tka spechtlabs/tka -n tka-system -f resources-values.yaml

# Or use --set for quick changes
$ helm upgrade tka spechtlabs/tka -n tka-system --set resources.limits.memory=512Mi --set resources.requests.memory=256Mi
```

:::

## Debug Mode

Enable debug logging for more detailed troubleshooting:

::: terminal Enable debug mode

```bash
# Client debug
$ tka --debug login

# Server debug
$ tka-server serve --debug

# Via environment
$ export TKA_DEBUG=true
```

:::

## Getting Help

If you're still experiencing issues:

1. **Check Server Logs**: `kubectl logs -l app=tka-server -n tka-system`
2. **Check Controller Logs**: `kubectl logs -l control-plane=controller-manager -n tka-system`
3. **Enable Debug Mode**: Add `--debug` flag to commands
4. **Review Configuration**: Verify ACLs, network policies, and RBAC
5. **Open an Issue**: [GitHub Issues](https://github.com/spechtlabs/tka/issues) with debug output

## Related Guides

- [Configuration Reference](../reference/configuration.md)
- [Security Model](../understanding/security.md)
- [Production Deployment](../getting-started/comprehensive.md#production-deployment)
