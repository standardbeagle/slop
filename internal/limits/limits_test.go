package limits

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	t.Run("nil limiter passes through", func(t *testing.T) {
		var r *RateLimiter
		err := r.Wait(context.Background())
		assert.NoError(t, err)
	})

	t.Run("first call immediate", func(t *testing.T) {
		r := NewRateLimiter(10) // 10 ops/sec
		start := time.Now()
		err := r.Wait(context.Background())
		assert.NoError(t, err)
		assert.Less(t, time.Since(start), 50*time.Millisecond)
	})

	t.Run("rate limiting enforced", func(t *testing.T) {
		r := NewRateLimiter(10) // 10 ops/sec = 100ms between ops
		ctx := context.Background()

		// First call - immediate
		err := r.Wait(ctx)
		require.NoError(t, err)

		// Second call - should wait ~100ms
		start := time.Now()
		err = r.Wait(ctx)
		require.NoError(t, err)
		elapsed := time.Since(start)

		// Should have waited at least 80ms (with some tolerance)
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond)
		assert.Less(t, elapsed, 150*time.Millisecond)
	})

	t.Run("context cancellation interrupts wait", func(t *testing.T) {
		r := NewRateLimiter(1) // 1 op/sec = 1000ms between ops
		ctx, cancel := context.WithCancel(context.Background())

		// First call
		err := r.Wait(ctx)
		require.NoError(t, err)

		// Cancel before second call
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		// Second call should be interrupted
		err = r.Wait(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("zero rate returns nil limiter", func(t *testing.T) {
		r := NewRateLimiter(0)
		assert.Nil(t, r)

		r = NewRateLimiter(-1)
		assert.Nil(t, r)
	})
}

func TestParseRate(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		wantErr  bool
	}{
		{"10/s", 10.0, false},
		{"100/s", 100.0, false},
		{"60/m", 1.0, false},   // 60/min = 1/sec
		{"120/m", 2.0, false},  // 120/min = 2/sec
		{"3600/h", 1.0, false}, // 3600/hr = 1/sec
		{"10/sec", 10.0, false},
		{"10/min", 10.0 / 60.0, false},
		{"10/hr", 10.0 / 3600.0, false},
		{"invalid", 0, true},
		{"10/x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rate, err := ParseRate(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, rate, 0.001)
			}
		})
	}
}

func TestLoopController(t *testing.T) {
	t.Run("limit enforcement", func(t *testing.T) {
		ctx := context.Background()
		lc := NewLoopController(ctx, LoopOptions{Limit: 5})
		defer lc.Done()

		// Should allow 5 iterations
		for i := 0; i < 5; i++ {
			err := lc.BeforeIteration()
			assert.NoError(t, err)
		}

		// 6th should fail
		err := lc.BeforeIteration()
		assert.Equal(t, ErrLimitExceeded, err)
		assert.Equal(t, int64(5), lc.Iterations())
	})

	t.Run("no limit", func(t *testing.T) {
		ctx := context.Background()
		lc := NewLoopController(ctx, LoopOptions{})
		defer lc.Done()

		// Should allow many iterations
		for i := 0; i < 100; i++ {
			err := lc.BeforeIteration()
			require.NoError(t, err)
		}
		assert.Equal(t, int64(100), lc.Iterations())
	})

	t.Run("timeout enforcement", func(t *testing.T) {
		ctx := context.Background()
		lc := NewLoopController(ctx, LoopOptions{Timeout: 100 * time.Millisecond})
		defer lc.Done()

		// First iteration should be fast
		err := lc.BeforeIteration()
		require.NoError(t, err)

		// Wait for timeout
		time.Sleep(150 * time.Millisecond)

		// Next iteration should fail
		err = lc.BeforeIteration()
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("rate limiting", func(t *testing.T) {
		ctx := context.Background()
		lc := NewLoopController(ctx, LoopOptions{Rate: 20}) // 20/sec = 50ms between
		defer lc.Done()

		start := time.Now()

		// Run 3 iterations
		for i := 0; i < 3; i++ {
			err := lc.BeforeIteration()
			require.NoError(t, err)
		}

		elapsed := time.Since(start)
		// 3 iterations at 20/sec should take at least 100ms (2 waits)
		assert.GreaterOrEqual(t, elapsed, 80*time.Millisecond)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		lc := NewLoopController(ctx, LoopOptions{})
		defer lc.Done()

		// First iteration
		err := lc.BeforeIteration()
		require.NoError(t, err)

		// Cancel
		cancel()

		// Next iteration should fail
		err = lc.BeforeIteration()
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("combined options", func(t *testing.T) {
		ctx := context.Background()
		lc := NewLoopController(ctx, LoopOptions{
			Limit:   10,
			Rate:    100, // Fast rate for test
			Timeout: 5 * time.Second,
		})
		defer lc.Done()

		for i := 0; i < 10; i++ {
			err := lc.BeforeIteration()
			require.NoError(t, err)
		}

		// 11th should fail due to limit
		err := lc.BeforeIteration()
		assert.Equal(t, ErrLimitExceeded, err)
	})
}

func TestExecutionStats(t *testing.T) {
	stats := NewExecutionStats()

	stats.AddIterations(100)
	stats.AddLLMCall()
	stats.AddLLMCall()
	stats.AddAPICall()
	stats.AddCost(0.05)
	stats.SetDuration(5 * time.Second)

	assert.Equal(t, int64(100), stats.TotalIterations)
	assert.Equal(t, int64(2), stats.TotalLLMCalls)
	assert.Equal(t, int64(1), stats.TotalAPICalls)
	assert.InDelta(t, 0.05, stats.TotalCost, 0.001)
	assert.Equal(t, 5*time.Second, stats.TotalDuration)

	// Test string representation
	s := stats.String()
	assert.Contains(t, s, "iterations=100")
	assert.Contains(t, s, "llm_calls=2")
	assert.Contains(t, s, "api_calls=1")
}

func TestTimeoutContext(t *testing.T) {
	t.Run("with timeout", func(t *testing.T) {
		ctx, cancel := TimeoutContext(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Should not be done yet
		select {
		case <-ctx.Done():
			t.Fatal("context should not be done yet")
		default:
		}

		// Wait for timeout
		time.Sleep(150 * time.Millisecond)

		// Should be done now
		select {
		case <-ctx.Done():
			assert.Equal(t, context.DeadlineExceeded, ctx.Err())
		default:
			t.Fatal("context should be done")
		}
	})

	t.Run("zero timeout returns parent", func(t *testing.T) {
		parent := context.Background()
		ctx, cancel := TimeoutContext(parent, 0)
		defer cancel()

		// Should be the same as parent (never times out on its own)
		select {
		case <-ctx.Done():
			t.Fatal("zero timeout should not cause immediate done")
		default:
		}
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", time.Hour, false},
		{"100ms", 100 * time.Millisecond, false},
		{"1h30m", 90 * time.Minute, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, d)
			}
		})
	}
}
