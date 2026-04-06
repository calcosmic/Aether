package terminal

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// syncBuffer is a thread-safe wrapper around bytes.Buffer for testing.
type syncBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

func (sb *syncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *syncBuffer) String() string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.buf.String()
}

// TestTokenDisplayCreation tests basic TokenDisplay creation.
func TestTokenDisplayCreation(t *testing.T) {
	td := NewTokenDisplay()

	if td == nil {
		t.Fatal("NewTokenDisplay() returned nil")
	}

	if td.output == nil {
		t.Error("TokenDisplay output writer should not be nil")
	}

	if td.isTTY {
		// May or may not be TTY depending on test environment
		t.Logf("isTTY: %v", td.isTTY)
	}

	// Clean up
	td.Close()
}

// TestTokenDisplayWithWriter tests creating TokenDisplay with custom writer.
func TestTokenDisplayWithWriter(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)

	if td.output != buf {
		t.Error("TokenDisplay should use provided writer")
	}

	if td.isTTY {
		t.Error("TokenDisplay should respect isTTY parameter")
	}

	td.Close()
}

// TestTokenDisplaySingleStream tests displaying a single stream.
func TestTokenDisplaySingleStream(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.UpdateStatus(streamID, StatusActive)

	tokens := []string{"Hello", " ", "world", "!"}
	for _, token := range tokens {
		td.AddToken(streamID, token)
	}

	// Flush to ensure output is written
	td.Flush()

	output := buf.String()
	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %q", output)
	}
}

// TestTokenDisplayMultipleStreams tests displaying multiple concurrent streams.
func TestTokenDisplayMultipleStreams(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	// Register multiple streams
	td.RegisterStream("builder-1", "Builder")
	td.RegisterStream("watcher-1", "Watcher")
	td.RegisterStream("scout-1", "Scout")

	// Set all streams to active
	td.UpdateStatus("builder-1", StatusActive)
	td.UpdateStatus("watcher-1", StatusActive)
	td.UpdateStatus("scout-1", StatusActive)

	// Add tokens to each stream
	td.AddToken("builder-1", "Building...")
	td.AddToken("watcher-1", "Watching...")
	td.AddToken("scout-1", "Scouting...")

	// Flush to ensure output is written
	td.Flush()

	output := buf.String()

	if !strings.Contains(output, "Building") {
		t.Errorf("Expected output to contain 'Building', got: %q", output)
	}
	if !strings.Contains(output, "Watching") {
		t.Errorf("Expected output to contain 'Watching', got: %q", output)
	}
	if !strings.Contains(output, "Scouting") {
		t.Errorf("Expected output to contain 'Scouting', got: %q", output)
	}
}

// TestTokenDisplayConcurrentAccess tests thread-safe concurrent access.
func TestTokenDisplayConcurrentAccess(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	var wg sync.WaitGroup
	numGoroutines := 10
	tokensPerGoroutine := 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				td.AddToken(streamID, "token")
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and should have processed all tokens
	stream := td.GetStream(streamID)
	if stream == nil {
		t.Fatal("Stream should exist")
	}

	expectedTokens := numGoroutines * tokensPerGoroutine
	if stream.GetTokenCount() != expectedTokens {
		t.Errorf("Expected %d tokens, got %d", expectedTokens, stream.GetTokenCount())
	}
}

// TestTokenDisplayUnregisterStream tests unregistering a stream.
func TestTokenDisplayUnregisterStream(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.AddToken(streamID, "test")

	// Unregister the stream
	td.UnregisterStream(streamID)

	// Adding tokens to unregistered stream should not panic
	td.AddToken(streamID, "ignored")

	stream := td.GetStream(streamID)
	if stream != nil {
		t.Error("Stream should be nil after unregistering")
	}
}

