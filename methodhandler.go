package main

import (
	"net/http"
	"strings"
)

// MethodHandler is an http.Handler that dispatches to a handler whose field name matches
// the name of the HTTP request's method, eg: GET
//
// If the request's method is OPTIONS and Options is not set, then the handler
// responds with a status of 200 and sets the Allow header to a comma-separated list of
// available methods.
//
// If the request's method has no handler for it, the MethodHandler responds with
// a status of 405, Method not allowed and sets the Allow header to a comma-separated list
// of available methods.
type MethodHandler struct {
	Get, Head, Post, Put, Patch, Delete, Options http.Handler
}

func (h MethodHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		if h.Get != nil {
			h.Get.ServeHTTP(w, r)
			return
		}
	case "HEAD":
		if h.Head != nil {
			h.Head.ServeHTTP(w, r)
			return
		}
		if h.Get != nil {
			h.Get.ServeHTTP(w, r)
			return
		}
	case "POST":
		if h.Post != nil {
			h.Post.ServeHTTP(w, r)
			return
		}
	case "PUT":
		if h.Put != nil {
			h.Put.ServeHTTP(w, r)
			return
		}
	case "PATCH":
		if h.Patch != nil {
			h.Patch.ServeHTTP(w, r)
			return
		}
	case "DELETE":
		if h.Delete != nil {
			h.Delete.ServeHTTP(w, r)
			return
		}
	case "OPTIONS":
		if h.Options != nil {
			h.Options.ServeHTTP(w, r)
			return
		}
		h.setAllowHeader(w.Header())
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func (h MethodHandler) setAllowHeader(header http.Header) {
	allow := make([]string, 0, 6)
	if h.Delete != nil {
		allow = append(allow, "DELETE")
	}
	if h.Get != nil {
		allow = append(allow, "GET")
	}
	if h.Head != nil {
		allow = append(allow, "HEAD")
	}
	if h.Patch != nil {
		allow = append(allow, "PATCH")
	}
	if h.Post != nil {
		allow = append(allow, "POST")
	}
	if h.Put != nil {
		allow = append(allow, "PUT")
	}
	header.Set("Allow", strings.Join(allow, ", "))

}
