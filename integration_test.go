package dispel_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vincent-petithory/dispel"
)

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

func TestGenerateAllFromJSONSchemaNoUserImplWithAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	generateAllFromJSONSchemaNoUserImpl(t, installWithDispelAPI)
}

func TestGenerateAllFromJSONSchemaNoUserImplWithCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	generateAllFromJSONSchemaNoUserImpl(t, installWithDispelCmd)
}

func generateAllFromJSONSchemaNoUserImpl(t *testing.T, installFn func(tb testing.TB, destdir string)) {
	tmpdir, err := ioutil.TempDir("", "dispel-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if t.Failed() {
			t.Logf("Failed in %q", tmpdir)
		} else {
			os.RemoveAll(tmpdir)
		}
	}()

	// Copy testdata tree
	ok(t, filepath.Walk("testdata/rpg", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if fi.IsDir() {
			if path == "testdata/rpg" {
				return nil
			}
			destPath := strings.Replace(path, "testdata/rpg/", "", 1)
			if destPath == "" {
				return nil
			}
			return os.MkdirAll(filepath.Join(tmpdir, destPath), 0777)
		}

		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		destPath := filepath.Join(tmpdir, strings.Replace(path, "testdata/rpg/", "", 1))
		dest, err := os.Create(destPath)
		if err != nil {
			return err
		}
		defer dest.Close()

		_, err = io.Copy(dest, src)
		if err != nil {
			return err
		}

		return err
	}))
	pkgdir := filepath.Join(tmpdir, "src", "rpg")

	installFn(t, pkgdir)

	// Install deps and compile generated project
	_, err = exec.LookPath("go")
	ok(t, err)

	installCmd := exec.Command("go", "install", "-v", "./...")
	installCmd.Dir = pkgdir
	installCmd.Env = []string{}
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GOPATH=") || strings.HasPrefix(env, "GOBIN=") {
			continue
		}
		installCmd.Env = append(installCmd.Env, env)
	}
	installCmd.Env = append(installCmd.Env, "GOPATH="+tmpdir)
	out, err := installCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s\n\ngo install: %v", string(out), err)
	}

	// Control we have the binary
	prgm := filepath.Join(tmpdir, "bin", filepath.Base(pkgdir))
	_, err = os.Stat(prgm)
	ok(t, err)

	// Run the server
	var serverBuf bytes.Buffer
	cmd := exec.Command(prgm, "--addr", "localhost:7777")
	cmd.Stdout = &serverBuf
	cmd.Stderr = &serverBuf
	ok(t, cmd.Start())
	defer func() {
		cmd.Process.Signal(syscall.SIGTERM)
	}()
	// Wait the http server comes up
	time.Sleep(time.Millisecond * 500)

	// Test endpoints return the expected status code, for the default implementations.
	tests := []struct {
		Method string
		Path   string
		Body   []byte
		Code   int
	}{
		{Method: "POST", Path: "/characters", Body: []byte(`{"name": "catmeow"}`), Code: 501},
		{Method: "GET", Path: "/characters", Code: 501},
		{Method: "GET", Path: "/characters/catmeow", Code: 501},
		{Method: "GET", Path: "/characters/catmeow/spells", Code: 404},
		{Method: "PUT", Path: "/characters/luvia/spells/fira", Code: 501},
		{Method: "POST", Path: "/characters/luvia/spells/fira", Code: http.StatusMethodNotAllowed},
		{Method: "PUT", Path: "/characters/luvia/spells/fira", Body: []byte(`{}`), Code: 501},
		{Method: "DELETE", Path: "/characters/vivi/spells/blizzaga", Code: 501},
		{Method: "POST", Path: "/spells", Body: []byte(`{"element": "fire", "name": "fira", "power": 10}`), Code: 501},
		{Method: "POST", Path: "/spells", Body: []byte(`{"element": "fire", "name": "fira", "power": "not an integer"}`), Code: 400},
		{Method: "GET", Path: "/spells", Code: 501},
		{Method: "GET", Path: "/spell", Code: 404},
		{Method: "GET", Path: "/spells/fira", Code: 501},
	}
	for _, test := range tests {
		urlStr := "http://localhost:7777" + test.Path
		var body io.Reader
		if test.Body != nil {
			body = bytes.NewReader(test.Body)
		}
		req, err := http.NewRequest(test.Method, urlStr, body)
		req.Header.Set("Content-Type", "application/json")
		ok(t, err)
		resp, err := http.DefaultClient.Do(req)
		ok(t, err)
		resp.Body.Close()
		assert(t, resp.StatusCode == test.Code, "%s %q responded with %d", test.Method, test.Path, resp.StatusCode)
	}

	go func() {
		cmd.Process.Signal(syscall.SIGTERM)
	}()
	if err := cmd.Wait(); err != nil {
		t.Error(serverBuf.String())
	}
}

func installWithDispelAPI(tb testing.TB, pkgdir string) {
	var schema dispel.Schema
	f, err := os.Open(filepath.Join(pkgdir, "schema.json"))
	ok(tb, err)
	defer f.Close()

	ok(tb, json.NewDecoder(f).Decode(&schema))

	sp := &dispel.SchemaParser{RootSchema: &schema}
	routes, err := sp.ParseRoutes()
	ok(tb, err)

	tmpl, err := dispel.NewTemplateBundle(sp)
	if err != nil {
		log.Fatal(err)
	}

	ctx := &dispel.TemplateContext{
		Prgm:                "dispel",
		PkgName:             "main",
		Routes:              routes,
		HandlerReceiverType: "*App",
		ExistingHandlers:    []string{},
	}

	// Exec templates
	var buf bytes.Buffer
	for _, name := range tmpl.Names() {
		ok(tb, tmpl.ExecuteTemplate(&buf, name, ctx))
		// Format source with gofmt
		src, err := format.Source(buf.Bytes())
		if err != nil {
			tb.Errorf("%s\n\ngofmt: %s", buf.Bytes(), err)
			buf.Reset()
			continue
		}

		// Write file to disk
		ok(tb, ioutil.WriteFile(filepath.Join(pkgdir, fmt.Sprintf("dispel_%s.go", name)), src, 0666))
		buf.Reset()
	}

	// Write defaults
	defaultImpl, err := dispel.NewDefaultImplBundle()
	ok(tb, err)
	for _, name := range defaultImpl.Names() {
		ok(tb, defaultImpl.ExecuteTemplate(&buf, name, ctx.PkgName))
		// Format source with gofmt
		src, err := format.Source(buf.Bytes())
		if err != nil {
			tb.Errorf("%s\n\ngofmt: %s", buf.Bytes(), err)
			buf.Reset()
			continue
		}

		// Write file to disk
		ok(tb, ioutil.WriteFile(filepath.Join(pkgdir, name+".go"), src, 0666))
		buf.Reset()
	}
}

func installWithDispelCmd(tb testing.TB, pkgdir string) {
	installCmd := exec.Command("go", "install", "-v", "github.com/vincent-petithory/dispel/...")
	out, err := installCmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("%s\n\ngo install: %v", string(out), err)
	}

	_, err = exec.LookPath("dispel")
	ok(tb, err)

	dispelCmd := exec.Command(
		"dispel",
		"--handler-receiver-type=*App",
		"--pkgname=main",
		fmt.Sprintf("--pkgpath=%s", pkgdir),
		filepath.Join(pkgdir, "schema.json"),
	)
	out, err = dispelCmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("%s\n\ndispel: %v", string(out), err)
	}
}
