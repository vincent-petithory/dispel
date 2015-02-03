## Dispel [![GoDoc](https://godoc.org/github.com/vincent-petithory/dispel?status.png)](https://godoc.org/github.com/vincent-petithory/dispel)

This project aims to generate server code for REST APIs written in Go, based on a JSON Schema describing the API.

Though already usable, this is still a work in progress and APIs are unstable.
This will be updated in time when things stabilize.

## JSON Schema supported/unsupported features

* fetching referenced $schema _NOT_ supported
* absolute references
* reference to property of instance schema
