package dispel_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/vincent-petithory/dispel"
)

func makeGopathEnv(workspacedir string) []string {
	var envs []string
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "GOPATH=") || strings.HasPrefix(env, "GOBIN=") {
			continue
		}
		envs = append(envs, env)
	}
	envs = append(envs, "GOPATH="+workspacedir)
	return envs
}

// IntegrationTest represents a test which generates a complete Go project
// from a base of Go source files and a json schema, compiles it, runs the compiled program (usually a http server)
// and tests the endpoints of this http server.
type IntegrationTest struct {
	InstallFn func(tb testing.TB, workspacedir string, pkgdir string)
	TestFn    func(tb testing.TB, apiURL *url.URL)
}

func (it *IntegrationTest) Run(tb testing.TB) {
	tmpdir, err := ioutil.TempDir("", "dispel-")
	if err != nil {
		tb.Fatal(err)
	}
	defer func() {
		if tb.Failed() {
			tb.Logf("Failed in %q", tmpdir)
		} else {
			os.RemoveAll(tmpdir)
		}
	}()

	// Copy testdata tree
	pkgdir, err := copyWorkspace(tmpdir)
	if err != nil {
		tb.Error(err)
		return
	}

	// Run install func
	it.InstallFn(tb, tmpdir, pkgdir)

	// Install generated project
	_, err = exec.LookPath("go")
	if err != nil {
		tb.Error(err)
		return
	}

	installCmd := exec.Command("go", "install", "-v", "./...")
	installCmd.Dir = pkgdir
	installCmd.Env = makeGopathEnv(tmpdir)
	out, err := installCmd.CombinedOutput()
	if err != nil {
		tb.Fatalf("%s\n\ngo install: %v", string(out), err)
	}

	// Control we have the binary
	prgm := filepath.Join(tmpdir, "bin", filepath.Base(pkgdir))
	_, err = os.Stat(prgm)
	if err != nil {
		tb.Error(err)
		return
	}

	// Run the server
	var serverBuf bytes.Buffer
	cmd := exec.Command(prgm, "--addr", "localhost:7777")
	cmd.Stdout = &serverBuf
	cmd.Stderr = &serverBuf
	if err := cmd.Start(); err != nil {
		tb.Error(err)
		return
	}
	defer func() {
		cmd.Process.Signal(syscall.SIGTERM)
	}()
	// Wait the http server comes up
	time.Sleep(time.Millisecond * 500)

	apiURL, err := url.Parse("http://localhost:7777")
	if err != nil {
		tb.Error(err)
		return
	}
	// Run tests
	it.TestFn(tb, apiURL)

	go func() {
		cmd.Process.Signal(syscall.SIGTERM)
	}()
	if err := cmd.Wait(); err != nil {
		tb.Errorf("%v\nOutput: %s", err, serverBuf.String())
		return
	}
}

// copyWorkspace copies the test workspace and returns the abs path to the installed pkg.
func copyWorkspace(destdir string) (string, error) {
	workspaceRelpath := "testdata/test-workspace"
	err := filepath.Walk(workspaceRelpath, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if fi.IsDir() {
			if path == workspaceRelpath {
				return nil
			}
			destPath := strings.Replace(path, workspaceRelpath+"/", "", 1)
			if destPath == "" {
				return nil
			}
			return os.MkdirAll(filepath.Join(destdir, destPath), 0777)
		}

		destPath := filepath.Join(destdir, strings.Replace(path, workspaceRelpath+"/", "", 1))
		if err := copyFile(destPath, path); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return filepath.Join(destdir, "src", "github.com", "vincent-petithory", "dispel", "apptest"), nil
}

func copyFile(destPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if err != nil {
		return err
	}

	return nil
}

func TestGenerateAllFromRPGJSONSchemaNoUserImplWithAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	it := &IntegrationTest{
		InstallFn: func(tb testing.TB, workspacedir string, pkgdir string) {
			var schema dispel.Schema
			f, err := os.Open("testdata/rpg.json")
			if err != nil {
				t.Error(err)
				return
			}
			defer f.Close()

			if err := json.NewDecoder(f).Decode(&schema); err != nil {
				t.Error(err)
				return
			}

			sp := &dispel.SchemaParser{RootSchema: &schema}
			routes, err := sp.ParseRoutes()
			if err != nil {
				t.Error(err)
				return
			}

			bundle, err := dispel.NewBundle(sp)
			if err != nil {
				t.Error(err)
				return
			}

			ctx := &dispel.Context{
				Prgm:                "dispel",
				PkgName:             "main",
				Routes:              routes,
				HandlerReceiverType: "*App",
			}

			// Exec templates
			var buf bytes.Buffer
			for _, name := range bundle.Names() {
				if err := bundle.ByName(name).Generate(&buf, ctx); err != nil {
					t.Error(err)
					continue
				}
				// Format source with gofmt
				src, err := format.Source(buf.Bytes())
				if err != nil {
					tb.Errorf("%s\n\ngofmt: %s", buf.Bytes(), err)
					buf.Reset()
					continue
				}

				// Write file to disk
				if err := ioutil.WriteFile(filepath.Join(pkgdir, fmt.Sprintf("dispel_%s.go", name)), src, 0666); err != nil {
					t.Error(err)
					buf.Reset()
					continue
				}
				buf.Reset()
			}

			// Write defaults
			defaultImpl, err := dispel.NewDefaultImplBundle()
			if err != nil {
				t.Error(err)
				return
			}
			for _, name := range defaultImpl.Names() {
				if err := defaultImpl.ExecuteTemplate(&buf, name, ctx.Prgm, ctx.PkgName); err != nil {
					t.Error(err)
					continue
				}
				// Format source with gofmt
				src, err := format.Source(buf.Bytes())
				if err != nil {
					tb.Errorf("%s\n\ngofmt: %s", buf.Bytes(), err)
					buf.Reset()
					continue
				}

				// Write file to disk
				if err := ioutil.WriteFile(filepath.Join(pkgdir, name+".go"), src, 0666); err != nil {
					t.Error(err)
					buf.Reset()
					continue
				}
				buf.Reset()
			}
		},
		TestFn: testRPGSchemaAPINoImpl,
	}
	it.Run(t)
}

