---
title: Multi-Cluster Setup
permalink: /getting-started/clustering
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> This guide assumes you already have TKA running on at least one cluster. See the [Quick Start Guide](./quick.md) first.

Connect multiple TKA servers so users can discover and access all clusters from a single entry point.

## Overview

TKA servers share cluster information via a gossip protocol. Once connected:

- **Every TKA server** knows about all other TKA instances
- **Users configure one default server** and discover all clusters from it
- **Login to any cluster** using `--server` without reconfiguring

```mermaid
flowchart LR
    subgraph Tailnet
        tka1[TKA Server A<br/>default]
        tka2[TKA Server B]
        tka3[TKA Server C]
    end

    tka1 <-->|gossip| tka2
    tka2 <-->|gossip| tka3
    tka1 <-->|gossip| tka3

    cli[CLI] -->|1. memberlist| tka1
    tka1 -->|2. [A, B, C]| cli
    cli -->|3. login| tka2
```

> [!NOTE]
> For details on how the gossip protocol works, see [Understanding: Cluster Discovery & Gossip](../understanding/clustering.md).

## Prerequisites

- Two or more Kubernetes clusters with TKA installed
- Tailscale connectivity between all TKA servers (same tailnet)
- Unique hostnames for each TKA server

:::: steps

1. ## Enable Gossip on Each Server

   Update each TKA server's Helm values:

   ```yaml
   # values-cluster-a.yaml
   tka:
     tailscale:
       hostname: tka-prod-us  # Must be unique per cluster
       tailnet: your-tailnet.ts.net

     gossip:
       enabled: true
       port: 7946

     clusterInfo:
       apiEndpoint: "https://api.prod-us.example.com:6443"
       labels:
         environment: production
         region: us-west-2
   ```

   > [!IMPORTANT]
   > Each TKA server must have a **unique `tailscale.hostname`**.

2. ## Configure Bootstrap Peers

   The first server needs no bootstrap peers. Subsequent servers need at least one:

   ```yaml
   # values-cluster-b.yaml
   tka:
     tailscale:
       hostname: tka-prod-eu
       tailnet: your-tailnet.ts.net

     gossip:
       enabled: true
       port: 7946
       bootstrapPeers:
         - tka-prod-us.your-tailnet.ts.net:7946

     clusterInfo:
       apiEndpoint: "https://api.prod-eu.example.com:6443"
       labels:
         environment: production
         region: eu-west-1
   ```

   > [!TIP]
   > Once connected, peers discover each other automatically. You only need one bootstrap peer.

3. ## Deploy

   ::: terminal Deploy clusters

   ```bash
   # First cluster
   helm upgrade --install tka spechtlabs/tka -n tka-system -f values-cluster-a.yaml

   # Additional clusters
   helm upgrade --install tka spechtlabs/tka -n tka-system -f values-cluster-b.yaml
   ```

   :::

4. ## Verify

   ::: terminal Verify cluster discovery

   ```bash
   # Query memberlist from any TKA server
   curl -s https://tka-prod-us.your-tailnet.ts.net/api/v1alpha1/memberlist | jq

   # Check logs for gossip activity
   kubectl logs -n tka-system -l app.kubernetes.io/name=tka | grep -i gossip
   ```

   :::

5. ## Configure CLI

   Set your default TKA server:

   ```yaml
   # ~/.config/tka/config.yaml
   tailscale:
     hostname: tka-prod-us  # Your default entry point
     tailnet: your-tailnet.ts.net
   ```

6. ## Use Multiple Clusters

   ::: terminal Discover and login

   ```bash
   # Login to default cluster
   tka login

   # Login to a different cluster
   tka login --server tka-prod-eu

   # Login to staging
   tka login --server tka-staging
   ```

   :::

::::

## Quick Reference

| Setting | Default | Description |
|---------|---------|-------------|
| `gossip.enabled` | `false` | Enable clustering |
| `gossip.port` | `7946` | Gossip TCP port |
| `gossip.bootstrapPeers` | `[]` | Initial peers |

For all settings, see [Configuration Reference](../reference/configuration.md#cluster-gossip-protocol).

## Troubleshooting

**Peers not discovering?**

1. Check Tailscale connectivity between servers
2. Verify `gossip.enabled: true` in all servers
3. Check bootstrap peer format: `hostname.tailnet:port`

**See also**: [Understanding: Cluster Discovery](../understanding/clustering.md) for protocol details.
