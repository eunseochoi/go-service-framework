package poller

import (
	"fmt"
	"github.com/datadaodevs/go-service-framework/constants"
)

func (p *Poller) cacheKey() string {
	return fmt.Sprintf("%s-%s", p.driver.Blockchain(), constants.BlockKey)
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
