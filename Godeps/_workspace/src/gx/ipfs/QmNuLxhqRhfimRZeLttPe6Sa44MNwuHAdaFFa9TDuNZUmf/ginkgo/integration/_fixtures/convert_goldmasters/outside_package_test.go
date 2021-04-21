package tmp_test

import (
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("something important", func() {

		whatever := &UselessStruct{}
		if whatever.ImportantField != "SECRET_PASSWORD" {
			GinkgoT().Fail()
		}
	})
})

type UselessStruct struct {
	ImportantField string
}
