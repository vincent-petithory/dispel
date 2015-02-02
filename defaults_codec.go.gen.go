// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package dispel

var defaultsCodec = gofmtTmpl(asset.init(asset{Name: "defaults_codec.go", Content: "" +
	"package dispel\n\nimport (\n\t\"encoding/base64\"\n\t\"encoding/json\"\n\t\"hash/fnv\"\n\t\"net/http\"\n)\n\ntype JSONCodec struct {\n}\n\nfunc handleEtag(w http.ResponseWriter, r *http.Request) (ok bool) {\n\tif r.Method != \"GET\" && r.Method != \"HEAD\" {\n\t\treturn\n\t}\n\tetag := w.Header().Get(\"Etag\")\n\tif etag == \"\" {\n\t\treturn\n\t}\n\tinm := r.Header.Get(\"If-None-Match\")\n\tif inm == \"\" {\n\t\treturn\n\t}\n\n\tif inm == etag || inm == \"*\" {\n\t\tw.Header().Del(\"Content-Type\")\n\t\tw.Header().Del(\"Content-Length\")\n\t\tw.WriteHeader(http.StatusNotModified)\n\t\tok = true\n\t}\n\treturn\n}\n\nfunc makeEtag(b []byte) string {\n\th := fnv.New64a()\n\th.Write(b)\n\treturn `\"` + base64.StdEncoding.EncodeToString(h.Sum(nil)) + `\"`\n}\n\nfunc (j *JSONCodec) Encode(w http.ResponseWriter, r *http.Request, data interface{}, code int) error {\n\tb, err := json.Marshal(data)\n\tif err != nil {\n\t\treturn err\n\t}\n\n\tif w.Header().Get(\"ETag\") == \"\" {\n\t\tw.Header().Set(\"ETag\", makeEtag(b))\n\t}\n\tif code == 0 {\n\t\tcode = http.StatusOK\n\t}\n\n\tif code == http.StatusOK {\n\t\tif ok := handleEtag(w, r); ok {\n\t\t\treturn nil\n\t\t}\n\t}\n\tw.Header().Set(\"Content-Type\", \"application/json; charset=utf-8\")\n\tw.WriteHeader(code)\n\tif r.Method != \"HEAD\" {\n\t\tif _, err := w.Write(b); err != nil {\n\t\t\treturn err\n\t\t}\n\t}\n\treturn nil\n}\n\nfunc (j *JSONCodec) Decode(w http.ResponseWriter, r *http.Request, data interface{}) error {\n\tdefer r.Body.Close()\n\tif err := json.NewDecoder(r.Body).Decode(data); err != nil {\n\t\treturn err\n\t}\n\treturn nil\n}\n" +
	""}))
