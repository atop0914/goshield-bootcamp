package breaker

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestAdaptiveBreaker_Basic(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "adaptive-test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 5,
			SlidingWindowSize:    10,
		},
	})

	if ab.Name() != "adaptive-test" {
		t.Errorf("expected name 'adaptive-test', got '%s'", ab.Name())
	}

	if ab.State() != StateClosed {
		t.Errorf("expected initial state CLOSED, got %v", ab.State())
	}
}

func TestAdaptiveBreaker_SuccessfulCalls(t *testing.T) {
	ab := NewAdaptive(DefaultAdaptiveConfig("test"))

	for i := 0; i < 20; i++ {
		_, err := ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
		if err != nil {
			t.Fatalf("unexpected error on call %d: %v", i, err)
		}
	}

	if ab.State() != StateClosed {
		t.Errorf("expected state CLOSED after all successes, got %v", ab.State())
	}
}

func TestAdaptiveBreaker_TripOnFailures(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 5,
			SlidingWindowSize:    10,
			Timeout:              100 * time.Millisecond,
		},
	})

	// Cause failures to trip
	for i := 0; i < 5; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	if ab.State() != StateOpen {
		t.Errorf("expected state OPEN, got %v", ab.State())
	}

	// The OnStateChange callback runs in a goroutine, wait for it
	time.Sleep(20 * time.Millisecond)

	if ab.TripCount() != 1 {
		t.Errorf("expected trip count 1, got %d", ab.TripCount())
	}
}

func TestAdaptiveBreaker_ConsecutiveFailureLimit(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 100, // Disable normal tripping
			MinimumNumberOfCalls: 100,
			SlidingWindowSize:    200,
		},
		ConsecutiveFailureLimit: 3,
	})

	// 3 consecutive failures should force open
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	// Next call should be rejected due to consecutive failure limit
	_, err := ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	if err != ErrOpenState {
		t.Errorf("expected ErrOpenState, got %v", err)
	}
}

func TestAdaptiveBreaker_ConsecutiveFailureReset(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 100,
			MinimumNumberOfCalls: 100,
			SlidingWindowSize:    200,
		},
		ConsecutiveFailureLimit: 5,
	})

	// 2 failures, then success, then 2 failures — should not trip
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("failure")
	})
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("failure")
	})
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil // Reset consecutive count
	})
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("failure")
	})
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return nil, errors.New("failure")
	})

	// Should still be closed (consecutive count was reset)
	if ab.State() != StateClosed {
		t.Errorf("expected state CLOSED, got %v", ab.State())
	}
}

func TestAdaptiveBreaker_UpdateEMAs(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
		},
		FailureRateEMAAlpha: 0.5,
		LatencyEMAAlpha:     0.5,
	})

	// Make some calls
	for i := 0; i < 10; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			time.Sleep(5 * time.Millisecond)
			if i < 3 {
				return nil, errors.New("failure")
			}
			return "ok", nil
		})
	}

	ab.UpdateEMAs()

	params := ab.GetAdaptiveParams()
	if params.FailureRateEMA <= 0 {
		t.Errorf("expected positive failure rate EMA, got %f", params.FailureRateEMA)
	}
	if params.LatencyEMA <= 0 {
		t.Errorf("expected positive latency EMA, got %v", params.LatencyEMA)
	}
}

func TestAdaptiveBreaker_AdaptiveThreshold_Tighten(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
		},
		FailureRateEMAAlpha:     0.5,
		MinFailureRateThreshold: 20,
		MaxFailureRateThreshold: 80,
	})

	// All successes → low failure rate EMA
	for i := 0; i < 20; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}

	ab.UpdateEMAs()

	params := ab.GetAdaptiveParams()
	// With low failure rate, threshold should tighten below 50
	if params.AdaptiveThreshold >= 50 {
		t.Errorf("expected tightened threshold < 50, got %f", params.AdaptiveThreshold)
	}
	if params.AdaptiveThreshold < 20 {
		t.Errorf("threshold should not go below min (20), got %f", params.AdaptiveThreshold)
	}
}

