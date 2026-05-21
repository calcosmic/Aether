package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

func TestEventBusPublishCreatesEvent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{
		"event-bus-publish",
		"--topic", "test.topic",
		"--payload", `{"key":"value"}`,
		"--source", "unit-test",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["topic"] != "test.topic" {
		t.Errorf("topic = %v, want test.topic", result["topic"])
	}
	if result["source"] != "unit-test" {
		t.Errorf("source = %v, want unit-test", result["source"])
	}
	if result["id"] == "" {
		t.Error("expected non-empty event id")
	}

	// Verify filesystem state: event should be in JSONL
	data, err := s.ReadFile("event-bus.jsonl")
	if err != nil {
		t.Fatalf("event-bus.jsonl not found: %v", err)
	}
	if !strings.Contains(string(data), `"topic":"test.topic"`) {
		t.Errorf("event-bus.jsonl missing published event")
	}
}

func TestEventBusQueryReturnsMatchingEvents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed events directly via the bus
	bus := events.NewBus(s, events.DefaultConfig())
	payload, _ := json.Marshal(map[string]string{"msg": "hello"})
	if _, err := bus.Publish(context.Background(), "test.query", json.RawMessage(payload), "seed"); err != nil {
		t.Fatalf("seed publish failed: %v", err)
	}
	if _, err := bus.Publish(context.Background(), "other.topic", json.RawMessage(payload), "seed"); err != nil {
		t.Fatalf("seed publish failed: %v", err)
	}

	rootCmd.SetArgs([]string{
		"event-bus-query",
		"--pattern", "test.*",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["count"] != float64(1) {
		t.Errorf("count = %v, want 1", result["count"])
	}

	evts := result["events"].([]interface{})
	if len(evts) != 1 {
		t.Fatalf("events length = %d, want 1", len(evts))
	}
	evt := evts[0].(map[string]interface{})
	if evt["topic"] != "test.query" {
		t.Errorf("event topic = %v, want test.query", evt["topic"])
	}
}

func TestEventBusReplayReturnsChronologicalEvents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed multiple events for the same topic
	bus := events.NewBus(s, events.DefaultConfig())
	for i := 1; i <= 3; i++ {
		payload, _ := json.Marshal(map[string]int{"idx": i})
		if _, err := bus.Publish(context.Background(), "replay.topic", json.RawMessage(payload), "seed"); err != nil {
			t.Fatalf("seed publish failed: %v", err)
		}
		time.Sleep(5 * time.Millisecond)
	}

	rootCmd.SetArgs([]string{
		"event-bus-replay",
		"--topic", "replay.topic",
	})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["count"] != float64(3) {
		t.Errorf("count = %v, want 3", result["count"])
	}

	evts := result["events"].([]interface{})
	if len(evts) != 3 {
		t.Fatalf("events length = %d, want 3", len(evts))
	}
}

func TestEventBusCleanupRemovesExpiredEvents(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed one expired and one fresh event directly
	now := time.Now().UTC()
	expiredEvt := events.Event{
		ID:        "expired-1",
		Topic:     "cleanup.test",
		Payload:   json.RawMessage(`{}`),
		Source:    "unit-test",
		Timestamp: now.Add(-48 * time.Hour).Format(time.RFC3339),
		TTLDays:   1,
		ExpiresAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
	}
	freshEvt := events.Event{
		ID:        "fresh-1",
		Topic:     "cleanup.test",
		Payload:   json.RawMessage(`{}`),
		Source:    "unit-test",
		Timestamp: now.Format(time.RFC3339),
		TTLDays:   7,
		ExpiresAt: now.Add(7 * 24 * time.Hour).Format(time.RFC3339),
	}
	if err := s.AppendJSONL("event-bus.jsonl", expiredEvt); err != nil {
		t.Fatalf("append expired event: %v", err)
	}
	if err := s.AppendJSONL("event-bus.jsonl", freshEvt); err != nil {
		t.Fatalf("append fresh event: %v", err)
	}

	rootCmd.SetArgs([]string{"event-bus-cleanup"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["removed"] != float64(1) {
		t.Errorf("removed = %v, want 1", result["removed"])
	}
	if result["remaining"] != float64(1) {
		t.Errorf("remaining = %v, want 1", result["remaining"])
	}

	// Verify filesystem state
	data, err := s.ReadFile("event-bus.jsonl")
	if err != nil {
		t.Fatalf("event-bus.jsonl not found: %v", err)
	}
	if strings.Contains(string(data), `"id":"expired-1"`) {
		t.Error("event-bus.jsonl still contains expired event")
	}
	if !strings.Contains(string(data), `"id":"fresh-1"`) {
		t.Error("event-bus.jsonl missing fresh event")
	}
}

func TestEventBusCleanupDryRunDoesNotDelete(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Seed one expired event
	now := time.Now().UTC()
	expiredEvt := events.Event{
		ID:        "expired-dry",
		Topic:     "cleanup.dry",
		Payload:   json.RawMessage(`{}`),
		Source:    "unit-test",
		Timestamp: now.Add(-48 * time.Hour).Format(time.RFC3339),
		TTLDays:   1,
		ExpiresAt: now.Add(-24 * time.Hour).Format(time.RFC3339),
	}
	if err := s.AppendJSONL("event-bus.jsonl", expiredEvt); err != nil {
		t.Fatalf("append expired event: %v", err)
	}

	rootCmd.SetArgs([]string{"event-bus-cleanup", "--dry-run"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}

	result := env["result"].(map[string]interface{})
	if result["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", result["dry_run"])
	}
	if result["removed"] != float64(1) {
		t.Errorf("removed = %v, want 1", result["removed"])
	}

	// Verify filesystem state: event should still be present
	data, err := s.ReadFile("event-bus.jsonl")
	if err != nil {
		t.Fatalf("event-bus.jsonl not found: %v", err)
	}
	if !strings.Contains(string(data), `"id":"expired-dry"`) {
		t.Error("dry-run should not delete events from file")
	}
}
