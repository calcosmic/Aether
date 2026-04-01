package events

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aether-colony/aether/pkg/storage"
)

// subscriber holds a subscriber's channel and the topic pattern it matches.
type subscriber struct {
	pattern string
	ch      chan Event
}

// Bus provides in-memory pub/sub with JSONL persistence and TTL-based pruning.
type Bus struct {
	store  *storage.Store
	config Config

	mu          sync.RWMutex
	subscribers []subscriber
	closed      bool
}

// NewBus creates a new event bus backed by the given store.
func NewBus(store *storage.Store, config Config) *Bus {
	if config.DefaultTTL <= 0 {
		config.DefaultTTL = DefaultTTL
	}
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = DefaultLimit
	}
	if config.JSONLFile == "" {
		config.JSONLFile = "event-bus.jsonl"
	}
	return &Bus{
		store:  store,
		config: config,
	}
}

// Publish emits a typed event to the given topic. The event is persisted to JSONL
// and broadcast to all active subscribers whose patterns match the topic.
func (b *Bus) Publish(ctx context.Context, topic string, payload json.RawMessage, source string) (*Event, error) {
	if topic == "" {
		return nil, fmt.Errorf("events: publish requires a topic")
	}
	if len(payload) == 0 {
		return nil, fmt.Errorf("events: publish requires a payload")
	}

	now := time.Now().UTC()
	ttl := b.config.DefaultTTL

	// Generate ID matching shell: evt_{unix}_{4hex}
	rnd := make([]byte, 2)
	if _, err := rand.Read(rnd); err != nil {
		return nil, fmt.Errorf("events: generate random bytes: %w", err)
	}
	id := GenerateEventID(now, rnd)

	expiresAt := ComputeExpiry(now, ttl)

	evt := Event{
		ID:        id,
		Topic:     topic,
		Payload:   payload,
		Source:    source,
		Timestamp: FormatTimestamp(now),
		TTLDays:   ttl,
		ExpiresAt: FormatTimestamp(expiresAt),
	}

	// Persist to JSONL
	if err := b.store.AppendJSONL(b.config.JSONLFile, evt); err != nil {
		return nil, fmt.Errorf("events: persist event: %w", err)
	}

	// Broadcast to matching subscribers
	b.mu.RLock()
	defer b.mu.RUnlock()
	if b.closed {
		return &evt, nil
	}
	for _, sub := range b.subscribers {
		if TopicMatch(sub.pattern, topic) {
			select {
			case sub.ch <- evt:
			default:
				// Channel full; drop event for this subscriber rather than blocking.
				// Matches shell behavior where a slow consumer does not block publishers.
			}
		}
	}

	return &evt, nil
}

// Subscribe registers a subscriber for events matching the given topic pattern.
// Returns a read-only channel that receives matching events.
// The channel is buffered (capacity 256) to avoid blocking publishers.
func (b *Bus) Subscribe(topicPattern string) (<-chan Event, error) {
	if topicPattern == "" {
		return nil, fmt.Errorf("events: subscribe requires a topic pattern")
	}

	ch := make(chan Event, 256)

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, fmt.Errorf("events: bus is closed")
	}

	b.subscribers = append(b.subscribers, subscriber{
		pattern: topicPattern,
		ch:      ch,
	})

	return ch, nil
}

// Unsubscribe removes a subscriber channel and closes it.
// The publisher is never blocked by unsubscribe operations.
func (b *Bus) Unsubscribe(topicPattern string, ch <-chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.subscribers {
		if sub.pattern == topicPattern && sub.ch == ch {
			// Remove from slice without preserving order
			b.subscribers[i] = b.subscribers[len(b.subscribers)-1]
			b.subscribers[len(b.subscribers)-1] = subscriber{}
			b.subscribers = b.subscribers[:len(b.subscribers)-1]
			close(sub.ch)
			return
		}
	}
}

// Close marks the bus as closed and closes all subscriber channels.
// No further publishes or subscribes are accepted after Close.
func (b *Bus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	for _, sub := range b.subscribers {
		close(sub.ch)
	}
	b.subscribers = nil
}

