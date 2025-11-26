package entry

import (
	"backend/api/v1/structs"
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
	type CreateEntryRequest struct {
		Title       string        `json:"title"`
		Location    string        `json:"location"`
		Latitude    float64       `json:"latitude"`
		Longitude   float64       `json:"longitude"`
		Tags        []structs.Tag `json:"tags"`
		Description string        `json:"description"`
	}

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

	var username string

	username, err = utils.VerifyTokenAndReturnUsername(r)

	if err != nil {
		fmt.Println("Invalid token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

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

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("failed to start transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	var entryRevisionId int

	err = tx.QueryRow(`
		WITH entry_insert AS (
			INSERT INTO entry (address, creator_id, location) 
			VALUES ($2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326))
			RETURNING id
		),
		revision_number AS (
			SELECT COUNT(*) + 1 AS revision_number FROM entry_revision WHERE entry_id = (SELECT id FROM entry_insert)
		)
		INSERT INTO entry_revision (entry_id, title, content, revision_number, creator_id)
		SELECT (SELECT id FROM entry_insert), $1, $6, (SELECT revision_number FROM revision_number), $3
		RETURNING id
		`, payload.Title, payload.Location, userID, payload.Longitude, payload.Latitude, payload.Description).Scan(&entryRevisionId)

	if err != nil {
		fmt.Println("DB insert error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	utils.InsertTagAndEntryRevisionAssociation(tx, entryRevisionId, payload.Tags)
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
		SELECT entry.id,
			   address, 
			   er.content, 
			   views, 
			   date_created, 
			   username, 
			   first_name, 
			   last_name,
			   ST_X(location::geometry) AS longitude,
			   ST_Y(location::geometry) AS latitude
		FROM entry
		JOIN users ON entry.creator_id = users.id
		JOIN (
			SELECT DISTINCT ON (entry_id) entry_id, title, content, revision_number
			FROM entry_revision
			ORDER BY entry_id, revision_number DESC
		) er ON entry.id = er.entry_id
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

	var entries []structs.Entry

	for rows.Next() {
		var entry structs.Entry
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
	var cities []structs.City

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

	var results []structs.City

	cityByName := make(map[string]structs.City)
	for _, city := range cities {
			cityByName[city.Name] = city
	}

	for _, match := range matches {
			if city, ok := cityByName[match.Target]; ok {
					results = append(results, city)
			}
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

	var cities []structs.City

	err = json.Unmarshal(byteValue, &cities)

	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	var city structs.City

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
		SELECT e.id AS id,
			e.address,
			er.title,
			er.content,
			e.views,
			e.date_created,
			u.username,
			u.first_name,
			u.last_name,
			ST_X(e.location::geometry) AS longitude,
			ST_Y(e.location::geometry) AS latitude
		FROM entry e
		JOIN users u ON e.creator_id = u.id
		JOIN (
			SELECT DISTINCT ON (entry_id) entry_id, title, content, revision_number, date_created
			FROM entry_revision
			ORDER BY entry_id, revision_number DESC
		) er ON e.id = er.entry_id
		WHERE ST_DWithin(
			e.location,
			ST_MakePoint($1, $2)::geography,
			$3 * 1609.34
		)
		ORDER BY e.date_created DESC;
	`, city.Longitude, city.Latitude, distance)

	if err != nil {
		fmt.Println("Query error:", err)
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var entries []structs.Entry

	for rows.Next() {
		var numberOfComments int

		var entry structs.Entry
		err := rows.Scan(&entry.ID,
			&entry.Address,
			&entry.Title,
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

	token, err := jwt.Parse(r.Header.Get("Authorization"), func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})

	var username string

	if err != nil || !token.Valid {
		fmt.Println("No user is logged in")
	} else {
		claims := token.Claims.(jwt.MapClaims)
		username = claims["username"].(string)
	}

	var entry structs.Entry
	var entryRevisionId int

	err = db.QueryRow(`
		SELECT e.id, e.address, er.content, e.views, date_created,
			username, first_name, last_name, er.title, er.id,
			ST_X(location::geometry) AS longitude,
			ST_Y(location::geometry) AS latitude
		FROM entry e
		JOIN users ON e.creator_id = users.id
		JOIN (
			SELECT DISTINCT ON (entry_id) id, entry_id, title, content, revision_number
			FROM entry_revision
			ORDER BY entry_id, revision_number DESC
		) er ON e.id = er.entry_id
		WHERE e.id = $1
	`, entryID).Scan(&entry.ID,
		&entry.Address,
		&entry.Content,
		&entry.Views,
		&entry.DateCreated,
		&entry.Username,
		&entry.FirstName,
		&entry.LastName,
		&entry.Title,
		&entryRevisionId,
		&entry.Longitude,
		&entry.Latitude,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		fmt.Println("DB query error:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	rows, err := db.Query(`
		SELECT tags.name, tags.classification
		FROM tags
		JOIN tags_entry_revision ON tags.id = tags_entry_revision.tag_id
		WHERE tags_entry_revision.entry_revision_id = $1
	`, entryRevisionId)

	if err != nil {
		fmt.Println("DB query error for tags:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	var tags []structs.Tag

	for rows.Next() {
		var tagName string
		var classification string

		err := rows.Scan(&tagName, &classification)

		tag := structs.Tag{
			Name:           tagName,
			Classification: classification,
		}

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

	err = db.QueryRow(`
		SELECT interaction_type
		FROM entry_interactions
		WHERE user_id = (SELECT id FROM users WHERE username = $1) AND entry_id
		= $2
	`, username, entry.ID).Scan(&entry.UserInteraction)

	if err != nil && err != sql.ErrNoRows {
		fmt.Println("DB query error for user interaction:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	entry.Upvotes = upvotes
	entry.Downvotes = downvotes

	fmt.Println(entry.UserInteraction)

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

	var updatedInteractionType string
	err = db.QueryRow(`
			SELECT interaction_type
			FROM entry_interactions
			WHERE user_id = $1 AND entry_id = $2
		`, userID, entryID).Scan(&updatedInteractionType)
	if err == sql.ErrNoRows {
		updatedInteractionType = ""
	} else if err != nil {
		fmt.Println("DB query error for updated interaction type:", err)
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

	type VoteResponse struct {
		Upvotes         int    `json:"upvotes"`
		Downvotes       int    `json:"downvotes"`
		UserInteraction string `json:"user_interaction"`
	}

	response := VoteResponse{
		Upvotes:         upvotes,
		Downvotes:       downvotes,
		UserInteraction: updatedInteractionType,
	}

	err = json.NewEncoder(w).Encode(response)

	if err != nil {
		fmt.Println("Error encoding response:", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	fmt.Println("Upvote interaction processed successfully")

}

func EditEntry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	type EditEntryRequest struct {
		NewTitle   string        `json:"newTitle"`
		NewContent string        `json:"newContent"`
		NewTags    []structs.Tag `json:"newTags"`
		EntryID    int           `json:"entryId"`
	}

	var req EditEntryRequest
	var username string
	var userID int
	var entryRevisionId int

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		fmt.Println("Error decoding response", err)
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
	}

	username, err = utils.VerifyTokenAndReturnUsername(r)

	if err != nil {
		fmt.Println("Invalid token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1
	`, username).Scan(&userID)

	if err != nil {
		fmt.Println("Failed to retrieve user by username:", username, "err:", err)
		http.Error(w, "Failed to retrieve user by username", http.StatusNotFound)
	}

	tx, err := db.Begin()
	if err != nil {
		fmt.Println("failed to start transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = tx.QueryRow(`
		WITH revision_number AS (
			SELECT COUNT(*) + 1 AS revision_number FROM entry_revision WHERE entry_id = $1
		)
		INSERT INTO entry_revision (entry_id, title, content, revision_number, creator_id)
		SELECT $1, $2, $3, revision_number, $4 FROM revision_number
		RETURNING id
	`, req.EntryID, req.NewTitle, req.NewContent, userID).Scan(&entryRevisionId)

	if err != nil {
		fmt.Println("failed to insert entry_revision: %w", err)
		http.Error(w, "Failed to upload revision", http.StatusInternalServerError)
	}

	err = utils.InsertTagAndEntryRevisionAssociation(tx, entryRevisionId, req.NewTags)

	if err != nil {
		fmt.Println("failed to insert tags in entry_revision: %w", err)
		http.Error(w, "failed to upload revision", http.StatusInternalServerError)
	}
}
