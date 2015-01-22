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
	ReadOnly bool        `json:"readOnly,omitempty"`
	Example  interface{} `json:"example,omitempty"`
	Format   string      `json:"format,omitempty"`

	Type string `json:"type,omitempty"`

	Ref    string `json:"$ref,omitempty"`
	Schema string `json:"$schema,omitempty"`

	Definitions map[string]*Schema `json:"definitions,omitempty"`

	MultipleOf       float64 `json:"multipleOf,omitempty"`
	Maximum          float64 `json:"maximum,omitempty"`
	ExclusiveMaximum bool    `json:"exclusiveMaximum,omitempty"`
	Minimum          float64 `json:"minimum,omitempty"`
	ExclusiveMinimum bool    `json:"exclusiveMinimum,omitempty"`

	MinLength int    `json:"minLength,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	MinProperties        int                    `json:"minProperties,omitempty"`
	MaxProperties        int                    `json:"maxProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	PatternProperties    map[string]*Schema     `json:"patternProperties,omitempty"`

	Items           *Schema     `json:"items,omitempty"`
	MinItems        int         `json:"minItems,omitempty"`
	MaxItems        int         `json:"maxItems,omitempty"`
	UniqueItems     bool        `json:"uniqueItems,omitempty"`
	AdditionalItems interface{} `json:"additionalItems,omitempty"`

	Enum []string `json:"enum,omitempty"`

	OneOf []Schema `json:"oneOf,omitempty"`
	AnyOf []Schema `json:"anyOf,omitempty"`
	AllOf []Schema `json:"allOf,omitempty"`
	Not   *Schema  `json:"not,omitempty"`

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
	// InType is the JSON type coming in.
	InType JSONType
	// InType is the JSON type coming out.
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

type JSONTypeNamer interface {
	TypeNamer
	JSONType
}

