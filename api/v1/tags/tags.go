package v1

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type Tag struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func RetrieveTags(w http.ResponseWriter, r *http.Request, db *sql.DB) {

	rows, err := db.Query(`
		SELECT * FROM tags;
	`)

	if err != nil {
		fmt.Println("Error retrieving tags", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	var tags []Tag

	for rows.Next() {
		var tag Tag

		err := rows.Scan(
			&tag.Id,
			&tag.Name,
		)

		if err != nil {
			fmt.Println("Error scanning tag into JSON")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

		tags = append(tags, tag)

	}

	w.Header().Set("Content-Type", "application/json")

	if len(tags) == 0 {
		w.WriteHeader(http.StatusNoContent)
	}

	err = json.NewEncoder(w).Encode(tags)

	if err != nil {
		fmt.Println("Error encoding JSON")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

}
