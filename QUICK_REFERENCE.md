# Quick Reference

## Installation

```bash
# Install Crossplane
helm install crossplane --namespace crossplane-system --create-namespace crossplane-stable/crossplane

# Install provider-garage
kubectl crossplane install provider kikokikok/provider-garage:v0.1.0

# Create credentials
kubectl create secret generic garage-credentials \
  --from-literal=credentials='{"garage_endpoint":"http://garage:3903","garage_admin_token":"YOUR_TOKEN"}' \
  -n crossplane-system

# Create ProviderConfig
kubectl apply -f examples/providerconfig/providerconfig.yaml
```

## Resource Types

| Resource | API Group | Version | Scope |
|----------|-----------|---------|-------|
| Bucket | bucket.garage.crossplane.io | v1alpha1 | Namespaced |
| Key | key.garage.crossplane.io | v1alpha1 | Namespaced |
| KeyAccess | key.garage.crossplane.io | v1alpha1 | Namespaced |
| ProviderConfig | garage.crossplane.io | v1beta1 | Cluster |

## Common Commands

```bash
# List all resources
kubectl get buckets -A
kubectl get keys -A
kubectl get keyaccesses -A

# Describe a resource
kubectl describe bucket my-bucket -n my-namespace

# View resource status
kubectl get bucket my-bucket -n my-namespace -o yaml

# View connection secret
kubectl get secret my-credentials -n my-namespace -o yaml

# Delete resources (in order)
kubectl delete keyaccess my-access -n my-namespace
kubectl delete key my-key -n my-namespace
kubectl delete bucket my-bucket -n my-namespace
```

## Example: Complete Setup

```bash
# 1. Create namespace
kubectl create namespace my-app

# 2. Create bucket
cat <<EOF | kubectl apply -f -
apiVersion: bucket.garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: my-bucket
  namespace: my-app
spec:
  forProvider:
    globalAlias: my-application-data
  providerConfigRef:
    name: default
EOF

# 3. Create key
cat <<EOF | kubectl apply -f -
apiVersion: key.garage.crossplane.io/v1alpha1
kind: Key
metadata:
  name: my-key
  namespace: my-app
spec:
  forProvider:
    name: my-application-key
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-s3-credentials
    namespace: my-app
EOF

# 4. Grant access
cat <<EOF | kubectl apply -f -
apiVersion: key.garage.crossplane.io/v1alpha1
kind: KeyAccess
metadata:
  name: my-access
  namespace: my-app
spec:
  forProvider:
    keyIdRef:
      name: my-key
    bucketIdRef:
      name: my-bucket
    read: true
    write: true
  providerConfigRef:
    name: default
EOF

# 5. Wait for resources to be ready
kubectl wait --for=condition=Ready bucket/my-bucket -n my-app --timeout=300s
kubectl wait --for=condition=Ready key/my-key -n my-app --timeout=300s

# 6. Use credentials in your app
kubectl get secret my-s3-credentials -n my-app -o jsonpath='{.data.access_key_id}' | base64 -d
kubectl get secret my-s3-credentials -n my-app -o jsonpath='{.data.secret_access_key}' | base64 -d
```

## Troubleshooting

### Check provider logs
```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-garage
```

### Check resource events
```bash
kubectl describe bucket my-bucket -n my-app
```

### Verify ProviderConfig
```bash
kubectl get providerconfig default -o yaml
```

### Common Issues

**Bucket not created**
- Check Garage endpoint is accessible
- Verify admin token is valid
- Review provider logs

**Secret not generated**
- Ensure `writeConnectionSecretToRef` is specified
- Check namespace exists
- Verify provider permissions

**KeyAccess fails**
- Ensure Key and Bucket exist first
- Check references are correct
- Verify permissions in Garage

## Environment Variables for S3 Clients

When using the credentials in your applications:

```bash
export AWS_ACCESS_KEY_ID=$(kubectl get secret my-s3-credentials -n my-app -o jsonpath='{.data.access_key_id}' | base64 -d)
export AWS_SECRET_ACCESS_KEY=$(kubectl get secret my-s3-credentials -n my-app -o jsonpath='{.data.secret_access_key}' | base64 -d)
export S3_ENDPOINT=http://garage.example.com:3900
export S3_BUCKET=my-application-data
export S3_REGION=garage
```

## Documentation

- 📖 [README.md](README.md) - Overview and features
- 🚀 [GETTING_STARTED.md](GETTING_STARTED.md) - Detailed guide
- 🏗️ [ARCHITECTURE.md](ARCHITECTURE.md) - Technical details
- 🤝 [CONTRIBUTING.md](CONTRIBUTING.md) - How to contribute
- 📝 [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) - Implementation details

## Examples

- [examples/providerconfig/](examples/providerconfig/) - ProviderConfig setup
- [examples/bucket/](examples/bucket/) - Bucket examples
- [examples/key/](examples/key/) - Key and KeyAccess examples
- [examples/composition/](examples/composition/) - Crossplane v2 XRD examples

## Support

- 🐛 [Report a bug](https://github.com/kikokikok/provider-garage/issues)
- 💬 [Ask a question](https://github.com/kikokikok/provider-garage/discussions)
- 🤝 [Contribute](CONTRIBUTING.md)
