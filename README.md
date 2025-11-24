# provider-garage

A native Crossplane provider for [Garage](https://garagehq.deuxfleurs.fr/), built with [Upjet](https://github.com/crossplane/upjet) to provide Crossplane v2-compatible managed resources for Garage S3-compatible object storage.

## Overview

This provider enables you to manage Garage resources (buckets, keys, and key access) as native Kubernetes resources through Crossplane, following the Crossplane v2 patterns with namespaced Managed Resources.

### Why a Native Provider?

- ✅ **v2 Native** - No deprecation warnings, uses namespaced Managed Resources
- ✅ **Better K8s Integration** - Full CRD schema, conditions, and events
- ✅ **Multi-tenancy** - Namespace + RBAC (v2 pattern)
- ✅ **Future-proof** - Aligns with Crossplane v2 direction
- ✅ **Direct API Access** - No Terraform overhead, faster reconciliation

## Features

This provider supports the following Garage resources:

- **Buckets** (`bucket.garage.crossplane.io/v1alpha1`)
  - Create and manage S3-compatible buckets
  - Set global aliases
  - Namespaced managed resource

- **Keys** (`key.garage.crossplane.io/v1alpha1`)
  - Create and manage access keys
  - Credentials stored in Kubernetes secrets
  - Namespaced managed resource

- **KeyAccess** (`key.garage.crossplane.io/v1alpha1`)
  - Grant keys access to buckets
  - Configure read/write/owner permissions
  - Cross-resource references

## Quick Start

### Installation

```bash
# Install the provider (once published)
kubectl crossplane install provider kikokikok/provider-garage:v0.1.0
```

### Configure Provider

```bash
# Create credentials secret
kubectl create secret generic garage-credentials \
  --from-literal=credentials='{
    "garage_endpoint": "http://garage.example.com:3903",
    "garage_admin_token": "your-admin-token-here"
  }' \
  -n crossplane-system

# Create ProviderConfig
kubectl apply -f examples/providerconfig/providerconfig.yaml
```

### Create Resources

```bash
# Create a bucket
kubectl apply -f examples/bucket/bucket.yaml

# Create an access key
kubectl apply -f examples/key/key.yaml
```

## Documentation

- **[Getting Started Guide](GETTING_STARTED.md)** - Step-by-step installation and usage
- **[Architecture](ARCHITECTURE.md)** - Technical architecture and design decisions
- **[Contributing](CONTRIBUTING.md)** - How to contribute to the project
- **[Implementation Summary](IMPLEMENTATION_SUMMARY.md)** - Complete implementation overview

## Usage Examples

### Create a Bucket

```yaml
apiVersion: bucket.garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: my-app-bucket
  namespace: my-app
spec:
  forProvider:
    globalAlias: my-app-data
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-app-bucket-connection
    namespace: my-app
```

### Create an Access Key

```yaml
apiVersion: key.garage.crossplane.io/v1alpha1
kind: Key
metadata:
  name: my-app-key
  namespace: my-app
spec:
  forProvider:
    name: my-app-access-key
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-app-credentials
    namespace: my-app
```

### Grant Key Access to Bucket

```yaml
apiVersion: key.garage.crossplane.io/v1alpha1
kind: KeyAccess
metadata:
  name: my-app-key-access
  namespace: my-app
spec:
  forProvider:
    keyIdRef:
      name: my-app-key
    bucketIdRef:
      name: my-app-bucket
    read: true
    write: true
    owner: false
  providerConfigRef:
    name: default
```

## Crossplane v2 Compatibility

This provider is designed for Crossplane v2 with namespaced resources:

### Architecture Pattern

```
User namespace: my-app
  ↓
  Namespaced MR: Bucket (v2 pattern!)
  ↓
  Provider Controller → Garage Admin API
  ↓
  Connection Secret (in my-app namespace)
```

See [examples/composition/](examples/composition/) for complete v2 XRD and Composition examples.

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/kikokikok/provider-garage.git
cd provider-garage

# Install dependencies
go mod tidy

# Build the provider
make build

# Run tests
make test
```

### Code Generation

This provider uses Upjet to generate CRDs and controllers:

```bash
# Generate all code
make generate
```

## Architecture

This provider is built using:

- **[Upjet](https://github.com/crossplane/upjet)** - Framework for building Crossplane providers from Terraform providers
- **[Garage Terraform Provider](https://github.com/deuxfleurs-org/garage-terraform-provider)** - Upstream Terraform provider for Garage
- **[Crossplane Runtime](https://github.com/crossplane/crossplane-runtime)** - Core Crossplane libraries

### How it Works

1. Upjet generates Crossplane CRDs from the Garage Terraform provider schema
2. Controllers reconcile Kubernetes resources with Garage API state
3. Terraform is used under the hood for state management
4. All resources are namespaced for v2 compatibility

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas for Contribution

- Bug fixes and improvements
- Additional Garage resources
- Integration tests
- Documentation improvements
- Example compositions

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## References

- [Garage Documentation](https://garagehq.deuxfleurs.fr/)
- [Crossplane Documentation](https://docs.crossplane.io/)
- [Upjet Documentation](https://github.com/crossplane/upjet)
- [Garage Terraform Provider](https://github.com/deuxfleurs-org/garage-terraform-provider)

## Status

🚧 **Development Status**: Ready for code generation and testing

This implementation provides a complete foundation for a native Garage Crossplane provider. The next steps are:

1. Generate CRDs with `make generate`
2. Test with a Garage instance
3. Build and publish container image
4. Publish Crossplane package

See [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) for complete details.