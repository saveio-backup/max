package asyncassertion_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"

	"testing"
)

func TestAsyncAssertion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AsyncAssertion Suite")
}
