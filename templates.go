package dispel

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

//go:generate -command asset go run ./asset.go
//go:generate asset --var=routesTmpl routes.go.tmpl
//go:generate asset --var=handlersTmpl handlers.go.tmpl
//go:generate asset --var=handlerfuncsTmpl handlerfuncs.go.tmpl
//go:generate asset --var=typesTmpl types.go.tmpl

var templatesMap = map[string]string{
	TemplateRoutes:       routesTmpl,
	TemplateHandlers:     handlersTmpl,
	TemplateHandlerfuncs: handlerfuncsTmpl,
	TemplateTypes:        typesTmpl,
}

// The templates available in a TemplateBundle.
const (
	TemplateRoutes       = "routes"
	TemplateHandlers     = "handlers"
	TemplateHandlerfuncs = "handlerfuncs"
	TemplateTypes        = "types"
)

// TemplateNames returns the same list than TemplateBundle.Names(), but doesn't require
// to create a Template instance.
func TemplateNames() []string {
	var a []string
	for name := range templatesMap {
		a = append(a, name)
	}
	sort.Strings(a)
	return a
}

func tmpl(a asset) string {
	return a.Content
}

// TemplateBundle represents a bundle of templates for generating the various dispel API source files.
type TemplateBundle struct {
	t  *template.Template
	sp *SchemaParser
}

// ExecuteTemplate executes the template named by name, using the ctx TemplateContext.
func (t *TemplateBundle) ExecuteTemplate(wr io.Writer, name string, ctx *TemplateContext) error {
	ctx.sp = t.sp
	return t.t.ExecuteTemplate(wr, name, ctx)
}

// ExecuteCustomTemplate parses then executes the template text, using the ctx TemplateContext.
func (t *TemplateBundle) ExecuteCustomTemplate(wr io.Writer, text string, ctx *TemplateContext) error {
	ctx.sp = t.sp
	tt, err := t.t.New("custom").Parse(text)
	if err != nil {
		return err
	}
	return tt.Execute(wr, ctx)
}

// Names returns the list of templates available in the Template bundle.
func (t *TemplateBundle) Names() []string {
	var a []string
	for _, tmpl := range t.t.Templates() {
		a = append(a, tmpl.Name())
	}
	return a
}

// TemplateContext represents the context passed to a Template.
type TemplateContext struct {
	Prgm                string   // name of the program generating the source
	PkgName             string   // package name for which source code is generated
	Routes              Routes   // routes parsed by the SchemaParser
	HandlerReceiverType string   // type which acts as the receiver of the handler funcs.
	ExistingHandlers    []string // list of existing handler funcs in the target package, with HandlerReceiverType as the receiver
	ExistingTypes       []string // list of existing types in the target package.

	sp *SchemaParser
}

func handlerFuncName(routeMethod string, routeName string) string {
	return strings.ToLower(routeMethod) + symbolName(routeName)
}

// NewTemplateBundle returns a new Template based on the SchemaParser.
func NewTemplateBundle(sp *SchemaParser) (*TemplateBundle, error) {
	t := template.New("").Funcs(NewTemplateFuncMap(sp))
	for name, tmpl := range templatesMap {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("template %s: %v", name, err)
		}
	}
	return &TemplateBundle{t: t, sp: sp}, nil
}

