package v1

import (
	"backend/api/v1/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type CreateEntryRequest struct {
	Title       string   `json:"title"`
	Location    string   `json:"location"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

func AutocompleteAddress(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	tomTomApiKey := os.Getenv("TOM_TOM_API_KEY")

	url := os.Getenv("TOM_TOM_BASE_URL") + "/search/2/search/" + url.QueryEscape(query) + ".json?key=" + tomTomApiKey + "&typeahead=true" + "&limit=3" + "&countrySet=US"

	fmt.Println("Requesting TomTom API URL:", url)

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error making request to TomTom API:", err)
		http.Error(w, "Failed to fetch autocomplete suggestions", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		fmt.Println("Error response from TomTom API:", string(bodyBytes))
		http.Error(w, "Failed to fetch autocomplete suggestions", resp.StatusCode)
		return
	}

	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var tomTomResponse struct {
		Results []struct {
			Address struct {
				FreeformAddress string `json:"freeformAddress"`
			} `json:"address"`
			Position struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"position"`
		} `json:"results"`
	}

	err = json.Unmarshal(responseData, &tomTomResponse)

	if err != nil {
		fmt.Println("Error parsing TomTom API response:", err)
		http.Error(w, "Failed to parse autocomplete suggestions", http.StatusInternalServerError)
		return
	}

	fmt.Println("Parsed TomTom API response:", tomTomResponse)

	type AddressSuggestion struct {
		Address string  `json:"address"`
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
	}

	var suggestions []AddressSuggestion

	for _, result := range tomTomResponse.Results {
		suggestions = append(suggestions, AddressSuggestion{
			Address: result.Address.FreeformAddress,
			Lat:     result.Position.Lat,
			Lon:     result.Position.Lon,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(suggestions)
	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	fmt.Println("Autocomplete suggestions sent successfully")

}

func CreateEntry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var payload CreateEntryRequest

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		fmt.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Title == "" || payload.Location == "" || payload.Description == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	fmt.Println("Received payload:", payload)

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM entry
			WHERE ST_DWithin(location, geography(ST_MakePoint($1, $2)), 50)
		);
	`

	var entryAlreadyExists bool
	err = db.QueryRow(query, payload.Longitude, payload.Latitude).Scan(&entryAlreadyExists)
	if err != nil {
		log.Printf("Database error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if entryAlreadyExists {
		fmt.Println("Entry already exists in this location")
		http.Error(w, "Entry already exists in this location", http.StatusConflict)
		return
	}

	token, err := jwt.Parse(r.Header.Get("Authorization"), func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})

	if err != nil || !token.Valid {
		fmt.Println("Invalid token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	claims := token.Claims.(jwt.MapClaims)

	username := claims["username"].(string)

	userID := 0

	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1
	`, username).Scan(&userID)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
			return
		}
		fmt.Println("DB query error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	fmt.Println("User ID:", userID)

	tagInDatabase := ""

	for _, tag := range payload.Tags {
		err = db.QueryRow(`
			INSERT INTO tags (name) 
			VALUES ($1) 
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name 
			RETURNING name
		`, tag).Scan(&tagInDatabase) // Functions as a "get or create" for tags

		if err != nil {
			fmt.Println("DB insert error for tags:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		print("Tag in database:", tagInDatabase)
	}

	entryID := 0

	err = db.QueryRow(`
		INSERT INTO entry (title, address, content, creator_id, location) 
		VALUES ($1, $2, $3, $4, ST_SetSRID(ST_MakePoint($5, $6), 4326))
		RETURNING id
		`, payload.Title, payload.Location, payload.Description, userID, payload.Longitude, payload.Latitude).Scan(&entryID)

	if err != nil {
		fmt.Println("DB insert error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO tags_entry (entry_id, tag_id)
		SELECT $1, id FROM tags WHERE name = $2
	`, entryID, tagInDatabase)

	if err != nil {
		fmt.Println("DB insert error for entry_tags:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Entry created successfully"))
}

func RetrieveEntriesWithinVisibleBounds(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	type Bounds struct {
		North float64 `json:"north"`
		South float64 `json:"south"`
		East  float64 `json:"east"`
		West  float64 `json:"west"`
	}

	var bounds Bounds

	err := json.NewDecoder(r.Body).Decode(&bounds)

	fmt.Println("Received bounds:", bounds)

	if err != nil {
		fmt.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT id, address, content, upvotes, downvotes, views, date_created, creator_id,
			ST_X(location::geometry) AS longitude,
			ST_Y(location::geometry) AS latitude
		FROM entry
		WHERE location::geometry && ST_MakeEnvelope($1, $2, $3, $4, 4326)
	`,
		bounds.West,
		bounds.South,
		bounds.East,
		bounds.North,
	)

	if err != nil {
		fmt.Println("Query error:", err)
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Entry struct {
		ID          int     `json:"id"`
		Address     string  `json:"address"`
		Content     string  `json:"content"`
		Upvotes     int     `json:"upvotes"`
		Downvotes   int     `json:"downvotes"`
		Views       int     `json:"views"`
		DateCreated string  `json:"date_created"`
		CreatorID   int     `json:"creator_id"`
		Longitude   float64 `json:"longitude"`
		Latitude    float64 `json:"latitude"`
	}

	var entries []Entry

	for rows.Next() {
		var entry Entry
		err := rows.Scan(&entry.ID,
			&entry.Address,
			&entry.Content,
			&entry.Upvotes,
			&entry.Downvotes,
			&entry.Views,
			&entry.DateCreated,
			&entry.CreatorID,
			&entry.Longitude,
			&entry.Latitude)
		if err != nil {
			fmt.Println("Row scan error:", err)
			http.Error(w, "Failed to read entry data", http.StatusInternalServerError)
			return
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Row iteration error:", err)
		http.Error(w, "Failed to read entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(entries) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = json.NewEncoder(w).Encode(entries)

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}
