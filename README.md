# Цепочка обмена

MVP сервиса многостороннего обмена: пользователь добавляет вещь и описывает, что хочет
получить взамен, сервис собирает участников в цепочку, где каждый отдаёт ненужное и
получает нужное.

В backend есть схема БД, пользователи и вход по JWT, CRUD объявлений, автоматический подбор
замкнутых обменов, решения участников, тред обмена и блокировки. Frontend закрывает основные
экраны: вход, профиль, свои вещи и обмены.

## Запуск

```bash
cp .env.example .env
make up      # собирает образы и поднимает БД, миграции, API и фронт
```

Приложение целиком — <http://localhost>. Swagger — <http://localhost/swagger/>,
см. `docs/swagger.md`.

Нужен только Docker с compose и buildx (`--mount=type=cache` в Dockerfile работает
через BuildKit; на Ubuntu это пакет `docker-buildx`). Если порт 80 занят, впиши в `.env`
свободный: `WEB_PORT=8081`.

Погасить — `make down`. Снести вместе с данными БД и подняться с нуля — `make reset`.

### Разработка

Гонять фронт и API вживую удобнее без пересборки образов:

```bash
make db      # только PostgreSQL с миграциями
make run     # API на порту из HTTP_ADDR (по умолчанию :8080)
make smoke   # проверка схемы на живой БД

cd frontend && npm ci && npm run dev   # Vite на :5173, /api уходит на :8080
```

Сервис `api` наружу не публикуется, поэтому `make up` и `make run` не спорят за `:8080`.

Здесь дополнительно нужны Go 1.26+ (goose и sqlc) и Node 20.19+ (vite 8 и eslint 10).

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

Фронтенд проверяет отдельный `.github/workflows/frontend.yml`: `npm ci`, `npm run lint:ci` и
`npm run build` на Node 22. Сборка там та же, что внутри образа веба, поэтому ошибка
TypeScript падает в CI, а не в `docker build`.

## Деплой

Push в `main` запускает `.github/workflows/cd.yml`: он собирает образы `api` и `web`,
пушит их в GHCR под тегом `github.sha`, копирует на сервер оба compose-файла и делает
`docker compose pull && up -d --no-build`. На сервере ничего не собирается, миграции
катает сервис `migrate` — тот же, что и локально.

Тесты в CD не дублируются: в `main` попадает только смёрженный PR, на котором прошёл CI.
Включи в настройках репозитория branch protection «require status checks», иначе прямой
push в `main` уедет на сервер непроверенным.

Секреты репозитория — три: `SSH_HOST`, `SSH_USER`, `SSH_KEY` (приватный ключ, публичный
лежит в `~/.ssh/authorized_keys` на сервере). `GITHUB_TOKEN` встроенный, заводить его не
нужно.

Сервер готовится один раз руками — нужен docker с compose и каталог `/opt/chain-of-trades`
с `.env` в нём. Compose-файлы туда положит сам деплой, а `.env` через GitHub не ходит:
`JWT_SECRET` и пароль базы живут там же, где том с данными. Отличия от локального `.env` —
`COOKIE_SECURE=true` и `SITE_ADDRESS=<домен>`, по которому Caddy сам выпустит сертификат.

Откат — прошлый образ ещё в GHCR, поэтому на сервере:

```bash
cd /opt/chain-of-trades
IMAGE_TAG=<sha предыдущего деплоя> docker compose \
  -f docker-compose.yml -f docker-compose.prod.yml up -d --no-build
```

Бэкапов БД пока нет: том `pgdata` — единственная копия данных.

## Что где

| Путь              | Что там                                                          |
|-------------------|------------------------------------------------------------------|
| `Dockerfile`      | Образ Go: бинари `api` и `migrate` из одной сборки               |
| `docker-compose.yml` | Четыре сервиса: `postgres`, `migrate`, `api`, `web`           |
| `docker-compose.prod.yml` | Оверлей для сервера: образы из GHCR вместо сборки на месте |
| `frontend/Dockerfile` | Образ веба: сборка фронта и Caddy, который её отдаёт         |
| `frontend/Caddyfile`  | Один origin: статика, `/api/*` на API, TLS при домене        |
| `migrations/`     | SQL-миграции goose, вшиты в binary через `embed.FS`              |
| `queries/`        | SQL-запросы, из которых sqlc генерирует Go-код                   |
| `internal/db/`    | Сгенерированный sqlc код — РУКАМИ НЕ ПРАВИТЬ! Только `make sqlc` |
| `cmd/migrate/`    | Накат миграций (`up`, `down`, `status`, `reset`)                 |
| `cmd/api/`        | Точка запуска HTTP API                                           |
| `db/smoke.sql`    | Проверка констрейнтов и триггеров на живой БД                    |
| `.github/`        | Шаблоны issue/PR и workflow CI и CD                              |
| `frontend/`       | Вся папка с фронтэндом                                           |
| `docs/db.md`      | Схема, обоснование выбора PostgreSQL и типов данных              |
| `docs/users.md`   | CRU пользователей, маршруты и коды ответа                        |
| `docs/items.md`   | CRUD объявлений: фотографии, желаемые категории, коды ответа     |
| `docs/auth.md`    | Вход, JWT, cookie и защищённые маршруты                          |
| `docs/exchanges.md` | Обмены: чтение, решения участников и счётчик непрочитанного    |
| `docs/exchange-search.md` | Как DFS находит замкнутый обмен по графу объявлений      |
| `docs/exchange-messages.md` | Тред обмена: переписка участников и события сделки    |
| `docs/user-blocks.md` | Блокировка пользователей и её влияние на подбор обменов      |
| `docs/admin-exchange-cancellation.md` | Принудительная отмена обмена администратором      |
| `docs/reports.md` | Жалобы на сообщения треда: причины, запреты и очередь модерации   |
| `docs/swagger.md` | Интерактивная документация: как открыть и как перегенерировать   |
| `docs/swagger/`   | Сгенерированная спека — РУКАМИ НЕ ПРАВИТЬ! Только `make swagger` |
| `internal/user/`  | Handler, service, repository, DTO и model пользователей          |
| `internal/item/`  | Handler, service, repository, DTO и model объявлений             |
| `internal/auth/`  | Вход, JWT и middleware аутентификации                            |
| `internal/report/`| Жалобы на сообщения треда: приём и запись в очередь модерации     |
