package matchers_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/matchers"
)

var _ = Describe("BeAnExistingFileMatcher", func() {
	Context("when passed a string", func() {
		It("should do the right thing", func() {
			Expect("/dne/test").ShouldNot(BeAnExistingFile())

			tmpFile, err := ioutil.TempFile("", "gomega-test-tempfile")
			Expect(err).ShouldNot(HaveOccurred())
			defer os.Remove(tmpFile.Name())
			Expect(tmpFile.Name()).Should(BeAnExistingFile())

			tmpDir, err := ioutil.TempDir("", "gomega-test-tempdir")
			Expect(err).ShouldNot(HaveOccurred())
			defer os.Remove(tmpDir)
			Expect(tmpDir).Should(BeAnExistingFile())
		})
	})

	Context("when passed something else", func() {
		It("should error", func() {
			success, err := (&BeAnExistingFileMatcher{}).Match(nil)
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())

			success, err = (&BeAnExistingFileMatcher{}).Match(true)
			Expect(success).Should(BeFalse())
			Expect(err).Should(HaveOccurred())
		})
	})
})
