package dispel

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/template"
	"unicode"
)

//go:generate -command asset go run ./asset.go
//go:generate asset --var=routesTmpl routes.go.tmpl
//go:generate asset --var=handlersTmpl handlers.go.tmpl
//go:generate asset --var=handlerfuncsTmpl handlerfuncs.go.tmpl
//go:generate asset --var=typesTmpl types.go.tmpl
var templatesMap = map[string]string{
	"routes":       routesTmpl,
	"handlers":     handlersTmpl,
	"handlerfuncs": handlerfuncsTmpl,
	"types":        typesTmpl,
}

func tmpl(a asset) string {
	return a.Content
}

type Template struct {
	t  *template.Template
	sp *SchemaParser
}

func (t *Template) ExecuteTemplate(wr io.Writer, name string, ctx *TemplateContext) error {
	ctx.writtenRefs = make(map[string]bool)
	ctx.sp = t.sp
	return t.t.ExecuteTemplate(wr, name, ctx)
}

func (t *Template) Names() []string {
	var a []string
	for _, tmpl := range t.t.Templates() {
		a = append(a, tmpl.Name())
	}
	return a
}

// TemplateContext represents the context passed to a Template.
type TemplateContext struct {
	Prgm                string
	PkgName             string
	Routes              Routes
	HandlerReceiverType string
	ExistingHandlers    []string

	sp          *SchemaParser
	writtenRefs map[string]bool
	wrm         sync.Mutex
}

func (ctx *TemplateContext) DisplayType(j JSONTypeNamer) string {
	if j.Ref() == "" {
		return fmt.Sprintf("type %s %s", capitalize(j.TypeName()), ctx.sp.JSONToGoType(j, true))
	}

	// Keep track of written types, and don't write them twice.
	ctx.wrm.Lock()
	defer ctx.wrm.Unlock()
	if ok := ctx.writtenRefs[j.Ref()]; ok {
		return ""
	}
	ctx.writtenRefs[j.Ref()] = true
	//// Don't use the name of the type; use it's ref to build it.
	//// Strip leading #/definitions/
	//name := strings.Replace(j.Ref(), "#/definitions/", "", 1)
	//// Hyphenify the remaining ones
	//// TODO allow customize this
	//name = strings.Replace(name, "/definitions/", "-", -1)
	//tn := symbolName(name)
	return fmt.Sprintf("type %s %s", ctx.sp.JSONToGoType(j, false), ctx.sp.JSONToGoType(j, true))
}

func NewTemplate(sp *SchemaParser) (*Template, error) {
	t := template.New("").Funcs(template.FuncMap{
		"tolower":    strings.ToLower,
		"capitalize": capitalize,
		"symbolName": func(s string) string {
			return capitalize(toUpperAfterAny(s, ".- "))
		},
		"hasItem": func(a []string, s string) bool {
			for _, item := range a {
				if s == item {
					return true
				}
			}
			return false

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
		"jsonToGoType": sp.JSONToGoType,
	})
	for name, tmpl := range templatesMap {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("template %s: %v", name, err)
		}
	}
	return &Template{t: t, sp: sp}, nil
}
