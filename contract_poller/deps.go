package contract_poller

import (
	"context"
	"github.com/coherentopensource/go-service-framework/pool"
)

type Driver interface {
	Blockchain() string
	GetChainTipNumber(ctx context.Context) (uint64, error)
	// FetchSequence involves fetching trace and receipt from ethrpc node
	FetchSequence(blockNumber uint64) map[string]pool.Runner
	// Fetchers involves fetching contract's abi and metadata, using results from FetchContractAddresses
	Fetchers() map[string]pool.FeedTransformer
	// Accumulate involves combining abi, metadata to form a complete contract, building fragments
	Accumulate(res interface{}) pool.Runner
	// Writers involves writing to postgresDB
	Writers() []pool.FeedTransformer
}

type Cache interface {
	GetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string) (uint64, error)
	SetCurrentBlockNumber(ctx context.Context, blockChainInfoKey string, blockNumber uint64) error
}
