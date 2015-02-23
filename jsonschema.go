package dispel

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"reflect"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Schema represents a JSON Hyper Schema.
type Schema struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`

	Default  interface{} `json:"default,omitempty"`
	ReadOnly bool        `json:"readOnly,omitempty"` // unsupported
	Example  interface{} `json:"example,omitempty"`
	Format   string      `json:"format,omitempty"` // unsupported

	Type string `json:"type,omitempty"`

	Ref    string `json:"$ref,omitempty"`
	Schema string `json:"$schema,omitempty"` // unsupported

	Definitions map[string]*Schema `json:"definitions,omitempty"`

	MultipleOf       float64 `json:"multipleOf,omitempty"`       // unsupported
	Maximum          float64 `json:"maximum,omitempty"`          // unsupported
	ExclusiveMaximum bool    `json:"exclusiveMaximum,omitempty"` // unsupported
	Minimum          float64 `json:"minimum,omitempty"`          // unsupported
	ExclusiveMinimum bool    `json:"exclusiveMinimum,omitempty"` // unsupported

	MinLength int    `json:"minLength,omitempty"` // unsupported
	MaxLength int    `json:"maxLength,omitempty"` // unsupported
	Pattern   string `json:"pattern,omitempty"`   // unsupported

	MinProperties        int                    `json:"minProperties,omitempty"` // unsupported
	MaxProperties        int                    `json:"maxProperties,omitempty"` // unsupported
	Required             []string               `json:"required,omitempty"`      // unsupported
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty"`         // unsupported
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"` // unsupported
	PatternProperties    map[string]*Schema     `json:"patternProperties,omitempty"`    // unsupported

	Items           *Schema     `json:"items,omitempty"`
	MinItems        int         `json:"minItems,omitempty"`        // unsupported
	MaxItems        int         `json:"maxItems,omitempty"`        // unsupported
	UniqueItems     bool        `json:"uniqueItems,omitempty"`     // unsupported
	AdditionalItems interface{} `json:"additionalItems,omitempty"` // unsupported

	Enum []string `json:"enum,omitempty"` // unsupported

	OneOf []Schema `json:"oneOf,omitempty"` // unsupported
	AnyOf []Schema `json:"anyOf,omitempty"` // unsupported
	AllOf []Schema `json:"allOf,omitempty"` // unsupported
	Not   *Schema  `json:"not,omitempty"`   // unsupported

	Links []Link `json:"links,omitempty"`
}

// Link represents a Link description.
type Link struct {
	Title        string  `json:"title,omitempty"`
	Description  string  `json:"description,omitempty"`
	HRef         string  `json:"href,omitempty"`
	Rel          string  `json:"rel,omitempty"`
	Method       string  `json:"method,omitempty"`
	Schema       *Schema `json:"schema,omitempty"`
	TargetSchema *Schema `json:"targetSchema,omitempty"`
	EncType      string  `json:"encType,omitempty"`
	MediaType    string  `json:"mediaType,omitempty"`
}

func (l *Link) ApplyDefaults() {
	if l.EncType == "" {
		l.EncType = "application/json"
	}
	if l.MediaType == "" {
		l.MediaType = "application/json"
	}
}

func (l Link) ReceivesJSON() bool {
	return strings.HasPrefix(l.EncType, "application/json")
}

func (l Link) SendsJSON() bool {
	return strings.HasPrefix(l.MediaType, "application/json")
}

// Route represents an HTTP endpoint for a resource, with JSON on the wire.
type Route struct {
	Path        string
	Name        string
	RouteParams []RouteParam
	Method      string
	RouteIO
}

// RouteIO represents JSON types for input and output.
type RouteIO struct {
	InputIsNotJSON  bool
	OutputIsNotJSON bool
	// InType is the JSON type coming in.
	InType JSONType
	// OutType is the JSON type coming out.
	OutType JSONType
}

// RouteParam represents a variable chunk in an HTTP endpoint path.
type RouteParam struct {
	Name    string
	Varname string
	Type    JSONType
}

// Routes represents a list of Routes.
type Routes []Route

