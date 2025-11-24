# provider-garage

A native Crossplane provider for [Garage](https://garagehq.deuxfleurs.fr/) v2+ object storage, built from scratch using the native Garage Admin API v2.

## Features

- **Native Implementation**: Direct integration with Garage Admin API v2 (no Terraform/Upjet)
- **Crossplane v2 Only**: Supports only Crossplane v2+ with namespaced resources
- **Garage v2+ Only**: Supports only Garage v2+ Admin API

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
apiVersion: garage.crossplane.io/v1beta1
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

### Build

```bash
# Build provider binary
make build

# Run tests
make test

# Format code
make fmt
```

### Project Structure

```
provider-garage/
├── apis/                    # API type definitions
│   ├── v1alpha1/           # Managed resource types
│   └── v1beta1/            # ProviderConfig types
├── cmd/provider/           # Provider entry point
├── internal/controller/    # Resource controllers
├── pkg/garage/             # Garage Admin API client
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
