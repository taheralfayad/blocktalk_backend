package utils

import (
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseTokenAndReturnUsername(t *testing.T) {
	claims := jwt.MapClaims{"username": "alice"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(JwtSecret)

	username, err := ParseTokenAndReturnUsername(tokenString)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if username != "alice" {
		t.Fatalf("expected username alice, got %v", username)
	}
}

func TestParseTokenWithInvalidToken(t *testing.T) {
	claims := jwt.MapClaims{"username": "alice"}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, _ := token.SignedString([]byte("wrong-secret"))

	username, err := ParseTokenAndReturnUsername(tokenString)

	if err == nil {
		t.Fatalf("expected error, got no error")
	}

	if !strings.Contains(err.Error(), "failed to parse token") {
		t.Fatalf("expected error to contain 'failed to parse token', got %v", err)
	}

	if username != "" {
		t.Fatalf("expected empty username, got %s", username)
	}
}
