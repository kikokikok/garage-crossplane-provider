#!/usr/bin/env bash
set -e

# setting up colors
BLU='\033[0;34m'
YLW='\033[0;33m'
GRN='\033[0;32m'
RED='\033[0;31m'
NOC='\033[0m' # No Color

echo_info() {
    printf "\n${BLU}%s${NOC}" "$1"
}
echo_step() {
    printf "\n${BLU}>>>>>>> %s${NOC}\n" "$1"
}
echo_sub_step() {
    printf "\n${BLU}>>> %s${NOC}\n" "$1"
}
echo_step_completed() {
    printf "${GRN} [✔]${NOC}"
}
echo_success() {
    printf "\n${GRN}%s${NOC}\n" "$1"
}
echo_warn() {
    printf "\n${YLW}%s${NOC}" "$1"
}
echo_error() {
    printf "\n${RED}%s${NOC}" "$1"
    exit 1
}

# ------------------------------
projectdir="$( cd "$( dirname "${BASH_SOURCE[0]}")"/../.. && pwd )"
scriptdir="$(dirname "$0")"

# ------------------------------
# Configuration
SAFEHOSTARCH="${SAFEHOSTARCH:-amd64}"
KIND_VERSION="${KIND_VERSION:-v0.20.0}"
KIND_NODE_IMAGE_TAG="${KIND_NODE_IMAGE_TAG:-v1.28.0}"
HELM_VERSION="${HELM_VERSION:-v3.13.0}"
KUBECTL="${KUBECTL:-kubectl}"
KIND="${KIND:-kind}"
HELM="${HELM:-helm}"

K8S_CLUSTER="${K8S_CLUSTER:-provider-garage-inttests}"
PACKAGE_NAME="provider-garage"
GARAGE_ADMIN_TOKEN="test-admin-token"

# cleanup on exit
if [ "$skipcleanup" != true ]; then
  function cleanup {
    echo_step "Cleaning up..."
    export KUBECONFIG=
    "${KIND}" delete cluster --name="${K8S_CLUSTER}" 2>/dev/null || true
  }
  trap cleanup EXIT
fi

setup_cluster() {
  echo_step "Creating Kind cluster ${K8S_CLUSTER}"
  
  local node_image="kindest/node:${KIND_NODE_IMAGE_TAG}"
  "${KIND}" create cluster --name="${K8S_CLUSTER}" --wait=5m --image="${node_image}"
  
  echo_step "Creating crossplane-system namespace"
  "${KUBECTL}" create ns crossplane-system
}

cleanup_cluster() {
  "${KIND}" delete cluster --name="${K8S_CLUSTER}"
}

setup_crossplane() {
  echo_step "Installing Crossplane from stable channel"
  
  "${HELM}" repo add crossplane-stable https://charts.crossplane.io/stable/ --force-update
  local chart_version="$("${HELM}" search repo crossplane-stable/crossplane | awk 'FNR == 2 {print $2}')"
  echo_info "Using Crossplane version ${chart_version}"
  echo
  "${HELM}" install crossplane --namespace crossplane-system crossplane-stable/crossplane --version ${chart_version} --wait
}

build_and_load_provider() {
  echo_step "Building provider binary"
  cd "${projectdir}"
  
  # Build for the target platform
  GOOS=linux GOARCH=${SAFEHOSTARCH} CGO_ENABLED=0 go build -o bin/linux_${SAFEHOSTARCH}/provider cmd/provider/main.go
  
  echo_step "Building provider Docker image"
  docker build -f cluster/images/provider-garage/Dockerfile -t "xpkg.crossplane.io/${PACKAGE_NAME}:latest" .
  
  echo_step "Loading provider image into Kind cluster"
  "${KIND}" load docker-image "xpkg.crossplane.io/${PACKAGE_NAME}:latest" --name="${K8S_CLUSTER}"
}

setup_provider() {
  echo_step "Installing provider via Provider CRD"
  
  # Apply CRDs first
  echo_step "Applying CRDs"
  "${KUBECTL}" apply -f "${projectdir}/package/crds/"
  
  local yaml="$( cat <<EOF
apiVersion: pkg.crossplane.io/v1beta1
kind: DeploymentRuntimeConfig
metadata:
  name: debug-config
spec:
  deploymentTemplate:
    spec:
      selector: {}
      template:
        spec:
          containers:
            - name: package-runtime
              args:
                - --debug
---
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: "${PACKAGE_NAME}"
spec:
  runtimeConfigRef:
    name: debug-config
  package: "xpkg.crossplane.io/${PACKAGE_NAME}:latest"
  packagePullPolicy: Never
EOF
  )"
  
  echo "${yaml}" | "${KUBECTL}" apply -f -
  
  echo_step "Waiting for provider to be installed and healthy"
  "${KUBECTL}" wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" --for=condition=healthy --timeout=120s
  
  echo_step "Checking provider pod status"
  "${KUBECTL}" get pods -n crossplane-system -l pkg.crossplane.io/provider=${PACKAGE_NAME}
  
  echo_step "Provider logs"
  "${KUBECTL}" logs -n crossplane-system -l pkg.crossplane.io/provider=${PACKAGE_NAME} --tail=50 || true
}

cleanup_provider() {
  echo_step "Uninstalling provider"
  "${KUBECTL}" delete provider.pkg.crossplane.io "${PACKAGE_NAME}" --ignore-not-found=true
  "${KUBECTL}" delete deploymentruntimeconfig.pkg.crossplane.io debug-config --ignore-not-found=true
  
  echo_step "Waiting for provider pods to be deleted"
  timeout=60
  current=0
  step=3
  while [[ $(kubectl get providerrevision.pkg.crossplane.io -o name 2>/dev/null | wc -l | tr -d '[:space:]') != "0" ]]; do
    echo "Waiting another $step seconds..."
    current=$((current + step))
    if [[ $current -ge $timeout ]]; then
      echo_warn "Timeout of ${timeout}s reached, continuing anyway"
      break
    fi
    sleep $step
  done
}

