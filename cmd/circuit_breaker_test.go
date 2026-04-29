package cmd

import (
	"sync"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

func TestCircuitBreaker_Trip(t *testing.T) {
	cb := NewCircuitBreaker(3)

	// Two failures should not trip
	cb.RecordFailure("Builder-Mason-67")
	cb.RecordFailure("Builder-Mason-67")
	if !cb.Allow("Builder-Mason-67") {
		t.Fatal("expected Allow=true after 2 failures with threshold 3")
	}

	// Third failure should trip
	tripped := cb.RecordFailure("Builder-Mason-67")
	if !tripped {
		t.Fatal("expected RecordFailure to return true (tripped) after 3 failures")
	}
	if cb.Allow("Builder-Mason-67") {
		t.Fatal("expected Allow=false after breaker tripped")
	}
}

func TestCircuitBreaker_SuccessReset(t *testing.T) {
	cb := NewCircuitBreaker(3)

	cb.RecordFailure("Builder-Mason-67")
	cb.RecordFailure("Builder-Mason-67")
	cb.RecordSuccess("Builder-Mason-67")

	if cb.FailureCount("Builder-Mason-67") != 0 {
		t.Fatalf("expected failure count 0 after success, got %d", cb.FailureCount("Builder-Mason-67"))
	}
	if !cb.Allow("Builder-Mason-67") {
		t.Fatal("expected Allow=true after success reset")
	}
}

func TestCircuitBreaker_WaveReset(t *testing.T) {
	cb := NewCircuitBreaker(3)

	cb.RecordFailure("Builder-Mason-67")
	cb.RecordFailure("Builder-Mason-67")
	cb.RecordFailure("Builder-Mason-67")
	if cb.Allow("Builder-Mason-67") {
		t.Fatal("expected Allow=false before reset")
	}

	cb.Reset()
	if !cb.Allow("Builder-Mason-67") {
		t.Fatal("expected Allow=true after Reset()")
	}
	if cb.FailureCount("Builder-Mason-67") != 0 {
		t.Fatalf("expected failure count 0 after Reset(), got %d", cb.FailureCount("Builder-Mason-67"))
	}
}

func TestCircuitBreaker_TrippedWorkers(t *testing.T) {
	cb := NewCircuitBreaker(2)

	cb.RecordFailure("Worker-A")
	cb.RecordFailure("Worker-A") // trips Worker-A
	cb.RecordFailure("Worker-B")
	cb.RecordFailure("Worker-B") // trips Worker-B

	tripped := cb.TrippedWorkers()
	if len(tripped) != 2 {
		t.Fatalf("expected 2 tripped workers, got %d", len(tripped))
	}
	if tripped[0] != "Worker-A" || tripped[1] != "Worker-B" {
		t.Fatalf("expected sorted tripped workers [Worker-A, Worker-B], got %v", tripped)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := NewCircuitBreaker(100)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			name := "Worker-" + string(rune('A'+id%26))
			cb.RecordFailure(name)
			cb.Allow(name)
			cb.RecordSuccess(name)
			cb.FailureCount(name)
			cb.TrippedWorkers()
		}(i)
	}
	wg.Wait()

	// No panic or race condition = success (go test -race validates this)
}

func TestCircuitBreaker_CustomThreshold(t *testing.T) {
	// Threshold of 1: first failure trips
	cb1 := NewCircuitBreaker(1)
	tripped := cb1.RecordFailure("Worker-X")
	if !tripped {
		t.Fatal("expected trip with threshold 1 after first failure")
	}

	// Threshold of 5: need 5 failures
	cb5 := NewCircuitBreaker(5)
	for i := 0; i < 4; i++ {
		if cb5.RecordFailure("Worker-Y") {
			t.Fatalf("expected no trip after %d failures with threshold 5", i+1)
		}
	}
	if !cb5.RecordFailure("Worker-Y") {
		t.Fatal("expected trip after 5th failure with threshold 5")
	}

	// Threshold < 1 should be clamped to 3
	cbInvalid := NewCircuitBreaker(0)
	for i := 0; i < 2; i++ {
		cbInvalid.RecordFailure("Worker-Z")
	}
	if cbInvalid.RecordFailure("Worker-Z") != true {
		t.Fatal("expected threshold 0 to be clamped to 3, tripping on 3rd failure")
	}
}

func TestFindSameCastePeer(t *testing.T) {
	dispatches := []codex.WorkerDispatch{
		{WorkerName: "Builder-A", Caste: "builder"},
		{WorkerName: "Builder-B", Caste: "builder"},
		{WorkerName: "Watcher-C", Caste: "watcher"},
	}

	cb := NewCircuitBreaker(3)

	// Peer available: Builder-A tripped, should find Builder-B
	cb.RecordFailure("Builder-A")
	cb.RecordFailure("Builder-A")
	cb.RecordFailure("Builder-A") // trips Builder-A

	peer := findSameCastePeer(dispatches, dispatches[0], cb)
	if peer == nil {
		t.Fatal("expected to find Builder-B as peer for tripped Builder-A")
	}
	if peer.WorkerName != "Builder-B" {
		t.Fatalf("expected peer WorkerName=Builder-B, got %s", peer.WorkerName)
	}

	// No peer available: both builders tripped
	cb.RecordFailure("Builder-B")
	cb.RecordFailure("Builder-B")
	cb.RecordFailure("Builder-B") // trips Builder-B

	peer = findSameCastePeer(dispatches, dispatches[0], cb)
	if peer != nil {
		t.Fatalf("expected nil when all same-caste peers are tripped, got %s", peer.WorkerName)
	}

	// No same-caste workers: watcher has no peer
	cb2 := NewCircuitBreaker(3)
	peer = findSameCastePeer(dispatches, dispatches[2], cb2)
	if peer != nil {
		t.Fatalf("expected nil when no same-caste workers exist, got %s", peer.WorkerName)
	}
}

func TestCircuitBreaker_PartialTripAndReset(t *testing.T) {
	cb := NewCircuitBreaker(3)

	// Partial failures for multiple workers
	cb.RecordFailure("Worker-1")
	cb.RecordFailure("Worker-1")
	cb.RecordFailure("Worker-2")

	// Worker-1 has 2 failures, Worker-2 has 1 failure
	if cb.FailureCount("Worker-1") != 2 {
		t.Fatalf("expected Worker-1 failure count 2, got %d", cb.FailureCount("Worker-1"))
	}
	if cb.FailureCount("Worker-2") != 1 {
		t.Fatalf("expected Worker-2 failure count 1, got %d", cb.FailureCount("Worker-2"))
	}

	// Reset clears everything
	cb.Reset()
	if cb.FailureCount("Worker-1") != 0 {
		t.Fatalf("expected Worker-1 failure count 0 after reset, got %d", cb.FailureCount("Worker-1"))
	}
	if cb.FailureCount("Worker-2") != 0 {
		t.Fatalf("expected Worker-2 failure count 0 after reset, got %d", cb.FailureCount("Worker-2"))
	}
	if len(cb.TrippedWorkers()) != 0 {
		t.Fatalf("expected 0 tripped workers after reset, got %d", len(cb.TrippedWorkers()))
	}
}
