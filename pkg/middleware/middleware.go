// Package middleware provides HTTP middleware for Go's net/http package.
//
// This package integrates all GoShield resilience patterns as standard
// HTTP middleware that can be used with any Go HTTP framework.
//
// Example:
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/api/resource", handler)
//
//	// Apply middleware chain
//	handler := middleware.Chain(
//	    middleware.CircuitBreaker("my-api", breakerCfg),
//	    middleware.RateLimit(limiter),
//	    middleware.Timeout(5*time.Second),
//	    middleware.Bulkhead(bulkheadCfg),
//	)(mux)
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/atop0914/goshield/internal/breaker"
	"github.com/atop0914/goshield/internal/bulkhead"
	"github.com/atop0914/goshield/internal/ratelimit"
)

// Middleware is an HTTP middleware function.
type Middleware func(http.Handler) http.Handler

// Chain composes multiple middleware into a single middleware.
// Middleware are applied in order: the first middleware is the outermost wrapper.
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			next = middlewares[i](next)
		}
		return next
	}
}

// CircuitBreaker returns middleware that applies circuit breaker protection.
func CircuitBreaker(name string, config breaker.Config) Middleware {
	cb := breaker.New(config)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := cb.Execute(r.Context(), func(ctx context.Context) (any, error) {
				// Create a response recorder to capture the status code
				rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
				next.ServeHTTP(rec, r.WithContext(ctx))

				// Consider 5xx errors as failures for the circuit breaker
				if rec.statusCode >= 500 {
					return nil, fmt.Errorf("server error: %d", rec.statusCode)
				}
				return nil, nil
			})

			if err != nil {
				if err == breaker.ErrOpenState || err == breaker.ErrTooManyRequests {
					http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
					return
				}
			}
		})
	}
}

// RateLimit returns middleware that applies rate limiting.
func RateLimit(limiter ratelimit.Limiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limiter.Wait(r.Context()); err != nil {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitWithResponse returns middleware that applies rate limiting with custom response.
func RateLimitWithResponse(limiter ratelimit.Limiter, responseFunc func(w http.ResponseWriter, r *http.Request)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limiter.Wait(r.Context()); err != nil {
				responseFunc(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Timeout returns middleware that enforces a timeout on requests.
func Timeout(duration time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TimeoutWithResponse returns middleware that enforces a timeout with custom response.
func TimeoutWithResponse(duration time.Duration, responseFunc func(w http.ResponseWriter, r *http.Request)) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), duration)
			defer cancel()

			done := make(chan struct{})
			go func() {
				next.ServeHTTP(w, r.WithContext(ctx))
				close(done)
			}()

			select {
			case <-done:
				return
			case <-ctx.Done():
				responseFunc(w, r)
			}
		})
	}
}

// Bulkhead returns middleware that limits concurrent requests.
func Bulkhead(config bulkhead.Config) Middleware {
	bh := bulkhead.New(config)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := bh.Execute(r.Context(), func(ctx context.Context) (any, error) {
				next.ServeHTTP(w, r.WithContext(ctx))
				return nil, nil
			})

			if err != nil {
				if err == bulkhead.ErrBulkheadFull || err == bulkhead.ErrBulkheadTimeout {
					http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
					return
				}
			}
		})
	}
}

// Retry returns middleware that retries failed requests.
func Retry(maxRetries int, backoff time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var lastErr error
			for i := 0; i <= maxRetries; i++ {
				rec := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}
				next.ServeHTTP(rec, r)

				if rec.statusCode < 500 {
					return
				}

				lastErr = fmt.Errorf("server error: %d", rec.statusCode)

				if i < maxRetries {
					time.Sleep(backoff * time.Duration(i+1))
				}
			}

			if lastErr != nil {
				http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			}
		})
	}
}

// ConcurrencyLimit is an alias for Bulkhead for clarity.
func ConcurrencyLimit(maxConcurrent int) Middleware {
	return Bulkhead(bulkhead.Config{MaxConcurrent: maxConcurrent})
}

// responseRecorder captures the status code from the wrapped handler.
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rec *responseRecorder) WriteHeader(code int) {
	if !rec.written {
		rec.statusCode = code
		rec.written = true
		rec.ResponseWriter.WriteHeader(code)
	}
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if !rec.written {
		rec.statusCode = http.StatusOK
		rec.written = true
	}
	return rec.ResponseWriter.Write(b)
}
