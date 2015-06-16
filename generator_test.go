package dispel

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestTemplateCompiles(t *testing.T) {
	_, err := NewBundle(&SchemaParser{})
	if err != nil {
		t.Error(err)
		return
	}
}

func TestTemplateRoutes(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
    "net/url"
)

// RouteRegisterer is the interface implemented by objects that can register a name for a route path.
type RouteRegisterer interface {
    RegisterRoute(path string, name string)
}

// RouteReverser is the interface implemented by objects that can retrieve the url of a route based on
// its registered name and the route param names and values.
type RouteReverser interface {
    ReverseRoute(name string, params ...string) *url.URL 
}

// RouteLocation is the interface implemented by objects that can return an url for a route, using
// a RouteReverser.
type RouteLocation interface {
	Location(RouteReverser) *url.URL
}

// registerRoutes uses rr to register the routes by path and name.
func registerRoutes(rr RouteRegisterer) {
    rr.RegisterRoute("/spells", routeSpells)
    rr.RegisterRoute("/spells/{spell-name}", routeSpellsOne)
}

// Constants defining the name of all the routes of the API.
const (
    routeSpells = "spells"
    routeSpellsOne = "spells.one"
)

// Types defining the parameters of all the routes of the API.
type (
    // RouteSpells represents the parameters of the path /spells.
    RouteSpells struct{}
    // RouteSpellsOne represents the parameters of the path /spells/{spell-name}.
    RouteSpellsOne struct{
        SpellName string
    }
)

// Location implements building an absolute URL for a RouteSpells using a RouteReverser.
func (r RouteSpells) Location(rr RouteReverser) *url.URL {
    return rr.ReverseRoute(routeSpells)
}
// Location implements building an absolute URL for a RouteSpellsOne using a RouteReverser.
func (r RouteSpellsOne) Location(rr RouteReverser) *url.URL {
    return rr.ReverseRoute(routeSpellsOne, "spell-name", r.SpellName)
}

`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, routesTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
	}
}

func TestTemplateHandlers(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
	"errors"
	"net/http"
)

// HandlerRegisterer is the interface implemented by objects that can register a http handler
// for an http route.
type HandlerRegisterer interface {
    RegisterHandler(routeName string, handler http.Handler)
}

// registerHandlerFunc is an adapter to use funcs as HandlerRegisterer. 
type registerHandlerFunc func(routeName string, handler http.Handler)

// RegisterHandler calls f(routeName, handler).
func (f registerHandlerFunc) RegisterHandler(routeName string, handler http.Handler) {
	f(routeName, handler)
}

// RouteParamGetter is the interface implemented by objects that can retrieve
// the value of a parameter of a route, by name.
type RouteParamGetter interface {
    GetRouteParam(r *http.Request, name string) string
}

// HTTPEncoder is the interface implemented by objects that can encode values to a http response,
// with the specified http status.
//
// Implementors must handle nil data.
type HTTPEncoder interface {
    Encode(w http.ResponseWriter, r *http.Request, data interface{}, code int) error
}

// HTTPDecoder is the interface implemented by objects that can decode data received from a http request.
//
// Implementors have to close the request.Body.
// Decode() shouldn't write to http.ResponseWriter: it's up to the caller to e.g, handle errors.
type HTTPDecoder interface {
    Decode(http.ResponseWriter, *http.Request, interface{}) error
}

// errorHTTPHandlerFunc defines the signature of the generated http handlers used in registerHandlers().
//
// The basic contract of this handler is it write the status code to w (and the body, if any), unless an error is returned;
// in this case, the caller has to write to w.
type errorHTTPHandlerFunc func (w http.ResponseWriter, r *http.Request) (status int, err error)

// registerHandlers registers resource handlers for each unique named route.
// registerHandlers must be called after the registerRoutes().
func registerHandlers(hr HandlerRegisterer, rpg RouteParamGetter, a *App, hd HTTPDecoder, he HTTPEncoder, ehhf func(errorHTTPHandlerFunc) http.Handler) {
	hr.RegisterHandler(routeSpells, &MethodHandler{
		Get: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			status, vresp, err := a.getSpells(w, r)
			if err != nil {
				return status, err
			}
			return status, he.Encode(w, r, vresp, status)
		}),
		Post: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			var vreq Spell
			if err := hd.Decode(w, r, &vreq); err != nil {
				return http.StatusBadRequest, err
			}
			status, vresp, err := a.postSpells(w, r, &vreq)
			if err != nil {
				return status, err
			}
			return status, he.Encode(w, r, vresp, status)
		}),
	})
	hr.RegisterHandler(routeSpellsOne, &MethodHandler{
		Get: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			spellName := rpg.GetRouteParam(r, "spell-name")
			if spellName == "" {
				return http.StatusBadRequest, errors.New("empty route parameter \"spell-name\"")
			}
			status, vresp, err := a.getSpellsOne(w, r, spellName)
			if err != nil {
				return status, err
			}
			return status, he.Encode(w, r, vresp, status)
		}),
	})
}`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, handlersTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
	}
}

