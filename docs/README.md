---
pageLayout: home
externalLinkIcon: false

config:
  - type: doc-hero
    hero:
      name: Tailscale-native Kubernetes access
      text: Skip the proxies, VPNs, and OIDC headaches.
      tagline: Secure, short-lived cluster access — powered entirely by your Tailscale identity and network.
      image: /logo.png
      actions:
        - text: Overview →
          link: /overview/overview
          theme: brand
          icon: fa:book
        - text: Get Started →
          link: /guide/getting-started
          theme: alt
          icon: fa:paper-plane
        - text: GitHub Releases →
          link: https://github.com/spechtlabs/tka/releases
          theme: alt
          icon: fa:github

  - type: features
    title: Why Tailscale K8s Auth?
    description: Built for Kubernetes. Powered by your tailnet.
    features:
      - title: Zero-trust, zero-ingress
        icon: mdi:shield-lock-outline
        details: No public endpoints. Access is gated by your Tailscale ACLs, identity, and devices — and nothing else.

      - title: Ephemeral credentials, by default
        icon: mdi:timer-sand
        details: All kubeconfigs are short-lived and auto-expiring. You get access when you need it, and not a second longer.

      - title: Kubernetes-native RBAC
        icon: mdi:kubernetes
        details: Grants are mapped to native ClusterRoles, and credentials are provisioned as real Kubernetes ServiceAccounts.

      - title: Declarative grant-to-role mapping
        icon: mdi:account-key-outline
        details: Use a CRD to define how Tailscale users, groups, or tags map to Kubernetes roles — GitOps ready.

      - title: No complex proxies or auth chains
        icon: mdi:link-off
        details: Say goodbye to reverse proxies and auth headers. Access is handled with direct, secure API calls inside the tailnet.

      - title: Central login API, federation-ready
        icon: mdi:server-network
        details: Supports a central login cluster with other clusters registering dynamically — ideal for multi-cluster orgs.

  - type: custom

  - type: VPReleasesCustom
    repo: SpechtLabs/tka

  - type: VPContributorsCustom
    repo: SpechtLabs/tka
---

## Welcome to TKA

Traditional Kubernetes access control is often a headache:

- **Painful to manage** — OIDC integrations, kubeconfig sprawl, endless context switching
- **Overly centralized & complex** — auth proxies, bastion gateways, and extra moving parts

We think Kubernetes access should be different:

- **Secure by default** → ephemeral, scoped credentials
- **Simple to use** → a `tsh login`‑like UX, but for Kubernetes
- **Network‑gated** → powered by your existing [Tailscale ACLs and Grants](https://tailscale.com/kb/1324/grants)
- **Kubernetes‑native** → built on ServiceAccounts and RBAC

`tka` makes this possible by issuing **short‑lived cluster credentials**, backed by your [Tailscale](https://tailscale.com) identity and Kubernetes’ own RBAC rules.

No extra proxies. No kubeconfig sprawl. Just clean, auditable access.

## Best in class CLI UX

Ever been debugging across multiple clusters and lost track of your current kube context?

Or needed to stay logged in all day as a cluster admin without juggling tokens?

`tka` gives you two simple workflows:

- **`tka shell`** → ephemeral, auto‑cleaned sessions (perfect for quick debugging)
- **`tka login`** → longer‑lived sessions with full control (great for admins)

::: tabs

@tab `tka shell`

```shell
$ kubectl get ns
error: You must be logged in to the server (Unauthorized)

(tka) $ tka shell
✓ sign-in successful!
    ╭───────────────────────────────────────╮
    │ User:  cedi                           │
    │ Role:  developer                      │
    │ Until: Sun, 07 Sep 2025 19:24:29 CEST │
    ╰───────────────────────────────────────╯

(tka) $ kubectl version | grep Server
Server Version: v1.31.1+k3s1

(tka) $ exit
✓ You have been signed out

$ kubectl get ns
error: You must be logged in to the server (Unauthorized)
```

@tab `tka login`

```shell
$ tka login --no-eval
✓ sign-in successful!
    ╭───────────────────────────────────────╮
    │ User:  cedi                           │
    │ Role:  cluster-admin                  │
    │ Until: Mon, 08 Sep 2025 19:25:01 CEST │
    ╰───────────────────────────────────────╯
✓ kubeconfig written to:
    /tmp/kubeconfig-2950671502.yaml
→ To use this session, run:
    export KUBECONFIG=/tmp/kubeconfig-2950671502.yaml

$ export KUBECONFIG=/tmp/kubeconfig-2950671502.yaml
$ kubectl version | grep Server
Server Version: v1.31.1+k3s1

$ tka logout
✓ You have been signed out

$ kubectl get ns
error: You must be logged in to the server (Unauthorized)
```

:::

::: tip
Check out the [Overview](./overview/overview.md) page to learn how it works on a high level.
:::

## Security Notice

:::warning Experimental Security Model

This project’s security model is still **experimental**.

I’ve designed it with care, but I’m not a professional security auditor or
pentester. While it should be reasonably safe for most use cases, it’s not guaranteed to be bullet‑proof.

If you need **strong, production‑grade security**, consider using a professionally reviewed solution.

For more details, see the [Security Model documentation](overview/security.md).
:::
