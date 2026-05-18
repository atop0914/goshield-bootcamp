// Package bulkhead provides the bulkhead pattern for limiting concurrent executions.
package bulkhead

import (
    "context"
    "errors"
    "sync"
    "time"
)

var (
    // ErrBulkheadFull is returned when the bulkhead is at capacity.
    ErrBulkheadFull = errors.New("bulkhead is full")
    // ErrBulkheadTimeout is returned when waiting for a slot times out.
    ErrBulkheadTimeout = errors.New("bulkhead wait timeout")
)

// Config holds the configuration for a bulkhead.
type Config struct {
    // MaxConcurrent is the maximum number of concurrent executions.
    MaxConcurrent int
    // MaxWaitDuration is the maximum time to wait for a slot.
    // If 0, the bulkhead will not wait and return ErrBulkheadFull immediately.
    MaxWaitDuration time.Duration
    // OnCallRejected is called when a call is rejected.
    OnCallRejected func()
}

// Bulkhead limits the number of concurrent executions.
type Bulkhead struct {
    config    Config
    semaphore chan struct{}
    mutex     sync.RWMutex
    
    // Metrics
    activeCount    int
    totalCalls     uint64
    totalRejected  uint64
    totalCompleted uint64
}

// New creates a new Bulkhead with the given configuration.
func New(config Config) *Bulkhead {
    if config.MaxConcurrent <= 0 {
        config.MaxConcurrent = 10
    }
    
    return &Bulkhead{
        config:    config,
        semaphore: make(chan struct{}, config.MaxConcurrent),
    }
}

// Name returns the name of this bulkhead.
func (b *Bulkhead) Name() string {
    return "bulkhead"
}

// Execute wraps a function with bulkhead protection.
func (b *Bulkhead) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
    // Try to acquire a slot
    if err := b.acquire(ctx); err != nil {
        b.totalRejected++
        if b.config.OnCallRejected != nil {
            go b.config.OnCallRejected()
        }
        return nil, err
    }
    defer b.release()
    
    b.totalCalls++
    return fn(ctx)
}

func (b *Bulkhead) acquire(ctx context.Context) error {
    if b.config.MaxWaitDuration <= 0 {
        // Non-blocking acquire
        select {
        case b.semaphore <- struct{}{}:
            b.mutex.Lock()
            b.activeCount++
            b.mutex.Unlock()
            return nil
        default:
            return ErrBulkheadFull
        }
    }
    
    // Blocking acquire with timeout
    timer := time.NewTimer(b.config.MaxWaitDuration)
    defer timer.Stop()
    
    select {
    case b.semaphore <- struct{}{}:
        b.mutex.Lock()
        b.activeCount++
        b.mutex.Unlock()
        return nil
    case <-timer.C:
        return ErrBulkheadTimeout
    case <-ctx.Done():
        return ctx.Err()
    }
}

func (b *Bulkhead) release() {
    <-b.semaphore
    b.mutex.Lock()
    b.activeCount--
    b.totalCompleted++
    b.mutex.Unlock()
}

// Metrics returns the current metrics of the bulkhead.
type Metrics struct {
    // ActiveCount is the number of currently active executions.
    ActiveCount int
    // AvailableSlots is the number of available slots.
    AvailableSlots int
    // TotalCalls is the total number of calls.
    TotalCalls uint64
    // TotalRejected is the total number of rejected calls.
    TotalRejected uint64
    // TotalCompleted is the total number of completed calls.
    TotalCompleted uint64
}

// GetMetrics returns the current metrics.
func (b *Bulkhead) GetMetrics() Metrics {
    b.mutex.RLock()
    defer b.mutex.RUnlock()
    
    return Metrics{
        ActiveCount:    b.activeCount,
        AvailableSlots: b.config.MaxConcurrent - b.activeCount,
        TotalCalls:     b.totalCalls,
        TotalRejected:  b.totalRejected,
        TotalCompleted: b.totalCompleted,
    }
}

// GetMetricsForCollection returns raw int64 metrics for the metrics collector.
func (b *Bulkhead) GetMetricsForCollection() (available, maxConcurrent, totalExecutions, totalRejections, currentRunning int64) {
    b.mutex.RLock()
    defer b.mutex.RUnlock()
    return int64(b.config.MaxConcurrent - b.activeCount),
        int64(b.config.MaxConcurrent),
        int64(b.totalCalls),
        int64(b.totalRejected),
        int64(b.activeCount)
}
