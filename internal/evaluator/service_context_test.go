package evaluator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ctxAwareService implements both Service and ContextService. It records the
// context it received and optionally blocks until that context is cancelled so
// tests can assert the deadline actually propagates to the call.
type ctxAwareService struct {
	gotContext context.Context
	block      bool
}

func (s *ctxAwareService) Call(method string, args []Value, kwargs map[string]Value) (Value, error) {
	// Plain path should never be taken when CallWithContext is present.
	return &StringValue{Value: "plain"}, nil
}

func (s *ctxAwareService) CallWithContext(ctx context.Context, method string, args []Value, kwargs map[string]Value) (Value, error) {
	s.gotContext = ctx
	if s.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return &StringValue{Value: "ctx"}, nil
}

// plainService implements only Service (no context support).
type plainService struct{ called bool }

func (s *plainService) Call(method string, args []Value, kwargs map[string]Value) (Value, error) {
	s.called = true
	return &StringValue{Value: "plain"}, nil
}

func TestCallServiceMethodPrefersContext(t *testing.T) {
	t.Run("context-aware service receives a deadline rooted in MaxDuration", func(t *testing.T) {
		ctx := NewContextWithLimits(&ExecutionLimits{MaxDuration: 30})
		eval := NewWithContext(ctx)

		svc := &ctxAwareService{}
		bound := &BoundMethodValue{ServiceName: "svc", Service: svc, Method: "do"}

		res, err := eval.callServiceMethod(bound, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, "ctx", res.(*StringValue).Value)

		require.NotNil(t, svc.gotContext)
		deadline, ok := svc.gotContext.Deadline()
		require.True(t, ok, "call context must carry a deadline when MaxDuration is set")
		// Deadline should be at most MaxDuration from the start anchor.
		assert.LessOrEqual(t, time.Until(deadline), 30*time.Second)
	})

	t.Run("no deadline when MaxDuration is unset", func(t *testing.T) {
		ctx := NewContextWithLimits(&ExecutionLimits{})
		eval := NewWithContext(ctx)

		svc := &ctxAwareService{}
		bound := &BoundMethodValue{ServiceName: "svc", Service: svc, Method: "do"}

		_, err := eval.callServiceMethod(bound, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, svc.gotContext)
		_, ok := svc.gotContext.Deadline()
		assert.False(t, ok, "no deadline should be set when MaxDuration <= 0")
	})

	t.Run("hung call unwinds via deadline instead of blocking forever", func(t *testing.T) {
		// Budget still has ~2s left at the CheckDuration gate (it passes), but the
		// call context deadline fires shortly after, so a blocking service unwinds
		// with DeadlineExceeded rather than hanging its goroutine forever.
		ctx := NewContextWithLimits(&ExecutionLimits{
			MaxDuration: 3,
			StartTime:   time.Now().Add(-1 * time.Second).Unix(),
		})
		eval := NewWithContext(ctx)

		svc := &ctxAwareService{block: true}
		bound := &BoundMethodValue{ServiceName: "svc", Service: svc, Method: "do"}

		done := make(chan error, 1)
		go func() {
			_, err := eval.callServiceMethod(bound, nil, nil)
			done <- err
		}()

		select {
		case err := <-done:
			require.Error(t, err)
			assert.ErrorIs(t, err, context.DeadlineExceeded)
		case <-time.After(5 * time.Second):
			t.Fatal("hung service call did not unwind via deadline")
		}
	})

	t.Run("service without context support falls back to plain Call", func(t *testing.T) {
		ctx := NewContextWithLimits(&ExecutionLimits{MaxDuration: 30})
		eval := NewWithContext(ctx)

		svc := &plainService{}
		bound := &BoundMethodValue{ServiceName: "svc", Service: svc, Method: "do"}

		res, err := eval.callServiceMethod(bound, nil, nil)
		require.NoError(t, err)
		assert.True(t, svc.called, "plain Call must be invoked when CallWithContext is absent")
		assert.Equal(t, "plain", res.(*StringValue).Value)
	})
}
