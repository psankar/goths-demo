package server

import (
	"log"
	"net/http"
)

type server struct {
	mux *http.ServeMux
}

func Run() {
	srv := server{
		mux: http.DefaultServeMux,
	}
	srv.mux.HandleFunc("/", srv.rootHandler)

	log.Println("Started server http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
