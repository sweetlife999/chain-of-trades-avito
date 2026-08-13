package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	// Регистрирует сгенерированную спеку, которую отдаёт /swagger. Обновляется через make swagger.
	_ "github.com/sweetlife999/chain-of-trades-avito/docs/swagger"
	adminaudithandler "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/handler"
	adminauditrepository "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/repository"
	adminauditservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminaudit/service"
	admindashboardhandler "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/handler"
	admindashboardrepository "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/repository"
	admindashboardservice "github.com/sweetlife999/chain-of-trades-avito/internal/admindashboard/service"
	adminexchangehandler "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/handler"
	adminexchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/adminexchange/service"
	authhandler "github.com/sweetlife999/chain-of-trades-avito/internal/auth/handler"
	authmiddleware "github.com/sweetlife999/chain-of-trades-avito/internal/auth/middleware"
	authservice "github.com/sweetlife999/chain-of-trades-avito/internal/auth/service"
	authtoken "github.com/sweetlife999/chain-of-trades-avito/internal/auth/token"
	"github.com/sweetlife999/chain-of-trades-avito/internal/config"
	"github.com/sweetlife999/chain-of-trades-avito/internal/database"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	exchangehandler "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/handler"
	exchangerepository "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/repository"
	exchangesearch "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/search"
	exchangeservice "github.com/sweetlife999/chain-of-trades-avito/internal/exchange/service"
	itemhandler "github.com/sweetlife999/chain-of-trades-avito/internal/item/handler"
	itemrepository "github.com/sweetlife999/chain-of-trades-avito/internal/item/repository"
	itemservice "github.com/sweetlife999/chain-of-trades-avito/internal/item/service"
	"github.com/sweetlife999/chain-of-trades-avito/internal/llm"
	notificationhandler "github.com/sweetlife999/chain-of-trades-avito/internal/notification/handler"
	notificationrepository "github.com/sweetlife999/chain-of-trades-avito/internal/notification/repository"
	notificationservice "github.com/sweetlife999/chain-of-trades-avito/internal/notification/service"
	pickuppointhandler "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/handler"
	pickuppointrepository "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/repository"
	pickuppointservice "github.com/sweetlife999/chain-of-trades-avito/internal/pickuppoint/service"
	ratinghandler "github.com/sweetlife999/chain-of-trades-avito/internal/rating/handler"
	ratingrepository "github.com/sweetlife999/chain-of-trades-avito/internal/rating/repository"
	ratingservice "github.com/sweetlife999/chain-of-trades-avito/internal/rating/service"
	reporthandler "github.com/sweetlife999/chain-of-trades-avito/internal/report/handler"
	reportrepository "github.com/sweetlife999/chain-of-trades-avito/internal/report/repository"
	reportservice "github.com/sweetlife999/chain-of-trades-avito/internal/report/service"
	supporthandler "github.com/sweetlife999/chain-of-trades-avito/internal/support/handler"
	supportrepository "github.com/sweetlife999/chain-of-trades-avito/internal/support/repository"
	supportservice "github.com/sweetlife999/chain-of-trades-avito/internal/support/service"
	uploadhandler "github.com/sweetlife999/chain-of-trades-avito/internal/upload/handler"
	uploadservice "github.com/sweetlife999/chain-of-trades-avito/internal/upload/service"
	userhandler "github.com/sweetlife999/chain-of-trades-avito/internal/user/handler"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

