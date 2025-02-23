package reporters_test

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/config"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/codelocation"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/reporters"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/types"
)

var _ = Describe("JUnit Reporter", func() {
	var (
		outputFile string
		reporter   Reporter
	)

	readOutputFile := func() reporters.JUnitTestSuite {
		bytes, err := ioutil.ReadFile(outputFile)
		Ω(err).ShouldNot(HaveOccurred())
		var suite reporters.JUnitTestSuite
		err = xml.Unmarshal(bytes, &suite)
		Ω(err).ShouldNot(HaveOccurred())
		return suite
	}

	BeforeEach(func() {
		f, err := ioutil.TempFile("", "output")
		Ω(err).ShouldNot(HaveOccurred())
		f.Close()
		outputFile = f.Name()

		reporter = reporters.NewJUnitReporter(outputFile)

		reporter.SpecSuiteWillBegin(config.GinkgoConfigType{}, &types.SuiteSummary{
			SuiteDescription:           "My test suite",
			NumberOfSpecsThatWillBeRun: 1,
		})
	})

	AfterEach(func() {
		os.RemoveAll(outputFile)
	})

	Describe("a passing test", func() {
		BeforeEach(func() {
			beforeSuite := &types.SetupSummary{
				State: types.SpecStatePassed,
			}
			reporter.BeforeSuiteDidRun(beforeSuite)

			afterSuite := &types.SetupSummary{
				State: types.SpecStatePassed,
			}
			reporter.AfterSuiteDidRun(afterSuite)

			spec := &types.SpecSummary{
				ComponentTexts: []string{"[Top Level]", "A", "B", "C"},
				State:          types.SpecStatePassed,
				RunTime:        5 * time.Second,
			}
			reporter.SpecWillRun(spec)
			reporter.SpecDidComplete(spec)

			reporter.SpecSuiteDidEnd(&types.SuiteSummary{
				NumberOfSpecsThatWillBeRun: 1,
				NumberOfFailedSpecs:        0,
				RunTime:                    10 * time.Second,
			})
		})

		It("should record the test as passing", func() {
			output := readOutputFile()
			Ω(output.Name).Should(Equal("My test suite"))
			Ω(output.Tests).Should(Equal(1))
			Ω(output.Failures).Should(Equal(0))
			Ω(output.Time).Should(Equal(10.0))
			Ω(output.TestCases).Should(HaveLen(1))
			Ω(output.TestCases[0].Name).Should(Equal("A B C"))
			Ω(output.TestCases[0].ClassName).Should(Equal("My test suite"))
			Ω(output.TestCases[0].FailureMessage).Should(BeNil())
			Ω(output.TestCases[0].Skipped).Should(BeNil())
			Ω(output.TestCases[0].Time).Should(Equal(5.0))
		})
	})

	Describe("when the BeforeSuite fails", func() {
		var beforeSuite *types.SetupSummary

		BeforeEach(func() {
			beforeSuite = &types.SetupSummary{
				State:   types.SpecStateFailed,
				RunTime: 3 * time.Second,
				Failure: types.SpecFailure{
					Message:               "failed to setup",
					ComponentCodeLocation: codelocation.New(0),
					Location:              codelocation.New(2),
				},
			}
			reporter.BeforeSuiteDidRun(beforeSuite)

			reporter.SpecSuiteDidEnd(&types.SuiteSummary{
				NumberOfSpecsThatWillBeRun: 1,
				NumberOfFailedSpecs:        1,
				RunTime:                    10 * time.Second,
			})
		})

		It("should record the test as having failed", func() {
			output := readOutputFile()
			Ω(output.Name).Should(Equal("My test suite"))
			Ω(output.Tests).Should(Equal(1))
			Ω(output.Failures).Should(Equal(1))
			Ω(output.Time).Should(Equal(10.0))
			Ω(output.TestCases[0].Name).Should(Equal("BeforeSuite"))
			Ω(output.TestCases[0].Time).Should(Equal(3.0))
			Ω(output.TestCases[0].ClassName).Should(Equal("My test suite"))
			Ω(output.TestCases[0].FailureMessage.Type).Should(Equal("Failure"))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring("failed to setup"))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(beforeSuite.Failure.ComponentCodeLocation.String()))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(beforeSuite.Failure.Location.String()))
			Ω(output.TestCases[0].Skipped).Should(BeNil())
		})
	})

	Describe("when the AfterSuite fails", func() {
		var afterSuite *types.SetupSummary

		BeforeEach(func() {
			afterSuite = &types.SetupSummary{
				State:   types.SpecStateFailed,
				RunTime: 3 * time.Second,
				Failure: types.SpecFailure{
					Message:               "failed to setup",
					ComponentCodeLocation: codelocation.New(0),
					Location:              codelocation.New(2),
				},
			}
			reporter.AfterSuiteDidRun(afterSuite)

			reporter.SpecSuiteDidEnd(&types.SuiteSummary{
				NumberOfSpecsThatWillBeRun: 1,
				NumberOfFailedSpecs:        1,
				RunTime:                    10 * time.Second,
			})
		})

		It("should record the test as having failed", func() {
			output := readOutputFile()
			Ω(output.Name).Should(Equal("My test suite"))
			Ω(output.Tests).Should(Equal(1))
			Ω(output.Failures).Should(Equal(1))
			Ω(output.Time).Should(Equal(10.0))
			Ω(output.TestCases[0].Name).Should(Equal("AfterSuite"))
			Ω(output.TestCases[0].Time).Should(Equal(3.0))
			Ω(output.TestCases[0].ClassName).Should(Equal("My test suite"))
			Ω(output.TestCases[0].FailureMessage.Type).Should(Equal("Failure"))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring("failed to setup"))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(afterSuite.Failure.ComponentCodeLocation.String()))
			Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(afterSuite.Failure.Location.String()))
			Ω(output.TestCases[0].Skipped).Should(BeNil())
		})
	})

	specStateCases := []struct {
		state   types.SpecState
		message string
	}{
		{types.SpecStateFailed, "Failure"},
		{types.SpecStateTimedOut, "Timeout"},
		{types.SpecStatePanicked, "Panic"},
	}

	for _, specStateCase := range specStateCases {
		specStateCase := specStateCase
		Describe("a failing test", func() {
			var spec *types.SpecSummary
			BeforeEach(func() {
				spec = &types.SpecSummary{
					ComponentTexts: []string{"[Top Level]", "A", "B", "C"},
					State:          specStateCase.state,
					RunTime:        5 * time.Second,
					Failure: types.SpecFailure{
						ComponentCodeLocation: codelocation.New(0),
						Location:              codelocation.New(2),
						Message:               "I failed",
					},
				}
				reporter.SpecWillRun(spec)
				reporter.SpecDidComplete(spec)

				reporter.SpecSuiteDidEnd(&types.SuiteSummary{
					NumberOfSpecsThatWillBeRun: 1,
					NumberOfFailedSpecs:        1,
					RunTime:                    10 * time.Second,
				})
			})

			It("should record test as failing", func() {
				output := readOutputFile()
				Ω(output.Name).Should(Equal("My test suite"))
				Ω(output.Tests).Should(Equal(1))
				Ω(output.Failures).Should(Equal(1))
				Ω(output.Time).Should(Equal(10.0))
				Ω(output.TestCases[0].Name).Should(Equal("A B C"))
				Ω(output.TestCases[0].ClassName).Should(Equal("My test suite"))
				Ω(output.TestCases[0].FailureMessage.Type).Should(Equal(specStateCase.message))
				Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring("I failed"))
				Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(spec.Failure.ComponentCodeLocation.String()))
				Ω(output.TestCases[0].FailureMessage.Message).Should(ContainSubstring(spec.Failure.Location.String()))
				Ω(output.TestCases[0].Skipped).Should(BeNil())
			})
		})
	}

	for _, specStateCase := range []types.SpecState{types.SpecStatePending, types.SpecStateSkipped} {
		specStateCase := specStateCase
		Describe("a skipped test", func() {
			var spec *types.SpecSummary
			BeforeEach(func() {
				spec = &types.SpecSummary{
					ComponentTexts: []string{"[Top Level]", "A", "B", "C"},
					State:          specStateCase,
					RunTime:        5 * time.Second,
				}
				reporter.SpecWillRun(spec)
				reporter.SpecDidComplete(spec)

				reporter.SpecSuiteDidEnd(&types.SuiteSummary{
					NumberOfSpecsThatWillBeRun: 1,
					NumberOfFailedSpecs:        0,
					RunTime:                    10 * time.Second,
				})
			})

			It("should record test as failing", func() {
				output := readOutputFile()
				Ω(output.Tests).Should(Equal(1))
				Ω(output.Failures).Should(Equal(0))
				Ω(output.Time).Should(Equal(10.0))
				Ω(output.TestCases[0].Name).Should(Equal("A B C"))
				Ω(output.TestCases[0].Skipped).ShouldNot(BeNil())
			})
		})
	}
})
