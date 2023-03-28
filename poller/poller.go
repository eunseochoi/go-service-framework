package poller

import (
	"context"
	"github.com/datadaodevs/go-service-framework/pool"
	"github.com/datadaodevs/go-service-framework/util"
	"sync"
	"time"
)

// Modes determine what the poller does on each iteration of its main routine's loop; these are determined by
// the distance of the cursor from chaintip and the success/failure state of the previous iteration
const (
	ModeReady = iota
	ModeSleep
	ModeBackfill
	ModeChaintip
)

// Poller is a chain-agnostic module for ETLing blockchain data, utilizing worker pools to optimize
// efficiency and speed
type Poller struct {
	logger         util.Logger
	metrics        util.Metrics
	cfg            *PollerConfig
	mode           int
	driver         Driver
	cache          Cache
	fetchPool      *pool.WorkerPool
	accumulatePool *pool.WorkerPool
	writePool      *pool.WorkerPool
}

// New constructs a new poller, given a config, a chain-specific driver, and a variadic array of options
func New(cfg *PollerConfig, driver Driver, opts ...opt) *Poller {
	p := Poller{cfg: cfg, driver: driver}
	for _, opt := range opts {
		opt(&p)
	}

	p.accumulatePool.SetInputFeed(p.fetchPool.Results(), p.driver.Accumulate)
	p.writePool.SetInputFeed(p.accumulatePool.Results(), p.driver.Writers()...)

	return &p
}

// Run executes the main program loop inside of a dedicated goroutine; the loop can be terminated from the
// outside via context cancellation
func (p *Poller) Run(ctx context.Context) error {
	go func() {
		for {
			//	Check for context cancellation
			select {
			default:
			case <-ctx.Done():
				p.logger.Warn("Context cancelled; poller will now stop")
				return
			}

			//	Deduce mode and get "cursor" representing the current local chaintip
			start := time.Now()
			cursor, err := p.setModeAndGetCursor(ctx)
			if err != nil {
				p.logger.Errorf("Error setting mode: %v", err)
				continue
			}

			switch p.mode {
			case ModeSleep:
				//	If in "sleep" mode, hold for 1 second then start another iteration of the main loop
				p.logger.Info("Sleep mode detected; sleeping for this cycle")
				time.Sleep(1 * time.Second)
				continue
			case ModeBackfill:
				//	If in ""backfill" mode, consume a batch of blocks and update the cursor
				p.logger.Infof("Batch mode: start polling at block %d with batch size %d", cursor, p.cfg.BatchSize)
				wg := sync.WaitGroup{}
				startIndex := cursor
				for i := 0; i < p.cfg.BatchSize; i++ {
					wg.Add(1)
					p.fetchPool.PushGroup(p.driver.FetchSequence(startIndex+uint64(i)), &wg)
				}
				wg.Wait()
				cursor = startIndex + uint64(p.cfg.BatchSize)
			case ModeChaintip:
				//	If in "chaintip" mode, pull the latest block, validate it, then consume it
				p.logger.Infof("Chaintip mode: pulling block %d", cursor)
				if p.driver.IsValidBlock(ctx, cursor); err != nil {
					p.logger.Errorf("Invalid block (possible reorg detected) - %v", err)
					//	Sleep for N seconds if invalid block is detected
					p.setSleepMode()
					continue
				}
				wg := sync.WaitGroup{}
				p.fetchPool.PushGroup(p.driver.FetchSequence(cursor), &wg)
				wg.Wait()
				cursor++
			}

			//	Cache new cursor value
			if err := p.setCurrentChaintip(ctx, cursor); err != nil {
				p.logger.Errorf("failed to update block chain tip within redis: %v", err)
				continue
			}

			//	Log/stat update
			p.logger.Infof("finished polling at block %d with batch size %d", cursor, p.cfg.BatchSize)
			p.metrics.Gauge("keep_up_with_chain_tip", float64(time.Since(start).Milliseconds()), []string{}, 1.0)
		}
	}()

	return nil
}

func (p *Poller) Mode() int {
	return p.mode
}
