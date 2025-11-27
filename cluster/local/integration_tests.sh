#!/usr/bin/env bash
set -e

# setting up colors
BLU='\033[0;34m'
YLW='\033[0;33m'
GRN='\033[0;32m'
RED='\033[0;31m'
NOC='\033[0m' # No Color

echo_info(){
    printf "\n${BLU}%s${NOC}" "$1"
}
echo_step(){
    printf "\n${BLU}>>>>>>> %s${NOC}\n" "$1"
}
echo_sub_step(){
    printf "\n${BLU}>>> %s${NOC}\n" "$1"
}

echo_step_completed(){
    printf "${GRN} [✔]${NOC}"
}

echo_success(){
    printf "\n${GRN}%s${NOC}\n" "$1"
}
echo_warn(){
    printf "\n${YLW}%s${NOC}" "$1"
}
echo_error(){
    printf "\n${RED}%s${NOC}" "$1"
    exit 1
}


# The name of your provider
PACKAGE_NAME="provider-garage"


# ------------------------------
projectdir="$( cd "$( dirname "${BASH_SOURCE[0]}")"/../.. && pwd )"

# get the build environment variables from the special build.vars target in the main makefile
eval $(make --no-print-directory -C ${projectdir} build.vars)

# ------------------------------

SAFEHOSTARCH="${SAFEHOSTARCH:-amd64}"
BUILD_IMAGE="${BUILD_REGISTRY}/${PROJECT_NAME}-${SAFEHOSTARCH}"
PACKAGE_IMAGE="crossplane.io/inttests/${PROJECT_NAME}:${VERSION}"
# The controller image is the same as the build image (no -controller- suffix)
CONTROLLER_IMAGE="${BUILD_REGISTRY}/${PROJECT_NAME}-${SAFEHOSTARCH}"

version_tag="$(cat ${projectdir}/_output/version)"
# tag as latest version to load into kind cluster
# Use BUILD_REGISTRY since DOCKER_REGISTRY may be empty
PACKAGE_CONTROLLER_IMAGE="${BUILD_REGISTRY}/${PROJECT_NAME}-controller:${VERSION}"
K8S_CLUSTER="${K8S_CLUSTER:-${BUILD_REGISTRY}-inttests}"

CROSSPLANE_NAMESPACE="crossplane-system"

# cleanup on exit
if [ "$skipcleanup" != true ]; then
  function cleanup {
    echo_step "Cleaning up..."
    export KUBECONFIG=
    "${KIND}" delete cluster --name="${K8S_CLUSTER}"
  }

  trap cleanup EXIT
fi

# setup package cache
echo_step "setting up local package cache"
CACHE_PATH="${projectdir}/.work/inttest-package-cache"
mkdir -p "${CACHE_PATH}"
echo "created cache dir at ${CACHE_PATH}"

# Find the xpkg file from the build output
XPKG_FILE=$(find "${projectdir}/_output/xpkg" -name "${PACKAGE_NAME}-*.xpkg" 2>/dev/null | head -1)
if [ -n "${XPKG_FILE}" ] && [ -f "${XPKG_FILE}" ]; then
  echo "Using pre-built xpkg: ${XPKG_FILE}"
  cp "${XPKG_FILE}" "${CACHE_PATH}/${PACKAGE_NAME}.gz"
  chmod 644 "${CACHE_PATH}/${PACKAGE_NAME}.gz"
else
  # Fallback: try to extract from docker image (legacy approach)
  echo "No xpkg file found, trying to extract from docker image..."
  docker tag "${BUILD_IMAGE}" "${PACKAGE_IMAGE}"
  "${CROSSPLANE_CLI}" xpkg build --package-root "${projectdir}/package" --embed-runtime-image "${BUILD_IMAGE}" -o "${CACHE_PATH}/${PACKAGE_NAME}.gz" && chmod 644 "${CACHE_PATH}/${PACKAGE_NAME}.gz"
fi

# Determine the KIND node image tag
# Use a sensible default if KIND_NODE_IMAGE_TAG is not set
if [ -z "${KIND_NODE_IMAGE_TAG}" ]; then
  # Get the latest supported k8s version for the current kind version
  KIND_NODE_IMAGE_TAG="v1.32.2"
  echo "KIND_NODE_IMAGE_TAG not set, using default: ${KIND_NODE_IMAGE_TAG}"
fi

# create kind cluster with extra mounts
KIND_NODE_IMAGE="kindest/node:${KIND_NODE_IMAGE_TAG}"
echo_step "creating k8s cluster using kind ${KIND_VERSION} and node image ${KIND_NODE_IMAGE}"
KIND_CONFIG="$( cat <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  extraMounts:
  - hostPath: "${CACHE_PATH}/"
    containerPath: /cache
EOF
)"
echo "${KIND_CONFIG}" | "${KIND}" create cluster --name="${K8S_CLUSTER}" --wait=5m --image="${KIND_NODE_IMAGE}" --config=-

# tag controller image and load it into kind cluster
docker tag "${CONTROLLER_IMAGE}" "${PACKAGE_CONTROLLER_IMAGE}"
"${KIND}" load docker-image "${PACKAGE_CONTROLLER_IMAGE}" --name="${K8S_CLUSTER}"

echo_step "create crossplane-system namespace"
"${KUBECTL}" create ns crossplane-system