const (
	authTokenTTL          = 12 * time.Hour
	searchQueueCapacity   = 256
	serverShutdownTimeout = 10 * time.Second
	workerShutdownTimeout = 35 * time.Second
	llmProbeTimeout       = 3 * time.Second
)

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
	searchQueue, err := exchangesearch.NewQueue(searchQueueCapacity)
	if err != nil {
		log.Fatal(err)
	}
	exchanges := exchangeservice.New(exchangesRepository, searchQueue)
	searchWorker := exchangesearch.NewWorker(searchQueue, exchanges)
	workerCtx, stopWorker := context.WithCancel(context.Background())
	searchWorkerDone := make(chan struct{})
	go func() {
		searchWorker.Run(workerCtx)
		close(searchWorkerDone)
	}()
	exchangesHandler := exchangehandler.New(exchanges)
	items := itemservice.New(itemrepository.New(pool), exchanges)
	pickupPoints := pickuppointservice.New(pickuppointrepository.New(queries))
	adminDashboard := admindashboardservice.New(admindashboardrepository.New(queries))
	adminExchanges := adminexchangeservice.New(usersRepository, exchangesRepository)
	reportsRepository := reportrepository.New(queries)
	reports := reportservice.New(reportsRepository)
	adminReports := reportservice.NewAdmin(reportsRepository, exchangesRepository)
	adminAudit := adminauditservice.New(adminauditrepository.New(queries))
	notifications := notificationservice.New(notificationrepository.New(queries))
	ratings := ratingservice.New(ratingrepository.New(queries))
	supportRepository := supportrepository.New(queries)
	// Модель — не зависимость API, а инструмент одной фичи: пустой OLLAMA_URL отдаёт
	// нулевого бота, и поддержка работает как раньше, только без автоответов.
	supportBot := newSupportBot(cfg, supportRepository)
	supportBotDone := make(chan struct{})
	go func() {
		supportBot.Run(workerCtx)
		close(supportBotDone)
	}()
	support := supportservice.New(supportRepository, supportBot)
	adminSupport := supportservice.NewAdmin(supportRepository)
	// Каталог создаётся здесь же: прав на запись не окажется — упадём на старте, а не на
	// первой загрузке пользователя.
	uploads, err := uploadservice.New(cfg.UploadsDirectory)
	if err != nil {
		log.Fatal(err)
	}

	tokens := authtoken.NewManager(cfg.JWTSecret, authTokenTTL)
	authenticator := authmiddleware.New(tokens, users)
	adminAuthorizer := authmiddleware.NewAdminAuthorizer(users)
	auth := authservice.New(usersRepository, tokens)

	userhandler.New(users).RegisterRoutes(router, authenticator.RequireAuthentication)
	itemhandler.New(items).RegisterRoutes(router, authenticator.RequireAuthentication)
	authhandler.New(auth, cfg.CookieSecure, authTokenTTL).
		RegisterRoutes(router, authenticator.RequireAuthentication)
	exchangesHandler.RegisterRoutes(router, authenticator.RequireAuthentication)
	pickupPointsHandler := pickuppointhandler.New(pickupPoints)
	// Справочник пунктов нужен обычному пользователю, чтобы выбрать, куда нести вещь.
	// Заводить и править их по-прежнему может только администратор — ниже, в /admin.
	pickupPointsHandler.RegisterPublicRoutes(router, authenticator.RequireAuthentication)
	// Жалуется обычный участник обмена, поэтому маршрут живёт вне группы /admin.
	reporthandler.New(reports).RegisterRoutes(router, authenticator.RequireAuthentication)
	notificationhandler.New(notifications).RegisterRoutes(router, authenticator.RequireAuthentication)
	// Оценку ставит участник завершённого обмена, а лента отзывов публична, как профиль,
	// поэтому модуль живёт вне /admin: администратор в оценки не вмешивается.
	ratinghandler.New(ratings).RegisterRoutes(router, authenticator.RequireAuthentication)
	supporthandler.New(support).RegisterRoutes(router, authenticator.RequireAuthentication)
	// Загрузка отделена от объявлений и профиля: сначала файл, потом ссылка на него в любом
	// из них. Иначе multipart пришлось бы тащить в создание и в редактирование вещи разом.
	uploadhandler.New(uploads).RegisterRoutes(router, authenticator.RequireAuthentication)

	// Все следующие административные модули регистрируются только внутри этой группы.
	// JWT сначала определяет пользователя, затем роль проверяется по актуальным данным БД.
	router.Route("/admin", func(adminRouter chi.Router) {
		adminRouter.Use(authenticator.RequireAuthentication)
		adminRouter.Use(adminAuthorizer.RequireAdmin)
		admindashboardhandler.New(adminDashboard).RegisterRoutes(adminRouter)
		adminexchangehandler.New(adminExchanges).RegisterRoutes(adminRouter)
		pickupPointsHandler.RegisterRoutes(adminRouter)
		exchangesHandler.RegisterAdminRoutes(adminRouter)
		reporthandler.NewAdmin(adminReports).RegisterRoutes(adminRouter)
		adminaudithandler.New(adminAudit).RegisterRoutes(adminRouter)
		supporthandler.NewAdmin(adminSupport).RegisterRoutes(adminRouter)
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

	serverErrors := make(chan error, 1)
	go func() {
		err := server.ListenAndServe()
		if err == http.ErrServerClosed {
			err = nil
		}
		serverErrors <- err
	}()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	var serverErr error
	select {
	case <-signalCtx.Done():
		log.Print("shutdown signal received")
	case serverErr = <-serverErrors:
	}
	stopSignals()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), serverShutdownTimeout)
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful HTTP shutdown failed: %v", err)
		_ = server.Close()
	}
	cancelShutdown()

	// Новые HTTP-задачи уже не появятся. Закрытие очередей даёт воркерам дочитать буфер.
	searchQueue.Close()
	supportBot.Close()
	drainWorker("search", searchWorkerDone, stopWorker)
	drainWorker("support bot", supportBotDone, stopWorker)
	stopWorker()

	if serverErr != nil {
		log.Printf("HTTP server stopped with error: %v", serverErr)
	}
}

