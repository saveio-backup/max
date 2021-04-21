package tmp_test

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

func TestConvertFixtures(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ConvertFixtures Suite")
}
