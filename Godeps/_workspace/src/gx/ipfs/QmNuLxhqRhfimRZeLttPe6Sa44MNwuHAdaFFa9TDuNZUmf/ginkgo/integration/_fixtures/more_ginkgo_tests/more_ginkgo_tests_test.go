package more_ginkgo_tests_test

import (
	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/integration/_fixtures/more_ginkgo_tests"
)

var _ = Describe("MoreGinkgoTests", func() {
	It("should pass", func() {
		Ω(AlwaysTrue()).Should(BeTrue())
	})

	It("should always pass", func() {
		Ω(AlwaysTrue()).Should(BeTrue())
	})
})
