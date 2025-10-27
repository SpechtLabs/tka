# Contributing

## Releases

This project uses [release-please](https://github.com/googleapis/release-please) and [goreleaser](https://goreleaser.com/) to automate releases based on [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

Releases are **fully automated** and follow this flow:

### 1. Merge Code to `main`

All new features, fixes, and changes are merged into the `main` branch via pull requests using Conventional Commits.

**Commit Types:**

- `feat:` - New feature (triggers minor version bump)
- `fix:` - Bug fix (triggers patch version bump)
- `feat!:` or `fix!:` - Breaking change (triggers major version bump)
- `chore:`, `docs:`, `style:`, `refactor:`, `test:` - No version bump

**Examples:**

```text
feat: add cluster info API endpoint
fix: correct kubeconfig expiration time
feat!: change authentication API contract
chore: update dependencies
docs: improve getting started guide
```

### 2. Release PR is Auto-Created

After commits are merged to `main`, the `release-please` GitHub Action:

- Analyzes commits since the last release
- Calculates the next version based on conventional commits (e.g., `v1.2.3`)
- Creates or updates a release pull request (e.g., `chore: release v1.2.3`)
- Generates/updates `CHANGELOG.md` with all changes

> [!IMPORTANT]
> The release PR should **not be edited manually**. If something is wrong, fix the commit messages and release-please will update the PR automatically.

### 3. Merge the Release PR

Once the release PR is reviewed and merged:

- `release-please` creates a new Git tag (e.g., `v1.2.3`)
- The tag points to the merge commit on `main`
- This automatically triggers the release workflow

### 4. Automated Release Build

The release workflow automatically:

- Runs all validations and tests
- Builds binaries for all platforms (Linux, macOS, Windows; amd64, arm64)
- Creates multi-arch Docker images
  - `ghcr.io/spechtlabs/tka:v1.2.3`
  - `ghcr.io/spechtlabs/tka:v1`
  - `ghcr.io/spechtlabs/tka:latest`
- Signs images with Cosign (keyless, GitHub OIDC)
- Creates OS packages (deb, rpm, archlinux)
- Creates a GitHub Release with all artifacts

Once complete, the new release is available at:
[github.com/SpechtLabs/tka/releases](https://github.com/SpechtLabs/tka/releases)

## Container Images

Three Docker image tags are maintained:

- `ghcr.io/spechtlabs/tka:main` - Latest from main branch (updated on every merge)
- `ghcr.io/spechtlabs/tka:v1.2.3` - Specific release version
- `ghcr.io/spechtlabs/tka:latest` - Latest stable release

All images are multi-architecture (linux/amd64 and linux/arm64).

## Notes

- **Never push tags manually** - Tags are created by release-please
- **Never edit CHANGELOG.md manually** - It's generated from commits
- **Use conventional commits** - They drive the entire release process
- **Test with `:main` tag** - Before cutting a release, test the main branch image
- **Retry failed releases** - Delete the failed tag and re-run the workflow
