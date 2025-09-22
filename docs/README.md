---
pageLayout: home
externalLinkIcon: false

config:
  - type: doc-hero
    hero:
      name: Tailscale-native Kubernetes access
      text: Skip the proxies, gateways, and OIDC complexity.
      tagline: Secure, ephemeral cluster access — powered entirely by your Tailscale network and identity.
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
      - title: Zero-trust, zero-ingress
        icon: mdi:shield-lock-outline
        details: No public endpoints or exposed services. Access flows through your private Tailscale network with device-level attestation and ACL enforcement.

      - title: Ephemeral credentials by design
        icon: mdi:timer-sand
        details: Short-lived tokens that auto-expire. Get access exactly when you need it, for exactly as long as you need it — and not a second longer.

      - title: Kubernetes-native architecture
        icon: mdi:kubernetes
        details: Built on ServiceAccounts, ClusterRoles, and standard APIs. No custom auth protocols or vendor lock-in — just native Kubernetes security.

      - title: GitOps-ready configuration
        icon: mdi:account-key-outline
        details: Declarative grant-to-role mapping via CRDs. Define who gets what access in code, review in pull requests, deploy via GitOps.

      - title: Zero-proxy simplicity
        icon: mdi:link-off
        details: Direct API access without auth proxies, reverse tunnels, or complex gateway chains. Your kubectl talks directly to the cluster.

      - title: Multi-cluster federation
        icon: mdi:server-network
        details: Central authentication with distributed clusters. One login service, many target clusters. Perfect for platform teams and enterprise deployments.

  - type: custom

  - type: VPReleasesCustom
    repo: SpechtLabs/tka

  - type: VPContributorsCustom
    repo: SpechtLabs/tka
---

## Welcome to TKA

Traditional Kubernetes access control is broken:

- **Painful to manage** → OIDC integrations, kubeconfig sprawl, endless context switching
- **Overly complex** → auth proxies, bastion gateways, and brittle auth chains
- **Security gaps** → long-lived tokens, shared credentials, unclear audit trails

TKA solves this with a fundamentally simpler approach:

- **Secure by default** → ephemeral, scoped credentials that auto-expire
- **Zero infrastructure** → leverage your existing Tailscale network and ACLs
- **Kubernetes-native** → built on ServiceAccounts, RBAC, and standard APIs
- **Easy deployment** → One-command Helm chart deployment with production-ready defaults

TKA issues **short-lived cluster credentials** backed by your [Tailscale](https://tailscale.com) identity and Kubernetes' native RBAC. No proxies, no sprawl, no complexity.

## Best-in-Class Developer Experience

Stop fighting your Kubernetes access tooling. TKA provides two intuitive workflows designed by SREs for real-world operations:

- **`tka shell`** → ephemeral, isolated sessions (perfect for quick debugging and production safety)
- **`tka login`** → persistent sessions with full control (ideal for development and administration)

Both workflows provide instant, secure access without the usual ceremony.

::: tabs

@tab `tka shell`

  ::: collapse accordion expand

- Quiet output

  ```shell
  $ kubectl get ns
  error: You must be logged in to the server (Unauthorized)

  $ tka shell --quiet
  (tka) $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  (tka) $ exit

  $ kubectl get ns
  error: You must be logged in to the server (Unauthorized)
  ```

- :- Verbose output

  ```shell
  $ kubectl get ns
  error: You must be logged in to the server (Unauthorized)

  $ tka shell
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

  :::

@tab `tka login`

  ::: collapse accordion expand

- With [Shell Integration](./how-to/shell-integration.md)

  ```shell
  $ tka login

  $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  $ tka logout --quiet

  $ kubectl get ns
  error: You must be logged in to the server (Unauthorized)
  ```

- :- Quiet output / Without [Shell integration](./how-to/shell-integration.md)

  You can use `tka login --quiet` to only print the use statement for your kubeconfig

  ```shell
  $ tka login --no-eval --quiet
  export KUBECONFIG=/tmp/kubeconfig-2950671502.yaml

  $ export KUBECONFIG=/tmp/kubeconfig-2950671502.yaml
  $ kubectl version | grep Server
  Server Version: v1.31.1+k3s1

  $ tka logout --quiet

  $ kubectl get ns
  error: You must be logged in to the server (Unauthorized)
  ```

  > [!TIP]
  > This is what the [Shell Integration](./how-to/shell-integration.md) uses under the hood, to essentially do a
  >
  > ```bash
  > eval "$(command ts-k8s-auth login --quiet)"
  > ```
  >
  > when you run `tka login`.
  >
  > You can leverage this for your own, custom shell integrations.

- :- Verbose output without [Shell integration](./how-to/shell-integration.md)

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

TKA is built by SREs who understand production operations. Every workflow is designed for real-world reliability, security, and ease of use.

Found a rough edge? Have an idea for improvement? [Open an issue](https://github.com/spechtlabs/tka/issues/new/choose) — we're always working to make Kubernetes access better.

## Security Notice

:::warning Early Stage Security Model

TKA's security model is thoughtfully designed but still **evolving**.

While built with security best practices and suitable for many use cases, it hasn't undergone professional security auditing. For mission-critical production environments requiring the highest security assurance, consider professionally audited solutions.

For detailed security information, see the [Security Model documentation](explanation/security.md).
:::
