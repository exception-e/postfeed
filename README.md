# postfeed

# PostFeed

Мини-сервис для постов и комментариев с поддержкой двух хранилищ:
**in-memory** и **PostgreSQL**. Выбор хранилища управляется переменной окружения
`STORAGE_TYPE`.

## Стек

- Go 1.26
- GraphQL (gqlgen)
- PostgreSQL 18
- Docker / Docker Compose

## API
API позволяет получать posts и первые root comments для каждого поста.
Reply-комментарии не входят в выдачу comments: root comment — это комментарий с parent_id = null.

Комментарии упорядочены по id ASC. В качестве id используется UUIDv7, поэтому такой порядок соответствует порядку создания с практической точностью.

### Получение posts

```graphql
query {
  posts(limit: 20) {
    id
    userId
    body
    createdAt
  }
}
```
Получение posts с preview комментариев
### Получение posts с preview комментариев

Аргумент `limit` у `comments` применяется **отдельно к каждому post**, а не ко всему списку posts.

```graphql
query {
  posts(limit: 20) {
    id
    body
    createdAt

    comments(limit: 3) {
      id
      userId
      body
      createdAt
    }
  }
}
```

Пример результата:

```json
{
  "data": {
    "posts": [
      {
        "id": "019c1cd4-3b54-7a22-80b9-2fd5f88c83de",
        "body": "Первый post",
        "createdAt": "2026-09-21T11:00:00Z",
        "comments": [
          {
            "id": "019c1cd4-5b31-7564-b5ea-7ccf9207619a",
            "userId": "019c1cd4-1da1-7be6-8f7c-d4c0847527fd",
            "body": "Первый root comment",
            "createdAt": "2026-09-21T11:01:00Z"
          }
        ]
      }
    ]
  }
}
```

### Получение replies комментария

```graphql
query {
  comment(id: "019c1cd4-5b31-7564-b5ea-7ccf9207619a") {
    id
    body

    replies(limit: 20) {
      id
      userId
      body
      createdAt
    }
  }
}
```

## Старт через Docker

### 1. Клонировать репозиторий

```bash
git clone <repo-url>
cd postfeed
```

### 2. Создать `.env`

Скопируйте пример и при необходимости поправьте значения:

```bash
cp .env.example .env
```

`.env.example`:

```env
# Хранилище: inmemory или psql
STORAGE_TYPE=inmemory

# Порт приложения на хосте
APP_PORT=8080

# Параметры PostgreSQL (используются, если STORAGE_TYPE=psql)
POSTGRES_USER=app
POSTGRES_PASSWORD=secret
POSTGRES_DB=posts
```
### 3. Запуск

**In-memory режим** (Postgres не поднимается):

```bash
docker compose up --build
```

**PostgreSQL режим** (поднимается контейнер с базой):

```bash
docker compose --profile psql up --build
```

Приложение будет доступно по адресу `http://localhost:${APP_PORT}` (по
умолчанию `http://localhost:8080`).

### 4. Подключение к базе (psql-режим)

С хоста база доступна на `localhost:5433` (проброс порта в compose).
Параметры:

| Параметр | Значение |
|---|---|
| Host | `localhost` |
| Port | `5433` |
| Database | `posts` |
| User | `app` |
| Password | `secret` |

JDBC URL для IntelliJ IDEA:

```
jdbc:postgresql://localhost:5433/posts
```


### 5. Остановка

```bash
docker compose down
```

С удалением данных:

```bash
docker compose down -v
```

## Переменные окружения

| Переменная | По умолчанию | Описание |
|---|---|---|
| `STORAGE_TYPE` | `inmemory` | `inmemory` или `psql` |
| `APP_PORT` | `8080` | Порт приложения на хосте |
| `POSTGRES_HOST` | `postgres` | Хост базы (задаётся в compose) |
| `POSTGRES_PORT` | `5432` | Порт базы внутри compose-сети |
| `POSTGRES_USER` | `app` | Пользователь базы |
| `POSTGRES_PASSWORD` | `secret` | Пароль |
| `POSTGRES_DB` | `posts` | Имя базы |

## Запуск без Docker

```bash
# In-memory
STORAGE_TYPE=inmemory go run ./cmd/postfeed

# PostgreSQL (нужна запущенная база)
STORAGE_TYPE=psql \
POSTGRES_HOST=localhost \
POSTGRES_PORT=5432 \
POSTGRES_USER=app \
POSTGRES_PASSWORD=secret \
POSTGRES_DB=posts \
go run ./cmd/postfeed
```