func TestGenerateAllFromRPGJSONSchemaNoUserImplWithCmd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	it := &IntegrationTest{
		InstallFn: func(tb testing.TB, workspacedir string, pkgdir string) {
			installDispelCmd := exec.Command("go", "install", "-v", "github.com/vincent-petithory/dispel/...")
			out, err := installDispelCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo install: %v", string(out), err)
			}

			_, err = exec.LookPath("dispel")
			if err != nil {
				t.Error(err)
				return
			}

			dispelCmd := exec.Command(
				"dispel",
				"-t", "all",
				"-hrt", "*App",
				"-d", "all",
				"-pn", "main",
				"-pp", pkgdir,
				"testdata/rpg.json",
			)
			out, err = dispelCmd.CombinedOutput()
			if err != nil {
				tb.Errorf("%s\n\ndispel: %v", string(out), err)
				return
			}
		},
		TestFn: testRPGSchemaAPINoImpl,
	}
	it.Run(t)
}

func TestGenerateAllFromRPGJSONSchemaNoUserImplWithGoGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	it := &IntegrationTest{
		InstallFn: func(tb testing.TB, workspacedir string, pkgdir string) {
			installDispelCmd := exec.Command("go", "install", "-v", "github.com/vincent-petithory/dispel/...")
			out, err := installDispelCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo install: %v", string(out), err)
			}

			_, err = exec.LookPath("dispel")
			if err != nil {
				t.Error(err)
				return
			}

			if err := copyFile(filepath.Join(pkgdir, "schema.json"), "testdata/rpg.json"); err != nil {
				t.Error(err)
				return
			}

			data := fmt.Sprintf("package main\n\n//go:generate %s\n", strings.Join([]string{
				"dispel",
				"-t", "all",
				"-hrt", "*App",
				"-d", "all",
				"schema.json",
			}, " "))

			if err := ioutil.WriteFile(filepath.Join(pkgdir, "dispelgen.go"), []byte(data), 0666); err != nil {
				t.Error(err)
				return
			}

			pkgname, err := filepath.Rel(filepath.Join(workspacedir, "src"), pkgdir)
			if err != nil {
				t.Error(err)
				return
			}
			goGenerateCmd := exec.Command(
				"go", "generate", "-x", pkgname,
			)
			goGenerateCmd.Env = makeGopathEnv(workspacedir)

			out, err = goGenerateCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo:generate: %v", string(out), err)
			}
		},
		TestFn: testRPGSchemaAPINoImpl,
	}
	it.Run(t)
}