setup_garage() {
  echo_step "Installing Garage v2.1.0"
  
  # Generate RPC secret
  local rpc_secret=$(openssl rand -hex 32)
  
  # Create Garage configuration
  cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: garage-config
  namespace: default
data:
  garage.toml: |
    metadata_dir = "/data/meta"
    data_dir = "/data/data"
    db_engine = "sqlite"
    replication_factor = 1
    
    [rpc]
    bind_addr = "[::]:3901"
    secret = "${rpc_secret}"
    
    [s3_api]
    s3_region = "garage"
    api_bind_addr = "[::]:3900"
    root_domain = ".s3.garage.localhost"
    
    [s3_web]
    bind_addr = "[::]:3902"
    root_domain = ".web.garage.localhost"
    
    [admin]
    api_bind_addr = "[::]:3903"
    admin_token = "${GARAGE_ADMIN_TOKEN}"
EOF

  # Deploy Garage StatefulSet
  cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: garage
  namespace: default
spec:
  serviceName: garage
  replicas: 1
  selector:
    matchLabels:
      app: garage
  template:
    metadata:
      labels:
        app: garage
    spec:
      containers:
      - name: garage
        image: dxflrs/garage:v2.1.0
        ports:
        - containerPort: 3900
          name: s3
        - containerPort: 3901
          name: rpc
        - containerPort: 3902
          name: web
        - containerPort: 3903
          name: admin
        volumeMounts:
        - name: config
          mountPath: /etc/garage.toml
          subPath: garage.toml
        - name: data
          mountPath: /data
      volumes:
      - name: config
        configMap:
          name: garage-config
      - name: data
        emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: garage
  namespace: default
spec:
  selector:
    app: garage
  ports:
  - name: s3
    port: 3900
    targetPort: 3900
  - name: rpc
    port: 3901
    targetPort: 3901
  - name: web
    port: 3902
    targetPort: 3902
  - name: admin
    port: 3903
    targetPort: 3903
EOF

  echo_step "Waiting for Garage to be ready"
  "${KUBECTL}" wait --for=create pod garage-0 --timeout=60s
  "${KUBECTL}" wait --for=condition=ready pod -l app=garage --timeout=180s
  
  echo_step "Configuring Garage cluster layout"
  sleep 5  # Give Garage a moment to fully initialize
  local node_id=$("${KUBECTL}" exec garage-0 -- /garage node id -q | cut -d@ -f1)
  if [ -z "$node_id" ]; then
    echo_error "Failed to get Garage node ID"
  fi
  echo_info "Garage node ID: $node_id"
  echo
  "${KUBECTL}" exec garage-0 -- /garage layout assign -z dc1 -c 1G "$node_id"
  "${KUBECTL}" exec garage-0 -- /garage layout apply --version 1
  
  echo_success "Garage v2.1.0 is ready!"
}

cleanup_garage() {
  echo_step "Uninstalling Garage"
  "${KUBECTL}" delete statefulset garage -n default --ignore-not-found=true
  "${KUBECTL}" delete service garage -n default --ignore-not-found=true
  "${KUBECTL}" delete configmap garage-config -n default --ignore-not-found=true
}

setup_provider_config() {
  echo_step "Creating ProviderConfig"
  
  # Create credentials secret
  "${KUBECTL}" create secret generic garage-creds -n crossplane-system \
    --from-literal=credentials="{\"endpoint\":\"http://garage.default.svc.cluster.local:3903\",\"adminToken\":\"${GARAGE_ADMIN_TOKEN}\"}"
  
  cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: garage.crossplane.io/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: garage-creds
      namespace: crossplane-system
      key: credentials
EOF
}

cleanup_provider_config() {
  echo_step "Cleaning up ProviderConfig"
  "${KUBECTL}" delete providerconfig.garage.crossplane.io default --ignore-not-found=true
  "${KUBECTL}" delete secret garage-creds -n crossplane-system --ignore-not-found=true
}

test_create_bucket() {
  echo_step "Test creating Bucket resource"
  
  cat <<EOF | "${KUBECTL}" apply -f -
apiVersion: garage.crossplane.io/v1alpha1
kind: Bucket
metadata:
  name: test-bucket
  namespace: default
spec:
  forProvider:
    globalAlias: integration-test-bucket
  providerConfigRef:
    name: default
EOF
  
  echo_info "Checking if bucket becomes ready"
  "${KUBECTL}" wait --timeout 2m --for condition=Ready bucket.garage.crossplane.io/test-bucket -n default
  echo_step_completed
  
  echo_info "Bucket status:"
  "${KUBECTL}" get bucket.garage.crossplane.io/test-bucket -n default -o yaml
}

cleanup_test_resources() {
  echo_step "Cleaning up test resources"
  "${KUBECTL}" delete bucket.garage.crossplane.io test-bucket -n default --ignore-not-found=true
}

run_integration_tests() {
  echo_step "=== STARTING INTEGRATION TESTS ==="
  
  setup_cluster
  setup_crossplane
  build_and_load_provider
  setup_provider
  setup_garage
  setup_provider_config
  
  echo_step "=== RUNNING TESTS ==="
  test_create_bucket
  
  echo_step "=== CLEANUP ==="
  cleanup_test_resources
  cleanup_provider_config
  cleanup_garage
  cleanup_provider
  
  echo_success "All integration tests passed!"
}

# Run the tests
run_integration_tests
