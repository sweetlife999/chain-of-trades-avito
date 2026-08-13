-include .env
export


.PHONY: lint run up db llm down reset migrate-up migrate-down migrate-status sqlc smoke swagger test-ratings-integration test-exchange-integration test-user-blocks-integration test-exchange-recovery-integration test-exchange-refusal-integration test-exchange-messages-integration test-item-search-visibility-integration test-delivery-integration test-reports-integration test-admin-audit-integration test-notifications-integration test-antiscam test-support-bot-llm test-item-assistant-llm test-support-admin-integration

# Линтер. Требует golangci-lint v2 той же версии, что пиннится в CI:
# go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
lint:
	golangci-lint run ./...

run:
	go run ./cmd/api

# Поднять приложение целиком: БД, миграции, API и фронт на http://localhost.
# Пересборку держит pull_policy: build в compose, поэтому здесь --build не нужен.
up:
	docker compose up -d

# Только БД с миграциями — для разработки, дальше make run и npm run dev
db:
	docker compose up -d postgres migrate

# Только модель: поднимает Ollama и докачивает веса. Отдельно от db, потому что
# большинству задач модель не нужна, а первый запуск тянет сотни мегабайт.
# Проверить, чем кончилось скачивание: docker compose logs ollama-pull
llm:
	docker compose up -d ollama ollama-pull

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
		-run 'Test(ThreeUserExchange|ExchangeDecisions|AsyncExchangeSearch|ExchangeRanking)Integration' -count=1

# Живой сценарий треда: переписка, события сделки и счётчик непрочитанного.
test-exchange-messages-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run TestExchangeMessagesIntegration -count=1

# Живой сценарий управления поиском: снятие вещи, отмена предложений, событие в чате,
# повторная публикация и запрет снятия уже зарезервированной вещи.
test-item-search-visibility-integration:
	go test -tags=integration ./internal/item/handler \
		-run TestItemSearchVisibilityIntegration -count=1

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

# Живой сценарий доставки: сдача вещей в пункт, переход в delivering с обеих сторон,
# запрет «Товар получен» до выдачи и снятие пункта с завершённых вещей.
test-delivery-integration:
	go test -tags=integration ./internal/exchange/handler \
		-run TestExchangeDeliveryIntegration -count=1

# Живой сценарий жалоб: успешная жалоба и четыре запрета, два из которых держит база.
test-reports-integration:
	go test -tags=integration ./internal/report/handler \
		-run TestReportsIntegration -count=1

# Живой сценарий глобальной блокировки: атомарная запись аудита и немедленный
# запрет уже выданного JWT через актуальное состояние аккаунта в БД.
test-admin-audit-integration:
	go test -tags=integration ./internal/adminaudit/handler \
		-run TestAdminUserBlockAuditIntegration -count=1

# Живой сценарий центра уведомлений: предложение, сообщение, событие сделки,
# изоляция пользователей и отметки одного/всех уведомлений прочитанными.
test-notifications-integration:
	go test -tags=integration ./internal/notification/handler \
		-run TestNotificationsIntegration -count=1

# Живой сценарий оценки: партнёра назначает сам цикл, перезапись оценки, пересчёт
# среднего балла триггером и отказы постороннему, незавершённому обмену и просроченному сроку.
test-ratings-integration:
	go test -tags=integration ./internal/rating/handler \
		-run TestRatingsIntegration -count=1

test-antiscam:
	go test ./internal/antiscam/... -count=1

# Точность роутера поддержки на замороженном наборе обращений. Требует живую Ollama
# (OLLAMA_URL) и нужна при любой правке промпта в internal/support/service/bot.go:
# 0.5B гиперчувствительна к формулировке, поэтому промпт меряется, а не обсуждается.
# Без OLLAMA_URL тест скипается.
test-support-bot-llm:
	OLLAMA_URL=$${OLLAMA_URL:-http://localhost:11434} \
	go test -tags=integration ./internal/support/service \
		-run TestSupportBotAccuracy -count=1 -v

# Живой набор объявлений проверяет и текст, и выбор категории на той же 0.5B-модели,
# что используется в проде. Без поднятой Ollama тест скипается.
test-item-assistant-llm:
	OLLAMA_URL=$${OLLAMA_URL:-http://localhost:11434} \
	go test -tags=integration ./internal/itemassistant/service \
		-run TestItemAssistantModel -count=1 -v

# Живой сценарий очереди модерации поддержки: превью берётся из сообщения автора
# обращения, в том числе когда автор — сам администратор. Тест написан по факту падения:
# на таком обращении список отдавал 500 целиком.
test-support-admin-integration:
	go test -tags=integration ./internal/support/repository \
		-run TestAdminSupportQueueIntegration -count=1 -v
