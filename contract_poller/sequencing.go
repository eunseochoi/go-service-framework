package contract_poller

import (
	"context"
	"github.com/coherentopensource/go-service-framework/retry"
	"github.com/pkg/errors"
	"time"
)

// setModeAndGetCursor uses the delta between local and remote chaintip values to deduce whether poller
// should run in backfill mode, chaintip mode, or sleep mode (if not enough blocks are finalized), then
// returns the current cursor
func (p *Poller) setModeAndGetCursor(ctx context.Context) (uint64, error) {
	p.modeMu.Lock()
	defer p.modeMu.Unlock()

	cursor, err := p.getCurrentChaintip(ctx)
	if err != nil {
		return 0, errors.Errorf("Error getting current chaintip: %v", err)
	}

	if p.mode == ModePaused || p.mode == ModeSleep {
		return cursor, nil
	}

	chainTip, err := p.getRemoteChaintip(ctx)
	if err != nil {
		return 0, errors.Errorf("Error getting remote chaintip: %v", err)
	}

	maxBlock := chainTip - uint64(p.cfg.ReorgDepth)
	distanceToMaxBlock := maxBlock - cursor

	switch {
	//	Cursor is within reorg
	case cursor >= maxBlock:
		p.setSleepMode()
		p.logger.Warn("Cursor is within reorg range; poller going to sleep")
	// Cursor is close enough that we should be in Chaintip mode
	case distanceToMaxBlock < uint64(p.cfg.BatchSize):
		p.mode = ModeChaintip
	//	Cursor is distant enough from chaintip that we can pull batches
	default:
		p.mode = ModeBackfill
	}

	return cursor, nil
}

// getCurrentChaintip pulls the current local chaintip from cache
func (p *Poller) getCurrentChaintip(ctx context.Context) (uint64, error) {
	currentTip, err := p.cache.GetCurrentBlockNumber(ctx, p.cacheKey())
	if err != nil {
		p.logger.Errorf("error thrown getting chain tip from redis: %v", err)
		return 0, err
	}
	return currentTip, nil
}

// setCurrentChaintip overwrites the current cached local chaintip value
func (p *Poller) setCurrentChaintip(ctx context.Context, newTip uint64) error {
	return retry.Exec(p.cfg.HttpRetries, func() error {
		return p.cache.SetCurrentBlockNumber(ctx, p.cacheKey(), newTip)
	}, nil)
}

// getRemoteChaintip pulls the remote chaintip value
func (p *Poller) getRemoteChaintip(ctx context.Context) (uint64, error) {
	var chainTip uint64
	var err error
	retry.Exec(p.cfg.HttpRetries, func() error {
		chainTip, err = p.driver.GetChainTipNumber(ctx)
		if err != nil {
			return err
		}
		return nil
	}, nil)
	return chainTip, err
}

// setSleepMode puts the poller to sleep for a configurable number of seconds, then resets it to
// "ready" mode so the next iteration can freshly assess what mode it should switch to
func (p *Poller) setSleepMode() {
	p.mode = ModeSleep
	go func() {
		select {
		case <-p.runCtx.Done():
			return
		case <-time.Tick(p.cfg.SleepTime):
			p.mode = ModeReady
		}
	}()
}
