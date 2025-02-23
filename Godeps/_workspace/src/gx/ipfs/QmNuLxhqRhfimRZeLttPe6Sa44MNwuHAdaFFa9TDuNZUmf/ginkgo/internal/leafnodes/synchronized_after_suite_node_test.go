package leafnodes_test

import (
	"sync"

	. "github.com/onsi/gomega"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo"
	. "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/leafnodes"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/types"

	"net/http"

	"github.com/onsi/gomega/ghttp"

	"time"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/codelocation"
	Failer "github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNuLxhqRhfimRZeLttPe6Sa44MNwuHAdaFFa9TDuNZUmf/ginkgo/internal/failer"
)

var _ = Describe("SynchronizedAfterSuiteNode", func() {
	var failer *Failer.Failer
	var node SuiteNode
	var codeLocation types.CodeLocation
	var innerCodeLocation types.CodeLocation
	var outcome bool
	var server *ghttp.Server
	var things []string
	var lock *sync.Mutex

	BeforeEach(func() {
		things = []string{}
		server = ghttp.NewServer()
		codeLocation = codelocation.New(0)
		innerCodeLocation = codelocation.New(0)
		failer = Failer.New()
		lock = &sync.Mutex{}
	})

	AfterEach(func() {
		server.Close()
	})

	newNode := func(bodyA interface{}, bodyB interface{}) SuiteNode {
		return NewSynchronizedAfterSuiteNode(bodyA, bodyB, codeLocation, time.Millisecond, failer)
	}

	ranThing := func(thing string) {
		lock.Lock()
		defer lock.Unlock()
		things = append(things, thing)
	}

	thingsThatRan := func() []string {
		lock.Lock()
		defer lock.Unlock()
		return things
	}

	Context("when not running in parallel", func() {
		Context("when all is well", func() {
			BeforeEach(func() {
				node = newNode(func() {
					ranThing("A")
				}, func() {
					ranThing("B")
				})

				outcome = node.Run(1, 1, server.URL())
			})

			It("should run A, then B", func() {
				Ω(thingsThatRan()).Should(Equal([]string{"A", "B"}))
			})

			It("should report success", func() {
				Ω(outcome).Should(BeTrue())
				Ω(node.Passed()).Should(BeTrue())
				Ω(node.Summary().State).Should(Equal(types.SpecStatePassed))
			})
		})

		Context("when A fails", func() {
			BeforeEach(func() {
				node = newNode(func() {
					ranThing("A")
					failer.Fail("bam", innerCodeLocation)
				}, func() {
					ranThing("B")
				})

				outcome = node.Run(1, 1, server.URL())
			})

			It("should still run B", func() {
				Ω(thingsThatRan()).Should(Equal([]string{"A", "B"}))
			})

			It("should report failure", func() {
				Ω(outcome).Should(BeFalse())
				Ω(node.Passed()).Should(BeFalse())
				Ω(node.Summary().State).Should(Equal(types.SpecStateFailed))
			})
		})

		Context("when B fails", func() {
			BeforeEach(func() {
				node = newNode(func() {
					ranThing("A")
				}, func() {
					ranThing("B")
					failer.Fail("bam", innerCodeLocation)
				})

				outcome = node.Run(1, 1, server.URL())
			})

			It("should run all the things", func() {
				Ω(thingsThatRan()).Should(Equal([]string{"A", "B"}))
			})

			It("should report failure", func() {
				Ω(outcome).Should(BeFalse())
				Ω(node.Passed()).Should(BeFalse())
				Ω(node.Summary().State).Should(Equal(types.SpecStateFailed))
			})
		})
	})

	Context("when running in parallel", func() {
		Context("as the first node", func() {
			BeforeEach(func() {
				server.AppendHandlers(ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/RemoteAfterSuiteData"),
					func(writer http.ResponseWriter, request *http.Request) {
						ranThing("Request1")
					},
					ghttp.RespondWithJSONEncoded(200, types.RemoteAfterSuiteData{CanRun: false}),
				), ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/RemoteAfterSuiteData"),
					func(writer http.ResponseWriter, request *http.Request) {
						ranThing("Request2")
					},
					ghttp.RespondWithJSONEncoded(200, types.RemoteAfterSuiteData{CanRun: false}),
				), ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/RemoteAfterSuiteData"),
					func(writer http.ResponseWriter, request *http.Request) {
						ranThing("Request3")
					},
					ghttp.RespondWithJSONEncoded(200, types.RemoteAfterSuiteData{CanRun: true}),
				))

				node = newNode(func() {
					ranThing("A")
				}, func() {
					ranThing("B")
				})

				outcome = node.Run(1, 3, server.URL())
			})

			It("should run A and, when the server says its time, run B", func() {
				Ω(thingsThatRan()).Should(Equal([]string{"A", "Request1", "Request2", "Request3", "B"}))
			})

			It("should report success", func() {
				Ω(outcome).Should(BeTrue())
				Ω(node.Passed()).Should(BeTrue())
				Ω(node.Summary().State).Should(Equal(types.SpecStatePassed))
			})
		})

		Context("as any other node", func() {
			BeforeEach(func() {
				node = newNode(func() {
					ranThing("A")
				}, func() {
					ranThing("B")
				})

				outcome = node.Run(2, 3, server.URL())
			})

			It("should run A, and not run B", func() {
				Ω(thingsThatRan()).Should(Equal([]string{"A"}))
			})

			It("should not talk to the server", func() {
				Ω(server.ReceivedRequests()).Should(BeEmpty())
			})

			It("should report success", func() {
				Ω(outcome).Should(BeTrue())
				Ω(node.Passed()).Should(BeTrue())
				Ω(node.Summary().State).Should(Equal(types.SpecStatePassed))
			})
		})
	})
})
