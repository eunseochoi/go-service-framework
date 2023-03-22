package pool

import (
	"context"
	"errors"
	"github.com/datadaodevs/go-service-framework/utils"
	"github.com/segmentio/ksuid"
	"sync"
)

const (
	defaultBandwidth = 100
)

// WorkerPool is a configurable container for running concurrent tasks, both as one-offs and in groups
// with a receipt signal
type WorkerPool struct {
	id               string
	groups           map[string]*group
	groupMu          *sync.Mutex
	workerWg         *sync.WaitGroup
	feedTransformers []FeedTransformer
	bandwidth        int
	errHandler       ErrHandler
	errCh            chan error
	jobCh            chan job
	resultCh         chan interface{}
	feedCh           <-chan interface{}
	cancel           context.CancelFunc
	throttler        *Throttler
	useOutputCh      bool
	logger           utils.Logger
}

// NewWorkerPool instantiates a worker pool with default options
func NewWorkerPool(id string, opts ...opt) *WorkerPool {
	wp := WorkerPool{
		id:        id,
		groups:    map[string]*group{},
		groupMu:   &sync.Mutex{},
		workerWg:  &sync.WaitGroup{},
		bandwidth: defaultBandwidth,
	}
	wp.errHandler = wp.defaultErrHandler
	for _, opt := range opts {
		opt(&wp)
	}

	wp.jobCh = make(chan job, wp.bandwidth)
	wp.errCh = make(chan error, wp.bandwidth)
	wp.resultCh = make(chan interface{}, wp.bandwidth)

	return &wp
}

// Start initializes workers and readies the worker pool to receive jobs and groups
func (wp *WorkerPool) Start(parentCtx context.Context) error {
	wp.logger.Infof("Logger starting for worker pool [%s]", wp.id)

	if wp.logger == nil {
		return errors.New("Logger not configured")
	}

	innerCtx, cancel := context.WithCancel(parentCtx)
	wp.cancel = cancel

	wp.startJobWorkers(innerCtx)
	wp.startErrorWorkers(innerCtx)
	wp.startFeeder(innerCtx)

	return nil
}

// Stop performs a graceful shutdown of all workers
func (wp *WorkerPool) Stop(ctx context.Context) {
	wp.cancel()
	wp.workerWg.Wait()
}

// SetInputFeed configures the workerpool to receive jobs from an input channel, with a "transformer" method
// that converts a generic input interface into a Runner
func (wp *WorkerPool) SetInputFeed(feed <-chan interface{}, transformers ...FeedTransformer) {
	wp.feedCh = feed
	wp.feedTransformers = transformers
}

// PushGroup queues a group of Runners for execution, with a receipt signal to be sent to the supplied wg when
// all Runners are completed
func (wp *WorkerPool) PushGroup(fns map[string]Runner, wg *sync.WaitGroup) {
	groupID := ksuid.New().String()
	wp.groupMu.Lock()
	defer wp.groupMu.Unlock()
	wp.groups[groupID] = &group{
		results:   ResultSet{},
		receiptWg: wg,
		cursor:    0,
		jobCount:  len(fns),
	}
	go func() {
		for jobID, fn := range fns {
			wp.jobCh <- job{fn: fn, groupID: groupID, id: jobID}
		}
	}()
}

// PushJob queues a one-off job for execution
func (wp *WorkerPool) PushJob(fn Runner) {
	id := ksuid.New().String()
	wp.jobCh <- job{fn: fn, id: id}
}

// Results gives public access to a channel that will receive results as they are processed; requires that the
// WithOutputChannel() option be passed to the constructor for proper functionality
func (wp *WorkerPool) Results() <-chan interface{} {
	return wp.resultCh
}

// startFeeder spins up a worker to read off of a feed channel, if a feeder has been specified with the SetInputFeed()
// method
func (wp *WorkerPool) startFeeder(ctx context.Context) {
	if wp.feedTransformers == nil || wp.feedCh == nil {
		return
	}

	wp.workerWg.Add(1)
	go func() {
		defer wp.workerWg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-wp.feedCh:
				for _, transformer := range wp.feedTransformers {
					wp.jobCh <- job{fn: transformer(res)}
				}
			}
		}
	}()
}

// startJobWorkers spins up workers to process jobs
func (wp *WorkerPool) startJobWorkers(ctx context.Context) {
	for i := 0; i < wp.bandwidth; i++ {
		wp.workerWg.Add(1)
		go func(ctx context.Context) {
			defer wp.workerWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-wp.jobCh:
					if wp.throttler != nil {
						wp.throttler.WaitForGo()
					}
					switch {
					case job.groupID != "":
						res, err := job.fn(ctx)
						wp.processGroupResult(job.groupID, job.id, res, err)
					default:
						res, err := job.fn(ctx)
						if err != nil {
							wp.errCh <- err
							break
						}
						if wp.useOutputCh {
							wp.resultCh <- res
						}
					}
				}
			}
		}(ctx)
	}
}

// startErrorWorkers spins up workers to process errors
func (wp *WorkerPool) startErrorWorkers(ctx context.Context) {
	for i := 0; i < wp.bandwidth; i++ {
		wp.workerWg.Add(1)
		go func(ctx context.Context) {
			defer wp.workerWg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case err := <-wp.errCh:
					wp.errHandler(err)
				}
			}
		}(ctx)
	}
}

// processGroupResult processes the result (or error) for a job, given the job is part of a group
func (wp *WorkerPool) processGroupResult(groupID string, jobID string, res interface{}, err error) {
	//	lock group mutex
	wp.groupMu.Lock()
	defer wp.groupMu.Unlock()

	//	pull group from memory
	group := wp.groups[groupID]

	//	increment cursor
	group.cursor++

	//	accumulate result
	group.results[jobID] = res

	//	if we're complete with the group, send wg receipt and push result to results ch, if applicable
	if group.cursor == group.jobCount {
		//	receiptWg
		group.receiptWg.Done()

		//	if this is an error, push the error and exit
		if err != nil {
			wp.errCh <- err
			return
		}

		if wp.useOutputCh {
			wp.resultCh <- ResultSet(group.results)
		}
	}
}

// defaultErrHandler logs an error generically
func (wp *WorkerPool) defaultErrHandler(err error) {
	wp.logger.Errorf("Error captured by worker in pool [%s]: %v", wp.id, err)
}
