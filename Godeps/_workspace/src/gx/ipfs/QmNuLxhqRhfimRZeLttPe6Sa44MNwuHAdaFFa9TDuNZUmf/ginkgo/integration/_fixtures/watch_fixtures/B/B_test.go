package B_test

import (
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/integration/_fixtures/watch_fixtures/B"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("B", func() {
	It("should do it", func() {
		Ω(DoIt()).Should(Equal("done!"))
	})
})
