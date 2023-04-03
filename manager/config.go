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
	AppName     string      `env:"app,required"`
	Env         Environment `env:"env" envDefault:"local"`
	DatadogIP   string      `env:"datadog_ip"`
	DatadogPort string      `env:"datadog_port"`
}

func mustParseConfig() *Config {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("Failed to parse manager config: %v", err)
	}
	return &cfg
}
