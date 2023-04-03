package poller

import (
	"github.com/datadaodevs/go-service-framework/constants"
	"time"
)

type Config struct {
	Blockchain  constants.Blockchain `env:"blockchain,required"`
	BatchSize   int                  `env:"batch_size" envDefault:"100"`
	ReorgDepth  int                  `env:"reorg_depth" envDefault:"8"`
	HttpRetries int                  `env:"http_retries" envDefault:"10"`
	SleepTime   time.Duration        `env:"poller_sleep_time" envDefault:"12s"`
	Tick        time.Duration        `env:"poller_tick_duration" envDefault:"1s"`
}
