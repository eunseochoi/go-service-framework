package pool

import (
	"context"
	"sync"
	"time"
)

type Throttler struct {
	burst      int
	duration   time.Duration
	ticker     *time.Ticker
	tokens     int
	tokenMu    *sync.Mutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewThrottler(burst int, duration time.Duration) *Throttler {
	return &Throttler{
		burst:    burst,
		duration: duration,
		tokenMu:  &sync.Mutex{},
	}
}

func (t *Throttler) Start(ctx context.Context) error {
	t.ctx, t.cancelFunc = context.WithCancel(ctx)
	t.tokens = t.burst
	t.ticker = time.NewTicker(t.duration)

	go func() {
		for {
			select {
			case <-t.ctx.Done():
				t.ticker.Stop()
				return
			case <-t.ticker.C:
				t.tokenMu.Lock()
				t.tokens = t.burst
				t.tokenMu.Unlock()
			}
		}
	}()

	return nil
}

func (t *Throttler) Stop() {
	t.cancelFunc()
}

func (t *Throttler) WaitForGo() {
	for {
		t.tokenMu.Lock()
		if t.tokens > 0 {
			t.tokens--
			t.tokenMu.Unlock()
			break
		}
		t.tokenMu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}
