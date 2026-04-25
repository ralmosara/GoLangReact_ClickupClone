package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort       string
	AppEnv        string
	DBDSN         string
	RedisAddr     string
	JWTSecret     string
	StorageDir    string
	MigrationsDir string
	AutoMigrate   bool
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		AppPort:       getenv("APP_PORT", "8080"),
		AppEnv:        getenv("APP_ENV", "development"),
		DBDSN:         getenv("DB_DSN", ""),
		RedisAddr:     getenv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:     getenv("JWT_SECRET", "dev-secret"),
		StorageDir:    getenv("STORAGE_DIR", "./storage"),
		MigrationsDir: getenv("MIGRATIONS_DIR", "./migrations"),
		AutoMigrate:   getbool("AUTO_MIGRATE", true),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getbool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}
