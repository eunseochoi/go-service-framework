package rate_limiter

import (
	"context"
	"golang.org/x/time/rate"
	"time"
)

type RateLimitedClient struct {
	RateLimiter     *rate.Limiter
	ErrorSleep      time.Duration // sleep period if the server is not ok
	RateIntervalMs  time.Duration // measurement unit for rate limiter (5 / second) -> 1 * time.Second
	MaxRateRequests int           // max requests per measurement unit (5 / second) -> 5
}

func NewClient(errorSleep time.Duration, rateIntervalMs time.Duration, maxRateRequests int) *RateLimitedClient {
	rl := rate.NewLimiter(rate.Every(rateIntervalMs*time.Millisecond), maxRateRequests)
	return &RateLimitedClient{
		RateLimiter: rl,
		ErrorSleep:  errorSleep,
	}
}

func (r *RateLimitedClient) Exec(ctx context.Context, fn RunnerFunc) error {
	if err := r.RateLimiter.Wait(ctx); err != nil {
		return err
	}
	var err error
	err = fn()
	if err == nil {
		return nil
	}
	return err
}

type RunnerFunc func() error
