/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The Kubevirt Authors
 *
 */

package kubevirt_job

import (
	"context"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/kubevirt/kubevirt-job/pkg/client"
	fake2 "github.com/kubevirt/kubevirt-job/pkg/client-go/kubevirt/clientset/versioned/fake"
	util "github.com/kubevirt/kubevirt-job/pkg/kubevirt-job/env-manager"
	"github.com/kubevirt/kubevirt-job/pkg/libvmi"
	"k8s.io/client-go/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	virtv1 "kubevirt.io/api/core/v1"
)

const (
	machineTypeGlob        = "*glob8.*"
	machineTypeNeedsUpdate = "smth-glob8.10.0"
	machineTypeNoUpdate    = "smth-glob9.10.0"
)

var _ = Describe("MachineTypeUpdater", func() {
	var (
		ctrl               *gomock.Controller
		virtClient         *fake2.Clientset
		kubevirtJobClient  *client.MockKubevirtJobClient
		machineTypeUpdater *MachineTypeUpdater
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = fake2.NewSimpleClientset()
		kubevirtJobClient = client.NewMockKubevirtJobClient(ctrl)
		kubevirtJobClient.KubevirtClient()
		kubevirtJobClient.EXPECT().KubevirtClient().Return(virtClient)
		EnvVarManager = &util.EnvVarManagerImpl{}
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	When("there is no MACHINE_TYPE environment variable set", func() {
		It("should return an error", func() {
			_, err := NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("no machine type was specified")))
		})
	})

	When("MACHINE_TYPE environment variable is set", func() {
		const badGlob = "[--"

		It("should return an error in case of syntax error in pattern", func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, badGlob)
			Expect(err).ToNot(HaveOccurred())
			err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
			Expect(err).ToNot(HaveOccurred())
			_, err = NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("syntax error in pattern of %s environment variable, value \"%s\"", machineTypeEnvName, badGlob)))
		})

		When("glob is correct", func() {
			BeforeEach(func() {
				var err error
				err = EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater, err = NewMachineTypeUpdater(kubevirtJobClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(machineTypeUpdater.machineTypeGlob).To(BeEquivalentTo(machineTypeGlob))
			})

			DescribeTable("", func(machineType string, expectUpdate bool) {
				vmi := libvmi.New(
					libvmi.WithNamespace(v1.NamespaceDefault),
					withMachineType(machineType),
				)
				vm := libvmi.NewVirtualMachine(
					vmi,
				)
				_, err := virtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater.Run()
				Expect(virtClient.Actions()[0].GetVerb()).To(Equal("list"))
				Expect(virtClient.Actions()[0].GetResource().Resource).To(Equal("virtualmachines"))

				if expectUpdate {
					Expect(virtClient.Actions()).To(HaveLen(2))
					Expect(virtClient.Actions()[1].GetVerb()).To(Equal("patch"))
					Expect(virtClient.Actions()[1].GetResource().Resource).To(Equal("virtualmachines"))
				} else {
					Expect(virtClient.Actions()).To(HaveLen(1))
				}
			},
				Entry("should remove machineType if the vm machine type match", machineTypeNeedsUpdate, true),
				Entry("should not update machineType if the vm machine type does not match", machineTypeNoUpdate, false),
			)
		})
	})

	When("there is no NAMESPACE environment variable set", func() {
		BeforeEach(func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error", func() {
			_, err := NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fmt.Errorf("no namespace was specified")))
		})

	})

	When("NAMESPACE environment variable is set", func() {
		const badNamespaceName = "bad namespace pattern"

		It("should return an error in case of syntax error", func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
			Expect(err).ToNot(HaveOccurred())
			err = EnvVarManager.Setenv(namespaceEnvName, badNamespaceName)
			Expect(err).ToNot(HaveOccurred())
			_, err = NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("syntax error in %s environment variable, value \"%s\"", namespaceEnvName, badNamespaceName)))
		})

		When("it is correct", func() {
			const namespaceName = "filter-namespace"

			BeforeEach(func() {
				var err error
				err = EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(namespaceEnvName, namespaceName)
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater, err = NewMachineTypeUpdater(kubevirtJobClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(machineTypeUpdater.namespace).To(BeEquivalentTo(namespaceName))
			})

			It("should only return vm in that namespace", func() {
				vmi := libvmi.New(
					libvmi.WithNamespace(namespaceName),
					withMachineType(machineTypeNoUpdate),
				)
				vm := libvmi.NewVirtualMachine(
					vmi,
				)
				_, err := virtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater.Run()

				Expect(virtClient.Actions()).To(HaveLen(1))
				Expect(virtClient.Actions()[0].GetVerb()).To(Equal("list"))
				Expect(virtClient.Actions()[0].GetResource().Resource).To(Equal("virtualmachines"))
				Expect(virtClient.Actions()[0].GetNamespace()).To(Equal(namespaceName))

			})
		})
	})

	When("optional environment variables are not set", func() {
		BeforeEach(func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
			Expect(err).ToNot(HaveOccurred())
			err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should use the default values", func() {
			updater, err := NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).ToNot(HaveOccurred())
			Expect(updater.labelSelector).To(BeEquivalentTo(labels.Everything()))
			Expect(updater.restartRequired).To(BeFalse())
		})
	})

	When("RESTART_REQUIRED environment variable is set", func() {
		const badBoolean = "not_a_boolean"

		BeforeEach(func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
			Expect(err).ToNot(HaveOccurred())
			err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return an error in case of not boolean value", func() {
			EnvVarManager.Setenv(restartRequiredEnvName, badBoolean)
			_, err := NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("error parsing %s environment variable, value \"%s\"", restartRequiredEnvName, badBoolean)))
		})

		When("it is true", func() {
			BeforeEach(func() {
				var err error
				err = EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(restartRequiredEnvName, "true")
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater, err = NewMachineTypeUpdater(kubevirtJobClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(machineTypeUpdater.restartRequired).To(BeTrue())
			})

			DescribeTable("", func(running bool) {
				vmi := libvmi.New(
					libvmi.WithNamespace(v1.NamespaceDefault),
					withMachineType(machineTypeNeedsUpdate),
				)
				var opts []libvmi.VMOption
				if running {
					opts = []libvmi.VMOption{
						libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
						withPrintableStatus(virtv1.VirtualMachineStatusRunning),
					}
				}
				vm := libvmi.NewVirtualMachine(
					vmi,
					opts...,
				)
				_, err := virtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater.Run()

				Expect(virtClient.Actions()[0].GetVerb()).To(Equal("list"))
				Expect(virtClient.Actions()[0].GetResource().Resource).To(Equal("virtualmachines"))
				if running {
					Expect(virtClient.Actions()).To(HaveLen(2))
					Expect(virtClient.Actions()[1].GetVerb()).To(Equal("patch"))
					Expect(virtClient.Actions()[1].GetResource().Resource).To(Equal("virtualmachines"))
				} else {
					Expect(virtClient.Actions()).To(HaveLen(1))
				}
			},
				Entry("should restart running vm after the patch", true),
				Entry("should not restart non-running vm after the patch", false),
			)
		})
	})

	When("LABEL_SELECTOR environment variable is set", func() {
		const badLabelSelector = "non_a_valid for create error"

		BeforeEach(func() {
			err := EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
			Expect(err).To(HaveOccurred())
			err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
			Expect(err).To(HaveOccurred())
		})

		It("should return an error in case of parsing error", func() {
			err := EnvVarManager.Setenv(labelSelectorEnvName, badLabelSelector)
			Expect(err).To(HaveOccurred())
			_, err = NewMachineTypeUpdater(kubevirtJobClient)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("error parsing %s environment variable, value \"%s\"", labelSelectorEnvName, badLabelSelector)))
		})

		When("it is correct", func() {
			const labelSelector = "valid_label in (value1,value2)"

			BeforeEach(func() {
				var err error
				err = EnvVarManager.Setenv(machineTypeEnvName, machineTypeGlob)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(namespaceEnvName, v1.NamespaceDefault)
				Expect(err).ToNot(HaveOccurred())
				err = EnvVarManager.Setenv(labelSelectorEnvName, labelSelector)
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater, err = NewMachineTypeUpdater(kubevirtJobClient)
				Expect(err).ToNot(HaveOccurred())
				Expect(machineTypeUpdater.labelSelector.String()).To(BeEquivalentTo(labelSelector))
			})

			It("should only return vm that matches label selector", func() {
				vmi := libvmi.New(
					libvmi.WithNamespace(v1.NamespaceDefault),
					withMachineType(machineTypeNoUpdate),
				)
				vm := libvmi.NewVirtualMachine(
					vmi,
					withLabel("valid_label", "value1"),
				)
				_, err := virtClient.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, v1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				machineTypeUpdater.Run()

				Expect(virtClient.Actions()).To(HaveLen(1))
				Expect(virtClient.Actions()[0].GetVerb()).To(Equal("list"))
				Expect(virtClient.Actions()[0].GetResource().Resource).To(Equal("virtualmachines"))

				// Retrieve the label selector from the list action
				listAction, ok := virtClient.Actions()[0].(testing.ListAction)
				Expect(ok).To(BeTrue(), "Expected the action to be of type ListAction")

				// Extract the label selector from the action
				labelSelectorObj := listAction.GetListRestrictions().Labels

				// Parse the expected label selector from string to a labels.Selector
				expectedSelector, err := v1.ParseToLabelSelector(labelSelector)
				Expect(err).ToNot(HaveOccurred())

				parsedSelector, err := v1.LabelSelectorAsSelector(expectedSelector)
				Expect(err).ToNot(HaveOccurred())

				// Finally, compare the two selectors (string representations)
				Expect(labelSelectorObj.String()).To(Equal(parsedSelector.String()))

			})
		})
	})

})

func withMachineType(machineType string) libvmi.Option {
	return func(vmi *virtv1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Machine == nil {
			vmi.Spec.Domain.Machine = &virtv1.Machine{}
		}

		vmi.Spec.Domain.Machine.Type = machineType
	}
}

func withPrintableStatus(status virtv1.VirtualMachinePrintableStatus) libvmi.VMOption {
	return func(vm *virtv1.VirtualMachine) {
		vm.Status.PrintableStatus = status
	}
}

func withLabel(key, value string) libvmi.VMOption {
	return func(vm *virtv1.VirtualMachine) {
		if vm.Labels == nil {
			vm.Labels = map[string]string{}
		}
		vm.Labels[key] = value
	}
}
