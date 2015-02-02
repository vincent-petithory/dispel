// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package dispel

var routesTmpl = tmpl(asset.init(asset{Name: "routes.go.tmpl", Content: "" +
	"// generated by {{ .Prgm }}; DO NOT EDIT\n\npackage {{ .PkgName }}\n\nimport (\n    \"net/url\"\n)\n\n// RouteRegisterer is the interface implemented by objects that can register a name for a route path.\ntype RouteRegisterer interface {\n    RegisterRoute(path string, name string)\n}\n\n// RouteReverser is the interface implemented by objects that can retrieve the url of a route based on\n// its registered name and the route param names and values.\ntype RouteReverser interface {\n    ReverseRoute(name string, params ...string) *url.URL \n}\n\n// RouteLocation is the interface implemented by objects that can return an url for a route, using\n// a RouteReverser.\ntype RouteLocation interface {\n\tLocation(RouteReverser) *url.URL\n}\n\n// registerRoutes uses rr to register the routes by path and name.\nfunc registerRoutes(rr RouteRegisterer) {\n{{ range .Routes.ByResource }}    rr.RegisterRoute(\"{{ .Path }}\", route{{ symbolName .Name }})\n{{end}}}\n\nconst (\n{{ range .Routes.ByResource }}    route{{ symbolName .Name }} = \"{{ .Name }}\"\n{{end}}\n)\n\ntype (\n{{ range .Routes.ByResource }}    Route{{ symbolName .Name }} struct { {{ range .RouteParams }}\n    {{ symbolName .Varname }} string {{ end }}}\n{{end}}\n)\n\n{{ range .Routes.ByResource }}func (r Route{{ symbolName .Name }}) Location(rr RouteReverser) *url.URL {\n    return rr.ReverseRoute(route{{ symbolName .Name }}, {{ range .RouteParams }}\"{{ .Name }}\", r.{{ symbolName .Varname }},{{end}})\n}\n{{end}}\n" +
	""}))
