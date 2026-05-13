// Package ratelimit provides rate limiting for Go applications.
//
// Supported algorithms:
//   - Token Bucket: Smooth rate limiting with burst support
//   - Sliding Window: Precise rate limiting over a time window
//
// Example:
//
//    limiter := ratelimit.NewTokenBucket(100, 200) // 100 req/s, burst 200
//    if !limiter.Allow() {
//        return ErrRateLimited
//    }
//    // or with context waiting:
//    if err := limiter.Wait(ctx); err != nil {
//        return err
//    }
package ratelimit

import (
    "context"
    "time"
    
    "github.com/atop0914/goshield/internal/ratelimit"
)

// Limiter defines the interface for rate limiters.
type Limiter = ratelimit.Limiter

// Reservation indicates when an event can happen.
type Reservation = ratelimit.Reservation

// ErrRateLimitExceeded is returned when the rate limit is exceeded.
var ErrRateLimitExceeded = ratelimit.ErrRateLimitExceeded

// TokenBucket implements the token bucket rate limiter.
type TokenBucket = ratelimit.TokenBucket

// NewTokenBucket creates a new token bucket rate limiter.
func NewTokenBucket(rate float64, burst int) *TokenBucket {
    return ratelimit.NewTokenBucket(rate, burst)
}

// SlidingWindow implements a sliding window rate limiter.
type SlidingWindow = ratelimit.SlidingWindow

// NewSlidingWindow creates a new sliding window rate limiter.
func NewSlidingWindow(limit int, window time.Duration) *SlidingWindow {
    return ratelimit.NewSlidingWindow(limit, window)
}

// Allow is a convenience function that checks a limiter and returns an error if not allowed.
func Allow(limiter Limiter) error {
    if !limiter.Allow() {
        return ErrRateLimitExceeded
    }
    return nil
}

// Wait is a convenience function that waits for a limiter.
func Wait(ctx context.Context, limiter Limiter) error {
    return limiter.Wait(ctx)
}
