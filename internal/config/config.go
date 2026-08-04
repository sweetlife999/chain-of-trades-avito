package config

import (
	"errors"
	"os"
)

const defaultHTTPAddress = ":8080"

type Config struct {
	HTTPAddress string
	DatabaseURL string
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

	return Config{
		HTTPAddress: httpAddress,
		DatabaseURL: databaseURL,
	}, nil
}
