package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	entry "backend/api/v1/entry"
	messages "backend/api/v1/messages"
	users "backend/api/v1/users"
	utils "backend/api/v1/utils"
)

var db *sql.DB

func AuthMiddleware() gin.HandlerFunc {
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

	userRoutes := r.Group("/users")
	entryRoutes := r.Group("/entries")
	entryPrivilegedRoutes := r.Group("/entries")
	entryPrivilegedRoutes.Use(AuthMiddleware())

	userRoutes.POST("/create-user", func(c *gin.Context) {
		users.CreateUser(c, db)
	})

	userRoutes.POST("/login", func(c *gin.Context) {
		users.LoginUser(c, db)
	})

	userRoutes.POST("/refresh-token", users.RefreshToken)

	entryRoutes.POST("/retrieve-entries-within-visible-bounds", func(c *gin.Context) {
		entry.RetrieveEntriesWithinVisibleBounds(c, db)
	})

	entryRoutes.GET("/retrieve-city", func(c *gin.Context) {
		entry.RetrieveCity(c, db)
	})

	entryRoutes.GET("/feed", func(c *gin.Context) {
		entry.RetrieveFeed(c, db)
	})

	entryPrivilegedRoutes.POST("/create-entry", func(c *gin.Context) {
		entry.CreateEntry(c, db)
	})

	entryPrivilegedRoutes.GET("/autocomplete-address", func(c *gin.Context) {
		entry.AutocompleteAddress(c, db)
	})

	entryPrivilegedRoutes.POST("/vote-entry", func(c *gin.Context) {
		entry.VoteEntry(c, db)
	})

	entryPrivilegedRoutes.POST("/edit-entry", func(c *gin.Context) {
		entry.EditEntry(c, db)
	})

	r.Run()
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
