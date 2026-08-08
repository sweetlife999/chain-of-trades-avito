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
	admindashboardhandler "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/handler"
	admindashboardrepository "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/repository"
	admindashboardservice "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/service"
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
	pickuppointhandler "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/handler"
	pickuppointrepository "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/repository"
	pickuppointservice "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/service"
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
// @description Защищённые маршруты читают HttpOnly cookie `access_token` — они помечены замком.
// @description Кнопки «Authorize» здесь нет и не нужно: выполните `POST /auth/login` прямо из этой
// @description страницы — браузер сохранит cookie и будет отправлять её со всеми следующими
// @description запросами сам. JavaScript до неё не дотянется, поэтому вписать её руками нельзя.
// @BasePath    /
//
// @securityDefinitions.apikey CookieAuth
// @in                         cookie
// @name                       access_token
// @description                HttpOnly cookie, которую ставит POST /auth/login.
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
	exchangesRepository := exchangerepository.New(pool)
	exchanges := exchangeservice.New(exchangesRepository)
	exchangesHandler := exchangehandler.New(exchanges)
	items := itemservice.New(itemrepository.New(pool), exchanges)
	pickupPoints := pickuppointservice.New(pickuppointrepository.New(queries))
	adminDashboard := admindashboardservice.New(admindashboardrepository.New(queries))

	tokens := authtoken.NewManager(cfg.JWTSecret, authTokenTTL)
	authenticator := authmiddleware.New(tokens)
	adminAuthorizer := authmiddleware.NewAdminAuthorizer(users)
	auth := authservice.New(usersRepository, tokens)

	userhandler.New(users).RegisterRoutes(router, authenticator.RequireAuthentication)
	itemhandler.New(items).RegisterRoutes(router, authenticator.RequireAuthentication)
	authhandler.New(auth, cfg.CookieSecure, authTokenTTL).
		RegisterRoutes(router, authenticator.RequireAuthentication)
	exchangesHandler.RegisterRoutes(router, authenticator.RequireAuthentication)

	// Все следующие административные модули регистрируются только внутри этой группы.
	// JWT сначала определяет пользователя, затем роль проверяется по актуальным данным БД.
	router.Route("/admin", func(adminRouter chi.Router) {
		adminRouter.Use(authenticator.RequireAuthentication)
		adminRouter.Use(adminAuthorizer.RequireAdmin)
		admindashboardhandler.New(adminDashboard).RegisterRoutes(adminRouter)
		pickuppointhandler.New(pickupPoints).RegisterRoutes(adminRouter)
		exchangesHandler.RegisterAdminRoutes(adminRouter)
	})

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
