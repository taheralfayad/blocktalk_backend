package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"

	comments "backend/api/v1/comments"
	entry "backend/api/v1/entry"
	users "backend/api/v1/users"
	utils "backend/api/v1/utils"
	messages "backend/api/v1/messages"
)

var db *sql.DB

func validateToken() gin.HandlerFunc {
	return func(c *gin.Context) {

		cookie, err := c.Cookie("access_token")
		if err != nil {
			messages.StatusUnauthorized(c, err)
			return
		}

		token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
			return utils.JwtSecret, nil
		})

		if err != nil || !token.Valid {
			log.Printf("Invalid token: %v", err)
			messages.StatusUnauthorized(c, err)
			return
		}

		c.Next()
	}
}

func handleRequests() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: true,
	}))

	usersGroup := r.Group("/users")

	{
		usersGroup.POST("/create-user", func(c *gin.Context) {
			users.CreateUser(c, db)
		})

		usersGroup.POST("/login", func(c *gin.Context) {
			users.LoginUser(c, db)
		})

		usersGroup.POST("/refresh-token", users.RefreshToken)
	}


	myRouter.HandleFunc("/retrieve-entries-within-visible-bounds", func(w http.ResponseWriter, r *http.Request) {
		entry.RetrieveEntriesWithinVisibleBounds(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/create-entry", func(w http.ResponseWriter, r *http.Request) {
		entry.CreateEntry(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/autocomplete-address", entry.AutocompleteAddress).Methods("GET")
	myRouter.HandleFunc("/retrieve-city", entry.RetrieveCity).Methods("GET")
	myRouter.HandleFunc("/feed", func(w http.ResponseWriter, r *http.Request) {
		entry.RetrieveFeed(w, r, db)
	}).Methods("GET")
	myRouter.HandleFunc("/retrieve-entry", func(w http.ResponseWriter, r *http.Request) {
		entry.RetrieveEntry(w, r, db)
	}).Methods("GET")
	myRouter.HandleFunc("/vote-entry", func(w http.ResponseWriter, r *http.Request) {
		entry.VoteEntry(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/edit-entry", func(w http.ResponseWriter, r *http.Request) {
		entry.EditEntry(w, r, db)
	})

	// =========================

	// Comment API routes
	myRouter.HandleFunc("/add-comment", func(w http.ResponseWriter, r *http.Request) {
		comments.AddComment(w, r, db)
	}).Methods("POST")
	myRouter.HandleFunc("/retrieve-comments", func(w http.ResponseWriter, r *http.Request) {
		comments.GetEntryComments(w, r, db)
	}).Methods("GET")
	myRouter.HandleFunc("/retrieve-comment-replies", func(w http.ResponseWriter, r *http.Request) {
		comments.GetCommentReplies(w, r, db)
	}).Methods("GET")
	myRouter.HandleFunc("/vote-on-comment", func(w http.ResponseWriter, r *http.Request) {
		comments.VoteOnComment(w, r, db)
	})

	headersOk := handlers.AllowedHeaders([]string{
		"X-Requested-With",
		"Content-Type",
		"Authorization",
		"Accept",
		"Origin",
	})
	originsOk := handlers.AllowedOrigins([]string{"http://localhost:5173"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS(originsOk, headersOk, methodsOk)(myRouter)))

}

func main() {
	var err error
	db, err = initDB()

	for i := 0; i < 10 && err != nil; i++ {
		log.Printf("Failed to connect to DB, attempt %d: %v", i+1, err)
		time.Sleep(2 * time.Second)
		err = db.Ping()
	}

	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer db.Close()

	handleRequests()
}
