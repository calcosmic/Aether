package cmd

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/calcosmic/Aether/pkg/events"
)

// emitColonyLive is the ONE emission boundary for every live-colony event.
// Every lifecycle lane -- build, continue, plan, swarm, oracle, recovery --
// must publish a live.* topic through this function and no other.
// cmd/live_projection_test.go's TestEveryLiveEventGoesThroughOneBoundary
// enforces this structurally, by walking the package's own parsed syntax
// tree, rather than by review: a future lane that grows its own live-event
// writer fails that test by name.
//
// Emission failure is never allowed to change the outcome of the work being
// described. A nil store, an empty topic, a marshal failure, or a bus
// publish failure all return silently -- no error is surfaced to the
// caller, and the caller's own work proceeds exactly as if emission had
// never been attempted.
func emitColonyLive(topic string, payload events.ColonyLivePayload) {
	if store == nil || strings.TrimSpace(topic) == "" {
		return
	}
	payload.SchemaVersion = events.ColonyLiveSchemaVersion
	if colonyLiveSchemaVersionOverride != "" {
		// Test-only seam (see doc comment below): lets a test produce a
		// genuine, boundary-emitted event carrying an unrecognized schema
		// version, so the replay reducer's version-skip path is proven
		// against real published events rather than a hand-typed JSON
		// fixture. Empty in production; never set outside a test.
		payload.SchemaVersion = colonyLiveSchemaVersionOverride
	}
	payload.Sequence = nextLiveSequence(payload.EpisodeID)

	raw, err := payload.RawMessage()
	if err != nil {
		return
	}

	bus := events.NewBus(store, events.DefaultConfig())
	_, _ = bus.Publish(context.Background(), topic, raw, "aether-live")
}

// colonyLiveSchemaVersionOverride is the test-only seam emitColonyLive
// checks above. It must never be set outside a test, and every test that
// sets it must restore it to "" (e.g. via t.Cleanup) before returning.
var colonyLiveSchemaVersionOverride string

var (
	liveSequenceMu       sync.Mutex
	liveSequenceCounters = map[string]*int64{}
)

// nextLiveSequence returns the next monotonic sequence number for the given
// episode, starting at 1. Each episode keeps its own independent, strictly
// increasing series -- concurrent episodes emitting at the same time never
// interfere with each other's numbering, and the projection's ordering of
// events sharing an identical (second-precision) timestamp falls back to
// this sequence number.
func nextLiveSequence(episodeID string) int64 {
	liveSequenceMu.Lock()
	counter, ok := liveSequenceCounters[episodeID]
	if !ok {
		counter = new(int64)
		liveSequenceCounters[episodeID] = counter
	}
	liveSequenceMu.Unlock()
	return atomic.AddInt64(counter, 1)
}
