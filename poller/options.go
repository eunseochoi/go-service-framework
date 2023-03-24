package poller

import (
	"github.com/datadaodevs/go-service-framework/pool"
	"github.com/datadaodevs/go-service-framework/utils"
)

type opt func(p *Poller)

func WithFetchPool(wp *pool.WorkerPool) opt {
	return func(p *Poller) {
		p.fetchPool = wp
	}
}
func WithAccumulatePool(wp *pool.WorkerPool) opt {
	return func(p *Poller) {
		p.accumulatePool = wp
	}
}
func WithWritePool(wp *pool.WorkerPool) opt {
	return func(p *Poller) {
		p.writePool = wp
	}
}
func WithCache(c Cache) opt {
	return func(p *Poller) {
		p.cache = c
	}
}
func WithLogger(logger utils.Logger) opt {
	return func(p *Poller) {
		p.logger = logger
	}
}
func WithMetrics(metrics utils.Metrics) opt {
	return func(p *Poller) {
		p.metrics = metrics
	}
}
