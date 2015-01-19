package dispel

import (
	"bytes"
	"fmt"
	"io"
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
	"routes":       routesTmpl,
	"handlers":     handlersTmpl,
	"handlerfuncs": handlerfuncsTmpl,
	"types":        typesTmpl,
}

func tmpl(a asset) string {
	return a.Content
}

type Template struct {
	t *template.Template
}

func (t *Template) ExecuteTemplate(wr io.Writer, name string, ctx *TemplateContext) error {
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
}

func NewTemplate() (*Template, error) {
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
	})
	for name, tmpl := range templatesMap {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("template %s: %v", name, err)
		}
	}
	return &Template{t}, nil
}
