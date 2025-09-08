package structs

type Tag struct {
	Name           string `json:"name"`
	Classification string `json:"classification"`
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
	Tags             []Tag     `json:"tags,omitempty"`
	Comments         []Comment `json:"comments,omitempty"`
	UserInteraction  string    `json:"user_interaction,omitempty"`
}
