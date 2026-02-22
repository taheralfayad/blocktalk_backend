package structs

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
