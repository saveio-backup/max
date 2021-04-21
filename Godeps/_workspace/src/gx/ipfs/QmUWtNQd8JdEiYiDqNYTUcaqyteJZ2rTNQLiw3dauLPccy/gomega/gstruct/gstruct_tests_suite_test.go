package gstruct_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"
)

func Test(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gstruct Suite")
}