func (routes Routes) Len() int           { return len(routes) }
func (routes Routes) Swap(i, j int)      { routes[i], routes[j] = routes[j], routes[i] }
func (routes Routes) Less(i, j int) bool { return routes[i].Path < routes[j].Path }

// ByResource map-reduces Routes to a ResourceRoutes.
func (routes Routes) ByResource() ResourceRoutes {
	resourceRouteMap := make(map[string]*ResourceRoute)
	for _, route := range routes {
		resourceRoute, ok := resourceRouteMap[route.Path]
		if !ok {
			resourceRoute = &ResourceRoute{
				Path:             route.Path,
				Name:             route.Name,
				RouteParams:      route.RouteParams,
				MethodRouteIOMap: MethodRouteIOMap{route.Method: route.RouteIO},
			}
			resourceRouteMap[route.Path] = resourceRoute
			continue
		}
		resourceRoute.MethodRouteIOMap[route.Method] = route.RouteIO
	}
	resourceRoutes := make(ResourceRoutes, 0, len(resourceRouteMap))
	for _, resourceRoute := range resourceRouteMap {
		resourceRoutes = append(resourceRoutes, *resourceRoute)
	}
	sort.Sort(resourceRoutes)
	return resourceRoutes
}

func (routes Routes) walkType(typ JSONType, walkFn func(JSONType)) {
	walkFn(typ)
	switch j := typ.(type) {
	case JSONObject:
		for _, field := range j.Fields {
			routes.walkType(field.Type, walkFn)
		}
	case JSONArray:
		routes.walkType(j.Items, walkFn)
	}
}

// JSONNamedTypes returns a list of all unique types found in the Routes.
func (routes Routes) JSONNamedTypes() []JSONTypeNamer {
	visited := make(map[string]bool)
	var a []JSONTypeNamer

	for _, route := range routes {
		for _, typ := range []JSONType{route.InType, route.OutType} {
			if typ == nil {
				continue
			}
			routes.walkType(typ, func(jt JSONType) {
				jtn, ok := jt.(JSONTypeNamer)
				if !ok {
					return
				}
				tn := jtn.TypeName()
				// Skip anonymous types
				if tn == "" {
					log.Panicf("no unnamed type should exist")
					return
				}
				if ok := visited[tn]; !ok {
					visited[tn] = true
					a = append(a, jtn)
				}
			})
		}
	}
	sort.Sort(byTypeName(a))
	return a
}

// byTypeName implements sorting by TypeName for a slice of JSONTypeNamer objects.
type byTypeName []JSONTypeNamer

func (btn byTypeName) Len() int           { return len(btn) }
func (btn byTypeName) Swap(i, j int)      { btn[i], btn[j] = btn[j], btn[i] }
func (btn byTypeName) Less(i, j int) bool { return btn[i].TypeName() < btn[j].TypeName() }

// JSONTypeNamer combines the TypeNamer and JSONType interfaces.
type JSONTypeNamer interface {
	TypeNamer
	JSONType
}

// MethodRouteIOMap maps a method to a RouteIO.
type MethodRouteIOMap map[string]RouteIO

// Methods is a list of HTTP methods.
type Methods []string

func (m Methods) Len() int      { return len(m) }
func (m Methods) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m Methods) Less(i, j int) bool {
	var (
		ii = -1
		ji = -1
	)
	for k, um := range methodsOrder {
		if strings.ToUpper(m[i]) == um {
			ii = k
		}
		if strings.ToUpper(m[j]) == um {
			ji = k
		}
		if ii > -1 && ji > -1 {
			break
		}
	}
	return ii < ji
}

var methodsOrder = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

// ResourceRoute represents a set of routes related to a resource, in the scope of a RESTful design.
type ResourceRoute struct {
	Path             string
	Name             string
	RouteParams      []RouteParam
	MethodRouteIOMap MethodRouteIOMap
}

// Methods lists the available HTTP methods on the resource.
func (resourceRoutes *ResourceRoute) Methods() Methods {
	var methods Methods
	for method := range resourceRoutes.MethodRouteIOMap {
		methods = append(methods, method)
	}
	sort.Sort(methods)
	return methods
}

// ResourceRoutes is a list of resource routes.
type ResourceRoutes []ResourceRoute

