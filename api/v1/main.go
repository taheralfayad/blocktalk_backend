package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func createEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entry Created")
}

func getEntry(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Entry Retrieved")
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)

	myRouter.HandleFunc("/create-entry", createEntry)
	myRouter.HandleFunc("/view-entry", getEntry)

	log.Fatal(http.ListenAndServe(":8080", myRouter))
}

func main() {
	handleRequests()
}
