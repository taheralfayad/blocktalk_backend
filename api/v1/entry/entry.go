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
	"github.com/lithammer/fuzzysearch/fuzzy"
)

type CreateEntryRequest struct {
	Title       string   `json:"title"`
	Location    string   `json:"location"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
}

type City struct {
	Name         string  `json:"city"`
	StateId      string  `json:"state_id"`
	StateName    string  `json:"state_name"`
	CountyFips   int     `json:"county_fips"`
	CountyName   string  `json:"county_name"`
	Latitude     float64 `json:"lat"`
	Longitude    float64 `json:"lng"`
	CityAscii    string  `json:"city_ascii"`
	Population   int     `json:"population"`
	Density      float64 `json:"density"`
	Timezone     string  `json:"timezone"`
	Ranking      int     `json:"ranking"`
	Id           int     `json:"id"`
	Source       string  `json:"source"`
	Military     bool    `json:"military"`
	Incorporated bool    `json:"incorporated"`
}

type Comment struct {
	ID        int    `json:"id"`
	EntryID   int    `json:"entry_id"`
	UserID    int    `json:"user_id"`
	ParentID  *int   `json:"parent_id,omitempty"`
	Context   string `json:"context"`
	Upvotes   int    `json:"upvotes"`
	Downvotes int    `json:"downvotes"`
	Type      string `json:"type"`
}

type Entry struct {
	ID               int       `json:"id"`
	Title            string    `json:"title"`
	Address          string    `json:"address"`
	Content          string    `json:"content"`
	Upvotes          int       `json:"upvotes"`
	Downvotes        int       `json:"downvotes"`
	NumberOfComments int       `json:"number_of_comments"`
	Views            int       `json:"views"`
	DateCreated      string    `json:"date_created"`
	Username         string    `json:"username"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Longitude        float64   `json:"longitude"`
	Latitude         float64   `json:"latitude"`
	Tags             []string  `json:"tags,omitempty"`
	Comments         []Comment `json:"comments,omitempty"`
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

	tagsInDatabase := []string{}

	for _, tag := range payload.Tags {
		var tmp string
		err = db.QueryRow(`
			INSERT INTO tags (name) 
			VALUES ($1) 
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name 
			RETURNING name
		`, tag).Scan(&tmp) // Functions as a "get or create" for tags

		tagsInDatabase = append(tagsInDatabase, tmp)

		if err != nil {
			fmt.Println("DB insert error for tags:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

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

	for _, tagInDatabase := range tagsInDatabase {
		_, err = db.Exec(`
			INSERT INTO tags_entry (entry_id, tag_id)
			SELECT $1, id FROM tags WHERE name = $2
		`, entryID, tagInDatabase)
		if err != nil {
			fmt.Println("DB insert error for entry_tags:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
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
		SELECT entry.id, address, content, views, date_created, username, first_name, last_name,
			ST_X(location::geometry) AS longitude,
			ST_Y(location::geometry) AS latitude
		FROM entry
		JOIN users ON entry.creator_id = users.id
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

	var entries []Entry

	for rows.Next() {
		var entry Entry
		err := rows.Scan(&entry.ID,
			&entry.Address,
			&entry.Content,
			&entry.Views,
			&entry.DateCreated,
			&entry.Username,
			&entry.FirstName,
			&entry.LastName,
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

func RetrieveCity(w http.ResponseWriter, r *http.Request) {
	var cities []City

	jsonFile, err := os.Open("/app/static/us_cities.json")

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &cities)

	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	query := r.URL.Query().Get("city")

	fmt.Println("Received query for city:", query)

	var cityNames []string

	for _, city := range cities {
		cityNames = append(cityNames, city.Name)
	}

	matches := fuzzy.RankFind(query, cityNames)

	if len(matches) > 3 {
		matches = matches[:3] // Limit to top 3 matches
	}

	var results []string

	for _, match := range matches {
		results = append(results, match.Target)
	}

	w.Header().Set("Content-Type", "application/json")

	if len(results) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err = json.NewEncoder(w).Encode(results)

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)

}

func RetrieveFeed(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	location := r.URL.Query().Get("location")
	distance := r.URL.Query().Get("distance")

	fmt.Println("Received distance:", distance)

	jsonFile, err := os.Open("/app/static/us_cities.json")

	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var cities []City

	err = json.Unmarshal(byteValue, &cities)

	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var city City

	for _, c := range cities {
		if c.Name == location {
			city = c
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if city.Name == "" {
		fmt.Println("City not found:", location)
		http.Error(w, "City not found", http.StatusNotFound)
		return
	}

	rows, err := db.Query(`
		SELECT entry.id as id, address, content, views, date_created,
			username, first_name, last_name,
			ST_X(location::geometry) AS longitude,
			ST_Y(location::geometry) AS latitude
		FROM entry
		JOIN users ON entry.creator_id = users.id
		WHERE ST_DWithin(
			location,
			ST_MakePoint($1, $2)::geography,
			$3 * 1609.34
		)
	`, city.Longitude, city.Latitude, distance)

	if err != nil {
		fmt.Println("Query error:", err)
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var entries []Entry

	for rows.Next() {
		var numberOfComments int

		var entry Entry
		err := rows.Scan(&entry.ID,
			&entry.Address,
			&entry.Content,
			&entry.Views,
			&entry.DateCreated,
			&entry.Username,
			&entry.FirstName,
			&entry.LastName,
			&entry.Longitude,
			&entry.Latitude)
		if err != nil {
			fmt.Println("Row scan error:", err)
			http.Error(w, "Failed to read entry data", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM conversation
			WHERE entry_id = $1
		`, entry.ID).Scan(&numberOfComments)

		if err != nil {
			fmt.Println("DB query error for number of comments:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		entry.NumberOfComments = numberOfComments

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		fmt.Println("Row iteration error:", err)
		http.Error(w, "Failed to read entries", http.StatusInternalServerError)
		return
	}

	for i, entry := range entries {
		var upvotes, downvotes int

		err = db.QueryRow(`
			SELECT COUNT(*) as number_of_upvotes
			FROM entry_interactions
			WHERE entry_id = $1 AND interaction_type = 'upvote'
		`, entry.ID).Scan(&upvotes)

		if err != nil {
			fmt.Println("DB query error for upvotes:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			SELECT COUNT(*) as number_of_downvotes
			FROM entry_interactions
			WHERE entry_id = $1 AND interaction_type = 'downvote'
		`, entry.ID).Scan(&downvotes)

		if err != nil {
			fmt.Println("DB query error for downvotes:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		entries[i].Upvotes = upvotes
		entries[i].Downvotes = downvotes
	}

	if len(entries) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	fmt.Println(entries)

	err = json.NewEncoder(w).Encode(entries)

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Feed entries sent successfully")
}

func RetrieveEntry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	entryID := r.URL.Query().Get("id")

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	var entry Entry

	err := db.QueryRow(`
		SELECT entry.id, address, content, views, date_created,
			username, first_name, last_name, title,
			ST_X(location::geometry) AS longitude,
			ST_Y(location::geometry) AS latitude
		FROM entry
		JOIN users ON entry.creator_id = users.id
		WHERE entry.id = $1
	`, entryID).Scan(&entry.ID,
		&entry.Address,
		&entry.Content,
		&entry.Views,
		&entry.DateCreated,
		&entry.Username,
		&entry.FirstName,
		&entry.LastName,
		&entry.Title,
		&entry.Longitude,
		&entry.Latitude)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		fmt.Println("DB query error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var tags []string

	rows, err := db.Query(`
		SELECT tags.name
		FROM tags
		JOIN tags_entry ON tags.id = tags_entry.tag_id
		WHERE tags_entry.entry_id = $1
	`, entry.ID)

	if err != nil {
		fmt.Println("DB query error for tags:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	for rows.Next() {
		var tag string
		err := rows.Scan(&tag)
		if err != nil {
			fmt.Println("Row scan error for tags:", err)
			http.Error(w, "Failed to read tags", http.StatusInternalServerError)
			return
		}
		tags = append(tags, tag)
	}

	entry.Tags = tags

	var upvotes, downvotes int

	err = db.QueryRow(`
		SELECT COUNT(*) as number_of_upvotes
		FROM entry_interactions
		WHERE entry_id = $1 AND interaction_type = 'upvote'
	`, entry.ID).Scan(&upvotes)

	if err != nil {
		fmt.Println("DB query error for upvotes:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`
		SELECT COUNT(*) as number_of_downvotes
		FROM entry_interactions
		WHERE entry_id = $1 AND interaction_type = 'downvote'
	`, entry.ID).Scan(&downvotes)

	if err != nil {
		fmt.Println("DB query error for downvotes:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	entry.Upvotes = upvotes
	entry.Downvotes = downvotes

	if err != nil {
		fmt.Println("DB query error for downvotes:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(entry)

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Entry retrieved successfully")
}

func VoteEntry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
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

	type UpvoteRequest struct {
		EntryID         string `json:"entry_id"`
		InteractionType string `json:"interaction_type"`
	}

	var req UpvoteRequest

	err = json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		fmt.Println("Error decoding request body:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.EntryID == "" || req.InteractionType == "" {
		http.Error(w, "Entry ID and interaction type are required", http.StatusBadRequest)
		return
	}

	entryID := req.EntryID

	if entryID == "" {
		http.Error(w, "Entry ID is required", http.StatusBadRequest)
		return
	}

	var currentInteraction string

	err = db.QueryRow(`
		SELECT interaction_type
		FROM entry_interactions
		WHERE user_id = $1 AND entry_id = $2
	`, userID, entryID).Scan(&currentInteraction)

	if err != nil && err != sql.ErrNoRows {
		fmt.Println("DB query error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if currentInteraction == req.InteractionType {
		_, err = db.Exec(`
			DELETE FROM entry_interactions 
			WHERE user_id = $1 AND entry_id = $2
		`, userID, entryID)
	} else if currentInteraction != "" {
		_, err = db.Exec(`
			UPDATE entry_interactions
			SET interaction_type = $3, created_at = CURRENT_TIMESTAMP
			WHERE user_id = $1 AND entry_id = $2
		`, userID, entryID, req.InteractionType)
	} else {
		_, err = db.Exec(`
			INSERT INTO entry_interactions (entry_id, user_id, interaction_type)
			VALUES ($1, $2, $3)
		`, entryID, userID, req.InteractionType)
	}

	if err != nil {
		fmt.Println("DB exec error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var upvotes int
	var downvotes int

	err = db.QueryRow(`
		SELECT COUNT(*) as number_of_upvotes
		FROM entry_interactions
		WHERE entry_id = $1 AND interaction_type = 'upvote'
	`, entryID).Scan(&upvotes)

	if err != nil {
		fmt.Println("DB query error for upvotes:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(`
		SELECT COUNT(*) as number_of_downvotes
		FROM entry_interactions
		WHERE entry_id = $1 AND interaction_type = 'downvote'
	`, entryID).Scan(&downvotes)

	if err != nil {
		fmt.Println("DB query error for downvotes:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(map[string]int{"upvotes": upvotes, "downvotes": downvotes})

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Upvote interaction processed successfully")

}
