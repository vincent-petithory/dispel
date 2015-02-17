// The dispel command generates source code based on a JSON Hyper-Schema for quickly building REST APIs in Go.
//
// It requires a unique argument, SCHEMA, which is the path to the JSON Hyper-Schema.
//
// It is best used in conjunction with go generate, by making use of $GOPACKAGE and $GOFILE envvars.
//
// Flags
//
// The --version flag makes dispel to print the API version of its generated code, and exits. See the Version constant in the github.com/vincent-petithory/dispel package for its meaning.
//
// The -v flag makes dispel more verbose about what the entities it discovers while parsing the json schema.
//
// The -t flag specifies which templates to execute, with a comma-separated list of template names.
// The names must be in the following list:
//
//     handlerfuncs
//     handlers
//     routes
//     types
//
//
// If empty (the default), none is executed. If set to the special value all, all known templates are executed.
// dispel will write a file in the package dir (see -pp flag) for each name provided with a filename using the pattern {prefix}{name}.go, where prefix is defined by the -p flag.
//
// The -d flag specifies which default implementations provided by dispel to execute,
// like -t, using a comma-separated list of default implementation names.
// The names must be in the following list:
//
//     defaults_codec
//     defaults_mux
//     methodhandler
//     methodhandler_test
//
//
// If empty (the default), none is executed. If set to the special value all, all default implementations are executed.
// dispel will write a file in the package dir (see -pp flag) for each default implementation
// with a filename using the pattern {impl-name}.go
//
// The -p flag specifies which prefix to use for each generated template file. By default, it is set to 'dispel_'.
// This doesn't apply to default implementations, which have fixed names.
//
// The -hrt flag specifies the Go type in the target package which
// will be the receiver for the handler functions dispel generates.
// For example, with a value of *AppHandlers, dispel will generate something like:
//
//     func (ah *AppHandlers) getUsers(w http.ResponseWriter, r *http.Request, ....
//
//
// The -pp flag specifies which package dir to generate and analyze code into.
// It is mandatory to set this flag if dispel is not invoked with go:generate.
// If set when dispel is invoked with go:generate, it overrides the package path resolved from $GOFILE.
//
// The -pn flag specifies the package name of the code generated by dispel.
// It is mandatory to set a value if not invoked with go:generate.
// If set when dispel is invoked with go:generate, it overrides the value of $GOPACKAGE.
//
// The -f flag specifies the path to the file for an alternate format for the template to use, using the Go template syntax.
// If the value is -, then the template is read from STDIN.
// If set, then -t and -d flags are ignored: only this template is executed. The result is printed to what the -o flag is set to, which by default is STDOUT.
//
// The -o flag is only useful when -f is specified. It specifies a path where to write the output from -f.
// By default, its value is -, which means it writes to STDOUT.
//
// The context passed to the template is TemplateContext.
//
// Template Context
//
// The following struct is passed to the templates:
//
//     type TemplateContext struct {
//         Prgm                string   // name of the program generating the source
//         PkgName             string   // package name for which source code is generated
//         Routes              Routes   // routes parsed by the SchemaParser
//         HandlerReceiverType string   // type which acts as the receiver of the handler funcs.
//         ExistingHandlers    []string // list of existing handler funcs in the target package, with HandlerReceiverType as the receiver
//         ExistingTypes       []string // list of existing types in the target package.
//     }
//
// The template has those functions available:
//
//  * tolower            : calls strings.ToLower
//  * capitalize         : uppercase the first rune of a string
//  * symbolName         : uppercase each rune following one of ".- ", then uppercase the first rune 
//  * hasItem            : takes 2 arguments: ([]string, string); returns true if string is one of the elements of []string
//  * varname            : creates a short variable name from a type. e.g MyLongType would return mlt
//  * printTypeDef       : prints a valid Go type from a JSONType
//  * printTypeName      : prints the name of the Go type for a JSONType
//  * printSmartDerefType: is like printTypeName, but if the argument is a JSONObject, it return *TheType instead of TheType.
//
// For more information, see the documentation of the github.com/vincent-petithory/dispel package's TemplateContext type.
package main
