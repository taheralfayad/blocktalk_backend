package utils

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
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

func GenerateRefreshToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(30 * 24 * time.Hour).Unix(), // 30 days expiration
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JwtSecret)
}
