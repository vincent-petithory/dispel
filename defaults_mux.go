// +build impl

package dispel

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

// GorillaRouter is an implementation of all major interfaces exposed by dispel.
// It registers routes, maps them to handlers and can perform route reversing.
type GorillaRouter struct {
	Router  *mux.Router
	BaseURL *url.URL
}

// ServeHTTP calls the gorilla/mux router's ServeHTTP.
func (gr *GorillaRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gr.Router.ServeHTTP(w, r)
}

// RegisterHandler makes the named route be handled by handler.
func (gr *GorillaRouter) RegisterHandler(routeName string, handler http.Handler) {
	gr.Router.Get(routeName).Handler(handler)
}

// RegisterRoute associates a name to the specified path.
func (gr *GorillaRouter) RegisterRoute(path string, name string) {
	gr.Router.Path(path).Name(name)
}

// GetRouteParam retrieves the parameter name in the request's url path.
// It returns "" if there is no such param name.
func (gr *GorillaRouter) GetRouteParam(r *http.Request, name string) string {
	return mux.Vars(r)[name]
}

// ReverseRoute builds an URL using the named route and params.
// It panics if the named route can't be found or couldn't be built.
func (gr *GorillaRouter) ReverseRoute(name string, params ...string) *url.URL {
	var urlPath string
	if u, err := gr.Router.Get(name).URLPath(params...); err != nil {
		panic(err)
	} else {
		urlPath = u.Path
	}
	var u url.URL
	if gr.BaseURL != nil {
		u = (*gr.BaseURL)
	}
	u.Path = urlPath
	return &u
}
