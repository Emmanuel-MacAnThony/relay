FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o relay ./cmd/relay
RUN go install -tags 'pgx5' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:3.21
WORKDIR /app

COPY --from=builder /app/relay .
COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/db/migrations ./db/migrations
COPY entrypoint.sh .

EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