func TestTemplateHandlerFuncs(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
    "net/http"
)

func (a *App) getSpells(w http.ResponseWriter, r *http.Request) (int, []Spell, error) {
    return http.StatusNotImplemented, nil, nil
}

func (a *App) postSpells(w http.ResponseWriter, r *http.Request, vreq *Spell) (int, *Spell, error) {
    return http.StatusNotImplemented, nil, nil
}

func (a *App) getSpellsOne(w http.ResponseWriter, r *http.Request, spellName string) (int, *Spell, error) {
    return http.StatusNotImplemented, nil, nil
}

`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, handlerfuncsTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
	}
}

func TestTemplateTypesOneResource(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

// Spell represents the data structure sent/received on the following routes:
//
//  * Response body of GET /spells (as []Spell)
//  * Request body of POST /spells
//  * Response body of POST /spells
//  * Response body of GET /spells/{spell-name}
type Spell struct {
    All bool     `+"`"+`json:"all"`+"`"+`
    Element string `+"`"+`json:"element"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Power int   `+"`"+`json:"power"`+"`"+`
}
`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, typesTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
	}
}

func TestTemplateTypesWithImports(t *testing.T) {
	schema := getSchemaString(t, `{
    "$schema": "http://json-schema.org/draft-04/hyper-schema",
    "title": "Spell",
    "type": "object",
    "definitions": {
        "spell": {
            "definitions": {
                "name": {
                    "type": "string"
                },
                "element": {
                    "type": "string"
                },
                "power": {
                    "type": "integer"
                },
                "all": {
                    "type": "boolean"
                },
                "levelon": {
                    "type": "string",
                    "format": "date-time"
                }
            },
            "links": [
                {
                    "title": "Info for a spell",
                    "href": "/spells/{(#/definitions/spell/definitions/name)}",
                    "method": "GET",
                    "rel": "one",
                    "targetSchema": {
                        "$ref": "#/definitions/spell"
                    }
                }
            ],
            "properties": {
                "name": {
                    "$ref": "#/definitions/spell/definitions/name"
                },
                "element": {
                    "$ref": "#/definitions/spell/definitions/element"
                },
                "power": {
                    "$ref": "#/definitions/spell/definitions/power"
                },
                "all": {
                    "$ref": "#/definitions/spell/definitions/all"
                },
                "level_on": {
                    "$ref": "#/definitions/spell/definitions/levelon"
                }
            }
        }
    },
    "properties": {
        "spell": {
            "$ref": "#/definitions/spell"
        }
    }
}`)
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import "time"

// Spell represents the data structure sent/received on the following routes:
//
//  * Response body of GET /spells/{spell-name}
type Spell struct {
    All bool     `+"`"+`json:"all"`+"`"+`
    Element string `+"`"+`json:"element"`+"`"+`
    LevelOn time.Time    `+"`"+`json:"level_on"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Power int   `+"`"+`json:"power"`+"`"+`
}
`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, typesTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}

func TestTemplateTypesCompositeResources(t *testing.T) {
	schema := getSchema(t, "testdata/rpg.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

// Character represents the data structure sent/received on the following routes:
//
//  * Response body of POST /characters
//  * Response body of GET /characters/{character-name}
type Character struct {
    Level int   `+"`"+`json:"level"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Spells []Spell   `+"`"+`json:"spells"`+"`"+`
}

// CreateCharacterIn represents the data structure sent/received on the following routes:
//
//  * Request body of POST /characters
type CreateCharacterIn struct {
    Name string    `+"`"+`json:"name"`+"`"+`
}

// ListCharacterOutOne represents the data structure sent/received on the following routes:
//
//  * Response body of GET /characters (as []ListCharacterOutOne)
type ListCharacterOutOne struct {
    Level int   `+"`"+`json:"level"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
}

// Spell represents the data structure sent/received on the following routes:
//
//  * Response body of GET /spells (as []Spell)
//  * Request body of POST /spells
//  * Response body of POST /spells
//  * Response body of GET /spells/{spell-name}
type Spell struct {
    Element string `+"`"+`json:"element"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Power int   `+"`"+`json:"power"`+"`"+`
}

`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, typesTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}

