package dispel

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRequest(tb testing.TB, method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	ok(tb, err)
	if err != nil {
		panic(err)
	}
	return req
}

func TestMethodHandler(t *testing.T) {
	var (
		ok         = "ok\n"
		notAllowed = http.StatusText(http.StatusMethodNotAllowed) + "\n"
		okHandler  = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Write([]byte(ok))
		})
	)
	tests := []struct {
		req     *http.Request
		handler http.Handler
		code    int
		allow   string // Contents of the Allow header
		body    string
	}{
		// No handlers
		{newRequest(t, "GET", "/foo"), MethodHandler{}, http.StatusMethodNotAllowed, "", notAllowed},
		{newRequest(t, "OPTIONS", "/foo"), MethodHandler{}, http.StatusOK, "", ""},

		// A single handler
		{newRequest(t, "GET", "/foo"), MethodHandler{Get: okHandler}, http.StatusOK, "", ok},
		{newRequest(t, "POST", "/foo"), MethodHandler{Get: okHandler}, http.StatusMethodNotAllowed, "GET", notAllowed},

		// Multiple handlers
		{newRequest(t, "GET", "/foo"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, "", ok},
		{newRequest(t, "POST", "/foo"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, "", ok},
		{newRequest(t, "DELETE", "/foo"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusMethodNotAllowed, "GET, POST", notAllowed},
		{newRequest(t, "OPTIONS", "/foo"), MethodHandler{Get: okHandler, Post: okHandler}, http.StatusOK, "GET, POST", ""},

		// Override OPTIONS
		{newRequest(t, "OPTIONS", "/foo"), MethodHandler{Options: okHandler}, http.StatusOK, "", ok},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		test.handler.ServeHTTP(rec, test.req)
		equals(t, test.code, rec.Code)
		if test.req.Method == "OPTIONS" {
			equals(t, test.allow, rec.HeaderMap.Get("Allow"))
		}
		equals(t, test.body, rec.Body.String())
	}
}
