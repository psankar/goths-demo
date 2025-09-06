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

const (
	cookieName = "goths-session"
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
	srv.mux.HandleFunc("/", srv.RootHandler)
	srv.mux.HandleFunc("GET /login", srv.LoginGetHandler)
	srv.mux.HandleFunc("POST /login", srv.LoginPostHandler)
	srv.mux.HandleFunc("GET /logout", srv.LogoutHandler)
	srv.mux.HandleFunc("GET /home", checkAuth(srv.HomeHandler))
	srv.mux.HandleFunc("POST /posts", checkAuth(srv.AddPostHandler))

	log.Println("Applying DB migrations")
	srv.initDB()

	log.Println("Started server http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (srv *server) initDB() {
	ddl, err := os.ReadFile("sqlc/schema.sql")
	if err != nil {
		log.Fatal(err)
	}

	srv.sqldb, err = sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}

	// create table
	if _, err := srv.sqldb.Exec(string(ddl)); err != nil {
		log.Fatal(err)
	}

	srv.queries = db.New(srv.sqldb)

	// create users
	tx, err := srv.sqldb.Begin()
	if err != nil {
		log.Fatal(err)
	}

	qtx := srv.queries.WithTx(tx)
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
