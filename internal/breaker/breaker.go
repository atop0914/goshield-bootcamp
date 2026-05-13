// Package breaker implements the circuit breaker pattern.
package breaker

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed     State = iota
	StateOpen
	StateHalfOpen
	StateDisabled
	StateForcedOpen
)

var (
	ErrOpenState       = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests in half-open state")
	ErrForcedOpen      = errors.New("circuit breaker is forced open")
)

// Config holds the configuration for a circuit breaker.
type Config struct {
	Name                  string
	MaxRequests           uint32
	Interval              time.Duration
	Timeout               time.Duration
	FailureRateThreshold  uint8
	SlowCallRateThreshold uint8
	SlowCallDuration      time.Duration
	MinimumNumberOfCalls  uint32
	SlidingWindowSize     uint32
	SlidingWindowType     SlidingWindowType
	OnStateChange         func(name string, from, to State)
	IsSuccessful          func(err error) bool
}

type SlidingWindowType int

const (
	SlidingWindowCount SlidingWindowType = iota
	SlidingWindowTime
)

func DefaultConfig(name string) Config {
	return Config{
		Name:                  name,
		MaxRequests:           1,
		Timeout:               60 * time.Second,
		FailureRateThreshold:  50,
		SlowCallRateThreshold: 100,
		MinimumNumberOfCalls:  10,
		SlidingWindowSize:     100,
		SlidingWindowType:     SlidingWindowCount,
	}
}

type CircuitBreaker struct {
	config Config
	state  State
	mutex  sync.RWMutex

	successes  uint32
	failures   uint32
	slowCalls  uint32
	totalCalls uint32

	ringBuffer []callResult
	ringIndex  int
	ringFull   bool

	halfOpenAllowed uint32
	halfOpenCount   uint32

	openSince time.Time

	totalSuccesses   uint64
	totalFailures    uint64
	totalRejected    uint64
	totalSlowCalls   uint64
	totalCallsAll    uint64
	stateTransitions uint64
}

type callResult struct {
	success bool
	slow    bool
	time    time.Time
}

func New(config Config) *CircuitBreaker {
	if config.MaxRequests == 0 {
		config.MaxRequests = 1
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}
	if config.FailureRateThreshold == 0 {
		config.FailureRateThreshold = 50
	}
	if config.SlowCallRateThreshold == 0 {
		config.SlowCallRateThreshold = 100
	}
	if config.MinimumNumberOfCalls == 0 {
		config.MinimumNumberOfCalls = 10
	}
	if config.SlidingWindowSize == 0 {
		config.SlidingWindowSize = 100
	}

	cb := &CircuitBreaker{
		config: config,
		state:  StateClosed,
	}

	if config.SlidingWindowType == SlidingWindowCount {
		cb.ringBuffer = make([]callResult, config.SlidingWindowSize)
	}

	return cb
}

func (cb *CircuitBreaker) Name() string { return cb.config.Name }

func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.getStateUnsafe()
}

// getStateUnsafe returns the current state, performing lazy transition to half-open.
// Must be called with mutex held.
func (cb *CircuitBreaker) getStateUnsafe() State {
	if cb.state == StateOpen && time.Since(cb.openSince) >= cb.config.Timeout {
		// Lazy transition to half-open
		cb.state = StateHalfOpen
		cb.halfOpenAllowed = cb.config.MaxRequests
		cb.halfOpenCount = 0
		cb.totalCalls = 0
		cb.successes = 0
		cb.failures = 0
		cb.slowCalls = 0
		if cb.config.SlidingWindowType == SlidingWindowCount {
			cb.ringBuffer = make([]callResult, cb.config.SlidingWindowSize)
			cb.ringIndex = 0
			cb.ringFull = false
		}
	}
	return cb.state
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(ctx context.Context) (any, error)) (any, error) {
	err := cb.beforeCall()
	if err != nil {
		cb.totalRejected++
		return nil, err
	}

	start := time.Now()
	result, err := fn(ctx)
	duration := time.Since(start)

	cb.afterCall(err, duration)

	return result, err
}

func (cb *CircuitBreaker) beforeCall() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	state := cb.getStateUnsafe()

	switch state {
	case StateClosed:
		return nil
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.halfOpenAllowed {
			return ErrTooManyRequests
		}
		cb.halfOpenCount++
		return nil
	case StateOpen:
		return ErrOpenState
	case StateForcedOpen:
		return ErrForcedOpen
	default:
		return nil
	}
}

