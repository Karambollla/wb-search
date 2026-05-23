package config

import (
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	HTTPAddr       string        `env:"HTTP_ADDR" env-default:":8080"`
	GRPCAddr       string        `env:"GRPC_ADDR" env-default:":9090"`
	NATSURL        string        `env:"NATS_URL" env-default:"nats://localhost:4222"`
	NATSSubject    string        `env:"NATS_SUBJECT" env-default:"search.events"`
	Window         time.Duration `env:"WINDOW" env-default:"5m"`
	BucketDuration time.Duration `env:"BUCKET_DURATION" env-default:"5s"`
	RefreshEvery   time.Duration `env:"REFRESH_EVERY" env-default:"2s"`
	FraudTTL       time.Duration `env:"FRAUD_TTL" env-default:"1m"`
	MaxTopSize     int           `env:"MAX_TOP_SIZE" env-default:"100"`
	StopWordsRaw   string        `env:"STOP_WORDS"`
	StopWords      []string      `env:"-"`
}

func Load() Config {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		panic(err)
	}
	cfg.StopWords = splitCSV(cfg.StopWordsRaw)
	return cfg
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
