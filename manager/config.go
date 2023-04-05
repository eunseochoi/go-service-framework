package manager

import (
	"github.com/caarlos0/env/v7"
	"log"
)

type Environment string

const (
	EnvLocal       Environment = "local"
	EnvTest                    = "test"
	EnvDevelopment             = "development"
	EnvProduction              = "production"
)

type Config struct {
	AppName     string      `env:"APP"`
	Env         Environment `env:"ENV" envDefault:"local"`
	DatadogIP   string      `env:"DATADOG_IP"`
	DatadogPort string      `env:"DATADOG_PORT"`
}

func mustParseConfig() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse manager config: %v", err)
	}
	return &cfg
}
