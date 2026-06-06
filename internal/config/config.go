package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port             string
	AllowedOrigins   string
	JWTSecret        string
	JWTTTLHours      int
	MagicLinkTTLMin  int
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPass         string
	FromEmail        string
	DatabaseURL      string
	VAPIDPublicKey   string
	VAPIDPrivateKey  string
	VAPIDEmail       string
}

func Load() *Config {
	return &Config{
		Port:            getEnv("PORT", "8080"),
		AllowedOrigins:  getEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		JWTSecret:       mustEnv("JWT_SECRET"),
		JWTTTLHours:     getEnvInt("JWT_TTL_HOURS", 24),
		MagicLinkTTLMin: getEnvInt("MAGIC_LINK_TTL_MIN", 15),
		SMTPHost:        getEnv("SMTP_HOST", "smtp.resend.com"),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUser:        getEnv("SMTP_USER", "resend"),
		SMTPPass:        getEnv("SMTP_PASS", ""),
		FromEmail:       getEnv("FROM_EMAIL", "noreply@rize.app"),
		DatabaseURL:     mustEnv("DATABASE_URL"),
		VAPIDPublicKey:  getEnv("VAPID_PUBLIC_KEY", ""),
		VAPIDPrivateKey: getEnv("VAPID_PRIVATE_KEY", ""),
		VAPIDEmail:      getEnv("VAPID_EMAIL", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("missing required env var: " + key)
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