func (r ResourceRoutes) Len() int           { return len(r) }
func (r ResourceRoutes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r ResourceRoutes) Less(i, j int) bool { return r[i].Name < r[j].Name }

func href2path(href string) (string, error) {
	var firstErr error
	p, err := mapHRefVar(href, func(v string) string {
		if v[0] != '{' && v[len(v)-1] != '}' {
			return ""
		}
		v = v[1 : len(v)-1]
		uv, err := unescapePctEnc(v)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return ""
		}

		// Strip leading #/definitions/
		v = strings.Replace(string(uv), "#/definitions/", "", 1)
		// Hyphenify the remaining ones
		// TODO allow customize this
		v = strings.Replace(v, "/definitions/", "-", -1)

		return fmt.Sprintf("{%s}", v)
	})
	if err != nil {
		return "", err
	}
	if firstErr != nil {
		return "", firstErr
	}
	return p, nil
}

func href2name(href string) (string, error) {
	const varnameRepl = "one"
	var firstErr error
	name, err := mapHRefVar(href, func(v string) string {
		return varnameRepl
	})
	if err != nil {
		return "", err
	}
	if firstErr != nil {
		return "", firstErr
	}
	if strings.HasPrefix(name, "/") {
		name = name[1:]
	}
	name = strings.Replace(name, "/", ".", -1)
	return name, nil
}

func preProcessHRefVar(v string) string {
	if len(v) < 2 {
		return v
	}
	if v[0] != '{' && v[len(v)-1] != '}' {
		return v
	}
	v = v[1 : len(v)-1]
	var (
		escbuf     bytes.Buffer
		buf        bytes.Buffer
		inEscBlock bool
	)

	reader := strings.NewReader(v)
	for {
		r, _, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		switch r {
		case '(':
			if !inEscBlock {
				inEscBlock = true
			} else {
				_, _ = escbuf.WriteRune(r)
			}
		case ')':
			r2, _, err := reader.ReadRune()
			if err != io.EOF {
				if r2 != ')' {
					_ = reader.UnreadRune()
				} else {
					_, _ = escbuf.WriteRune(r2)
					continue
				}
			}

			if inEscBlock {
				inEscBlock = false
			}
			// Escape string
			v := rfc6570Escape(escbuf.Bytes())
			escbuf.Reset()
			if v == "" {
				v = "%65mpty"
			}
			_, _ = io.WriteString(&buf, v)
		default:
			if inEscBlock {
				_, _ = escbuf.WriteRune(r)
			} else {
				_, _ = buf.WriteRune(r)
			}
		}
	}
	return fmt.Sprintf("{%s}", buf.String())
}

func rfc6570Escape(data []byte) string {
	var buf = new(bytes.Buffer)
	for _, b := range data {
		switch {
		case (b >= 'a' && b <= 'z') ||
			(b >= 'A' && b <= 'Z') ||
			(b >= '0' && b <= '9') ||
			b == '_' ||
			b == '.':
			_ = buf.WriteByte(b)
		default:
			fmt.Fprintf(buf, "%%%X", b)
		}
	}
	return buf.String()
}

func isHex(c byte) bool {
	switch {
	case c >= 'a' && c <= 'f':
		return true
	case c >= 'A' && c <= 'F':
		return true
	case c >= '0' && c <= '9':
		return true
	}
	return false
}

// borrowed from net/url/url.go
func unhex(c byte) byte {
	switch {
	case '0' <= c && c <= '9':
		return c - '0'
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10
	case 'A' <= c && c <= 'F':
		return c - 'A' + 10
	}
	return 0
}

func unescapePctEnc(s string) ([]byte, error) {
	var buf = new(bytes.Buffer)
	reader := strings.NewReader(s)

	for {
		r, size, err := reader.ReadRune()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if size > 1 {
			return nil, fmt.Errorf("non-ASCII char detected")
		}

		switch r {
		case '%':
			eb1, err := reader.ReadByte()
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected end of unescape sequence")
			}
			if err != nil {
				return nil, err
			}
			if !isHex(eb1) {
				return nil, fmt.Errorf("invalid char 0x%x in unescape sequence", r)
			}
			eb0, err := reader.ReadByte()
			if err == io.EOF {
				return nil, fmt.Errorf("unexpected end of unescape sequence")
			}
			if err != nil {
				return nil, err
			}
			if !isHex(eb0) {
				return nil, fmt.Errorf("invalid char 0x%x in unescape sequence", r)
			}
			_ = buf.WriteByte(unhex(eb0) + unhex(eb1)*16)
		default:
			_ = buf.WriteByte(byte(r))
		}
	}
	return buf.Bytes(), nil
}

