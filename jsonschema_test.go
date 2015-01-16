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

// getSchemaString is like getSchema but takes a string as json schema data.
func getSchemaString(tb testing.TB, s string) *Schema {
	var schema Schema
	ok(tb, json.Unmarshal([]byte(s), &schema))
	return &schema
}

func TestParseSimpleJSONStruct(t *testing.T) {
	schema := getSchemaString(t, `{
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
}`)

	expectedObj := JSONObject{
		Name: schema.Title,
		Fields: []JSONField{
			{Name: "name", Type: JSONString{}},
			{Name: "element", Type: JSONString{}},
			{Name: "power", Type: JSONInteger{}},
			{Name: "all", Type: JSONBoolean{}},
		},
	}
	sort.Sort(expectedObj.Fields)

	sp := SchemaParser{RootSchema: schema}
	obj, err := sp.JSONTypeFromSchema(schema.Title, schema)
	ok(t, err)
	equals(t, expectedObj, obj)
}

func TestParseJSONStructWithMixedRef(t *testing.T) {
	schema := getSchemaString(t, `{
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
}`)

	spellSchema, exists := schema.Definitions["spell"]
	assert(t, exists, "definition %q not found in schema", "spell")
	expectedObj := JSONObject{
		Name: "Spell",
		Fields: JSONFieldList{
			{Name: "name", Type: JSONString{}},
			{Name: "element", Type: JSONString{}},
			{Name: "power", Type: JSONInteger{}},
			{Name: "all", Type: JSONBoolean{}},
			{Name: "combinable_spells", Type: JSONArray{Name: "combinable_spells", Items: JSONString{}}},
		},
	}
	sort.Sort(expectedObj.Fields)
	sp := SchemaParser{RootSchema: schema}
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
						{Name: "all", Type: JSONBoolean{}},
						{Name: "element", Type: JSONString{}},
						{Name: "name", Type: JSONString{}},
						{Name: "power", Type: JSONInteger{}},
					},
				},
				OutType: JSONObject{
					Name: "createSpellOut",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{}},
						{Name: "element", Type: JSONString{}},
						{Name: "name", Type: JSONString{}},
						{Name: "power", Type: JSONInteger{}},
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
					Name: "listSpellOut",
					Items: JSONObject{
						Name: "",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "all", Type: JSONBoolean{}},
							{Name: "element", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
							{Name: "power", Type: JSONInteger{}},
						},
					},
				},
			},
		},
		{
			Path: "/spells/{spell-name}",
			Name: "spells.one",
			RouteParams: []RouteParam{
				{Name: "spell-name", Varname: "spellName", Type: JSONString{}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "oneSpellOut",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{}},
						{Name: "element", Type: JSONString{}},
						{Name: "name", Type: JSONString{}},
						{Name: "power", Type: JSONInteger{}},
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
							{Name: "can_break", Type: JSONBoolean{}},
							{Name: "name", Type: JSONString{}},
						},
					},
					OutType: JSONObject{
						Name: "createArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBoolean{}},
							{Name: "id", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
						},
					},
				},
				"GET": RouteIO{
					OutType: JSONArray{
						Name: "listArmorOut",
						Items: JSONObject{
							Name: "",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBoolean{}},
								{Name: "id", Type: JSONString{}},
								{Name: "name", Type: JSONString{}},
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
				{Name: "armor-id", Varname: "armorId", Type: JSONString{}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
					OutType: JSONObject{
						Name: "oneArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBoolean{}},
							{Name: "id", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
						},
					},
				},
				"DELETE": RouteIO{
					OutType: JSONObject{
						Name: "deleteArmorOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "can_break", Type: JSONBoolean{}},
							{Name: "id", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
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
							{Name: "damage", Type: JSONInteger{}},
							{Name: "name", Type: JSONString{}},
						},
					},
					OutType: JSONObject{
						Name: "createWeaponOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "damage", Type: JSONInteger{}},
							{Name: "id", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
						},
					},
				},
				"GET": RouteIO{
					OutType: JSONArray{
						Name: "listWeaponOut",
						Items: JSONObject{
							Name: "",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "damage", Type: JSONInteger{}},
								{Name: "id", Type: JSONString{}},
								{Name: "name", Type: JSONString{}},
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
				{Name: "weapon-id", Varname: "weaponId", Type: JSONString{}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
					OutType: JSONObject{
						Name: "oneWeaponOut",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "damage", Type: JSONInteger{}},
							{Name: "id", Type: JSONString{}},
							{Name: "name", Type: JSONString{}},
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
