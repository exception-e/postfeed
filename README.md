# PostFeed

Мини-сервис для создания постов и комментариев с поддержкой двух хранилищ: in-memory и PostgreSQL и GraphQL api.

Выбор хранилища управляется переменной окружения `STORAGE_TYPE`.

## Стек

- Go 1.26
- GraphQL (gqlgen)
- PostgreSQL 18
- Docker / Docker Compose

## API
Сервис позволяет посмотреть список постов, пост и комментарии к нему, запретить комментарии к своему посту.
API позволяет получать список постов и первые/корневые комментарии для каждого поста.

Reply-комментарии не входят в выдачу comments: корневой комментарий — это комментарий с parent_id = null.

Комментарии организованы иерархически и упорядочены по id ASC. В качестве id используется UUIDv7, такой порядок, с некоторым приближением, соответствует порядку создания.

Для устранения N+1 запросов comments загружаются batch-ом через DataLoader.

Запрос вида `posts { comments(limit: 3) }` выполняет одну batch-загрузку root comments для всех posts из текущей страницы, а не отдельный SQL-запрос для каждого post.

Предполагается, что userId передан из сервиса авторизации

## Схема данных

```mermaid
erDiagram
    USERS ||--o{ POSTS : user
    USERS ||--o{ COMMENTS : user
    POSTS ||--o{ COMMENTS : 
    COMMENTS ||--o{ COMMENTS : replies

    POSTS {
        uuid id PK
        uuid user_id 
        text body
        boolean comments_enabled
        timestamptz created_at
        timestamptz updated_at
    }
    COMMENTS {
        uuid id PK
        uuid user_id
        uuid post_id FK
        uuid parent_id FK "nullable"
        text body
        timestamp created_at
    }
```

## Иерархия комментариев

```mermaid
graph TD
    P[Post]
    C1[Comment 1<br/>parent_id = null]
    C2[Comment 2<br/>parent_id = null]
    R1[Reply 1<br/>parent_id = C1]
    R2[Reply 2<br/>parent_id = C1]
    R3[Reply 3<br/>parent_id = R1]

    P --> C1
    P --> C2
    C1 --> R1
    C1 --> R2
    R1 --> R3
```

## Создание post

```graphql
mutation {
  createPost(input: {
    body: "First post"
  }) {
    id
    body
    commentsEnabled
    createdAt
    updatedAt
  }
}
```

Headers:
```json
{
  "user-id": "47c52a73-c599-4c34-86cb-4969bfc0e730"
}
```

Пример ответа:

```json
{
  "data": {
    "createPost": {
      "id": "01a1110a-68b1-7889-a0bf-b249757cd581",
      "body": "Hello in docker!",
      "commentsEnabled": true,
      "createdAt": "2026-10-06T11:47:37Z",
      "updatedAt": "2026-10-06T11:47:37Z"
    }
  }
}
```

`userId` определяется из авторизованного пользователя на сервере и не передаётся клиентом в input.

## Создание root comment

Root comment создаётся без `parentId`.

```graphql
mutation {
  createComment(input: {
    postId: "01a1110a-68b1-7889-a0bf-b249757cd581"
    body: "Root comment!"
  }) {
    id
    postId
    parentId
    body
    createdAt
  }
}
```

Headers:
```json
{
"user-id": "47c52a73-c599-4c34-86cb-4969bfc0e730"
}
```

Пример ответа:
```json
{
  "data": {
    "createComment": {
      "id": "019c1cd4-5b31-7564-b5ea-7ccf9207619a",
      "postId": "019c1cd4-3b54-7a22-80b9-2fd5f88c83de",
      "parentId": null,
      "body": "Root comment!",
      "createdAt": "2026-09-21T11:01:00Z"
    }
  }
}
```

### Получение posts

```graphql
query {
  posts(first: 50) {
    nodes {
      id
      userId
      body
      commentsEnabled
      comments(first: 20) {
        nodes {
          id
          body
          replies(first: 20) {
            nodes {
              id
              body
            }
          }
        }
      }
      createdAt
      updatedAt
    }
    nextCursor
  }
}
```

Headers:
```json
{
"user-id": "47c52a73-c599-4c34-86cb-4969bfc0e730"
}
```
## Batch-загрузка comments

Запрос posts с вложенным полем `comments` потенциально создаёт N+1 проблему

Для устранения N+1 используется DataLoader. Все обращения к `comments(limit: N)` в рамках одного GraphQL-request объединяются в batch-запрос.

```graphql
query {
  posts(first: 50) {
    nodes {
      id
      userId
      body
      commentsEnabled
      comments(first: 20) {
        nodes {
          id
          body
          replies(first: 20) {
            nodes {
              id
              body
            }
          }
        }
      }
      createdAt
      updatedAt
    }
    nextCursor
  }
}
```

## DataLoader

```mermaid
sequenceDiagram
participant C as Client
participant G as GraphQL
participant D as DataLoader
participant DB as PostgreSQL

    C->>G: posts { comments(limit: 3) }
    G->>DB: SELECT posts
    DB-->>G: posts
    G->>D: load comments for post_ids
    D->>DB: SELECT comments WHERE post_id IN (...)
    DB-->>D: comments
    D-->>G: batched
    G-->>C: response
```

## Изменение commentsEnabled

```graphql
mutation {
  updatePostCommentsEnabled(input: {
    postId: "01a1110a-68b1-7889-a0bf-b249757cd581"
    commentsEnabled: false
  }) {
    id
    commentsEnabled
    updatedAt
  }
}
```
## Старт через Docker

### 1. Клонировать репозиторий

```bash
git clone https://github.com/exception-e/postfeed
cd postfeed
```

### 2. Создать `.env`

Скопируйте пример и при необходимости поправьте значения:

```bash
cp .env.example .env
```

`.env.example`:

```env
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
STORAGE_TYPE=psql docker compose --profile psql up --build
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
| `POSTGRES_USER` | `app` | Пользователь базы |
| `POSTGRES_PASSWORD` | `secret` | Пароль |
| `POSTGRES_DB` | `posts` | Имя базы |

STORAGE_TYPE передается в командной строке при запуске для psql версии, по умолчанию STORAGE_TYPE=inmemory 
## Запуск без Docker

```bash
# In-memory
go run cmd/main.go

# PostgreSQL (нужна запущенная база)
POSTGRES_HOST=localhost \
POSTGRES_PORT=5433 \
POSTGRES_USER=app \
POSTGRES_PASSWORD=secret \
POSTGRES_DB=posts \

STORAGE_TYPE=psql DATABASE_URL="postgres://app:secret@localhost:5433/posts?sslmode=disable" go run cmd/main.go 
```