func varsFromHRef(href string) ([]string, error) {
	var vars []string
	var firstErr error
	_, err := mapHRefVar(href, func(v string) string {
		if v[0] != '{' && v[len(v)-1] != '}' {
			vars = append(vars, v)
			return ""
		}
		v = v[1 : len(v)-1]
		uv, err := unescapePctEnc(v)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return ""
		}
		vars = append(vars, string(uv))
		return ""
	})
	if err != nil {
		return nil, err
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return vars, err
}

// mapHRefVar runs varFunc on each variable in href. It can also serve as a for-each.
func mapHRefVar(href string, varFunc func(string) string) (string, error) {
	var (
		varbuf bytes.Buffer
		inVar  bool
		buf    bytes.Buffer
	)
	for _, r := range href {
		switch r {
		case '{':
			if inVar {
				return "", fmt.Errorf("varsFromHRef: %q: found opening { while already in a var", href)
			}
			inVar = true
			_, _ = varbuf.WriteRune(r)
		case '}':
			if !inVar {
				return "", fmt.Errorf("varsFromHRef: %q: found closing } while not in a var", href)
			}
			inVar = false
			_, _ = varbuf.WriteRune(r)

			ppv := preProcessHRefVar(varbuf.String())
			varbuf.Reset()

			v := varFunc(ppv)
			_, _ = io.WriteString(&buf, v)
		default:
			if inVar {
				// TODO should check r is an allowed char (href substitution variable)
				// when we also support URI template operators.
				_, _ = varbuf.WriteRune(r)
			} else {
				_, _ = buf.WriteRune(r)
			}
		}
	}
	return buf.String(), nil
}

func capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return fmt.Sprintf("%c%s", unicode.ToUpper(r), s[size:])
}

func symbolName(s string) string {
	return capitalize(toUpperAfterAny(s, ".-_ "))
}

func ref2name(ref string) string {
	// Strip leading #/definitions/
	name := strings.Replace(ref, "#/definitions/", "", 1)
	// Hyphenify the remaining ones
	name = strings.Replace(name, "/definitions/", "-", -1)
	return symbolName(name)
}

func toUpperAfterAny(s string, chars string) string {
	var buf bytes.Buffer
	var upnext bool
OuterLoop:
	for _, r := range s {
		for _, c := range chars {
			if r == c {
				upnext = true
				continue OuterLoop
			}
		}
		if upnext {
			_, _ = buf.WriteRune(unicode.ToUpper(r))
			upnext = false
			continue
		}
		_, _ = buf.WriteRune(r)
	}
	return buf.String()
}

// json types

// JSONObject represents the object primitive type of the JSON format.
type JSONObject struct {
	Name   string
	Fields JSONFieldList
	ref    string
}

// Type implements Type() of the JSONType interface.
func (o JSONObject) Type() string {
	return "object"
}

// Ref implements Ref() of the JSONType interface.
func (o JSONObject) Ref() string {
	return o.ref
}

// TypeName implements the TypeNamer interface.
func (o JSONObject) TypeName() string {
	return o.Name
}

// JSONArray represents the array primitive type of the JSON format.
type JSONArray struct {
	Name  string
	ref   string
	Items JSONType
}

// Type implements Type() of the JSONType interface.
func (a JSONArray) Type() string {
	return "array"
}

// Ref implements Ref() of the JSONType interface.
func (a JSONArray) Ref() string {
	return a.ref
}

// TypeName implements the TypeNamer interface.
func (a JSONArray) TypeName() string {
	return a.Name
}

// JSONString represents the string primitive type of the JSON format.
type JSONString struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (s JSONString) Type() string {
	return "string"
}

// Ref implements Ref() of the JSONType interface.
func (s JSONString) Ref() string {
	return s.ref
}

