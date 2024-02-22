# Multiarch Manager Operator

The Multiarch Manager Operator (MMO) enhances the user experience for administrators of Openshift clusters with
multi-architecture compute nodes or Site Reliability Engineers willing to migrate from single-arch to multi-arch
OpenShift. When diverse CPU architectures coexist within a cluster, the MMO operator stands out as a pivotal tool to
enhance efficiency and streamline operations such as architecture-aware scheduling of workloads.

The development work is still ongoing and there is no official, general available, release of it yet.

## Description

- **Architecture aware Pod Placement**: The pod placement operand aims to automate the 
  inspection of the container images, derive a set of architectures supported by a pod and use it to
  automatically define strong predicates based on the `kubernetes.io/arch` label in the pod's nodeAffinity. 
  This operand is based on the [KEP-3521](https://github.com/kubernetes/enhancements/blob/afad6f270c7ac2ae853f4d1b72c379a6c3c7c042/keps/sig-scheduling/3521-pod-scheduling-readiness/README.md) and
  [KEP-3838](https://github.com/kubernetes/enhancements/blob/afad6f270c7ac2ae853f4d1b72c379a6c3c7c042/keps/sig-scheduling/3838-pod-mutable-scheduling-directives/README.md), as
  described in the [Openshift EP](https://github.com/openshift/enhancements/blob/6cebc13f0672c601ebfae669ea4fc8ca632721b5/enhancements/multi-arch/multiarch-manager-operator.md) introducing it.
  When a pod is created, the mutating webhook will add the `multiarch.openshift.io/scheduling-gate` scheduling gate, that will
  prevent the pod from being scheduled until the controller computes a predicate for the `kubernetes.io/arch` label,
  adds it as node affinity requirement to the pod spec and removes the scheduling gate.

## Getting Started

The aim of this operator will be to run on any Kubernetes cluster, although the main focus of development and testing
will be carried out on Openshift clusters.


### Development

### Build the operator

```shell
# Multi-arch image build
make docker-buildx IMG=<some-registry>/multiarch-manager-operator:tag

# Single arch image build
make docker-build IMG=<some-registry>/multiarch-manager-operator:tag

# Local build
make build
```

If you aim to use the multi-arch build and would avoid the deletion of the buildx instance, you can
create an empty `.persistent-buildx` file in the root of the repository.

```shell
touch .persistent-buildx
make docker-buildx IMG=<some-registry>/multiarch-manager-operator:tag
```

### Deploy the operator

```shell
# Deploy the operator on the cluster
make deploy IMG=<some-registry>/multiarch-manager-operator:tag
```

### Deploy the Pod Placement Operand

After the operator is running, you can deploy the Pod Placement Operand on the cluster through the PodPlacementConfig CR.
It is expected to be a singleton, cluster-scoped resource, named `cluster`.

The following is an example of a PodPlacementConfig CR that sets the log verbosity level to `Normal` and 
will watch and setup CPU architecture node affinities on all the pods, except the ones in namespaces labeled with 
`multiarch.openshift.io/exclude-pod-placement`.

**Note**: the namespaces `openshift-*`, `kube-*1`, and `hypershift-*` are excluded by default and cannot be included back for
safety reasons.

```shell
# Deploy the pod placement operand on the cluster
kubectl create -f - <<EOF
apiVersion: multiarch.openshift.io/v1alpha1
kind: PodPlacementConfig
metadata:
  name: cluster
spec:
  logVerbosityLevel: Normal
  namespaceSelector:
    matchExpressions:
      - key: multiarch.openshift.io/exclude-pod-placement
        operator: DoesNotExist
EOF
```

### Undeploy the PodPlacementConfig operand

```shell
kubectl delete podplacementconfig/cluster
```

Ordered uninstallation of the operand will be implemented in the future, and will remove the pod placement controller
only after all the scheduling gated pods have been ungated.

To overcome this limitation currently, you can execute the following to ensure the deletion of the scheduling gate 
from all the pods:
```shell
kubectl get pods -A -l multiarch.openshift.io/scheduling-gate=gated -o json  | jq 'del(.items[].spec.schedulingGates[] | select(.name=="multiarch.openshift.io/scheduling-gate"))' | kubectl apply -f -
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller

UnDeploy the controller from the cluster:

```sh
make undeploy
```

### Modifying the API definitions

If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

### Running tests

The following units are available as Makefile targets:

```shell
# Lint
make lint
# gosec (SAST)
make gosec
# vet
make vet
# goimports
make goimports
# gofmt
make fmt
# Run unit tests
make test
# Run e2e tests (after the operator is deployed)
make e2e KUBECONFIG=/path/to/cluster/kubeconfig NAMESPACE=openshift-multiarch-manager-operator
```

All the checks run on a containerized environment by default. 
You can run them locally by setting the `NO_DOCKER` variable to `1`:

```shell
NO_DOCKER=1 make test
```

or adding the `NO_DOCKER=1` row in the `.env` file.

See the [dotenv.example](./dotenv.example) file for other available settings.

### How it works

See [Openshift Enhancement Proposal](https://github.com/openshift/enhancements/blob/6cebc13f0672c601ebfae669ea4fc8ca632721b5/enhancements/multi-arch/multiarch-manager-operator.md).


## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

## License

Copyright 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

