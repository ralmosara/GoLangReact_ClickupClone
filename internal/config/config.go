package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort                  string
	AppEnv                   string
	DBDSN                    string
	RedisAddr                string
	JWTSecret                string
	StorageDir               string
	MigrationsDir            string
	AutoMigrate              bool
	CredentialsEncryptionKey string // hex-encoded 32-byte AES-256 key
	AttachmentMaxBytes       int64  // upload size cap (bytes); 0 = use default

	// Observability (Phase 1) — every field is optional. The corresponding
	// subsystem is disabled when its primary endpoint/DSN is empty.
	OTLPEndpoint        string
	OTLPInsecure        bool
	OTELSampleRatio     float64
	OTELServiceName     string
	OTELServiceVersion  string
	SentryDSN           string
	SentryRelease       string
	SentrySampleRate    float64

	// Identity (Phase 2) — MFA + OIDC.
	MFAIssuer            string
	GoogleOIDCClientID   string
	GoogleOIDCSecret     string
	GoogleOIDCRedirect   string
	OIDCSuccessRedirect  string
	OIDCFailureRedirect  string
	OIDCCookieSecure     bool
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		AppPort:                  getenv("APP_PORT", "8080"),
		AppEnv:                   getenv("APP_ENV", "development"),
		DBDSN:                    getenv("DB_DSN", ""),
		RedisAddr:                getenv("REDIS_ADDR", "localhost:6379"),
		JWTSecret:                getenv("JWT_SECRET", "dev-secret"),
		StorageDir:               getenv("STORAGE_DIR", "./storage"),
		MigrationsDir:            getenv("MIGRATIONS_DIR", "./migrations"),
		AutoMigrate:              getbool("AUTO_MIGRATE", true),
		CredentialsEncryptionKey: getenv("CREDENTIALS_ENCRYPTION_KEY", ""),
		AttachmentMaxBytes:       getint64("ATTACHMENT_MAX_BYTES", 25<<20),

		OTLPEndpoint:        getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
		OTLPInsecure:        getbool("OTEL_EXPORTER_OTLP_INSECURE", true),
		OTELSampleRatio:     getfloat("OTEL_TRACES_SAMPLE_RATIO", 0.1),
		OTELServiceName:     getenv("OTEL_SERVICE_NAME", "clickup-clone"),
		OTELServiceVersion:  getenv("OTEL_SERVICE_VERSION", "dev"),
		SentryDSN:           getenv("SENTRY_DSN", ""),
		SentryRelease:       getenv("SENTRY_RELEASE", "dev"),
		SentrySampleRate:    getfloat("SENTRY_SAMPLE_RATE", 1.0),

		MFAIssuer:           getenv("MFA_ISSUER", "ClickUp Clone"),
		GoogleOIDCClientID:  getenv("GOOGLE_OIDC_CLIENT_ID", ""),
		GoogleOIDCSecret:    getenv("GOOGLE_OIDC_CLIENT_SECRET", ""),
		GoogleOIDCRedirect:  getenv("GOOGLE_OIDC_REDIRECT_URL", ""),
		OIDCSuccessRedirect: getenv("OIDC_SUCCESS_REDIRECT", ""),
		OIDCFailureRedirect: getenv("OIDC_FAILURE_REDIRECT", ""),
		OIDCCookieSecure:    getbool("OIDC_COOKIE_SECURE", false),
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

func getint64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getfloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}
