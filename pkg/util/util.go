package util

import (
	utils "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/resources"
)

const (
	// KubevirtJobLabel is the labe applied to all non operator resources
	KubevirtJobLabel = "kubevirt-job.io"
	// AppKubernetesManagedByLabel is the Kubernetes recommended managed-by label
	AppKubernetesManagedByLabel = "app.kubernetes.io/managed-by"
	// AppKubernetesComponentLabel is the Kubernetes recommended component label
	AppKubernetesComponentLabel = "app.kubernetes.io/component"
	KubevirtJobResourceName     = "kubevirt-job"
)

var commonLabels = map[string]string{
	KubevirtJobLabel:            "",
	AppKubernetesManagedByLabel: "kubevirt-job",
	AppKubernetesComponentLabel: "virtualization",
}

var JobLabels = map[string]string{
	"kubevirt-job.io": "",
	"tier":            "node",
}

// ResourceBuilder helps in creating k8s resources
var ResourceBuilder = utils.NewResourceBuilder(commonLabels, JobLabels)
