package dispel

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

// getSchemaString is like getSchema but takes a string as input.
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
        },
        "level_on": {
            "type": "string",
            "format": "date-time"
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
			{Name: "level_on", Type: JSONDateTime{}},
		},
	}
	sort.Sort(expectedObj.Fields)

	sp := SchemaParser{RootSchema: schema}
	obj, err := sp.JSONTypeFromSchema(schema.Title, schema, "")
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

	spellSchema, exists := schema.Properties["spell"]
	assert(t, exists, "definition %q not found in schema", "spell")
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
	ok(t, err)
	equals(t, expectedObj, obj)
}

func TestParseSchemaWithRoutesOneResource(t *testing.T) {
	schema := getSchema(t, "testdata/spells.json")

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
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedRoutes, routes)
}

func TestParseSchemaByResource(t *testing.T) {
	schema := getSchema(t, "testdata/weapons-and-armors.json")
	expectedResourceRoutes := ResourceRoutes{
		{
			Path:        "/armors",
			Name:        "armors",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIO{
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
				"GET": RouteIO{
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
			},
		},
		{
			Path: "/armors/{armor-id}",
			Name: "armors.one",
			RouteParams: []RouteParam{
				{Name: "armor-id", Varname: "armorId", Type: JSONString{ref: "#/definitions/armor/definitions/id"}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
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
				"DELETE": RouteIO{
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
			},
		},
		{
			Path:        "/weapons",
			Name:        "weapons",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIO{
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
				"GET": RouteIO{
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
			},
		},
		{
			Path: "/weapons/{weapon-id}",
			Name: "weapons.one",
			RouteParams: []RouteParam{
				{Name: "weapon-id", Varname: "weaponId", Type: JSONString{ref: "#/definitions/weapon/definitions/id"}},
			},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
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
			},
		},
	}
	sort.Sort(expectedResourceRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedResourceRoutes, routes.ByResource())
}

func TestMixedRouteParams(t *testing.T) {
	schema := getSchema(t, "testdata/one-route-mixed-params.json")

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
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedRoutes, routes)
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
		equals(t, test.ppHRef, preProcessHRefVar(test.href))
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
		if !test.valid {
			assert(t, err != nil, "Expected invalid href %q, got %q %v", test.href, vars, err)
			continue
		}
		ok(t, err)
		equals(t, test.vars, vars)
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
		ok(t, err)
		equals(t, test.name, name)
	}
}

func TestParseKrakenSchema(t *testing.T) {
	schema := getSchema(t, "testdata/kraken.json")
	expectedResourceRoutes := ResourceRoutes{
		{
			Path:        "/fileservers",
			Name:        "fileservers",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"GET": RouteIO{
					OutType: JSONArray{
						Name:  "ListAllFileServerTypeOut",
						Items: JSONString{ref: "#/definitions/fileservertype"},
					},
				},
			},
		},
		{
			Path:        "/servers",
			Name:        "servers",
			RouteParams: []RouteParam{},
			MethodRouteIOMap: MethodRouteIOMap{
				"POST": RouteIO{
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
				"DELETE": RouteIO{
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
				"GET": RouteIO{
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
				"DELETE": RouteIO{
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
				"PUT": RouteIO{
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
				"GET": RouteIO{
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
				"DELETE": RouteIO{
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
				"POST": RouteIO{
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
				"GET": RouteIO{
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
				"GET": RouteIO{
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
				"DELETE": RouteIO{
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
			},
		},
	}
	sort.Sort(expectedResourceRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedResourceRoutes, routes.ByResource())
}

func TestParseSchemaWithNonJSONEndpoints(t *testing.T) {
	schema := getSchema(t, "testdata/files.json")

	expectedRoutes := Routes{
		{
			Path:        "/files",
			Name:        "files",
			RouteParams: []RouteParam{},
			Method:      "POST",
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
			RouteIO: RouteIO{
				OutputIsNotJSON: true,
			},
		},
	}
	sort.Sort(expectedRoutes)

	sp := SchemaParser{RootSchema: schema}
	routes, err := sp.ParseRoutes()
	ok(t, err)
	equals(t, expectedRoutes, routes)
}
