# Один образ на оба бинаря: api и migrate собираются из одного дерева одними
# зависимостями, делить их на два образа нечем.
FROM golang:1.26-alpine AS build

WORKDIR /src

# Зависимости отдельным слоем: правка кода не роняет кеш go mod download.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# CGO не нужен: pgx — чистый Go. Статический бинарь запускается в любом образе.
# Миграции вшиты через migrations/embed.go, SQL-файлы копировать не надо.
# -o в директорию именует бинари по их пакетам: /out/api и /out/migrate.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /out/ ./cmd/api ./cmd/migrate

FROM alpine:3.22

COPY --from=build /out/api /out/migrate /usr/local/bin/

# Каталог с владельцем создаётся здесь, а не на сервере: пустой том Docker наследует
# владельца точки монтирования из образа. Без этой строки том создался бы root:root,
# и api под nobody не смог бы записать в него ни одной фотографии — причём только на
# чистой машине, где тома ещё нет.
RUN mkdir -p /data/uploads && chown nobody /data/uploads

# nobody уже есть в alpine, заводить своего пользователя незачем.
USER nobody

CMD ["api"]
