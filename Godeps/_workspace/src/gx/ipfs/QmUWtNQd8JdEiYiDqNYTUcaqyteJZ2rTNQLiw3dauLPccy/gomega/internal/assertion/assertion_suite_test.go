package assertion_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega"

	"testing"
)

func TestAssertion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Assertion Suite")
}