func TestAdaptiveBreaker_AdaptiveThreshold_Relax(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    200,
		},
		FailureRateEMAAlpha:     0.8, // Quick adaptation
		MinFailureRateThreshold: 20,
		MaxFailureRateThreshold: 80,
	})

	// Simulate high failure rate over multiple EMA updates
	for round := 0; round < 5; round++ {
		for i := 0; i < 10; i++ {
			ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
				return nil, errors.New("failure")
			})
		}
		ab.UpdateEMAs()
	}

	params := ab.GetAdaptiveParams()
	// With high failure rate, threshold should relax above 50
	if params.AdaptiveThreshold <= 50 {
		t.Errorf("expected relaxed threshold > 50, got %f", params.AdaptiveThreshold)
	}
	if params.AdaptiveThreshold > 80 {
		t.Errorf("threshold should not exceed max (80), got %f", params.AdaptiveThreshold)
	}
}

func TestAdaptiveBreaker_SlowCallMultiplier(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 100,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
		},
		SlowCallMultiplier:  2.0,
		MinSlowCallDuration: 10 * time.Millisecond,
		MaxSlowCallDuration: 10 * time.Second,
		LatencyEMAAlpha:     0.5,
	})

	// Make calls with consistent 20ms latency
	for i := 0; i < 10; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			time.Sleep(20 * time.Millisecond)
			return "ok", nil
		})
	}

	ab.UpdateEMAs()

	params := ab.GetAdaptiveParams()
	// Slow call duration should be ~40ms (20ms * 2.0)
	if params.EffectiveSlowCall < 10*time.Millisecond {
		t.Errorf("expected slow call duration > 10ms, got %v", params.EffectiveSlowCall)
	}
}

func TestAdaptiveBreaker_TimeoutMultiplier(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
			Timeout:              100 * time.Millisecond,
		},
		TimeoutMultiplier: 2.0,
		MaxTimeout:        10 * time.Second,
	})

	// Trip 1: cause failures
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}
	time.Sleep(20 * time.Millisecond) // wait for async callback

	if ab.TripCount() != 1 {
		t.Fatalf("expected 1 trip, got %d", ab.TripCount())
	}

	// Recover: wait for half-open, then succeed
	time.Sleep(120 * time.Millisecond)
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	time.Sleep(20 * time.Millisecond)

	// Trip 2: cause failures again
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}
	time.Sleep(20 * time.Millisecond) // wait for async callback

	params := ab.GetAdaptiveParams()
	if params.TripCount != 2 {
		t.Errorf("expected 2 trips, got %d", params.TripCount)
	}
	// After 2 trips, effective timeout = 100ms * 2^(2-1) = 200ms
	if params.EffectiveTimeout != 200*time.Millisecond {
		t.Errorf("expected effective timeout 200ms, got %v", params.EffectiveTimeout)
	}
}

func TestAdaptiveBreaker_TimeoutMultiplier_Cap(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
			Timeout:              1 * time.Second,
		},
		TimeoutMultiplier: 10.0,
		MaxTimeout:        5 * time.Second,
	})

	// Trip 5 times to exceed max timeout
	for trip := 0; trip < 5; trip++ {
		// Use forced open to avoid waiting for half-open
		ab.ForceOpen()
		ab.Reset()
		for i := 0; i < 3; i++ {
			ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
				return nil, errors.New("failure")
			})
		}
	}

	params := ab.GetAdaptiveParams()
	// Should be capped at MaxTimeout
	if params.EffectiveTimeout > 5*time.Second {
		t.Errorf("effective timeout should be capped at 5s, got %v", params.EffectiveTimeout)
	}
}

func TestAdaptiveBreaker_Reset(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
		},
		ConsecutiveFailureLimit: 3,
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	ab.Reset()

	if ab.State() != StateClosed {
		t.Errorf("expected state CLOSED after reset, got %v", ab.State())
	}
	if ab.TripCount() != 0 {
		t.Errorf("expected trip count 0 after reset, got %d", ab.TripCount())
	}

	params := ab.GetAdaptiveParams()
	if params.ConsecutiveFailures != 0 {
		t.Errorf("expected consecutive failures 0 after reset, got %d", params.ConsecutiveFailures)
	}
}

func TestAdaptiveBreaker_ForceOpen(t *testing.T) {
	ab := NewAdaptive(DefaultAdaptiveConfig("test"))

	ab.ForceOpen()

	if ab.State() != StateForcedOpen {
		t.Errorf("expected state FORCED_OPEN, got %v", ab.State())
	}

	_, err := ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	if err != ErrForcedOpen {
		t.Errorf("expected ErrForcedOpen, got %v", err)
	}
}

