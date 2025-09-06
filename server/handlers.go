package server

import "net/http"

func (server) rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello world"))
}
