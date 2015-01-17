package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vincent-petithory/dispel"
)

func init() {
}

var (
	schemaFilepath string
	templateName   string
	noGofmt        bool
)

func main() {
	t, err := dispel.NewTemplate()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&schemaFilepath, "schema", "", "JSON Schema file name (relative to this file's dir)")
	flag.StringVar(&templateName, "type", "", fmt.Sprintf("the type of code to generate. One of %q", t.Names()))
	flag.BoolVar(&noGofmt, "no-gofmt", false, "do not run gofmt on the source")

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
	if templateName == "" {
		log.Fatal("no type of code to generate specified")
	}

	// name of the generated file
	dotIndex := strings.LastIndex(goFileName, ".go")
	if dotIndex == -1 {
		log.Fatalf("%s: not a .go file", goFileName)
	}
	genFileName := fmt.Sprintf("%s.gen.%s.go", goFileName[:dotIndex], templateName)
	genPath := filepath.Join(filepath.Dir(absGoFileName), genFileName)

	f, err := os.Open(filepath.Clean(filepath.Join(filepath.Dir(absGoFileName), schemaFilepath)))
	if err != nil {
		log.Fatal(err)
	}

	var schema dispel.Schema
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		_ = f.Close()
		log.Fatal(err)
	}
	_ = f.Close()

	schemaParser := dispel.SchemaParser{RootSchema: &schema}

	handlerFuncDecls, err := dispel.FindTypesFuncs(filepath.Dir(goFileName), goPkgName, []string{"Handler"}, []string{genFileName})
	if err != nil {
		log.Fatal(err)
	}
	var existingHandlers []string
	for name := range handlerFuncDecls {
		existingHandlers = append(existingHandlers, name)
	}

	routes, err := schemaParser.ParseRoutes()
	if err != nil {
		switch t := err.(type) {
		case dispel.InvalidSchemaError:
			log.Fatalf("Schema: %#v\nMsg: %s", t.Schema, t.Msg)
		default:
			log.Fatal(err)
		}
	}

	// Prepare context for template
	ctx := &dispel.TemplateContext{
		Prgm:             strings.Join(append([]string{prgmName}, os.Args[1:]...), " "),
		PkgName:          goPkgName,
		Routes:           routes,
		ExistingHandlers: existingHandlers,
	}

	// Exec template
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, templateName, ctx); err != nil {
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