func TestAdaptiveBreaker_GetMetrics(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    10,
		},
	})

	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
	for i := 0; i < 2; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	m := ab.GetMetrics()
	if m.TotalSuccesses != 3 {
		t.Errorf("expected 3 successes, got %d", m.TotalSuccesses)
	}
	if m.TotalFailures != 2 {
		t.Errorf("expected 2 failures, got %d", m.TotalFailures)
	}
}

func TestAdaptiveBreaker_StateChangeCallback(t *testing.T) {
	var callbackCalled atomic.Bool

	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    5,
			OnStateChange: func(name string, from, to State) {
				callbackCalled.Store(true)
			},
		},
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	time.Sleep(10 * time.Millisecond)

	if !callbackCalled.Load() {
		t.Error("expected state change callback to be called")
	}
}

func TestAdaptiveBreaker_HalfOpenTransition(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 50,
			MinimumNumberOfCalls: 3,
			SlidingWindowSize:    5,
			Timeout:              100 * time.Millisecond,
			MaxRequests:          1,
		},
	})

	// Trip the breaker
	for i := 0; i < 3; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return nil, errors.New("failure")
		})
	}

	if ab.State() != StateOpen {
		t.Errorf("expected state OPEN, got %v", ab.State())
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	if ab.State() != StateHalfOpen {
		t.Errorf("expected state HALF_OPEN, got %v", ab.State())
	}

	// Successful call should close it
	ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
		return "ok", nil
	})
	time.Sleep(10 * time.Millisecond)

	if ab.State() != StateClosed {
		t.Errorf("expected state CLOSED after half-open success, got %v", ab.State())
	}
}

func TestAdaptiveBreaker_EMAUpdates_MultipleRounds(t *testing.T) {
	ab := NewAdaptive(AdaptiveConfig{
		Base: Config{
			Name:                 "test",
			FailureRateThreshold: 100, // Don't trip
			MinimumNumberOfCalls: 100,
			SlidingWindowSize:    200,
		},
		FailureRateEMAAlpha: 0.5,
	})

	// Round 1: 30% failure rate
	for i := 0; i < 10; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			if i < 3 {
				return nil, errors.New("failure")
			}
			return "ok", nil
		})
	}
	ab.UpdateEMAs()

	params1 := ab.GetAdaptiveParams()
	ema1 := params1.FailureRateEMA

	// Round 2: 0% failure rate
	for i := 0; i < 10; i++ {
		ab.Execute(context.Background(), func(ctx context.Context) (any, error) {
			return "ok", nil
		})
	}
	ab.UpdateEMAs()

	params2 := ab.GetAdaptiveParams()
	ema2 := params2.FailureRateEMA

	// EMA should decrease after all successes
	if ema2 >= ema1 {
		t.Errorf("EMA should decrease after all successes: ema1=%f, ema2=%f", ema1, ema2)
	}
}

func TestAdaptiveBreaker_NoWindowData(t *testing.T) {
	ab := NewAdaptive(DefaultAdaptiveConfig("test"))

	// UpdateEMAs with no data should be a no-op
	ab.UpdateEMAs()

	params := ab.GetAdaptiveParams()
	if params.FailureRateEMA != 0 {
		t.Errorf("expected 0 EMA with no data, got %f", params.FailureRateEMA)
	}
}

func TestAdaptiveBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultAdaptiveConfig("my-breaker")

	if cfg.Base.Name != "my-breaker" {
		t.Errorf("expected base name 'my-breaker', got '%s'", cfg.Base.Name)
	}
	if cfg.FailureRateEMAAlpha != 0.5 {
		t.Errorf("expected EMA alpha 0.5, got %f", cfg.FailureRateEMAAlpha)
	}
	if cfg.LatencyEMAAlpha != 0.3 {
		t.Errorf("expected latency alpha 0.3, got %f", cfg.LatencyEMAAlpha)
	}
	if cfg.MinFailureRateThreshold != 20 {
		t.Errorf("expected min threshold 20, got %f", cfg.MinFailureRateThreshold)
	}
	if cfg.MaxFailureRateThreshold != 80 {
		t.Errorf("expected max threshold 80, got %f", cfg.MaxFailureRateThreshold)
	}
}
