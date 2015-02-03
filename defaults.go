package dispel

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"strings"
	"text/template"
)

//go:generate -command asset go run ./asset.go
//go:generate asset --var=methodHandler --wrap=gofmtTmpl methodhandler.go
//go:generate asset --var=methodHandlerTest --wrap=gofmtTmpl methodhandler_test.go
//go:generate asset --var=defaultsMux --wrap=gofmtTmpl defaults_mux.go
//go:generate asset --var=defaultsCodec --wrap=gofmtTmpl defaults_codec.go

func gofmtTmpl(a asset) string {
	b, err := format.Source([]byte(a.Content))
	if err != nil {
		panic(err)
	}
	// Remove build tags and prepare package template
	var buf bytes.Buffer
	scanner := bufio.NewScanner(bytes.NewReader(b))
	scanner.Split(bufio.ScanLines)
	var skipNextLine bool
	for scanner.Scan() {
		if skipNextLine {
			skipNextLine = false
			continue
		}
		text := scanner.Text()
		if strings.HasPrefix(text, "// +build") {
			skipNextLine = true
			continue
		}
		if strings.HasPrefix(text, "package ") {
			_, _ = fmt.Fprintln(&buf, "package {{ .PkgName }}")
			continue
		}
		_, _ = fmt.Fprintln(&buf, text)
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return buf.String()
}

var defaultsMap = map[string]string{
	"methodhandler":      methodHandler,
	"methodhandler_test": methodHandlerTest,
	"defaults_mux":       defaultsMux,
	"defaults_codec":     defaultsCodec,
}

type DefaultImpl struct {
	t *template.Template
}

func NewDefaultImpl() (*DefaultImpl, error) {
	t := template.New("_")
	for name, tmpl := range defaultsMap {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("defaultImpl %s: %v", name, err)
		}
	}
	return &DefaultImpl{t: t}, nil
}

func (d *DefaultImpl) ExecuteTemplate(wr io.Writer, name string, pkgName string) error {
	return d.t.ExecuteTemplate(wr, name, &struct {
		PkgName string
	}{
		PkgName: pkgName,
	})
}

func (d *DefaultImpl) Names() []string {
	var a []string
	for _, tmpl := range d.t.Templates() {
		a = append(a, tmpl.Name())
	}
	return a
}

func DefaultNames() []string {
	var a []string
	for name := range defaultsMap {
		a = append(a, name)
	}
	return a
}
