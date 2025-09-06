package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"
)

type server struct {
	mux *http.ServeMux
	db  *sql.DB
}

func Run() {
	srv := server{
		mux: http.DefaultServeMux,
	}
	srv.mux.HandleFunc("/", srv.rootHandler)

	log.Println("Applying DB migrations")
	srv.initDB()

	log.Println("Started server http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *server) initDB() {
	ddl, err := os.ReadFile("sqlc/schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	s.db, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// create table
	if _, err := s.db.ExecContext(
		context.Background(),
		string(ddl),
	); err != nil {
		log.Fatal(err)
	}
}
