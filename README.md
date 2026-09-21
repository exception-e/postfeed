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

Для устранения N+1 запросов comments загружаются batch-ом через DataLoader.

Запрос вида `posts { comments(limit: 3) }` выполняет одну batch-загрузку root comments для всех posts из текущей страницы, а не отдельный SQL-запрос для каждого post.

В текущей реализации totalCount не работают

## Создание post

```graphql
mutation {
  createPost(input: {
    body: "First post"
  }) {
    id
    userId
    body
    commentsEnabled
    createdAt
    updatedAt
  }
}
```

Пример ответа:

```json
{
  "data": {
    "createPost": {
      "id": "019c1cd4-3b54-7a22-80b9-2fd5f88c83de",
      "userId": "019c1cd4-1da1-7be6-8f7c-d4c0847527fd",
      "body": "First post",
      "commentsEnabled": true,
      "createdAt": "2026-09-21T11:00:00Z",
      "updatedAt": "2026-09-21T11:00:00Z"
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
    postId: "019c1cd4-3b54-7a22-80b9-2fd5f88c83de"
    body: "Hello world!"
  }) {
    id
    postId
    parentId
    userId
    body
    createdAt
  }
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
      "userId": "019c1cd4-1da1-7be6-8f7c-d4c0847527fd",
      "body": "Hello world!",
      "createdAt": "2026-09-21T11:01:00Z"
    }
  }
}
```

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
## Batch-загрузка comments

Запрос posts с вложенным полем `comments` потенциально создаёт N+1 проблему

Для устранения N+1 используется DataLoader. Все обращения к `comments(limit: N)` в рамках одного GraphQL-request объединяются в batch-запрос.

```graphql
query {
  posts(limit: 3) {
    id
    body

    comments(limit: 2) {
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
## Изменение commentsEnabled

```graphql
mutation {
  updatePostCommentsEnabled(input: {
    postId: "019c1cd4-3b54-7a22-80b9-2fd5f88c83de"
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

## Точки роста
 - Счётчики комментариев как метрика для кеширования и пагинации, а также показывать количество комментариев к посту сразу
Хранить `comments_count` прямо в `posts`
 - Разделить `title` и `body`, чтобы использовать NoSQL-хранилище, грузить заголовок и начало поста для ленты, для индексации
 - Кеширование популярных постов для увеличения скорости и уменьшения нагрузки
 - Полнотекстовый поиск по постам (`tsvector`)
 - Rate limiting на создание постов и комментариев.
 - Асинхронная обработка (очередь на создание комментария)
 - Метрики и трейсинг (Prometheus + OpenTelemetry) для
  наблюдаемости