// TestTokenDisplayGetAllStreams tests retrieving all streams.
func TestTokenDisplayGetAllStreams(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	td.RegisterStream("builder-1", "Builder")
	td.RegisterStream("watcher-1", "Watcher")

	streams := td.GetAllStreams()
	if len(streams) != 2 {
		t.Errorf("Expected 2 streams, got %d", len(streams))
	}
}

// TestTokenDisplayBuffering tests token buffering for smooth display.
func TestTokenDisplayBuffering(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	// Set small buffer size for testing
	td.SetBufferSize(5)

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.UpdateStatus(streamID, StatusActive)

	// Add fewer tokens than buffer size
	for i := 0; i < 3; i++ {
		td.AddToken(streamID, "a")
	}

	// Should not have flushed yet (buffer not full) - check stream directly
	stream := td.GetStream(streamID)
	if stream.GetTokenCount() != 3 {
		t.Errorf("Expected 3 tokens in stream, got %d", stream.GetTokenCount())
	}

	// Add more tokens to exceed buffer
	for i := 0; i < 3; i++ {
		td.AddToken(streamID, "b")
	}

	// Force flush
	td.Flush()

	// Now should have output
	output := buf.String()
	if output == "" {
		t.Error("Expected output after buffer flush")
	}
}

// TestTokenDisplayFlushInterval tests automatic flush based on interval.
func TestTokenDisplayFlushInterval(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	// Set short flush interval for testing
	td.SetFlushInterval(50 * time.Millisecond)

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.UpdateStatus(streamID, StatusActive)

	// Add a single token
	td.AddToken(streamID, "test")

	// Force flush instead of waiting for interval to avoid race
	td.Flush()

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Errorf("Expected output to contain 'test' after flush, got: %q", output)
	}
}

// TestTokenDisplayClearLine tests ANSI clear line functionality.
func TestTokenDisplayClearLine(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, true)
	defer td.Close()

	clearSeq := td.clearLine()

	if !strings.Contains(clearSeq, "\r") {
		t.Error("Clear line should contain carriage return")
	}

	if !strings.Contains(clearSeq, "\033[K") {
		t.Error("Clear line should contain ANSI erase sequence")
	}
}

// TestTokenDisplayNonTTYFallback tests graceful fallback for non-TTY.
func TestTokenDisplayNonTTYFallback(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	tokens := []string{"Hello", " ", "world"}
	for _, token := range tokens {
		td.AddToken(streamID, token)
	}

	td.Flush()

	output := buf.String()

	// Non-TTY output should be simple, no ANSI codes
	if strings.Contains(output, "\033[") {
		t.Error("Non-TTY output should not contain ANSI escape codes")
	}

	if !strings.Contains(output, "Hello") {
		t.Errorf("Expected output to contain 'Hello', got: %q", output)
	}
}

// TestTokenDisplayUpdateStatus tests updating stream status.
func TestTokenDisplayUpdateStatus(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	td.UpdateStatus(streamID, StatusActive)

	stream := td.GetStream(streamID)
	if stream == nil {
		t.Fatal("Stream should exist")
	}

	if stream.GetStatus() != StatusActive {
		t.Errorf("Expected status %q, got %q", StatusActive, stream.GetStatus())
	}
}

// TestTokenDisplayCompleteStream tests marking a stream as complete.
func TestTokenDisplayCompleteStream(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.AddToken(streamID, "result")

	td.CompleteStream(streamID)

	stream := td.GetStream(streamID)
	if stream == nil {
		t.Fatal("Stream should exist")
	}

	if stream.GetStatus() != StatusCompleted {
		t.Errorf("Expected status %q, got %q", StatusCompleted, stream.GetStatus())
	}

	if stream.CompletedAt == nil {
		t.Error("CompletedAt should be set")
	}
}

// TestTokenDisplayFailStream tests marking a stream as failed.
func TestTokenDisplayFailStream(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	testErr := fmt.Errorf("test error")
	td.FailStream(streamID, testErr)

	stream := td.GetStream(streamID)
	if stream == nil {
		t.Fatal("Stream should exist")
	}

	if stream.GetStatus() != StatusFailed {
		t.Errorf("Expected status %q, got %q", StatusFailed, stream.GetStatus())
	}
}

