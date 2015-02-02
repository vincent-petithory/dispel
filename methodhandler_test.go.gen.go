// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package dispel

var methodHandlerTest = gofmtTmpl(asset.init(asset{Name: "methodhandler_test.go", Content: "" +
	"package dispel\n\nimport (\n\t\"net/http\"\n\t\"net/http/httptest\"\n\t\"testing\"\n)\n\nfunc newRequest(tb testing.TB, method, url string) *http.Request {\n\treq, err := http.NewRequest(method, url, nil)\n\tif err != nil {\n\t\ttb.Fatal(err)\n\t}\n\treturn req\n}\n\nfunc TestMethodHandler(t *testing.T) {\n\tvar (\n\t\tok         = \"ok\\n\"\n\t\tnotAllowed = http.StatusText(http.StatusMethodNotAllowed) + \"\\n\"\n\t\tokHandler  = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {\n\t\t\tw.Write([]byte(ok))\n\t\t})\n\t)\n\ttests := []struct {\n\t\treq     *http.Request\n\t\thandler http.Handler\n\t\tcode    int\n\t\tallow   string // Contents of the Allow header\n\t\tbody    string\n\t}{\n\t\t// No handlers\n\t\t{newRequest(t, \"GET\", \"/foo\"), MethodHandler{}, http.StatusMethodNotAllowed, \"\", notAllowed},\n\t\t{newRequest(t, \"OPTIONS\", \"/foo\"), MethodHandler{}, http.StatusOK, \"\", \"\"},\n\n\t\t// A single handler\n\t\t{newRequest(t, \"GET\", \"/foo\"), MethodHandler{Get: okHandler}, http.StatusOK, \"\", ok},\n\t\t{newRequest(t, \"POST\", \"/foo\"), MethodHandler{Get: okHandler}, http.StatusMethodNotAllowed, \"GET\", notAllowed},\n\n\t\t// Multiple handlers\n\t\t{newRequest(t, \"GET\", \"/foo\"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, \"\", ok},\n\t\t{newRequest(t, \"POST\", \"/foo\"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, \"\", ok},\n\t\t{newRequest(t, \"DELETE\", \"/foo\"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusMethodNotAllowed, \"GET, POST\", notAllowed},\n\t\t{newRequest(t, \"OPTIONS\", \"/foo\"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, \"GET, POST\", \"\"},\n\n\t\t// Override OPTIONS\n\t\t{newRequest(t, \"OPTIONS\", \"/foo\"), MethodHandler{Options: okHandler}, http.StatusOK, \"\", ok},\n\t}\n\n\tfor _, test := range tests {\n\t\trec := httptest.NewRecorder()\n\t\ttest.handler.ServeHTTP(rec, test.req)\n\t\tif test.code != rec.Code {\n\t\t\tt.Errorf(\"Expected %d, got %d\", test.code, rec.Code)\n\t\t}\n\t\tif test.req.Method == \"OPTIONS\" {\n\t\t\tif test.allow != rec.HeaderMap.Get(\"Allow\") {\n\t\t\t\tt.Errorf(\"Expected %q, got %q\", test.allow, rec.HeaderMap.Get(\"Allow\"))\n\t\t\t}\n\t\t}\n\t\tif test.body != rec.Body.String() {\n\t\t\tt.Errorf(\"Expected %q, got %q\", test.body, rec.Body.String())\n\t\t}\n\t}\n}\n" +
	""}))