// Query reads events from JSONL matching the topic pattern, not expired,
// and optionally since the given time. Returns up to limit events.
// Returns an empty slice (not error) if the file does not exist.
func (b *Bus) Query(ctx context.Context, topicPattern string, since time.Time, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = b.config.DefaultLimit
	}

	lines, err := b.store.ReadJSONL(b.config.JSONLFile)
	if err != nil {
		// File does not exist -- return empty
		return []Event{}, nil
	}

	now := FormatTimestamp(time.Now().UTC())
	var results []Event
	for _, raw := range lines {
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue // skip malformed
		}
		if !TopicMatch(topicPattern, evt.Topic) {
			continue
		}
		if evt.ExpiresAt <= now {
			continue // expired
		}
		if !since.IsZero() && evt.Timestamp < FormatTimestamp(since.UTC()) {
			continue
		}
		results = append(results, evt)
		if len(results) >= limit {
			break
		}
	}

	if results == nil {
		results = []Event{}
	}
	return results, nil
}

// Replay reads events from JSONL matching the exact topic, not expired,
// and since the given time. Events are sorted by timestamp ascending.
// Returns up to limit events.
// Returns an empty slice (not error) if the file does not exist.
func (b *Bus) Replay(ctx context.Context, topic string, since time.Time, limit int) ([]Event, error) {
	if limit <= 0 {
		limit = b.config.DefaultLimit
	}

	lines, err := b.store.ReadJSONL(b.config.JSONLFile)
	if err != nil {
		return []Event{}, nil
	}

	now := FormatTimestamp(time.Now().UTC())
	var results []Event
	for _, raw := range lines {
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue
		}
		if evt.Topic != topic {
			continue
		}
		if evt.ExpiresAt <= now {
			continue
		}
		if !since.IsZero() && evt.Timestamp < FormatTimestamp(since.UTC()) {
			continue
		}
		results = append(results, evt)
	}

	// Sort by timestamp ascending (matching shell: sort_by(.timestamp))
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Timestamp > results[j].Timestamp {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit < len(results) {
		results = results[:limit]
	}

	if results == nil {
		results = []Event{}
	}
	return results, nil
}

// Cleanup removes expired events from the JSONL file.
// If dryRun is true, it reports counts without modifying the file.
// Returns the number of events removed and remaining.
func (b *Bus) Cleanup(ctx context.Context, dryRun bool) (removed int, remaining int, err error) {
	lines, err := b.store.ReadJSONL(b.config.JSONLFile)
	if err != nil {
		return 0, 0, nil // file doesn't exist, nothing to clean
	}

	now := FormatTimestamp(time.Now().UTC())
	var kept []json.RawMessage
	for _, raw := range lines {
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			kept = append(kept, raw) // keep malformed lines to avoid data loss
			continue
		}
		if evt.ExpiresAt > now {
			kept = append(kept, raw)
		}
	}

	removed = len(lines) - len(kept)
	remaining = len(kept)

	if dryRun {
		return removed, remaining, nil
	}

	// Rewrite file atomically with kept events
	var data []byte
	for _, line := range kept {
		data = append(data, line...)
		data = append(data, '\n')
	}

	if err := b.store.AtomicWrite(b.config.JSONLFile, data); err != nil {
		return 0, 0, fmt.Errorf("events: cleanup rewrite: %w", err)
	}

	return removed, remaining, nil
}

// LoadAndReplay replays all non-expired events from JSONL to matching subscribers.
// This is the crash recovery mechanism -- call it on startup to ensure no events
// are lost after an unclean shutdown.
func (b *Bus) LoadAndReplay(ctx context.Context) error {
	lines, err := b.store.ReadJSONL(b.config.JSONLFile)
	if err != nil {
		return nil // file doesn't exist, nothing to replay
	}

	now := FormatTimestamp(time.Now().UTC())

	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, raw := range lines {
		var evt Event
		if err := json.Unmarshal(raw, &evt); err != nil {
			continue // skip malformed
		}
		if evt.ExpiresAt <= now {
			continue // expired
		}
		// Broadcast to matching subscribers
		for _, sub := range b.subscribers {
			if TopicMatch(sub.pattern, evt.Topic) {
				select {
				case sub.ch <- evt:
				default:
					// Drop if full
				}
			}
		}
	}

	return nil
}
