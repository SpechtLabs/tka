---
title: Authentication and Credential Model
permalink: /architecture/authentication-model
createTime: 2025/08/25 06:33:49
---

### Options considered

Two primary approaches to human access were considered:

1) Client certificate auth
2) ServiceAccount + token auth

#### Client certificate auth

Pros: designed for human access; familiar kubeconfigs. Cons: operator complexity (CSR generation/approval), certificate lifecycle management (rotation, revocation/CRL), and elevated cluster privileges required for CSR approval (`certificates.k8s.io`).

#### ServiceAccount + token

Pros: simpler operator implementation, built-in expiry on tokens, straightforward auditing, ephemeral by design. Cons: originally intended for workloads; requires careful audience and scope consideration in sensitive setups.

Given TKA’s goals (simplicity, auditability, short-lived access), ServiceAccount tokens are the better fit.

> For very security-sensitive setups, ensure audience scoping is handled appropriately.
