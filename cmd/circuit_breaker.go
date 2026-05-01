package cmd

import (
	"fmt"
	"sort"
	"sync"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// CircuitBreaker tracks consecutive failures per worker instance and prevents
// dispatch to workers that exceed the failure threshold.
// Per D-04: consecutive failure count triggers the breaker (configurable, default 3).
// Per D-07: per-worker instance granularity.
// Per D-06: per-wave reset via Reset().
// All methods are goroutine-safe.
type CircuitBreaker struct {
	mu        sync.Mutex
	threshold int
	failures  map[string]int  // workerName -> consecutive failure count
	tripped   map[string]bool // workerName -> tripped state
}

// NewCircuitBreaker creates a circuit breaker with the given failure threshold.
// Thresholds below 1 are clamped to 3.
func NewCircuitBreaker(threshold int) *CircuitBreaker {
	if threshold < 1 {
		threshold = 3
	}
	return &CircuitBreaker{
		threshold: threshold,
		failures:  make(map[string]int),
		tripped:   make(map[string]bool),
	}
}

// Allow returns false if the worker has been tripped (consecutive failures >= threshold).
func (cb *CircuitBreaker) Allow(workerName string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return !cb.tripped[workerName]
}

// RecordSuccess resets the failure counter for a worker. Per D-04: a single success resets.
func (cb *CircuitBreaker) RecordSuccess(workerName string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[workerName] = 0
	cb.tripped[workerName] = false
}

// RecordFailure increments the failure counter. Returns true if the worker just tripped.
func (cb *CircuitBreaker) RecordFailure(workerName string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures[workerName]++
	if cb.failures[workerName] >= cb.threshold {
		cb.tripped[workerName] = true
		return true
	}
	return false
}

// Reset clears all breaker state. Call at the start of each wave (per D-06).
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = make(map[string]int)
	cb.tripped = make(map[string]bool)
}

// FailureCount returns the current consecutive failure count for a worker.
func (cb *CircuitBreaker) FailureCount(workerName string) int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.failures[workerName]
}

// TrippedWorkers returns the names of all currently tripped workers, sorted.
func (cb *CircuitBreaker) TrippedWorkers() []string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	var names []string
	for name, tripped := range cb.tripped {
		if tripped {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// findSameCastePeer finds a non-tripped worker of the same caste for task redistribution.
// Per D-05: redistributes to same-caste peer. Per research Pitfall 3: peer must not be tripped.
// Returns nil if no suitable peer exists.
func findSameCastePeer(dispatches []codex.WorkerDispatch, current codex.WorkerDispatch, cb *CircuitBreaker) *codex.WorkerDispatch {
	for i := range dispatches {
		d := &dispatches[i]
		if d.WorkerName == current.WorkerName {
			continue
		}
		if d.Caste != current.Caste {
			continue
		}
		if !cb.Allow(d.WorkerName) {
			continue
		}
		return d
	}
	return nil
}

// emitCircuitBreakerTripped publishes a circuit breaker trip event via the ceremony event bus.
func (cb *CircuitBreaker) emitCircuitBreakerTripped(phase colony.Phase, wave int, workerName string) {
	cb.mu.Lock()
	count := cb.failures[workerName]
	threshold := cb.threshold
	cb.mu.Unlock()

	emitBuildCeremonyCircuitBreak(phase, wave, CircuitBreakerEvent{
		WorkerName: workerName,
		Event:      "tripped",
		Reason:     fmt.Sprintf("after %d consecutive failures (threshold: %d)", count, threshold),
	})

	emitLoopBreakEvent("circuit_break",
		fmt.Sprintf("%d consecutive worker failures (threshold: %d)", count, threshold),
		fmt.Sprintf("circuit breaker tripped for %s", workerName),
		"aether-build")
}

// emitCircuitBreakerRedistributed publishes a circuit breaker redistribution event via the ceremony event bus.
func emitCircuitBreakerRedistributed(phase colony.Phase, wave int, fromWorker, toWorker string) {
	emitBuildCeremonyCircuitBreak(phase, wave, CircuitBreakerEvent{
		WorkerName: fromWorker,
		Event:      "skipped",
		Reason:     fmt.Sprintf("redistributing to %s", toWorker),
		PeerName:   toWorker,
	})
}

// emitCircuitBreakerNoPeer publishes a circuit breaker no-peer event via the ceremony event bus.
func emitCircuitBreakerNoPeer(phase colony.Phase, wave int, workerName string) {
	emitBuildCeremonyCircuitBreak(phase, wave, CircuitBreakerEvent{
		WorkerName: workerName,
		Event:      "skipped",
		Reason:     "no same-caste peer available for redistribution",
	})
}

// CircuitBreakerEvent describes a circuit breaker state change for ceremony output.
type CircuitBreakerEvent struct {
	WorkerName string
	Event      string // "tripped", "skipped", "redistributed"
	Reason     string
	PeerName   string // set when Event == "redistributed"
}

// String returns a human-readable description of the circuit breaker event.
func (e CircuitBreakerEvent) String() string {
	switch e.Event {
	case "tripped":
		return fmt.Sprintf("Circuit breaker: %s tripped (%s)", e.WorkerName, e.Reason)
	case "skipped":
		if e.PeerName != "" {
			return fmt.Sprintf("Circuit breaker: %s skipped -- redistributing to %s (%s)", e.WorkerName, e.PeerName, e.Reason)
		}
		return fmt.Sprintf("Circuit breaker: %s skipped -- no peer available (%s)", e.WorkerName, e.Reason)
	case "redistributed":
		return fmt.Sprintf("Circuit breaker: redistributed %s task to %s", e.WorkerName, e.PeerName)
	default:
		return fmt.Sprintf("Circuit breaker: %s %s", e.WorkerName, e.Event)
	}
}
