package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

const defaultHTTPAddress = ":8080"

type Config struct {
	HTTPAddress  string
	DatabaseURL  string
	JWTSecret    string
	CookieSecure bool
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

	return Config{
		HTTPAddress:  httpAddress,
		DatabaseURL:  databaseURL,
		JWTSecret:    jwtSecret,
		CookieSecure: cookieSecure,
	}, nil
}
