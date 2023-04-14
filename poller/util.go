package poller

func (p *Poller) cacheKey() string {
	return p.cursorKey
}

func (p *Poller) driverTaskLoad() int {
	//	count of fetchers + count of accumulators (always 1) + count of writers == number of queued jobs per block
	return len(p.driver.FetchSequence(0)) + 1 + len(p.driver.Writers())
}

func modeToString(mode int) string {
	out := "unknown"
	switch mode {
	case ModePaused:
		out = "paused"
	case ModeSleep:
		out = "sleep"
	case ModeReady:
		out = "ready"
	case ModeBackfill:
		out = "backfill"
	case ModeChaintip:
		out = "chaintip"
	}
	return out
}
