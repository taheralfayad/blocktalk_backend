package users

import (
	"database/sql"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/lib/pq"

	data "backend/api/v1/data"
	utils "backend/api/v1/utils"
	messages "backend/api/v1/messages"
)


func CreateUser(c *gin.Context, db *sql.DB) {
	var payload data.CreateUserRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		messages.InternalServerError(c, err)
		return
	}

	hashedPassword, err := utils.HashPassword(payload.Password)

	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (username, first_name, last_name, password, email, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, payload.Username, payload.FirstName, payload.LastName, hashedPassword, payload.Email, payload.PhoneNumber)

	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
			messages.StatusConflict(c, err)
		}
		
		messages.InternalServerError(c, err)
	}

	accessToken, accessTokenExpDate, err := utils.GenerateAccessToken(payload.Username)

	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	refreshToken, refreshTokenExpDate, err := utils.GenerateRefreshToken(payload.Username)

	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	utils.SetCookies(
		c,
		accessToken,
		refreshToken,
		int(accessTokenExpDate),
		int(refreshTokenExpDate),
	)

	messages.StatusOk(c, "User has been successfully created.")
}

func LoginUser(c *gin.Context ,db *sql.DB) {
	var payload data.LoginRequest

	if err := c.ShouldBindJSON(&payload); err != nil {
		messages.InternalServerError(c, err)
		return
	}

	var hashedPassword string

	err := db.QueryRow(`
		SELECT password FROM users WHERE username = $1
	`, payload.Username).Scan(&hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			messages.StatusUnauthorized(c, err)
			return
		}

		messages.InternalServerError(c, err)
		return
	}

	if !utils.VerifyPassword(hashedPassword, payload.Password) {
		messages.StatusUnauthorized(c, errors.New("Wrong password"))
		return
	}

	accessToken, accessTokenExpDate, err := utils.GenerateAccessToken(payload.Username)

	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	refreshToken, refreshTokenExpDate, err := utils.GenerateRefreshToken(payload.Username)

	if err != nil {
		messages.InternalServerError(c, err)
	}

	utils.SetCookies(
		c,
		accessToken,
		refreshToken,
		int(accessTokenExpDate),
		int(refreshTokenExpDate),
	)

	messages.StatusOk(c, "User successfully logged in")
}

func RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})
	if err != nil {
		messages.StatusUnauthorized(c, err)
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		messages.StatusUnauthorized(c, errors.New("invalid token claims"))
		return
	}

	username, ok := claims["username"].(string)
	if !ok {
		messages.StatusUnauthorized(c, errors.New("invalid username in token"))
		return
	}

	newAccessToken, expirationDate, err := utils.GenerateAccessToken(username)
	if err != nil {
		messages.InternalServerError(c, err)
		return
	}

	utils.SetAuthCookie(
		c,
		"access_token",
		newAccessToken,
		int(expirationDate),
	)

	messages.StatusOk(c, "Access token has been refreshed!")
}
