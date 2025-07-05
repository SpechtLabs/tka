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
        - text: Get Started →
          link: /guide/overview
          theme: brand
          icon: simple-icons:bookstack
        - text: GitHub Releases →
          link: https://github.com/SpechtLabs/tailscale-k8s-auth/releases
          theme: alt
          icon: simple-icons:github

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

      - title: Built for ephemeral clusters
        icon: mdi:cloud-sync-outline
        details: Great for dev/test environments that spin up and down frequently — join the tailnet, register, and go.

      - title: Central login API, federation-ready
        icon: mdi:server-network
        details: Supports a central login cluster with other clusters registering dynamically — ideal for multi-cluster orgs.

      - title: Tiny footprint, hackable core
        icon: mdi:code-tags
        details: Lightweight Go components designed to be extended, embedded, or scripted into your own workflows.


  - type: VPReleasesCustom
    repo: SpechtLabs/tailscale-k8s-auth

  - type: VPContributorsCustom
    repo: SpechtLabs/tailscale-k8s-auth
---
