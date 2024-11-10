#!/usr/bin/env bash

# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit
set -o nounset
set -o pipefail
set -x
export GO111MODULE=on

export SCRIPT_ROOT="$(cd "$(dirname $0)/../" && pwd -P)"
CODEGEN_PKG=${CODEGEN_PKG:-$(
    cd ${SCRIPT_ROOT}
    ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator
)}
OPENAPI_PKG=${OPENAPI_PKG:-$(
    cd ${SCRIPT_ROOT}
    ls -d -1 ./vendor/k8s.io/kube-openapi 2>/dev/null || echo ../kube-openapi
)}

(GOPROXY=off go install ${CODEGEN_PKG}/cmd/deepcopy-gen)
(GOPROXY=off go install ${CODEGEN_PKG}/cmd/client-gen)
(GOPROXY=off go install ${CODEGEN_PKG}/cmd/informer-gen)
(GOPROXY=off go install ${CODEGEN_PKG}/cmd/lister-gen)

find "${SCRIPT_ROOT}/pkg/" -name "*generated*.go" -exec rm {} -f \;
rm -rf "${SCRIPT_ROOT}/pkg/generated"

mkdir "${SCRIPT_ROOT}/pkg/generated"

mkdir "${SCRIPT_ROOT}/pkg/generated/kubevirt"
mkdir "${SCRIPT_ROOT}/pkg/generated/kubevirt/clientset"


client-gen \
	--clientset-name versioned \
	--input-base kubevirt.io/api \
    --output-dir "${SCRIPT_ROOT}/pkg/generated/kubevirt/clientset" \
	--output-pkg github.com/kubevirt/kubevirt-job/pkg/generated/kubevirt/clientset \
	--apply-configuration-package '' \
	--go-header-file "${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt" \
    --input core/v1