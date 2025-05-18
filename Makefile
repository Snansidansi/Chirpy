DATABASE_URL := postgres://postgres:postgres@localhost:5432/chirpy
GOOSE_PREFIX := goose -dir sql/schema
APP_NAME := chirpy

goose:
	@$(GOOSE_PREFIX) postgres "$(DATABASE_URL)" $(filter-out $@,$(MAKECMDGOALS))

new-migration:
	@$(GOOSE_PREFIX) create -s $(filter-out $@,$(MAKECMDGOALS)) sql

build:
	@mkdir -p bin
	@go build -o ./bin/$(APP_NAME)

run: build
	@echo Running $(APP_NAME).
	@./bin/$(APP_NAME)

connect:
	@psql $(DATABASE_URL)

test:
	@go test ./...

%:
	@:
