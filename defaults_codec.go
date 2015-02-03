// +build impl

package dispel

import (
	"encoding/base64"
	"encoding/json"
	"hash/fnv"
	"net/http"
)

type JSONCodec struct {
}

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
	if r.Method != "HEAD" {
		if _, err := w.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func (j *JSONCodec) Decode(w http.ResponseWriter, r *http.Request, data interface{}) error {
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		return err
	}
	return nil
}
