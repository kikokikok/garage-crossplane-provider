# CI/CD and Release Guide

## Overview

This provider includes comprehensive CI/CD automation using GitHub Actions for testing, building, and releasing.

## Continuous Integration (CI)

The CI workflow runs automatically on every push and pull request to `main` or `master` branches.

### CI Pipeline Steps

1. **Lint** - Code quality checks using golangci-lint
2. **Unit Tests** - All tests with race detection and coverage
3. **Build** - Binary compilation verification
4. **Integration Tests** - Tests in Kind Kubernetes cluster with Crossplane

### Running CI Locally

```bash
# Lint
make lint

# Unit tests with coverage
make test
go test -v -race -coverprofile=coverage.out ./...

# Build
make build

# Integration tests (requires Kind cluster)
kind create cluster --name garage-test
make test-integration
```

## Release Process

### Creating a Release

Releases are automated through GitHub Actions and triggered by version tags.

#### Step 1: Prepare for Release

```bash
# Ensure all tests pass
make test

# Update version in documentation if needed
# Review CHANGELOG or create one

# Commit all changes
git add .
git commit -m "Prepare for release v0.1.0"
git push
```

#### Step 2: Create and Push Tag

```bash
# Create annotated tag
git tag -a v0.1.0 -m "Release v0.1.0

- Native Garage Admin API v2 integration
- Comprehensive unit and integration tests
- CI/CD with GitHub Actions
- Multi-platform binaries
- Upbound marketplace support
"

# Push tag to trigger release
git push origin v0.1.0
```

#### Step 3: Monitor Release Workflow

The release workflow will automatically:

1. **Build Multi-Platform Binaries**
   - Linux AMD64
   - Linux ARM64
   - Darwin AMD64
   - Darwin ARM64

2. **Generate CRDs**
   - Using controller-gen
   - Output to `package/crds/`

3. **Build and Push Docker Image**
   - Multi-stage build
   - Multi-architecture support (linux/amd64, linux/arm64)
   - Push to `ghcr.io/kikokikok/provider-garage:v0.1.0`
   - Also tagged as `:latest` for stable releases

4. **Create Crossplane Package**
   - Uses UP CLI (`up xpkg build`)
   - Package all CRDs and manifests
   - Create `.xpkg` file
   - Include package metadata

5. **Create GitHub Release**
   - Attach all binaries
   - Attach Crossplane package
   - Generate release notes

6. **Publish to Upbound Marketplace**
   - Uses UP CLI (`up xpkg push`)
   - Automatic if secrets are configured
   - Published to `xpkg.upbound.io/YOUR_ORG/provider-garage`

### Release Artifacts

After the release workflow completes, you'll have:

- **Binaries**: `provider-garage-{platform}-{arch}`
- **Docker Image**: `ghcr.io/kikokikok/provider-garage:v0.1.0`
- **Crossplane Package**: `provider-garage-v0.1.0.xpkg`
- **GitHub Release**: With all artifacts attached

## Upbound Marketplace Integration

### Prerequisites

To publish to Upbound Marketplace, configure these GitHub repository secrets:

1. Go to: `https://github.com/kikokikok/provider-garage/settings/secrets/actions`

2. Add secrets:
   - **UPBOUND_TOKEN**: Your Upbound CLI token
     - Get from: `up login` then `cat ~/.up/config.json`
   - **UPBOUND_ORG**: Your Upbound organization name
     - Find at: https://marketplace.upbound.io/

### Getting Upbound Token

```bash
# Install UP CLI
curl -sL https://cli.upbound.io | sh
sudo mv up /usr/local/bin/

# Login to Upbound
up login

# Get your token (stored in config)
cat ~/.up/config.json | jq -r '.upbound.default.session'

# Get your organization
up org list
```

### Publishing Process

Once secrets are configured, the release workflow will automatically:

1. Authenticate with Upbound
2. Build the project package
3. Push to Upbound Marketplace
4. Tag with version number
5. Tag as `latest` for stable releases (non-alpha/beta/rc)

### Manual Publishing (if needed)

```bash
# Set variables
export IMAGE_PATH="xpkg.upbound.io/YOUR_ORG/provider-garage"
export VERSION_TAG="v0.1.0"

# Login
up login

# Build and push
up project build --repository "${IMAGE_PATH}"
up project push --repository "${IMAGE_PATH}" --tag="${VERSION_TAG}"

# Push as latest (for stable releases)
up project push --repository "${IMAGE_PATH}" --tag="latest"
```

## Version Numbering

Follow semantic versioning (semver):

- **v0.1.0**: Initial release
- **v0.1.1**: Bug fixes
- **v0.2.0**: New features (backwards compatible)
- **v1.0.0**: First stable release
- **v1.1.0-alpha.1**: Pre-release versions

### Pre-release Versions

Pre-release versions (alpha, beta, rc) will:
- Create GitHub release marked as "pre-release"
- Publish to Upbound but NOT tag as "latest"

Example:
```bash
git tag -a v0.2.0-alpha.1 -m "Alpha release for testing"
git push origin v0.2.0-alpha.1
```

## Monitoring Releases

### GitHub Actions

View workflow runs:
- https://github.com/kikokikok/provider-garage/actions

### GitHub Releases

View published releases:
- https://github.com/kikokikok/provider-garage/releases

### Upbound Marketplace

View published packages:
- https://marketplace.upbound.io/providers/YOUR_ORG/provider-garage

### Docker Images

View published images:
- https://github.com/kikokikok/provider-garage/pkgs/container/provider-garage

## Troubleshooting

### Release Workflow Fails

1. **Check logs**: Go to Actions tab in GitHub
2. **Common issues**:
   - Missing secrets (UPBOUND_TOKEN, UPBOUND_ORG)
   - Build failures (run tests locally first)
   - CRD generation failures (check controller-gen version)

### Upbound Publishing Fails

1. **Verify secrets are set correctly**
2. **Check Upbound organization name**
3. **Verify token hasn't expired**
4. **Test manually**:
   ```bash
   up login
   up org list
   ```

### Binary Build Fails for Specific Platform

1. **Test locally**:
   ```bash
   GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build cmd/provider/main.go
   ```

2. **Check Go version compatibility**
3. **Review build flags**

## Best Practices

### Before Releasing

- ✅ Run all tests locally
- ✅ Update documentation
- ✅ Review code changes
- ✅ Update CHANGELOG
- ✅ Test in development environment
- ✅ Create clear commit messages

### Tag Messages

Include meaningful tag messages:
```bash
git tag -a v0.1.0 -m "Release v0.1.0

Changes:
- Added feature X
- Fixed bug Y
- Improved performance Z

Breaking changes:
- None
"
```

### Release Cadence

- **Patch releases** (v0.1.x): As needed for bugs
- **Minor releases** (v0.x.0): Monthly or bi-monthly
- **Major releases** (vx.0.0): When breaking changes occur

## Next Steps

1. Configure Upbound marketplace secrets
2. Create your first release (v0.1.0)
3. Verify all artifacts are published
4. Install from marketplace to test
5. Monitor CI/CD pipelines for future changes
