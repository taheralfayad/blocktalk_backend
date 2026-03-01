package v1

import (
	utils "backend/api/v1/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type CommentRequest struct {
	EntryID  int    `json:"entry_id"`
	ParentID *int   `json:"parent_id,omitempty"`
	Context  string `json:"context"`
	Type     string `json:"classification"`
}

type Comment struct {
	ID                        int    `json:"id"`
	UserID                    int    `json:"user_id"`
	EntryID                   int    `json:"entry_id"`
	ParentID                  *int   `json:"parent_id,omitempty"`
	Context                   string `json:"context"`
	Type                      string `json:"type"`
	Username                  string `json:"username,omitempty"`
	NumOfReplies              int    `json:"num_of_replies"`
	NumberOfUpvotes           int    `json:"num_of_upvotes"`
	NumberOfDownvotes         int    `json:"num_of_downvotes"`
	CurrentCommentInteraction string `json:"current_comment_interaction,omitempty"`
}

func AddComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	username, err := utils.VerifyTokenAndReturnUsername(r)

	if err != nil {
		fmt.Println("Invalid Token, err: ", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var userID int

	err = db.QueryRow("SELECT id FROM users WHERE username=$1", username).Scan(&userID)

	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var req CommentRequest
	var comment Comment

	err = json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.EntryID == -1 || req.Context == "" || req.Type == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	err = db.QueryRow(
		`INSERT INTO conversation (user_id, entry_id, parent_id, context, type)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, entry_id, parent_id, context, type`,
		userID, req.EntryID, req.ParentID, req.Context, req.Type,
	).Scan(&comment.ID, &comment.UserID, &comment.EntryID, &comment.ParentID, &comment.Context, &comment.Type)

	if err != nil {
		http.Error(w, "Failed to add comment", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	comment.Username = username

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

func GetEntryComments(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	entryID := r.URL.Query().Get("entry_id")

	if entryID == "" {
		http.Error(w, "Missing entry_id parameter", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT 
			c.id, 
			c.user_id, 
			(SELECT u.username FROM users u WHERE u.id = c.user_id) AS username,
			c.entry_id, 
			c.parent_id, 
			c.context, 
			c.type
		FROM conversation c
		WHERE c.entry_id = $1 AND c.parent_id IS NULL
	`, entryID)

	if err != nil {
		print(err)
		http.Error(w, "Failed to retrieve comments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment

	var commentCount int

	for rows.Next() {
		var comment Comment
		var currentInteractionType string
		var upvotes, downvotes int

		err = rows.Scan(
			&comment.ID,
			&comment.UserID,
			&comment.Username,
			&comment.EntryID,
			&comment.ParentID,
			&comment.Context,
			&comment.Type,
		)

		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to scan comment", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM conversation c
			WHERE c.parent_id = $1
		`, comment.ID).Scan(&commentCount)

		if err != nil {
			comment.NumOfReplies = 0
		}

		err = db.QueryRow(`
			SELECT interaction_type
			FROM conversation_interactions
			WHERE user_id = $1 AND conversation_id = $2
		`, comment.UserID, comment.ID).Scan(&currentInteractionType)

		if err != nil && err != sql.ErrNoRows {
			fmt.Println(err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}

		upvotes, downvotes = utils.RetrieveNumberOfUpvotesAndDownvotesForTable("conversation", comment.ID, db)

		fmt.Println(upvotes, downvotes)

		comment.CurrentCommentInteraction = currentInteractionType

		comment.NumOfReplies = commentCount
		comment.NumberOfUpvotes = upvotes
		comment.NumberOfDownvotes = downvotes

		comments = append(comments, comment)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

func GetCommentReplies(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	commentID := r.URL.Query().Get("comment_id")
	entryID := r.URL.Query().Get("entry_id")

	if commentID == "" {
		http.Error(w, "Missing comment_id parameter", http.StatusBadRequest)
		return
	}

	if entryID == "" {
		http.Error(w, "Missing entry_id parameter", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(`
		SELECT 
			c.id, 
			c.user_id, 
			(SELECT u.username FROM users u WHERE u.id = c.user_id) AS username,
			c.entry_id, 
			c.parent_id,
			c.context, 
			c.type
		FROM conversation c
		WHERE c.parent_id = $1
	`, commentID)

	if err != nil {
		http.Error(w, "Failed to retrieve replies", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var replies []Comment
	var replyCount int

	for rows.Next() {
		var reply Comment
		var upvotes, downvotes int

		err := rows.Scan(&reply.ID, &reply.UserID, &reply.Username, &reply.EntryID, &reply.ParentID, &reply.Context, &reply.Type)

		if err != nil {
			http.Error(w, "Failed to scan reply", http.StatusInternalServerError)
			return
		}

		upvotes, downvotes = utils.RetrieveNumberOfUpvotesAndDownvotesForTable("conversation", reply.ID, db)

		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM conversation c
			WHERE c.parent_id = $1
		`, reply.ID).Scan(&replyCount)

		if err != nil {
			reply.NumOfReplies = 0
		}

		reply.NumOfReplies = replyCount
		reply.NumberOfUpvotes = upvotes
		reply.NumberOfDownvotes = downvotes
		reply.CurrentCommentInteraction = utils.RetrieveCurrentInteractionTypeForTable(
			"conversation",
			reply.UserID,
			reply.ID,
			db,
		)

		replies = append(replies, reply)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(replies)
}

func VoteOnComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	token, err := jwt.Parse(r.Header.Get("Authorization"), func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})

	if err != nil || !token.Valid {
		fmt.Println("Invalid token:", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}

	claims := token.Claims.(jwt.MapClaims)

	username := claims["username"].(string)

	var userID int

	err = db.QueryRow(`
		SELECT id FROM users WHERE username = $1	
	`, username).Scan(&userID)

	if err != nil {
		fmt.Println("DB query error:", err)
		http.Error(w, "internal Server Error", http.StatusInternalServerError)
		return
	}

	type InteractionRequest struct {
		CommentID       int    `json:"comment_id"`
		InteractionType string `json:"interaction_type"`
	}

	var req InteractionRequest

	err = json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		fmt.Println("Error decoding json", err)
		http.Error(w, "internal server error", http.StatusBadRequest)
		return
	}

	if req.CommentID == 0 || req.InteractionType == "" {
		http.Error(w, "Comment ID and interaction type are required", http.StatusBadRequest)
		return
	}

	var currentInteraction string

	err = db.QueryRow(`
		SELECT interaction_type
		FROM conversation_interactions
		WHERE user_id = $1 and conversation_id = $2
	`, userID, req.CommentID).Scan(&currentInteraction)

	if err != nil && err != sql.ErrNoRows {
		fmt.Println("DB query error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Printf("DEBUG currentInteraction='%s', req.InteractionType='%s'\n", currentInteraction, req.InteractionType)

	if currentInteraction == req.InteractionType {
		_, err = db.Exec(`
			DELETE FROM conversation_interactions
			WHERE user_id = $1 AND conversation_id = $2
		`, userID, req.CommentID)
	} else if currentInteraction != "" {
		fmt.Println("here?")
		_, err = db.Exec(`
			UPDATE conversation_interactions
			SET interaction_type = $3, created_at = CURRENT_TIMESTAMP
			WHERE user_id = $1 AND conversation_id = $2
		`, userID, req.CommentID, req.InteractionType)
	} else {
		_, err = db.Exec(`
			INSERT into conversation_interactions (conversation_id, user_id, interaction_type)
			VALUES ($1, $2, $3)
		`, req.CommentID, userID, req.InteractionType)
	}

	if err != nil {
		fmt.Println("DB exec error:", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	var updatedInteractionType string

	err = db.QueryRow(`
		SELECT interaction_type
		FROM conversation_interactions
		WHERE user_id = $1 and conversation_id = $2
	`, userID, req.CommentID).Scan(&updatedInteractionType)

	if err == sql.ErrNoRows {
		updatedInteractionType = ""
	} else if err != nil {
		fmt.Println("DB query error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	upvotes, downvotes := utils.RetrieveNumberOfUpvotesAndDownvotesForTable(
		"conversation",
		req.CommentID,
		db,
	)

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
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	fmt.Println("Vote interaction processed successfully")

}