func TestAlternateTemplate(t *testing.T) {
	schema := getSchema(t, "testdata/rpg.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	// sample custom template for a basic markdown documentation of the API
	templateText := `{{ range .Routes.ByResource }}## Endpoint {{ .Name }}
{{ $route := . }}{{ range .Methods }}
{{ . }} {{ $route.Path }}
{{ $io := index $route.MethodRouteIOMap . }}
Request: {{ if $io.InType }}

{{ printTypeName $io.InType }}
{{ else }}empty
{{ end }}
Response: {{ if $io.OutType }}

{{ printTypeName $io.OutType }}{{ else }}empty{{ end }}
{{ end }}
{{ end }}`

	expectedOut := `## Endpoint characters

GET /characters

Request: empty

Response: 

[]ListCharacterOutOne

POST /characters

Request: 

CreateCharacterIn

Response: 

Character

## Endpoint characters.one

GET /characters/{character-name}

Request: empty

Response: 

Character

## Endpoint characters.one.spells.one

PUT /characters/{character-name}/spells/{spell-name}

Request: empty

Response: empty

DELETE /characters/{character-name}/spells/{spell-name}

Request: empty

Response: empty

## Endpoint spells

GET /spells

Request: empty

Response: 

[]Spell

POST /spells

Request: 

Spell

Response: 

Spell

## Endpoint spells.one

GET /spells/{spell-name}

Request: empty

Response: 

Spell

`
	tmpl, err := NewTemplate(sp, templateText)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	if expectedOut != buf.String() {
		t.Errorf("expected %#v, got %#v", expectedOut, buf.String())
		return
	}
}

func TestSchemaHasDuplicateRel(t *testing.T) {
	f, err := os.Open("testdata/documents-and-products.json")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	// Modify schema to give it duplicated "rel" attrs
	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		re := regexp.MustCompile(`\s*"rel":\s*"(\w+)-(un)?(link)",`)
		if re.MatchString(line) {
			line = fmt.Sprintf("\"rel\": \"%s\",", re.ReplaceAllString(line, "${2}${3}"))
		}
		fmt.Fprintln(&buf, line)
	}

	schema := getSchemaString(t, buf.String())
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	_, err = sp.ParseRoutes()
	if err == nil {
		t.Error("expected an error, got nil")
		return
	}

	switch e := err.(type) {
	case InvalidSchemaError:
		if !strings.HasPrefix(e.Msg, "duplicate link \"rel\"") {
			t.Errorf("Expected duplicate link rel error msg, got %s", e.Msg)
		}
	default:
		t.Errorf("Got unexpected error %v", err)
	}
}

