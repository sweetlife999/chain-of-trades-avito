-include .env
export


.PHONY: lint run up db down reset migrate-up migrate-down migrate-status sqlc smoke swagger test-exchange-integration test-user-blocks-integration test-exchange-recovery-integration test-exchange-refusal-integration test-exchange-messages-integration test-reports-integration

# Линтер. Требует golangci-lint v2 той же версии, что пиннится в CI:
# go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
lint:
	golangci-lint run ./...

run:
	go run ./cmd/api

# Поднять приложение целиком: БД, миграции, API и фронт на http://localhost
up:
	docker compose up -d --build

# Только БД с миграциями — для разработки, дальше make run и npm run dev
db:
	docker compose up -d postgres migrate

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

# Пересобрать Swagger из аннотаций в хэндлерах. Требует swag:
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

# Живые сценарии: поиск обмена и конкурентное подтверждение/отказ с резервированием.
test-exchange-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run 'Test(ThreeUserExchange|ExchangeDecisions)Integration' -count=1

# Живой сценарий треда: переписка, события сделки и счётчик непрочитанного.
test-exchange-messages-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run TestExchangeMessagesIntegration -count=1

# Живой сценарий блокировок: API, отмена proposed-обмена и фильтрация DFS.
test-user-blocks-integration:
	go test -tags=integration ./internal/user/handler \
		-run TestUserBlocksIntegration -count=1

# Живые сценарии восстановления: отказ, освобождение reserved-вещей и новый DFS,
# плюс возврат цепочки, которую вытеснило чужое подтверждение.
test-exchange-recovery-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run RecoveryIntegration -count=1

# Живой сценарий отказа: вырезанное ребро графа и уцелевшие рёбра отказанного цикла.
test-exchange-refusal-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run TestExchangeRefusalIntegration -count=1

# Живой сценарий жалоб: успешная жалоба и четыре запрета, два из которых держит база.
test-reports-integration:
	go test -tags=integration ./internal/report/handler \
		-run TestReportsIntegration -count=1
