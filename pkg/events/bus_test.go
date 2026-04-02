package events

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aether-colony/aether/pkg/storage"
)

func newTestBus(t *testing.T) (*Bus, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bus := NewBus(store, Config{JSONLFile: "event-bus.jsonl"})
	return bus, dir
}

func TestEventJSONMarshal(t *testing.T) {
	payload := json.RawMessage(`{"action":"observe","trust":0.75}`)
	evt := Event{
		ID:        "evt_1712000000_a1b2",
		Topic:     "learning.observe",
		Payload:   payload,
		Source:    "builder",
		Timestamp: "2026-04-01T12:00:00Z",
		TTLDays:   30,
		ExpiresAt: "2026-05-01T12:00:00Z",
	}
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	for _, field := range []string{`"id"`, `"topic"`, `"payload"`, `"source"`, `"timestamp"`, `"ttl_days"`, `"expires_at"`} {
		if !strings.Contains(s, field) {
			t.Errorf("missing field %s in JSON", field)
		}
	}
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ID != evt.ID {
		t.Errorf("ID mismatch")
	}
	if decoded.TTLDays != evt.TTLDays {
		t.Errorf("TTLDays mismatch")
	}
}

func TestTopicMatchExact(t *testing.T) {
	if !TopicMatch("learning.observe", "learning.observe") {
		t.Error("should match exact topic")
	}
	if TopicMatch("learning.observe", "learning.promote") {
		t.Error("should not match different topic")
	}
}

func TestTopicMatchWildcard(t *testing.T) {
	if !TopicMatch("learning.*", "learning.observe") {
		t.Error("should match with wildcard")
	}
	if TopicMatch("learning.*", "learning") {
		t.Error("learning.* should NOT match bare learning")
	}
	if TopicMatch("learning.*", "memory.observe") {
		t.Error("should not match different prefix")
	}
}

func TestGenerateEventIDFormat(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	random := []byte{0xa1, 0xb2}
	id := GenerateEventID(now, random)
	expected := fmt.Sprintf("evt_%d_a1b2", now.Unix())
	if id != expected {
		t.Errorf("ID = %q, want %q", id, expected)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultTTL != 30 {
		t.Errorf("DefaultTTL = %d, want 30", cfg.DefaultTTL)
	}
	if cfg.DefaultLimit != 50 {
		t.Errorf("DefaultLimit = %d, want 50", cfg.DefaultLimit)
	}
	if cfg.JSONLFile != "event-bus.jsonl" {
		t.Errorf("JSONLFile = %q, want event-bus.jsonl", cfg.JSONLFile)
	}
}

func TestFormatTimestamp(t *testing.T) {
	now := time.Date(2026, 4, 1, 12, 30, 45, 0, time.UTC)
	got := FormatTimestamp(now)
	if got != "2026-04-01T12:30:45Z" {
		t.Errorf("FormatTimestamp = %q, want 2026-04-01T12:30:45Z", got)
	}
}

func TestComputeExpiry(t *testing.T) {
	created := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	expiry := ComputeExpiry(created, 30)
	want := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if !expiry.Equal(want) {
		t.Errorf("ComputeExpiry = %v, want %v", expiry, want)
	}
}

func TestPublishAndSubscribe(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	ch, err := bus.Subscribe("learning.*")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	payload := json.RawMessage(`{"action":"observe"}`)
	evt, err := bus.Publish(ctx, "learning.observe", payload, "builder")
	if err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if evt == nil {
		t.Fatal("Publish returned nil")
	}
	select {
	case received := <-ch:
		if received.ID != evt.ID {
			t.Errorf("received ID mismatch")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestPublishMultipleSubscribers(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	ch1, _ := bus.Subscribe("learning.*")
	ch2, _ := bus.Subscribe("learning.*")
	ch3, _ := bus.Subscribe("memory.*")
	evt, _ := bus.Publish(ctx, "learning.observe", json.RawMessage(`{"test":true}`), "system")
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case received := <-ch:
			if received.ID != evt.ID {
				t.Errorf("subscriber %d: ID mismatch", i)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d timed out", i)
		}
	}
	select {
	case <-ch3:
		t.Error("memory.* should not receive learning.observe")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPublishOrdering(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	ch, _ := bus.Subscribe("test.*")
	for i := 0; i < 5; i++ {
		bus.Publish(ctx, "test.order", json.RawMessage(fmt.Sprintf(`{"seq":%d}`, i)), "system")
	}
	for i := 0; i < 5; i++ {
		select {
		case evt := <-ch:
			var p map[string]int
			json.Unmarshal(evt.Payload, &p)
			if p["seq"] != i {
				t.Errorf("out of order: got %d, want %d", p["seq"], i)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out at event %d", i)
		}
	}
}

func TestPublishRequiresTopic(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := bus.Publish(context.Background(), "", json.RawMessage(`{}`), "system")
	if err == nil {
		t.Error("expected error for empty topic")
	}
}

func TestPublishRequiresPayload(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := bus.Publish(context.Background(), "test", nil, "system")
	if err == nil {
		t.Error("expected error for nil payload")
	}
}

func TestSubscribeRequiresPattern(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	_, err := bus.Subscribe("")
	if err == nil {
		t.Error("expected error for empty pattern")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	ch, _ := bus.Subscribe("test.*")
	bus.Publish(ctx, "test.event", json.RawMessage(`{"first":true}`), "system")
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("should have received first event")
	}
	bus.Unsubscribe("test.*", ch)
	bus.Publish(ctx, "test.event", json.RawMessage(`{"second":true}`), "system")
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("should not receive after unsubscribe")
		}
	case <-time.After(50 * time.Millisecond):
	}
}

func TestCloseClosesAllChannels(t *testing.T) {
	bus, _ := newTestBus(t)
	ch1, _ := bus.Subscribe("test.*")
	ch2, _ := bus.Subscribe("other.*")
	bus.Close()
	_, ok1 := <-ch1
	_, ok2 := <-ch2
	if ok1 || ok2 {
		t.Error("channels should be closed")
	}
}

func TestSubscribeAfterClose(t *testing.T) {
	bus, _ := newTestBus(t)
	bus.Close()
	_, err := bus.Subscribe("test.*")
	if err == nil {
		t.Error("expected error subscribing to closed bus")
	}
}

func TestPublishPersistsToJSONL(t *testing.T) {
	bus, dir := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	evt, _ := bus.Publish(ctx, "test.persist", json.RawMessage(`{"persisted":true}`), "system")
	data, err := os.ReadFile(filepath.Join(dir, "event-bus.jsonl"))
	if err != nil {
		t.Fatalf("read JSONL: %v", err)
	}
	var stored Event
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &stored)
	if stored.ID != evt.ID {
		t.Errorf("stored ID = %q, want %q", stored.ID, evt.ID)
	}
}

func TestMultiplePublishesAppendLines(t *testing.T) {
	bus, dir := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		bus.Publish(ctx, "test.multi", json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)), "system")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "event-bus.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestQueryReturnsMatchingEvents(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	bus.Publish(ctx, "learning.observe", json.RawMessage(`{"a":1}`), "system")
	bus.Publish(ctx, "memory.store", json.RawMessage(`{"b":2}`), "system")
	bus.Publish(ctx, "learning.promote", json.RawMessage(`{"c":3}`), "system")
	events, _ := bus.Query(ctx, "learning.*", time.Time{}, 10)
	if len(events) != 2 {
		t.Errorf("expected 2 learning events, got %d", len(events))
	}
}

func TestQueryWithLimit(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		bus.Publish(ctx, "test.limit", json.RawMessage(fmt.Sprintf(`{"i":%d}`, i)), "system")
	}
	events, _ := bus.Query(ctx, "test.limit", time.Time{}, 3)
	if len(events) != 3 {
		t.Errorf("expected 3 events with limit, got %d", len(events))
	}
}