// NewTemplateFuncMap returns the template.FuncMap used in a TemplateBundle.
func NewTemplateFuncMap(sp *SchemaParser) template.FuncMap {
	return template.FuncMap{
		"tolower":    strings.ToLower,
		"capitalize": capitalize,
		"symbolName": symbolName,
		"hasItem": func(a []string, s string) bool {
			for _, item := range a {
				if s == item {
					return true
				}
			}
			return false
		},
		"handlerFuncName": handlerFuncName,
		"allHandlerFuncsImplemented": func(routes Routes, existingHandlers []string) bool {
		LRoutesLoop:
			for _, route := range routes {
				fname := handlerFuncName(route.Method, route.Name)
				for _, h := range existingHandlers {
					if h == fname {
						continue LRoutesLoop
					}
				}
				return false
			}
			return true
		},
		"varname": func(s string) string {
			var buf bytes.Buffer
			var pickedFirstNonSymbolRune bool
			for _, r := range s {
				switch {
				case r == '*':
					continue
				case unicode.IsUpper(r):
					_, _ = buf.WriteRune(unicode.ToLower(r))
					pickedFirstNonSymbolRune = true
				default:
					if !pickedFirstNonSymbolRune {
						_, _ = buf.WriteRune(unicode.ToLower(r))
						pickedFirstNonSymbolRune = true
					}
				}
			}
			return buf.String()
		},
		"typeImports": func(routes Routes) []string {
			var imports []string
			for _, route := range routes {
				for _, typ := range []JSONType{route.InType, route.OutType} {
					if typ == nil {
						continue
					}
					routes.walkType(typ, func(jt JSONType) {
						_, ok := jt.(JSONDateTime)
						if ok {
							imports = append(imports, "time")
						}
					})
				}
			}
			sort.Strings(imports)
			return imports
		},
		"printTypeDef": func(j JSONType) string {
			// we don't want type with a slice as underlying type.
			// See https://golang.org/ref/spec#Assignability
			_, ok := j.(JSONArray)
			if ok {
				return ""
			}
			return fmt.Sprintf("type %s %s", sp.JSONToGoType(j, false), sp.JSONToGoType(j, true))
		},
		"typeNeedsAddr": func(j JSONType) bool {
			switch j.(type) {
			case JSONArray:
				return false
			default:
				return true
			}
		},
		"printTypeName": func(j JSONType) string {
			return sp.JSONToGoType(j, false)
		},
		"printSmartDerefType": func(j JSONType) string {
			// really smart
			_, ok := j.(JSONObject)
			if ok {
				return "*" + sp.JSONToGoType(j, false)
			}
			return sp.JSONToGoType(j, false)
		},
		"routesForType": func(routes Routes, j JSONType) []RouteAndIOTypeNames {
			var froutes []RouteAndIOTypeNames
			def := sp.JSONToGoType(j, false)
			for _, route := range routes {
				if route.InType != nil {
					inDef := sp.JSONToGoType(route.InType, false)
					if inDef == def || inDef == "[]"+def {
						froutes = append(froutes, RouteAndIOTypeNames{
							Route:         route,
							InputTypeName: inDef,
						})
					}
				}
				if route.OutType != nil {
					outDef := sp.JSONToGoType(route.OutType, false)
					if outDef == def || outDef == "[]"+def {
						froutes = append(froutes, RouteAndIOTypeNames{
							Route:          route,
							OutputTypeName: outDef,
						})
					}
				}
			}
			sort.Sort(RoutesAndIOTypeNames(froutes))
			return froutes
		},
	}
}

// RouteAndIOTypeNames represents a Route with the names of the types on its input and output.
type RouteAndIOTypeNames struct {
	Route          Route
	InputTypeName  string
	OutputTypeName string
}

type RoutesAndIOTypeNames []RouteAndIOTypeNames

func (routes RoutesAndIOTypeNames) Len() int      { return len(routes) }
func (routes RoutesAndIOTypeNames) Swap(i, j int) { routes[i], routes[j] = routes[j], routes[i] }
func (routes RoutesAndIOTypeNames) Less(i, j int) bool {
	if routes[i].Route.Path == routes[j].Route.Path {
		iIdx, jIdx := -1, -1
		for k, m := range methodsOrder {
			if routes[i].Route.Method == m {
				iIdx = k
			}
			if routes[j].Route.Method == m {
				jIdx = k
			}
		}
		return iIdx < jIdx
	}
	return routes[i].Route.Path < routes[j].Route.Path
}
