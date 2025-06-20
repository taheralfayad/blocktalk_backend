package main

import (
	"database/sql"
	"os"
)

func initDB() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL") // Ex: "postgres://user:pass@localhost:5432/dbname?sslmode=disable"
	return sql.Open("postgres", connStr)
}
