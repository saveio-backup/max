package assertion

import (
	"fmt"
	"reflect"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmUWtNQd8JdEiYiDqNYTUcaqyteJZ2rTNQLiw3dauLPccy/gomega/types"
)

type Assertion struct {
	actualInput interface{}
	fail        types.GomegaFailHandler
	offset      int
	extra       []interface{}
}

func New(actualInput interface{}, fail types.GomegaFailHandler, offset int, extra ...interface{}) *Assertion {
	return &Assertion{
		actualInput: actualInput,
		fail:        fail,
		offset:      offset,
		extra:       extra,
	}
}

func (assertion *Assertion) Should(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return assertion.vetExtras(optionalDescription...) && assertion.match(matcher, true, optionalDescription...)
}

func (assertion *Assertion) ShouldNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return assertion.vetExtras(optionalDescription...) && assertion.match(matcher, false, optionalDescription...)
}

func (assertion *Assertion) To(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return assertion.vetExtras(optionalDescription...) && assertion.match(matcher, true, optionalDescription...)
}

func (assertion *Assertion) ToNot(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return assertion.vetExtras(optionalDescription...) && assertion.match(matcher, false, optionalDescription...)
}

func (assertion *Assertion) NotTo(matcher types.GomegaMatcher, optionalDescription ...interface{}) bool {
	return assertion.vetExtras(optionalDescription...) && assertion.match(matcher, false, optionalDescription...)
}

func (assertion *Assertion) buildDescription(optionalDescription ...interface{}) string {
	switch len(optionalDescription) {
	case 0:
		return ""
	default:
		return fmt.Sprintf(optionalDescription[0].(string), optionalDescription[1:]...) + "\n"
	}
}

func (assertion *Assertion) match(matcher types.GomegaMatcher, desiredMatch bool, optionalDescription ...interface{}) bool {
	matches, err := matcher.Match(assertion.actualInput)
	description := assertion.buildDescription(optionalDescription...)
	if err != nil {
		assertion.fail(description+err.Error(), 2+assertion.offset)
		return false
	}
	if matches != desiredMatch {
		var message string
		if desiredMatch {
			message = matcher.FailureMessage(assertion.actualInput)
		} else {
			message = matcher.NegatedFailureMessage(assertion.actualInput)
		}
		assertion.fail(description+message, 2+assertion.offset)
		return false
	}

	return true
}

func (assertion *Assertion) vetExtras(optionalDescription ...interface{}) bool {
	success, message := vetExtras(assertion.extra)
	if success {
		return true
	}

	description := assertion.buildDescription(optionalDescription...)
	assertion.fail(description+message, 2+assertion.offset)
	return false
}

func vetExtras(extras []interface{}) (bool, string) {
	for i, extra := range extras {
		if extra != nil {
			zeroValue := reflect.Zero(reflect.TypeOf(extra)).Interface()
			if !reflect.DeepEqual(zeroValue, extra) {
				message := fmt.Sprintf("Unexpected non-nil/non-zero extra argument at index %d:\n\t<%T>: %#v", i+1, extra, extra)
				return false, message
			}
		}
	}
	return true, ""
}