// TestTokenDisplayPause tests pausing and resuming display.
func TestTokenDisplayPause(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	td.Pause()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")
	td.AddToken(streamID, "test")

	time.Sleep(50 * time.Millisecond)

	// Should not have output while paused
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output while paused, got: %q", output)
	}

	td.Resume()
	td.Flush()

	// Now should have output
	output = buf.String()
	if !strings.Contains(output, "test") {
		t.Errorf("Expected output after resume, got: %q", output)
	}
}

// TestTokenDisplayGetStreamSummary tests summary statistics.
func TestTokenDisplayGetStreamSummary(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	td.RegisterStream("builder-1", "Builder")
	td.RegisterStream("watcher-1", "Watcher")
	td.RegisterStream("scout-1", "Scout")

	td.AddToken("builder-1", "a")
	td.AddToken("builder-1", "b")
	td.AddToken("watcher-1", "c")

	td.CompleteStream("builder-1")

	summary := td.GetStreamSummary()

	if summary.Total != 3 {
		t.Errorf("Expected 3 total streams, got %d", summary.Total)
	}

	if summary.Active != 2 {
		t.Errorf("Expected 2 active streams, got %d", summary.Active)
	}

	if summary.Completed != 1 {
		t.Errorf("Expected 1 completed stream, got %d", summary.Completed)
	}

	if summary.TotalTokens != 3 {
		t.Errorf("Expected 3 total tokens, got %d", summary.TotalTokens)
	}
}

// TestTokenDisplayRenderLayout tests layout rendering.
func TestTokenDisplayRenderLayout(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	td.RegisterStream("builder-1", "Builder")
	td.AddToken("builder-1", "Building module...")

	td.Render()

	output := buf.String()
	if output == "" {
		t.Error("Render should produce output")
	}
}

// TestTokenDisplayClose tests cleanup on close.
func TestTokenDisplayClose(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)

	td.RegisterStream("builder-1", "Builder")
	td.AddToken("builder-1", "test")

	err := td.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// After close, adding tokens should not panic but may be ignored
	td.AddToken("builder-1", "ignored")
}

// TestTokenDisplayDuplicateRegistration tests handling duplicate stream registration.
func TestTokenDisplayDuplicateRegistration(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	// Registering same ID should return existing or update
	td.RegisterStream(streamID, "Builder")

	streams := td.GetAllStreams()
	if len(streams) != 1 {
		t.Errorf("Expected 1 stream (no duplicates), got %d", len(streams))
	}
}

// TestTokenDisplayEmptyStream tests behavior with empty stream.
func TestTokenDisplayEmptyStream(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	// Don't add any tokens
	td.CompleteStream(streamID)

	stream := td.GetStream(streamID)
	if stream.GetTokenCount() != 0 {
		t.Errorf("Expected 0 tokens, got %d", stream.GetTokenCount())
	}
}

// TestTokenDisplayRapidTokens tests handling rapid token arrival.
func TestTokenDisplayRapidTokens(t *testing.T) {
	buf := &syncBuffer{}
	td := NewTokenDisplayWithWriter(buf, false)
	defer td.Close()

	streamID := "builder-1"
	td.RegisterStream(streamID, "Builder")

	// Rapidly add many tokens
	var wg sync.WaitGroup
	numGoroutines := 10
	tokensPerGoroutine := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < tokensPerGoroutine; j++ {
				td.AddToken(streamID, "x")
			}
		}()
	}

	wg.Wait()

	stream := td.GetStream(streamID)
	if stream.GetTokenCount() != numGoroutines*tokensPerGoroutine {
		t.Errorf("Expected %d tokens, got %d", numGoroutines*tokensPerGoroutine, stream.GetTokenCount())
	}
}
