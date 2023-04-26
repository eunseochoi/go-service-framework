package contract_poller

import (
	"github.com/coherentopensource/go-service-framework/pool"
	"github.com/coherentopensource/go-service-framework/util"
)

type opt func(p *Poller)

func WithFetchPool(wp *pool.WorkerPool) opt {
	return func(p *Poller) {
		p.fetchPool = wp
	}
}

func WithAddressFetchPool(wp *pool.WorkerPool) opt {
	return func(p *Poller) {
		p.getAddressPool = wp
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
func WithLogger(logger util.Logger) opt {
	return func(p *Poller) {
		p.logger = logger
	}
}
func WithMetrics(metrics util.Metrics) opt {
	return func(p *Poller) {
		p.metrics = metrics
	}
}
