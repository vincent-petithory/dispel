// +build impl

package dispel

import (
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"net/http"
)

// JSONCodec represents a codec for http request decoding and response encoding using JSON.
//
// JSONCodec relies on encoding/json in its implementation.
type JSONCodec struct{}

func handleEtag(w http.ResponseWriter, r *http.Request) (ok bool) {
	if r.Method != "GET" && r.Method != "HEAD" {
		return
	}
	etag := w.Header().Get("Etag")
	if etag == "" {
		return
	}
	inm := r.Header.Get("If-None-Match")
	if inm == "" {
		return
	}

	if inm == etag || inm == "*" {
		w.Header().Del("Content-Type")
		w.Header().Del("Content-Length")
		w.WriteHeader(http.StatusNotModified)
		ok = true
	}
	return
}

func makeEtag(b []byte) string {
	h := fnv.New64a()
	h.Write(b)
	return `"` + base64.StdEncoding.EncodeToString(h.Sum(nil)) + `"`
}

// Encode implements the HTTPEncoder interface with JSON encoding.
//
// It writes to the response writer using
// encoding/json.Marshal(), handles Etags, sets inconditionnally a "application/json; charset=utf-8" Content-Type header.
// It skips writing a response body if any of the conditions are met:
//
//  * the status code is [100, 200)
//  * the status code is http.StatusNoContent (204) or http.StatusNotModified (304)
//  * the request method is HEAD.
func (j *JSONCodec) Encode(w http.ResponseWriter, r *http.Request, data interface{}, code int) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if w.Header().Get("ETag") == "" {
		w.Header().Set("ETag", makeEtag(b))
	}
	if code == 0 {
		code = http.StatusOK
	}

	if code == http.StatusOK {
		if ok := handleEtag(w, r); ok {
			return nil
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	switch {
	case code >= 100 && code <= 199:
		return nil
	case code == 204:
		return nil
	case code == 304:
		return nil
	case r.Method == "HEAD":
		return nil
	default:
		if _, err := w.Write(b); err != nil {
			return err
		}
		return nil
	}
}

// Decode implements the HTTPDecoder interface with JSON decoding.
//
// It simply decodes the request body using json.NewDecoder() and closes it.
func (j *JSONCodec) Decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		return err
	}
	return nil
}
