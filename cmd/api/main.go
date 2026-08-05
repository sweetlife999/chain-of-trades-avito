package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	// Регистрирует сгенерированную спеку, которую отдаёт /swagger. Обновляется через make swagger.
	_ "github.com/sweetlife999/chain-of-trades-avito/docs/swagger"
	authhandler "github.com/sweetlife999/chain-of-trades-avito/internal/auth/handler"
	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	authservice "github.com/sweetlife999/chain-of-trades-avito/internal/auth/service"
	authtoken "github.com/sweetlife999/chain-of-trades-avito/internal/auth/token"
	"github.com/sweetlife999/chain-of-trades-avito/internal/config"
	"github.com/sweetlife999/chain-of-trades-avito/internal/database"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangehandler "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/handler"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
	itemhandler "github.com/sweetlife999/chain-of-trades-avito/internal/item/handler"
	itemrepository "github.com/sweetlife999/chain-of-trades-avito/internal/item/repository"
	itemservice "github.com/sweetlife999/chain-of-trades-avito/internal/item/service"
	userhandler "github.com/sweetlife999/chain-of-trades-avito/internal/user/handler"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

const authTokenTTL = 12 * time.Hour

// @title       Цепочка обмена — API
// @version     0.1.0
// @description HTTP API сервиса многостороннего обмена вещами: профили пользователей, вход по JWT
// @description и объявления о вещах, которые владелец готов обменять.
// @description
// @description Защищённые маршруты читают HttpOnly cookie `access_token`. Кнопки «Authorize» здесь нет
// @description и не нужно: выполните `POST /auth/login` прямо из этой страницы — браузер сохранит cookie
// @description и будет отправлять её со всеми следующими запросами сам.
// @BasePath    /
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := database.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	queries := db.New(pool)
	usersRepository := userrepository.New(queries)
	users := userservice.New(usersRepository)
	items := itemservice.New(itemrepository.New(pool))

	tokens := authtoken.NewManager(cfg.JWTSecret, authTokenTTL)
	authenticator := authmiddleware.New(tokens)
	auth := authservice.New(usersRepository, tokens)
	exchangesRepository := exchangerepository.New(pool)
	exchanges := exchangeservice.New(exchangesRepository)

	userhandler.New(users).RegisterRoutes(router, authenticator.RequireAuthentication)
	itemhandler.New(items).RegisterRoutes(router, authenticator.RequireAuthentication)
	authhandler.New(auth, cfg.CookieSecure, authTokenTTL).
		RegisterRoutes(router, authenticator.RequireAuthentication)
	exchangehandler.New(exchanges).RegisterRoutes(router, authenticator.RequireAuthentication)

	router.Get("/health", health)

	// chi не матчит "/swagger" шаблоном "/swagger/*", поэтому путь без слеша уводим на index.
	router.Get("/swagger", http.RedirectHandler("/swagger/index.html", http.StatusMovedPermanently).ServeHTTP)
	router.Get("/swagger/*", httpSwagger.WrapHandler)

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("HTTP server started on %s", cfg.HTTPAddress)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// @Summary     Проверка живости сервиса
// @Description Отвечает 200, если процесс поднят. Состояние БД не проверяет.
// @Tags        system
// @Produce     json
// @Success     200 {object} map[string]string "сервис работает"
// @Router      /health [get]
func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
