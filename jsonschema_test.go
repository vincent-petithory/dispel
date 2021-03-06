package dispel

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"
)

func routesEquals(expectedRoutes Routes, actualRoutes Routes) error {
	le := len(expectedRoutes)
	la := len(actualRoutes)
	if le != la {
		return fmt.Errorf("expected %d routes, got %d", le, la)
	}
	for i := 0; i < le; i++ {
		if err := routeEquals(expectedRoutes[i], actualRoutes[i]); err != nil {
			return fmt.Errorf("routes[%d]: %v", i, err)
		}
	}
	return nil
}

func routeEquals(expectedRoute Route, actualRoute Route) error {
	if err := linkEquals(expectedRoute.Link, actualRoute.Link); err != nil {
		return fmt.Errorf("route.Link: %v", err)
	}
	// ignore checking the schema and targetSchema
	expectedRoute.Link = Link{}
	actualRoute.Link = Link{}
	err := fmt.Errorf("expected:\n%#v\ngot:\n%#v", expectedRoute, actualRoute)
	if !reflect.DeepEqual(expectedRoute, actualRoute) {
		return err
	}
	return nil
}

func resourceRoutesEquals(expectedResourceRoutes ResourceRoutes, actualResourceRoutes ResourceRoutes) error {
	le := len(expectedResourceRoutes)
	la := len(actualResourceRoutes)
	if le != la {
		return fmt.Errorf("expected %d resource routes, got %d", le, la)
	}
	for i := 0; i < le; i++ {
		if err := resourceRouteEquals(expectedResourceRoutes[i], actualResourceRoutes[i]); err != nil {
			return fmt.Errorf("at %d: %v", i, err)
		}
	}
	return nil
}

func resourceRouteEquals(expectedResourceRoute ResourceRoute, actualResourceRoute ResourceRoute) error {
	err := fmt.Errorf("expected:\n%#v\ngot:\n%#v", expectedResourceRoute, actualResourceRoute)
	for m := range actualResourceRoute.MethodRouteIOMap {
		if err := linkEquals(expectedResourceRoute.MethodRouteIOMap[m].Link, actualResourceRoute.MethodRouteIOMap[m].Link); err != nil {
			return err
		}

		erio := expectedResourceRoute.MethodRouteIOMap[m]
		erio.Link = Link{}
		expectedResourceRoute.MethodRouteIOMap[m] = erio

		ario := actualResourceRoute.MethodRouteIOMap[m]
		ario.Link = Link{}
		actualResourceRoute.MethodRouteIOMap[m] = ario
	}
	if !reflect.DeepEqual(expectedResourceRoute, actualResourceRoute) {
		return err
	}
	return nil
}

func linkEquals(expectedLink Link, actualLink Link) error {
	err := fmt.Errorf("expected:\n%#v\ngot:\n%#v", expectedLink, actualLink)
	// ignore checking the schema and targetSchema
	actualLink.Schema = nil
	actualLink.TargetSchema = nil
	if !reflect.DeepEqual(expectedLink, actualLink) {
		return err
	}
	return nil
}

// getSchema opens a json schema, unmarshals it and returns it.
// If an error occurs, the test which called getSchema will fail now.
func getSchema(tb testing.TB, name string) *Schema {
	var schema Schema
	f, err := os.Open(name)
	if err != nil {
		tb.Error(err)
		return nil
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&schema); err != nil {
		tb.Error(err)
		return nil
	}
	return &schema
}

