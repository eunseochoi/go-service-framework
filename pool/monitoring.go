package pool

func (wp *WorkerPool) incrInProgress() {
	wp.inProgressMu.Lock()
	wp.countInProgress++
	wp.inProgressMu.Unlock()
}

func (wp *WorkerPool) decrInProgress() {
	wp.inProgressMu.Lock()
	wp.countInProgress--
	wp.inProgressMu.Unlock()
}

func (wp *WorkerPool) incrWaiting() {
	wp.waitingMu.Lock()
	wp.countWaiting++
	wp.waitingMu.Unlock()
}

func (wp *WorkerPool) decrWaiting() {
	wp.waitingMu.Lock()
	wp.countWaiting--
	wp.waitingMu.Unlock()
}

func (wp *WorkerPool) Insights() map[string]int {
	wp.groupMu.Lock()
	defer wp.groupMu.Unlock()
	return map[string]int{
		"bandwidth":  wp.bandwidth,
		"inProgress": wp.countInProgress,
		"waiting":    wp.countWaiting,
		"jobCh":      len(wp.jobCh),
		"errCh":      len(wp.errCh),
		"feedCh":     len(wp.feedCh),
		"groups":     len(wp.groups),
	}
}
