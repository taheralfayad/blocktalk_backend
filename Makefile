# Makefile 

CONTAINER_NAME=web

migrate-up:
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go up

migrate-all-down:
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go allDown

migrate-to-version:
ifeq ($(VERSION),)
	$(error USAGE: make migrate-to-version VERSION=<version_number>)
endif
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go toVersion $(VERSION)

force-migrate:
ifeq ($(VERSION),)
	$(error USAGE: make force-migrate VERSION=<version_number>)
endif
	docker compose exec $(CONTAINER_NAME) go run migrate/migrate.go force $(VERSION)