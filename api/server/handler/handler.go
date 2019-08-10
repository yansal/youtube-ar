package handler

import (
	"net/http"
)

func serveHTTP(w http.ResponseWriter, r *http.Request, fn handlerFunc) {
	resp, err := fn(r)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.code)
		w.Write(resp.body)
		return
	}
	herr, ok := err.(httpError)
	if !ok {
		herr = httpError{
			code: http.StatusInternalServerError,
			err:  err,
		}
	}
	w.WriteHeader(herr.code)
	w.Write([]byte(herr.Error()))
}

type handlerFunc func(*http.Request) (*response, error)

type response struct {
	body []byte
	code int
}

type httpError struct {
	code int
	err  error
}

func (e httpError) Error() string {
	if e.err == nil {
		return http.StatusText(e.code)
	}
	return e.err.Error()
}