// JSONBoolean represents the boolean primitive type of the JSON format.
type JSONBoolean struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (b JSONBoolean) Type() string {
	return "boolean"
}

// Ref implements Ref() of the JSONType interface.
func (b JSONBoolean) Ref() string {
	return b.ref
}

// JSONInteger represents the integer primitive type of the JSON format.
type JSONInteger struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (i JSONInteger) Type() string {
	return "integer"
}

// Ref implements Ref() of the JSONType interface.
func (i JSONInteger) Ref() string {
	return i.ref
}

// JSONNumber represents the number primitive type of the JSON format.
type JSONNumber struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (n JSONNumber) Type() string {
	return "number"
}

// Ref implements Ref() of the JSONType interface.
func (n JSONNumber) Ref() string {
	return n.ref
}

// JSONNull represents the null primitive type of the JSON format.
type JSONNull struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (n JSONNull) Type() string {
	return "null"
}

// Ref implements Ref() of the JSONType interface.
func (n JSONNull) Ref() string {
	return n.ref
}

// JSONDateTime represents the string primitive type of the JSON format.
// Its underlying time is a RFC3339 date time.
type JSONDateTime struct {
	ref string
}

// Type implements Type() of the JSONType interface.
func (dt JSONDateTime) Type() string {
	return "date-time"
}

// Ref implements Ref() of the JSONType interface.
func (dt JSONDateTime) Ref() string {
	return dt.ref
}

// JSONField represents one property of a JSONObject.
type JSONField struct {
	Name string
	Type JSONType
}

// JSONFieldList implements alphabetical sorting of a JSONObject property list.
type JSONFieldList []JSONField

func (fl JSONFieldList) Len() int           { return len(fl) }
func (fl JSONFieldList) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl JSONFieldList) Less(i, j int) bool { return fl[i].Name < fl[j].Name }

// JSONType is the interface implemented by objects that represents a JSON type defined in a JSON Schema.
type JSONType interface {
	// Type returns a string representation of this type.
	Type() string
	// Ref returns the absolute reference of type as found in the JSON Schema which defined it.
	Ref() string
}

// JSONToGoType prints Go source code for the JSONType.
// globalScope tells whether we are in the package scope of the go source file,
// or inside a type.
func (sp *SchemaParser) JSONToGoType(jt JSONType, globalScope bool) string {
	ref := jt.Ref()
	if ref != "" {
		tjt, ok := sp.RefJSONTypeMap[ref]
		if !ok {
			log.Panicf("unregistered json type %s", ref)
		}
		jt = tjt
	}
	// if we're not in the package scope, we'll just write the name of
	// the type, if it has one.
	n, ok := jt.(TypeNamer)
	if ok && !globalScope {
		if n.TypeName() == "" {
			log.Panicf("no unnamed type should exist")
		}
		// we don't want a type with a slice as underlying type.
		// See https://golang.org/ref/spec#Assignability
		a, ok := jt.(JSONArray)
		if ok {
			return fmt.Sprintf("[]%s", sp.JSONToGoType(a.Items, false))
		}
		return symbolName(n.TypeName())
	}
	switch j := jt.(type) {
	case JSONString:
		return "string"
	case JSONDateTime:
		return "time.Time"
	case JSONBoolean:
		return "bool"
	case JSONInteger:
		return "int"
	case JSONNumber:
		return "float64"
	case JSONObject:
		var buf bytes.Buffer
		_, _ = buf.WriteString("struct {\n")
		for _, f := range j.Fields {
			// we don't want a type with a slice as underlying type.
			// See https://golang.org/ref/spec#Assignability
			a, ok := f.Type.(JSONArray)
			var fieldTypeName string
			if ok {
				fieldTypeName = fmt.Sprintf("[]%s", sp.JSONToGoType(a.Items, false))
			} else {
				fieldTypeName = sp.JSONToGoType(f.Type, false)
			}
			fmt.Fprintf(&buf, "%s %s `json:\"%s\"`\n", symbolName(f.Name), fieldTypeName, f.Name)
		}
		_, _ = buf.WriteString("}")
		return buf.String()
	case JSONArray:
		return fmt.Sprintf("[]%s", sp.JSONToGoType(j.Items, false))
	default:
		log.Panicf("unhandled jsonType %#v", j)
	}
	return ""
}

