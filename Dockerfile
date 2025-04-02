# syntax=docker/dockerfile:1
FROM alpine:3.21 AS base-alpine
WORKDIR /app

FROM golang:1.23.6-alpine3.21 AS base-go
WORKDIR /app

FROM base-go AS build
COPY go.mod go.sum ./
COPY cmd/web/main.go .
COPY configs/ ./configs/
COPY internal/ ./internal/
COPY repository/ ./repository/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o app

FROM base-alpine AS final
COPY --from=build /app/app .
COPY .env .
ENTRYPOINT ["/app/app"]