func TestQueryEmptyFile(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	events, _ := bus.Query(context.Background(), "test.*", time.Time{}, 10)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestReplayReturnsSortedEvents(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	bus.Publish(ctx, "test.replay", json.RawMessage(`{"seq":0}`), "system")
	bus.Publish(ctx, "test.replay", json.RawMessage(`{"seq":1}`), "system")
	bus.Publish(ctx, "test.replay", json.RawMessage(`{"seq":2}`), "system")
	events, _ := bus.Replay(ctx, "test.replay", time.Time{}, 10)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp < events[i-1].Timestamp {
			t.Errorf("events not sorted at index %d", i)
		}
	}
}

func TestReplayRequiresTopic(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	bus.Publish(ctx, "test.replay", json.RawMessage(`{}`), "system")
	bus.Publish(ctx, "other.topic", json.RawMessage(`{}`), "system")
	events, _ := bus.Replay(ctx, "test.replay", time.Time{}, 10)
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestReplayEmptyFile(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	events, _ := bus.Replay(context.Background(), "test", time.Time{}, 10)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestCleanupRemovesExpiredEvents(t *testing.T) {
	bus, dir := newTestBus(t)
	ctx := context.Background()
	expiredEvent := Event{
		ID: "evt_expired", Topic: "test.old", Payload: json.RawMessage(`{"old":true}`),
		Source: "system", Timestamp: "2020-01-01T00:00:00Z", TTLDays: 30, ExpiresAt: "2020-02-01T00:00:00Z",
	}
	bus.store.AppendJSONL(bus.config.JSONLFile, expiredEvent)
	bus.Publish(ctx, "test.new", json.RawMessage(`{"new":true}`), "system")
	removed, remaining, _ := bus.Cleanup(ctx, false)
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	if remaining != 1 {
		t.Errorf("remaining = %d, want 1", remaining)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "event-bus.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	bus.Close()
}

func TestCleanupDryRun(t *testing.T) {
	bus, dir := newTestBus(t)
	expiredEvent := Event{
		ID: "evt_expired", Topic: "test.old", Payload: json.RawMessage(`{}`),
		Source: "system", Timestamp: "2020-01-01T00:00:00Z", TTLDays: 30, ExpiresAt: "2020-02-01T00:00:00Z",
	}
	bus.store.AppendJSONL(bus.config.JSONLFile, expiredEvent)
	removed, _, _ := bus.Cleanup(context.Background(), true)
	if removed != 1 {
		t.Errorf("removed = %d, want 1", removed)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "event-bus.jsonl"))
	if len(data) == 0 {
		t.Error("file should not be empty after dry run")
	}
	bus.Close()
}

func TestCleanupNoFile(t *testing.T) {
	bus, _ := newTestBus(t)
	removed, remaining, err := bus.Cleanup(context.Background(), false)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if removed != 0 || remaining != 0 {
		t.Errorf("removed=%d remaining=%d, want 0,0", removed, remaining)
	}
	bus.Close()
}

func TestCrashRecoveryReplaysEvents(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	ctx := context.Background()
	cfg := Config{JSONLFile: "event-bus.jsonl"}
	bus1 := NewBus(store, cfg)
	bus1.Publish(ctx, "recovery.test", json.RawMessage(`{"seq":1}`), "system")
	bus1.Publish(ctx, "recovery.test", json.RawMessage(`{"seq":2}`), "system")
	bus1.Close()
	store2, _ := storage.NewStore(dir)
	bus2 := NewBus(store2, cfg)
	ch, _ := bus2.Subscribe("recovery.*")
	bus2.LoadAndReplay(ctx)
	count := 0
	timeout := time.After(time.Second)
	for count < 2 {
		select {
		case <-ch:
			count++
		case <-timeout:
			t.Fatalf("timed out, received %d of 2", count)
		}
	}
	bus2.Close()
}

func TestCrashRecoverySkipsExpired(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	ctx := context.Background()
	cfg := Config{JSONLFile: "event-bus.jsonl"}
	expired := Event{
		ID: "evt_expired_recovery", Topic: "recovery.old", Payload: json.RawMessage(`{"old":true}`),
		Source: "system", Timestamp: "2020-01-01T00:00:00Z", TTLDays: 30, ExpiresAt: "2020-02-01T00:00:00Z",
	}
	store.AppendJSONL(cfg.JSONLFile, expired)
	now := time.Now().UTC()
	current := Event{
		ID: "evt_current_recovery", Topic: "recovery.new", Payload: json.RawMessage(`{"new":true}`),
		Source: "system", Timestamp: now.Format("2006-01-02T15:04:05Z"),
		TTLDays: 30, ExpiresAt: now.AddDate(0, 0, 30).Format("2006-01-02T15:04:05Z"),
	}
	store.AppendJSONL(cfg.JSONLFile, current)
	bus := NewBus(store, cfg)
	ch, _ := bus.Subscribe("recovery.*")
	bus.LoadAndReplay(ctx)
	count := 0
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case <-ch:
			count++
		case <-timeout:
			goto done
		}
	}
done:
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
	bus.Close()
}

