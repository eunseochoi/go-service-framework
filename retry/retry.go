package retry

import (
	"errors"
	"time"
)

var errMaxRetriesReached = errors.New("exceeded retry limit")

type RunnerFunc func() error
type SleeperFunc func(attempt int)

func Exec(maxRetries int, fn RunnerFunc, sleep SleeperFunc) error {
	if sleep == nil {
		sleep = DefaultSleeper
	}
	var err error
	attempt := 1
	for {
		err = fn()
		if err == nil {
			return nil
		}
		attempt++
		if attempt > maxRetries {
			break
		}
		sleep(attempt)
		continue
	}
	return err
}
func DefaultSleeper(attempt int) {
	exp := time.Duration(2 ^ (attempt - 1))
	time.Sleep(exp * time.Second)
}
