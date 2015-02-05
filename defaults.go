package dispel

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"sort"
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

var defaultImplsMap = map[string]string{
	DefaultImplMethodHandler:     methodHandler,
	DefaultImplMethodHandlerTest: methodHandlerTest,
	DefaultImplMux:               defaultsMux,
	DefaultImplCodec:             defaultsCodec,
}

// The default implementations available in a DefaultImplBundle.
const (
	DefaultImplMethodHandler     = "methodhandler"
	DefaultImplMethodHandlerTest = "methodhandler_test"
	DefaultImplMux               = "defaults_mux"
	DefaultImplCodec             = "defaults_codec"
)

// DefaultImplBundle represents a bundle of source files
// providing default implementations for dispel's interfaces.
type DefaultImplBundle struct {
	t *template.Template
}

// NewDefaultImplBundle returns a new DefaultImplBundle bundle.
func NewDefaultImplBundle() (*DefaultImplBundle, error) {
	t := template.New("_")
	for name, tmpl := range defaultImplsMap {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("defaultImpl %s: %v", name, err)
		}
	}
	return &DefaultImplBundle{t: t}, nil
}

// ExecuteTemplate writes the source file named by name to the writer wr.
//
// The name must be one of those returned by DefaultImpl.Names().
// It sets the package of the generated source file to pkgName.
func (d *DefaultImplBundle) ExecuteTemplate(wr io.Writer, name string, pkgName string) error {
	return d.t.ExecuteTemplate(wr, name, &struct {
		PkgName string
	}{
		PkgName: pkgName,
	})
}

// Names returns the list of available default implementations.
func (d *DefaultImplBundle) Names() []string {
	var a []string
	for _, tmpl := range d.t.Templates() {
		a = append(a, tmpl.Name())
	}
	return a
}

// DefaultImplNames returns the same list than DefaultImpl.Names(), but doesn't require
// to create a DefaultImpl instance.
func DefaultImplNames() []string {
	var a []string
	for name := range defaultImplsMap {
		a = append(a, name)
	}
	sort.Strings(a)
	return a
}