// getSchemaString is like getSchema but takes a string as input.
func getSchemaString(tb testing.TB, s string) *Schema {
	var schema Schema
	if err := json.Unmarshal([]byte(s), &schema); err != nil {
		tb.Error(err)
		return nil
	}
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
        },
        "level_on": {
            "type": "string",
            "format": "date-time"
        }
    }
}`)
	if t.Failed() {
		return
	}

	expectedObj := JSONObject{
		Name: schema.Title,
		Fields: []JSONField{
			{Name: "name", Type: JSONString{}},
			{Name: "element", Type: JSONString{}},
			{Name: "power", Type: JSONInteger{}},
			{Name: "all", Type: JSONBoolean{}},
			{Name: "level_on", Type: JSONDateTime{}},
		},
	}
	sort.Sort(expectedObj.Fields)

	sp := SchemaParser{RootSchema: schema}
	obj, err := sp.JSONTypeFromSchema(schema.Title, schema, "")
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(expectedObj, obj) {
		t.Errorf("expected %#v, got %#v", expectedObj, obj)
		return
	}
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
	if t.Failed() {
		return
	}

	spellSchema, exists := schema.Properties["spell"]
	if !exists {
		t.Errorf("definition %q not found in schema", "spell")
		return
	}
	expectedObj := JSONObject{
		Name: "Spell",
		ref:  "#/definitions/spell",
		Fields: JSONFieldList{
			{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
			{Name: "element", Type: JSONString{}},
			{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
			{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
			{Name: "combinable_spells", Type: JSONArray{Name: "CombinableSpells", Items: JSONString{ref: "#/definitions/spell/definitions/name"}}},
		},
	}
	sort.Sort(expectedObj.Fields)
	sp := SchemaParser{RootSchema: schema}
	obj, err := sp.JSONTypeFromSchema("Spell", spellSchema, spellSchema.Ref)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(expectedObj, obj) {
		t.Errorf("expected %#v, got %#v", expectedObj, obj)
		return
	}
}

func TestParseSchemaWithRoutesOneResource(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")
	if t.Failed() {
		return
	}

	expectedRoutes := Routes{
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "POST",
			RouteIO: RouteIO{
				InType: JSONObject{
					Name: "Spell",
					ref:  "#/definitions/spell",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
						{Name: "element", Type: JSONString{ref: "#/definitions/spell/definitions/element"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
						{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
					},
				},
				OutType: JSONObject{
					Name: "Spell",
					ref:  "#/definitions/spell",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
						{Name: "element", Type: JSONString{ref: "#/definitions/spell/definitions/element"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
						{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
					},
				},
			},
			Link: Link{
				Title:     "Create a spell",
				HRef:      "/spells",
				Rel:       "create",
				Method:    "POST",
				EncType:   "application/json",
				MediaType: "application/json",
			},
		},
		{
			Path:        "/spells",
			Name:        "spells",
			RouteParams: []RouteParam{},
			Method:      "GET",
			RouteIO: RouteIO{
				OutType: JSONArray{
					Name: "ListSpellOut",
					ref:  "",
					Items: JSONObject{
						Name: "Spell",
						ref:  "#/definitions/spell",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
							{Name: "element", Type: JSONString{ref: "#/definitions/spell/definitions/element"}},
							{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
							{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
						},
					},
				},
			},
			Link: Link{
				Title:     "List spells",
				HRef:      "/spells",
				Rel:       "list",
				Method:    "GET",
				EncType:   "application/json",
				MediaType: "application/json",
			},
		},
		{
			Path: "/spells/{spell-name}",
			Name: "spells.one",
			RouteParams: []RouteParam{
				{Name: "spell-name", Varname: "spellName", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "Spell",
					ref:  "#/definitions/spell",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
						{Name: "element", Type: JSONString{ref: "#/definitions/spell/definitions/element"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
						{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
					},
				},
			},
			Link: Link{
				Title:     "Info for a spell",
				HRef:      "/spells/{(#/definitions/spell/definitions/name)}",
				Rel:       "one",
				Method:    "GET",
				EncType:   "application/json",
				MediaType: "application/json",
			},
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}
	if err := routesEquals(expectedRoutes, routes); err != nil {
		t.Error(err)
	}
}

func TestParseSchemaByResource(t *testing.T) {
	schema := getSchema(t, "testdata/weapons-and-armors.json")
	if t.Failed() {
		return
	}
	expectedResourceRoutes := ResourceRoutes{
		{
			Path:        "/armors",
			Name:        "armors",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIOAndLink{
					RouteIO: RouteIO{
						InType: JSONObject{
							Name: "CreateArmorIn",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBoolean{ref: "#/definitions/armor/definitions/canbreak"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/armor/definitions/name"}},
							},
						},
						OutType: JSONObject{
							Name: "Armor",
							ref:  "#/definitions/armor",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBoolean{ref: "#/definitions/armor/definitions/canbreak"}},
								{Name: "id", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/armor/definitions/name"}},
							},
						},
					},
					Link: Link{
						Title:     "Create an armor",
						HRef:      "/armors",
						Rel:       "create",
						Method:    "POST",
						EncType:   "application/json",
						MediaType: "application/json",
					},
				},
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "ListArmorOut",
							Items: JSONObject{
								Name: "Armor",
								ref:  "#/definitions/armor",
								Fields: JSONFieldList{ // .Name natural sort
									{Name: "can_break", Type: JSONBoolean{ref: "#/definitions/armor/definitions/canbreak"}},
									{Name: "id", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
									{Name: "name", Type: JSONString{ref: "#/definitions/armor/definitions/name"}},
								},
							},
						},
					},
					Link: Link{
						Title:       "List armors",
						Description: "",
						HRef:        "/armors",
						Rel:         "list",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path: "/armors/{armor-id}",
			Name: "armors.one",
			RouteParams: []RouteParam{
				{Name: "armor-id", Varname: "armorId", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Armor",
							ref:  "#/definitions/armor",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBoolean{ref: "#/definitions/armor/definitions/canbreak"}},
								{Name: "id", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/armor/definitions/name"}},
							},
						},
					},
					Link: Link{
						Title:       "Info for an armor",
						Description: "",
						HRef:        "/armors/{(#/definitions/armor/definitions/id)}",
						Rel:         "one",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"DELETE": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Armor",
							ref:  "#/definitions/armor",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "can_break", Type: JSONBoolean{ref: "#/definitions/armor/definitions/canbreak"}},
								{Name: "id", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/armor/definitions/name"}},
							},
						},
					},
					Link: Link{
						Title:       "Deletes an existing armor",
						Description: "",
						HRef:        "/armors/{(#/definitions/armor/definitions/id)}",
						Rel:         "delete",
						Method:      "DELETE",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path:        "/weapons",
			Name:        "weapons",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIOAndLink{
					RouteIO: RouteIO{
						InType: JSONObject{
							Name: "CreateWeaponIn",
							ref:  "",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "damage", Type: JSONInteger{ref: "#/definitions/weapon/definitions/damage"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/weapon/definitions/name"}},
							},
						},
						OutType: JSONObject{
							Name: "Weapon",
							ref:  "#/definitions/weapon",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "damage", Type: JSONInteger{ref: "#/definitions/weapon/definitions/damage"}},
								{Name: "id", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/weapon/definitions/name"}},
							},
						},
					},
					Link: Link{
						Title:       "Create a weapon",
						Description: "",
						HRef:        "/weapons",
						Rel:         "create",
						Method:      "POST",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "ListWeaponOut",
							Items: JSONObject{
								Name: "Weapon",
								ref:  "#/definitions/weapon",
								Fields: JSONFieldList{ // .Name natural sort
									{Name: "damage", Type: JSONInteger{ref: "#/definitions/weapon/definitions/damage"}},
									{Name: "id", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
									{Name: "name", Type: JSONString{ref: "#/definitions/weapon/definitions/name"}},
								},
							},
						},
					},
					Link: Link{
						Title:       "List weapons",
						Description: "",
						HRef:        "/weapons",
						Rel:         "list",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path: "/weapons/{weapon-id}",
			Name: "weapons.one",
			RouteParams: []RouteParam{
				{Name: "weapon-id", Varname: "weaponId", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Weapon",
							ref:  "#/definitions/weapon",
							Fields: JSONFieldList{ // .Name natural sort
								{Name: "damage", Type: JSONInteger{ref: "#/definitions/weapon/definitions/damage"}},
								{Name: "id", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
								{Name: "name", Type: JSONString{ref: "#/definitions/weapon/definitions/name"}},
							},
						},
					},
					Link: Link{
						Title:       "Info for a weapon",
						Description: "",
						HRef:        "/weapons/{(#/definitions/weapon/definitions/id)}",
						Rel:         "one",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
	}
	sort.Sort(expectedResourceRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}
	if err := resourceRoutesEquals(expectedResourceRoutes, routes.ByResource()); err != nil {
		t.Error(err)
	}
}

func TestMixedRouteParams(t *testing.T) {
	schema := getSchema(t, "testdata/one-route-mixed-params.json")
	if t.Failed() {
		return
	}

	expectedRoutes := Routes{
		{
			Path: "/spells/{spell-name}",
			Name: "spells.one",
			RouteParams: []RouteParam{
				{Name: "spell-name", Varname: "spellName", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "Spell",
					ref:  "#/definitions/spell",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "all", Type: JSONBoolean{ref: "#/definitions/spell/definitions/all"}},
						{Name: "element", Type: JSONString{ref: "#/definitions/spell/definitions/element"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/spell/definitions/name"}},
						{Name: "power", Type: JSONInteger{ref: "#/definitions/spell/definitions/power"}},
					},
				},
			},
			Link: Link{
				Title:       "Info for a spell",
				Description: "",
				HRef:        "/spells/{(#/definitions/spell/definitions/name)}",
				Rel:         "one",
				Method:      "GET",
				EncType:     "application/json",
				MediaType:   "application/json",
			},
		},
		{
			Path: "/locations/{location-id}",
			Name: "locations.one",
			RouteParams: []RouteParam{
				{Name: "location-id", Varname: "locationId", Type: JSONInteger{ref: "#/definitions/location/definitions/id"}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "Location",
					ref:  "#/definitions/location",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "id", Type: JSONInteger{ref: "#/definitions/location/definitions/id"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/location/definitions/name"}},
					},
				},
			},
			Link: Link{
				Title:       "Info for a location",
				Description: "",
				HRef:        "/locations/{(#/definitions/location/definitions/id)}",
				Rel:         "one",
				Method:      "GET",
				EncType:     "application/json",
				MediaType:   "application/json",
			},
		},
		{
			Path: "/weapons/{weapon-id}",
			Name: "weapons.one",
			RouteParams: []RouteParam{
				{Name: "weapon-id", Varname: "weaponId", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "Weapon",
					ref:  "#/definitions/weapon",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "id", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
						{Name: "magical", Type: JSONBoolean{ref: "#/definitions/weapon/definitions/magical"}},
						{Name: "name", Type: JSONString{ref: "#/definitions/weapon/definitions/name"}},
					},
				},
			},
			Link: Link{
				Title:       "Info for a weapon",
				Description: "",
				HRef:        "/weapons/{(#/definitions/weapon/definitions/id)}",
				Rel:         "one",
				Method:      "GET",
				EncType:     "application/json",
				MediaType:   "application/json",
			},
		},
		{
			Path: "/materias/{name}",
			Name: "materias.one",
			RouteParams: []RouteParam{
				{Name: "name", Varname: "name", Type: JSONString{ref: "name"}},
			},
			Method: "GET",
			RouteIO: RouteIO{
				OutType: JSONObject{
					Name: "Materia",
					ref:  "#/definitions/materia",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "name", Type: JSONString{ref: "#/definitions/materia/definitions/name"}},
					},
				},
			},
			Link: Link{
				Title:       "Info for a materia",
				Description: "",
				HRef:        "/materias/{name}",
				Rel:         "one",
				Method:      "GET",
				EncType:     "application/json",
				MediaType:   "application/json",
			},
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}
	if err := routesEquals(expectedRoutes, routes); err != nil {
		t.Error(err)
	}
}

func TestPreProcessHRefVar(t *testing.T) {
	tests := []struct {
		href   string
		ppHRef string
	}{
		{href: "{(#/definitions/spell/definitions/name)}", ppHRef: "{%23%2Fdefinitions%2Fspell%2Fdefinitions%2Fname}"},
		{href: "{(escape space)}", ppHRef: "{escape%20space}"},
		{href: "{(escape+plus)}", ppHRef: "{escape%2Bplus}"},
		{href: "{(escape*asterisk)}", ppHRef: "{escape%2Aasterisk}"},
		{href: "{(escape(bracket)}", ppHRef: "{escape%28bracket}"},
		{href: "{(escape))bracket)}", ppHRef: "{escape%29bracket}"},
		{href: "{(a))b)}", ppHRef: "{a%29b}"},
		{href: "{(a (b)))}", ppHRef: "{a%20%28b%29}"},
		{href: "{()}", ppHRef: "{%65mpty}"},

		// We don't support those
		//{href: "{+$*}", ppHRef: "{+%73elf*}"},
		//{href: "{+($)*}", ppHRef: "{+%24*}"},
	}
	for _, test := range tests {
		if test.ppHRef != preProcessHRefVar(test.href) {
			t.Errorf("expected %s, got %s", test.ppHRef, preProcessHRefVar(test.href))
			continue
		}
	}
}

func TestVarsFromHRef(t *testing.T) {
	tests := []struct {
		href  string
		vars  []string
		valid bool
	}{
		{href: "/spells/{(#/definitions/spell/definitions/name)}", vars: []string{"#/definitions/spell/definitions/name"}, valid: true},
		{href: "/documents/{(#/definitions/document/definitions/id)}/pages/{(#/definitions/page/definitions/id)}", vars: []string{"#/definitions/document/definitions/id", "#/definitions/page/definitions/id"}, valid: true},
	}

	for _, test := range tests {
		vars, err := varsFromHRef(test.href)
		if !test.valid && err == nil {
			t.Errorf("Expected invalid href %q, got %q %v", test.href, vars, err)
			continue
		}
		if err != nil {
			t.Error(err)
			return
		}
		if !reflect.DeepEqual(test.vars, vars) {
			t.Errorf("expected %#v, got %#v", test.vars, vars)
			return
		}
	}
}

func TestHRef2name(t *testing.T) {
	tests := []struct {
		href string
		name string
	}{
		{href: "/spells/{(#/definitions/spell/definitions/name)}", name: "spells.one"},
		{href: "/spells/{(#/definitions/id)}", name: "spells.one"},
		{href: "/spells/{id}", name: "spells.one"},
	}

	for _, test := range tests {
		name, err := href2name(test.href)
		if err != nil {
			t.Error(err)
			return
		}
		if test.name != name {
			t.Errorf("expected %s, got %s", test.name, name)
			return
		}
	}
}

func TestParseKrakenSchema(t *testing.T) {
	schema := getSchema(t, "testdata/kraken.json")
	if t.Failed() {
		return
	}
	expectedResourceRoutes := ResourceRoutes{
		{
			Path:        "/fileservers",
			Name:        "fileservers",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name:  "ListAllFileServerTypeOut",
							Items: JSONString{ref: "#/definitions/fileservertype"},
						},
					},
					Link: Link{
						Title:       "List existing file server types",
						Description: "",
						HRef:        "/fileservers",
						Rel:         "list-all",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path:        "/servers",
			Name:        "servers",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIOAndLink{
					RouteIO: RouteIO{
						InType: JSONObject{
							Name: "CreateRandomServerIn",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
							},
						},
						OutType: JSONObject{
							Name: "Server",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
								JSONField{
									Name: "mounts",
									Type: JSONArray{
										Name: "ServerMounts",
										ref:  "#/definitions/server/definitions/mounts",
										Items: JSONObject{
											Name: "Mount",
											Fields: JSONFieldList{
												JSONField{
													Name: "id",
													Type: JSONString{ref: "#/definitions/mount/definitions/id"},
												},
												JSONField{
													Name: "source",
													Type: JSONString{ref: "#/definitions/mount/definitions/source"},
												},
												JSONField{
													Name: "target",
													Type: JSONString{ref: "#/definitions/mount/definitions/target"}},
											},
											ref: "#/definitions/mount",
										},
									},
								},
								JSONField{
									Name: "port",
									Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
								},
							},
							ref: "#/definitions/server",
						},
					},
					Link: Link{
						Title:       "Create a new server listening on a random port",
						Description: "",
						HRef:        "/servers",
						Rel:         "create-random",
						Method:      "POST",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"DELETE": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "DeleteAllServerOut",
							ref:  "",
							Items: JSONObject{
								Name: "Server",
								Fields: JSONFieldList{
									JSONField{
										Name: "bind_address",
										Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
									},
									JSONField{
										Name: "mounts",
										Type: JSONArray{
											Name: "ServerMounts",
											ref:  "#/definitions/server/definitions/mounts",
											Items: JSONObject{
												Name: "Mount",
												Fields: JSONFieldList{
													JSONField{
														Name: "id",
														Type: JSONString{ref: "#/definitions/mount/definitions/id"},
													},
													JSONField{
														Name: "source",
														Type: JSONString{ref: "#/definitions/mount/definitions/source"},
													},
													JSONField{
														Name: "target",
														Type: JSONString{ref: "#/definitions/mount/definitions/target"},
													},
												},
												ref: "#/definitions/mount",
											},
										},
									},
									JSONField{
										Name: "port",
										Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
									},
								},
								ref: "#/definitions/server"},
						},
					},
					Link: Link{
						Title:       "Delete all existing servers and all their mounts",
						Description: "",
						HRef:        "/servers",
						Rel:         "delete-all",
						Method:      "DELETE",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "ListAllServerOut",
							Items: JSONObject{
								Name: "Server",
								Fields: JSONFieldList{
									JSONField{
										Name: "bind_address",
										Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
									},
									JSONField{
										Name: "mounts",
										Type: JSONArray{
											Name: "ServerMounts",
											ref:  "#/definitions/server/definitions/mounts",
											Items: JSONObject{
												Name: "Mount",
												Fields: JSONFieldList{
													JSONField{
														Name: "id",
														Type: JSONString{ref: "#/definitions/mount/definitions/id"},
													},
													JSONField{
														Name: "source",
														Type: JSONString{ref: "#/definitions/mount/definitions/source"},
													},
													JSONField{
														Name: "target",
														Type: JSONString{ref: "#/definitions/mount/definitions/target"},
													},
												},
												ref: "#/definitions/mount"},
										},
									},
									JSONField{
										Name: "port",
										Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
									},
								},
								ref: "#/definitions/server"},
						},
					},
					Link: Link{
						Title:       "List existing servers",
						Description: "",
						HRef:        "/servers",
						Rel:         "list-all",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path: "/servers/{server-port}",
			Name: "servers.one",
			RouteParams: []RouteParam{
				RouteParam{
					Name:    "server-port",
					Varname: "serverPort",
					Type:    JSONInteger{ref: "#/definitions/server/definitions/port"},
				},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"DELETE": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Server",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
								JSONField{
									Name: "mounts",
									Type: JSONArray{
										Name: "ServerMounts",
										ref:  "#/definitions/server/definitions/mounts",
										Items: JSONObject{
											Name: "Mount",
											Fields: JSONFieldList{
												JSONField{
													Name: "id",
													Type: JSONString{ref: "#/definitions/mount/definitions/id"},
												},
												JSONField{
													Name: "source",
													Type: JSONString{ref: "#/definitions/mount/definitions/source"},
												},
												JSONField{
													Name: "target",
													Type: JSONString{ref: "#/definitions/mount/definitions/target"},
												},
											},
											ref: "#/definitions/mount"},
									},
								},
								JSONField{
									Name: "port",
									Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
								},
							},
							ref: "#/definitions/server"},
					},
					Link: Link{
						Title:       "Delete an existing server and all its mounts",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}",
						Rel:         "delete",
						Method:      "DELETE",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"PUT": RouteIOAndLink{
					RouteIO: RouteIO{
						InType: JSONObject{
							Name: "CreateServerIn",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
							},
							ref: "",
						},
						OutType: JSONObject{
							Name: "Server",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
								JSONField{
									Name: "mounts",
									Type: JSONArray{
										Name: "ServerMounts",
										ref:  "#/definitions/server/definitions/mounts",
										Items: JSONObject{
											Name: "Mount",
											Fields: JSONFieldList{
												JSONField{
													Name: "id",
													Type: JSONString{ref: "#/definitions/mount/definitions/id"},
												},
												JSONField{
													Name: "source",
													Type: JSONString{ref: "#/definitions/mount/definitions/source"},
												},
												JSONField{
													Name: "target",
													Type: JSONString{ref: "#/definitions/mount/definitions/target"},
												},
											},
											ref: "#/definitions/mount"},
									},
								},
								JSONField{
									Name: "port",
									Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
								},
							},
							ref: "#/definitions/server",
						},
					},
					Link: Link{
						Title:       "Create a new server listening on a specific port",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}",
						Rel:         "create",
						Method:      "PUT",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Server",
							Fields: JSONFieldList{
								JSONField{
									Name: "bind_address",
									Type: JSONString{ref: "#/definitions/server/definitions/bindaddress"},
								},
								JSONField{
									Name: "mounts",
									Type: JSONArray{
										Name: "ServerMounts",
										ref:  "#/definitions/server/definitions/mounts",
										Items: JSONObject{
											Name: "Mount",
											Fields: JSONFieldList{
												JSONField{
													Name: "id",
													Type: JSONString{ref: "#/definitions/mount/definitions/id"},
												},
												JSONField{
													Name: "source",
													Type: JSONString{ref: "#/definitions/mount/definitions/source"},
												},
												JSONField{
													Name: "target",
													Type: JSONString{ref: "#/definitions/mount/definitions/target"},
												},
											},
											ref: "#/definitions/mount",
										},
									},
								},
								JSONField{
									Name: "port",
									Type: JSONInteger{ref: "#/definitions/server/definitions/port"},
								},
							},
							ref: "#/definitions/server",
						},
					},
					Link: Link{
						Title:       "Info for a server",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}",
						Rel:         "self",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path: "/servers/{server-port}/mounts",
			Name: "servers.one.mounts",
			RouteParams: []RouteParam{
				RouteParam{
					Name:    "server-port",
					Varname: "serverPort",
					Type:    JSONInteger{ref: "#/definitions/server/definitions/port"},
				},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"DELETE": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "DeleteAllMountOut",
							Items: JSONObject{
								Name: "Mount",
								Fields: JSONFieldList{
									JSONField{
										Name: "id",
										Type: JSONString{ref: "#/definitions/mount/definitions/id"},
									},
									JSONField{
										Name: "source",
										Type: JSONString{ref: "#/definitions/mount/definitions/source"},
									},
									JSONField{
										Name: "target",
										Type: JSONString{ref: "#/definitions/mount/definitions/target"},
									},
								},
								ref: "#/definitions/mount",
							},
						},
					},
					Link: Link{
						Title:       "Delete all existing mounts of a server",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}/mounts",
						Rel:         "delete-all",
						Method:      "DELETE",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"POST": RouteIOAndLink{
					RouteIO: RouteIO{
						InType: JSONObject{
							Name: "CreateMountIn",
							Fields: JSONFieldList{
								JSONField{
									Name: "fs_params",
									Type: JSONObject{
										Name: "FsParams",
									},
								},
								JSONField{
									Name: "fs_type",
									Type: JSONString{},
								},
								JSONField{
									Name: "source",
									Type: JSONString{ref: "#/definitions/mount/definitions/source"},
								},
								JSONField{
									Name: "target",
									Type: JSONString{ref: "#/definitions/mount/definitions/target"},
								},
							},
						},
						OutType: JSONObject{
							Name: "Mount",
							Fields: JSONFieldList{
								JSONField{
									Name: "id",
									Type: JSONString{ref: "#/definitions/mount/definitions/id"},
								},
								JSONField{
									Name: "source",
									Type: JSONString{ref: "#/definitions/mount/definitions/source"},
								},
								JSONField{
									Name: "target",
									Type: JSONString{ref: "#/definitions/mount/definitions/target"},
								},
							},
							ref: "#/definitions/mount",
						},
					},
					Link: Link{
						Title:       "Create a new mount on a server",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}/mounts",
						Rel:         "create",
						Method:      "POST",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONArray{
							Name: "ListAllMountOut",
							Items: JSONObject{
								Name: "Mount",
								Fields: JSONFieldList{
									JSONField{
										Name: "id",
										Type: JSONString{ref: "#/definitions/mount/definitions/id"},
									},
									JSONField{
										Name: "source",
										Type: JSONString{ref: "#/definitions/mount/definitions/source"},
									},
									JSONField{
										Name: "target",
										Type: JSONString{ref: "#/definitions/mount/definitions/target"},
									},
								},
								ref: "#/definitions/mount",
							},
						},
					},
					Link: Link{
						Title:       "List existing mounts for a server",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}/mounts",
						Rel:         "list-all",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
		{
			Path: "/servers/{server-port}/mounts/{mount-id}",
			Name: "servers.one.mounts.one",
			RouteParams: []RouteParam{
				RouteParam{
					Name:    "server-port",
					Varname: "serverPort",
					Type:    JSONInteger{ref: "#/definitions/server/definitions/port"},
				},
				RouteParam{
					Name:    "mount-id",
					Varname: "mountId",
					Type:    JSONString{ref: "#/definitions/mount/definitions/id"},
				},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Mount",
							Fields: JSONFieldList{
								JSONField{
									Name: "id",
									Type: JSONString{ref: "#/definitions/mount/definitions/id"},
								},
								JSONField{
									Name: "source",
									Type: JSONString{ref: "#/definitions/mount/definitions/source"},
								},
								JSONField{
									Name: "target",
									Type: JSONString{ref: "#/definitions/mount/definitions/target"},
								},
							},
							ref: "#/definitions/mount",
						},
					},
					Link: Link{
						Title:       "Info for a mount",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}/mounts/{(#/definitions/mount/definitions/id)}",
						Rel:         "self",
						Method:      "GET",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
				"DELETE": RouteIOAndLink{
					RouteIO: RouteIO{
						OutType: JSONObject{
							Name: "Mount",
							Fields: JSONFieldList{
								JSONField{
									Name: "id",
									Type: JSONString{ref: "#/definitions/mount/definitions/id"},
								},
								JSONField{
									Name: "source",
									Type: JSONString{ref: "#/definitions/mount/definitions/source"},
								},
								JSONField{
									Name: "target",
									Type: JSONString{ref: "#/definitions/mount/definitions/target"},
								},
							},
							ref: "#/definitions/mount",
						},
					},
					Link: Link{
						Title:       "Delete an existing mount on a server",
						Description: "",
						HRef:        "/servers/{(#/definitions/server/definitions/port)}/mounts/{(#/definitions/mount/definitions/id)}",
						Rel:         "delete",
						Method:      "DELETE",
						EncType:     "application/json",
						MediaType:   "application/json",
					},
				},
			},
		},
	}
	sort.Sort(expectedResourceRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}
	if err := resourceRoutesEquals(expectedResourceRoutes, routes.ByResource()); err != nil {
		t.Error(err)
	}
}

func TestParseSchemaWithNonJSONEndpoints(t *testing.T) {
	schema := getSchema(t, "testdata/files.json")
	if t.Failed() {
		return
	}
	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	if err != nil {
		t.Error(err)
		return
	}

	expectedRoutes := Routes{
		{
			Path:        "/files",
			Name:        "files",
			RouteParams: []RouteParam{},
			Method:      "POST",
			Link: Link{
				Title:     "Create a new file using a raw binary body.",
				HRef:      "/files",
				Rel:       "create",
				Method:    "POST",
				EncType:   "application/octet-stream",
				MediaType: "application/json",
			},
			RouteIO: RouteIO{
				InputIsNotJSON: true,
				OutType: JSONObject{
					Name: "File",
					ref:  "#/definitions/file",
					Fields: JSONFieldList{ // .Name natural sort
						{Name: "content_type", Type: JSONString{ref: "#/definitions/file/definitions/contenttype"}},
						{Name: "creation_date", Type: JSONDateTime{ref: "#/definitions/file/definitions/creationdate"}},
						{Name: "id", Type: JSONString{ref: "#/definitions/file/definitions/id"}},
					},
				},
			},
		},
		{
			Path:        "/files",
			Name:        "files",
			RouteParams: []RouteParam{},
			Method:      "GET",
			Link: Link{
				Title:     "List existing files",
				HRef:      "/files",
				Rel:       "list",
				Method:    "GET",
				EncType:   "application/json",
				MediaType: "application/json",
			},
			RouteIO: RouteIO{
				OutType: JSONArray{
					Name: "ListFileOut",
					ref:  "",
					Items: JSONObject{
						Name: "File",
						ref:  "#/definitions/file",
						Fields: JSONFieldList{ // .Name natural sort
							{Name: "content_type", Type: JSONString{ref: "#/definitions/file/definitions/contenttype"}},
							{Name: "creation_date", Type: JSONDateTime{ref: "#/definitions/file/definitions/creationdate"}},
							{Name: "id", Type: JSONString{ref: "#/definitions/file/definitions/id"}},
						},
					},
				},
			},
		},
		{
			Path: "/files/{file-id}",
			Name: "files.one",
			RouteParams: []RouteParam{
				{Name: "file-id", Varname: "fileId", Type: JSONString{ref: "#/definitions/file/definitions/id"}},
			},
			Method: "GET",
			Link: Link{
				Title:     "Binary data of an existing file.",
				HRef:      "/files/{(#/definitions/file/definitions/id)}",
				Rel:       "self",
				Method:    "GET",
				EncType:   "application/json",
				MediaType: "application/octet-stream",
			},
			RouteIO: RouteIO{
				OutputIsNotJSON: true,
			},
		},
	}
	sort.Sort(expectedRoutes)

	if err := routesEquals(expectedRoutes, routes); err != nil {
		t.Error(err)
	}
}
