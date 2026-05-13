package bulkhead

import (
    "context"
    "sync"
    "sync/atomic"
    "testing"
    "time"
)

func TestBulkhead_Allow(t *testing.T) {
    bh := New(Config{
        MaxConcurrent: 3,
    })
    
    ctx := context.Background()
    
    // Should allow up to MaxConcurrent
    var wg sync.WaitGroup
    var active atomic.Int32
    
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            _, err := bh.Execute(ctx, func(ctx context.Context) (any, error) {
                active.Add(1)
                time.Sleep(100 * time.Millisecond)
                return nil, nil
            })
            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }
        }()
    }
    
    // Wait a bit for goroutines to start
    time.Sleep(10 * time.Millisecond)
    
    // Should reject when at capacity
    _, err := bh.Execute(ctx, func(ctx context.Context) (any, error) {
        return nil, nil
    })
    if err != ErrBulkheadFull {
        t.Fatalf("expected ErrBulkheadFull, got %v", err)
    }
    
    wg.Wait()
}

func TestBulkhead_WaitTimeout(t *testing.T) {
    bh := New(Config{
        MaxConcurrent:   1,
        MaxWaitDuration: 50 * time.Millisecond,
    })
    
    ctx := context.Background()
    
    // Start a long-running task
    go func() {
        bh.Execute(ctx, func(ctx context.Context) (any, error) {
            time.Sleep(200 * time.Millisecond)
            return nil, nil
        })
    }()
    
    time.Sleep(10 * time.Millisecond)
    
    // Should timeout waiting for slot
    _, err := bh.Execute(ctx, func(ctx context.Context) (any, error) {
        return nil, nil
    })
    if err != ErrBulkheadTimeout {
        t.Fatalf("expected ErrBulkheadTimeout, got %v", err)
    }
}

func TestBulkhead_Metrics(t *testing.T) {
    bh := New(Config{
        MaxConcurrent: 5,
    })
    
    ctx := context.Background()
    
    var wg sync.WaitGroup
    for i := 0; i < 3; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            bh.Execute(ctx, func(ctx context.Context) (any, error) {
                time.Sleep(50 * time.Millisecond)
                return nil, nil
            })
        }()
    }
    
    time.Sleep(10 * time.Millisecond)
    
    metrics := bh.GetMetrics()
    if metrics.ActiveCount != 3 {
        t.Fatalf("expected 3 active, got %d", metrics.ActiveCount)
    }
    if metrics.AvailableSlots != 2 {
        t.Fatalf("expected 2 available slots, got %d", metrics.AvailableSlots)
    }
    
    wg.Wait()
}

func TestBulkhead_RejectedCallback(t *testing.T) {
    var rejected atomic.Int32
    
    bh := New(Config{
        MaxConcurrent: 1,
        OnCallRejected: func() {
            rejected.Add(1)
        },
    })
    
    ctx := context.Background()
    
    // Start a long-running task
    go func() {
        bh.Execute(ctx, func(ctx context.Context) (any, error) {
            time.Sleep(100 * time.Millisecond)
            return nil, nil
        })
    }()
    
    time.Sleep(10 * time.Millisecond)
    
    // This should be rejected
    bh.Execute(ctx, func(ctx context.Context) (any, error) {
        return nil, nil
    })
    
    time.Sleep(10 * time.Millisecond)
    
    if rejected.Load() != 1 {
        t.Fatalf("expected 1 rejected callback, got %d", rejected.Load())
    }
}
