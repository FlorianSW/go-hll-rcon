package response

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

const (
	contentTypeHeader  = "Content-Type"
	defaultContentType = "application/octet-stream"
)

var (
	Jr = &Response{
		ContentType: "application/json",
		ErrorWrapper: func(m string) []byte {
			err := map[string]string{"message": m}
			d, _ := json.Marshal(&err)
			return d
		},
	}
)

type ErrorWrapper func(message string) []byte

type Response struct {
	ContentType  string
	ErrorWrapper ErrorWrapper
}

func (r *Response) Ok(w http.ResponseWriter, content []byte) {
	r.Respond(w, http.StatusOK, content)
}

func (r *Response) NotFound(w http.ResponseWriter, content ...[]byte) {
	r.Respond(w, http.StatusNotFound, content...)
}

func (r *Response) InternalServerError(w http.ResponseWriter, content ...[]byte) {
	r.Respond(w, http.StatusInternalServerError, content...)
}

func (r *Response) MethodNotAllowed(w http.ResponseWriter) {
	r.Respond(w, http.StatusMethodNotAllowed)
}

func (r *Response) Respond(w http.ResponseWriter, code int, content ...[]byte) {
	w.WriteHeader(code)
	if code > 399 && r.ErrorWrapper != nil {
		wrapped := r.ErrorWrapper(string(bytes.Join(content, nil)))
		content = [][]byte{wrapped}
	}
	if len(content) > 0 {
		r.setContentType(w)
	}

	for _, c := range content {
		_, err := w.Write(c)
		if err != nil {
			log.Printf("writing content failed: %v", err)
			break
		}
	}
}

func (r *Response) setContentType(w http.ResponseWriter) {
	switch {
	case w.Header().Get(contentTypeHeader) != "":
		return
	case r == nil || r.ContentType == "":
		w.Header().Set(contentTypeHeader, defaultContentType)
	default:
		w.Header().Set(contentTypeHeader, r.ContentType)
	}
}

func B(c string) []byte {
	return []byte(c)
}
