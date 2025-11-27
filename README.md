# provider-garage

[![CI](https://github.com/kikokikok/provider-garage/actions/workflows/ci.yaml/badge.svg)](https://github.com/kikokikok/provider-garage/actions/workflows/ci.yaml)
[![Release](https://github.com/kikokikok/provider-garage/actions/workflows/release.yaml/badge.svg)](https://github.com/kikokikok/provider-garage/actions/workflows/release.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kikokikok/provider-garage)](https://goreportcard.com/report/github.com/kikokikok/provider-garage)
[![codecov](https://codecov.io/gh/kikokikok/provider-garage/branch/main/graph/badge.svg)](https://codecov.io/gh/kikokikok/provider-garage)

A native Crossplane provider for [Garage](https://garagehq.deuxfleurs.fr/) v2+ object storage, built from scratch using the native Garage Admin API v2.

## Features

- **Native Implementation**: Direct integration with Garage Admin API v2 (no Terraform/Upjet)
- **Crossplane v2 Only**: Supports only Crossplane v2+ with namespaced resources
- **Garage v2+ Only**: Supports only Garage v2+ Admin API
- **TDD Approach**: Comprehensive unit and integration tests
- **CI/CD**: GitHub Actions for continuous integration and deployment
- **Multi-platform**: Binaries for Linux/Darwin on AMD64/ARM64

### Supported Resources

- **Bucket** (`garage.crossplane.io/v1alpha1`): Manage S3-compatible buckets
- **Key** (`garage.crossplane.io/v1alpha1`): Manage access keys with credentials
- **KeyAccess** (`garage.crossplane.io/v1alpha1`): Manage key permissions on buckets

## Installation

### Prerequisites

- Kubernetes cluster (v1.25+)
- Crossplane v2.0+
- Garage v2.0+ with Admin API enabled

### Install Provider

```bash
# Install from package (once published)
kubectl crossplane install provider kikokikok/provider-garage:latest
```

### Configure Provider

Create a secret with Garage credentials:

```bash
kubectl create secret generic garage-credentials \
  --from-literal=credentials='{
    "endpoint": "http://garage.example.com:3903",
    "adminToken": "your-admin-token"
  }' \
  -n crossplane-system
```

Create a ProviderConfig:

```yaml
apiVersion: garage.crossplane.io/v1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: garage-credentials
      namespace: crossplane-system
      key: credentials
```

## Usage

### Create a Bucket

```yaml
apiVersion: garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: my-bucket
  namespace: default
spec:
  forProvider:
    globalAlias: my-application-data
  providerConfigRef:
    name: default
```

### Create an Access Key

```yaml
apiVersion: garage.crossplane.io/v1alpha1
kind: Key
metadata:
  name: my-key
  namespace: default
spec:
  forProvider:
    name: my-app-key
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-key-credentials
    namespace: default
```

### Grant Key Access to Bucket

```yaml
apiVersion: garage.crossplane.io/v1alpha1
kind: KeyAccess
metadata:
  name: my-key-access
  namespace: default
spec:
  forProvider:
    bucketIdRef:
      name: my-bucket
    accessKeyIdRef:
      name: my-key
    permissions:
      read: true
      write: true
      owner: false
  providerConfigRef:
    name: default
```

## Development

### Prerequisites

- Go 1.21+
- Docker (for building images)
- Kind (for integration tests)
- kubectl

### Build

```bash
# Build provider binary
make build

# Run unit tests
make test

# Run integration tests (requires Kind cluster)
make test-integration

# Format code
make fmt

# Run linter
make lint

# Generate code (CRDs, deepcopy)
make generate
```

### Testing

The project follows TDD practices with comprehensive test coverage:

#### Unit Tests

```bash
# Run all unit tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

Unit tests are located in `*_test.go` files next to the code they test:
- `pkg/garage/client_test.go`: Tests for Garage API client
- `internal/controller/bucket/bucket_test.go`: Tests for bucket controller

#### Integration Tests

Integration tests run against a real Kubernetes cluster (Kind):

```bash
# Set up Kind cluster
kind create cluster --name garage-test

# Install Crossplane
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm install crossplane crossplane-stable/crossplane \
  --namespace crossplane-system --create-namespace

# Run integration tests
make test-integration
```

### CI/CD

The project uses GitHub Actions for continuous integration and deployment:

#### CI Workflow (`.github/workflows/ci.yaml`)

Runs on every push and pull request:
- **Lint**: Code quality checks with golangci-lint
- **Unit Tests**: Run all unit tests with race detection
- **Build**: Verify binary compilation
- **Integration Tests**: Run integration tests in Kind cluster
- **Coverage**: Upload coverage to Codecov

#### Release Workflow (`.github/workflows/release.yaml`)

Triggers on version tags (e.g., `v0.1.0`):
- Build binaries for multiple platforms (Linux/Darwin, AMD64/ARM64)
- Generate CRDs
- Build and push Docker image to GitHub Container Registry
- Create Crossplane package (`.xpkg`)
- Create GitHub Release with artifacts
- Publish to Upbound Marketplace

### Creating a Release

```bash
# Tag a new version
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0

# The release workflow will automatically:
# 1. Build multi-platform binaries
# 2. Create Docker image
# 3. Package Crossplane provider
# 4. Publish to GitHub Releases
# 5. Push to Upbound Marketplace (if configured)
```

### Publishing to Upbound Marketplace

To publish to Upbound Marketplace, configure these secrets in your repository:

- `UPBOUND_TOKEN`: Your Upbound CLI token
- `UPBOUND_ORG`: Your Upbound organization name

Then the release workflow will automatically publish on tagged releases.

### Project Structure

```
provider-garage/
├── .github/workflows/      # GitHub Actions CI/CD
│   ├── ci.yaml            # Continuous integration
│   └── release.yaml       # Release automation
├── apis/                   # API type definitions
│   ├── v1alpha1/          # Managed resource types
│   └── v1/           # ProviderConfig types
├── cmd/provider/          # Provider entry point
├── config/                # Kubernetes manifests
│   └── crd/bases/        # Generated CRDs
├── internal/controller/   # Resource controllers
│   └── bucket/           # Bucket controller with tests
├── pkg/garage/           # Garage Admin API client
│   ├── client.go         # API client implementation
│   └── client_test.go    # Client unit tests
├── test/integration/     # Integration test suite
└── Makefile             # Build automation
└── Makefile               # Build automation
```

## Architecture

This provider uses native Go to interact directly with the Garage Admin API v2:

1. **API Client** (`pkg/garage`): Native HTTP client for Garage Admin API
2. **CRDs** (`apis`): Kubernetes Custom Resource Definitions
3. **Controllers** (`internal/controller`): Reconciliation logic using crossplane-runtime

### No Upjet/Terraform

Unlike typical Crossplane providers, this implementation:
- ❌ Does NOT use Upjet code generation
- ❌ Does NOT use Terraform providers
- ✅ Implements native API calls to Garage Admin API v2
- ✅ Supports only Crossplane v2 patterns
- ✅ Supports only Garage v2+ API

## API References

- [Garage Admin API v2 Spec](https://garagehq.deuxfleurs.fr/api/garage-admin-v2.json)
- [Garage Admin API Documentation](https://garagehq.deuxfleurs.fr/documentation/reference-manual/admin-api/)
- [Garage S3 Compatibility](https://garagehq.deuxfleurs.fr/documentation/reference-manual/s3-compatibility/)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or pull request.
