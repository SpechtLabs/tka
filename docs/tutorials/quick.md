---
title: Getting Started with TKA (Quick Start)
permalink: /tutorials/quick
createTime: 2025/01/27 10:00:00
---

> [!TIP]
> There is also a [Getting Started with TKA (comprehensive guide)](./comprehensive.md) tutorial available

<!-- @include: prerequisites.md -->

## Quick Path

For the fastest setup, follow these condensed steps:

:::: steps

1. ### Get TKA Binaries

   ```bash
   # Download latest release
   curl -fsSL https://github.com/spechtlabs/tka/releases/latest/download/tka-linux-amd64.tar.gz | tar -xz

   # Make executable and add to PATH
   chmod +x tka tka-server
   sudo mv tka tka-server /usr/local/bin/
   ```

2. ### Install Kubernetes Resources

   ```bash
   # Install CRDs and RBAC
   kubectl apply -f https://github.com/spechtlabs/tka/releases/latest/download/tka-k8s.yaml
   ```

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

4. ### Start TKA Server

   ```bash
   # Set configuration
   export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net
   export TS_AUTHKEY=tskey-auth-your-key-here

   # Start server
   tka-server serve --server tka --port 443
   ```

5. ### Test Authentication

   In a new terminal:

   ```bash
   # Configure CLI
   export TKA_TAILSCALE_TAILNET=your-tailnet.ts.net

   # Authenticate and test
   tka login
   kubectl get pods -A
   tka logout
   ```

::::

**Success!** If you can run `kubectl get pods -A` after `tka login`, you're done!

<!-- @include: troubleshooting_and_next_steps.md -->
