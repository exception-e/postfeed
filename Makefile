up-memory:
	docker compose up --build

up-psql:
	STORAGE_TYPE=psql docker compose --profile psql up --build

down:
	docker compose --profile psql down -v