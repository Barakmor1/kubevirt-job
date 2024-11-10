package tests

import (
	"context"
	"fmt"
	"github.com/kubevirt/kubevirt-job/tests/framework"
	. "github.com/onsi/ginkgo/v2"
)

/*
Before you run these tests please make sure swap is on
*/
var _ = Describe("Wasp tests", func() {
	f := framework.NewFramework("wasp-test")
	Context("Wasp", func() {
		BeforeEach(func() {
			fmt.Printf(f.Namespace.Name)
		})
		It("Simple eviction test", func(ctx context.Context) {
		})
	})
})
