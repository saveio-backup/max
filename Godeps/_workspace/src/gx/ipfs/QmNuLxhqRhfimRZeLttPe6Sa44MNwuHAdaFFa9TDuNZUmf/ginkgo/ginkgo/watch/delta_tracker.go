package watch

import (
	"fmt"

	"regexp"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/ginkgo/testsuite"
)

type SuiteErrors map[testsuite.TestSuite]error

type DeltaTracker struct {
	maxDepth      int
	watchRegExp   *regexp.Regexp
	suites        map[string]*Suite
	packageHashes *PackageHashes
}

func NewDeltaTracker(maxDepth int, watchRegExp *regexp.Regexp) *DeltaTracker {
	return &DeltaTracker{
		maxDepth:      maxDepth,
		watchRegExp:   watchRegExp,
		packageHashes: NewPackageHashes(watchRegExp),
		suites:        map[string]*Suite{},
	}
}

func (d *DeltaTracker) Delta(suites []testsuite.TestSuite) (delta Delta, errors SuiteErrors) {
	errors = SuiteErrors{}
	delta.ModifiedPackages = d.packageHashes.CheckForChanges()

	providedSuitePaths := map[string]bool{}
	for _, suite := range suites {
		providedSuitePaths[suite.Path] = true
	}

	d.packageHashes.StartTrackingUsage()

	for _, suite := range d.suites {
		if providedSuitePaths[suite.Suite.Path] {
			if suite.Delta() > 0 {
				delta.modifiedSuites = append(delta.modifiedSuites, suite)
			}
		} else {
			delta.RemovedSuites = append(delta.RemovedSuites, suite)
		}
	}

	d.packageHashes.StopTrackingUsageAndPrune()

	for _, suite := range suites {
		_, ok := d.suites[suite.Path]
		if !ok {
			s, err := NewSuite(suite, d.maxDepth, d.packageHashes)
			if err != nil {
				errors[suite] = err
				continue
			}
			d.suites[suite.Path] = s
			delta.NewSuites = append(delta.NewSuites, s)
		}
	}

	return delta, errors
}

func (d *DeltaTracker) WillRun(suite testsuite.TestSuite) error {
	s, ok := d.suites[suite.Path]
	if !ok {
		return fmt.Errorf("unknown suite %s", suite.Path)
	}

	return s.MarkAsRunAndRecomputedDependencies(d.maxDepth)
}
