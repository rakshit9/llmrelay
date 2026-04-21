package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port          string
	GatewayAPIKey string // key clients must send to use this proxy
	OpenAIAPIKey  string // key we use to call OpenAI upstream
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		GatewayAPIKey: os.Getenv("GATEWAY_API_KEY"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
	}

	if cfg.GatewayAPIKey == "" {
		return nil, fmt.Errorf("GATEWAY_API_KEY is required")
	}
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
