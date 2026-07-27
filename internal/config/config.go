package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

const minJWTSecretBytes = 32

type Config struct {
	DatabaseURL string
	RedisURL    string
	RabbitMQURL string

	S3InternalEndpoint string
	S3PublicEndpoint   string
	S3AccessKey        string
	S3SecretKey        string
	S3Bucket           string
	S3UseSSL           bool

	SMTPHost string
	SMTPPort string

	JWTSecret     string
	PresignExpiry time.Duration
	RetentionDays int
}

func Load() (*Config, error) {
	var missing []string
	req := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	c := &Config{
		DatabaseURL:        req("DATABASE_URL"),
		RedisURL:           req("REDIS_URL"),
		RabbitMQURL:        req("RABBITMQ_URL"),
		S3InternalEndpoint: req("S3_INTERNAL_ENDPOINT"),
		S3PublicEndpoint:   req("S3_PUBLIC_ENDPOINT"),
		S3AccessKey:        req("S3_ACCESS_KEY"),
		S3SecretKey:        req("S3_SECRET_KEY"),
		S3Bucket:           req("S3_BUCKET"),
		S3UseSSL:           getBool("S3_USE_SSL", false),
		SMTPHost:           getEnv("SMTP_HOST", "mailpit"),
		SMTPPort:           getEnv("SMTP_PORT", "1025"),
		JWTSecret:          req("JWT_SECRET"),
		PresignExpiry:      getDuration("PRESIGN_EXPIRY", 15*time.Minute),
		RetentionDays:      getInt("RETENTION_DAYS", 7),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("variáveis de ambiente obrigatórias ausentes: %v", missing)
	}
	if len(c.JWTSecret) < minJWTSecretBytes {
		return nil, fmt.Errorf("JWT_SECRET deve ter ao menos %d bytes", minJWTSecretBytes)
	}
	return c, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
