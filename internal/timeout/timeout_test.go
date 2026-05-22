package timeout

import (
    "context"
    "sync/atomic"
    "testing"
    "time"
)

func TestExecute_Success(t *testing.T) {
    config := Config{
        Duration: 1 * time.Second,
    }
    
    result, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        return "ok", nil
    })
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "ok" {
        t.Fatalf("expected 'ok', got %v", result)
    }
}

func TestExecute_Timeout(t *testing.T) {
    config := Config{
        Duration: 50 * time.Millisecond,
    }
    
    _, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        time.Sleep(200 * time.Millisecond)
        return "ok", nil
    })
    
    if err != ErrTimeout {
        t.Fatalf("expected ErrTimeout, got %v", err)
    }
}

func TestExecute_NoTimeout(t *testing.T) {
    config := Config{
        Duration: 0, // No timeout
    }
    
    result, err := Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        return "ok", nil
    })
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if result != "ok" {
        t.Fatalf("expected 'ok', got %v", result)
    }
}

func TestExecute_ContextCancel(t *testing.T) {
    config := Config{
        Duration: 1 * time.Second,
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    _, err := Execute(ctx, config, func(ctx context.Context) (any, error) {
        return "ok", nil
    })
    
    if err != context.Canceled {
        t.Fatalf("expected context.Canceled, got %v", err)
    }
}

func TestExecute_OnTimeoutCallback(t *testing.T) {
    var called atomic.Bool
    
    config := Config{
        Duration: 50 * time.Millisecond,
        OnTimeout: func(d time.Duration) {
            called.Store(true)
        },
    }
    
    Execute(context.Background(), config, func(ctx context.Context) (any, error) {
        time.Sleep(200 * time.Millisecond)
        return "ok", nil
    })
    
    // Give callback time to execute
    time.Sleep(10 * time.Millisecond)
    
    if !called.Load() {
        t.Fatal("expected OnTimeout callback to be called")
    }
}
