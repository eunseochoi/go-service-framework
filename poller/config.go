package poller

import "github.com/datadaodevs/go-service-framework/constants"

type Config struct {
	Blockchain  constants.Blockchain `env:"blockchain,required"`
	BatchSize   int                  `env:"batch_size" envDefault:"100"`
	ReorgDepth  int                  `env:"reorg_depth" envDefault:"8"`
	HttpRetries int                  `env:"http_retries" envDefault:"10"`
	SleepTime   int                  `env:"poller_sleep_time" envDefault:"1"`
}
