#!/bin/bash -e

echo kubevirt-job

source ./hack/build/config.sh
source ./hack/build/common.sh
source ./cluster-up/hack/common.sh
source ./cluster-up/cluster/${KUBEVIRT_PROVIDER}/provider.sh

if [ "${KUBEVIRT_PROVIDER}" = "external" ]; then
   KUBEVIRT_JOB_SYNC_PROVIDER="external"
else
   KUBEVIRT_JOB_SYNC_PROVIDER="kubevirtci"
fi
source ./cluster-sync/${KUBEVIRT_JOB_SYNC_PROVIDER}/provider.sh


KUBEVIRT_JOB_NAMESPACE=${KUBEVIRT_JOB_NAMESPACE:-kubevirt-job}
KUBEVIRT_JOB_INSTALL_TIMEOUT=${KUBEVIRT_JOB_INSTALL_TIMEOUT:-120}
KUBEVIRT_JOB_AVAILABLE_TIMEOUT=${KUBEVIRT_JOB_AVAILABLE_TIMEOUT:-600}

# Set controller verbosity to 3 for functional tests.
export VERBOSITY=3

PULL_POLICY=${PULL_POLICY:-IfNotPresent}
# The default DOCKER_PREFIX is set to kubevirt and used for builds, however we don't use that for cluster-sync
# instead we use a local registry; so here we'll check for anything != "external"
# wel also confuse this by swapping the setting of the DOCKER_PREFIX variable around based on it's context, for
# build and push it's localhost, but for manifests, we sneak in a change to point a registry container on the
# kubernetes cluster.  So, we introduced this MANIFEST_REGISTRY variable specifically to deal with that and not
# have to refactor/rewrite any of the code that works currently.
MANIFEST_REGISTRY=$DOCKER_PREFIX

if [ "${KUBEVIRT_PROVIDER}" != "external" ]; then
  registry=${IMAGE_REGISTRY:-localhost:$(_port registry)}
  DOCKER_PREFIX=${registry}
  MANIFEST_REGISTRY="registry:5000"
fi

if [ "${KUBEVIRT_PROVIDER}" == "external" ]; then
  # No kubevirtci local registry, likely using something external
  if [[ $(${KUBEVIRT_JOB_CRI} login --help | grep authfile) ]]; then
    registry_provider=$(echo "$DOCKER_PREFIX" | cut -d '/' -f 1)
    echo "Please log in to "${registry_provider}", bazel push expects external registry creds to be in ~/.docker/config.json"
    ${KUBEVIRT_JOB_CRI} login --authfile "${HOME}/.docker/config.json" $registry_provider
  fi
fi

# Need to set the DOCKER_PREFIX appropriately in the call to `make docker push`, otherwise make will just pass in the default `kubevirt`

DOCKER_PREFIX=$MANIFEST_REGISTRY PULL_POLICY=$PULL_POLICY make manifests
DOCKER_PREFIX=$DOCKER_PREFIX make push

function check_kubevirt_job_exists() {
  # Check if the kubevirt-job Job exists in the specified namespace
  kubectl get job kubevirt-job -n "$KUBEVIRT_JOB_NAMESPACE" &> /dev/null

  if [ $? -eq 0 ]; then
    echo "Job kubevirt-job exists in namespace $KUBEVIRT_JOB_NAMESPACE."
    return 0
  else
    echo "Job kubevirt-job does not exist in namespace $KUBEVIRT_JOB_NAMESPACE."
    return 1
  fi
}

function wait_kubevirt_job_available {
  retry_count="${KUBEVIRT_JOB_INSTALL_TIMEOUT}"
  echo "Waiting for kubevirt-job Job in namespace '$KUBEVIRT_JOB_NAMESPACE' to be ready..."

  # Loop for the specified number of retries
  for ((i = 0; i < retry_count; i++)); do
    # Check if DaemonSet pods are ready
    if check_kubevirt_job_exists ; then
      echo "Job kubevirt-job is available."
      exit 0
    fi

    # Wait for 1 second before retrying
    sleep 1
  done
    echo "Warning: kubevirt-job doesn't exist!"
}

mkdir -p ./_out/tests

# Install KUBEVIRT JOB
install_kubevirt_job

wait_kubevirt_job_available
