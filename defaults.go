package dispel

import (
	"go/format"
	"strings"
)

//go:generate -command asset go run ./asset.go
//go:generate asset --var=MethodHandlerContent --wrap=gofmt methodhandler.go
//go:generate asset --var=MethodHandlerTestContent --wrap=gofmt methodhandler_test.go
//go:generate asset --var=DefaultsMux --wrap=gofmt defaults_mux.go
//go:generate asset --var=DefaultsCodec --wrap=gofmt defaults_codec.go

func gofmt(a asset) string {
	b, err := format.Source([]byte(a.Content))
	if err != nil {
		panic(err)
	}
	return strings.Replace(string(b), "package dispel", "package %s", 1)
}
