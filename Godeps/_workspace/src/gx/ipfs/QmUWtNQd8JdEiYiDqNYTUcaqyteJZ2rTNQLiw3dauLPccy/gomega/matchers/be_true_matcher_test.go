package matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/matchers"
)

var _ = Describe("BeTrue", func() {
	It("should handle true and false correctly", func() {
		Expect(true).Should(BeTrue())
		Expect(false).ShouldNot(BeTrue())
	})

	It("should only support booleans", func() {
		success, err := (&BeTrueMatcher{}).Match("foo")
		Expect(success).Should(BeFalse())
		Expect(err).Should(HaveOccurred())
	})
})
