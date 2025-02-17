package integration_test

import (
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("Emitting progress", func() {
	var pathToTest string
	var session *gexec.Session
	var args []string

	BeforeEach(func() {
		args = []string{"--noColor"}
		pathToTest = tmpPath("progress")
		copyIn(fixturePath("progress_fixture"), pathToTest, false)
	})

	JustBeforeEach(func() {
		session = startGinkgo(pathToTest, args...)
		Eventually(session).Should(gexec.Exit(0))
	})

	Context("with the -progress flag, but no -v flag", func() {
		BeforeEach(func() {
			args = append(args, "-progress")
		})

		It("should not emit progress", func() {
			Ω(session).ShouldNot(gbytes.Say("[bB]efore"))
		})
	})

	Context("with the -v flag", func() {
		BeforeEach(func() {
			args = append(args, "-v")
		})

		It("should not emit progress", func() {
			Ω(session).ShouldNot(gbytes.Say(`\[BeforeEach\]`))
			Ω(session).Should(gbytes.Say(`>outer before<`))
		})
	})

	Context("with the -progress flag and the -v flag", func() {
		BeforeEach(func() {
			args = append(args, "-progress", "-v")
		})

		It("should emit progress (by writing to the GinkgoWriter)", func() {
			// First spec

			Ω(session).Should(gbytes.Say(`\[BeforeEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer before<`))

			Ω(session).Should(gbytes.Say(`\[BeforeEach\] Inner Context`))
			Ω(session).Should(gbytes.Say(`>inner before<`))

			Ω(session).Should(gbytes.Say(`\[BeforeEach\] when Inner When`))
			Ω(session).Should(gbytes.Say(`>inner before<`))

			Ω(session).Should(gbytes.Say(`\[JustBeforeEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer just before<`))

			Ω(session).Should(gbytes.Say(`\[JustBeforeEach\] Inner Context`))
			Ω(session).Should(gbytes.Say(`>inner just before<`))

			Ω(session).Should(gbytes.Say(`\[It\] should emit progress as it goes`))
			Ω(session).Should(gbytes.Say(`>it<`))

			Ω(session).Should(gbytes.Say(`\[AfterEach\] Inner Context`))
			Ω(session).Should(gbytes.Say(`>inner after<`))

			Ω(session).Should(gbytes.Say(`\[AfterEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer after<`))

			// Second spec

			Ω(session).Should(gbytes.Say(`\[BeforeEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer before<`))

			Ω(session).Should(gbytes.Say(`\[JustBeforeEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer just before<`))

			Ω(session).Should(gbytes.Say(`\[It\] should emit progress as it goes`))
			Ω(session).Should(gbytes.Say(`>specify<`))

			Ω(session).Should(gbytes.Say(`\[AfterEach\] ProgressFixture`))
			Ω(session).Should(gbytes.Say(`>outer after<`))
		})
	})
})
