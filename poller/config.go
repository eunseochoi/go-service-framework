package poller

type PollerConfig struct {
	BatchSize   int    `env:"BATCH_SIZE"`
	ReorgDepth  int    `env:"REORG_DEPTH"`
	HttpRetries int    `env:"HTTP_RETRIES"`
	Blockchain  string `env:"BLOCKCHAIN"`
	SleepTime   int    `env:"SLEEP_TIME" envDefault:"1"`
}
