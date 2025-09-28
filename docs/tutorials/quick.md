---
title: Getting Started with TKA (Quick Start)
permalink: /tutorials/quick
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> There is also a [advanced tutorial)](./comprehensive.md) available

<!-- @include: prerequisites.md -->

## Quick Start

For the fastest setup, follow these condensed steps:

:::: steps

1. ### Get TKA CLI

   <!-- markdownlint-disable MD033 -->
   <Terminal title="Download and install TKA CLI">

   ```bash
   # Download latest CLI release
   $ curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/ts-k8s-auth-linux-amd64 -o ts-k8s-auth

   # Make executable and add to PATH
   $ chmod +x ts-k8s-auth
   $ sudo mv ts-k8s-auth /usr/local/bin/
   ```

   </Terminal>
   <!-- markdownlint-enable MD033 -->

2. ### Install TKA Server with Helm

   <!-- markdownlint-disable MD033 -->
   <Terminal title="Install TKA Server with Helm">

   ```bash
   # Add Helm repository
   $ helm repo add spechtlabs https://charts.specht-labs.de
   $ helm repo update

   # Create namespace
   $ kubectl create namespace tka-system

   # Get cluster information needed for TKA
   $ CLUSTER_ENDPOINT=$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')
   $ CLUSTER_CA_DATA=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')

   # Create Tailscale secret
   $ kubectl create secret generic tka-tailscale \
     --from-literal=TS_AUTHKEY=tskey-auth-your-key-here \
     -n tka-system

   # Install TKA with cluster configuration
   $ helm install tka spechtlabs/tka -n tka-system \
     --set tka.tailscale.tailnet=your-tailnet.ts.net \
     --set tka.clusterInfo.apiEndpoint="$CLUSTER_ENDPOINT" \
     --set tka.clusterInfo.caData="$CLUSTER_CA_DATA" \
     --set tka.clusterInfo.labels.environment=quickstart
   ```

   </Terminal>
   <!-- markdownlint-enable MD033 -->

3. ### Configure Tailscale ACLs

   Add this to your Tailscale ACL policy in the [admin console](https://login.tailscale.com/admin/acls):

   ```jsonc
   {
     "tagOwners": {
       "tag:tka": ["autogroup:admin"]
     },
     "grants": [
       {
         "src": ["autogroup:admin"],
         "dst": ["tag:tka"],
         "ip": ["443"],
         "app": {
           "specht-labs.de/cap/tka": [
             {
               "role": "cluster-admin",
               "period": "4h",
               "priority": 100
             }
           ]
         }
       }
     ]
   }
   ```

4. ### Configure CLI

   <!-- markdownlint-disable MD033 -->
   <Terminal title="Configure TKA CLI">

   ```bash
   # Create config directory and file
   $ mkdir -p ~/.config/tka
   $ cat > ~/.config/tka/config.yaml << EOF
   tailscale:
     hostname: tka
     tailnet: your-tailnet.ts.net
   EOF

   # Install shell integration for the tka login wrapper
   $ eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish
   ```

   </Terminal>
   <!-- markdownlint-enable MD033 -->

5. ### Test Authentication

   <!-- markdownlint-disable MD033 -->
   <Terminal title="Test authentication">

   ```bash
   # Wait for TKA to be ready
   $ kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tka -n tka-system

   # Use shell integration for seamless experience
   $ tka shell
   (tka) $ kubectl get pods -A
   (tka) $ exit
   ```

   </Terminal>
   <!-- markdownlint-enable MD033 -->

::::

**Success!** If you can run `kubectl get pods -A` after `tka shell`, you're done!

<!-- @include: troubleshooting_and_next_steps.md -->
