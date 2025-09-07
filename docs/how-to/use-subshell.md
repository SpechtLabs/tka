---
title: Use a subshell with ephemeral access
permalink: /how-to/use-subshell
createTime: 2025/01/27 10:00:00
---

The `tka shell` command provides a clean way to get temporary Kubernetes access without affecting your main shell environment.

## How It Works

`tka shell` creates an isolated environment where:

- Your existing shell remains untouched
- `KUBECONFIG` is automatically set to a temporary file
- All kubectl operations use ephemeral credentials
- Access is automatically revoked when you exit

## Basic Usage

```bash
# Start a subshell with temporary access
tka shell

# You're now in the subshell with active credentials
kubectl get pods -n default
kubectl get nodes

# Exit the subshell
exit

# Back in your original shell - no access
kubectl get pods  # This will fail
```

## What Happens Behind the Scenes

1. **Authentication**: TKA authenticates you and provisions credentials
2. **Subshell Creation**: A new shell process starts (using your `$SHELL`)
3. **Environment Setup**: `KUBECONFIG` points to a temporary file
4. **Cleanup**: On exit, credentials are revoked and temp files removed

## Advantages

### Isolation

- Your main shell environment is never modified
- No risk of polluting existing `KUBECONFIG` settings
- Multiple subshells can run simultaneously for different clusters

### Automatic Cleanup

- Credentials are revoked immediately on exit
- Temporary kubeconfig files are removed
- No lingering access or credential files

### Clear Access Boundaries

- Obvious when you have access (inside subshell)
- Obvious when you don't (outside subshell)
- No confusion about which credentials are active

## Platform Support

### Supported Platforms

- **Linux**: Full support
- **macOS**: Full support
- **WSL**: Full support

### Windows Limitations

- **PowerShell/Command Prompt**: Not supported
- **Workaround**: Use `tka login` instead
- **WSL**: Full support within WSL environment

## Advanced Usage

### Custom Shell

The subshell uses your default shell (`$SHELL`):

```bash
# Check your default shell
echo $SHELL

# Set a different shell for TKA
export SHELL=/bin/zsh
tka shell  # Uses zsh

# Or temporarily
SHELL=/bin/fish tka shell  # Uses fish
```

### Multiple Clusters

Access different clusters in separate subshells:

```bash
# Terminal 1: Production cluster
export TKA_TAILSCALE_HOSTNAME=tka-prod
tka shell

# Terminal 2: Development cluster
export TKA_TAILSCALE_HOSTNAME=tka-dev
tka shell
```

### Scripting with Subshells

Combine with scripts for specific tasks:

```bash
# Create a deployment script
cat > deploy.sh << 'EOF'
#!/bin/bash
echo "Deploying to cluster..."
kubectl apply -f deployment.yaml
kubectl rollout status deployment/myapp
echo "Deployment complete!"
EOF

# Run script in TKA subshell
tka shell -c "./deploy.sh"
```

## Comparison with `tka login`

| Aspect | `tka shell` | `tka login` |
|--------|-------------|-------------|
| Environment | Isolated subshell | Modifies current shell |
| Cleanup | Automatic | Manual logout required |
| Persistence | Session only | Until logout/expiry |
| Multiple access | Easy (separate shells) | Requires env management |
| Scripting | Natural isolation | Requires careful handling |
| Platform support | Unix-like systems | All platforms |

## Troubleshooting

### Subshell Won't Start

**Problem**: `tka shell` fails to create subshell

**Solutions**:

```bash
# Check if your shell is accessible
which $SHELL

# Try with explicit shell
SHELL=/bin/bash tka shell

# Check shell permissions
ls -la $SHELL
```

### Environment Not Set

**Problem**: kubectl still shows "unauthorized" in subshell

**Solutions**:

```bash
# Verify KUBECONFIG is set
echo $KUBECONFIG

# Check kubeconfig content
kubectl config view

# Try manual login first
tka login --no-eval
```

### Access Denied

**Problem**: TKA authentication fails

**Solutions**:

- Verify you're on the Tailscale network: `tailscale status`
- Check ACL capability grants in Tailscale admin console
- Test basic connectivity: `curl -k https://tka.your-tailnet.ts.net/metrics`

### Shell Behavior Issues

**Problem**: Subshell behaves differently than expected

**Solutions**:

```bash
# Check shell initialization files
ls -la ~/.*rc ~/.*profile

# Use a clean shell
env -i SHELL=/bin/bash tka shell

# Debug shell startup
bash -x -c 'tka shell'
```

## Best Practices

### Development Workflow

```bash
# Start with subshell for exploration
tka shell
kubectl get namespaces
kubectl describe deployment myapp
exit

# Use login for longer development sessions
tka login
# ... extended development work ...
tka logout
```

### Production Access

```bash
# Use subshell for production to minimize risk
tka shell
kubectl get pods -n production
kubectl logs deployment/critical-app
exit  # Immediate access revocation
```

### Debugging

```bash
# Isolate debugging session
tka shell
kubectl debug pod/failing-pod --image=busybox
# ... debug session ...
exit  # Clean up debug resources and access
```

## Security Considerations

- **Access Duration**: Limited to your ACL-defined period
- **Process Isolation**: Subshell runs as your user with same permissions
- **Network Access**: Still requires Tailscale network connectivity
- **Audit Trail**: All kubectl operations are logged normally

## Related Commands

- [`tka login`](../reference/cli.md#login) - Persistent access in current shell
- [`tka logout`](../reference/cli.md#logout) - Revoke access
- [`tka get login`](../reference/cli.md#get-login) - Check current status

## Next Steps

- [Configure Shell Integration](./shell-integration.md) for seamless environment updates
- [Production Deployment](./deploy-production.md) for production-ready TKA setup
- [Troubleshooting Guide](./troubleshooting.md) for common issues
