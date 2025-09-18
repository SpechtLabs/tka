---
title: Request Flows
permalink: /reference/developer/request-flows
createTime: 2025/08/25 06:33:41
---

This page is here to provide more in-depth information of the request flows

```mermaid
sequenceDiagram
    autonumber
    participant cli as User (tka cli)
    participant api as API Server (tailnet only)
    participant acl as Tailscale
    participant k8s as Kubernetes
    participant operator as TKA-Operator

    cli->>api: POST /api/v1alpha1/login
    activate api

    api->>acl: Check ACL policy
    acl-->>api: return role, period

    alt is Not Allowed
        api-->>cli: 403 Forbidden
    else Capability error
        api-->>cli: 400 Bad Request
    else is allowed
        api->>k8s: write TkaSignIn CRD
        api-->> cli: 202 Accepted
    end

    deactivate api

    loop Reconcile
        operator->>k8s: observe new signins
        k8s-->>operator: process new signin
        activate operator
        operator->>operator: Provisions user
        operator->>k8s: Update TkaSignin Status CRD
        deactivate operator
    end


    cli->>api: GET /api/v1alpha1/kubeconfig
    activate api
    api->>operator: Kubeconfig()
    activate operator

    operator->>k8s: Get TkaSignIn
    alt No Signin Found
        operator->>api: Error: SignIn not found
        api-->>cli: 404 Not Found - User not authenticated
    else SignIn not ready
        operator-->>api: Error: Not ready yet
        api-->>cli: 202 + RetryAfter header - Credentials not ready yet
    else Is provisioned
        operator->>k8s: Generate token
        operator-->>api: Kubeconfig
        api-->>cli: 200 - Kubeconfig
    end

    deactivate operator
    deactivate api
```

## Details

### Tailscale Auth Middleware

```mermaid
sequenceDiagram
    participant cli as tka cli (cmd/cli)
    participant api as API (pkg/api)
    participant auth as gin tailscale auth middleware<br/> (pkg/middleware/auth)
    participant ts as Tailscale API

    cli->>api: POST /api/v1alpha1/login
    api->>auth: every request passes through auth middleware
    activate auth

    alt request is funnel
        auth-->>cli: HTTP 403 - Unauthorized request from Funnel
    end

    auth->>auth: get WhoIs resolver
    Note right of auth: WhoIs resolver is defined as tailscale.WhoIsResolver in <br/> pkg/tailscale

    alt request is from a tagged node
        auth-->>cli: HTTP 403 - Requests can only originate from users
    end

    auth->>auth: Extract username

    auth->>ts: Parse capability (Grant) from Tailscale ACL

    alt no capability found
        ts-->>auth: 0 cap rules
        auth-->>cli: HTTP 403 - User not authorized
    else more than one capability found
        ts-->>auth: >1 cap rules
        auth-->>cli: HTTP 400 - Found more than one capability
        Note left of auth: What to do with more than one cap?<br/>We can't apply least privilege, as we don't know which privilege are granted by a K8s RBAC role.<br/>We only know the name of it
    else failed to parse capRule
        auth-->>cli: HTTP 400 - capRule invalid
    end

    auth->>auth: SetUsername
    auth->>auth: SetRule
    auth->>api: Process next request handeler
    activate api
    deactivate auth
    api->>api: Process request as usual
    api-->>cli: Return response
    deactivate api

```

### SignIn Request

```mermaid
sequenceDiagram
    participant cli as User (tka cli)
    participant api as API Server (tailnet only)
    participant k8s as Kubernetes
    participant operator as TKA-Operator

    cli->>api: POST /api/v1alpha1/login
    activate api

    alt is Not Allowed
        api-->>cli: 403 Forbidden
    else Bad Request
        api-->>cli: 400 Forbidden
    end

    loop Reconcile
        operator->>k8s: observe new signins
        activate k8s
    end

    api-->> cli: 202 Accepted
    api->>k8s: write TkaSignIn CRD
    deactivate api

    k8s-->>operator: process new signin
    deactivate k8s
    activate operator

        operator->>operator: Provisions user
        operator->>k8s: Update TkaSignin Status CRD
        deactivate operator
```

### Provision SignIn

```mermaid
sequenceDiagram
    participant api as tka API
    participant k8s as Kubernetes
    participant operator as TKA-Operator

    loop Reconcile
        operator->>k8s: observe new signins
        k8s-->>operator: process new signin
        activate operator

        operator->>k8s: create ServiceAccount
        operator->>k8s: create ClusterRoleBinding
        operator->>k8s: update LastAttemptedSignIn annotation
        operator->>operator: set validUntil to current time + valid period from ACL
        operator->>k8s: update TkaSignIn CRD status
        deactivate operator
    end

    opt getKubeconfig
        api->>operator: Kubeconfig()
        activate operator

        operator->>k8s: check TkaSignIn CRD
        k8s-->>operator: response: user is provisioned

        Note over k8s,operator: token expiry == TkaSignIn CRD status "validUntil" field

        operator->>k8s: generate token for ServiceAccount
        k8s-->>operator: return token (not persisted anywhere)

        Note over operator: Assemble Kubeconfig

        operator-->>operator: Kubeconfig used by the operator
        operator->>operator: set ServiceAccount as user in kubeconfig
        operator->>operator: set token for ServiceAccount in kubeconfig

        operator-->>api: return kubeconfig
        deactivate operator
    end
```

## Request-Flow Get Kubeconfig

```mermaid
sequenceDiagram
    participant api as tka API
    participant k8s as Kubernetes
    participant operator as TKA-Operator

    api->>operator: Kubeconfig()
    activate operator

    operator->>k8s: check TkaSignIn CRD
    k8s-->>operator: response: user is provisioned

    Note over k8s,operator: token expiry == TkaSignIn CRD status "validUntil" field

    operator->>k8s: generate token for ServiceAccount
    k8s-->>operator: return token (not persisted anywhere)

    Note over operator: Assemble Kubeconfig

    operator-->>operator: Kubeconfig used by the operator
    operator->>operator: set ServiceAccount as user in kubeconfig
    operator->>operator: set token for ServiceAccount in kubeconfig

    operator-->>api: return kubeconfig
    deactivate operator
```
