FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/nanojira ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /bin/nanojira /app/nanojira
COPY migrations /app/migrations

ENV MIGRATIONS_DIR=/app/migrations

EXPOSE 8080

CMD ["/app/nanojira"]
