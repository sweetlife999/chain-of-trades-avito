package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sweetlife999/chain-of-trades-avito/internal/config"
	"github.com/sweetlife999/chain-of-trades-avito/internal/database"
	db "github.com/sweetlife999/chain-of-trades-avito/internal/db"
	userhandler "github.com/sweetlife999/chain-of-trades-avito/internal/user/handler"
	userrepository "github.com/sweetlife999/chain-of-trades-avito/internal/user/repository"
	userservice "github.com/sweetlife999/chain-of-trades-avito/internal/user/service"
)

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
	users := userservice.New(userrepository.New(queries))
	userhandler.New(users).RegisterRoutes(router)

	router.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

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