func TestGenerateAllFromRPGJSONSchemaNoUserImplWithGoGenerateAndSomeTypesAlreadyDefined(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	it := &IntegrationTest{
		InstallFn: func(tb testing.TB, workspacedir string, pkgdir string) {
			installDispelCmd := exec.Command("go", "install", "-v", "github.com/vincent-petithory/dispel/...")
			out, err := installDispelCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo install: %v", string(out), err)
			}

			_, err = exec.LookPath("dispel")
			if err != nil {
				t.Error(err)
				return
			}

			if err := copyFile(filepath.Join(pkgdir, "schema.json"), "testdata/rpg.json"); err != nil {
				t.Error(err)
				return
			}

			// Override a type generated by dispel
			if err := ioutil.WriteFile(filepath.Join(pkgdir, "types.go"), []byte(`package main
import (
    "time"
)

type Character struct {
    Level int   `+"`"+`json:"level"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Spells []Spell   `+"`"+`json:"spells"`+"`"+`
    CreatedAt time.Time   `+"`"+`json:"createdAt"`+"`"+`
}

`), 0666); err != nil {
				t.Error(err)
				return
			}

			data := fmt.Sprintf("package main\n\n//go:generate %s\n", strings.Join([]string{
				"dispel",
				"-t", "all",
				"-hrt", "*App",
				"-d", "all",
				"schema.json",
			}, " "))

			if err := ioutil.WriteFile(filepath.Join(pkgdir, "dispelgen.go"), []byte(data), 0666); err != nil {
				t.Error(err)
				return
			}

			pkgname, err := filepath.Rel(filepath.Join(workspacedir, "src"), pkgdir)
			if err != nil {
				t.Error(err)
				return
			}
			goGenerateCmd := exec.Command(
				"go", "generate", "-x", pkgname,
			)
			goGenerateCmd.Env = makeGopathEnv(workspacedir)

			out, err = goGenerateCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo:generate: %v", string(out), err)
			}
		},
		TestFn: testRPGSchemaAPINoImpl,
	}
	it.Run(t)
}

func testRPGSchemaAPINoImpl(tb testing.TB, apiURL *url.URL) {
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
		u := &(*apiURL)
		u.Path = test.Path
		var body io.Reader
		if test.Body != nil {
			body = bytes.NewReader(test.Body)
		}
		req, err := http.NewRequest(test.Method, u.String(), body)
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			tb.Error(err)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			tb.Error(err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != test.Code {
			tb.Errorf("%s %q responded with %d", test.Method, test.Path, resp.StatusCode)
		}
	}
}

// TestGenerateAllFromFilesJSONSchemaNoUserImplWithGoGenerate tests an API with non-JSON endpoints,
// and also serves at testing that importing the time is done when a type requires a time.Time.
func TestGenerateAllFromFilesJSONSchemaNoUserImplWithGoGenerate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	it := &IntegrationTest{
		InstallFn: func(tb testing.TB, workspacedir string, pkgdir string) {
			installDispelCmd := exec.Command("go", "install", "-v", "github.com/vincent-petithory/dispel/...")
			out, err := installDispelCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo install: %v", string(out), err)
			}

			_, err = exec.LookPath("dispel")
			if err != nil {
				t.Error(err)
				return
			}

			if err := copyFile(filepath.Join(pkgdir, "schema.json"), "testdata/files.json"); err != nil {
				t.Error(err)
				return
			}

			data := fmt.Sprintf("package main\n\n//go:generate %s\n", strings.Join([]string{
				"dispel",
				"-t", "all",
				"-hrt", "*App",
				"-d", "all",
				"schema.json",
			}, " "))

			if err := ioutil.WriteFile(filepath.Join(pkgdir, "dispelgen.go"), []byte(data), 0666); err != nil {
				t.Error(err)
				return
			}

			pkgname, err := filepath.Rel(filepath.Join(workspacedir, "src"), pkgdir)
			if err != nil {
				t.Error(err)
				return
			}
			goGenerateCmd := exec.Command("go", "generate", "-x", pkgname)
			goGenerateCmd.Env = makeGopathEnv(workspacedir)

			out, err = goGenerateCmd.CombinedOutput()
			if err != nil {
				tb.Fatalf("%s\n\ngo:generate: %v", string(out), err)
			}
		},
		TestFn: testFilesSchemaAPINoImpl,
	}
	it.Run(t)
}

func testFilesSchemaAPINoImpl(tb testing.TB, apiURL *url.URL) {
	// Test endpoints return the expected status code, for the default implementations.
	tests := []struct {
		Method string
		Path   string
		Body   []byte
		Code   int
	}{
		{Method: "POST", Path: "/files", Body: []byte(`hello dispel!`), Code: 501},
		{Method: "GET", Path: "/files", Code: 501},
		{Method: "GET", Path: "/files/560edea", Code: 501},
	}
	for _, test := range tests {
		u := &(*apiURL)
		u.Path = test.Path
		var body io.Reader
		if test.Body != nil {
			body = bytes.NewReader(test.Body)
		}
		req, err := http.NewRequest(test.Method, u.String(), body)
		if err != nil {
			tb.Error(err)
			continue
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			tb.Error(err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != test.Code {
			tb.Errorf("%s %q responded with %d", test.Method, test.Path, resp.StatusCode)
		}
	}
}