func TestConcurrentPublish(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	ctx := context.Background()
	ch, _ := bus.Subscribe("concurrent.*")
	var wg sync.WaitGroup
	var errors atomic.Int64
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			payload := json.RawMessage(fmt.Sprintf(`{"worker":%d}`, n))
			if _, err := bus.Publish(ctx, "concurrent.test", payload, "system"); err != nil {
				errors.Add(1)
			}
		}(i)
	}
	wg.Wait()
	if errors.Load() > 0 {
		t.Errorf("got %d errors", errors.Load())
	}
	received := 0
	timeout := time.After(2 * time.Second)
	for received < 20 {
		select {
		case <-ch:
			received++
		case <-timeout:
			t.Fatalf("timed out at %d of 20", received)
		}
	}
}

func TestQuerySkipsExpired(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	expired := Event{
		ID: "evt_expired_query", Topic: "test.query", Payload: json.RawMessage(`{"expired":true}`),
		Source: "system", Timestamp: "2020-01-01T00:00:00Z", TTLDays: 30, ExpiresAt: "2020-02-01T00:00:00Z",
	}
	bus.store.AppendJSONL(bus.config.JSONLFile, expired)
	ctx := context.Background()
	bus.Publish(ctx, "test.query", json.RawMessage(`{"current":true}`), "system")
	events, _ := bus.Query(ctx, "test.query", time.Time{}, 10)
	if len(events) != 1 {
		t.Fatalf("expected 1 non-expired event, got %d", len(events))
	}
}

func TestReplaySkipsExpired(t *testing.T) {
	bus, _ := newTestBus(t)
	defer bus.Close()
	expired := Event{
		ID: "evt_expired_replay", Topic: "test.replay", Payload: json.RawMessage(`{"expired":true}`),
		Source: "system", Timestamp: "2020-01-01T00:00:00Z", TTLDays: 30, ExpiresAt: "2020-02-01T00:00:00Z",
	}
	bus.store.AppendJSONL(bus.config.JSONLFile, expired)
	events, _ := bus.Replay(context.Background(), "test.replay", time.Time{}, 10)
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}
