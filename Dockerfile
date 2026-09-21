# Этап сборки
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Копируем файлы зависимостей и скачиваем их
#COPY go.mod go.sum ./

# Копируем исходный код и собираем бинарник
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o /app/postfeed ./cmd/main.go

# Финальный этап
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из этапа сборки
COPY --from=builder /app/postfeed .

# Переменные окружения по умолчанию
ENV STORAGE_TYPE=inmemory

EXPOSE 8080

CMD ["./postfeed"]