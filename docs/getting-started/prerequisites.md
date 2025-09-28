---
title: Prerequisites
createTime: 2025/09/07 23:16:49
permalink: /getting-started/prerequisites
---

Learn to set up TKA in a Kubernetes cluster and authenticate using your Tailscale identity.

## What You'll Build

By the end of this tutorial:

- TKA server running in your Kubernetes cluster
- Ephemeral kubeconfig issued for your user
- RBAC bound to the role defined in your ACL capability
- Automatic credential cleanup on logout

## Prerequisites

**Essential:**

- Kubernetes cluster you can reach (kind, k3s, existing cluster, etc.)
- [Tailscale account](https://tailscale.com/kb/1017/install) with your device joined to the tailnet
- [kubectl](https://kubernetes.io/docs/tasks/tools/) installed

**For HTTPS (recommended):**

- [HTTPS enabled in your tailnet](https://tailscale.com/kb/1153/enabling-https)
- [Tailscale authentication key](https://tailscale.com/kb/1245/set-up-servers)

**Optional:**

- [Docker](https://docs.docker.com/get-docker/) or [Podman](https://podman.io/getting-started/installation) (for creating test cluster)
- [Go 1.21+](https://golang.org/dl/) (for building from source)
