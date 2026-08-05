-include .env
export

.PHONY: run up down reset migrate-up migrate-down migrate-status sqlc smoke swagger test-exchange-integration

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

# Пересобрать спеку Swagger из аннотаций в хэндлерах. Требует swag:
# go install github.com/swaggo/swag/cmd/swag@latest
# Ищем от cmd/api и идём по графу импортов: в корне репозитория нет .go-файлов,
# а без них swag не может определить путь модуля и не находит типы из internal/.
swagger:
	swag init -g main.go -d cmd/api -o docs/swagger --packageName swagger \
		--parseInternal --parseDependencyLevel 3 \
		--packagePrefix github.com/sweetlife999/chain-of-trades-avito

# Проверка схемы: констрейнты, индексы и триггер на живой БД. Ничего не оставляет после себя.
smoke:
	docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U $(POSTGRES_USER) -d $(POSTGRES_DB) < db/smoke.sql

# Живой сценарий: три пользователя и три объявления -> поиск -> сохранение -> HTTP API.
test-exchange-integration:
	go test -tags=integration ./internal/exchange/handler -run TestThreeUserExchangeIntegration -count=1
