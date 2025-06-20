package v1

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"golang.org/x/crypto/bcrypt"
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
		log.Printf("DB insert error: %v", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User created successfully"))
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
