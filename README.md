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

## Installation

### Prerequisites

- Kubernetes cluster (v1.25+)
- Crossplane (v1.14+)
- Garage cluster with admin API access

### Install the Provider

```bash
# Install the provider (once published)
kubectl crossplane install provider kikokikok/provider-garage:v0.1.0

# Or use a Provider manifest
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-garage
spec:
  package: kikokikok/provider-garage:v0.1.0
EOF
```

### Configure Provider Credentials

Create a secret with your Garage credentials:

```bash
kubectl create secret generic garage-credentials \
  --from-literal=credentials='{
    "garage_endpoint": "http://garage.example.com:3903",
    "garage_admin_token": "your-admin-token-here"
  }' \
  -n crossplane-system
```

Create a ProviderConfig:

```yaml
apiVersion: garage.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: garage-credentials
      key: credentials
```

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

The connection secret will contain:
- `access_key_id`
- `secret_access_key`

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

### Building v2 Compositions

You can create v2 XRDs that compose these managed resources:

```yaml
apiVersion: apiextensions.crossplane.io/v2
kind: CompositeResourceDefinition
metadata:
  name: garagebuckets.s3.example.com
spec:
  scope: Namespaced  # v2 default
  group: s3.example.com
  names:
    kind: GarageBucket
    plural: garagebuckets
```

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

This provider uses Upjet to generate CRDs and controllers from the Garage Terraform provider:

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

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Setup

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## References

- [Garage Documentation](https://garagehq.deuxfleurs.fr/)
- [Crossplane Documentation](https://docs.crossplane.io/)
- [Upjet Documentation](https://github.com/crossplane/upjet)
- [Garage Terraform Provider](https://github.com/deuxfleurs-org/garage-terraform-provider)