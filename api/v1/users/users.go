package users

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lib/pq"

	utils "backend/api/v1/utils"
)

type CreateUserRequest struct {
	Username    string `json:"username"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Password    string `json:"password"`
	Email       string `json:"email"`
	PhoneNumber string `json:"phone_number"`
}

func CreateUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var payload CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Fatal(w, "ParseForm() error", http.StatusBadRequest)
	}

	if payload.Username == "" || payload.Password == "" || payload.Email == "" || payload.PhoneNumber == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	hashedPassword, err := hashPassword(payload.Password)

	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec(`
		INSERT INTO users (username, first_name, last_name, password, email, phone_number)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, payload.Username, payload.FirstName, payload.LastName, hashedPassword, payload.Email, payload.PhoneNumber)

	if err != nil {
		if err, ok := err.(*pq.Error); ok && err.Code == "23505" {
			if err.Constraint == "users_username_key" {
				http.Error(w, "Username already exists", http.StatusConflict)
				return
			}
			if err.Constraint == "users_email_key" {
				http.Error(w, "Email already exists", http.StatusConflict)
				return
			}
			if err.Constraint == "users_phone_number_key" {
				http.Error(w, "Phone number already exists", http.StatusConflict)
				return
			}
		}

		log.Printf("DB insert error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created successfully"))
}

func LoginUser(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if payload.Username == "" || payload.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	var hashedPassword string

	err = db.QueryRow(`
		SELECT password FROM users WHERE username = $1
	`, payload.Username).Scan(&hashedPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "User not found", http.StatusUnauthorized)
		} else {
			log.Printf("DB query error: %v", err)
			http.Error(w, "Database error", http.StatusInternalServerError)
		}
		return
	}

	if !verifyPassword(hashedPassword, payload.Password) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	accessToken, expirationDate, err := utils.GenerateAccessToken(payload.Username)
	if err != nil {
		http.Error(w, "Could not generate access token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := utils.GenerateRefreshToken(payload.Username)
	if err != nil {
		http.Error(w, "Could not generate refresh token", http.StatusInternalServerError)
		return
	}

	tokens := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_at":    strconv.FormatInt(expirationDate, 10),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

func RefreshToken(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RefreshToken string `json:"refresh_token"`
	}

	err := json.NewDecoder(r.Body).Decode(&payload)

	if err != nil || payload.RefreshToken == "" {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	token, err := jwt.Parse(payload.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		return utils.JwtSecret, nil
	})

	if err != nil || !token.Valid {
		http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	claims := token.Claims.(jwt.MapClaims)
	username := claims["username"].(string)

	newAccessToken, expirationDate, err := utils.GenerateAccessToken(username)
	if err != nil {
		http.Error(w, "Failed to generate access token", http.StatusInternalServerError)
		return
	}

	resp := map[string]string{
		"access_token": newAccessToken,
		"expires_at":   strconv.FormatInt(expirationDate, 10),
	}
	json.NewEncoder(w).Encode(resp)
}

func hashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func verifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
