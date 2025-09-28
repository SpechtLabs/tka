---
title: Troubleshooting and Next Steps
createTime: 2025/09/07 23:05:53
permalink: /getting-started/troubleshooting
---

If something isn't working, check our [troubleshooting guide](../how-to/troubleshooting.md) for common issues:

- **403 errors**: Ensure you're on the tailnet, not using Funnel
- **400 capability errors**: Check for multiple or malformed ACL rules
- **Connection issues**: Verify Tailscale connectivity and DNS
- **RBAC errors**: Ensure the role in your ACL exists in Kubernetes

## Next Steps

Now that TKA is working:

1. **Production Setup**: Follow the [production deployment section](./comprehensive.md#production-deployment) in the comprehensive guide
2. **Shell Integration**: Set up [automatic environment updates](../how-to/shell-integration.md)
3. **Multi-cluster**: Configure [multiple clusters](../how-to/multi-cluster-setup.md)
4. **Advanced ACLs**: Learn more about [ACL configuration](../how-to/configure-acl.md)

## Understanding What Happened

TKA just demonstrated a complete zero-trust authentication flow:

1. **Network Authentication**: Your request came via the Tailscale network
2. **Identity Resolution**: TKA identified you via Tailscale WhoIs
3. **Capability Check**: Your ACL grants were validated
4. **Resource Provisioning**: A ServiceAccount and RBAC were created
5. **Token Generation**: A short-lived token was issued
6. **Access Granted**: You could use kubectl with proper permissions
7. **Cleanup**: Resources were removed when you logged out

This provides ephemeral, auditable access without permanent credentials or complex OIDC integrations.
