---
title: Getting Started with TKA
permalink: /getting-started/quick
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> For detailed explanations and production deployment, see the [Comprehensive Guide](./comprehensive.md)

Get TKA running in under 5 minutes:

:::: steps

1. ## Install CLI & Server

   ::: terminal One-command setup

   ```bash
   # Download CLI
   $ curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/ts-k8s-auth-$(uname)-$(uname -m) -o ts-k8s-auth
   $ chmod +x ts-k8s-auth && sudo mv ts-k8s-auth /usr/local/bin/

   # Install server
   $ helm repo add spechtlabs https://charts.specht-labs.de && helm repo update
   $ kubectl create namespace tka-system
   $ kubectl create secret generic tka-tailscale --from-literal=TS_AUTHKEY=tskey-auth-your-key-here -n tka-system
   $ helm install tka spechtlabs/tka -n tka-system \
     --set tka.tailscale.tailnet=your-tailnet.ts.net \
     --set tka.clusterInfo.apiEndpoint="$(kubectl config view --minify -o jsonpath='{.clusters[0].cluster.server}')" \
     --set tka.clusterInfo.insecureSkipTLSVerify=true
   ```

   :::

2. ### Configure Tailscale ACLs

   Add to your [Tailscale ACL policy](https://login.tailscale.com/admin/acls):

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

3. ### Configure & Test

   ::: terminal Configure and test

   ```bash
   # Configure CLI
   $ mkdir -p ~/.config/tka
   $ echo "tailscale:\n  hostname: tka\n  tailnet: your-tailnet.ts.net" > ~/.config/tka/config.yaml
   $ eval "$(ts-k8s-auth generate integration bash)"  # or zsh/fish

   # Test
   $ kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tka -n tka-system
   $ tka shell
   (tka) $ kubectl get pods -A
   ```

   :::

::::

**Done!** You now have TKA running. For production deployments, monitoring, and advanced configuration, see the [Comprehensive Guide](./comprehensive.md).
