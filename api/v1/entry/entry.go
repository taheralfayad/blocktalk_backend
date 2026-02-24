package entry

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"

	"backend/api/v1/utils"

	"github.com/gin-gonic/gin"
	"github.com/lithammer/fuzzysearch/fuzzy"

	data "backend/api/v1/data"
	messages "backend/api/v1/messages"
)

func AutocompleteAddress(c *gin.Context, db *sql.DB) {
	query := c.DefaultQuery("query", "")

	if query == "" {
		messages.StatusBadRequest(c, errors.New("No query provided"))
		return
	}

	tomTomApiKey := os.Getenv("TOM_TOM_API_KEY")

	url := os.Getenv("TOM_TOM_BASE_URL") + "/search/2/search/" + url.QueryEscape(query) + ".json?key=" + tomTomApiKey + "&typeahead=true" + "&limit=3" + "&countrySet=US"

	resp, err := http.Get(url)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		messages.InternalServerError(c, errors.New(string(bodyBytes)))
		return
	}

	defer resp.Body.Close()

	responseData, err := io.ReadAll(resp.Body)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var tomTomResponse data.TomTomResponse

	err = json.Unmarshal(responseData, &tomTomResponse)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var suggestions []data.AddressSuggestion

	for _, result := range tomTomResponse.Results {
		var suggestion data.AddressSuggestion

		suggestion.Address = result.Address.FreeformAddress
		suggestion.Lat = result.Position.Lat
		suggestion.Lon = result.Position.Lon

		suggestions = append(suggestions, suggestion)
	}

	c.JSON(http.StatusOK, suggestions)
}

