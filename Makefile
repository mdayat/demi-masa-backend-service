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
	docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:15
	@until docker exec postgres pg_isready -U postgres; do \
		sleep 1; \
	done
	atlas migrate apply --env prod -u "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable" --revisions-schema public
	go run cmd/seed/main.go

test: seed
	go test ./internal/handlers -v

govulncheck:
	govulncheck ./...

staticcheck:
	staticcheck ./...

revive:
	revive -config revive.toml -formatter friendly ./...