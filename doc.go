// Package dispel implements code generation of server code for REST APIs written in Go.
// The generated code is based on the JSON Hyper Schema describing the API.
//
// Its primary use is through go:generate, but it's also possible to invoke the dispel command directly,
// or use this package in your code.
//
// JSON Schema
//
// Usage of this package usually begins with parsing a JSON Schema and the routes it contains:
//
//     var schema dispel.Schema
//     err := json.NewDecoder(reader).Decode(&schema)
//     // ...
//     schemaParser := &dispel.SchemaParser{RootSchema: &schema}
//     routes, err := schemaParser.ParseRoutes()
//     // ...
//
// Then you want to generate code using these routes. Dispel provides several builtin templates
// which provide code generation for registering API routes and API handlers, generating input and output types
// for the data structures of these routes, and default implementations of API handlers.
//
// Templates
//
// The TemplateBundle type is a bundle of all these templates; all templates for generating this code use the same template context:
//
//     tmpl, err := dispel.NewTemplateBundle(schemaParser)
//     // ...
//     ctx := &dispel.TemplateContext{
//         Prgm:                "dispel",
//         PkgName:             "main",
//         Routes:              routes,
//         HandlerReceiverType: "*App",
//     }
//     err := tmpl.ExecuteTemplate(os.Stdout, dispel.TemplateRoutes, ctx))
//     // ...
//
// Dispel tries to be as unopiniated as possible for the generated code API. Most of the generated code makes use
// of interfaces to let the developer implement them as he needs.
// However, it's possible to generate default implementations of these interfaces to quickly have code that compiles.
//
// This is done with DefaultImplBundle type, which works similarly to the TemplateBundle.
//
//     tmpl, err := dispel.NewDefaultImplBundle()
//     // ...
//     pkgName := "main"
//     err := tmpl.ExecuteTemplate(os.Stdout, dispel.DefaultImplMux, pkgName))
//     // ...
//
// For a more detailed usage of this package, take a look at integration_test.go and cmd/dispel source.
package dispel
