package config

import (
	"log"
	"os"
)

type Config struct {
	DB     DBConfig
	Server ServerConfig
}

type DBConfig struct {
	URL string
}

type ServerConfig struct {
	Port    string
	BaseURL string
}

func Load() Config {
	return Config{
		DB: DBConfig{
			URL: mustEnv("DATABASE_URL"),
		},
		Server: ServerConfig{
			Port:    env("PORT", "8080"),
			BaseURL: env("BASE_URL", "http://localhost:8080"),
		},
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
