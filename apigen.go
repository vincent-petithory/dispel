package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func init() {
	flag.StringVar(&schemaFilepath, "schema", "", "JSON Schema file name (relative to this file's dir)")
	genTypeNames := make([]string, 0, 2)
	for name := range genTypes {
		genTypeNames = append(genTypeNames, name)
	}
	flag.StringVar(&genType, "type", "", fmt.Sprintf("the type of code to generate. One of %q", genTypeNames))
	flag.BoolVar(&noGofmt, "no-gofmt", false, "do not run gofmt on the source")
}

//go:generate -command asset go run ./asset.go
//go:generate asset --var=routesTmpl routes.go.tmpl
//go:generate asset --var=handlersTmpl handlers.go.tmpl
//go:generate asset --var=handlerfuncsTmpl handlerfuncs.go.tmpl
var genTypes = map[string]string{
	"routes":       routesTmpl,
	"handlers":     handlersTmpl,
	"handlerfuncs": handlerfuncsTmpl,
}

func tmpl(a asset) string {
	return a.Content
}

var (
	schemaFilepath string
	genType        string
	noGofmt        bool
)

// walker adapts a function to satisfy the ast.Visitor interface.
// The function return whether the walk should proceed into the node's children.
type walker func(ast.Node) bool

func (w walker) Visit(node ast.Node) ast.Visitor {
	if w(node) {
		return w
	}
	return nil
}

func main() {
	flag.Parse()
	prgmName := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(prgmName + ": ")

	// Check envvars from go:generate are set
	goPkgName := os.Getenv("GOPACKAGE")
	goFileName := os.Getenv("GOFILE")
	if goPkgName == "" {
		log.Fatal("$GOPACKAGE is empty")
	}
	if goFileName == "" {
		log.Fatal("$GOFILE is empty")
	}
	absGoFileName, err := filepath.Abs(goFileName)
	if err != nil {
		log.Fatal(err)
	}

	// Setting the json schema is mandatory
	if schemaFilepath == "" {
		log.Fatal("no jsonschema file provided")
	}
	// Setting the type of code to generate is mandatory
	if genType == "" {
		log.Fatal("no type of code to generate specified")
	}
	_, ok := genTypes[genType]
	if !ok {
		log.Fatalf("%s: unknown gentype", genType)
	}

	// name of the generated file
	dotIndex := strings.LastIndex(goFileName, ".go")
	if dotIndex == -1 {
		log.Fatalf("%s: not a .go file", goFileName)
	}
	genFileName := fmt.Sprintf("%s.gen.%s.go", goFileName[:dotIndex], genType)
	genPath := filepath.Join(filepath.Dir(absGoFileName), genFileName)

	f, err := os.Open(filepath.Clean(filepath.Join(filepath.Dir(absGoFileName), schemaFilepath)))
	if err != nil {
		log.Fatal(err)
	}

	var schema Schema
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		_ = f.Close()
		log.Fatal(err)
	}
	_ = f.Close()

	// Analyse AST of handler package, look for *Handler methods.
	// However, we exclude the ones we generated previously.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, filepath.Dir(goFileName), func(fi os.FileInfo) bool {
		return fi.Name() != genFileName
	}, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}
	handlerPkg, ok := pkgs["handler"]
	if !ok {
		log.Fatalf("%s: package not found in %q", "handler", filepath.Dir(goFileName))
	}

	handlerFuncNames := make([]string, 0, 200)
	for _, astFile := range handlerPkg.Files {
		ast.Walk(walker(func(node ast.Node) bool {
			switch v := node.(type) {
			case *ast.FuncDecl:
				if v.Recv != nil {
					// this is a method, find the type of the receiver
					field := v.Recv.List[0]
					se, ok := field.Type.(*ast.StarExpr)
					if !ok {
						return true
					}
					ident, ok := se.X.(*ast.Ident)
					if !ok {
						return true
					}
					if ident.Name != "Handler" {
						return true
					}
					handlerFuncNames = append(handlerFuncNames, v.Name.String())
				}
			}
			return true
		}), astFile)
	}

	// Compile template
	t := template.New("").Funcs(template.FuncMap{
		"tolower":    strings.ToLower,
		"capitalize": capitalize,
		"symbolName": func(s string) string {
			return afterRuneUpper(s, ".- ")
		},
		"handlerFuncMissing": func(s string) bool {
			for _, handlerFuncName := range handlerFuncNames {
				if s == handlerFuncName {
					return false
				}
			}
			return true

		},
	})
	for name, tmpl := range genTypes {
		var err error
		t, err = t.New(name).Parse(tmpl)
		if err != nil {
			log.Fatalf("template %s: %v", name, err)
		}
	}

	// Prepare context for template
	// Note: we use the same context for all types of templates
	ctx := struct {
		Prgm    string
		PkgName string
		Routes  routes
	}{
		Prgm:    strings.Join(append([]string{prgmName}, os.Args[1:]...), " "),
		PkgName: goPkgName,
		Routes:  parseRoutes(schema),
	}

	// Exec template
	var buf bytes.Buffer
	tmpl := t.Lookup(genType)
	if err := tmpl.Execute(&buf, ctx); err != nil {
		log.Fatal(err)
	}

	// Format source with gofmt
	var src []byte
	if noGofmt {
		src = buf.Bytes()
	} else {
		src, err = format.Source(buf.Bytes())
		if err != nil {
			log.Fatalf("%s\n\ngofmt: %s", buf.Bytes(), err)
		}
	}

	// Write file to disk
	if err := ioutil.WriteFile(genPath, src, 0666); err != nil {
		log.Fatal(err)
	}
}
