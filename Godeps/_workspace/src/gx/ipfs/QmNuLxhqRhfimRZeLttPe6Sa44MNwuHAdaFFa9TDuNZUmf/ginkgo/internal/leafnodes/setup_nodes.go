package leafnodes

import (
	"time"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/failer"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/types"
)

type SetupNode struct {
	runner *runner
}

func (node *SetupNode) Run() (outcome types.SpecState, failure types.SpecFailure) {
	return node.runner.run()
}

func (node *SetupNode) Type() types.SpecComponentType {
	return node.runner.nodeType
}

func (node *SetupNode) CodeLocation() types.CodeLocation {
	return node.runner.codeLocation
}

func NewBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration, failer *failer.Failer, componentIndex int) *SetupNode {
	return &SetupNode{
		runner: newRunner(body, codeLocation, timeout, failer, types.SpecComponentTypeBeforeEach, componentIndex),
	}
}

func NewAfterEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration, failer *failer.Failer, componentIndex int) *SetupNode {
	return &SetupNode{
		runner: newRunner(body, codeLocation, timeout, failer, types.SpecComponentTypeAfterEach, componentIndex),
	}
}

func NewJustBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration, failer *failer.Failer, componentIndex int) *SetupNode {
	return &SetupNode{
		runner: newRunner(body, codeLocation, timeout, failer, types.SpecComponentTypeJustBeforeEach, componentIndex),
	}
}
