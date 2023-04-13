package manager

import (
	"github.com/coherentopensource/go-service-framework/metrics"
	"github.com/coherentopensource/go-service-framework/util"
	"log"
)

func mustInitMetrics(mgrCfg *Config) util.Metrics {
	var met util.Metrics
	var err error
	cfg := metrics.Config{
		DatadogIP:   mgrCfg.DatadogIP,
		DatadogPort: mgrCfg.DatadogPort,
		AppName:     mgrCfg.AppName,
		Env:         string(mgrCfg.Env),
	}
	switch {
	case mgrCfg.Env == EnvLocal || cfg.Env == EnvTest:
		met, err = metrics.NewNoopMetrics()
		if err != nil {
			log.Fatalf("Failed to instantiate metrics: %v", err)
		}
	default:
		met, err = metrics.NewMetrics(&cfg)
		if err != nil {
			log.Fatalf("Failed to instantiate metrics: %v", err)
		}
	}
	return met
}
