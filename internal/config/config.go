package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr string
	DBPath   string

	KafkaBootstrap     string
	KafkaTopic         string
	KafkaConsumerGroup string

	ConsulEnabled      bool
	ConsulHTTPAddr     string
	ServiceName        string
	ServiceID          string
	ServiceAddress     string
	ServicePort        int
	ServiceHealthCheck string
	ShutdownTimeout    time.Duration

	VaultEnabled         bool
	VaultAddr            string
	VaultToken           string
	VaultTokenSecretPath string
	AppTokenSecret       string
}

func Load() Config {
	servicePort := envInt("SERVICE_PORT", 8000)

	return Config{
		HTTPAddr: ":" + strconv.Itoa(servicePort),
		DBPath:   env("DB_PATH", "app_data/auth.db"),

		KafkaBootstrap:     env("KAFKA_BOOTSTRAP", "localhost:9092"),
		KafkaTopic:         env("KAFKA_TOPIC", "user_events"),
		KafkaConsumerGroup: env("KAFKA_CONSUMER_GROUP", "fastapi-auth-group"),

		ConsulEnabled:      envBool("CONSUL_ENABLED", true),
		ConsulHTTPAddr:     env("CONSUL_HTTP_ADDR", "http://localhost:8500"),
		ServiceName:        env("SERVICE_NAME", "fastapi-auth"),
		ServiceID:          env("SERVICE_ID", "fastapi-auth-8000"),
		ServiceAddress:     env("SERVICE_ADDRESS", "127.0.0.1"),
		ServicePort:        servicePort,
		ServiceHealthCheck: env("SERVICE_HEALTH_CHECK_URL", "http://host.docker.internal:8000/health"),
		ShutdownTimeout:    5 * time.Second,

		VaultEnabled:         envBool("VAULT_ENABLED", true),
		VaultAddr:            env("VAULT_ADDR", "http://localhost:8200"),
		VaultToken:           os.Getenv("VAULT_TOKEN"),
		VaultTokenSecretPath: env("VAULT_TOKEN_SECRET_PATH", "secret/data/fastapi-auth"),
		AppTokenSecret:       os.Getenv("APP_TOKEN_SECRET"),
	}
}

func env(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
