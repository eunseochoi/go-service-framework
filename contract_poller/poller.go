package contract_poller

import (
	"context"
	"github.com/coherentopensource/go-service-framework/pool"
	"github.com/coherentopensource/go-service-framework/util"
	"sync"
	"time"
)

// Modes determine what the poller does on each iteration of its main routine's loop; these are determined by
// the distance of the cursor from chaintip and the success/failure state of the previous iteration
const (
	//	ModeReady means the poller is ready to have its mode reassessed based on chaintip position
	ModeReady = iota
	//	ModeSleep means the poller is within reorg depth and is waiting to chaintip to progress before reassessing
	ModeSleep
	//	ModePaused means the poller has been manually paused - it will stay in this state until manually resumed
	ModePaused
	//	ModeBackfill means the poller is >= batchSize blocks from chaintip and will pull a batch of past blocks
	ModeBackfill
	//	ModeChaintip means the poller is < batchSize blocks from chaintip and will pull one block at a time
	ModeChaintip
)

// Poller is a chain-agnostic module for ETLing blockchain data, utilizing worker pools to optimize
// efficiency and speed
type Poller struct {
	modeMu         *sync.Mutex
	logger         util.Logger
	metrics        util.Metrics
	cfg            *Config
	mode           int
	driver         Driver
	cache          Cache
	fetchPool      *pool.WorkerPool
	getAddressPool *pool.WorkerPool
	accumulatePool *pool.WorkerPool
	writePool      *pool.WorkerPool
	cancelFunc     context.CancelFunc
	runCtx         context.Context
}

// New constructs a new poller, given a config, a chain-specific driver, and a variadic array of options
func New(cfg *Config, driver Driver, opts ...opt) *Poller {
	startMode := ModePaused
	if cfg.AutoStart {
		startMode = ModeReady
	}

	p := Poller{
		cfg:    cfg,
		driver: driver,
		modeMu: &sync.Mutex{},
		mode:   startMode,
	}
	for _, opt := range opts {
		opt(&p)
	}
	p.fetchPool.SetGroupInputFeed(p.getAddressPool.Results(), p.driver.Fetchers())
	p.accumulatePool.SetInputFeed(p.fetchPool.Results(), p.driver.Accumulate)
	p.writePool.SetInputFeed(p.accumulatePool.Results(), p.driver.Writers()...)

	return &p
}

// Run executes the main program loop inside of a dedicated goroutine; the loop can be terminated from the
// outside via context cancellation
func (p *Poller) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	p.runCtx = ctx
	p.cancelFunc = cancel

	p.logger.Infof("Contract poller worker starting for blockchain [%s]", p.driver.Blockchain())
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

			p.logger.Infof("Top of main loop; mode is [%s]; cursor is [%d]", modeToString(p.mode), cursor)

			switch p.mode {
			case ModePaused:
				p.logger.Info("Paused mode detected; sleeping for this cycle")
				time.Sleep(1 * time.Second)
				continue
			case ModeSleep:
				//	If in "sleep" mode, hold for 1 second then start another iteration of the main loop
				p.logger.Info("Sleep mode detected; sleeping for this cycle")
				time.Sleep(1 * time.Second)
				continue
			case ModeBackfill:
				//	If in ""backfill" mode, consume a batch of blocks and update the cursor
				p.logger.Infof("Batch mode: start polling contracts at block %d with batch size %d", cursor, p.cfg.BatchSize)
				wg := sync.WaitGroup{}
				startIndex := cursor
				for i := 0; i < p.cfg.BatchSize; i++ {
					wg.Add(p.driverTaskLoad())
					p.getAddressPool.PushGroup(p.driver.FetchSequence(cursor), &wg)
				}
				wg.Wait()
				cursor = startIndex + uint64(p.cfg.BatchSize)
			case ModeChaintip:
				//	If in "chaintip" mode, pull the latest block, validate it, then consume it
				p.logger.Infof("Chaintip mode: pulling block %d", cursor)
				wg := sync.WaitGroup{}
				wg.Add(p.driverTaskLoad())
				p.getAddressPool.PushGroup(p.driver.FetchSequence(cursor), &wg)
				wg.Wait()
				cursor++
			}

			//	Cache new cursor value
			if err := p.setCurrentChaintip(ctx, cursor); err != nil {
				p.logger.Errorf("failed to update block chain tip within redis: %v", err)
				continue
			}

			//	Log/stat update
			p.logger.Infof("finished polling contracts at block %d with batch size %d", cursor, p.cfg.BatchSize)
			p.metrics.Gauge("keep_up_with_chain_tip", float64(time.Since(start).Milliseconds()), []string{}, 1.0)
		}
	}()

	return nil
}

func (p *Poller) Stop() {
	p.cancelFunc()
}

func (p *Poller) Mode() int {
	return p.mode
}
