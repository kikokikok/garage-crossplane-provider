# Getting Started with provider-garage

This guide will walk you through setting up and using the Garage Crossplane provider.

## Prerequisites

- Kubernetes cluster (v1.25+)
- Crossplane installed (v1.14+)
- Garage cluster with admin API access
- `kubectl` configured to access your cluster

## Installation

### Step 1: Install Crossplane

If you haven't already installed Crossplane:

```bash
# Add Crossplane Helm repository
helm repo add crossplane-stable https://charts.crossplane.io/stable
helm repo update

# Install Crossplane
helm install crossplane \
  --namespace crossplane-system \
  --create-namespace \
  crossplane-stable/crossplane
```

Wait for Crossplane to be ready:

```bash
kubectl wait --for=condition=available --timeout=300s \
  deployment/crossplane -n crossplane-system
```

### Step 2: Install provider-garage

Once the provider is published to a registry, you can install it using a Provider manifest:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-garage
spec:
  package: kikokikok/provider-garage:v0.1.0
EOF
```

Verify the provider is healthy:

```bash
kubectl get providers
kubectl get crds | grep garage
```

You should see CRDs for:
- `buckets.bucket.garage.crossplane.io`
- `keys.key.garage.crossplane.io`
- `keyaccesses.key.garage.crossplane.io`
- `providerconfigs.garage.crossplane.io`

## Configuration

### Step 3: Create Garage Credentials Secret

Create a Kubernetes secret with your Garage admin API credentials:

```bash
kubectl create secret generic garage-credentials \
  --from-literal=credentials='{
    "garage_endpoint": "http://your-garage-host:3903",
    "garage_admin_token": "your-admin-token-here"
  }' \
  -n crossplane-system
```

**Security Note**: In production, use more secure methods to manage credentials, such as:
- External Secrets Operator
- Sealed Secrets
- Vault
- Cloud provider secret managers

### Step 4: Create ProviderConfig

Create a ProviderConfig that references the secret:

```bash
cat <<EOF | kubectl apply -f -
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
EOF
```

Verify the ProviderConfig:

```bash
kubectl get providerconfigs
```

## Usage

### Creating a Bucket

Create a namespaced Garage bucket:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: bucket.garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: my-app-bucket
  namespace: default
spec:
  forProvider:
    globalAlias: my-application-data
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-app-bucket-connection
    namespace: default
EOF
```

Check the bucket status:

```bash
kubectl get buckets -n default
kubectl describe bucket my-app-bucket -n default
```

### Creating an Access Key

Create an access key for authenticating to Garage:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: key.garage.crossplane.io/v1alpha1
kind: Key
metadata:
  name: my-app-key
  namespace: default
spec:
  forProvider:
    name: my-application-key
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: my-app-credentials
    namespace: default
EOF
```

The credentials will be stored in a secret:

```bash
kubectl get secret my-app-credentials -n default -o yaml
```

The secret will contain:
- `access_key_id`: The S3 access key ID
- `secret_access_key`: The S3 secret access key

### Granting Key Access to Bucket

Grant the access key permissions to the bucket:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: key.garage.crossplane.io/v1alpha1
kind: KeyAccess
metadata:
  name: my-app-key-access
  namespace: default
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
EOF
```

Verify the access:

```bash
kubectl get keyaccesses -n default
kubectl describe keyaccess my-app-key-access -n default
```

## Using Credentials in Applications

Now you can mount the credentials secret in your application pods:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app
  namespace: default
spec:
  containers:
  - name: app
    image: my-app:latest
    env:
    - name: S3_ENDPOINT
      value: "http://your-garage-host:3900"
    - name: S3_BUCKET
      value: "my-application-data"
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: my-app-credentials
          key: access_key_id
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: my-app-credentials
          key: secret_access_key
```

## Multi-Tenancy Pattern

With namespaced resources, you can easily support multi-tenancy:

```bash
# Create namespace for team-a
kubectl create namespace team-a

# Team A creates their bucket
cat <<EOF | kubectl apply -f -
apiVersion: bucket.garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: team-a-bucket
  namespace: team-a
spec:
  forProvider:
    globalAlias: team-a-data
  providerConfigRef:
    name: default
  writeConnectionSecretToRef:
    name: team-a-credentials
    namespace: team-a
EOF

# Team A's resources are isolated in their namespace
# Team B can have similar resources in their own namespace
```

## Cleanup

To delete resources:

```bash
# Delete KeyAccess first (depends on Key and Bucket)
kubectl delete keyaccess my-app-key-access -n default

# Delete Key
kubectl delete key my-app-key -n default

# Delete Bucket
kubectl delete bucket my-app-bucket -n default
```

## Troubleshooting

### Check Provider Logs

```bash
kubectl logs -n crossplane-system -l pkg.crossplane.io/provider=provider-garage
```

### Check Resource Status

```bash
kubectl describe bucket my-app-bucket -n default
```

Look for events and conditions that indicate what's happening.

### Check ProviderConfig

```bash
kubectl get providerconfig default -o yaml
```

Ensure the credentials secret exists and is accessible.

### Common Issues

**Issue**: Bucket not being created

**Solution**: 
- Verify Garage endpoint is accessible from the cluster
- Check admin token is valid
- Review provider logs for errors

**Issue**: Connection secret not created

**Solution**:
- Ensure `writeConnectionSecretToRef` is specified
- Check RBAC permissions for the provider
- Verify the namespace exists

## Next Steps

- Explore [Crossplane Compositions](https://docs.crossplane.io/latest/concepts/compositions/) to create higher-level abstractions
- Set up GitOps with ArgoCD or Flux to manage your Garage resources
- Implement backup and disaster recovery strategies
- Monitor your Garage resources with Prometheus and Grafana

## Additional Resources

- [Garage Documentation](https://garagehq.deuxfleurs.fr/)
- [Crossplane Documentation](https://docs.crossplane.io/)
- [Provider GitHub Repository](https://github.com/kikokikok/provider-garage)
