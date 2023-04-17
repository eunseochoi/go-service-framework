package pool_test

import (
	"context"
	"fmt"
	"github.com/coherentopensource/go-service-framework/pool"
	"go.uber.org/zap"
	"sync"
	"testing"
	"time"
)

func TestChainedWorkerPools(t *testing.T) {
	//	Static number of test runs
	testRuns := 5

	//	Increment as the test is set up
	var jobsPerRun int

	//	Instantiate logger
	midLogger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Error instantiating logger: %v", err)
	}
	logger := midLogger.Sugar()

	//	Instantiate the pools, with no output channel for the last one
	pool1 := pool.NewWorkerPool("pool1", pool.WithLogger(logger), pool.WithOutputChannel())
	pool2 := pool.NewWorkerPool("pool2", pool.WithLogger(logger), pool.WithOutputChannel())
	pool3 := pool.NewWorkerPool("pool3", pool.WithLogger(logger))

	//	Set up counters to keep track of jobs
	completedJobs := 0
	completedJobsMutex := &sync.Mutex{}

	//	Set up a single runner that will lock on the name of the job
	getRunner := func(jobName string) pool.Runner {
		return func(ctx context.Context) (interface{}, error) {
			logger.Infof("[%s]: Starting a job", jobName)
			time.Sleep(100 * time.Millisecond)
			completedJobsMutex.Lock()
			completedJobs++
			logger.Infof("[%s]: Finishing a job; [%d] jobs completed", jobName, completedJobs)
			if completedJobs == testRuns*jobsPerRun {
				logger.Infof("%d jobs have now completed; the test should now exit promptly", testRuns*jobsPerRun)
			}
			completedJobsMutex.Unlock()
			return struct{}{}, nil
		}
	}

	//	Wrap the runner in a transformer
	getTransformer := func(jobName string) pool.FeedTransformer {
		return func(res interface{}) pool.Runner {
			return getRunner(jobName)
		}
	}

	//	Pool2 feeds from pool1 using the transformer
	jobsPerRun++
	pool2.SetInputFeed(pool1.Results(), getTransformer("pool2 transform"))

	//	Pool3 feeds from pool2 using the transformer
	jobsPerRun++
	pool3.SetInputFeed(pool2.Results(), getTransformer("pool3 transform"))

	//	Prepare to kill all processes if there is a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//	Start the pools and defer stopping them
	pool1.Start(ctx)
	pool2.Start(ctx)
	pool3.Start(ctx)
	defer pool1.Stop()
	defer pool2.Stop()
	defer pool3.Stop()

	//	For pool1, use a group
	fns := map[string]pool.Runner{
		"fn1": getRunner("pool1 job1"),
		"fn2": getRunner("pool1 job2"),
		"fn3": getRunner("pool1 job3"),
		"fn4": getRunner("pool1 job4"),
	}
	jobsPerRun += len(fns)

	//	Queue work into pool1
	logger.Infof("Starting group pushes; expect %d jobs per test run, %d test runs, and %d total jobs run", jobsPerRun, testRuns, jobsPerRun*testRuns)
	wg := &sync.WaitGroup{}
	for i := 0; i < testRuns; i++ {
		wg.Add(jobsPerRun)
		pool1.PushGroup(fns, wg)
	}

	//	Wait for all work to complete
	endCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(endCh)
	}()
	select {
	case <-endCh:
	case <-ctx.Done():
		t.Fatal("Test timed out")
	}

	//	Check completed versus expected jobs
	if completedJobs != testRuns*jobsPerRun {
		t.Errorf("Expected %d completed jobs, but got %d", testRuns*jobsPerRun, completedJobs)
		return
	}
}

func TestGroupFeed(t *testing.T) {
	//	Increment as the test is set up
	var totalExpectedJobs int

	//	Instantiate logger
	midLogger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Error instantiating logger: %v", err)
	}
	logger := midLogger.Sugar()

	//	Instantiate the pools, with no output channel for the last one
	pool1 := pool.NewWorkerPool("pool1", pool.WithLogger(logger), pool.WithOutputChannel())
	pool2 := pool.NewWorkerPool("pool2", pool.WithLogger(logger), pool.WithOutputChannel())
	pool3 := pool.NewWorkerPool("pool3", pool.WithLogger(logger))

	//	Set up counters to keep track of jobs
	completedJobs := 0
	completedJobsMutex := &sync.Mutex{}

	//	Set up a single runner that will lock on the name of the job
	getRunner := func(jobName string) pool.Runner {
		return func(ctx context.Context) (interface{}, error) {
			logger.Infof("[%s]: Starting a job", jobName)
			time.Sleep(100 * time.Millisecond)
			completedJobsMutex.Lock()
			completedJobs++
			logger.Infof("[%s]: Finishing a job; [%d] jobs completed", jobName, completedJobs)
			if completedJobs == totalExpectedJobs {
				logger.Infof("%d jobs have now completed; the test should now exit promptly", totalExpectedJobs)
			}

			//	if this is the accumulator, it should be the last job run
			if jobName == "accumulator" {
				if completedJobs != totalExpectedJobs {
					t.Fatalf("Expected accumulator to be the %dth job run, but instead it was the %dth", totalExpectedJobs, completedJobs)
				}
			}

			completedJobsMutex.Unlock()

			return struct{}{}, nil
		}
	}

	//	Wrap the runner in a transformer
	getTransformer := func(jobName string) pool.FeedTransformer {
		return func(res interface{}) pool.Runner {
			return getRunner(jobName)
		}
	}

	//	For the middle pool, use a group
	fns := map[string]pool.FeedTransformer{
		"fn1": getTransformer("group job1"),
		"fn2": getTransformer("group job2"),
		"fn3": getTransformer("group job3"),
		"fn4": getTransformer("group job4"),
	}
	totalExpectedJobs += len(fns)

	//	Pool2 feeds from pool1 using the transformers
	totalExpectedJobs++
	pool2.SetGroupInputFeed(
		pool1.Results(),
		fns,
	)

	//	Pool3 feeds from pool2 using the transformer
	totalExpectedJobs++
	pool3.SetInputFeed(pool2.Results(), getTransformer("accumulator"))

	//	Prepare to kill all processes if there is a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//	Start the pools and defer stopping them
	pool1.Start(ctx)
	pool2.Start(ctx)
	pool3.Start(ctx)
	defer pool1.Stop()
	defer pool2.Stop()
	defer pool3.Stop()

	//	Queue work into pool1
	logger.Infof("Starting group pushes; expect %d jobs to run, with accumulator last", totalExpectedJobs)
	wg := &sync.WaitGroup{}
	wg.Add(totalExpectedJobs)
	pool1.PushJob(getRunner(fmt.Sprintf("entry job")), wg)

	//	Wait for all work to complete
	endCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(endCh)
	}()
	select {
	case <-endCh:
	case <-ctx.Done():
		t.Fatal("Test timed out")
	}

	//	Check completed versus expected jobs
	if completedJobs != totalExpectedJobs {
		t.Errorf("Expected %d completed jobs, but got %d", totalExpectedJobs, completedJobs)
		return
	}
}
