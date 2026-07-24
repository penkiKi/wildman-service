package config

import (
	"log/slog"
	"net"
	"os"
	"strings"
)

type Config struct {
	Host            string
	Port            string
	Environment     string
	Log             string
	AllowedOrigins  []string
	ProviderContact string
	AcoustIDAPIKey  string
	DatabaseURL     string
}

func Load() Config {
	environment := env("WILDMAN_ENV", "development")
	allowedOrigins := ""
	if environment != "production" {
		allowedOrigins = "http://localhost:5173,http://127.0.0.1:5173,http://[::1]:5173"
	}
	return Config{
		Host:            env("WILDMAN_HOST", "0.0.0.0"),
		Port:            env("WILDMAN_PORT", "8080"),
		Environment:     environment,
		Log:             env("WILDMAN_LOG_LEVEL", "info"),
		AllowedOrigins:  splitCSV(env("WILDMAN_ALLOWED_ORIGINS", allowedOrigins)),
		ProviderContact: env("WILDMAN_PROVIDER_CONTACT", ""),
		AcoustIDAPIKey:  env("WILDMAN_ACOUSTID_API_KEY", ""),
		DatabaseURL:     env("WILDMAN_DATABASE_URL", ""),
	}
}

func (c Config) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func (c Config) LogLevel() slog.Level {
	switch strings.ToLower(c.Log) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	values := make([]string, 0)
	for item := range strings.SplitSeq(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, strings.TrimSuffix(item, "/"))
		}
	}
	return values
}
