# Цепочка обмена

MVP сервиса многостороннего обмена: пользователь добавляет вещь и описывает, что хочет
получить взамен, сервис собирает участников в цепочку, где каждый отдаёт ненужное и
получает нужное.

В backend уже есть схема БД, CRU пользователей, вход по JWT, защита обновления профиля и
CRUD объявлений с фотографиями и желаемыми категориями. Матчинг цепочек, механика сделки и
frontend добавляются отдельными задачами.

## Запуск

```bash
cp .env.example .env
make up      # поднимает PostgreSQL и накатывает миграции
make smoke   # проверяет схему на живой БД
make run     # HTTP API на порту из HTTP_ADDR (по умолчанию :8080)
```

Нужны Docker с плагином compose и Go 1.26+ из-за goose и sqlc

Маршруты можно потрогать через Swagger: <http://localhost:8080/swagger/> — см. `docs/swagger.md`

## CI

На каждый PR и push в `main`/`develop` работает `.github/workflows/ci.yml`:

- **backend** — поднимает `postgres:17-alpine`, катает миграции и гоняет `gofmt`,
  `go vet ./...`, `go test ./... -race` и integration-тест обмена. БД живая, поэтому тесты
  репозиториев не скипаются.
- **generated** — ставит sqlc v1.31.1 и swag v1.16.6, прогоняет `make sqlc` и
  `make swagger` и падает, если `internal/db/` или `docs/swagger/` разъехались с
  исходниками. Обновляешь версию генератора — правь и workflow.

Повторить локально (после `make up`; `DATABASE_URL` из `.env` должен быть в окружении,
иначе тесты с БД молча скипнутся):

```bash
gofmt -l . && go vet ./... && go test ./... -race
make test-exchange-integration
make sqlc swagger && git status --short   # дерево должно остаться чистым
```

Фронтенд в CI пока не участвует.

## Что где

| Путь              | Что там                                                          |
|-------------------|------------------------------------------------------------------|
| `migrations/`     | SQL-миграции goose, вшиты в binary через `embed.FS`              |
| `queries/`        | SQL-запросы, из которых sqlc генерирует Go-код                   |
| `internal/db/`    | Сгенерированный sqlc код — РУКАМИ НЕ ПРАВИТЬ! Только `make sqlc` |
| `cmd/migrate/`    | Накат миграций (`up`, `down`, `status`, `reset`)                 |
| `cmd/api/`        | Точка запуска HTTP API                                           |
| `db/smoke.sql`    | Проверка констрейнтов и триггеров на живой БД                    |
| `.github/`        | Шаблоны issue/PR и workflow CI                                   |
| `frontend/`       | Вся папка с фронтэндом                                           |
| `docs/db.md`      | Схема, обоснование выбора PostgreSQL и типов данных              |
| `docs/users.md`   | CRU пользователей, маршруты и коды ответа                        |
| `docs/items.md`   | CRUD объявлений: фотографии, желаемые категории, коды ответа     |
| `docs/auth.md`    | Вход, JWT, cookie и защищённые маршруты                          |
| `docs/swagger.md` | Интерактивная документация: как открыть и как перегенерировать   |
| `docs/swagger/`   | Сгенерированная спека — РУКАМИ НЕ ПРАВИТЬ! Только `make swagger` |
| `internal/user/`  | Handler, service, repository, DTO и model пользователей          |
| `internal/item/`  | Handler, service, repository, DTO и model объявлений             |
| `internal/auth/`  | Вход, JWT и middleware аутентификации                            |
