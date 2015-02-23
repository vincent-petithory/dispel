## Dispel [![Build Status](https://travis-ci.org/vincent-petithory/dispel.svg?branch=master)](https://travis-ci.org/vincent-petithory/dispel)

This project aims to generate server code for REST APIs written in Go, based on a JSON Schema describing the API.

Though already usable, this is still a work in progress and APIs are unstable.
This will be updated in time when things stabilize.

Package documentation: [![GoDoc](https://godoc.org/github.com/vincent-petithory/dispel?status.png)](https://godoc.org/github.com/vincent-petithory/dispel)

Command documentation: [![GoDoc](https://godoc.org/github.com/vincent-petithory/dispel/cmd/dispel?status.png)](https://godoc.org/github.com/vincent-petithory/dispel/cmd/dispel)

## JSON Schema supported/unsupported features

* fetching referenced $schema _NOT_ supported
* absolute references
* reference to property of instance schema

## TODO

 * [x] Ignore resources with MediaType not application/json
 * [ ] Add var type to route param
 * [ ] Preserve order of json object keys in structs
 * [ ] allow customize generate names. Possible solutions: text/template or program through stdin, stdout?
 * [ ] generate blank project to serve as godoc documentation for interfaces and default implementations
 * [x] support format="date-time" => time.Time
 * [ ] (maybe) support nullable types
 * [ ] support bare type="object" => map[string]interface{}
