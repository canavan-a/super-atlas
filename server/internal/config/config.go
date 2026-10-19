package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabasePath     string
	JWTAccessSecret  []byte
	JWTRefreshSecret []byte
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	SMTPEmail        string
	SMTPAppPassword  string
}

func Load() (*Config, error) {
	// .env is optional; real deployments set env vars directly.
	_ = godotenv.Load()

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "open-location.db"
	}
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:             port,
		DatabasePath:     dbPath,
		JWTAccessSecret:  []byte(accessSecret),
		JWTRefreshSecret: []byte(refreshSecret),
		AccessTokenTTL:   15 * time.Minute,
		RefreshTokenTTL:  30 * 24 * time.Hour,
		SMTPEmail:        os.Getenv("SMTP_EMAIL"),
		SMTPAppPassword:  os.Getenv("SMTP_APP_PASSWORD"),
	}, nil
}
