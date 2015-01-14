package main

import (
	"encoding/json"
	"sort"
	"testing"
)

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
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		t.Fatal(err)
	}

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
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		t.Fatal(err)
	}

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
	schemaStr := `{
    "$schema": "http://json-schema.org/draft-04/hyper-schema",
    "title": "Test API",
    "type": "object",
    "definitions": {
        "spell": {
            "type": "object",
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
                }
            },
            "links": [
                {
                    "title": "List",
                    "href": "/spells",
                    "method": "POST",
                    "rel": "create",
                    "schema": {
                        "$ref": "#/definitions/spell"
                    }
                },
                {
                    "title": "List",
                    "href": "/spells",
                    "method": "GET",
                    "rel": "list",
                    "targetSchema": {
                        "items": {
                            "$ref": "#/definitions/spell"
                        },
                        "type": "array"
                    }
                },
                {
                    "title": "List",
                    "href": "/spells/{(%23%2Fdefinitions%2Fspell%2Fdefinitions%2Fname)}",
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
	if err := json.Unmarshal([]byte(schemaStr), &schema); err != nil {
		t.Fatal(err)
	}

	expectedRoutes := Routes{
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "POST",
			InType: JSONObject{
				Name: "createSpell",
				Fields: JSONFieldList{ // .Name natural sort
					{Name: "all", Type: JSONBasicType("boolean")},
					{Name: "element", Type: JSONBasicType("string")},
					{Name: "name", Type: JSONBasicType("string")},
					{Name: "power", Type: JSONBasicType("integer")},
				},
			},
		},
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "GET",
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
		{
			Path:        "/spells/{spell-name}",
			Name:        "spells.one",
			RouteParams: []RouteParam{},
			Method:      "GET",
			OutType: JSONObject{
				Name: "oneSpell",
				Fields: JSONFieldList{ // .Name natural sort
					{Name: "all", Type: JSONBasicType("boolean")},
					{Name: "element", Type: JSONBasicType("string")},
					{Name: "name", Type: JSONBasicType("string")},
					{Name: "power", Type: JSONBasicType("integer")},
				},
			},
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: &schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedRoutes, routes)
}