func TestSchemaHasRedefinedTypes(t *testing.T) {
	f, err := os.Open("testdata/documents-and-products.json")
	if err != nil {
		t.Error(err)
		return
	}
	defer f.Close()
	// Modify schema to give it duplicated "rel" attrs
	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var count int
	for scanner.Scan() {
		line := scanner.Text()
		re := regexp.MustCompile(`\s*"rel":\s*"(\w+)-link",`)
		// We should find it twice: document-link, product-link
		if re.MatchString(line) {
			rel := "link"
			if count > 0 {
				// fake diff rel by capitalizing
				// This will still to colliding named types
				rel = "Link"
			}
			line = fmt.Sprintf("\"rel\": \"%s\",", rel)
			count++
		}
		fmt.Fprintln(&buf, line)
	}

	schema := getSchemaString(t, buf.String())
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	_, err = sp.ParseRoutes()
	if err == nil {
		t.Error("expected an error, got nil")
		return
	}

	switch e := err.(type) {
	case *TypeRedefinitionError:
		candidateRenames := []string{"UnlinkCategoryOut", "LinkCategoryOut"}
		var found bool
		for _, candidate := range candidateRenames {
			if e.Name == candidate {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected redefined type to be one of %q, got %s", candidateRenames, e.Name)
			return
		}
	default:
		t.Errorf("Got unexpected error %v", err)
		return
	}
}

func TestTemplateHandlerFuncsAlreadyDefined(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
		ExistingHandlers:    []string{"getSpells", "postSpells"},
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
    "net/http"
)

func (a *App) getSpellsOne(w http.ResponseWriter, r *http.Request, spellName string) (int, *Spell, error) {
    return http.StatusNotImplemented, nil, nil
}

`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, handlerfuncsTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}

func TestTemplateTypesCompositeResourcesAlreadyDefined(t *testing.T) {
	schema := getSchema(t, "testdata/rpg.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
		ExistingTypes:       []string{"Character", "ListSpellOut", "CharacterSpells", "ListCharacterOut"},
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

// CreateCharacterIn represents the data structure sent/received on the following routes:
//
//  * Request body of POST /characters
type CreateCharacterIn struct {
    Name string    `+"`"+`json:"name"`+"`"+`
}

// ListCharacterOutOne represents the data structure sent/received on the following routes:
//
//  * Response body of GET /characters (as []ListCharacterOutOne)
type ListCharacterOutOne struct {
    Level int   `+"`"+`json:"level"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
}

// Spell represents the data structure sent/received on the following routes:
//
//  * Response body of GET /spells (as []Spell)
//  * Request body of POST /spells
//  * Response body of POST /spells
//  * Response body of GET /spells/{spell-name}
type Spell struct {
    Element string `+"`"+`json:"element"`+"`"+`
    Name string    `+"`"+`json:"name"`+"`"+`
    Power int   `+"`"+`json:"power"`+"`"+`
}

`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, typesTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}

