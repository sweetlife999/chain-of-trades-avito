# Frontend

Во время локальной разработки браузер обращается только к Vite на
`http://localhost:5173`. Запросы с префиксом `/api` Vite перенаправляет в Go backend и
удаляет `/api` из пути:

```text
POST http://localhost:5173/api/auth/login
                         ↓
POST http://localhost:8080/auth/login
```

Так frontend и cookie `access_token` работают через один origin, поэтому отдельная
настройка CORS для локальной разработки не нужна.

## Запуск

Сначала из корня проекта запустите базу и backend:

```bash
cp .env.example .env
make up
make run
```

Затем во втором терминале запустите frontend:

```bash
cd frontend
cp .env.example .env
npm install
npm run dev
```

Frontend должен использовать относительные API-адреса:

```ts
await axios.post("/api/auth/login", { nickname, password });
await axios.get("/api/auth/me");
await axios.get("/api/items");
await axios.get("/api/exchanges");
```

Не нужно писать `http://localhost:8080` внутри компонентов. Если backend запущен на
другом адресе, измените `VITE_BACKEND_URL` в `frontend/.env` и перезапустите Vite.
