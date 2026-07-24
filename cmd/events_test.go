package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewEventLineShape verifies the constructor produces correct fields.
func TestNewEventLineShape(t *testing.T) {
	payload := EventPayload{"phase": "1", "plan": "01"}
	line := NewEventLine(EventTypePhaseStart, payload)

	if line.Type != EventTypePhaseStart {
		t.Errorf("expected type %q, got %q", EventTypePhaseStart, line.Type)
	}
	if line.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if line.Payload["phase"] != "1" {
		t.Errorf("expected payload.phase=1, got %v", line.Payload["phase"])
	}

	// Timestamp should be parseable as RFC3339Nano.
	_, err := time.Parse(time.RFC3339Nano, line.Timestamp)
	if err != nil {
		t.Errorf("timestamp not RFC3339Nano parseable: %v", err)
	}
}

// TestWriteEventAppends verifies that WriteEvent creates a file and appends NDJSON.
func TestWriteEventAppends(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "test.ndjson")

	line := NewEventLine(EventTypeTaskPlanned, EventPayload{"task_id": "t1"})
	if err := WriteEvent(streamPath, line); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	data, err := os.ReadFile(streamPath)
	if err != nil {
		t.Fatalf("reading stream file: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("expected non-empty file after WriteEvent")
	}

	// Should end with newline.
	if data[len(data)-1] != '\n' {
		t.Error("expected NDJSON line to end with newline")
	}

	// Should contain the type.
	if !strings.Contains(string(data), `"type":"task_planned"`) {
		t.Errorf("expected JSON to contain task_planned type, got %s", string(data))
	}
}

// TestReadEventsReturnsParsedEvents verifies round-trip write then read.
func TestReadEventsReturnsParsedEvents(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "test.ndjson")

	lines := []EventLine{
		NewEventLine(EventTypePhaseStart, EventPayload{"phase": "1"}),
		NewEventLine(EventTypeAgentSelected, EventPayload{"agent_id": "a1"}),
	}

	for _, line := range lines {
		if err := WriteEvent(streamPath, line); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	events, err := ReadEvents(streamPath)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Type != EventTypePhaseStart {
		t.Errorf("expected event[0] type %q, got %q", EventTypePhaseStart, events[0].Type)
	}
	if events[1].Type != EventTypeAgentSelected {
		t.Errorf("expected event[1] type %q, got %q", EventTypeAgentSelected, events[1].Type)
	}
}

// TestRoundTripWriteRead verifies that written events are read back identically.
func TestRoundTripWriteRead(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "roundtrip.ndjson")

	original := NewEventLine(EventTypeMemoryUpdate, EventPayload{"key": "k", "value": 42, "source": "test"})
	if err := WriteEvent(streamPath, original); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	events, err := ReadEvents(streamPath)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	readBack := events[0]
	if readBack.Type != original.Type {
		t.Errorf("type mismatch: expected %q, got %q", original.Type, readBack.Type)
	}
	if readBack.Payload["key"] != original.Payload["key"] {
		t.Errorf("payload.key mismatch: expected %v, got %v", original.Payload["key"], readBack.Payload["key"])
	}
	if readBack.Payload["value"] != float64(42) {
		// JSON numbers unmarshal as float64.
		t.Errorf("payload.value mismatch: expected 42, got %v", readBack.Payload["value"])
	}
}

// TestNDJSONFormat verifies one JSON object per line, no trailing commas.
func TestNDJSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "format.ndjson")

	for i := 0; i < 3; i++ {
		line := NewEventLine(EventTypeCustom, EventPayload{"idx": i})
		if err := WriteEvent(streamPath, line); err != nil {
			t.Fatalf("WriteEvent failed: %v", err)
		}
	}

	data, err := os.ReadFile(streamPath)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}

	for i, l := range lines {
		if strings.HasSuffix(l, ",") {
			t.Errorf("line %d has trailing comma: %s", i, l)
		}
		if !strings.HasPrefix(l, "{") || !strings.HasSuffix(l, "}") {
			t.Errorf("line %d is not a JSON object: %s", i, l)
		}
	}
}

