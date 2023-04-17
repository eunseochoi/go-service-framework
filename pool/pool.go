package pool

import (
	"context"
	"errors"
	"github.com/coherentopensource/go-service-framework/util"
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
	parentCtx        context.Context
	countWaiting     int
	countInProgress  int
	waitingMu        *sync.Mutex
	inProgressMu     *sync.Mutex
	groups           map[string]*group
	groupMu          *sync.Mutex
	workerWg         *sync.WaitGroup
	feedTransformers map[string]FeedTransformer
	useGroupForFeed  bool
	bandwidth        int
	errHandler       ErrHandler
	errCh            chan error
	jobCh            chan job
	resultCh         chan result
	feedCh           <-chan result
	cancel           context.CancelFunc
	throttler        *Throttler
	useOutputCh      bool
	logger           util.Logger
}

// NewWorkerPool instantiates a worker pool with default options
func NewWorkerPool(id string, opts ...opt) *WorkerPool {
	wp := WorkerPool{
		id:        id,
		bandwidth: defaultBandwidth,
		// this is default anyway, but specifying for explicitness
		useGroupForFeed: false,
	}
	wp.errHandler = wp.defaultErrHandler
	for _, opt := range opts {
		opt(&wp)
	}
	wp.refreshControls()

	return &wp
}

// Start initializes workers and readies the worker pool to receive jobs and groups
func (wp *WorkerPool) Start(parentCtx context.Context) error {
	wp.logger.Infof("Logger starting for worker pool [%s]", wp.id)

	if wp.logger == nil {
		return errors.New("Logger not configured")
	}

	wp.parentCtx = parentCtx
	innerCtx, cancel := context.WithCancel(parentCtx)
	wp.cancel = cancel

	wp.startJobWorkers(innerCtx)
	wp.startErrorWorkers(innerCtx)
	wp.startFeeder(innerCtx)

	return nil
}

func (wp *WorkerPool) refreshControls() {
	wp.groups = map[string]*group{}
	wp.groupMu = &sync.Mutex{}
	wp.workerWg = &sync.WaitGroup{}
	wp.inProgressMu = &sync.Mutex{}
	wp.waitingMu = &sync.Mutex{}
	wp.jobCh = make(chan job, wp.bandwidth)
	wp.errCh = make(chan error, wp.bandwidth)
	wp.resultCh = make(chan result, wp.bandwidth)
}

// Stop performs a graceful shutdown of all workers
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.workerWg.Wait()
	close(wp.jobCh)
	close(wp.errCh)
	close(wp.resultCh)
}

func (wp *WorkerPool) FlushAndRestart() {
	wp.Stop()
	wp.refreshControls()
	wp.Start(wp.parentCtx)
}

// SetInputFeed configures the workerpool to receive jobs from an input channel, with "transformer" methods
// that convert a generic input interface into a Runner
func (wp *WorkerPool) SetInputFeed(feed <-chan result, transformers ...FeedTransformer) {
	if wp.feedCh != nil {
		wp.logger.Warn("Attempting to set input feed, when input feed is already defined")
		return
	}

	wp.feedCh = feed
	wp.feedTransformers = map[string]FeedTransformer{}
	for _, transformer := range transformers {
		wp.feedTransformers[ksuid.New().String()] = transformer
	}
}

// SetGroupInputFeed configures the workerpool to receive jobs from an input channel, with "transformer" methods
// that convert a generic input interface into a Runner; with the runners executed as a group
func (wp *WorkerPool) SetGroupInputFeed(feed <-chan result, groupMap map[string]FeedTransformer) {
	if wp.feedCh != nil {
		wp.logger.Warn("Attempting to set group input feed, when input feed is already defined")
		return
	}

	wp.feedCh = feed
	wp.feedTransformers = groupMap
	wp.useGroupForFeed = true
}

// PushGroup queues a group of Runners for execution, with a receipt signal to be sent to the supplied receiptWg when
// all Runners are completed
func (wp *WorkerPool) PushGroup(fns map[string]Runner, wg *sync.WaitGroup) {
	groupID := ksuid.New().String()
	wp.groupMu.Lock()
	defer wp.groupMu.Unlock()
	wp.groups[groupID] = &group{
		results:  ResultSet{},
		cursor:   0,
		jobCount: len(fns),
	}
	go func() {
		for jobID, fn := range fns {
			wp.jobCh <- job{fn: fn, groupID: groupID, id: jobID, receiptWg: wg}
		}
	}()
}

// PushJob queues a one-off job for execution
func (wp *WorkerPool) PushJob(fn Runner, wg *sync.WaitGroup) {
	id := ksuid.New().String()
	wp.jobCh <- job{fn: fn, id: id, receiptWg: wg}
}

// Results gives public access to a channel that will receive results as they are processed; requires that the
// WithOutputChannel() option be passed to the constructor for proper functionality
func (wp *WorkerPool) Results() <-chan result {
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
				group := map[string]Runner{}
				for id, transformer := range wp.feedTransformers {
					switch {
					case wp.useGroupForFeed:
						group[id] = transformer(res.payload)
					default:
						wp.jobCh <- job{fn: transformer(res.payload), receiptWg: res.wg, id: ksuid.New().String()}
					}
				}
				if wp.useGroupForFeed {
					wp.PushGroup(group, res.wg)
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
						wp.incrWaiting()
						wp.throttler.WaitForGo()
						wp.decrWaiting()
					}
					wp.incrInProgress()
					switch {
					case job.groupID != "":
						res, err := job.fn(ctx)
						wp.processGroupResult(&job, res, err)
					default:
						res, err := job.fn(ctx)
						if err != nil {
							wp.errCh <- err
							job.receiptWg.Done()
							break
						}
						if wp.useOutputCh {
							wp.resultCh <- result{payload: res, wg: job.receiptWg}
						}
						job.receiptWg.Done()
					}
					wp.decrInProgress()
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
func (wp *WorkerPool) processGroupResult(job *job, res interface{}, err error) {
	//	lock group mutex
	wp.groupMu.Lock()
	defer wp.groupMu.Unlock()
	defer job.receiptWg.Done()

	//	pull group from memory
	group := wp.groups[job.groupID]

	//	increment cursor
	group.cursor++

	//	accumulate result
	group.results[job.id] = res

	//	if we're complete with the group, send wg receipt and push result to results ch, if applicable
	if group.cursor == group.jobCount {
		//	if this is an error, push the error and exit
		if err != nil {
			wp.errCh <- err
			delete(wp.groups, job.groupID)
			return
		}

		if wp.useOutputCh {
			wp.resultCh <- result{payload: ResultSet(group.results), wg: job.receiptWg}
		}

		delete(wp.groups, job.groupID)
	}
}

// defaultErrHandler logs an error generically
func (wp *WorkerPool) defaultErrHandler(err error) {
	wp.logger.Errorf("Error captured by worker in pool [%s]: %v", wp.id, err)
}
