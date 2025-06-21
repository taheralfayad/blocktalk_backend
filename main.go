package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/gorilla/mux"

	users "backend/api/v1/users"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

var db *sql.DB

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		log.Printf("Started %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		next.ServeHTTP(wrapped, r)

		log.Printf("Completed %s %s with status %d in %v",
			r.Method, r.URL.Path, wrapped.statusCode, time.Since(start))
	})
}

func getEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entry Retrieved")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Flip the switch on em"))
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.Use(loggingMiddleware)

	// User API routes
	myRouter.HandleFunc("/create-user", func(w http.ResponseWriter, r *http.Request) {
		users.CreateUser(w, r, db)
	}).Methods("POST")

	myRouter.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		users.LoginUser(w, r, db)
	}).Methods("POST")

	myRouter.HandleFunc("/refresh-token", users.RefreshToken).Methods("POST")

	// ========================

	// Entry API routes
	myRouter.HandleFunc("/view-entry", getEntry).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", myRouter))
}

func main() {
	var err error
	db, err = initDB()

	// retry connection to the database 10 times with a 2-second delay
	for i := 0; i < 10 && err != nil; i++ {
		log.Printf("Failed to connect to DB, attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
		err = db.Ping()
	}

	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	handleRequests()
}