// newSupportBot собирает автоответчик поддержки и попутно пишет в лог состояние модели.
// Проверка спрашивает у Ollama список моделей и не запускает инференс: греть веса ради
// строчки в логе значило бы занять единственное ядро сервера ровно в тот момент, когда
// API поднимается.
//
// Недоступная модель бота не отменяет: «недоступна» сразу после деплоя — нормальный
// ответ, Ollama в это время может ещё качать веса, а обращения появятся позже. Пустой
// OLLAMA_URL — это выключенная фича, и тогда возвращается nil: методы Bot на нём
// безопасны, поддержка работает без автоответов.
func newSupportBot(cfg config.Config, repository *supportrepository.Repository) *supportservice.Bot {
	if cfg.OllamaURL == "" {
		log.Print("llm: OLLAMA_URL is empty, model-backed features are disabled")
		return nil
	}

	client := llm.New(cfg.OllamaURL, cfg.OllamaModel)

	ctx, cancel := context.WithTimeout(context.Background(), llmProbeTimeout)
	defer cancel()

	if err := client.Available(ctx); err != nil {
		log.Printf("llm: model %s is not ready at startup: %v", cfg.OllamaModel, err)
	} else {
		log.Printf("llm: model %s is ready on %s", cfg.OllamaModel, cfg.OllamaURL)
	}

	// Ник служебного пользователя в коде и в миграции 00024 обязаны совпадать: ответ бота
	// находит автора джойном по нику. Разъехавшись, они не дают ошибки — вставка просто
	// возвращает ноль строк, и бот молчит на каждом обращении, а в логе это неотличимо от
	// «обращение уже не ждёт ответа». Поэтому сверяем один раз на старте и говорим прямо.
	switch exists, err := repository.BotUserExists(ctx, supportservice.BotNickname); {
	case err != nil:
		log.Printf("support bot: cannot check user %q, replies may be silent: %v",
			supportservice.BotNickname, err)
	case !exists:
		log.Printf("support bot is disabled: user %q is missing, apply migrations (00024)",
			supportservice.BotNickname)
		return nil
	}

	return supportservice.NewBot(repository, client)
}

// drainWorker ждёт, пока фоновый воркер дочитает очередь, и снимает его силой, если он
// на это не уложился.
func drainWorker(name string, done <-chan struct{}, stop context.CancelFunc) {
	select {
	case <-done:
	case <-time.After(workerShutdownTimeout):
		log.Printf("%s worker drain timed out, forcing shutdown", name)
		stop()
		<-done
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
