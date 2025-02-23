package suite

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/spec_iterator"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/config"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/containernode"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/failer"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/leafnodes"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/spec"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/specrunner"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/writer"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/reporters"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/types"
)

type ginkgoTestingT interface {
	Fail()
}

type Suite struct {
	topLevelContainer *containernode.ContainerNode
	currentContainer  *containernode.ContainerNode
	containerIndex    int
	beforeSuiteNode   leafnodes.SuiteNode
	afterSuiteNode    leafnodes.SuiteNode
	runner            *specrunner.SpecRunner
	failer            *failer.Failer
	running           bool
}

func New(failer *failer.Failer) *Suite {
	topLevelContainer := containernode.New("[Top Level]", types.FlagTypeNone, types.CodeLocation{})

	return &Suite{
		topLevelContainer: topLevelContainer,
		currentContainer:  topLevelContainer,
		failer:            failer,
		containerIndex:    1,
	}
}

func (suite *Suite) Run(t ginkgoTestingT, description string, reporters []reporters.Reporter, writer writer.WriterInterface, config config.GinkgoConfigType) (bool, bool) {
	if config.ParallelTotal < 1 {
		panic("ginkgo.parallel.total must be >= 1")
	}

	if config.ParallelNode > config.ParallelTotal || config.ParallelNode < 1 {
		panic("ginkgo.parallel.node is one-indexed and must be <= ginkgo.parallel.total")
	}

	r := rand.New(rand.NewSource(config.RandomSeed))
	suite.topLevelContainer.Shuffle(r)
	iterator, hasProgrammaticFocus := suite.generateSpecsIterator(description, config)
	suite.runner = specrunner.New(description, suite.beforeSuiteNode, iterator, suite.afterSuiteNode, reporters, writer, config)

	suite.running = true
	success := suite.runner.Run()
	if !success {
		t.Fail()
	}
	return success, hasProgrammaticFocus
}

func (suite *Suite) generateSpecsIterator(description string, config config.GinkgoConfigType) (spec_iterator.SpecIterator, bool) {
	specsSlice := []*spec.Spec{}
	suite.topLevelContainer.BackPropagateProgrammaticFocus()
	for _, collatedNodes := range suite.topLevelContainer.Collate() {
		specsSlice = append(specsSlice, spec.New(collatedNodes.Subject, collatedNodes.Containers, config.EmitSpecProgress))
	}

	specs := spec.NewSpecs(specsSlice)
	specs.RegexScansFilePath = config.RegexScansFilePath

	if config.RandomizeAllSpecs {
		specs.Shuffle(rand.New(rand.NewSource(config.RandomSeed)))
	}

	specs.ApplyFocus(description, config.FocusString, config.SkipString)

	if config.SkipMeasurements {
		specs.SkipMeasurements()
	}

	var iterator spec_iterator.SpecIterator

	if config.ParallelTotal > 1 {
		iterator = spec_iterator.NewParallelIterator(specs.Specs(), config.SyncHost)
		resp, err := http.Get(config.SyncHost + "/has-counter")
		if err != nil || resp.StatusCode != http.StatusOK {
			iterator = spec_iterator.NewShardedParallelIterator(specs.Specs(), config.ParallelTotal, config.ParallelNode)
		}
	} else {
		iterator = spec_iterator.NewSerialIterator(specs.Specs())
	}

	return iterator, specs.HasProgrammaticFocus()
}

func (suite *Suite) CurrentRunningSpecSummary() (*types.SpecSummary, bool) {
	return suite.runner.CurrentSpecSummary()
}

func (suite *Suite) SetBeforeSuiteNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.beforeSuiteNode != nil {
		panic("You may only call BeforeSuite once!")
	}
	suite.beforeSuiteNode = leafnodes.NewBeforeSuiteNode(body, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetAfterSuiteNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.afterSuiteNode != nil {
		panic("You may only call AfterSuite once!")
	}
	suite.afterSuiteNode = leafnodes.NewAfterSuiteNode(body, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetSynchronizedBeforeSuiteNode(bodyA interface{}, bodyB interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.beforeSuiteNode != nil {
		panic("You may only call BeforeSuite once!")
	}
	suite.beforeSuiteNode = leafnodes.NewSynchronizedBeforeSuiteNode(bodyA, bodyB, codeLocation, timeout, suite.failer)
}

func (suite *Suite) SetSynchronizedAfterSuiteNode(bodyA interface{}, bodyB interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.afterSuiteNode != nil {
		panic("You may only call AfterSuite once!")
	}
	suite.afterSuiteNode = leafnodes.NewSynchronizedAfterSuiteNode(bodyA, bodyB, codeLocation, timeout, suite.failer)
}

func (suite *Suite) PushContainerNode(text string, body func(), flag types.FlagType, codeLocation types.CodeLocation) {
	container := containernode.New(text, flag, codeLocation)
	suite.currentContainer.PushContainerNode(container)

	previousContainer := suite.currentContainer
	suite.currentContainer = container
	suite.containerIndex++

	body()

	suite.containerIndex--
	suite.currentContainer = previousContainer
}

func (suite *Suite) PushItNode(text string, body interface{}, flag types.FlagType, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call It from within a Describe, Context or When", codeLocation)
	}
	suite.currentContainer.PushSubjectNode(leafnodes.NewItNode(text, body, flag, codeLocation, timeout, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushMeasureNode(text string, body interface{}, flag types.FlagType, codeLocation types.CodeLocation, samples int) {
	if suite.running {
		suite.failer.Fail("You may only call Measure from within a Describe, Context or When", codeLocation)
	}
	suite.currentContainer.PushSubjectNode(leafnodes.NewMeasureNode(text, body, flag, codeLocation, samples, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call BeforeEach from within a Describe, Context or When", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewBeforeEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushJustBeforeEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call JustBeforeEach from within a Describe, Context or When", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewJustBeforeEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}

func (suite *Suite) PushAfterEachNode(body interface{}, codeLocation types.CodeLocation, timeout time.Duration) {
	if suite.running {
		suite.failer.Fail("You may only call AfterEach from within a Describe, Context or When", codeLocation)
	}
	suite.currentContainer.PushSetupNode(leafnodes.NewAfterEachNode(body, codeLocation, timeout, suite.failer, suite.containerIndex))
}
