---
pageLayout: home
externalLinkIcon: false

config:
  - type: doc-hero
    hero:
      name: Tailscale-native Kubernetes access
      text: Skip the proxies, gateways, and OIDC complexity.
      tagline: Secure, ephemeral cluster access - powered entirely by your Tailscale network and identity.
      image: /logo.png
      actions:
        - text: Overview →
          link: /explanation/overview
          theme: brand
          icon: fa:book
        - text: Get Started →
          link: /tutorials/quick
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
      - title: Ephemeral credentials by design
        icon: mdi:timer-sand
        details: Short-lived tokens that auto-expire. Get access exactly when you need it, for exactly as long as you need it - and not a second longer.

      - title: Zero-trust, zero-ingress
        icon: mdi:shield-lock-outline
        details: No public auth endpoints. Access request flow through your private Tailscale network with device-level attestation and ACL enforcement.

      - title: Kubernetes-native architecture
        icon: mdi:kubernetes
        details: Built on ServiceAccounts, ClusterRoles, and standard APIs. No custom auth protocols or vendor lock-in - just native Kubernetes security.

      - title: Zero-proxy simplicity
        icon: mdi:link-off
        details: Direct API access without auth proxies, reverse tunnels, or complex gateway chains that always break. Your kubectl talks directly to the cluster.

  - type: VPListCompareCustom
    title: "Why switch to TKA?"
    description: "See how TKA simplifies and secures Kubernetes access compared to legacy methods."
    left:
      title: "Traditional Kubernetes access is broken"
      description: "The old stack is hard to operate, fragile to scale, and risky by default."
      items:
        - title: "Painful to manage"
          description: "OIDC wiring, kubeconfig sprawl, and constant context switching"
        - title: "Fragile access chains"
          description: "Auth proxies, bastion hosts, and brittle hops that break at 2 a.m."
        - title: "Security gaps"
          description: "Long‑lived tokens, shared credentials, and limited auditability"
        - title: "Productivity drag"
          description: "Re‑auth loops, stale configs, and per‑cluster snowflakes"
        - title: "Onboarding friction"
          description: "Per‑env setup, docs drift, and a zoo of CLIs"

    right:
      title: "TKA is simpler, secure, and Kubernetes‑native"
      description: "A thin control plane that uses your tailnet and native RBAC. No gateways."
      items:
        - title: "Ephemeral by design"
          description: "Short‑lived, scoped credentials that auto‑expire (least privilege)"
        - title: "Zero infrastructure"
          description: "No proxies or gateways. Use your existing Tailscale network and ACLs"
        - title: "Native RBAC"
          description: "Built on ServiceAccounts, ClusterRoles, and standard Kubernetes APIs"
        - title: "Clear audit trail"
          description: "Identity via Tailscale and Kubernetes events/logs you already use"
        - title: "Fast onboarding"
          description: "One‑command Helm deploy. kubectl works out of the box"

  - type: custom

  - type: VPReleases
    repo: SpechtLabs/tka

  - type: VPContributors
    repo: SpechtLabs/tka
---

## Best-in-Class Developer Experience

TKA is built by SREs who understand production operations. Every workflow is designed for real-world reliability, security, and ease of use.

We provide two intuitive workflows to provide instant, secure access without the usual ceremony:

::: collapse accordion expand

- **`tka shell`** → ephemeral, isolated sessions (perfect for quick debugging and production safety)

  <!-- markdownlint-disable MD033 -->
  <Terminal>

  ```shell
  $ tka shell --quiet
  ✓ sign-in successful!

  (tka) $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  (tka) $ exit
  ✓ You have been signed out
  ```

  </Terminal>
  <!-- markdownlint-enable MD033 -->

- :- **`tka login`** → persistent sessions with full control (ideal for development and administration)

  <!-- markdownlint-disable MD033 -->
  <Terminal>

  ```shell
  $ tka login
  ✓ sign-in successful!

  $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  $ tka logout
  ✓ You have been signed out
  ```

  </Terminal>
  <!-- markdownlint-enable MD033 -->

  > [!NOTE]
  > This requires the [Shell Integration](./how-to/shell-integration.md) to be set-up

:::

<!-- markdownlint-disable MD033-->
<br />
<!-- markdownlint-enable MD033-->

::: tip Found a rough edge? Have an idea for improvement?
[Open an issue](https://github.com/spechtlabs/tka/issues/new/choose) - we're always working to make Kubernetes access better.
:::

## Security Notice

:::warning Early Stage Security Model

TKA's security model is thoughtfully designed but still **evolving**.

While built with security best practices and suitable for many use cases, it hasn't undergone professional security auditing. For mission-critical production environments requiring the highest security assurance, consider professionally audited solutions.

For detailed security information, see the [Security Model documentation](explanation/security.md).
:::
