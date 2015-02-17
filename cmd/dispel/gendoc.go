// +build ignore

package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/vincent-petithory/dispel"
)

var (
	helpvar bool
	godoc   bool
)

func init() {
	flag.BoolVar(&helpvar, "helpvar", false, "generate help var")
	flag.BoolVar(&godoc, "godoc", false, "generate cmd godoc")
}

func main() {
	flag.Parse()

	if !helpvar && !godoc {
		flag.Usage()
		log.Fatal("nothing to do")
	}

	if helpvar {
		if err := generateHelpVar(); err != nil {
			log.Println(err)
		}
	}

	if godoc {
		if err := generateGodoc(); err != nil {
			log.Println(err)
		}
	}
}

func generateHelpVar() error {
	f, err := os.Create("help.go")
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, "package main")
	fmt.Fprint(f, "\nvar helptext = \"")

	vw := NewVarWriter(f)
	if err := executeHelpTemplate(vw); err != nil {
		return err
	}

	fmt.Fprintln(f, "\"")
	return nil
}

func generateGodoc() error {
	var buf bytes.Buffer
	if err := executeHelpTemplate(&buf); err != nil {
		return err
	}

	f, err := os.Create("doc.go")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		text := scanner.Text()
		if text == "" {
			fmt.Fprintln(f, "//")
		} else {
			fmt.Fprintf(f, "// %s\n", scanner.Text())
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	fmt.Fprintln(f, "package main")
	return nil
}

func executeHelpTemplate(w io.Writer) error {
	t, err := template.ParseFiles("help.txt")
	if err != nil {
		return err
	}
	return t.Execute(w, struct {
		TemplateNames    []string
		DefaultImplNames []string
	}{
		TemplateNames:    dispel.TemplateNames(),
		DefaultImplNames: dispel.DefaultImplNames(),
	})
}

func NewVarWriter(w io.Writer) *VarWriter {
	return &VarWriter{w: w}
}

type VarWriter struct {
	w io.Writer
}

func (vw *VarWriter) Write(p []byte) (n int, err error) {
	_, err = vw.w.Write(
		bytes.Replace(
			bytes.Replace(p, []byte("\""), []byte("\\\""), -1),
			[]byte("\n"), []byte("\\n"), -1,
		),
	)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}
