package nested_test

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
)

func TestNested(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nested Suite")
}
