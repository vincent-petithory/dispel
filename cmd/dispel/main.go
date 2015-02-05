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

var (
	templateNameList    string
	defaultImplNameList string
	prefix              string
	handlerReceiverType string
	pkgpath             string
	pkgname             string
	altFormatPath       string
)

func init() {
	flag.StringVar(&templateNameList, "template", "all", fmt.Sprintf("\t\tExecute this template only.\n\t\t\t\tIt must be one of %q. If empty, noone is executed.\n\t\t\t\tIf set to the special value all (the default), all templates are executed.\n\t\t\t\tComma-separated lists are accepted.", dispel.TemplateNames()))
	flag.StringVar(&defaultImplNameList, "default-impl", "", fmt.Sprintf("\t\tExecute this default impl only.\n\t\t\t\tIt must be one of %q. If empty (the default), noone is executed.\n\t\t\t\tIf set to the special value all, all default impls are executed.\n\t\t\t\tComma-separated lists are accepted.", dispel.DefaultImplNames()))
	flag.StringVar(&prefix, "prefix", "dispel_", "\t\tThe prefix to use for each generated template file.\n\t\t\t\tThis doesn't apply to default implementations, which have fixed names.")
	flag.StringVar(&handlerReceiverType, "handler-receiver-type", "", "\tThe type which will receive the handler funcs.")
	flag.StringVar(&pkgpath, "pkgpath", "", "\t\t\tGenerate and analyze code in this package. It is mandatory to set a value if not invoked with go:generate.\n\t\t\t\tIf set when the program is invoked by go:generate, it overrides the package path resolved from $GOFILE.")
	flag.StringVar(&pkgname, "pkgname", "", "\t\t\tThe package name to use at the pkgpath location. It is mandatory to set a value if not invoked with go:generate.\n\t\t\t\tIf set when the program is invoked by go:generate, it overrides the value of $GOPACKAGE")
	flag.StringVar(&altFormatPath, "f", "", "\t\t\tPath to the file for an alternate format for the template to use, using the Go template syntax. The context passed to the template is TemplateContext.\n\t\t\t\tIf the value is -, then the template is read from STDIN.\n\t\t\t\tIf set, then --template and --default-impl flags are ignored: only this template is executed. The result is printed to STDOUT.")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: dispel [OPTION]... SCHEMA")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
	}
}

func parseSchema(path string) (*dispel.SchemaParser, error) {
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
		log.Fatal("no package found: $GOFILE or --pkgpath must be set")
	}
	switch {
	case pkgname != "":
	case os.Getenv("GOPACKAGE") != "":
		pkgname = os.Getenv("GOPACKAGE")
	default:
		flag.Usage()
		log.Fatal("no package name found: $GOPACKAGE or --pkgname must be set")
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
	schemaParser, err := parseSchema(schemaFilepath)
	if err != nil {
		log.Fatal(err)
	}

	// Create dispel template using the parser
	t, err := dispel.NewTemplateBundle(schemaParser)
	if err != nil {
		log.Fatal(err)
	}

	// Create the list of generated file names
	genPathFn := func(name string) string {
		return filepath.Join(pkgAbsPath, fmt.Sprintf("%s%s.go", prefix, strings.ToLower(name)))
	}

	var genFiles []string
	for _, tmplName := range t.Names() {
		genFiles = append(genFiles, genPathFn(tmplName))
	}

	// Find methods whose receiver's type is the one defined as holding handler funcs implementations
	// We exclude the ones we auto-generate.
	handlerFuncDecls, err := dispel.FindTypesFuncs(pkgAbsPath, pkgname, []string{strings.Replace(handlerReceiverType, "*", "", -1)}, genFiles)
	if err != nil {
		log.Fatal(err)
	}
	var existingHandlers []string
	for name := range handlerFuncDecls {
		existingHandlers = append(existingHandlers, name)
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
	ctx := &dispel.TemplateContext{
		Prgm:                strings.Join(append([]string{prgmName}, os.Args[1:]...), " "),
		PkgName:             pkgname,
		Routes:              routes,
		HandlerReceiverType: handlerReceiverType,
		ExistingHandlers:    existingHandlers,
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
		if err := t.ExecuteCustomTemplate(os.Stdout, altFormat, ctx); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Exec templates
	if len(templateNames) == 1 && templateNames[0] == "all" {
		templateNames = t.Names()
	}
	var buf bytes.Buffer
	for _, name := range templateNames {
		if err := t.ExecuteTemplate(&buf, name, ctx); err != nil {
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
		if err := defaultImpl.ExecuteTemplate(&buf, name, ctx.PkgName); err != nil {
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
