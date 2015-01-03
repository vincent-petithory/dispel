package main

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Schema represents a JSON Schema.
type Schema struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version,omitempty"`

	Default  interface{} `json:"default,omitempty"`
	ReadOnly bool        `json:"readOnly,omitempty"`
	Example  interface{} `json:"example,omitempty"`
	Format   string      `json:"format,omitempty"`

	Type interface{} `json:"type,omitempty"`

	Ref    string `json:"$ref,omitempty"`
	Schema string `json:"$schema,omitempty"`

	Definitions map[string]*Schema `json:"definitions,omitempty"`

	// Numbers
	MultipleOf       float64 `json:"multipleOf,omitempty"`
	Maximum          float64 `json:"maximum,omitempty"`
	ExclusiveMaximum bool    `json:"exclusiveMaximum,omitempty"`
	Minimum          float64 `json:"minimum,omitempty"`
	ExclusiveMinimum bool    `json:"exclusiveMinimum,omitempty"`

	// Strings
	MinLength int    `json:"minLength,omitempty"`
	MaxLength int    `json:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty"`

	// Objects
	MinProperties        int                    `json:"minProperties,omitempty"`
	MaxProperties        int                    `json:"maxProperties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	Properties           map[string]*Schema     `json:"properties,omitempty"`
	Dependencies         map[string]interface{} `json:"dependencies,omitempty"`
	AdditionalProperties interface{}            `json:"additionalProperties,omitempty"`
	PatternProperties    map[string]*Schema     `json:"patternProperties,omitempty"`

	// Arrays
	Items           *Schema     `json:"items,omitempty"`
	MinItems        int         `json:"minItems,omitempty"`
	MaxItems        int         `json:"maxItems,omitempty"`
	UniqueItems     bool        `json:"uniqueItems,omitempty"`
	AdditionalItems interface{} `json:"additionalItems,omitempty"`

	// All
	Enum []string `json:"enum,omitempty"`

	// Schemas
	OneOf []Schema `json:"oneOf,omitempty"`
	AnyOf []Schema `json:"anyOf,omitempty"`
	AllOf []Schema `json:"allOf,omitempty"`
	Not   *Schema  `json:"not,omitempty"`

	// Links
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
}

type route struct {
	Path         string
	Name         string
	RouteParams  []routeParam
	Methods      methods
	FuncBaseName string
}

type routeParam struct {
	Name    string
	Varname string
}

type routes []route

func (r routes) Len() int           { return len(r) }
func (r routes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }
func (r routes) Less(i, j int) bool { return r[i].Path < r[j].Path }

type methods []string

func (m methods) Len() int      { return len(m) }
func (m methods) Swap(i, j int) { m[i], m[j] = m[j], m[i] }
func (m methods) Less(i, j int) bool {
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

func routeParamsFromPath(s string) []routeParam {
	var (
		inVar       bool
		varbuf      bytes.Buffer
		routeParams []routeParam
	)
	for _, r := range s {
		if r == '{' {
			if inVar {
				panic("found opening { while already in var")
			}
			inVar = true
			continue
		}
		if r == '}' {
			inVar = false
			routeParams = append(routeParams, routeParam{
				Name:    varbuf.String(),
				Varname: strings.Replace(afterRuneUpper(varbuf.String(), "-"), "Uid", "UID", 1),
			})
			varbuf.Reset()
			continue
		}
		if inVar {
			_, _ = varbuf.WriteRune(r)
		}
	}
	return routeParams
}

func href2name(s string) string {
	name := refPattern.ReplaceAllString(s, "self")
	name = strings.Replace(name, "/", ".", -1)
	return name[1:]
}

func capitalize(s string) string {
	r, size := utf8.DecodeRuneInString(s)
	return fmt.Sprintf("%c%s", unicode.ToUpper(r), s[size:])
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

func parseRoutes(schema Schema) routes {
	pathRouteMap := make(map[string]*route, len(schema.Definitions)*4)
	var schemaRoutes routes
	for _, definition := range schema.Definitions {
		for _, link := range definition.Links {
			p := href2path(link.HRef)
			r, ok := pathRouteMap[p]
			if !ok {
				name := href2name(link.HRef)
				r = &route{
					Path:         p,
					Name:         name,
					FuncBaseName: capitalize(afterRuneUpper(name, ".-")),
					RouteParams:  routeParamsFromPath(p),
				}
				pathRouteMap[p] = r
			}
			r.Methods = append(r.Methods, link.Method)
			sort.Sort(r.Methods)
		}
	}
	for _, r := range pathRouteMap {
		schemaRoutes = append(schemaRoutes, *r)
	}
	sort.Sort(schemaRoutes)
	return schemaRoutes
}
