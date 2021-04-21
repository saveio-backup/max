package nested

import (
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("something less important", func() {

		whatever := &UselessStruct{}
		GinkgoT().Fail(whatever.ImportantField != "SECRET_PASSWORD")
	})
})