// TestConcurrentWrites verifies thread-safe EventStream writes.
func TestConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "concurrent.ndjson")

	es, err := CreateEventStream(streamPath)
	if err != nil {
		t.Fatalf("CreateEventStream failed: %v", err)
	}
	defer es.Close()

	var wg sync.WaitGroup
	workers := 10
	eventsPerWorker := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerWorker; j++ {
				line := NewEventLine(EventTypeWorkerSpawned, EventPayload{"worker_id": id, "seq": j})
				if err := es.Write(line); err != nil {
					t.Errorf("Write failed: %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	events, err := ReadEvents(streamPath)
	if err != nil {
		t.Fatalf("ReadEvents failed: %v", err)
	}

	expected := workers * eventsPerWorker
	if len(events) != expected {
		t.Fatalf("expected %d events, got %d", expected, len(events))
	}
}

// TestTailEventsYieldsNewEvents verifies that TailEvents picks up appended lines.
func TestTailEventsYieldsNewEvents(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "tail.ndjson")

	// Create the file with an initial event.
	line1 := NewEventLine(EventTypePhaseStart, EventPayload{"phase": "1"})
	if err := WriteEvent(streamPath, line1); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	done := make(chan struct{})
	ch, err := TailEvents(streamPath, done)
	if err != nil {
		t.Fatalf("TailEvents failed: %v", err)
	}

	// Give the tail goroutine time to start.
	time.Sleep(200 * time.Millisecond)

	// Append another event.
	line2 := NewEventLine(EventTypeRunSeal, EventPayload{"phase": "1", "plan": "01"})
	if err := WriteEvent(streamPath, line2); err != nil {
		t.Fatalf("WriteEvent failed: %v", err)
	}

	// Collect events with a timeout.
	var collected []EventLine
	timeout := time.AfterFunc(2*time.Second, func() { close(done) })
	defer timeout.Stop()

	for ev := range ch {
		collected = append(collected, ev)
		if len(collected) >= 1 {
			break
		}
	}
	close(done)

	if len(collected) == 0 {
		t.Fatal("expected at least one tailed event, got none")
	}

	last := collected[len(collected)-1]
	if last.Type != EventTypeRunSeal {
		t.Errorf("expected last tailed event type %q, got %q", EventTypeRunSeal, last.Type)
	}
}

// TestInitEventStreamCreatesDirectory verifies InitEventStream creates .aether/events.
func TestInitEventStreamCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Change into tmpDir so relative paths work.
	origWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origWd)

	streamPath, err := InitEventStream()
	if err != nil {
		t.Fatalf("InitEventStream failed: %v", err)
	}

	if _, err := os.Stat(streamPath); os.IsNotExist(err) {
		t.Fatalf("expected stream file to exist at %s", streamPath)
	}

	info, err := os.Stat(filepath.Join(".aether", "events"))
	if err != nil {
		t.Fatalf("expected events directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected events to be a directory")
	}
}

// TestMemoryBufferStub verifies in-memory stub helpers work.
func TestMemoryBufferStub(t *testing.T) {
	streamPath := "memory://test"
	ClearMemoryBuffer(streamPath)

	line := NewEventLine(EventTypeCustom, EventPayload{"hello": "world"})
	WriteEventToMemory(streamPath, line)

	buf := GetMemoryBuffer(streamPath)
	if len(buf) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(buf))
	}
	if buf[0].Payload["hello"] != "world" {
		t.Errorf("expected payload.hello=world, got %v", buf[0].Payload["hello"])
	}

	// ReadEventsFromMemory should return the same.
	readBack := ReadEventsFromMemory(streamPath)
	if len(readBack) != 1 {
		t.Fatalf("expected 1 event from memory reader, got %d", len(readBack))
	}
}

// TestEventStreamCloseIdempotent verifies Close is safe to call multiple times.
func TestEventStreamCloseIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "close.ndjson")

	es, err := CreateEventStream(streamPath)
	if err != nil {
		t.Fatalf("CreateEventStream failed: %v", err)
	}

	if err := es.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := es.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}

// TestEventStreamWriteAfterClose verifies writing after close returns an error.
func TestEventStreamWriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	streamPath := filepath.Join(tmpDir, "closed.ndjson")

	es, err := CreateEventStream(streamPath)
	if err != nil {
		t.Fatalf("CreateEventStream failed: %v", err)
	}

	if err := es.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	line := NewEventLine(EventTypeCustom, EventPayload{"x": 1})
	if err := es.Write(line); err == nil {
		t.Fatal("expected error writing to closed stream, got nil")
	}
}
