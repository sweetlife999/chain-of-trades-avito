FROM node:22-alpine AS build

WORKDIR /app

# Зависимости отдельным слоем: правка кода не роняет кеш npm ci.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci

COPY . .

# Vite запекает переменную в бандл, поэтому она нужна на сборке, а не на запуске.
ENV VITE_BASE_URL=/api
RUN npm run build

FROM caddy:2-alpine

COPY Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/dist /srv
