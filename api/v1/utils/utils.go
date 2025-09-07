package utils

import (
	"fmt"
	"net/http"
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
