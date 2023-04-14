package contract_poller

import (
	"context"
	"github.com/datadaodevs/go-service-framework/pool"
)

type Driver interface {
	Blockchain() string
	GetChainTipNumber(ctx context.Context) (uint64, error)
	// FetchSequenceForContract involves fetching trace from node, abi from contract source, and metadata from node in a sequential manner
	FetchSequenceForContract(index uint64) map[string]pool.Runner
	// Accumulate involves combining abi, metadata to form a complete contract, building fragments
	Accumulate(res interface{}) pool.Runner
	// Writers involves writing to postgresDB
	Writers() []pool.FeedTransformer
}

type Cache interface {
	GetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string) (uint64, error)
	SetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string, blockNumber uint64) error
}
