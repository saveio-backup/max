package route

import (
	"net/http"

	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmNeSwALyTCrgtCTsPiF7tcDN6uLtdi8qCMtFm7nct1nm1/httprouter"
	"github.com/saveio/max/Godeps/_workspace/src/gx/ipfs/QmRvYNctevGUW52urgmoFZscT6buMKqhHezLUS64WepGWn/go-net/context"
)

type param string

// Param returns param p for the context.
func Param(ctx context.Context, p string) string {
	return ctx.Value(param(p)).(string)
}

// WithParam returns a new context with param p set to v.
func WithParam(ctx context.Context, p, v string) context.Context {
	return context.WithValue(ctx, param(p), v)
}

// Router wraps httprouter.Router and adds support for prefixed sub-routers,
// per-request context injections and instrumentation.
type Router struct {
	rtr    *httprouter.Router
	prefix string
	instrh func(handlerName string, handler http.HandlerFunc) http.HandlerFunc
}

// New returns a new Router.
func New() *Router {
	return &Router{
		rtr: httprouter.New(),
	}
}

// WithInstrumentation returns a router with instrumentation support.
func (r *Router) WithInstrumentation(instrh func(handlerName string, handler http.HandlerFunc) http.HandlerFunc) *Router {
	return &Router{rtr: r.rtr, prefix: r.prefix, instrh: instrh}
}

// WithPrefix returns a router that prefixes all registered routes with prefix.
func (r *Router) WithPrefix(prefix string) *Router {
	return &Router{rtr: r.rtr, prefix: r.prefix + prefix, instrh: r.instrh}
}

// handle turns a HandlerFunc into an httprouter.Handle.
func (r *Router) handle(handlerName string, h http.HandlerFunc) httprouter.Handle {
	if r.instrh != nil {
		// This needs to be outside the closure to avoid data race when reading and writing to 'h'.
		h = r.instrh(handlerName, h)
	}
	return func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		ctx, cancel := context.WithCancel(req.Context())
		defer cancel()

		for _, p := range params {
			ctx = context.WithValue(ctx, param(p.Key), p.Value)
		}
		h(w, req.WithContext(ctx))
	}
}

// Get registers a new GET route.
func (r *Router) Get(path string, h http.HandlerFunc) {
	r.rtr.GET(r.prefix+path, r.handle(path, h))
}

// Options registers a new OPTIONS route.
func (r *Router) Options(path string, h http.HandlerFunc) {
	r.rtr.OPTIONS(r.prefix+path, r.handle(path, h))
}

// Del registers a new DELETE route.
func (r *Router) Del(path string, h http.HandlerFunc) {
	r.rtr.DELETE(r.prefix+path, r.handle(path, h))
}

// Put registers a new PUT route.
func (r *Router) Put(path string, h http.HandlerFunc) {
	r.rtr.PUT(r.prefix+path, r.handle(path, h))
}

// Post registers a new POST route.
func (r *Router) Post(path string, h http.HandlerFunc) {
	r.rtr.POST(r.prefix+path, r.handle(path, h))
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
		r.URL.Path = Param(r.Context(), "filepath")
		fs.ServeHTTP(w, r)
	}
}
