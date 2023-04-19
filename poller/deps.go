package poller

import (
	"context"

	"github.com/coherentopensource/go-service-framework/pool"
)

type Driver interface {
	Blockchain() string
	GetChainTipNumber(ctx context.Context) (uint64, error)
	IsValidBlock(ctx context.Context, index uint64) error
	FetchSequence(index uint64) map[string]pool.Runner
	Accumulate(res interface{}) pool.Runner
	Writers() []pool.FeedTransformer
}

type Cache interface {
	GetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string) (uint64, error)
	SetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string, blockNumber uint64) error
}
