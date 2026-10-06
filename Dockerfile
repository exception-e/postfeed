FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /app/postfeed ./cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/postfeed .
COPY --from=builder /app/internal/storage/psql/migrations ./internal/storage/psql/migrations
EXPOSE 8080

CMD ["./postfeed"]