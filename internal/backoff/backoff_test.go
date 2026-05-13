package backoff

import (
    "context"
    "errors"
    "testing"
    "time"
)

func TestFixedBackoff(t *testing.T) {
    b := &FixedBackoff{
        Interval:   100 * time.Millisecond,
        MaxRetryCount: 3,
    }
    
    for i := 0; i < 3; i++ {
        d := b.Next(i)
        if d != 100*time.Millisecond {
            t.Fatalf("expected 100ms at attempt %d, got %v", i, d)
        }
    }
    
    // Should return -1 after max retries
    if b.Next(3) != -1 {
        t.Fatal("expected -1 after max retries")
    }
}

func TestExponentialBackoff(t *testing.T) {
    b := &ExponentialBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
 MaxRetryCount: 5,
        Multiplier:      2.0,
    }
    
    expected := []time.Duration{
        100 * time.Millisecond,
        200 * time.Millisecond,
        400 * time.Millisecond,
        800 * time.Millisecond,
        1600 * time.Millisecond,
    }
    
    for i, exp := range expected {
        d := b.Next(i)
        if d != exp {
            t.Fatalf("expected %v at attempt %d, got %v", exp, i, d)
        }
    }
}

func TestExponentialBackoff_MaxInterval(t *testing.T) {
    b := &ExponentialBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     500 * time.Millisecond,
MaxRetryCount:   5,
        Multiplier:      2.0,
    }
    
    // At attempt 4, would be 1600ms but capped at 500ms
    d := b.Next(4)
    if d != 500*time.Millisecond {
        t.Fatalf("expected 500ms (max), got %v", d)
    }
}

func TestExponentialRandomBackoff(t *testing.T) {
    b := &ExponentialRandomBackoff{
        InitialInterval: 100 * time.Millisecond,
        MaxInterval:     10 * time.Second,
 MaxRetryCount: 5,
        Multiplier:      2.0,
    }
    
    for i := 0; i < 5; i++ {
        d := b.Next(i)
        if d < 0 {
            t.Fatalf("expected non-negative at attempt %d, got %v", i, d)
        }
        // Should be less than or equal to the non-random exponential
        maxExpected := time.Duration(float64(100*time.Millisecond) * 2.0 * float64(i+1))
        if d > maxExpected*2 { // Allow some variance
            t.Logf("warning: got %v at attempt %d, might be too high", d, i)
        }
    }
}

func TestFibonacciBackoff(t *testing.T) {
	b := &FibonacciBackoff{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     10 * time.Second,
		MaxRetryCount:   6,
	}
	
	// Fibonacci: 1, 2, 3, 5, 8, 13 (using retry+2)
	expected := []time.Duration{
		100 * time.Millisecond,  // fibonacci(2) = 1
		200 * time.Millisecond,  // fibonacci(3) = 2
		300 * time.Millisecond,  // fibonacci(4) = 3
		500 * time.Millisecond,  // fibonacci(5) = 5
		800 * time.Millisecond,  // fibonacci(6) = 8
		1300 * time.Millisecond, // fibonacci(7) = 13
	}
	
	for i, exp := range expected {
		d := b.Next(i)
		if d != exp {
			t.Fatalf("expected %v at attempt %d, got %v", exp, i, d)
		}
	}
}

func TestExecute_Success(t *testing.T) {
    config := RetryConfig{
        Backoff: &FixedBackoff{
            Interval:   10 * time.Millisecond,
            MaxRetryCount: 3,
        },
    }
    
    callCount := 0
    result, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        callCount++
        if callCount < 3 {
            return nil, errors.New("not yet")
        }
        return "success", nil
    })
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "success" {
        t.Fatalf("expected 'success', got %v", result)
    }
    if callCount != 3 {
        t.Fatalf("expected 3 calls, got %d", callCount)
    }
}

func TestExecute_MaxRetriesExceeded(t *testing.T) {
    config := RetryConfig{
        Backoff: &FixedBackoff{
            Interval:   10 * time.Millisecond,
            MaxRetryCount: 3,
        },
    }
    
    callCount := 0
    _, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        callCount++
        return nil, errors.New("always fail")
    })
    
    if !errors.Is(err, ErrMaxRetriesExceeded) {
        t.Fatalf("expected ErrMaxRetriesExceeded, got %v", err)
    }
    if callCount != 4 { // Initial + 3 retries
        t.Fatalf("expected 4 calls, got %d", callCount)
    }
}

func TestExecute_RetryOnFilter(t *testing.T) {
    config := RetryConfig{
        Backoff: &FixedBackoff{
            Interval:   10 * time.Millisecond,
            MaxRetryCount: 3,
        },
        RetryOn: func(err error) bool {
            return !errors.Is(err, context.Canceled)
        },
    }
    
    _, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        return nil, context.Canceled
    })
    
    if !errors.Is(err, context.Canceled) {
        t.Fatalf("expected context.Canceled, got %v", err)
    }
}

func TestExecute_ContextCancel(t *testing.T) {
    config := RetryConfig{
        Backoff: &FixedBackoff{
            Interval:   1 * time.Second,
            MaxRetryCount: 10,
        },
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
    defer cancel()
    
    _, err := Execute(ctx, config, func(ctx context.Context) (any, error) {
        return nil, errors.New("fail")
    })
    
    if err != context.DeadlineExceeded {
        t.Fatalf("expected DeadlineExceeded, got %v", err)
    }
}

func TestExecute_OnRetryCallback(t *testing.T) {
    var retries []int
    var lastErr error
    
    config := RetryConfig{
        Backoff: &FixedBackoff{
            Interval:   10 * time.Millisecond,
            MaxRetryCount: 3,
        },
        OnRetry: func(retry int, err error) {
            retries = append(retries, retry)
            lastErr = err
        },
    }
    
    Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        return nil, errors.New("test error")
    })
    
    if len(retries) != 3 {
        t.Fatalf("expected 3 retries, got %d", len(retries))
    }
    if lastErr.Error() != "test error" {
        t.Fatalf("expected 'test error', got %v", lastErr)
    }
}
