# Makefile 

CONTAINER_NAME=web

migrate-up:
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go up

migrate-all-down:
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go allDown
