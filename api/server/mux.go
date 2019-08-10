package server

import (
	"context"
	"net/http"
	"regexp"
)

// Mux is a router where path patterns are regular expressions.
type Mux struct {
	routes []route
}

// NewMux returns a new Mux.
func NewMux() *Mux { return &Mux{} }

type route struct {
	method  string
	path    *regexp.Regexp
	handler http.Handler
}

// HandleFunc registers handler.
func (mux *Mux) HandleFunc(method string, path *regexp.Regexp, handler http.HandlerFunc) {
	// TODO: ensure route does not already exist.
	mux.routes = append(mux.routes,
		route{method: method, path: path, handler: handler},
	)
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		h        http.Handler
		match    []string
		hasmatch bool
	)
	for _, route := range mux.routes {
		match = route.path.FindStringSubmatch(r.URL.Path)
		if match == nil {
			continue
		}
		hasmatch = true
		if route.method != r.Method {
			continue
		}
		h = route.handler
		break
	}

	if h == nil {
		status := http.StatusNotFound
		if hasmatch {
			status = http.StatusMethodNotAllowed
		}
		http.Error(w, http.StatusText(status), status)
		return
	}

	r = r.WithContext(context.WithValue(
		r.Context(), matchContextKey{}, match,
	))

	h.ServeHTTP(w, r)
}

type matchContextKey struct{}

// ContextMatch returns the match associated with ctx.
func ContextMatch(ctx context.Context) []string {
	match, _ := ctx.Value(matchContextKey{}).([]string)
	return match
}
