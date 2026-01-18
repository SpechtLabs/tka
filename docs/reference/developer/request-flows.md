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
    Note right of auth: WhoIs resolver is defined as tsnet.WhoIsResolver in <br/> pkg/tsnet

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
    auth->>api: Process next request handler
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

## Cluster Discovery (Gossip Protocol)

TKA servers can discover each other and share cluster metadata using a gossip-based protocol. This enables multi-cluster deployments where users can discover available clusters.

### Gossip Overview

The gossip protocol uses a 3-way handshake to synchronize state between nodes:

1. **Heartbeat**: Node A sends its version digest to Node B
2. **Diff**: Node B responds with state differences and its own digest
3. **Delta**: Node A sends any remaining differences to complete sync

```mermaid
sequenceDiagram
    autonumber
    participant nodeA as TKA Node A
    participant nodeB as TKA Node B

    Note over nodeA,nodeB: Gossip Round (repeats periodically)

    nodeA->>nodeB: SendHeartbeat(GossipHeartbeatRequest)
    activate nodeB
    Note right of nodeA: Contains: srcId, timestamp,<br/>version_map_digest

    nodeB->>nodeB: Update peer last_seen
    nodeB->>nodeB: Compare digests
    nodeB->>nodeB: Generate state diff

    nodeB-->>nodeA: GossipDiffResponse
    deactivate nodeB
    Note left of nodeB: Contains: srcId, state_delta,<br/>version_map_digest

    activate nodeA
    nodeA->>nodeA: Apply received state
    nodeA->>nodeA: Generate own diff

    alt Has updates for Node B
        nodeA->>nodeB: SendDiff(GossipDiffRequest)
        activate nodeB
        Note right of nodeA: Contains: srcId, state_delta,<br/>version_map_digest

        nodeB->>nodeB: Apply received state
        nodeB->>nodeB: Generate delta if needed

        nodeB-->>nodeA: GossipDeltaResponse
        deactivate nodeB
        Note left of nodeB: Contains: srcId, state_delta (if any),<br/>has_delta flag

        nodeA->>nodeA: Apply delta if present
    end
    deactivate nodeA
```

### Data Exchanged

Each TKA node shares its **NodeMetadata** through the gossip protocol:

| Field | Type | Description |
| :--- | :--- | :--- |
| `apiEndpoint` | string | The Kubernetes API server URL for this cluster |
| `apiPort` | int | The port TKA server listens on |
| `labels` | map[string]string | Cluster labels (environment, region, etc.) |

### Gossip Message Types

```mermaid
classDiagram
    class GossipHeartbeatMessage {
        +int64 ts_unix_nano
        +map~string,DigestEntry~ version_map_digest
    }

    class GossipDiffMessage {
        +map~string,GossipVersionedState~ state_delta
        +map~string,DigestEntry~ version_map_digest
    }

    class GossipDeltaMessage {
        +map~string,GossipVersionedState~ state_delta
    }

    class DigestEntry {
        +uint64 version
        +string address
        +int64 last_seen_unix_nano
        +PeerState peer_state
    }

    class GossipVersionedState {
        +DigestEntry digest_entry
        +bytes data
    }

    class NodeMetadata {
        +string apiEndpoint
        +int apiPort
        +map~string,string~ labels
    }

    GossipHeartbeatMessage --> DigestEntry : contains
    GossipDiffMessage --> DigestEntry : contains
    GossipDiffMessage --> GossipVersionedState : contains
    GossipDeltaMessage --> GossipVersionedState : contains
    GossipVersionedState --> DigestEntry : contains
    GossipVersionedState --> NodeMetadata : serialized as bytes
```

### Peer States

The gossip protocol tracks the health of peers using the following states:

| State | Description |
| :--- | :--- |
| `HEALTHY` | Peer is responding normally |
| `SUSPECTED_DEAD` | Peer has missed several heartbeat cycles |
| `DEAD` | Peer has exceeded the dead threshold and will be removed |

### Gossip Configuration

| Parameter | Default | Description |
| :--- | :--- | :--- |
| `gossipFactor` | 3 | Number of peers to gossip with per cycle |
| `gossipInterval` | 1s | Time between gossip cycles |
| `stalenessThreshold` | 5 | Consecutive failures before marking peer as suspected dead |
| `deadThreshold` | 10 | Consecutive failures before removing peer |

### Memberlist API

Users can query the current cluster membership via the API:

```mermaid
sequenceDiagram
    participant cli as User / Browser
    participant api as TKA API Server
    participant store as GossipStore

    cli->>api: GET /api/v1alpha1/memberlist
    activate api

    api->>store: GetDisplayData()
    activate store
    store-->>api: []NodeDisplayData
    deactivate store

    alt Accept: application/json
        api-->>cli: JSON response
    else Accept: text/html
        api-->>cli: HTML memberlist page
    end
    deactivate api
```

**Response includes for each node:**

| Field | Description |
| :--- | :--- |
| `id` | Unique node identifier |
| `address` | gRPC address for gossip communication |
| `lastSeen` | Timestamp of last successful communication |
| `version` | Current state version (vector clock) |
| `state` | Serialized NodeMetadata |
| `peerState` | Health state (HEALTHY, SUSPECTED_DEAD, DEAD) |
| `isLocal` | Whether this is the local node |
