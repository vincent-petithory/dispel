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

type NamedGenerator struct {
	Name string
	Generator
}

type Bundle struct {
	Routes       NamedGenerator
	Handlers     NamedGenerator
	Handlerfuncs NamedGenerator
	Types        NamedGenerator
}

func NewBundle(sp *SchemaParser) (*Bundle, error) {
	routes, err := NewTemplate(sp, routesTmpl)
	if err != nil {
		return nil, err
	}
	handlers, err := NewTemplate(sp, handlersTmpl)
	if err != nil {
		return nil, err
	}
	handlerfuncs, err := NewTemplate(sp, handlerfuncsTmpl)
	if err != nil {
		return nil, err
	}
	types, err := NewTemplate(sp, typesTmpl)
	if err != nil {
		return nil, err
	}
	return &Bundle{
		Routes:       NamedGenerator{Generator: routes, Name: "routes"},
		Handlers:     NamedGenerator{Generator: handlers, Name: "handlers"},
		Handlerfuncs: NamedGenerator{Generator: handlerfuncs, Name: "handlerfuncs"},
		Types:        NamedGenerator{Generator: types, Name: "types"},
	}, nil
}

func (b *Bundle) Names() []string {
	names := []string{
		b.Routes.Name,
		b.Handlers.Name,
		b.Handlerfuncs.Name,
		b.Types.Name,
	}
	sort.Strings(names)
	return names
}

func (b *Bundle) ByName(name string) Generator {
	switch name {
	case b.Routes.Name:
		return b.Routes
	case b.Handlers.Name:
		return b.Handlers
	case b.Handlerfuncs.Name:
		return b.Handlerfuncs
	case b.Types.Name:
		return b.Types
	default:
		return nil
	}
}

type Generator interface {
	Generate(wr io.Writer, ctx *Context) error
}

func tmpl(a asset) string {
	return a.Content
}

// Context represents the context passed to a Generator.
type Context struct {
	Schema              *SchemaParser // the SchemaParser which parsed the json schema
	Prgm                string        // name of the program generating the source
	PkgName             string        // package name for which source code is generated
	Routes              Routes        // routes parsed by the SchemaParser
	HandlerReceiverType string        // type which acts as the receiver of the handler funcs.
	ExistingHandlers    []string      // list of existing handler funcs in the target package, with HandlerReceiverType as the receiver
	ExistingTypes       []string      // list of existing types in the target package.
}

func handlerFuncName(routeMethod string, routeName string) string {
	return strings.ToLower(routeMethod) + symbolName(routeName)
}

// NewTemplate returns a new Template based on the SchemaParser using text.
func NewTemplate(sp *SchemaParser, text string) (*Template, error) {
	t, err := template.New("").Funcs(NewTemplateFuncMap(sp)).Parse(text)
	if err != nil {
		return nil, err
	}
	return &Template{T: t, Schema: sp}, nil
}

type Template struct {
	T      *template.Template
	Schema *SchemaParser

	name string
}

// Name returns the name of the template.
func (t *Template) Name() string {
	return t.name
}

// Generate implements the Generator interface.
// It executes the template with the ctx, and writes the output to w. The ctx is forced to use the Template's SchemaParser.
func (t *Template) Generate(w io.Writer, ctx *Context) error {
	ctx.Schema = t.Schema
	return t.T.Execute(w, ctx)
}

// NewTemplateFuncMap returns a template.FuncMap useful for building dispel text templates.
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