func (routes Routes) JSONNamedTypes() []JSONTypeNamer {
	visited := make(map[string]bool)
	var a []JSONTypeNamer
	for _, route := range routes {
		for _, typ := range []JSONType{route.InType, route.OutType} {
			if typ == nil {
				continue
			}
			t, ok := typ.(TypeNamer)
			if ok {
				tn := t.TypeName()
				// Skip anonymous types
				if tn == "" {
					continue
				}
				if _, ok := visited[tn]; !ok {
					visited[tn] = true
					a = append(a, t.(JSONTypeNamer))
				}
				continue
			}
		}
	}
	return a
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
	return capitalize(toUpperAfterAny(s, ".- "))
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

type JSONObject struct {
	Name   string
	Fields JSONFieldList
	ref    string
}

func (o JSONObject) Type() string {
	return "object"
}

func (o JSONObject) Ref() string {
	return o.ref
}

func (o JSONObject) TypeName() string {
	return o.Name
}

type JSONArray struct {
	Name  string
	ref   string
	Items JSONType
}

func (a JSONArray) Type() string {
	return "array"
}

func (a JSONArray) Ref() string {
	return a.ref
}

func (a JSONArray) TypeName() string {
	return a.Name
}

type JSONString struct {
	ref string
}

func (s JSONString) Type() string {
	return "string"
}

func (s JSONString) Ref() string {
	return s.ref
}

type JSONBoolean struct {
	ref string
}

func (b JSONBoolean) Type() string {
	return "boolean"
}

func (b JSONBoolean) Ref() string {
	return b.ref
}

type JSONInteger struct {
	ref string
}

func (i JSONInteger) Type() string {
	return "integer"
}

func (i JSONInteger) Ref() string {
	return i.ref
}

type JSONNumber struct {
	ref string
}

func (n JSONNumber) Type() string {
	return "number"
}

func (n JSONNumber) Ref() string {
	return n.ref
}

type JSONNull struct {
	ref string
}

func (n JSONNull) Type() string {
	return "null"
}

func (n JSONNull) Ref() string {
	return n.ref
}

type JSONField struct {
	Name string
	Type JSONType
}

type JSONFieldList []JSONField

func (fl JSONFieldList) Len() int           { return len(fl) }
func (fl JSONFieldList) Swap(i, j int)      { fl[i], fl[j] = fl[j], fl[i] }
func (fl JSONFieldList) Less(i, j int) bool { return fl[i].Name < fl[j].Name }

type JSONType interface {
	// Type returns a string representation of this type.
	Type() string
	// Ref returns the absolute reference of type as found in the JSON Schema which defined it.
	Ref() string
}

func (sp *SchemaParser) JSONToGoType(jt JSONType, root bool) string {
	ref := jt.Ref()
	if ref != "" {
		tjt, ok := sp.RefJSONTypeMap[ref]
		if !ok {
			log.Panicf("unregistered json type %s", ref)
		}
		jt = tjt
	}
	n, ok := jt.(TypeNamer)
	if ok && !root {
		if ref != "" {
			// Don't use the name of the type; use it's ref to build it.
			// Strip leading #/definitions/
			name := strings.Replace(ref, "#/definitions/", "", 1)
			// Hyphenify the remaining ones
			// TODO allow customize this
			name = strings.Replace(name, "/definitions/", "-", -1)
			return symbolName(name)
		}
		return n.TypeName()
	}
	switch j := jt.(type) {
	case JSONString:
		return "string"
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
			fmt.Fprintf(&buf, "%s %s `json:\"%s\"`\n", capitalize(f.Name), sp.JSONToGoType(f.Type, false), f.Name)
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

type InvalidSchemaRefError struct {
	Ref string
	Msg string
}

func (e InvalidSchemaRefError) Error() string {
	return fmt.Sprintf("invalid $ref %q: %s", string(e.Ref), e.Msg)
}

type InvalidSchemaError struct {
	Schema Schema
	Msg    string
}

func (e InvalidSchemaError) Error() string {
	return string(e.Msg)
}

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
		for _, link := range resProperty.Links {
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
			rp, err := sp.RouteParamsFromLink(&link, resProperty)
			if err != nil {
				return nil, err
			}
			if rp == nil {
				rp = make([]RouteParam, 0)
			}
			route.RouteParams = rp
			if link.Schema != nil {
				inType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sIn", link.Rel, symbolName(propertyName)), link.Schema, link.Schema.Ref)
				if err != nil {
					return nil, err
				}
				route.InType = inType
			}
			if link.TargetSchema != nil {
				outType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sOut", link.Rel, symbolName(propertyName)), link.TargetSchema, link.TargetSchema.Ref)
				if err != nil {
					return nil, err
				}
				route.OutType = outType
			}
			schemaRoutes = append(schemaRoutes, *route)
		}
	}
	sort.Sort(schemaRoutes)
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
func (sp *SchemaParser) JSONTypeFromSchema(name string, schema *Schema, ref string) (jt JSONType, err error) {
	defer func() {
		if !(err == nil && jt != nil && jt.Ref() != "") {
			return
		}
		if sp.RefJSONTypeMap == nil {
			sp.RefJSONTypeMap = make(map[string]JSONType)
		}
		if _, ok := sp.RefJSONTypeMap[jt.Ref()]; !ok {
			sp.RefJSONTypeMap[jt.Ref()] = jt
		}
	}()
	resSchema, err := sp.ResolveSchema(schema)
	if err != nil {
		return
	}

	switch t := resSchema.Type; {
	case t == "object" || t == "": // default value is "object"
		var fields JSONFieldList
		for propertyName, propertySchema := range resSchema.Properties {
			resPropertySchema, err := sp.ResolveSchema(propertySchema)
			if err != nil {
				return nil, err
			}
			typ, err := sp.JSONTypeFromSchema(propertyName, resPropertySchema, propertySchema.Ref)
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
		jst, err := sp.JSONTypeFromSchema("", resItems, items.Ref)
		if err != nil {
			return nil, err
		}
		return JSONArray{Name: name, ref: ref, Items: jst}, nil
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
	case t == "null":
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
		routeParams = append(routeParams, RouteParam{
			Name:    name,
			Varname: vname,
			Type:    typ,
		})
	}
	return routeParams, nil
}
