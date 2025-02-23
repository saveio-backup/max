package suite_test

import (
	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"

	"testing"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Suite")
}

var numBeforeSuiteRuns = 0
var numAfterSuiteRuns = 0

var _ = BeforeSuite(func() {
	numBeforeSuiteRuns++
})

var _ = AfterSuite(func() {
	numAfterSuiteRuns++
	Ω(numBeforeSuiteRuns).Should(Equal(1))
	Ω(numAfterSuiteRuns).Should(Equal(1))
})

//Fakes
type fakeTestingT struct {
	didFail bool
}

func (fakeT *fakeTestingT) Fail() {
	fakeT.didFail = true
}
