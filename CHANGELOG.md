# Changelog

## [1.0.0](https://github.com/SpechtLabs/tka/compare/v0.1.3...v1.0.0) (2025-10-27)


### ⚠ BREAKING CHANGES

* Import path changed from pkg/tailscale to pkg/tshttp

### Features

* **docs:** enhance CLI documentation generation with front matter options ([c73e54d](https://github.com/SpechtLabs/tka/commit/c73e54ddc8231dc7c2223ba79d79927e13a64511))
* **tailscale.Server:** add support for TLS and Funnel protocols in Tailecale server ([#67](https://github.com/SpechtLabs/tka/issues/67)) ([1526960](https://github.com/SpechtLabs/tka/commit/1526960f9035f8074e028b8f0b386db483a2e4d4))


### Bug Fixes

* **deps:** update kubernetes packages to v0.34.1 ([#63](https://github.com/SpechtLabs/tka/issues/63)) ([a789665](https://github.com/SpechtLabs/tka/commit/a789665a92799257315c8e83d023ff377da0de5a))
* **deps:** update module github.com/gin-gonic/gin to v1.11.0 ([#66](https://github.com/SpechtLabs/tka/issues/66)) ([c24481f](https://github.com/SpechtLabs/tka/commit/c24481f6b6b4c7eaa6a64807c6bf0e1f5d7141d6))
* **deps:** update module github.com/swaggo/files to v2 ([#51](https://github.com/SpechtLabs/tka/issues/51)) ([adc24e2](https://github.com/SpechtLabs/tka/commit/adc24e2153148dffcdd150db2e286792bc7a9a9a))
* **deps:** update module golang.org/x/sys to v0.37.0 ([#85](https://github.com/SpechtLabs/tka/issues/85)) ([9bc89a8](https://github.com/SpechtLabs/tka/commit/9bc89a88c5859e8209d1d302451ac68419c373e4))
* **deps:** update module golang.org/x/term to v0.36.0 ([#91](https://github.com/SpechtLabs/tka/issues/91)) ([e418857](https://github.com/SpechtLabs/tka/commit/e418857f9700755901db2ec2c6f908ba9603078d))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.22.2 ([#83](https://github.com/SpechtLabs/tka/issues/83)) ([02bd469](https://github.com/SpechtLabs/tka/commit/02bd4695d6960b93bff45da079d4a690d5b240b8))
* **deps:** update module sigs.k8s.io/controller-runtime to v0.22.3 ([#89](https://github.com/SpechtLabs/tka/issues/89)) ([13ccbee](https://github.com/SpechtLabs/tka/commit/13ccbeec416285452d36c5fdef621205b2a5807a))
* **deps:** update module tailscale.com to v1.88.3 ([#59](https://github.com/SpechtLabs/tka/issues/59)) ([4da3333](https://github.com/SpechtLabs/tka/commit/4da33332c828232e5c10c99ecac2739c23cc1a42))
* **deps:** update module tailscale.com to v1.88.4 ([#95](https://github.com/SpechtLabs/tka/issues/95)) ([a35934b](https://github.com/SpechtLabs/tka/commit/a35934b27784d72d17c70c0631b2ec6e00ecf3d6))
* **deps:** update module tailscale.com to v1.90.0 ([#99](https://github.com/SpechtLabs/tka/issues/99)) ([ed618b7](https://github.com/SpechtLabs/tka/commit/ed618b7a735d7d4d06215bc4873f0c78d96365e9))
* **deps:** update module tailscale.com to v1.90.2 ([#100](https://github.com/SpechtLabs/tka/issues/100)) ([96d6169](https://github.com/SpechtLabs/tka/commit/96d6169ae29b147b80ef4e159fd2591b5a20eb3c))


### Code Refactoring

* rename tailscale package to tshttp ([11c1ae9](https://github.com/SpechtLabs/tka/commit/11c1ae93ee8d1e4807ed04487f68fbe1a0ded1c1))
