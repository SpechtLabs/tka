---
title: API Reference
permalink: /reference/api
createTime: 2025/01/27 10:00:00
---

TKA exposes a REST API for authentication and kubeconfig management. All endpoints require Tailscale network access and valid capability grants.

## Authentication

Authentication is automatic via the Tailscale network:

1. **Network Authentication**: Requests must originate from within the tailnet
2. **Identity Resolution**: Server performs WhoIs lookup on client IP
3. **Capability Check**: Validates ACL grants for the requesting user

### Headers

No explicit authentication headers are required. The server automatically:

- Rejects Funnel requests (off-tailnet access)
- Extracts user identity via Tailscale WhoIs
- Validates capability grants from ACL policy

## OpenAPI Specification

The complete OpenAPI specification is available on the TKA Server on the following paths:

- **JSON**: `/swagger/swagger.json`
- **YAML**: `/swagger/swagger.yaml`
- **Interactive UI**: `/swagger/index.html`

### Swagger UI

<!-- markdownlint-disable MD033 -->

<ClientOnly>
    <VPSwaggerUI />
</ClientOnly>

<!-- markdownlint-enable MD033 -->

## Client Libraries

### Official CLI

The `tka` CLI is the primary client for the API:

```bash
# Login (POST /login + GET /kubeconfig)
tka login

# Check status (GET /login)
tka get login

# Fetch kubeconfig (GET /kubeconfig)
tka kubeconfig

# Logout (POST /logout)
tka logout
```

### Custom Clients

When building custom clients:

1. **Network Requirements**: Ensure client runs within tailnet
2. **User Agent**: Include meaningful user agent for debugging
3. **Retry Logic**: Handle 202 responses with exponential backoff
4. **Error Handling**: Parse error responses for actionable messages

**Example cURL Usage:**

```bash
# Authenticate
curl -X POST https://tka.your-tailnet.ts.net/api/v1alpha1/login

# Check status
curl https://tka.your-tailnet.ts.net/api/v1alpha1/login

# Get kubeconfig
curl https://tka.your-tailnet.ts.net/api/v1alpha1/kubeconfig

# Logout
curl -X POST https://tka.your-tailnet.ts.net/api/v1alpha1/logout
```

## Related Documentation

- [Configuration Reference](./configuration.md)
- [CLI Reference](./cli.md)
- [Security Model](../understanding/security.md)
- [Request Flows](./developer/request-flows.md)
