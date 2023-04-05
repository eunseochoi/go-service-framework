package pool

import (
	"context"
	"sync"
)

// Runner is an executable function that runs as a job
type Runner func(ctx context.Context) (interface{}, error)

// ErrHandler handles an error result from a Runner
type ErrHandler func(err error)

// FeedTransformer transforms a generic input into a Runner
type FeedTransformer func(res interface{}) Runner

// ResultSet is a set of results accumulated from a group
type ResultSet map[string]interface{}

type result struct {
	payload interface{}
	wg      *sync.WaitGroup
}

// job is an internal enclosure for a Runner that specifies and ID and group info
type job struct {
	fn        Runner
	id        string
	groupID   string
	receiptWg *sync.WaitGroup
}

// group is a collection of jobs meant to be run in parallel with the result processed as a unit
type group struct {
	results    map[string]interface{}
	pipelineWg *sync.WaitGroup
	cursor     int
	jobCount   int
}
