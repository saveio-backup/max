package route

import (
	"net/http"
	"sync"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNeSwALyTCrgtCTsPiF7tcDN6uLtdi8qCMtFm7nct1nm1/httprouter"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmTEmsyNnckEq8rEfALfdhLHjrEHGoSGFDrAYReuetn7MC/go-net/context"
)

var (
	mtx   = sync.RWMutex{}
	ctxts = map[*http.Request]context.Context{}
)

// Context returns the context for the request.
func Context(r *http.Request) context.Context {
	mtx.RLock()
	defer mtx.RUnlock()
	return ctxts[r]
}

type param string

// Param returns param p for the context.
func Param(ctx context.Context, p string) string {
	return ctx.Value(param(p)).(string)
}

// WithParam returns a new context with param p set to v.
func WithParam(ctx context.Context, p, v string) context.Context {
	return context.WithValue(ctx, param(p), v)
}

// handle turns a Handle into httprouter.Handle
func handle(h http.HandlerFunc) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for _, p := range params {
			ctx = context.WithValue(ctx, param(p.Key), p.Value)
		}

		mtx.Lock()
		ctxts[r] = ctx
		mtx.Unlock()

		h(w, r)

		mtx.Lock()
		delete(ctxts, r)
		mtx.Unlock()
	}
}

// Router wraps httprouter.Router and adds support for prefixed sub-routers.
type Router struct {
	rtr    *httprouter.Router
	prefix string
}

// New returns a new Router.
func New() *Router {
	return &Router{rtr: httprouter.New()}
}

// WithPrefix returns a router that prefixes all registered routes with prefix.
func (r *Router) WithPrefix(prefix string) *Router {
	return &Router{rtr: r.rtr, prefix: r.prefix + prefix}
}

// Get registers a new GET route.
func (r *Router) Get(path string, h http.HandlerFunc) {
	r.rtr.GET(r.prefix+path, handle(h))
}

// Del registers a new DELETE route.
func (r *Router) Del(path string, h http.HandlerFunc) {
	r.rtr.DELETE(r.prefix+path, handle(h))
}

// Put registers a new PUT route.
func (r *Router) Put(path string, h http.HandlerFunc) {
	r.rtr.PUT(r.prefix+path, handle(h))
}

// Post registers a new POST route.
func (r *Router) Post(path string, h http.HandlerFunc) {
	r.rtr.POST(r.prefix+path, handle(h))
}

// Redirect takes an absolute path and sends an internal HTTP redirect for it,
// prefixed by the router's path prefix. Note that this method does not include
// functionality for handling relative paths or full URL redirects.
func (r *Router) Redirect(w http.ResponseWriter, req *http.Request, path string, code int) {
	http.Redirect(w, req, r.prefix+path, code)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.rtr.ServeHTTP(w, req)
}

// FileServe returns a new http.HandlerFunc that serves files from dir.
// Using routes must provide the *filepath parameter.
func FileServe(dir string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))

	return func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = Param(Context(r), "filepath")
		fs.ServeHTTP(w, r)
	}
}
