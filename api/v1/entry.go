package v1

import (
	"database/sql"
	"log"
	"net/http"
)

func createEntry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	err := r.ParseForm()

	if err != nil {
		log.Fatal(w, "ParseForm() error", http.StatusBadRequest)
	}
}
