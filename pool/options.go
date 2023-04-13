package pool

import "github.com/coherentopensource/go-service-framework/util"

type opt func(wp *WorkerPool)

// WithOutputChannel configures the WorkerPool to push results to a channel for external consumption
func WithOutputChannel() opt {
	return func(wp *WorkerPool) {
		wp.useOutputCh = true
	}
}

// WithThrottler specifies a throttler for controlling workload
func WithThrottler(tt *Throttler) opt {
	return func(wp *WorkerPool) {
		wp.throttler = tt
	}
}

// WithBandwidth overrides the default bandwidth value
func WithBandwidth(bandwidth int) opt {
	return func(wp *WorkerPool) {
		wp.bandwidth = bandwidth
	}
}

// WithLogger overrides the default logger
func WithLogger(logger util.Logger) opt {
	return func(wp *WorkerPool) {
		wp.logger = logger
	}
}
