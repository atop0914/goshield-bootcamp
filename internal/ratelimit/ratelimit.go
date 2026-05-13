// Package ratelimit provides rate limiting implementations.
package ratelimit

import (
    "context"
    "errors"
    "sync"
    "time"
)

var (
    // ErrRateLimitExceeded is returned when the rate limit is exceeded.
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// Limiter defines the interface for rate limiters.
type Limiter interface {
    // Allow reports whether an event may happen now.
    Allow() bool
    // Wait blocks until the rate limit allows an event or the context is canceled.
    Wait(ctx context.Context) error
    // Reserve returns a Reservation that indicates how long the caller must wait.
    Reserve() *Reservation
    // Rate returns the configured rate limit.
    Rate() float64
    // Burst returns the configured burst size.
    Burst() int
}

// Reservation indicates when an event can happen.
type Reservation struct {
    // OK is true if the reservation was successful.
    OK bool
    // Delay is how long the caller must wait before the event can happen.
    Delay time.Duration
    // TimeToAct is when the event can happen.
    TimeToAct time.Time
}

// TokenBucket implements the token bucket rate limiter.
type TokenBucket struct {
    rate     float64 // tokens per second
    burst    int     // max tokens
    tokens   float64 // current tokens
    lastTime time.Time
    mutex    sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter.
// rate is the number of tokens added per second.
// burst is the maximum number of tokens.
func NewTokenBucket(rate float64, burst int) *TokenBucket {
    return &TokenBucket{
        rate:     rate,
        burst:    burst,
        tokens:   float64(burst),
        lastTime: time.Now(),
    }
}

func (tb *TokenBucket) Allow() bool {
    return tb.AllowN(1)
}

func (tb *TokenBucket) AllowN(n int) bool {
    tb.mutex.Lock()
    defer tb.mutex.Unlock()
    
    tb.refreshTokens()
    
    if tb.tokens >= float64(n) {
        tb.tokens -= float64(n)
        return true
    }
    return false
}

func (tb *TokenBucket) Wait(ctx context.Context) error {
    return tb.WaitN(ctx, 1)
}

func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
    for {
        tb.mutex.Lock()
        tb.refreshTokens()
        
        if tb.tokens >= float64(n) {
            tb.tokens -= float64(n)
            tb.mutex.Unlock()
            return nil
        }
        
        // Calculate how long to wait
        deficit := float64(n) - tb.tokens
        waitTime := time.Duration(deficit / tb.rate * float64(time.Second))
        tb.mutex.Unlock()
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(waitTime):
            // Try again
        }
    }
}

func (tb *TokenBucket) Reserve() *Reservation {
    return tb.ReserveN(1)
}

func (tb *TokenBucket) ReserveN(n int) *Reservation {
    tb.mutex.Lock()
    defer tb.mutex.Unlock()
    
    tb.refreshTokens()
    
    if tb.tokens >= float64(n) {
        tb.tokens -= float64(n)
        return &Reservation{
            OK:        true,
            TimeToAct: time.Now(),
        }
    }
    
    deficit := float64(n) - tb.tokens
    delay := time.Duration(deficit / tb.rate * float64(time.Second))
    
    return &Reservation{
        OK:        true,
        Delay:     delay,
        TimeToAct: time.Now().Add(delay),
    }
}

func (tb *TokenBucket) Rate() float64 {
    return tb.rate
}

func (tb *TokenBucket) Burst() int {
    return tb.burst
}

func (tb *TokenBucket) refreshTokens() {
    now := time.Now()
    elapsed := now.Sub(tb.lastTime).Seconds()
    tb.lastTime = now
    
    tb.tokens += elapsed * tb.rate
    if tb.tokens > float64(tb.burst) {
        tb.tokens = float64(tb.burst)
    }
}

// SlidingWindow implements a sliding window rate limiter.
type SlidingWindow struct {
    limit    int
    window   time.Duration
    requests []time.Time
    mutex    sync.Mutex
}

// NewSlidingWindow creates a new sliding window rate limiter.
// limit is the maximum number of requests in the window.
// window is the time window duration.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
    return &SlidingWindow{
        limit:    limit,
        window:   window,
        requests: make([]time.Time, 0, limit),
    }
}

func (sw *SlidingWindow) Allow() bool {
    sw.mutex.Lock()
    defer sw.mutex.Unlock()
    
    sw.cleanup()
    
    if len(sw.requests) < sw.limit {
        sw.requests = append(sw.requests, time.Now())
        return true
    }
    return false
}

func (sw *SlidingWindow) Wait(ctx context.Context) error {
    for {
        sw.mutex.Lock()
        sw.cleanup()
        
        if len(sw.requests) < sw.limit {
            sw.requests = append(sw.requests, time.Now())
            sw.mutex.Unlock()
            return nil
        }
        
        // Calculate wait time
        oldest := sw.requests[0]
        waitTime := sw.window - time.Since(oldest)
        sw.mutex.Unlock()
        
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(waitTime):
            // Try again
        }
    }
}

func (sw *SlidingWindow) Reserve() *Reservation {
    sw.mutex.Lock()
    defer sw.mutex.Unlock()
    
    sw.cleanup()
    
    if len(sw.requests) < sw.limit {
        sw.requests = append(sw.requests, time.Now())
        return &Reservation{
            OK:        true,
            TimeToAct: time.Now(),
        }
    }
    
    oldest := sw.requests[0]
    delay := sw.window - time.Since(oldest)
    
    return &Reservation{
        OK:        true,
        Delay:     delay,
        TimeToAct: time.Now().Add(delay),
    }
}

func (sw *SlidingWindow) Rate() float64 {
    return float64(sw.limit) / sw.window.Seconds()
}

func (sw *SlidingWindow) Burst() int {
    return sw.limit
}

func (sw *SlidingWindow) cleanup() {
    cutoff := time.Now().Add(-sw.window)
    i := 0
    for i < len(sw.requests) && sw.requests[i].Before(cutoff) {
        i++
    }
    sw.requests = sw.requests[i:]
}
