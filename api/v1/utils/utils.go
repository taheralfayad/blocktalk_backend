package utils

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	structs "backend/api/v1/structs"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var JwtSecret = []byte(os.Getenv("JWT_SECRET"))

func GenerateAccessToken(username string) (string, int64, error) {
	expirationTime := time.Now().Add(8 * 60 * time.Minute).Unix()
	fmt.Println("Expiration Time:", expirationTime)
	claims := jwt.MapClaims{
		"username": username,
		"exp":      expirationTime,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(JwtSecret)
	return signedToken, expirationTime, err
}

func GenerateRefreshToken(username string) (string, int64, error) {
	expirationTime := time.Now().Add(30 * 24 * time.Hour).Unix()
	claims := jwt.MapClaims{
		"username": username,
		"exp":      expirationTime,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(JwtSecret)
	return signedToken, expirationTime, err
}

func ParseTokenAndReturnUsername(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JwtSecret, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}
	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("unable to parse claims")
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", fmt.Errorf("username claim missing or invalid")
	}

	return username, nil
}

func VerifyTokenAndReturnUsername(r *http.Request) (string, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return "", fmt.Errorf("authorization header missing")
	}
	return ParseTokenAndReturnUsername(tokenString)
}

func InsertTagAndEntryRevisionAssociation(tx *sql.Tx, entryRevisionId int, tags []structs.Tag) error {
	for _, tag := range tags {
		var tagID int
		err := tx.QueryRow(`
			INSERT INTO tags (name, classification)
			VALUES ($1, $2)
			ON CONFLICT (name) DO UPDATE
			SET classification = EXCLUDED.classification
			RETURNING id
		`, tag.Name, tag.Classification).Scan(&tagID)
		if err != nil {
			fmt.Println("failed to insert or fetch tag: %w", err)
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO tags_entry_revision (entry_revision_id, tag_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, entryRevisionId, tagID)
		if err != nil {
			fmt.Println("failed to insert tag association: %w", err)
		}
	}

	return nil
}

func RetrieveNumberOfUpvotesAndDownvotesForTable(
	tableName string,
	rowId int,
	db *sql.DB,
) (int, int) {
	var upvotes, downvotes int

	if tableName != "conversation" && tableName != "entry" {
		return 0, 0
	}

	queryUp := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s_interactions
		WHERE %s_id = $1 AND interaction_type = 'upvote'
	`, tableName, tableName)

	queryDown := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM %s_interactions
		WHERE %s_id = $1 AND interaction_type = 'downvote'
	`, tableName, tableName)

	if err := db.QueryRow(queryUp, rowId).Scan(&upvotes); err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error retrieving upvotes for %s %d: %v\n", tableName, rowId, err)
	}

	if err := db.QueryRow(queryDown, rowId).Scan(&downvotes); err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error retrieving downvotes for %s %d: %v\n", tableName, rowId, err)
	}

	return upvotes, downvotes
}

func RetrieveCurrentInteractionTypeForTable(
	tableName string,
	userId int,
	rowId int,
	db *sql.DB,
) string {
	var currentInteractionType string

	if tableName != "conversation" && tableName != "entry" {
		return currentInteractionType
	}

	currentInteractionTypeQuery := fmt.Sprintf(`
		SELECT interaction_type
		FROM %s_interactions
		WHERE user_id = $1 AND %s_id = $2
	`, tableName, tableName)

	if err := db.QueryRow(currentInteractionTypeQuery, userId, rowId).Scan(&currentInteractionType); err != nil && err != sql.ErrNoRows {
		fmt.Printf("Error retrieving current interaction type for %s %d: %v\n", tableName, rowId, err)
	}

	return currentInteractionType
}

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func SetCookies(
	c *gin.Context,
	accessToken string,
	refreshToken string,
	accessTokenExpDate int,
	refreshTokenExpDate int,
) {

	SetAuthCookie(
		c,
		"refresh_token",
		refreshToken,
		refreshTokenExpDate,
	)

	SetAuthCookie(
		c,
		"accessToken",
		accessToken,
		accessTokenExpDate,
	)

}

func SetAuthCookie(
	c *gin.Context,
	tokenType string,
	token string,
	tokenExpDate int,
) {

	c.SetCookie(
		tokenType,
		token,
		tokenExpDate,
		"/",
		"",
		true,
		true,
	)

}
