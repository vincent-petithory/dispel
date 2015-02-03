// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package dispel

var defaultsMux = gofmtTmpl(asset.init(asset{Name: "defaults_mux.go", Content: "" +
	"// +build impl\n\npackage dispel\n\nimport (\n\t\"net/http\"\n\t\"net/url\"\n\n\t\"github.com/gorilla/mux\"\n)\n\ntype GorillaRouter struct {\n\tRouter  *mux.Router\n\tBaseURL *url.URL\n}\n\nfunc (gr *GorillaRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {\n\tgr.Router.ServeHTTP(w, r)\n}\n\nfunc (gr *GorillaRouter) RegisterHandler(routeName string, handler http.Handler) {\n\tgr.Router.Get(routeName).Handler(handler)\n}\n\nfunc (gr *GorillaRouter) RegisterRoute(path string, name string) {\n\tgr.Router.Path(path).Name(name)\n}\n\nfunc (gr *GorillaRouter) GetRouteParam(r *http.Request, name string) string {\n\treturn mux.Vars(r)[name]\n}\n\nfunc (gr *GorillaRouter) ReverseRoute(name string, params ...string) *url.URL {\n\tvar urlPath string\n\tif u, err := gr.Router.Get(name).URLPath(params...); err != nil {\n\t\tpanic(err)\n\t} else {\n\t\turlPath = u.Path\n\t}\n\tvar u url.URL\n\tif gr.BaseURL != nil {\n\t\tu = (*gr.BaseURL)\n\t}\n\tu.Path = urlPath\n\treturn &u\n}\n" +
	""}))
