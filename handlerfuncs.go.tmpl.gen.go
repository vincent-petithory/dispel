// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package main

var handlerfuncsTmpl = tmpl(asset.init(asset{Name: "handlerfuncs.go.tmpl", Content: "" +
	"// generated by {{ .Prgm }}; DO NOT EDIT\n\npackage {{ .PkgName }}\n\nimport (\n\t\"net/http\"\n)\n\n{{ range .Routes.ByResource }}{{ $route := . }}{{ range .Methods }}{{ $lmethod := . | tolower }}{{ with $funcName := printf \"%s%s\" $lmethod ($route.Name | symbolName) }}{{ if handlerFuncMissing $funcName }}func (h *Handler) {{ $funcName }}(w http.ResponseWriter, r *http.Request{{ range $route.RouteParams }}, {{ .Varname }} {{ .Type }}{{end}}) *endpointError {\n\thttp.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)\n\treturn nil\n}\n\n{{end}}{{end}}{{end}}{{end}}\n" +
	""}))