package helpers

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sony/gobreaker/v2"
)

const (
	defaultFailureRateThreshold      = 50.0
	defaultMinimumRequests           = uint32(2)
	defaultSlidingWindowInterval     = 10 * time.Second
	defaultSlidingWindowBucketPeriod = time.Second
	defaultWaitDurationInOpenState   = time.Second
	defaultPermittedCallsInHalfOpen  = uint32(1)
	percentMultiplier                = 100.0
)

type CircuitBreakerConfig struct {
	Enabled                       bool
	FailureRateThreshold          float64
	MinimumRequests               uint32
	SlidingWindowInterval         time.Duration
	SlidingWindowBucketPeriod     time.Duration
	WaitDurationInOpenState       time.Duration
	PermittedCallsInHalfOpenState uint32
}

func NewCircuitBreaker(
	name string,
	cfg CircuitBreakerConfig,
) *gobreaker.CircuitBreaker[struct{}] {
	if !cfg.Enabled {
		return nil
	}

	cfg = normalizeCircuitBreakerConfig(cfg)

	var state atomic.Int32
	state.Store(int32(gobreaker.StateClosed))

	var halfOpenMu sync.Mutex
	var halfOpenRequests uint32
	var halfOpenFailures uint32

	resetHalfOpenCounters := func() {
		halfOpenMu.Lock()
		defer halfOpenMu.Unlock()

		halfOpenRequests = 0
		halfOpenFailures = 0
	}

	settings := gobreaker.Settings{
		Name:         name,
		MaxRequests:  cfg.PermittedCallsInHalfOpenState,
		Interval:     cfg.SlidingWindowInterval,
		BucketPeriod: cfg.SlidingWindowBucketPeriod,
		Timeout:      cfg.WaitDurationInOpenState,

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.MinimumRequests {
				return false
			}

			failureRate := float64(counts.TotalFailures) / float64(counts.Requests) * percentMultiplier

			return failureRate >= cfg.FailureRateThreshold
		},

		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			state.Store(int32(to))

			if to == gobreaker.StateHalfOpen || from == gobreaker.StateHalfOpen {
				resetHalfOpenCounters()
			}
		},

		IsSuccessful: func(err error) bool {
			currentState := gobreaker.State(state.Load())

			if currentState != gobreaker.StateHalfOpen {
				return err == nil
			}

			halfOpenMu.Lock()
			defer halfOpenMu.Unlock()

			halfOpenRequests++
			if err != nil {
				halfOpenFailures++
			}

			if halfOpenRequests < cfg.PermittedCallsInHalfOpenState {
				return true
			}

			failureRate := float64(halfOpenFailures) / float64(halfOpenRequests) * percentMultiplier

			return failureRate <= cfg.FailureRateThreshold
		},
	}

	return gobreaker.NewCircuitBreaker[struct{}](settings)
}

func DoWithCircuitBreaker(
	cb *gobreaker.CircuitBreaker[struct{}],
	operation func() error,
) error {
	if cb == nil {
		return operation()
	}

	_, err := cb.Execute(func() (struct{}, error) {
		return struct{}{}, operation()
	})

	return err
}

func normalizeCircuitBreakerConfig(cfg CircuitBreakerConfig) CircuitBreakerConfig {
	if cfg.FailureRateThreshold <= 0 {
		cfg.FailureRateThreshold = defaultFailureRateThreshold
	}

	if cfg.MinimumRequests == 0 {
		cfg.MinimumRequests = defaultMinimumRequests
	}

	if cfg.SlidingWindowInterval <= 0 {
		cfg.SlidingWindowInterval = defaultSlidingWindowInterval
	}

	if cfg.SlidingWindowBucketPeriod <= 0 {
		cfg.SlidingWindowBucketPeriod = defaultSlidingWindowBucketPeriod
	}

	if cfg.WaitDurationInOpenState <= 0 {
		cfg.WaitDurationInOpenState = defaultWaitDurationInOpenState
	}

	if cfg.PermittedCallsInHalfOpenState == 0 {
		cfg.PermittedCallsInHalfOpenState = defaultPermittedCallsInHalfOpen
	}

	return cfg
}
