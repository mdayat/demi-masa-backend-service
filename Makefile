# uncomment two lines of code below when running `run` command 
include ./.test.env
export

.DEFAULT_GOAL := run

.PHONY:fmt vet build run govulncheck staticcheck revive

.SILENT:

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

build: vet
	go build -C cmd/web -o app

run:
	go run cmd/web/main.go

seed:
	docker run -d --name postgres -p 5432:5432 --env-file ./.test.env postgres:15
	@until docker exec postgres pg_isready -U postgres; do \
		sleep 1; \
	done
	atlas migrate apply --env prod -u "$(DATABASE_URL)" --revisions-schema public
	go run cmd/seed/main.go

test: seed
	go test ./internal/handlers -v

govulncheck:
	govulncheck ./...

staticcheck:
	staticcheck ./...

revive:
	revive -config revive.toml -formatter friendly ./...