package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"testing"
)

func init() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("\033[31m%s:%d: "+msg+"\033[39m\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if expected is not equal to actual.
func equals(tb testing.TB, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		_, file, line, _ := runtime.Caller(1)
		log.Printf("\033[31m%s:%d:\n\n\texpected: %#v\n\n\tgot: %#v\033[39m\n", filepath.Base(file), line, expected, actual)
		tb.FailNow()
	}
}

func TestFindTypesFuncs(t *testing.T) {
	pkgName := "funcs"
	types := []string{"FuncGroup1", "funcGroup2"}
	files := map[string]string{
		"funcs1.go": fmt.Sprintf(`package %s

type FuncGroup1 struct {}
func (fg *FuncGroup1) F11() {}
func (fg *FuncGroup1) F12() {}
func (fg *FuncGroup1) F13() {}

func something() {}
`, pkgName),
		"funcs2.go": fmt.Sprintf(`package %s

type funcGroup2 struct {}
func (fg funcGroup2) F21() {}
func (fg funcGroup2) F22() {}

func (fg FuncGroup) OTF3() {}
`, pkgName),
		"otfuncs.go": fmt.Sprintf(`package %s

type OTFuncGroup struct {}
func (fg FuncGroup) OTF1() {}
func (fg FuncGroup) OTF2() {}

func SF1() {}

func (fg funcGroup2) F23() {}
`, pkgName),
	}

	expectedFuncs := []string{"F11", "F12", "F13", "F21", "F22"}
	sort.Strings(expectedFuncs)

	tmpDir, err := ioutil.TempDir("", "find-types-funcs-")
	ok(t, err)
	defer os.RemoveAll(tmpDir)

	ok(t, os.Mkdir(filepath.Join(tmpDir, pkgName), 0700))

	for name, contents := range files {
		ok(t, ioutil.WriteFile(filepath.Join(tmpDir, pkgName, name), []byte(contents), 0600))
	}

	funcDecls, err := findTypesFuncs(filepath.Join(tmpDir, pkgName), pkgName, types, []string{"otfuncs.go"})
	ok(t, err)
	funcNames := make([]string, 0, len(funcDecls))
	for funcName := range funcDecls {
		funcNames = append(funcNames, funcName)
	}
	sort.Strings(funcNames)
	equals(t, expectedFuncs, funcNames)
}