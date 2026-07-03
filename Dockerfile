# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /web-security-proxy ./cmd/server

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /web-security-proxy /app/web-security-proxy
COPY config.yaml /app/config.yaml
COPY db/migrations /app/db/migrations
COPY web /app/web

RUN mkdir -p /app/data

EXPOSE 8080 8888

CMD ["/app/web-security-proxy"]
