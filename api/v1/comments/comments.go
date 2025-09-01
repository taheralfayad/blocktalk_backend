package v1

import (
	"backend/api/v1/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type CommentRequest struct {
	EntryID  int    `json:"entry_id"`
	ParentID int    `json:"parent_id,omitempty"`
	Context  string `json:"context"`
	Type     string `json:"classification"`
}

type Comment struct {
	ID           int    `json:"id"`
	UserID       int    `json:"user_id"`
	EntryID      int    `json:"entry_id"`
	ParentID     *int   `json:"parent_id,omitempty"`
	Context      string `json:"context"`
	Type         string `json:"type"`
	Username     string `json:"username,omitempty"`
	NumOfReplies int    `json:"num_of_replies,omitempty"`
}

func AddComment(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	token, err := jwt.Parse(r.Header.Get("Authorization"), func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})

	var username string

	if err != nil || !token.Valid {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	} else {
		claims := token.Claims.(jwt.MapClaims)
		username = claims["username"].(string)
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
		err := rows.Scan(&comment.ID, &comment.UserID, &comment.Username, &comment.EntryID, &comment.ParentID, &comment.Context, &comment.Type)

		if err != nil {
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

		comment.NumOfReplies = commentCount

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
		err := rows.Scan(&reply.ID, &reply.UserID, &reply.Username, &reply.EntryID, &reply.ParentID, &reply.Context, &reply.Type)

		if err != nil {
			http.Error(w, "Failed to scan reply", http.StatusInternalServerError)
			return
		}

		err = db.QueryRow(`
			SELECT COUNT(*)
			FROM conversation c
			WHERE c.parent_id = $1
		`, reply.ID).Scan(&replyCount)

		if err != nil {
			reply.NumOfReplies = 0
		}

		reply.NumOfReplies = replyCount

		fmt.Println("Reply:", reply)
		fmt.Println("Reply Count:", replyCount)

		replies = append(replies, reply)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(replies)
}
