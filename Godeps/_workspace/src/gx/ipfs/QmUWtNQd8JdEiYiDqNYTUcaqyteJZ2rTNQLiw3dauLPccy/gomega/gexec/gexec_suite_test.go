package gexec_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/gexec"

	"testing"
)

var fireflyPath string

func TestGexec(t *testing.T) {
	BeforeSuite(func() {
		var err error
		fireflyPath, err = gexec.Build("./_fixture/firefly")
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Gexec Suite")
}
