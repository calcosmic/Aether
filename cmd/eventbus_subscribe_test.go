package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

func TestEventBusSubscribeStreamsMatchingEventsAsNDJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	go func() {
		time.Sleep(30 * time.Millisecond)
		bus := events.NewBus(s, events.DefaultConfig())
		payload, err := (events.CeremonyPayload{
			Phase:  1,
			Wave:   1,
			Caste:  "builder",
			Name:   "Mason-67",
			Status: "starting",
		}).RawMessage()
		if err != nil {
			t.Errorf("payload marshal failed: %v", err)
			return
		}
		if _, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, payload, "unit-test"); err != nil {
			t.Errorf("publish failed: %v", err)
		}
	}()

	rootCmd.SetArgs([]string{
		"event-bus-subscribe",
		"--stream",
		"--filter", "ceremony.*",
		"--poll-interval", "10ms",
		"--timeout", "1s",
		"--max-events", "1",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("event-bus-subscribe returned error: %v", err)
	}

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("expected one NDJSON event line")
	}
	if strings.Contains(line, `{"ok":true`) {
		t.Fatalf("stream output must be raw NDJSON, got envelope: %s", line)
	}
	var evt events.Event
	if err := json.Unmarshal([]byte(line), &evt); err != nil {
		t.Fatalf("stream output is not an event: %v\n%s", err, line)
	}
	if evt.Topic != events.CeremonyTopicBuildSpawn {
		t.Fatalf("topic = %q, want %q", evt.Topic, events.CeremonyTopicBuildSpawn)
	}
}

func TestEventBusSubscribeQueryModeUsesEnvelope(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	s, _ := newTestStore(t)
	store = s
	var buf bytes.Buffer
	stdout = &buf

	bus := events.NewBus(s, events.DefaultConfig())
	payload, err := (events.CeremonyPayload{Phase: 1, Message: "hello"}).RawMessage()
	if err != nil {
		t.Fatalf("payload marshal failed: %v", err)
	}
	if _, err := bus.Publish(context.Background(), events.CeremonyTopicBuildPrewave, payload, "unit-test"); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	rootCmd.SetArgs([]string{"event-bus-subscribe", "--filter", "ceremony.*"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("event-bus-subscribe returned error: %v", err)
	}
	if !strings.Contains(buf.String(), `{"ok":true`) {
		t.Fatalf("query mode should use JSON envelope, got: %s", buf.String())
	}
}

func TestEventBusSubscribeStreamDefaultHasNoDeadline(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	resetFlags(rootCmd)

	ctx, cancel := eventBusSubscribeContext(eventBusSubscribeCmd, true)
	defer cancel()
	if deadline, ok := ctx.Deadline(); ok {
		t.Fatalf("stream context has default deadline %v; stream mode should be continuous unless --timeout is set", deadline)
	}

	if err := eventBusSubscribeCmd.Flags().Set("timeout", "10ms"); err != nil {
		t.Fatalf("failed to set timeout flag: %v", err)
	}
	ctx, cancel = eventBusSubscribeContext(eventBusSubscribeCmd, true)
	defer cancel()
	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("stream context should have deadline when --timeout is explicitly set")
	}
}

func TestStreamEventBusNDJSONDoesNotStarveSameSecondBurstWithLowLimit(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	bus := events.NewBus(s, events.DefaultConfig())
	var buf bytes.Buffer
	stdout = &buf

	timestamp := "2026-04-24T02:00:00Z"
	expiresAt := "2026-04-25T02:00:00Z"
	for i, name := range []string{"Mason-1", "Mason-2", "Mason-3"} {
		payload, err := (events.CeremonyPayload{Name: name, Status: "starting"}).RawMessage()
		if err != nil {
			t.Fatalf("payload marshal failed: %v", err)
		}
		evt := events.Event{
			ID:        "evt_same_second_" + name,
			Topic:     events.CeremonyTopicBuildSpawn,
			Payload:   payload,
			Source:    "unit-test",
			Timestamp: timestamp,
			TTLDays:   1,
			ExpiresAt: expiresAt,
		}
		if i == 0 {
			// Seed a non-matching event first to ensure filtering is still honored.
			other := evt
			other.ID = "evt_other"
			other.Topic = "learning.observe"
			if err := s.AppendJSONL("event-bus.jsonl", other); err != nil {
				t.Fatalf("append non-matching event: %v", err)
			}
		}
		if err := s.AppendJSONL("event-bus.jsonl", evt); err != nil {
			t.Fatalf("append event: %v", err)
		}
	}

	since := time.Date(2026, 4, 24, 1, 59, 59, 0, time.UTC)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := streamEventBusNDJSON(ctx, bus, "ceremony.*", since, 1, time.Millisecond, 3); err != nil {
		t.Fatalf("streamEventBusNDJSON returned error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d streamed lines, want 3\n%s", len(lines), buf.String())
	}
	for _, want := range []string{"Mason-1", "Mason-2", "Mason-3"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("stream output missing %s\n%s", want, buf.String())
		}
	}
}

func TestEventBusStreamPipesToNarratorRuntime(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found; skipping narrator pipe smoke")
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	narratorPath := filepath.Join(repoRoot, ".aether", "ts", "dist", "narrator.js")
	if _, err := os.Stat(narratorPath); err != nil {
		t.Fatalf("narrator runtime missing: %v", err)
	}

	saveGlobals(t)
	s, _ := newTestStore(t)
	bus := events.NewBus(s, events.DefaultConfig())

	reader, writer := io.Pipe()
	stdout = writer
	cmd := exec.Command(nodePath, narratorPath)
	cmd.Stdin = reader
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start narrator runtime: %v", err)
	}

	streamErr := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		defer writer.Close()
		since := time.Now().UTC().Add(-time.Second)
		streamErr <- streamEventBusNDJSON(ctx, bus, "ceremony.*", since, 100, 10*time.Millisecond, 1)
	}()

	time.Sleep(30 * time.Millisecond)
	payload, err := (events.CeremonyPayload{
		Phase:  2,
		Wave:   1,
		Caste:  "builder",
		Name:   "Mason-67",
		Status: "streamed",
	}).RawMessage()
	if err != nil {
		t.Fatalf("payload marshal failed: %v", err)
	}
	if _, err := bus.Publish(context.Background(), events.CeremonyTopicBuildSpawn, payload, "unit-test"); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	if err := <-streamErr; err != nil {
		t.Fatalf("streamEventBusNDJSON returned error: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("narrator runtime failed: %v\nstderr:\n%s", err, stderr.String())
	}
	if !strings.Contains(out.String(), "[CEREMONY] ceremony.build.spawn phase=2 wave=1 builder:Mason-67 status=streamed") {
		t.Fatalf("narrator output mismatch:\nstdout:\n%s\nstderr:\n%s", out.String(), stderr.String())
	}
}
