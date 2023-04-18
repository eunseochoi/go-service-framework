package contract_poller

import "context"

func (p *Poller) Insights() map[string]map[string]int {
	return map[string]map[string]int{
		"fetch-address-pool": p.getAddressPool.Insights(),
		"fetch-pool":         p.fetchPool.Insights(),
		"accumulate-pool":    p.accumulatePool.Insights(),
		"write-pool":         p.writePool.Insights(),
	}
}

func (p *Poller) Pause() {
	p.modeMu.Lock()
	defer p.modeMu.Unlock()

	p.writePool.FlushAndRestart()
	p.accumulatePool.FlushAndRestart()
	p.fetchPool.FlushAndRestart()
	p.getAddressPool.FlushAndRestart()
	p.mode = ModePaused
}

func (p *Poller) Resume() {
	p.modeMu.Lock()
	defer p.modeMu.Unlock()

	p.mode = ModeReady
}

func (p *Poller) SetCursor(ctx context.Context, newVal uint64) error {
	return p.cache.SetCurrentBlockNumber(ctx, p.cacheKey(), newVal)
}
