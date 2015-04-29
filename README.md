## Dispel [![Build Status](https://travis-ci.org/vincent-petithory/dispel.svg?branch=master)](https://travis-ci.org/vincent-petithory/dispel)

Dispel is a tool which generates server code for REST APIs written in Go, based on a JSON Schema describing the API.

Dispel has a few goals, listed below:

* generate code that you own: you're not locked in (as you would be with a framework). Use it or leave with the code it generated for you.
* stay in sync with the spec you wrote, so you can be sure clients of your spec talk the same protocol.
* be "package agnostic": generated code will be as flexible as possible, mostly through the use of interfaces.
* don't be silly and provide out-of-the-box implementations: despite it's flexible, dispel provides default implementations for its interfaces.

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
 * [ ] support nullable types
 * [ ] support bare type="object" => map[string]interface{}
