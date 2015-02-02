package dispel

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

type GorillaRouter struct {
	Router  *mux.Router
	BaseURL *url.URL
}

func (gr *GorillaRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	gr.Router.ServeHTTP(w, r)
}

func (gr *GorillaRouter) RegisterHandler(routeName string, handler http.Handler) {
	gr.Router.Get(routeName).Handler(handler)
}

func (gr *GorillaRouter) RegisterRoute(path string, name string) {
	gr.Router.Path(path).Name(name)
}

func (gr *GorillaRouter) GetRouteParam(r *http.Request, name string) string {
	return mux.Vars(r)[name]
}

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
