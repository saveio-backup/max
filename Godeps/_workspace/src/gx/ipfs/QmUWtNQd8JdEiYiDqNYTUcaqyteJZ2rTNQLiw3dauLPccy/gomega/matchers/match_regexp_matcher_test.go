package matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/matchers"
)

var _ = Describe("MatchRegexp", func() {
	Context("when actual is a string", func() {
		It("should match against the string", func() {
			Expect(" a2!bla").Should(MatchRegexp(`\d!`))
			Expect(" a2!bla").ShouldNot(MatchRegexp(`[A-Z]`))
		})
	})

	Context("when actual is a stringer", func() {
		It("should call the stringer and match agains the returned string", func() {
			Expect(&myStringer{a: "Abc3"}).Should(MatchRegexp(`[A-Z][a-z]+\d`))
		})
	})

	Context("when the matcher is called with multiple arguments", func() {
		It("should pass the string and arguments to sprintf", func() {
			Expect(" a23!bla").Should(MatchRegexp(`\d%d!`, 3))
		})
	})

	Context("when actual is neither a string nor a stringer", func() {
		It("should error", func() {
			success, err := (&MatchRegexpMatcher{Regexp: `\d`}).Match(2)
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())
		})
	})

	Context("when the passed in regexp fails to compile", func() {
		It("should error", func() {
			success, err := (&MatchRegexpMatcher{Regexp: "("}).Match("Foo")
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())
		})
	})
})
