package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type config struct {
	port     int
	taskTime int
}

type application struct {
	config    config
	logger    *log.Logger
	taskQueue chan int
	db        *sql.DB
}

func main() {
	var taskQueue = make(chan int, 100)

	var config config
	var dbName string

	flag.IntVar(&config.port, "port", 4000, "server port address")
	flag.IntVar(&config.taskTime, "taskTime", 30, "time in seconds for long running task")
	flag.StringVar(&dbName, "dbName", "data.db", "name of sqlite3 database")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	db, err := openDB(fmt.Sprintf("../%s", dbName))
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	app := &application{
		config:    config,
		logger:    logger,
		taskQueue: taskQueue,
		db:        db,
	}

	go app.taskServer()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", app.ping)
	mux.HandleFunc("POST /tasks", app.AuthMiddleware(app.createTask))
	mux.HandleFunc("GET /tasks/{id}", app.AuthMiddleware(app.getTask))
	mux.HandleFunc("POST /token", app.getToken)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.port),
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("starting server on port %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func openDB(dbName string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	// create tasks table
	_, err = db.Exec(`
    CREATE TABLE IF NOT EXISTS Tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    input INTEGER NOT NULL,
    output INTEGER DEFAULT 0,
    is_done INTEGER DEFAULT 0
    )
  `)
	if err != nil {
		return nil, err
	}

	// Create the auth table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS Auth (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			password BLOB NOT NULL,
			token TEXT,
			expiry DATETIME
		)
	`)
	if err != nil {
		return nil, err
	}
	// insert user in auth table
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), 16)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`
		INSERT OR IGNORE INTO Auth (username, password)
		VALUES (?, ?)
	`, "admin", []byte(hash))

	if err != nil {
		return nil, err
	}

	return db, nil
}

func (app *application) taskServer() {
	app.logger.Println("stating task server...")
	for id := range app.taskQueue {
		go func(i int) {
			app.logger.Printf("starting task for task %d", i)
			time.Sleep(time.Duration(app.config.taskTime) * time.Second)
			t := NewTask(app.db)
			t.Update(i)
			app.logger.Printf("task completed for task %d", i)
		}(id)
	}
}

func (app *application) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		authParts := strings.Split(authorizationHeader, " ")
		if len(authParts) != 2 || authParts[0] != "Bearer" {
			app.logger.Println("token is malformed")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		token := authParts[1]
		if !app.validToken(token) {
			app.logger.Println("token is invalid")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})

}
func (app *application) validToken(token string) bool {
	err := app.db.QueryRow(
		"SELECT token FROM Auth WHERE token = ? AND expiry > datetime('now')", token).
		Scan(&token)
	if err != nil {
		app.logger.Printf("%s", err.Error())
		return false
	}
	return true
}
