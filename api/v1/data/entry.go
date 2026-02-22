package data

import "backend/api/v1/structs"

type Bounds struct {
	North float64 `json:"north"`
	South float64 `json:"south"`
	East  float64 `json:"east"`
	West  float64 `json:"west"`
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

type Tag struct {
	Name           string `json:"name"`
	Classification string `json:"classification"`
}

type EditEntryRequest struct {
	NewTitle   string        `json:"newTitle"`
	NewContent string        `json:"newContent"`
	NewTags    []structs.Tag `json:"newTags"`
	EntryID    int           `json:"entryId"`
}

type VoteRequest struct {
	EntryID         string `json:"entry_id" binding:"required"`
	InteractionType string `json:"interaction_type" binding:"required"`
}

type VoteResponse struct {
	Upvotes         int    `json:"upvotes"`
	Downvotes       int    `json:"downvotes"`
	UserInteraction string `json:"user_interaction"`
}

type CreateEntryRequest struct {
	Title       string        `json:"title" binding:"required"`
	Location    string        `json:"location" binding:"required"`
	Latitude    float64       `json:"latitude" binding:"required"`
	Longitude   float64       `json:"longitude" binding:"required"`
	Tags        []structs.Tag `json:"tags" binding:"required"`
	Description string        `json:"description" binding:"required"`
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
	Tags             []Tag     `json:"tags,omitempty"`
	Comments         []Comment `json:"comments,omitempty"`
	UserInteraction  string    `json:"user_interaction,omitempty"`
}

type TomTomResponse struct {
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

type AddressSuggestion struct {
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
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

type FeedQuery struct {
	Location string `form:"location" binding:"required"`
	Distance string `form:"distance" binding:"required"`
}
