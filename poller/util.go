package poller

func modeToString(mode int) string {
	out := "unknown"
	switch mode {
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