func CreateEntry(c *gin.Context, db *sql.DB) {
	var payload data.CreateEntryRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		messages.InternalServerError(c, err)
		return
	}

	query := `
		SELECT EXISTS (
			SELECT 1
			FROM entry
			WHERE ST_DWithin(location, geography(ST_MakePoint($1, $2)), 50)
		);
	`

	var entryAlreadyExists bool
	err := db.QueryRow(query, payload.Longitude, payload.Latitude).Scan(&entryAlreadyExists)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	if entryAlreadyExists {
		messages.StatusConflict(c, err)
		return
	}

	var username string

	cookie, err := c.Cookie("access_token")
	if err != nil {
		messages.StatusUnauthorized(c, err)
	}

	username, err = utils.ParseTokenAndReturnUsername(cookie)
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	var userID int
	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1
	`, username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			messages.StatusUnauthorized(c, err)
			return
		}

		messages.InternalServerError(c, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		messages.InternalServerError(c, err)
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
		messages.InternalServerError(c, err)
		return
	}

	utils.InsertTagAndEntryRevisionAssociation(tx, entryRevisionId, payload.Tags)
	messages.StatusCreated(c, "Entry created successfully!")
}

func RetrieveEntriesWithinVisibleBounds(c *gin.Context, db *sql.DB) {
	var bounds data.Bounds

	if err := c.ShouldBindJSON(&bounds); err != nil {
		messages.InternalServerError(c, err)
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
				 er.id as revision_id,
				 er.title,
			   ST_X(location::geometry) AS longitude,
			   ST_Y(location::geometry) AS latitude
		FROM entry
		JOIN users ON entry.creator_id = users.id
		JOIN (
			SELECT DISTINCT ON (entry_id) id, entry_id, title, content, revision_number
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
		messages.InternalServerError(c, err)
		return
	}
	defer rows.Close()

	var entries []data.Entry

	for rows.Next() {
		var entry data.Entry
		var entryRevisionId int

		err := rows.Scan(
			&entry.ID,
			&entry.Address,
			&entry.Content,
			&entry.Views,
			&entry.DateCreated,
			&entry.Username,
			&entry.FirstName,
			&entry.LastName,
			&entryRevisionId,
			&entry.Title,
			&entry.Longitude,
			&entry.Latitude,
		)
		if err != nil {
			messages.InternalServerError(c, err)
			return
		}

		var tags []data.Tag

		tagRows, err := db.Query(`
					SELECT tags.name, tags.classification
					FROM tags
					JOIN tags_entry_revision ON tags.id = tags_entry_revision.tag_id
					WHERE tags_entry_revision.entry_revision_id = $1
			`, entryRevisionId)
		if err != nil {
			messages.InternalServerError(c, err)
			return
		}
		defer tagRows.Close()

		for tagRows.Next() {
			var tag data.Tag

			err := tagRows.Scan(
				&tag.Name,
				&tag.Classification,
			)
			if err != nil {
				messages.InternalServerError(c, err)
			}

			tags = append(tags, tag)
		}

		entry.Tags = tags

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		messages.StatusNoContent(c, err)
		return
	}

	c.JSON(http.StatusCreated, entries)
}

func RetrieveCity(c *gin.Context, db *sql.DB) {
	query := c.DefaultQuery("city", "")

	if query == "" {
		messages.StatusBadRequest(c, errors.New("City not found"))
		return
	}

	var cities []data.City

	jsonFile, err := os.Open("/app/static/us_cities.json")
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &cities)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var cityNames []string

	for _, city := range cities {
		cityNames = append(cityNames, city.Name)
	}

	matches := fuzzy.RankFind(query, cityNames)

	if len(matches) > 3 {
		matches = matches[:3]
	}

	var results []data.City

	cityByName := make(map[string]data.City)
	for _, city := range cities {
		cityByName[city.Name] = city
	}

	for _, match := range matches {
		if city, ok := cityByName[match.Target]; ok {
			results = append(results, city)
		}
	}

	if len(results) == 0 {
		messages.StatusNoContent(c, errors.New("No results found"))
		return
	}

	c.JSON(http.StatusOK, results)
}

func RetrieveFeed(c *gin.Context, db *sql.DB) {
	var query data.FeedQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		messages.StatusBadRequest(c, err)
		return
	}

	location := query.Location
	distance := query.Distance

	jsonFile, err := os.Open("/app/static/us_cities.json")
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var cities []data.City

	err = json.Unmarshal(byteValue, &cities)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var city data.City

	for _, c := range cities {
		if c.Name == location {
			city = c
			break
		}
	}

	if city.Name == "" {
		messages.StatusNoContent(c, errors.New("Unable to find city matching location."))
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
		messages.InternalServerError(c, err)
		return
	}

	defer rows.Close()

	var entries []data.Entry

	for rows.Next() {

		var entry data.Entry
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
			messages.InternalServerError(c, err)
			return
		}

		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		messages.InternalServerError(c, err)
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
			messages.InternalServerError(c, err)
			return
		}

		err = db.QueryRow(`
			SELECT COUNT(*) as number_of_downvotes
			FROM entry_interactions
			WHERE entry_id = $1 AND interaction_type = 'downvote'
		`, entry.ID).Scan(&downvotes)
		if err != nil {
			messages.InternalServerError(c, err)
			return
		}

		entries[i].Upvotes = upvotes
		entries[i].Downvotes = downvotes
	}

	if len(entries) == 0 {
		messages.StatusNoContent(c, errors.New("No entries found"))
		return
	}

	c.JSON(http.StatusOK, entries)
}

func VoteEntry(c *gin.Context, db *sql.DB) {
	cookie, err := c.Cookie("access_token")
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	username, err := utils.ParseTokenAndReturnUsername(cookie)
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	userID := 0

	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1
	`, username).Scan(&userID)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var req data.VoteRequest

	if err := c.ShouldBindJSON(req); err != nil {
		messages.StatusBadRequest(c, err)
		return
	}

	var currentInteraction string
	entryID := req.EntryID

	err = db.QueryRow(`
		SELECT interaction_type
		FROM entry_interactions
		WHERE user_id = $1 AND entry_id = $2
	`, userID, entryID).Scan(&currentInteraction)

	if err != nil && err != sql.ErrNoRows {
		messages.InternalServerError(c, err)
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
		messages.InternalServerError(c, err)
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
		messages.InternalServerError(c, err)
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
		messages.InternalServerError(c, err)
		return
	}

	err = db.QueryRow(`
		SELECT COUNT(*) as number_of_downvotes
		FROM entry_interactions
		WHERE entry_id = $1 AND interaction_type = 'downvote'
	`, entryID).Scan(&downvotes)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	response := data.VoteResponse{
		Upvotes:         upvotes,
		Downvotes:       downvotes,
		UserInteraction: updatedInteractionType,
	}

	c.JSON(http.StatusOK, response)
}

func EditEntry(c *gin.Context, db *sql.DB) {
	var req data.EditEntryRequest
	var username string
	var userID int
	var entryRevisionId int

	if err := c.ShouldBindJSON(req); err != nil {
		messages.StatusBadRequest(c, err)
	}

	cookie, err := c.Cookie("access_token")

	username, err = utils.ParseTokenAndReturnUsername(cookie)
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1
	`, username).Scan(&userID)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		messages.InternalServerError(c, err)
		return
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
		messages.InternalServerError(c, err)
		return
	}

	err = utils.InsertTagAndEntryRevisionAssociation(tx, entryRevisionId, req.NewTags)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	messages.StatusOk(c, "Entry edited successfully!")
}
