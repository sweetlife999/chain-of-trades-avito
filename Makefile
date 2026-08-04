-include .env
export

.PHONY: run up down reset migrate-up migrate-down migrate-status sqlc smoke

run:
	go run ./cmd/api

# Поднять БД и накатить миграции (сервис migrate отработает и выйдет)
up:
	docker compose up -d

down:
	docker compose down

# Снести вместе с томом и подняться с нуля
reset:
	docker compose down -v
	docker compose up -d

migrate-up:
	go run ./cmd/migrate up

migrate-down:
	go run ./cmd/migrate down

migrate-status:
	go run ./cmd/migrate status

sqlc:
	sqlc generate

# Проверка схемы: констрейнты, индексы и триггер на живой БД. Ничего не оставляет после себя.
smoke:
	docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB) < db/smoke.sql
