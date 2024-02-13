#!/bin/bash
set -euxo pipefail

if ! which kubectl >/dev/null; then
  mkdir -p /tmp/bin
  export PATH=/tmp/bin:${PATH}
  ln -s "$(which oc)" "/tmp/bin/kubectl"
  export NO_DOCKER=1
fi

NAMESPACE=openshift-multiarch-manager-operator
oc create namespace ${NAMESPACE}
oc annotate namespace ${NAMESPACE} \
  scheduler.alpha.kubernetes.io/node-selector="kubernetes.io/arch=amd64"

if [ "${USE_OLM:-}" == "true" ]; then
  export HOME=/tmp/home
  export XDG_RUNTIME_DIR=/tmp/home/containers
  OLD_KUBECONFIG=${KUBECONFIG}

  mkdir -p $XDG_RUNTIME_DIR
  unset KUBECONFIG
  oc registry login || sleepUntilUnlocked

  export KUBECONFIG="${OLD_KUBECONFIG}"
  operator-sdk run bundle "${OO_BUNDLE}" -n "${NAMESPACE}"
else
  make deploy IMG="${OPERATOR_IMAGE}"
fi

oc wait deployments -n ${NAMESPACE} \
  -l app.kubernetes.io/part-of=multiarch-manager-operator \
  --for=condition=Available=True
oc wait pods -n ${NAMESPACE} \
  -l control-plane=controller-manager \
  --for=condition=Ready=True

make e2e
