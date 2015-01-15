package main

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

// getSchema opens a json schema, unmarshals it and returns it.
// If an error occurs, the test which called getSchema will fail now.
func getSchema(tb testing.TB, name string) *Schema {
	var schema Schema
	f, err := os.Open(name)
	ok(tb, err)
	defer f.Close()

	ok(tb, json.NewDecoder(f).Decode(&schema))
	return &schema
}

func TestParseSimpleJSONStruct(t *testing.T) {
	schemaStr := `{
    "$schema": "http://json-schema.org/draft-04/hyper-schema",
    "title": "Spell",
    "type": "object",
    "properties": {
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
        }
    }
}`
	var schema Schema
	ok(t, json.Unmarshal([]byte(schemaStr), &schema))

	expectedObj := JSONObject{
		Name: schema.Title,
		Fields: []JSONField{
			{Name: "name", Type: JSONBasicType("string")},
			{Name: "element", Type: JSONBasicType("string")},
			{Name: "power", Type: JSONBasicType("integer")},
			{Name: "all", Type: JSONBasicType("boolean")},
		},
	}
	sort.Sort(expectedObj.Fields)

	sp := SchemaParser{RootSchema: &schema}
	obj, err := sp.JSONTypeFromSchema(schema.Title, &schema)
	ok(t, err)
	equals(t, expectedObj, obj)
}

func TestParseJSONStructWithMixedRef(t *testing.T) {
	schemaStr := `{
    "$schema": "http://json-schema.org/draft-04/hyper-schema",
    "type": "object",
    "definitions": {
        "spell": {
            "type": "object",
            "definitions": {
                "name": {
                    "type": "string"
                },
                "power": {
                    "type": "integer"
                },
                "all": {
                    "type": "boolean"
                }
            },
            "properties": {
                "name": {
                    "$ref": "#/definitions/spell/definitions/name"
                },
                "element": {
                    "type": "string"
                },
                "power": {
                    "$ref": "#/definitions/spell/definitions/power"
                },
                "all": {
                    "$ref": "#/definitions/spell/definitions/all"
                },
                "combinable_spells": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/spell/definitions/name"
                    }
                }
            }
        }
    },
    "properties": {
        "spell": {
            "$ref": "#/definitions/spell"
        }
    }
}`
	var schema Schema
	ok(t, json.Unmarshal([]byte(schemaStr), &schema))

	spellSchema, exists := schema.Definitions["spell"]
	assert(t, exists, "definition %q not found in schema", "spell")
	expectedObj := JSONObject{
		Name: "Spell",
		Fields: JSONFieldList{
			{Name: "name", Type: JSONBasicType("string")},
			{Name: "element", Type: JSONBasicType("string")},
			{Name: "power", Type: JSONBasicType("integer")},
			{Name: "all", Type: JSONBasicType("boolean")},
			{Name: "combinable_spells", Type: JSONArray{Items: JSONBasicType("string")}},
		},
	}
	sort.Sort(expectedObj.Fields)
	sp := SchemaParser{RootSchema: &schema}
	obj, err := sp.JSONTypeFromSchema("Spell", spellSchema)
	ok(t, err)
	equals(t, expectedObj, obj)
}

func TestParseSchemaWithRoutesOneResource(t *testing.T) {
	schema := getSchema(t, "spells.json")

	expectedRoutes := Routes{
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "POST",
			RouteIO: RouteIO{
				InType: JSONObject{
					Name: "createSpellIn",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBasicType("boolean")},
						{Name: "element", Type: JSONBasicType("string")},
						{Name: "name", Type: JSONBasicType("string")},
						{Name: "power", Type: JSONBasicType("integer")},
					},
				},
				OutType: JSONObject{
					Name: "createSpellOut",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBasicType("boolean")},
						{Name: "element", Type: JSONBasicType("string")},
						{Name: "name", Type: JSONBasicType("string")},
						{Name: "power", Type: JSONBasicType("integer")},
					},
				},
			},
		},
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "GET",
			RouteIO: RouteIO{
				OutType: JSONArray{
					Items: JSONObject{
						Name: "",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "all", Type: JSONBasicType("boolean")},
							{Name: "element", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
							{Name: "power", Type: JSONBasicType("integer")},
						},
					},
				},
			},
		},
		{
			Path: "/spells/{spell-name}",
			Name: "spells.one",
			RouteParams: []RouteParam{
				{Name: "spell-name", Varname: "spellName", Type: JSONBasicType("string")},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "oneSpellOut",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBasicType("boolean")},
						{Name: "element", Type: JSONBasicType("string")},
						{Name: "name", Type: JSONBasicType("string")},
						{Name: "power", Type: JSONBasicType("integer")},
					},
				},
			},
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedRoutes, routes)
}

func TestParseSchemaByResource(t *testing.T) {
	schema := getSchema(t, "weapons-and-armors.json")
	expectedResourceRoutes := ResourceRoutes{
		{
			Path:        "/armors",
			Name:        "armors",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIO{
					InType: JSONObject{
						Name: "createArmorIn",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBasicType("boolean")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
					OutType: JSONObject{
						Name: "createArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBasicType("boolean")},
							{Name: "id", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
				},
				"GET": RouteIO{
					OutType: JSONArray{
						Items: JSONObject{
							Name: "",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBasicType("boolean")},
								{Name: "id", Type: JSONBasicType("string")},
								{Name: "name", Type: JSONBasicType("string")},
							},
						},
					},
				},
			},
		},
		{
			Path: "/armors/{armor-id}",
			Name: "armors.one",
			RouteParams: []RouteParam{
				{Name: "armor-id", Varname: "armorId", Type: JSONBasicType("string")},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
					OutType: JSONObject{
						Name: "oneArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBasicType("boolean")},
							{Name: "id", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
				},
				"DELETE": RouteIO{
					OutType: JSONObject{
						Name: "deleteArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBasicType("boolean")},
							{Name: "id", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
				},
			},
		},
		{
			Path:        "/weapons",
			Name:        "weapons",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIO{
					InType: JSONObject{
						Name: "createWeaponIn",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "damage", Type: JSONBasicType("integer")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
					OutType: JSONObject{
						Name: "createWeaponOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "damage", Type: JSONBasicType("integer")},
							{Name: "id", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
				},
				"GET": RouteIO{
					OutType: JSONArray{
						Items: JSONObject{
							Name: "",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "damage", Type: JSONBasicType("integer")},
								{Name: "id", Type: JSONBasicType("string")},
								{Name: "name", Type: JSONBasicType("string")},
							},
						},
					},
				},
			},
		},
		{
			Path: "/weapons/{weapon-id}",
			Name: "weapons.one",
			RouteParams: []RouteParam{
				{Name: "weapon-id", Varname: "weaponId", Type: JSONBasicType("string")},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
					OutType: JSONObject{
						Name: "oneWeaponOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "damage", Type: JSONBasicType("integer")},
							{Name: "id", Type: JSONBasicType("string")},
							{Name: "name", Type: JSONBasicType("string")},
						},
					},
				},
			},
		},
	}
	sort.Sort(expectedResourceRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedResourceRoutes, routes.ByResource())
}
