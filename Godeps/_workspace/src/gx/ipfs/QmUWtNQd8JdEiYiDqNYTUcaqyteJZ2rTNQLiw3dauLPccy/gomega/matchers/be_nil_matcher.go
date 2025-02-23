package matchers

import "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/format"

type BeNilMatcher struct {
}

func (matcher *BeNilMatcher) Match(actual interface{}) (success bool, err error) {
	return isNil(actual), nil
}

func (matcher *BeNilMatcher) FailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "to be nil")
}

func (matcher *BeNilMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return format.Message(actual, "not to be nil")
}