func TestTemplateHandlersWithNonJSONEndpoints(t *testing.T) {
	schema := getSchema(t, "testdata/files.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
	"errors"
	"net/http"
)

// HandlerRegisterer is the interface implemented by objects that can register a http handler
// for an http route.
type HandlerRegisterer interface {
    RegisterHandler(routeName string, handler http.Handler)
}

// registerHandlerFunc is an adapter to use funcs as HandlerRegisterer. 
type registerHandlerFunc func(routeName string, handler http.Handler)

// RegisterHandler calls f(routeName, handler).
func (f registerHandlerFunc) RegisterHandler(routeName string, handler http.Handler) {
	f(routeName, handler)
}

// RouteParamGetter is the interface implemented by objects that can retrieve
// the value of a parameter of a route, by name.
type RouteParamGetter interface {
    GetRouteParam(r *http.Request, name string) string
}

// HTTPEncoder is the interface implemented by objects that can encode values to a http response,
// with the specified http status.
//
// Implementors must handle nil data.
type HTTPEncoder interface {
    Encode(w http.ResponseWriter, r *http.Request, data interface{}, code int) error
}

// HTTPDecoder is the interface implemented by objects that can decode data received from a http request.
//
// Implementors have to close the request.Body.
// Decode() shouldn't write to http.ResponseWriter: it's up to the caller to e.g, handle errors.
type HTTPDecoder interface {
    Decode(http.ResponseWriter, *http.Request, interface{}) error
}

// errorHTTPHandlerFunc defines the signature of the generated http handlers used in registerHandlers().
//
// The basic contract of this handler is it write the status code to w (and the body, if any), unless an error is returned;
// in this case, the caller has to write to w.
type errorHTTPHandlerFunc func (w http.ResponseWriter, r *http.Request) (status int, err error)

// registerHandlers registers resource handlers for each unique named route.
// registerHandlers must be called after the registerRoutes().
func registerHandlers(hr HandlerRegisterer, rpg RouteParamGetter, a *App, hd HTTPDecoder, he HTTPEncoder, ehhf func(errorHTTPHandlerFunc) http.Handler) {
	hr.RegisterHandler(routeFiles, &MethodHandler{
		Get: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			status, vresp, err := a.getFiles(w, r)
			if err != nil {
				return status, err
			}
			return status, he.Encode(w, r, vresp, status)
		}),
		Post: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			status, vresp, err := a.postFiles(w, r)
			if err != nil {
				return status, err
			}
			return status, he.Encode(w, r, vresp, status)
		}),
	})
	hr.RegisterHandler(routeFilesOne, &MethodHandler{
		Get: ehhf(func(w http.ResponseWriter, r *http.Request) (int, error) {
			fileId := rpg.GetRouteParam(r, "file-id")
			if fileId == "" {
				return http.StatusBadRequest, errors.New("empty route parameter \"file-id\"")
			}
			status, err := a.getFilesOne(w, r, fileId)
			if err != nil {
				return status, err
			}
			return status, nil
		}),
	})
}`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, handlersTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}

func TestTemplateHandlerFuncsWithNonJSONEndpoints(t *testing.T) {
	schema := getSchema(t, "testdata/files.json")
	if t.Failed() {
		return
	}
	sp := &SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	ctx := &Context{
		Prgm:                "dispel",
		PkgName:             "handler",
		Routes:              routes,
		HandlerReceiverType: "*App",
	}

	expectedOut, err := format.Source([]byte(fmt.Sprintf(`// generated by %s; DO NOT EDIT

package %s

import (
	"net/http"
)

func (a *App) getFiles(w http.ResponseWriter, r *http.Request) (int, []File, error) {
	return http.StatusNotImplemented, nil, nil
}

func (a *App) postFiles(w http.ResponseWriter, r *http.Request) (int, *File, error) {
	return http.StatusNotImplemented, nil, nil
}

func (a *App) getFilesOne(w http.ResponseWriter, r *http.Request, fileId string) (int, error) {
	http.Error(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
	return http.StatusNotImplemented, nil
}
`, ctx.Prgm, ctx.PkgName)))
	if err != nil {
		t.Error(err)
		return
	}

	tmpl, err := NewTemplate(sp, handlerfuncsTmpl)
	if err != nil {
		t.Error(err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Generate(&buf, ctx); err != nil {
		t.Error(err)
		return
	}
	out, err := format.Source(buf.Bytes())
	if err != nil {
		t.Logf(buf.String())
		t.Error(err)
		return
	}
	if string(expectedOut) != string(out) {
		t.Errorf("expected %#v, got %#v", string(expectedOut), string(out))
		return
	}
}