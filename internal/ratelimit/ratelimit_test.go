package ratelimit

import (
    "context"
    "testing"
    "time"
)

func TestTokenBucket_Allow(t *testing.T) {
    tb := NewTokenBucket(10, 10) // 10 tokens/sec, burst 10
    
    // Should allow up to burst
    for i := 0; i < 10; i++ {
        if !tb.Allow() {
            t.Fatalf("expected allow at attempt %d", i)
        }
    }
    
    // Should reject after burst
    if tb.Allow() {
        t.Fatal("expected reject after burst exhausted")
    }
}

func TestTokenBucket_Replenish(t *testing.T) {
    tb := NewTokenBucket(100, 10) // 100 tokens/sec, burst 10
    
    // Exhaust burst
    for i := 0; i < 10; i++ {
        tb.Allow()
    }
    
    // Wait for replenishment
    time.Sleep(60 * time.Millisecond)
    
    // Should have ~6 tokens
    if !tb.Allow() {
        t.Fatal("expected allow after replenishment")
    }
}

func TestTokenBucket_Wait(t *testing.T) {
    tb := NewTokenBucket(10, 1) // 10 tokens/sec, burst 1
    
    // Use the burst token
    tb.Allow()
    
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    
    start := time.Now()
    err := tb.Wait(ctx)
    elapsed := time.Since(start)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    // Should have waited ~100ms
    if elapsed < 50*time.Millisecond {
        t.Fatalf("expected wait ~100ms, got %v", elapsed)
    }
}

func TestTokenBucket_WaitContextCancel(t *testing.T) {
    tb := NewTokenBucket(1, 1) // Very slow rate
    
    tb.Allow()
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
    defer cancel()
    
    err := tb.Wait(ctx)
    if err != context.DeadlineExceeded {
        t.Fatalf("expected DeadlineExceeded, got %v", err)
    }
}

func TestSlidingWindow_Allow(t *testing.T) {
    sw := NewSlidingWindow(5, 1*time.Second)
    
    // Should allow up to limit
    for i := 0; i < 5; i++ {
        if !sw.Allow() {
            t.Fatalf("expected allow at attempt %d", i)
        }
    }
    
    // Should reject after limit
    if sw.Allow() {
        t.Fatal("expected reject after limit reached")
    }
}

func TestSlidingWindow_WindowExpiry(t *testing.T) {
    sw := NewSlidingWindow(5, 100*time.Millisecond)
    
    // Exhaust the window
    for i := 0; i < 5; i++ {
        sw.Allow()
    }
    
    // Wait for window to expire
    time.Sleep(150 * time.Millisecond)
    
    // Should allow again
    if !sw.Allow() {
        t.Fatal("expected allow after window expiry")
    }
}

func TestSlidingWindow_Wait(t *testing.T) {
    sw := NewSlidingWindow(1, 100*time.Millisecond)
    
    sw.Allow()
    
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()
    
    start := time.Now()
    err := sw.Wait(ctx)
    elapsed := time.Since(start)
    
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    
    if elapsed < 50*time.Millisecond {
        t.Fatalf("expected wait ~100ms, got %v", elapsed)
    }
}

func TestTokenBucket_Reserve(t *testing.T) {
    tb := NewTokenBucket(10, 5)
    
    // Reserve should succeed immediately
    r := tb.Reserve()
    if !r.OK {
        t.Fatal("expected OK reservation")
    }
    if r.Delay > 0 {
        t.Fatalf("expected no delay, got %v", r.Delay)
    }
}

func TestTokenBucket_Rate(t *testing.T) {
    tb := NewTokenBucket(42, 100)
    if tb.Rate() != 42 {
        t.Fatalf("expected rate 42, got %f", tb.Rate())
    }
    if tb.Burst() != 100 {
        t.Fatalf("expected burst 100, got %d", tb.Burst())
    }
}
