package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/vincent-petithory/dispel"
)

var (
	templateNameList    string
	defaultImplNameList string
	prefix              string
	handlerReceiverType string
	pkgpath             string
	pkgname             string
	altFormatPath       string
	altFormatOutPath    string
	verbose             bool
	showVersion         bool
)

//go:generate go run gendoc.go -helpvar
//go:generate go run gendoc.go -godoc

func init() {
	flag.StringVar(&templateNameList, "t", "", "")
	flag.StringVar(&defaultImplNameList, "d", "", "")
	flag.StringVar(&prefix, "p", "dispel_", "")
	flag.StringVar(&handlerReceiverType, "hrt", "", "")
	flag.StringVar(&pkgpath, "pp", "", "")
	flag.StringVar(&pkgname, "pn", "", "")
	flag.StringVar(&altFormatPath, "f", "", "")
	flag.StringVar(&altFormatOutPath, "o", "-", "")
	flag.BoolVar(&verbose, "v", false, "")
	flag.BoolVar(&showVersion, "version", false, "")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: dispel [--version] [-t names] [-d names] [-p prefix] [-hrt typename] [-pp packagepath] [-pn packagename] [-f path] [-o path] [-v] SCHEMA")
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, helptext)
	}
}

// NewSchemaParser creates a new SchemaParser for the json schema at path.
func NewSchemaParser(path string) (*dispel.SchemaParser, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	var schema dispel.Schema
	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		return nil, err
	}

	return &dispel.SchemaParser{RootSchema: &schema}, nil
}