// InvalidSchemaRefError represents an error which happens when an invalid $ref is found in a JSON Schema.
// Typically, it's a $ref which is unsupported or can't be dereferenced.
type InvalidSchemaRefError struct {
	Ref string
	Msg string
}

func (e InvalidSchemaRefError) Error() string {
	return fmt.Sprintf("invalid $ref %q: %s", string(e.Ref), e.Msg)
}

// InvalidSchemaError represents an error which happens when parsing a (sub)schema fails.
type InvalidSchemaError struct {
	Schema Schema
	Msg    string
}

func (e InvalidSchemaError) Error() string {
	return string(e.Msg)
}

// SchemaParser provides a parser for a Schema instance.
//
// It allows to parse its routes and data structures.
type SchemaParser struct {
	RootSchema     *Schema
	Log            *log.Logger
	RefJSONTypeMap map[string]JSONType
}

func (sp *SchemaParser) logf(format string, v ...interface{}) {
	if sp.Log != nil {
		sp.Log.Printf(format, v...)
	}
}

// ParseRoutes parses the Schema and returns a list of Route instances.
//
// Various errors may be returned, among them InvalidSchemaError and *TypeRedefinitionError.
func (sp *SchemaParser) ParseRoutes() (Routes, error) {
	if sp.RootSchema == nil {
		return nil, InvalidSchemaError{Schema{}, "no schema provided"}
	}
	if sp.RootSchema.Type != "object" {
		return nil, InvalidSchemaError{*sp.RootSchema, "root schema is not an object"}
	}

	var schemaRoutes Routes
	for propertyName, property := range sp.RootSchema.Properties {
		resProperty, err := sp.ResolveSchema(property)
		if err != nil {
			return nil, err
		}
		linksRelAttr := make(map[string]bool)
		for _, link := range resProperty.Links {
			link.ApplyDefaults()

			if exists := linksRelAttr[link.Rel]; exists {
				return nil, InvalidSchemaError{*property, fmt.Sprintf("duplicate link \"rel\" %s", link.Rel)}
			}
			linksRelAttr[link.Rel] = true

			p, err := href2path(link.HRef)
			if err != nil {
				return nil, err
			}
			n, err := href2name(link.HRef)
			if err != nil {
				return nil, err
			}
			route := &Route{
				Path:   p,
				Name:   n,
				Method: strings.ToUpper(link.Method),
			}
			sp.logf("discovered route %s -> %s %q ", route.Name, route.Method, route.Path)

			route.InputIsNotJSON = !link.ReceivesJSON()
			route.OutputIsNotJSON = !link.SendsJSON()

			rp, err := sp.RouteParamsFromLink(&link, resProperty)
			if err != nil {
				return nil, err
			}
			if rp == nil {
				rp = make([]RouteParam, 0)
			}
			route.RouteParams = rp

			// Ignore link input if it's not receiving application/json
			if link.Schema != nil && link.ReceivesJSON() {
				inType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sIn", symbolName(link.Rel), symbolName(propertyName)), link.Schema, link.Schema.Ref)
				if err != nil {
					return nil, err
				}
				route.InType = inType
				sp.logf(" --> found input type %s", inType.Type())
			}
			// Ignore link output if it's not sending application/json
			if link.TargetSchema != nil && link.SendsJSON() {
				outType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sOut", symbolName(link.Rel), symbolName(propertyName)), link.TargetSchema, link.TargetSchema.Ref)
				if err != nil {
					return nil, err
				}
				route.OutType = outType
				sp.logf(" --> found output type %s", outType.Type())
			}
			schemaRoutes = append(schemaRoutes, *route)
		}
	}
	sort.Sort(schemaRoutes)
	typeRedefs, ok := sp.checkNamedTypeRedefinitions(schemaRoutes)
	if !ok {
		for name, types := range typeRedefs {
			// Pick the first we find, for now; we don't want to flood errors
			return schemaRoutes, &TypeRedefinitionError{
				Name:   name,
				First:  types[0],
				Redefs: types[1:],
			}
		}
	}
	return schemaRoutes, nil
}