echo_step "create persistent volume and claim for mounting package-cache"
PV_YAML="$( cat <<EOF
apiVersion: v1
kind: PersistentVolume
metadata:
  name: package-cache
  labels:
    type: local
spec:
  storageClassName: manual
  capacity:
    storage: 5Mi
  accessModes:
    - ReadWriteOnce
  hostPath:
    path: "/cache"
EOF
)"
echo "${PV_YAML}" | "${KUBECTL}" create -f -

PVC_YAML="$( cat <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: package-cache
  namespace: crossplane-system
spec:
  accessModes:
    - ReadWriteOnce
  volumeName: package-cache
  storageClassName: manual
  resources:
    requests:
      storage: 1Mi
EOF
)"
echo "${PVC_YAML}" | "${KUBECTL}" create -f -

# install crossplane from stable channel
echo_step "installing crossplane from stable channel"
"${HELM}" repo add crossplane-stable https://charts.crossplane.io/stable/
chart_version="$("${HELM}" search repo crossplane-stable/crossplane | awk 'FNR == 2 {print $2}')"
echo_info "using crossplane version ${chart_version}"
echo
# we replace empty dir with our PVC so that the /cache dir in the kind node
# container is exposed to the crossplane pod
"${HELM}" install crossplane --namespace crossplane-system crossplane-stable/crossplane --version ${chart_version} --wait --set packageCache.pvc=package-cache

# ----------- integration tests
echo_step "--- INTEGRATION TESTS ---"

# install package
echo_step "installing ${PROJECT_NAME} into \"${CROSSPLANE_NAMESPACE}\" namespace"

# For Crossplane 2.x, we need a fully qualified OCI image reference
# Use a fake local registry format that satisfies the validation
XPKG_IMAGE="local.xpkg/${PACKAGE_NAME}:${VERSION}"

# Check if we can load the xpkg as a docker image
if [ -f "${CACHE_PATH}/${PACKAGE_NAME}.gz" ]; then
  echo "Loading xpkg from ${CACHE_PATH}/${PACKAGE_NAME}.gz"
  # The xpkg is an OCI tarball, load it and tag appropriately
  gunzip -c "${CACHE_PATH}/${PACKAGE_NAME}.gz" > "${CACHE_PATH}/${PACKAGE_NAME}.tar" 2>/dev/null || cp "${CACHE_PATH}/${PACKAGE_NAME}.gz" "${CACHE_PATH}/${PACKAGE_NAME}.tar"
  docker load -i "${CACHE_PATH}/${PACKAGE_NAME}.tar" 2>/dev/null || true
  
  # Find the loaded image and tag it with our expected name
  LOADED_IMAGE=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep "${PACKAGE_NAME}" | head -1)
  if [ -n "${LOADED_IMAGE}" ]; then
    echo "Tagging ${LOADED_IMAGE} as ${XPKG_IMAGE}"
    docker tag "${LOADED_IMAGE}" "${XPKG_IMAGE}"
  fi
fi

# Verify the image exists before loading into kind
echo "Verifying ${XPKG_IMAGE} exists in docker..."
docker images "${XPKG_IMAGE}"

# Load the xpkg image into kind
echo "Loading ${XPKG_IMAGE} into kind cluster ${K8S_CLUSTER}..."
"${KIND}" load docker-image "${XPKG_IMAGE}" --name="${K8S_CLUSTER}"

# Verify the image is available in the kind node
echo "Verifying image is loaded in kind node..."
docker exec "${K8S_CLUSTER}-control-plane" crictl images | grep -E "${PACKAGE_NAME}|local.xpkg" || echo "Warning: Image may not be visible via crictl"

INSTALL_YAML="$( cat <<EOF
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: "${PACKAGE_NAME}"
spec:
  package: "${XPKG_IMAGE}"
  packagePullPolicy: Never
EOF
)"

echo "${INSTALL_YAML}" | "${KUBECTL}" apply -f -

# printing the cache dir contents can be useful for troubleshooting failures
echo_step "check kind node cache dir contents"
docker exec "${K8S_CLUSTER}-control-plane" ls -la /cache

echo_step "waiting for provider to be installed"

# Wait a few seconds for the provider to start reconciling
sleep 5

# Check provider status for debugging
echo "Provider status:"
"${KUBECTL}" get provider "${PACKAGE_NAME}" -o yaml || true
echo "Provider revision status:"
"${KUBECTL}" get providerrevision -o yaml || true
echo "Crossplane pods:"
"${KUBECTL}" get pods -n crossplane-system || true
echo "Crossplane pod logs:"
"${KUBECTL}" logs -n crossplane-system -l app=crossplane --tail=50 || true

kubectl wait "provider.pkg.crossplane.io/${PACKAGE_NAME}" --for=condition=healthy --timeout=180s

echo_step "uninstalling ${PROJECT_NAME}"

echo "${INSTALL_YAML}" | "${KUBECTL}" delete -f -

# check pods deleted
timeout=60
current=0
step=3
while [[ $(kubectl get providerrevision.pkg.crossplane.io -o name | wc -l) != "0" ]]; do
  echo "waiting for provider to be deleted for another $step seconds"
  current=$current+$step
  if ! [[ $timeout > $current ]]; then
    echo_error "timeout of ${timeout}s has been reached"
  fi
  sleep $step;
done

echo_success "Integration tests succeeded!"
