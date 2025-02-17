package test_description_test

import (
	"fmt"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("TestDescription", func() {
	It("should pass", func() {
		Ω(true).Should(BeTrue())
	})

	It("should fail", func() {
		Ω(true).Should(BeFalse())
	})

	AfterEach(func() {
		description := CurrentGinkgoTestDescription()
		fmt.Printf("%s:%t\n", description.FullTestText, description.Failed)
	})
})
