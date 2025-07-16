package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"

	entry "backend/api/v1/entry"
	users "backend/api/v1/users"
	utils "backend/api/v1/utils"
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

func validateToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		fmt.Println("Validating token for request:", r.URL.Path)

		skipPaths := map[string]bool{
			"/create-user":   true,
			"/login":         true,
			"/refresh-token": true,
		}

		if skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		authToken := r.Header.Get("Authorization")

		log.Printf("Received token: %s", authToken)

		if authToken == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
			return utils.JwtSecret, nil
		})

		if err != nil || !token.Valid {
			log.Printf("Invalid token: %v", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.Use(loggingMiddleware)
	myRouter.Use(validateToken)

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
	myRouter.HandleFunc("/retrieve-entries-within-visible-bounds", func(w http.ResponseWriter, r *http.Request) {
		entry.RetrieveEntriesWithinVisibleBounds(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/create-entry", func(w http.ResponseWriter, r *http.Request) {
		entry.CreateEntry(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/autocomplete-address", entry.AutocompleteAddress).Methods("GET")

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