func main() {
	flag.Parse()
	prgmName := filepath.Base(os.Args[0])
	log.SetFlags(0)
	log.SetPrefix(prgmName + ": ")

	if showVersion {
		fmt.Println(dispel.Version)
		return
	}

	// Check envvars from go:generate are set
	var pkgAbsPath string
	switch {
	case pkgpath != "":
		p, err := filepath.Abs(pkgpath)
		if err != nil {
			log.Fatal(err)
		}
		pkgAbsPath = p
	case os.Getenv("GOFILE") != "":
		p, err := filepath.Abs(os.Getenv("GOFILE"))
		if err != nil {
			log.Fatal(err)
		}
		pkgAbsPath = filepath.Dir(p)
	default:
		flag.Usage()
		log.Fatal("no package found: $GOFILE or -pp must be set")
	}
	switch {
	case pkgname != "":
	case os.Getenv("GOPACKAGE") != "":
		pkgname = os.Getenv("GOPACKAGE")
	default:
		flag.Usage()
		log.Fatal("no package name found: $GOPACKAGE or --pn must be set")
	}

	// Abort if the generated files' prefix is empty
	if prefix == "" {
		flag.Usage()
		log.Fatal("generated files need a non-empty prefix")
	}

	// Setting the json schema path is mandatory
	if flag.NArg() < 1 {
		flag.Usage()
		log.Fatal("no jsonschema file provided")
	}
	schemaFilepath := flag.Arg(0)

	// Split comma-separated lists of templates/default impls
	templateNames := strings.Split(templateNameList, ",")
	for i := range templateNames {
		templateNames[i] = strings.TrimSpace(templateNames[i])
	}
	defaultImplNames := strings.Split(defaultImplNameList, ",")
	for i := range defaultImplNames {
		defaultImplNames[i] = strings.TrimSpace(defaultImplNames[i])
	}

	// Parse JSON Schema
	schemaParser, err := NewSchemaParser(schemaFilepath)
	if err != nil {
		log.Fatal(err)
	}
	if verbose {
		schemaParser.Log = log.New(os.Stdout, "dispel> ", 0)
	}

	// Create a dispel bundle  using the parser
	bundle, err := dispel.NewBundle(schemaParser)
	if err != nil {
		log.Fatal(err)
	}

	// Create the list of generated file names
	genPathFn := func(name string) string {
		return filepath.Join(pkgAbsPath, fmt.Sprintf("%s%s.go", prefix, strings.ToLower(name)))
	}

	var genFilenames []string
	for _, tmplName := range bundle.Names() {
		genFilenames = append(genFilenames, filepath.Base(genPathFn(tmplName)))
	}

	// Find methods whose receiver's type is the one defined as holding handler funcs implementations
	// We exclude the ones we auto-generate.
	handlerFuncDecls, err := dispel.FindTypesFuncs(pkgAbsPath, pkgname, []string{strings.Replace(handlerReceiverType, "*", "", -1)}, genFilenames)
	if err != nil {
		log.Fatal(err)
	}
	var existingHandlers []string
	for name := range handlerFuncDecls {
		existingHandlers = append(existingHandlers, name)
	}
	if verbose {
		log.Printf("existing handlers: %s", strings.Join(existingHandlers, "\n --> "))
	}

	// Find types already defined in the package.
	// We exclude the ones we auto-generate.
	typeSpecs, err := dispel.FindTypes(pkgAbsPath, pkgname, genFilenames)
	if err != nil {
		log.Fatal(err)
	}
	var existingTypes []string
	for name := range typeSpecs {
		existingTypes = append(existingTypes, name)
	}
	if verbose {
		log.Printf("existing types: %s", strings.Join(existingTypes, "\n --> "))
	}

	// Parse the routes in the schema
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
	ctx := &dispel.Context{
		Prgm:                fmt.Sprintf("%s v%d", prgmName, dispel.Version),
		PkgName:             pkgname,
		Routes:              routes,
		HandlerReceiverType: handlerReceiverType,
		ExistingHandlers:    existingHandlers,
		ExistingTypes:       existingTypes,
	}

	if altFormatPath != "" {
		var altFormat string
		if altFormatPath == "-" {
			b, err := ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal(err)
			}
			altFormat = string(b)
		} else {
			b, err := ioutil.ReadFile(altFormatPath)
			if err != nil {
				log.Fatal(err)
			}
			altFormat = string(b)
		}

		var out io.Writer
		if altFormatOutPath == "-" {
			out = os.Stdout
		} else {
			f, err := os.Create(altFormatOutPath)
			if err != nil {
				log.Fatal(err)
			}
			defer f.Close()
			out = f
		}
		t, err := dispel.NewTemplate(schemaParser, altFormat)
		if err != nil {
			log.Fatal(err)
		}
		if err := t.Generate(out, ctx); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Exec templates
	if len(templateNames) == 1 && templateNames[0] == "all" {
		templateNames = bundle.Names()
	}
	var buf bytes.Buffer
	for _, name := range templateNames {
		g := bundle.ByName(name)
		if g == nil {
			log.Fatalf("%s: no such generator", name)
		}
		if err := g.Generate(&buf, ctx); err != nil {
			log.Fatal(err)
		}
		// Format source with gofmt
		src, err := format.Source(buf.Bytes())
		if err != nil {
			log.Fatalf("%s\n\ngofmt: %s", buf.Bytes(), err)
		}

		// Write file to disk
		if err := ioutil.WriteFile(genPathFn(name), src, 0666); err != nil {
			log.Fatal(err)
		}
		buf.Reset()
	}

	defaultImpl, err := dispel.NewDefaultImplBundle()
	if err != nil {
		log.Fatal(err)
	}
	buf.Reset()
	if len(defaultImplNames) == 1 && defaultImplNames[0] == "all" {
		defaultImplNames = defaultImpl.Names()
	}
	for _, name := range defaultImplNames {
		if name == "" {
			continue
		}
		if err := defaultImpl.ExecuteTemplate(&buf, name, ctx.Prgm, ctx.PkgName); err != nil {
			log.Fatal(err)
		}
		// Format source with gofmt
		src, err := format.Source(buf.Bytes())
		if err != nil {
			log.Fatalf("%s\n\ngofmt: %s", buf.Bytes(), err)
		}

		// Write file to disk
		destpath := filepath.Join(pkgAbsPath, name+".go")
		if err := ioutil.WriteFile(destpath, src, 0666); err != nil {
			log.Fatal(err)
		}
		buf.Reset()
	}
}
