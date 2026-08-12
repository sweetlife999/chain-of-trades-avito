package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const (
	defaultHTTPAddress = ":8080"
	// Каталог рядом с процессом: в разработке это ./uploads в корне репозитория, в
	// контейнере — том, который задаёт docker-compose.
	defaultUploadsDirectory = "./uploads"
	// Единственный размер, влезающий в память прод-сервера вместе с Postgres, Go и
	// Caddy. Обоснование — docs/llm.md.
	defaultOllamaModel = "qwen2.5:0.5b"
)

type Config struct {
	HTTPAddress      string
	DatabaseURL      string
	JWTSecret        string
	CookieSecure     bool
	UploadsDirectory string
	OllamaURL        string
	OllamaModel      string
}

func Load() (Config, error) {
	httpAddress := os.Getenv("HTTP_ADDR")
	if httpAddress == "" {
		httpAddress = defaultHTTPAddress
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 characters")
	}

	cookieSecure := false
	if value := os.Getenv("COOKIE_SECURE"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return Config{}, fmt.Errorf("parse COOKIE_SECURE: %w", err)
		}
		cookieSecure = parsed
	}

	uploadsDirectory := os.Getenv("UPLOADS_DIR")
	if uploadsDirectory == "" {
		uploadsDirectory = defaultUploadsDirectory
	}

	// Единственная переменная, пустое значение которой не ошибка, а осознанное
	// «модели нет». Сервис обязан подниматься без Ollama: она нужна отдельным
	// фичам, а не самому API.
	ollamaURL := os.Getenv("OLLAMA_URL")

	ollamaModel := os.Getenv("OLLAMA_MODEL")
	if ollamaModel == "" {
		ollamaModel = defaultOllamaModel
	}

	return Config{
		HTTPAddress:      httpAddress,
		DatabaseURL:      databaseURL,
		JWTSecret:        jwtSecret,
		CookieSecure:     cookieSecure,
		UploadsDirectory: uploadsDirectory,
		OllamaURL:        ollamaURL,
		OllamaModel:      ollamaModel,
	}, nil
}
