package dispel

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

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
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, pkgName), 0700); err != nil {
		t.Error(err)
		return
	}

	for name, contents := range files {
		if err := ioutil.WriteFile(filepath.Join(tmpDir, pkgName, name), []byte(contents), 0600); err != nil {
			t.Error(err)
			continue
		}
	}

	funcDecls, err := FindTypesFuncs(filepath.Join(tmpDir, pkgName), pkgName, types, []string{"otfuncs.go"})
	if err != nil {
		t.Error(err)
		return
	}
	funcNames := make([]string, 0, len(funcDecls))
	for funcName := range funcDecls {
		funcNames = append(funcNames, funcName)
	}
	sort.Strings(funcNames)
	if !reflect.DeepEqual(expectedFuncs, funcNames) {
		t.Errorf("expected %#v, got %#v", expectedFuncs, funcNames)
		return
	}
}

func TestFindTypes(t *testing.T) {
	pkgName := "main"
	files := map[string]string{
		"t1.go": fmt.Sprintf(`package %s

type Spell struct {
    Name string
    Power int
}

type FooBar struct {
    Foo string
    Bar bool
}

`, pkgName),
		"t2.go": fmt.Sprintf(`package %s

type unexportedT struct {
    X, Y float64
}

type (
    V1 struct {}
    V2 int
    V3 Spell
)

func F() bool {
    return true
}

`, pkgName),
		"exc.go": fmt.Sprintf(`package %s

func (s *Spell) Cast() error {
    panic("forgot wand")
    return nil
}

type X int

const (
    X1 X = iota
    X2
    X3
    X4
)

`, pkgName),
	}

	expectedTypes := []string{
		"Spell",
		"FooBar",
		"unexportedT",
		"V1", "V2", "V3",
	}
	sort.Strings(expectedTypes)

	tmpDir, err := ioutil.TempDir("", "find-types-")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, pkgName), 0700); err != nil {
		t.Error(err)
		return
	}

	for name, contents := range files {
		if err := ioutil.WriteFile(filepath.Join(tmpDir, pkgName, name), []byte(contents), 0600); err != nil {
			t.Error(err)
			continue
		}
	}

	typeSpecs, err := FindTypes(filepath.Join(tmpDir, pkgName), pkgName, []string{"exc.go"})
	if err != nil {
		t.Error(err)
		return
	}
	typeNames := make([]string, 0, len(typeSpecs))
	for typeName := range typeSpecs {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)
	if !reflect.DeepEqual(expectedTypes, typeNames) {
		t.Errorf("expected %#v, got %#v", expectedTypes, typeNames)
		return
	}
}
