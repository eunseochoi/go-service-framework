package poller

import (
	"context"
	"fmt"
	"github.com/datadaodevs/go-service-framework/constants"
	"github.com/datadaodevs/go-service-framework/retry"
	"github.com/pkg/errors"
	"time"
)

// setModeAndGetCursor uses the delta between local and remote chaintip values to deduce whether poller
// should run in backfill mode, chaintip mode, or sleep mode (if not enough blocks are finalized), then
// returns the current cursor
func (p *Poller) setModeAndGetCursor(ctx context.Context) (uint64, error) {
	currentTip, err := p.getCurrentChaintip(ctx)
	if err != nil {
		return 0, errors.Errorf("Error getting current chaintip: %v", err)
	}

	chainTip, err := p.getRemoteChaintip(ctx)
	if err != nil {
		return 0, errors.Errorf("Error getting remote chaintip: %v", err)
	}

	// Skip rpc calls when blocks are not yet finalized
	if currentTip+uint64(p.cfg.BatchSize)+uint64(p.cfg.ReorgDepth) > chainTip {
		p.setSleepMode()
		return 0, errors.New("Blocks are not yet finalized")
	}

	//	Determine whether we're in range of chain tip and need a smaller batch size
	p.mode = ModeBackfill
	if chainTip-currentTip <= uint64(p.cfg.BatchSize) {
		p.mode = ModeChaintip
	}

	return currentTip, nil
}

// getCurrentChaintip pulls the current local chaintip from cache
func (p *Poller) getCurrentChaintip(ctx context.Context) (uint64, error) {
	blockRedisKey := fmt.Sprintf("%s-%s", p.driver.Blockchain(), constants.BlockKey)
	currentTip, err := p.cache.GetCurrentBlockNumber(ctx, blockRedisKey)
	if err != nil {
		p.logger.Errorf("error thrown getting chain tip from redis: %v", err)
		return 0, err
	}
	return currentTip, nil
}

// setCurrentChaintip overwrites the current cached local chaintip value
func (p *Poller) setCurrentChaintip(ctx context.Context, newTip uint64) error {
	blockRedisKey := fmt.Sprintf("%s-%s", p.driver.Blockchain(), constants.BlockKey)
	return retry.Exec(p.cfg.HttpRetries, func(attempt int) (bool, error) {
		err := p.cache.SetCurrentBlockNumber(ctx, blockRedisKey, newTip)
		if err != nil {
			exp := time.Duration(2 ^ (attempt - 1))
			time.Sleep(exp * time.Second)
		}
		return attempt < p.cfg.HttpRetries, err
	})
}

// getRemoteChaintip pulls the remote chaintip value
func (p *Poller) getRemoteChaintip(ctx context.Context) (uint64, error) {
	var chainTip uint64
	var err error
	err = retry.Exec(p.cfg.HttpRetries, func(attempt int) (bool, error) {
		var err error
		chainTip, err = p.driver.GetChainTipNumber(ctx)
		if err != nil {
			exp := time.Duration(2 ^ (attempt - 1))
			time.Sleep(exp * time.Second)
		}
		return attempt < p.cfg.HttpRetries, err
	})
	return chainTip, err
}

// setSleepMode puts the poller to sleep for a configurable number of seconds, then resets it to
// "ready" mode so the next iteration can freshly assess what mode it should switch to
func (p *Poller) setSleepMode() {
	p.mode = ModeSleep
	go func() {
		time.Sleep(time.Duration(p.cfg.SleepTime) * time.Second)
		p.mode = ModeReady
	}()
}
