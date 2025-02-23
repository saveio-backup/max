package integration_test

import (
	"os/exec"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

var _ = Describe("Interrupt", func() {
	var pathToTest string
	BeforeEach(func() {
		pathToTest = tmpPath("hanging")
		copyIn(fixturePath("hanging_suite"), pathToTest, false)
	})

	Context("when interrupting a suite", func() {
		var session *gexec.Session
		BeforeEach(func() {
			//we need to signal the actual process, so we must compile the test first
			var err error
			cmd := exec.Command("go", "test", "-c")
			cmd.Dir = pathToTest
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			//then run the compiled test directly
			cmd = exec.Command("./hanging.test", "--test.v=true", "--ginkgo.noColor")
			cmd.Dir = pathToTest
			session, err = gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Ω(err).ShouldNot(HaveOccurred())

			Eventually(session).Should(gbytes.Say("Sleeping..."))
			session.Interrupt()
			Eventually(session, 1000).Should(gexec.Exit(1))
		})

		It("should emit the contents of the GinkgoWriter", func() {
			Ω(session).Should(gbytes.Say("Just beginning"))
			Ω(session).Should(gbytes.Say("Almost there..."))
			Ω(session).Should(gbytes.Say("Hanging Out"))
		})

		It("should run the AfterSuite", func() {
			Ω(session).Should(gbytes.Say("Heading Out After Suite"))
		})
	})
})
