package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

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

func createEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entry Created")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Entry created successfully"))
}

func getEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entry Retrieved")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Flip the switch on em"))
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.Use(loggingMiddleware)

	myRouter.HandleFunc("/create-entry", createEntry).Methods("POST")
	myRouter.HandleFunc("/view-entry", getEntry).Methods("GET")

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", myRouter))
}

func main() {
	handleRequests()
}
