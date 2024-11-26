#Copyright 2023 The KubevirtJob Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

.PHONY: manifests \
		cluster-up cluster-down cluster-sync \
		test test-functional test-unit test-lint \
		publish \
		kubevirt_job \
		fmt \
		goveralls \
		release-description \
		bazel-build-images push-images \
		fossa
all: build

build:  kubevirt_job manifest-generator

KUBEVIRT_RELEASE ?= latest_nightly

all: manifests build-images

manifests:
	hack/build/bazel-docker.sh "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} VERBOSITY=${VERBOSITY} PULL_POLICY=${PULL_POLICY} CR_NAME=${CR_NAME} KUBEVIRT_JOB_NAMESPACE=${KUBEVIRT_JOB_NAMESPACE} TARGET_NAMESPACE=${TARGET_NAMESPACE} RESTART_REQUIRED=${RESTART_REQUIRED} MACHINE_TYPE_GLOBAL=${MACHINE_TYPE_GLOBAL} LABEL_SELECTOR=${LABEL_SELECTOR} ./hack/build/build-manifests.sh"

builder-push:
	./hack/build/build-builder.sh

generate:
	hack/build/bazel-docker.sh "./hack/update-codegen.sh"

cluster-up:
	eval "KUBEVIRT_RELEASE=${KUBEVIRT_RELEASE} ./cluster-up/up.sh"

cluster-down:
	./cluster-up/down.sh

push-images:
	eval "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG}  ./hack/build/build-docker.sh push"

build-images:
	eval "DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG}  ./hack/build/build-docker.sh"

push: build-images push-images

cluster-clean-kubevirt-job:
	./cluster-sync/clean.sh

cluster-sync: cluster-clean-kubevirt-job
	./cluster-sync/sync.sh KUBEVIRT_JOB_AVAILABLE_TIMEOUT=${KUBEVIRT_JOB_AVAILABLE_TIMEOUT} DOCKER_PREFIX=${DOCKER_PREFIX} DOCKER_TAG=${DOCKER_TAG} PULL_POLICY=${PULL_POLICY} kubevirt_job_NAMESPACE=${kubevirt_job_NAMESPACE}

test: WHAT = ./pkg/... ./cmd/...
test: bootstrap-ginkgo
	hack/build/bazel-docker.sh "ACK_GINKGO_DEPRECATIONS=${ACK_GINKGO_DEPRECATIONS} ./hack/build/run-unit-tests.sh ${WHAT}"

build-functest:
	hack/build/bazel-docker.sh ./hack/build/build-functest.sh

functest:  WHAT = ./tests/...
functest: build-functest
	./hack/build/run-functional-tests.sh ${WHAT} "${TEST_ARGS}"

bootstrap-ginkgo:
	hack/build/bazel-docker.sh ./hack/build/bootstrap-ginkgo.sh

manifest-generator:
	GO111MODULE=${GO111MODULE:-off} go build -o manifest-generator -v tools/manifest-generator/*.go
kubevirt-job:
	go build -o kubevirt_job -v cmd/kubevirt-job/*.go
	chmod 777 kubevirt_job

release-description:
	./hack/build/release-description.sh ${RELREF} ${PREREF}

clean:
	rm ./kubevirt_job -f

fmt:
	go fmt .

run: build
	sudo ./kubevirt_job
