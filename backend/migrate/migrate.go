package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
)

func runMigrations() error {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://sql",
		"postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

func migrateAllDown() error {
	m, err := migrate.New(
		"file://sql",
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration down failed: %v", err)
	}

	log.Println("Migration down complete.")

	return nil
}

func migrateToVersion(version int) error {
	m, err := migrate.New(
		"file://sql",
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Migrate(uint(version)); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("migration to version %d failed: %v", version, err)
	}

	log.Printf("Migration to version %d complete.", version)

	return nil
}

func forceVersion(version int) error {
	m, err := migrate.New(
		"file://sql",
		os.Getenv("DATABASE_URL"),
	)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		log.Fatalf("force migration to version %d failed: %v", version, err)
	}

	log.Printf("Force migration to version %d complete.", version)

	return nil
}

func main() {
	funcs := map[string]func() error{
		"up":      runMigrations,
		"allDown": migrateAllDown,
		"toVersion": func() error {
			if len(os.Args) < 3 {
				return log.Output(2, "Please provide a version number to migrate to.")
			}
			version := os.Args[2]

			intVersion, err := strconv.Atoi(version)

			if err != nil {
				return log.Output(2, "Invalid version number provided.")
			}

			if intVersion < 0 {
				return log.Output(2, "Version number must be a non-negative integer.")
			}

			return migrateToVersion(intVersion)
		},
		"force": func() error {
			if len(os.Args) < 3 {
				return log.Output(2, "Please provide a version number to force.")
			}
			version := os.Args[2]

			intVersion, err := strconv.Atoi(version)

			if err != nil {
				return log.Output(2, "Invalid version number provided.")
			}

			if intVersion < 0 {
				return log.Output(2, "Version number must be a non-negative integer.")
			}

			return forceVersion(intVersion)
		},
	}

	if len(os.Args) < 2 {
		log.Fatal("Please provide a command: up or allDown")
	}

	command := os.Args[1]

	if migrateFunc, exists := funcs[command]; exists {
		if err := migrateFunc(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		} else {
			log.Println("Migration completed successfully.")
		}
	} else {
		log.Fatalf("Unknown command: %s. Use 'up' or 'allDown'.", command)
	}

}
