package main

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"reflect"
	"regexp"
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

func (routes Routes) UniqueObjects() []JSONObject {
	visitedObjects := make(map[string]bool)
	var jsonObjects []JSONObject
	for _, route := range routes {
		for _, typ := range []JSONType{route.InType, route.OutType} {
			if typ == nil {
				continue
			}
			jsonObject, ok := typ.(JSONObject)
			if ok {
				// Skip anonymous types
				if jsonObject.Name == "" {
					continue
				}
				if _, ok := visitedObjects[jsonObject.Name]; !ok {
					visitedObjects[jsonObject.Name] = true
					jsonObjects = append(jsonObjects, jsonObject)
				}
				continue
			}
		}
	}
	return jsonObjects
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

func routeParamsFromPath(s string) ([]RouteParam, error) {
	var (
		inVar       bool
		varbuf      bytes.Buffer
		routeParams []RouteParam
	)
	for _, r := range s {
		if r == '{' {
			if inVar {
				return nil, fmt.Errorf("routeParamsFromPath: %q: found opening { while already in var", s)
			}
			inVar = true
			continue
		}
		if r == '}' {
			inVar = false
			routeParams = append(routeParams, RouteParam{
				Name:    varbuf.String(),
				Varname: strings.Replace(afterRuneUpper(varbuf.String(), "-"), "Uid", "UID", 1),
				Type:    JSONBasicType("string"),
			})
			varbuf.Reset()
			continue
		}
		if inVar {
			_, _ = varbuf.WriteRune(r)
		}
	}
	return routeParams, nil
}

func href2path(s string) string {
	p := refPattern.ReplaceAllStringFunc(s, func(m string) string {
		// Unescape string
		m, err := url.QueryUnescape(m)
		if err != nil {
			panic(err)
		}
		m = strings.Replace(m, "#/definitions/", "", 1)
		m = strings.Replace(m, "/definitions/", "-", -1)
		return "{" + m[2:len(m)-2] + "}"
	})
	return p
}

func href2name(s string) string {
	name := refPattern.ReplaceAllString(s, "one")
	name = strings.Replace(name, "/", ".", -1)
	return name[1:]
}

func capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return fmt.Sprintf("%c%s", unicode.ToUpper(r), s[size:])
}

func symbolName(s string) string {
	return capitalize(afterRuneUpper(s, ".- "))
}

func afterRuneUpper(s string, chars string) string {
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

var refPattern = regexp.MustCompile(`{\([^/]+\)}`)

// json types

type JSONObject struct {
	Name   string
	Fields JSONFieldList
}

func (jo JSONObject) Type() string {
	var buf bytes.Buffer
	_, _ = buf.WriteString("struct ")
	if jo.Name != "" {
		_, _ = buf.WriteString(jo.Name)
		_, _ = buf.WriteRune(' ')
	}
	_, _ = buf.WriteString("{\n")
	for _, f := range jo.Fields {
		fmt.Fprintf(&buf, "%s %s", f.Name, f.Type.Type())
	}
	_, _ = buf.WriteString("}")

	return buf.String()
}

type JSONArray struct {
	Items JSONType
}

func (ja JSONArray) Type() string {
	return fmt.Sprintf("[]%s", ja.Items.Type())
}

type JSONBasicType string

func (jbt JSONBasicType) Type() string {
	return string(jbt)
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
	Type() string
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
	RootSchema *Schema
	Log        *log.Logger
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
			p := href2path(link.HRef)
			route := &Route{
				Path:        p,
				Name:        href2name(link.HRef),
				RouteParams: []RouteParam{},
				Method:      strings.ToUpper(link.Method),
			}
			rp, err := routeParamsFromPath(p)
			if err != nil {
				return nil, err
			}
			if rp == nil {
				rp = make([]RouteParam, 0)
			}
			route.RouteParams = rp
			if link.Schema != nil {
				inType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sIn", link.Rel, symbolName(propertyName)), link.Schema)
				if err != nil {
					return nil, err
				}
				route.InType = inType
			}
			if link.TargetSchema != nil {
				outType, err := sp.JSONTypeFromSchema(fmt.Sprintf("%s%sOut", link.Rel, symbolName(propertyName)), link.TargetSchema)
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
		s, err = sp.ResolveSchemaRef(s.Ref)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

// ResolveSchemaRef takes an absolute $ref string and returns the pointed schema.
// An error is returned if the ref is either not absolute, or it doesn't point to a schema.
func (sp *SchemaParser) ResolveSchemaRef(schemaRef string) (*Schema, error) {
	if !strings.HasPrefix(schemaRef, "#/") {
		return nil, InvalidSchemaRefError{Ref: schemaRef, Msg: "ref is not absolute (missing leading #/)"}
	}
	schemaRef = schemaRef[2:]
	keys := strings.Split(schemaRef, "/")

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
			s, ok := t[key]
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

// JSONTypeFromSchema parses a JSON Schema and returns a value satisfying the jsonType interface.
// The name parameter, if not empty, is used to give a name to a json object.
func (sp *SchemaParser) JSONTypeFromSchema(name string, schema *Schema) (JSONType, error) {
	resSchema, err := sp.ResolveSchema(schema)
	if err != nil {
		return nil, err
	}

	switch t := resSchema.Type; {
	case t == "object" || t == "": // default value is "object"
		var fields JSONFieldList
		for propertyName, propertySchema := range resSchema.Properties {
			propertySchema, err := sp.ResolveSchema(propertySchema)
			if err != nil {
				return nil, err
			}
			typ, err := sp.JSONTypeFromSchema(propertyName, propertySchema)
			if err != nil {
				return nil, err
			}
			fields = append(fields, JSONField{
				Name: propertyName,
				Type: typ,
			})
		}
		sort.Sort(fields)
		return JSONObject{
			Name:   name,
			Fields: fields,
		}, nil
	case t == "array":
		items := resSchema.Items
		if items == nil {
			return nil, InvalidSchemaError{*schema, "schema: missing items property for type array"}
		}
		resItems, err := sp.ResolveSchema(items)
		if err != nil {
			return nil, err
		}
		jst, err := sp.JSONTypeFromSchema("", resItems)
		if err != nil {
			return nil, err
		}
		return JSONArray{Items: jst}, nil
	case t == "string" || t == "boolean" || t == "integer" || t == "number":
		return JSONBasicType(t), nil
	case t == "null":
		return nil, InvalidSchemaError{*schema, "null: type not supported"}
	default:
		return nil, InvalidSchemaError{*schema, fmt.Sprintf("unknown type %q", resSchema.Type)}
	}
}
