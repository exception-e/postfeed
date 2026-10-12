-include .env

.PHONY: up-memory down-memory up-psql down-psql app-logs generate test lint

up-memory:
	docker compose up --build --detach

up-memory-no-build:
	docker compose up --detach

down-memory:
	docker compose down

up-psql:
	STORAGE_TYPE=psql docker compose --profile psql up --build  --detach

up-psql-no-build:
	STORAGE_TYPE=psql docker compose --profile psql up --detach

down-psql:
	docker compose --profile psql down --volumes

app-logs:
	docker compose logs app -f

generate:
	go generate ./...

test:
	go test ./...
	go test -tags=integration ./...

lint:
	golangci-lint run ./...