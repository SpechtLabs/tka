---
pageLayout: home
externalLinkIcon: false

config:
  - type: doc-hero
    hero:
      name: One command gets you secure kubectl access. No proxies, no OIDC setup, no hassle.
      text: Tailscale-native Kubernetes access
      tagline: Ephemeral credentials that auto-expire, powered by your existing Tailscale network.
      image: /logo.png
      actions:
        - text: Get Started →
          link: /getting-started/quick
          theme: brand
          icon: mdi:rocket-launch
        - text: View Documentation →
          link: /getting-started/overview
          theme: alt
          icon: mdi:book-open-page-variant

  - type: features
    title: Why Tailscale K8s Auth?
    description: Built for Kubernetes. Powered by your tailnet.
    features:
      - title: Zero-proxy simplicity
        icon: mdi:link-off
        details: Direct API access without auth proxies, reverse tunnels, or complex gateway chains. Your kubectl talks directly to the cluster.

      - title: Ephemeral by design
        icon: mdi:timer-sand
        details: Short-lived tokens that auto-expire. Get access exactly when you need it, for exactly as long as you need it.

      - title: Zero-trust, zero-ingress
        icon: mdi:shield-lock-outline
        details: No public endpoints. Access flows through your private Tailscale network with device-level attestation.

      - title: Kubernetes-native
        icon: mdi:kubernetes
        details: Built on ServiceAccounts and ClusterRoles. No custom protocols or vendor lock-in - just native Kubernetes security.

  - type: VPListCompareCustom
    title: "Traditional vs. TKA"
    description: "Compare the old way with the TKA approach"
    left:
      title: "Traditional Kubernetes Access"
      description: "Complex, fragile, and hard to maintain"
      items:
        - title: "Complex Setup"
          description: "OIDC providers, auth proxies, bastion hosts"
        - title: "Fragile Chains"
          description: "Multiple hops that break at the worst times"
        - title: "Long-lived Tokens"
          description: "Shared credentials with limited rotation"
        - title: "Manual Onboarding"
          description: "Per-environment setup and documentation drift"
        - title: "Hard to Debug"
          description: "Complex auth flows with poor visibility"

    right:
      title: "TKA Approach"
      description: "Simple, secure, and Kubernetes-native"
      items:
        - title: "One-Command Deploy"
          description: "Helm install. That's it. kubectl works immediately"
        - title: "Zero Infrastructure"
          description: "Uses your existing Tailscale network"
        - title: "Ephemeral Credentials"
          description: "Auto-expiring tokens with least privilege"
        - title: "Instant Onboarding"
          description: "If you have Tailscale, you have access"
        - title: "Clear Audit Trail"
          description: "Standard Kubernetes events and logs"

  - type: custom

  - type: VPReleases
    repo: SpechtLabs/tka

  - type: VPContributors
    repo: SpechtLabs/tka
---

## Best-in-Class Developer Experience

TKA is built by SREs who understand production operations. Every workflow is designed for real-world reliability, security, and ease of use.

We provide two intuitive workflows to provide instant, secure access without the usual ceremony:

:::: collapse accordion expand

- **`tka shell`** → ephemeral, isolated sessions (perfect for quick debugging and production safety)

  ::: terminal

  ```shell
  $ tka shell --quiet
  ✓ sign-in successful!

  (tka) $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  (tka) $ exit
  ✓ You have been signed out
  ```

  :::

- :- **`tka login`** → persistent sessions with full control (ideal for development and administration)

  ::: terminal

  ```shell
  $ tka login
  ✓ sign-in successful!

  $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  $ tka logout
  ✓ You have been signed out
  ```

  :::

::::

::: tip Found a rough edge? Have an idea for improvement?
[Open an issue](https://github.com/spechtlabs/tka/issues/new/choose) - we're always working to make Kubernetes access better.
:::

## Security & Maturity

::: info Security Model Status
TKA's security model is thoughtfully designed and suitable for many production use cases. However, it hasn't undergone formal security auditing yet.

For mission-critical environments requiring the highest security assurance, consider professionally audited solutions or [review our security documentation](understanding/security.md) to make an informed decision.
:::
