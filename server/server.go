package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"goths-demo/sqlc/db"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	_ "modernc.org/sqlite"
)

const (
	cookieName = "goths-session"
)

type PostMessage struct {
	Username string
	Content  string
	PostID   int64
}

type server struct {
	mux        *http.ServeMux
	sqldb      *sql.DB
	queries    *db.Queries
	nats       *nats.Conn
	natsServer *natsserver.Server
}

func Run() {
	var err error
	srv := server{
		mux: http.DefaultServeMux,
	}

	// Start NATS server
	srv.natsServer, err = natsserver.NewServer(&natsserver.Options{
		Port: 4222,
		// Other options like logging can be configured here
	})
	if err != nil {
		log.Fatal("Error starting NATS server:", err)
	}
	go srv.natsServer.Start()
	if !srv.natsServer.ReadyForConnections(_TIMEOUT) {
		log.Fatal("NATS server did not become ready")
	}

	// Connect to NATS
	srv.nats, err = nats.Connect(srv.natsServer.ClientURL())
	if err != nil {
		log.Fatal("Error connecting to NATS:", err)
	}
	defer srv.nats.Close()

	srv.mux.HandleFunc("/", srv.RootHandler)
	srv.mux.HandleFunc("GET /login", srv.LoginGetHandler)
	srv.mux.HandleFunc("POST /login", srv.LoginPostHandler)
	srv.mux.HandleFunc("GET /logout", srv.LogoutHandler)
	srv.mux.HandleFunc("GET /home", checkAuth(srv.HomeHandler))
	srv.mux.HandleFunc("POST /add-post", checkAuth(srv.AddPostHandler))
	srv.mux.HandleFunc("/timeline", checkAuth(srv.GetTimelineHandler))

	log.Println("Applying DB migrations")
	srv.initDB()

	log.Println("Started server http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", srv.mux))
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

const _TIMEOUT = 2 * time.Second