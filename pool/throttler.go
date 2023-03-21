package pool

import (
	"context"
	"errors"
	"time"
)

// Throttler is a helper struct for controlling the pace of work in the WorkerPool
type Throttler struct {
	chs     []chan struct{}
	period  time.Duration
	counter int
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewThrottler constructs a new throttler, given a bandwidth and a time duration
func NewThrottler(bandwidth int, period time.Duration) *Throttler {
	chs := make([]chan struct{}, bandwidth)
	for i := 0; i < bandwidth; i++ {
		chs[i] = make(chan struct{}, 1)
	}
	return &Throttler{
		period:  period,
		counter: 0,
		chs:     chs,
	}
}

// Start kicks off processes need to maintain throttler
func (t *Throttler) Start(parentCtx context.Context) error {
	ctx, cancel := context.WithCancel(parentCtx)
	t.cancel = cancel
	t.ctx = ctx

	for i := 0; i < len(t.chs); i++ {
		go func(i int) {
			tick := time.NewTicker(t.period)
			first := make(chan struct{}, 1)
			first <- struct{}{}
			for {
				select {
				case <-ctx.Done():
					return
				case <-first:
					t.chs[i] <- struct{}{}
				case <-tick.C:
					t.chs[i] <- struct{}{}
				}
			}
		}(i)
	}
	return nil
}

// WaitForGo blocks until the throttle buffer is empty, given the bandwidth and duration constraints
func (t *Throttler) WaitForGo() error {
	select {
	case <-t.chs[t.rr()]:
	case <-t.ctx.Done():
		return errors.New("Context cancelled")
	}
	return nil
}

// Stop gracefully shuts down the throttler
func (t *Throttler) Stop() {
	t.cancel()
}

// rr rotates between throttle channels via round-robin
func (t *Throttler) rr() int {
	oldCounter := t.counter
	t.counter = t.counter + 1
	return oldCounter % len(t.chs)
}