func (cb *CircuitBreaker) afterCall(err error, duration time.Duration) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.totalCallsAll++

	isFailure := err != nil
	if cb.config.IsSuccessful != nil {
		isFailure = !cb.config.IsSuccessful(err)
	}

	isSlow := false
	if cb.config.SlowCallDuration > 0 {
		isSlow = duration > cb.config.SlowCallDuration
	}

	if isFailure {
		cb.totalFailures++
	} else {
		cb.totalSuccesses++
	}

	if isSlow {
		cb.totalSlowCalls++
	}

	state := cb.getStateUnsafe()

	switch state {
	case StateClosed:
		cb.recordResult(isFailure, isSlow)
		cb.checkThresholds()
	case StateHalfOpen:
		cb.recordResult(isFailure, isSlow)
		if isFailure {
			cb.transitionTo(StateOpen)
		} else {
			cb.checkHalfOpenSuccess()
		}
	}
}

func (cb *CircuitBreaker) recordResult(isFailure, isSlow bool) {
	result := callResult{
		success: !isFailure,
		slow:    isSlow,
		time:    time.Now(),
	}

	if cb.config.SlidingWindowType == SlidingWindowCount {
		cb.ringBuffer[cb.ringIndex] = result
		cb.ringIndex = (cb.ringIndex + 1) % len(cb.ringBuffer)
		if cb.ringIndex == 0 {
			cb.ringFull = true
		}
	}

	cb.totalCalls++
	if isFailure {
		cb.failures++
	} else {
		cb.successes++
	}
	if isSlow {
		cb.slowCalls++
	}
}

func (cb *CircuitBreaker) checkThresholds() {
	if cb.totalCalls < cb.config.MinimumNumberOfCalls {
		return
	}

	failureRate := float64(cb.failures) / float64(cb.totalCalls) * 100

	if cb.config.SlowCallDuration > 0 {
		slowCallRate := float64(cb.slowCalls) / float64(cb.totalCalls) * 100
		if slowCallRate >= float64(cb.config.SlowCallRateThreshold) {
			cb.transitionTo(StateOpen)
			return
		}
	}

	if failureRate >= float64(cb.config.FailureRateThreshold) {
		cb.transitionTo(StateOpen)
	}
}

func (cb *CircuitBreaker) checkHalfOpenSuccess() {
	if cb.totalCalls >= cb.halfOpenAllowed {
		cb.transitionTo(StateClosed)
	}
}

func (cb *CircuitBreaker) transitionTo(newState State) {
	oldState := cb.state
	if oldState == newState {
		return
	}

	cb.state = newState
	cb.stateTransitions++

	cb.totalCalls = 0
	cb.successes = 0
	cb.failures = 0
	cb.slowCalls = 0

	if cb.config.SlidingWindowType == SlidingWindowCount {
		cb.ringBuffer = make([]callResult, cb.config.SlidingWindowSize)
		cb.ringIndex = 0
		cb.ringFull = false
	}

	switch newState {
	case StateOpen:
		cb.openSince = time.Now()
	case StateHalfOpen:
		cb.halfOpenAllowed = cb.config.MaxRequests
		cb.halfOpenCount = 0
	}

	if cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, newState)
	}
}

func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateClosed
	cb.totalCalls = 0
	cb.successes = 0
	cb.failures = 0
	cb.slowCalls = 0
	cb.halfOpenCount = 0

	if cb.config.SlidingWindowType == SlidingWindowCount {
		cb.ringBuffer = make([]callResult, cb.config.SlidingWindowSize)
		cb.ringIndex = 0
		cb.ringFull = false
	}

	if oldState != StateClosed && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, StateClosed)
	}
}

func (cb *CircuitBreaker) ForceOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	oldState := cb.state
	cb.state = StateForcedOpen

	if oldState != StateForcedOpen && cb.config.OnStateChange != nil {
		go cb.config.OnStateChange(cb.config.Name, oldState, StateForcedOpen)
	}
}

type Metrics struct {
	FailureRate    float64
	SlowCallRate   float64
	TotalCalls     uint32
	TotalSuccesses uint64
	TotalFailures  uint64
	TotalRejected  uint64
	TotalSlowCalls uint64
	StateTransitions uint64
	State          State
}

func (cb *CircuitBreaker) GetMetrics() Metrics {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	m := Metrics{
		TotalSuccesses:   cb.totalSuccesses,
		TotalFailures:    cb.totalFailures,
		TotalRejected:    cb.totalRejected,
		TotalSlowCalls:   cb.totalSlowCalls,
		StateTransitions: cb.stateTransitions,
		State:            cb.getStateUnsafe(),
		TotalCalls:       cb.totalCalls,
	}

	if cb.totalCalls > 0 {
		m.FailureRate = float64(cb.failures) / float64(cb.totalCalls) * 100
		m.SlowCallRate = float64(cb.slowCalls) / float64(cb.totalCalls) * 100
	}

	return m
}
