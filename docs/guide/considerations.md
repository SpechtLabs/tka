---
title: Architecture
createTime: 2025/07/05 20:46:00
permalink: /guide/architecture
---

## Why using ServiceAccount + Token auth?

Generally speaking, there are two ways to authenticate requests to the K8s API Server:

1. ServiceAccount + Token Auth
2. Client Certificate Auth

### Client Certificate Auth

Using client certificate authentication is explicitly designed for "human" access to clusters and _most_ kubeconfigs do exactly that.

However, there is a catch: Implementing the client certificate auth in an Kubernetes Operator (like `tka`) involves
many moving parts, from generating certificates, certificate signing requests (csr), then posting the CSR to Kubernetes
and then approving the CSR.

Having so many moving parts, makes it easy to make a mistake and to maintain the functionality. Additionally, this creates
a bunch of new problems, like maintaining a certificate revocation list (CRL) because if we do certificates, we gotta do
it right. Then we also have to take care of cert rotations, and many more.
The last gotcha with this approach is, that the operator requires `certificates.k8s.io` API access
and cluster-admin privileges to approve the CSRs.

### Service Account + Token

Choosing the ServiceAccount + Token authentication seems to be the _wrong_ choice at first glance, as it's explicitly designed
for application-to-api authentication.

But on the other hand, it's way easier to implement and later audit. Tokens have expiry baked in, ServiceAccounts can
be easily audited, and everything is short-lived and ephemeral. Perfect for this use case.

> [!INFO]
> For very security-sensitive setups, audience scoping must be dealt with