// ResolveSchema takes a schema and recursively follows its $ref, if any.
// An error is returned if it fails to resolve a ref along the way.
func (sp *SchemaParser) ResolveSchema(schema *Schema) (*Schema, error) {
	var (
		s   *Schema
		err error
	)
	for s = schema; s.Ref != ""; {
		s, err = sp.ResolveSchemaRef(s.Ref, s)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// TypeNamer defines methods for types which are named types.
type TypeNamer interface {
	TypeName() string
}

// ResolveSchemaRef takes a $ref string and returns the pointed schema.
//
// The ref, if relative, is resolved against the relSchema schema. The ref is dereferenced only once.
// An error is returned if the ref or it doesn't point to a schema.
func (sp *SchemaParser) ResolveSchemaRef(schemaRef string, relSchema *Schema) (*Schema, error) {
	// Absolute ref
	if strings.HasPrefix(schemaRef, "#/") {
		keys := strings.Split(schemaRef[2:], "/")

		schv := reflect.ValueOf(sp.RootSchema)
		for _, key := range keys {
			// Dereference pointers
			for schv.Kind() == reflect.Ptr || schv.Kind() == reflect.Interface {
				schv = schv.Elem()
			}
			switch t := schv.Interface().(type) {
			case Schema:
				fv := schv.FieldByName(capitalize(key))
				schv = fv
			case map[string]*Schema:
				ukey, err := unescapePctEnc(key)
				if err != nil {
					return nil, err
				}
				s, ok := t[string(ukey)]
				if !ok {
					return nil, InvalidSchemaRefError{Ref: schemaRef, Msg: "invalid ref"}
				}
				schv = reflect.ValueOf(s)
			default:
				return nil, InvalidSchemaRefError{Ref: schemaRef, Msg: "value is not a valid Schema"}
			}
		}
		// This has been checked in the for loop
		schema, ok := schv.Interface().(*Schema)
		if !ok {
			return nil, InvalidSchemaRefError{Ref: schemaRef, Msg: "value is not a valid Schema"}
		}
		return schema, nil
	}
	propertySchema, ok := relSchema.Properties[schemaRef]
	if !ok {
		return nil, InvalidSchemaRefError{Ref: schemaRef, Msg: "value is not a valid Schema"}
	}
	return propertySchema, nil
}

// JSONTypeFromSchema parses a JSON Schema and returns a value satisfying the jsonType interface.
// The name parameter, if not empty, is used to give a name to a json object.
func (sp *SchemaParser) JSONTypeFromSchema(defaultName string, schema *Schema, ref string) (jt JSONType, err error) {
	defer func() {
		if !(err == nil && jt != nil && jt.Ref() != "") {
			return
		}
		if sp.RefJSONTypeMap == nil {
			sp.RefJSONTypeMap = make(map[string]JSONType)
		}
		if pjt, ok := sp.RefJSONTypeMap[jt.Ref()]; !ok {
			sp.logf("registering ref %q for type %s", jt.Ref(), jt.Type())
			sp.RefJSONTypeMap[jt.Ref()] = jt
		} else {
			sp.logf("[warn] type %s redefined to %s", sp.JSONToGoType(pjt, false), sp.JSONToGoType(jt, false))
		}
	}()
	resSchema, err := sp.ResolveSchema(schema)
	if err != nil {
		return
	}

	name := defaultName
	if ref != "" {
		name = ref2name(ref)
	}

	switch t := resSchema.Type; {
	case t == "object" || t == "": // default value is "object"
		var fields JSONFieldList
		for propertyName, propertySchema := range resSchema.Properties {
			resPropertySchema, err := sp.ResolveSchema(propertySchema)
			if err != nil {
				return nil, err
			}
			typ, err := sp.JSONTypeFromSchema(symbolName(propertyName), resPropertySchema, propertySchema.Ref)
			if err != nil {
				return nil, err
			}
			fields = append(fields, JSONField{
				Name: propertyName,
				Type: typ,
			})
		}
		sort.Sort(fields)

		jt = JSONObject{
			Name:   name,
			ref:    ref,
			Fields: fields,
		}
		return
	case t == "array":
		items := resSchema.Items
		if items == nil {
			return nil, InvalidSchemaError{*schema, "schema: missing items property for type array"}
		}
		resItems, err := sp.ResolveSchema(items)
		if err != nil {
			return nil, err
		}

		jst, err := sp.JSONTypeFromSchema(fmt.Sprintf("%sOne", name), resItems, items.Ref)
		if err != nil {
			return nil, err
		}
		return JSONArray{Name: name, ref: ref, Items: jst}, nil
	case t == "string" && resSchema.Format == "date-time":
		jt = JSONDateTime{ref: ref}
		return
	case t == "string":
		jt = JSONString{ref: ref}
		return
	case t == "boolean":
		jt = JSONBoolean{ref: ref}
		return
	case t == "integer":
		jt = JSONInteger{ref: ref}
		return
	case t == "number":
		jt = JSONNumber{ref: ref}
		return
	case t == "null": // ?
		jt = JSONNull{ref: ref}
		return
	default:
		err = InvalidSchemaError{*schema, fmt.Sprintf("unknown type %q", resSchema.Type)}
		return
	}
}

// RouteParamsFromLink parses the link to return a slice of RouteParam,
// dereferenced using the schema from which the link originates.
func (sp *SchemaParser) RouteParamsFromLink(link *Link, schema *Schema) ([]RouteParam, error) {
	var routeParams []RouteParam
	vars, err := varsFromHRef(link.HRef)
	if err != nil {
		return nil, err
	}

	for _, v := range vars {
		// Strip leading #/definitions/
		name := strings.Replace(v, "#/definitions/", "", 1)
		// Hyphenify the remaining ones
		// TODO allow customize this
		name = strings.Replace(name, "/definitions/", "-", -1)
		vname := toUpperAfterAny(name, "-")

		varRefSchema, err := sp.ResolveSchemaRef(v, schema)
		if err != nil {
			return nil, err
		}
		varRefSchema, err = sp.ResolveSchema(varRefSchema)
		if err != nil {
			return nil, err
		}

		// FIXME we rely on absolute $ref to construct the name here
		names := strings.Split(v, "/")
		typ, err := sp.JSONTypeFromSchema(names[len(names)-1], varRefSchema, v)
		if err != nil {
			return nil, err
		}
		sp.logf(" --> link %s: discovered route param %s", link.HRef, name)
		routeParams = append(routeParams, RouteParam{
			Name:    name,
			Varname: vname,
			Type:    typ,
		})
	}
	return routeParams, nil
}

// checkNamedTypeRedefinitions analyzes the routes just parsed by the SchemaParser and returns the
// redefinitions of named types it finds.
func (sp *SchemaParser) checkNamedTypeRedefinitions(routes Routes) (map[string][]JSONTypeNamer, bool) {
	noRedefinitions := true
	redefinitions := make(map[string][]JSONTypeNamer)
	definitions := make(map[string]JSONTypeNamer)

	for _, route := range routes {
		for _, typ := range []JSONType{route.InType, route.OutType} {
			if typ == nil {
				continue
			}
			routes.walkType(typ, func(jt JSONType) {
				jtn, ok := jt.(JSONTypeNamer)
				if !ok {
					return
				}
				tn := jtn.TypeName()
				if fjtn, ok := definitions[tn]; !ok {
					definitions[tn] = jtn
					redefinitions[tn] = []JSONTypeNamer{jtn}
				} else {
					// it's okay to compare with the string representation of the type.
					if sp.JSONToGoType(jtn, true) != sp.JSONToGoType(fjtn, true) {
						redefinitions[tn] = append(redefinitions[tn], jtn)
						noRedefinitions = false
					}
				}
			})
		}
	}
	if !noRedefinitions {
		for name, types := range redefinitions {
			if len(types) == 1 {
				delete(redefinitions, name)
			}
		}
	}
	return redefinitions, noRedefinitions
}

// TypeRedefinitionError represents a named type which has been redefined one or more times with a different definition.
type TypeRedefinitionError struct {
	Name   string
	First  JSONTypeNamer
	Redefs []JSONTypeNamer
}

func (e *TypeRedefinitionError) Error() string {
	return fmt.Sprintf("type %s defined multiple times", e.Name)
}
