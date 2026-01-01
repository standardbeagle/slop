// Package limits provides execution control and limits for SLOP programs.
package limits

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateLimiter controls the rate of operations.
type RateLimiter struct {
	rate     float64       // operations per second
	interval time.Duration // time between operations
	lastOp   time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a rate limiter with the given operations per second.
func NewRateLimiter(rate float64) *RateLimiter {
	if rate <= 0 {
		return nil
	}
	return &RateLimiter{
		rate:     rate,
		interval: time.Duration(float64(time.Second) / rate),
	}
}

// Wait blocks until the next operation is allowed.
// Returns error if the context is cancelled.
func (r *RateLimiter) Wait(ctx context.Context) error {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	if r.lastOp.IsZero() {
		r.lastOp = now
		return nil
	}

	nextAllowed := r.lastOp.Add(r.interval)
	if now.Before(nextAllowed) {
		waitTime := nextAllowed.Sub(now)

		select {
		case <-time.After(waitTime):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	r.lastOp = time.Now()
	return nil
}

// Rate returns the operations per second.
func (r *RateLimiter) Rate() float64 {
	if r == nil {
		return 0
	}
	return r.rate
}

// ParseRate parses a rate string like "10/s" or "100/m".
// Returns operations per second.
func ParseRate(rate string) (float64, error) {
	var count float64
	var unit string

	_, err := fmt.Sscanf(rate, "%f/%s", &count, &unit)
	if err != nil {
		return 0, fmt.Errorf("invalid rate format: %s (expected N/s or N/m)", rate)
	}

	switch unit {
	case "s", "sec", "second":
		return count, nil
	case "m", "min", "minute":
		return count / 60.0, nil
	case "h", "hr", "hour":
		return count / 3600.0, nil
	default:
		return 0, fmt.Errorf("unknown rate unit: %s (expected s, m, or h)", unit)
	}
}

// TimeoutContext creates a context with the given timeout duration.
func TimeoutContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return parent, func() {}
	}
	return context.WithTimeout(parent, timeout)
}

// ParseDuration parses a duration string like "30s", "5m", "1h".
func ParseDuration(s string) (time.Duration, error) {
	return time.ParseDuration(s)
}

// LoopController manages execution of a bounded loop.
type LoopController struct {
	ctx         context.Context
	cancel      context.CancelFunc
	rateLimiter *RateLimiter
	maxIter     int64
	currentIter int64
	timeout     time.Duration
	startTime   time.Time
	mu          sync.Mutex
}

// LoopOptions configures a loop controller.
type LoopOptions struct {
	Limit   int64         // Maximum iterations (0 = unlimited)
	Rate    float64       // Rate in operations per second (0 = unlimited)
	Timeout time.Duration // Maximum execution time (0 = unlimited)
}

// NewLoopController creates a controller for bounded loop execution.
func NewLoopController(ctx context.Context, opts LoopOptions) *LoopController {
	loopCtx, cancel := ctx, func() {}
	if opts.Timeout > 0 {
		loopCtx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}

	var rateLimiter *RateLimiter
	if opts.Rate > 0 {
		rateLimiter = NewRateLimiter(opts.Rate)
	}

	return &LoopController{
		ctx:         loopCtx,
		cancel:      cancel,
		rateLimiter: rateLimiter,
		maxIter:     opts.Limit,
		timeout:     opts.Timeout,
		startTime:   time.Now(),
	}
}

// BeforeIteration checks if the next iteration should proceed.
// It applies rate limiting and checks limits.
func (lc *LoopController) BeforeIteration() error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	// Check context cancellation
	select {
	case <-lc.ctx.Done():
		return lc.ctx.Err()
	default:
	}

	// Check iteration limit
	if lc.maxIter > 0 && lc.currentIter >= lc.maxIter {
		return ErrLimitExceeded
	}

	// Apply rate limiting
	if lc.rateLimiter != nil {
		if err := lc.rateLimiter.Wait(lc.ctx); err != nil {
			return err
		}
	}

	lc.currentIter++
	return nil
}

// Done releases resources.
func (lc *LoopController) Done() {
	lc.cancel()
}

// Iterations returns the number of completed iterations.
func (lc *LoopController) Iterations() int64 {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	return lc.currentIter
}

// Elapsed returns the time since the loop started.
func (lc *LoopController) Elapsed() time.Duration {
	return time.Since(lc.startTime)
}

// Limit errors
var (
	ErrLimitExceeded  = fmt.Errorf("iteration limit exceeded")
	ErrTimeoutReached = fmt.Errorf("timeout reached")
	ErrRateLimited    = fmt.Errorf("rate limit exceeded")
)

// ExecutionStats tracks execution statistics.
type ExecutionStats struct {
	TotalIterations int64
	TotalLLMCalls   int64
	TotalAPICalls   int64
	TotalDuration   time.Duration
	TotalCost       float64
	mu              sync.Mutex
}

// NewExecutionStats creates a new stats tracker.
func NewExecutionStats() *ExecutionStats {
	return &ExecutionStats{}
}

// AddIterations adds to the iteration count.
func (s *ExecutionStats) AddIterations(n int64) {
	s.mu.Lock()
	s.TotalIterations += n
	s.mu.Unlock()
}

// AddLLMCall increments the LLM call count.
func (s *ExecutionStats) AddLLMCall() {
	s.mu.Lock()
	s.TotalLLMCalls++
	s.mu.Unlock()
}

// AddAPICall increments the API call count.
func (s *ExecutionStats) AddAPICall() {
	s.mu.Lock()
	s.TotalAPICalls++
	s.mu.Unlock()
}

// AddCost adds to the total cost.
func (s *ExecutionStats) AddCost(cost float64) {
	s.mu.Lock()
	s.TotalCost += cost
	s.mu.Unlock()
}

// SetDuration sets the total duration.
func (s *ExecutionStats) SetDuration(d time.Duration) {
	s.mu.Lock()
	s.TotalDuration = d
	s.mu.Unlock()
}

// String returns a human-readable summary.
func (s *ExecutionStats) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("iterations=%d llm_calls=%d api_calls=%d duration=%v cost=%.4f",
		s.TotalIterations, s.TotalLLMCalls, s.TotalAPICalls, s.TotalDuration, s.TotalCost)
}
