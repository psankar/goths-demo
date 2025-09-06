package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"goths-demo/sqlc/db"

	_ "modernc.org/sqlite"
)

type server struct {
	mux     *http.ServeMux
	sqldb   *sql.DB
	queries *db.Queries
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

	s.sqldb, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// create table
	if _, err := s.sqldb.Exec(string(ddl)); err != nil {
		log.Fatal(err)
	}

	s.queries = db.New(s.sqldb)

	// create users
	tx, err := s.sqldb.Begin()
	if err != nil {
		log.Fatal(err)
	}

	qtx := s.queries.WithTx(tx)
	for i := 0; i < 100; i++ {
		err = qtx.AddUser(context.Background(), db.AddUserParams{
			Username: sql.NullString{
				String: fmt.Sprintf("user%d", i),
				Valid:  true,
			},
			Password: sql.NullString{
				String: "password",
				Valid:  true,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}
}
