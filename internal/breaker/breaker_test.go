package breaker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_Closed(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 5,
		SlidingWindowSize:    10,
	})

	// Should start in closed state
	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %v", cb.State())
	}

	// Successful calls should keep it closed
	for i := 0; i < 5; i++ {
		_, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %v", cb.State())
	}
}

func TestCircuitBreaker_Open(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 5,
		SlidingWindowSize:    10,
		Timeout:              1 * time.Second,
	})

	// Cause failures to trip the breaker
	for i := 0; i < 5; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %v", cb.State())
	}

	// Calls should be rejected
	_, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	if err != ErrOpenState {
		t.Errorf("expected ErrOpenState, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpen(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 3,
		SlidingWindowSize:    5,
		Timeout:              100 * time.Millisecond,
		MaxRequests:          1,
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %v", cb.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should be in half-open state
	if cb.State() != StateHalfOpen {
		t.Errorf("expected state HALF_OPEN, got %v", cb.State())
	}

	// Successful call should close the breaker
	cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})

	// Give a small delay for state transition to complete
	time.Sleep(10 * time.Millisecond)

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %v", cb.State())
	}
}

func TestCircuitBreaker_Metrics(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 5,
		SlidingWindowSize:    10,
	})

	// Execute some calls
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
	for i := 0; i < 2; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	metrics := cb.GetMetrics()
	if metrics.TotalSuccesses != 3 {
		t.Errorf("expected 3 successes, got %d", metrics.TotalSuccesses)
	}
	if metrics.TotalFailures != 2 {
		t.Errorf("expected 2 failures, got %d", metrics.TotalFailures)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 3,
		SlidingWindowSize:    5,
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN, got %v", cb.State())
	}

	// Reset
	cb.Reset()

	if cb.State() != StateClosed {
		t.Errorf("expected state CLOSED after reset, got %v", cb.State())
	}
}

func TestCircuitBreaker_ForceOpen(t *testing.T) {
	cb := New(Config{
		Name:                 "test",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 5,
	})

	cb.ForceOpen()

	if cb.State() != StateForcedOpen {
		t.Errorf("expected state FORCED_OPEN, got %v", cb.State())
	}

	_, err := cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	if err != ErrForcedOpen {
		t.Errorf("expected ErrForcedOpen, got %v", err)
	}
}

func TestCircuitBreaker_SlowCalls(t *testing.T) {
	cb := New(Config{
		Name:                  "test",
		FailureRateThreshold:  100, // Disable failure threshold
		SlowCallRateThreshold: 50,
		SlowCallDuration:      50 * time.Millisecond,
		MinimumNumberOfCalls:  5,
		SlidingWindowSize:     10,
	})

	// Make slow calls
	for i := 0; i < 5; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			time.Sleep(60 * time.Millisecond)
			return "ok", nil
		})
	}

	if cb.State() != StateOpen {
		t.Errorf("expected state OPEN (slow calls), got %v", cb.State())
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var callbackName string
	var fromState, toState State
	var wg sync.WaitGroup
	wg.Add(1)

	cb := New(Config{
		Name:                 "test-cb",
		FailureRateThreshold: 50,
		MinimumNumberOfCalls: 3,
		SlidingWindowSize:    5,
		OnStateChange: func(name string, from, to State) {
			callbackName = name
			fromState = from
			toState = to
			wg.Done()
		},
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		cb.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	// Wait for callback to execute
	wg.Wait()

	if callbackName != "test-cb" {
		t.Errorf("expected callback name 'test-cb', got '%s'", callbackName)
	}
	if fromState != StateClosed {
		t.Errorf("expected from state CLOSED, got %v", fromState)
	}
	if toState != StateOpen {
		t.Errorf("expected to state OPEN, got %v", toState)
	}
}
