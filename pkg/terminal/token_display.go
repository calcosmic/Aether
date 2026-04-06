// Package terminal provides terminal UI components for displaying streaming content.
package terminal

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// StreamStatus represents the current state of a display stream.
type StreamStatus string

const (
	// StatusPending indicates the stream is registered but not yet active.
	StatusPending StreamStatus = "pending"
	// StatusActive indicates the stream is currently receiving tokens.
	StatusActive StreamStatus = "active"
	// StatusCompleted indicates the stream finished successfully.
	StatusCompleted StreamStatus = "completed"
	// StatusFailed indicates the stream encountered an error.
	StatusFailed StreamStatus = "failed"
	// StatusCancelled indicates the stream was cancelled.
	StatusCancelled StreamStatus = "cancelled"
)

// DisplayStream represents a single agent's token stream for display.
type DisplayStream struct {
	ID          string
	AgentName   string
	Caste       string
	Status      StreamStatus
	Tokens      []string
	TokenCount  int
	Buffer      strings.Builder
	StartedAt   time.Time
	CompletedAt *time.Time
	mu          sync.RWMutex
}

// AddToken appends a token to the stream's buffer.
// Returns the new token count.
func (ds *DisplayStream) AddToken(token string) int {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.Tokens = append(ds.Tokens, token)
	ds.TokenCount++
	ds.Buffer.WriteString(token)
	return ds.TokenCount
}

// GetTokenCount returns the current token count thread-safely.
func (ds *DisplayStream) GetTokenCount() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.TokenCount
}

// GetContent returns the accumulated content of the stream.
func (ds *DisplayStream) GetContent() string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.Buffer.String()
}

// SetStatus updates the stream status thread-safely.
func (ds *DisplayStream) SetStatus(status StreamStatus) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.Status = status
	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		now := time.Now()
		ds.CompletedAt = &now
	}
}

// GetStatus returns the current stream status.
func (ds *DisplayStream) GetStatus() StreamStatus {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.Status
}

// TokenDisplay manages the live display of streaming tokens from multiple agents.
// It handles buffering, ANSI escape codes, and concurrent stream updates.
type TokenDisplay struct {
	output        io.Writer
	outputMu      sync.Mutex
	streams       map[string]*DisplayStream
	mu            sync.RWMutex
	isTTY         bool
	bufferSize    int
	bufferSizeMu  sync.RWMutex
	flushInterval time.Duration
	flushMu       sync.RWMutex
	paused        bool
	pausedMu      sync.RWMutex
	flushTimer    *time.Timer
	done          chan struct{}
}

// Default configuration values.
const (
	DefaultBufferSize    = 10
	DefaultFlushInterval = 100 * time.Millisecond
)

// NewTokenDisplay creates a new TokenDisplay that writes to stdout.
// It auto-detects TTY capability for ANSI escape code handling.
func NewTokenDisplay() *TokenDisplay {
	return NewTokenDisplayWithWriter(os.Stdout, isTerminal(os.Stdout))
}

// NewTokenDisplayWithWriter creates a new TokenDisplay with a custom writer.
// The isTTY parameter controls whether ANSI escape codes are used.
func NewTokenDisplayWithWriter(output io.Writer, isTTY bool) *TokenDisplay {
	td := &TokenDisplay{
		output:        output,
		streams:       make(map[string]*DisplayStream),
		isTTY:         isTTY,
		bufferSize:    DefaultBufferSize,
		flushInterval: DefaultFlushInterval,
		done:          make(chan struct{}),
	}

	// Start background flush goroutine
	go td.flushLoop()

	return td
}

// isTerminal checks if the writer is a terminal (best effort).
func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		// Check if it's a character device (terminal)
		return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	}
	return false
}

// RegisterStream registers a new stream for display.
// If a stream with the same ID already exists, it returns the existing stream.
func (td *TokenDisplay) RegisterStream(streamID, caste string) *DisplayStream {
	td.mu.Lock()
	defer td.mu.Unlock()

	if existing, ok := td.streams[streamID]; ok {
		return existing
	}

	stream := &DisplayStream{
		ID:        streamID,
		AgentName: streamID,
		Caste:     caste,
		Status:    StatusPending,
		Tokens:    make([]string, 0),
		StartedAt: time.Now(),
	}

	td.streams[streamID] = stream
	return stream
}

// UnregisterStream removes a stream from the display.
func (td *TokenDisplay) UnregisterStream(streamID string) {
	td.mu.Lock()
	defer td.mu.Unlock()
	delete(td.streams, streamID)
}

// GetStream retrieves a stream by ID.
func (td *TokenDisplay) GetStream(streamID string) *DisplayStream {
	td.mu.RLock()
	defer td.mu.RUnlock()
	return td.streams[streamID]
}

// GetAllStreams returns all registered streams.
func (td *TokenDisplay) GetAllStreams() []*DisplayStream {
	td.mu.RLock()
	defer td.mu.RUnlock()

	result := make([]*DisplayStream, 0, len(td.streams))
	for _, stream := range td.streams {
		result = append(result, stream)
	}
	return result
}

// AddToken adds a token to a stream's display buffer.
// The token is buffered and will be displayed on the next flush.
func (td *TokenDisplay) AddToken(streamID, token string) {
	td.mu.RLock()
	stream, ok := td.streams[streamID]
	td.mu.RUnlock()

	if !ok {
		return
	}

	newCount := stream.AddToken(token)

	// Trigger flush if buffer is full
	if newCount%td.getBufferSize() == 0 {
		td.Flush()
	}
}

// UpdateStatus updates the status of a stream.
func (td *TokenDisplay) UpdateStatus(streamID string, status StreamStatus) {
	td.mu.RLock()
	stream, ok := td.streams[streamID]
	td.mu.RUnlock()

	if !ok {
		return
	}

	stream.SetStatus(status)
	td.Flush()
}

// CompleteStream marks a stream as completed.
func (td *TokenDisplay) CompleteStream(streamID string) {
	td.UpdateStatus(streamID, StatusCompleted)
}

// FailStream marks a stream as failed with an error.
func (td *TokenDisplay) FailStream(streamID string, err error) {
	td.mu.RLock()
	stream, ok := td.streams[streamID]
	td.mu.RUnlock()

	if !ok {
		return
	}

	stream.SetStatus(StatusFailed)
	if err != nil {
		stream.AddToken(fmt.Sprintf(" [Error: %v]", err))
	}
	td.Flush()
}

// Pause temporarily pauses display updates.
func (td *TokenDisplay) Pause() {
	td.pausedMu.Lock()
	defer td.pausedMu.Unlock()
	td.paused = true
}

// Resume resumes display updates after a pause.
func (td *TokenDisplay) Resume() {
	td.pausedMu.Lock()
	defer td.pausedMu.Unlock()
	td.paused = false
}

// isPaused returns true if the display is currently paused.
func (td *TokenDisplay) isPaused() bool {
	td.pausedMu.RLock()
	defer td.pausedMu.RUnlock()
	return td.paused
}

// SetBufferSize sets the number of tokens to buffer before auto-flushing.
func (td *TokenDisplay) SetBufferSize(size int) {
	if size < 1 {
		size = 1
	}
	td.bufferSizeMu.Lock()
	defer td.bufferSizeMu.Unlock()
	td.bufferSize = size
}

// getBufferSize returns the current buffer size thread-safely.
func (td *TokenDisplay) getBufferSize() int {
	td.bufferSizeMu.RLock()
	defer td.bufferSizeMu.RUnlock()
	return td.bufferSize
}

// SetFlushInterval sets the maximum time between display updates.
func (td *TokenDisplay) SetFlushInterval(interval time.Duration) {
	if interval < 10*time.Millisecond {
		interval = 10 * time.Millisecond
	}
	td.flushMu.Lock()
	defer td.flushMu.Unlock()
	td.flushInterval = interval
}

// getFlushInterval returns the current flush interval thread-safely.
func (td *TokenDisplay) getFlushInterval() time.Duration {
	td.flushMu.RLock()
	defer td.flushMu.RUnlock()
	return td.flushInterval
}

// flushLoop runs in the background to periodically flush the display.
func (td *TokenDisplay) flushLoop() {
	ticker := time.NewTicker(td.getFlushInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			td.Flush()
		case <-td.done:
			return
		}
	}
}

// Flush immediately renders all pending output.
func (td *TokenDisplay) Flush() {
	if td.isPaused() {
		return
	}

	td.mu.RLock()
	streams := make([]*DisplayStream, 0, len(td.streams))
	for _, stream := range td.streams {
		streams = append(streams, stream)
	}
	td.mu.RUnlock()

	if len(streams) == 0 {
		return
	}

	// Render based on TTY capability
	if td.isTTY {
		td.renderTTY(streams)
	} else {
		td.renderNonTTY(streams)
	}
}

// Render forces an immediate display update (alias for Flush).
func (td *TokenDisplay) Render() {
	td.Flush()
}

// renderTTY renders output with ANSI escape codes for terminal display.
func (td *TokenDisplay) renderTTY(streams []*DisplayStream) {
	var output strings.Builder

	// Move cursor to beginning and clear screen section
	output.WriteString(td.clearScreen())

	// Render each stream
	for _, stream := range streams {
		statusIcon := td.getStatusIcon(stream.GetStatus())
		content := stream.GetContent()

		// Truncate long content for display
		if len(content) > 80 {
			content = content[:77] + "..."
		}

		// Replace newlines with spaces for single-line display
		content = strings.ReplaceAll(content, "\n", " ")

		line := fmt.Sprintf("%s [%s] %s: %s\n",
			statusIcon,
			stream.Caste,
			stream.ID,
			content,
		)
		output.WriteString(line)
	}

	// Write output
	td.outputMu.Lock()
	fmt.Fprint(td.output, output.String())
	td.outputMu.Unlock()
}

// renderNonTTY renders output suitable for non-terminal environments.
func (td *TokenDisplay) renderNonTTY(streams []*DisplayStream) {
	var output strings.Builder

	for _, stream := range streams {
		status := stream.GetStatus()
		if status == StatusActive || status == StatusPending {
			statusIcon := td.getStatusIcon(status)
			content := stream.GetContent()

			line := fmt.Sprintf("%s [%s] %s: %s\n",
				statusIcon,
				stream.Caste,
				stream.ID,
				content,
			)
			output.WriteString(line)
		}
	}

	if output.Len() > 0 {
		td.outputMu.Lock()
		fmt.Fprint(td.output, output.String())
		td.outputMu.Unlock()
	}
}

// getStatusIcon returns an emoji icon for the stream status.
func (td *TokenDisplay) getStatusIcon(status StreamStatus) string {
	switch status {
	case StatusPending:
		return "⏳"
	case StatusActive:
		return "▶️"
	case StatusCompleted:
		return "✅"
	case StatusFailed:
		return "❌"
	case StatusCancelled:
		return "🚫"
	default:
		return "❓"
	}
}

// clearScreen returns ANSI escape sequence to clear display area.
func (td *TokenDisplay) clearScreen() string {
	// For now, just clear from cursor to end of screen
	// In a full implementation, this would track line count for proper clearing
	return "\033[2K\r"
}

// clearLine returns ANSI escape sequence to clear current line.
func (td *TokenDisplay) clearLine() string {
	return "\r\033[K"
}

// StreamSummary provides statistics about all streams.
type StreamSummary struct {
	Total       int
	Active      int
	Completed   int
	Failed      int
	TotalTokens int
	ByCaste     map[string]int
	ByStatus    map[StreamStatus]int
}

// GetStreamSummary returns a summary of all streams.
func (td *TokenDisplay) GetStreamSummary() StreamSummary {
	td.mu.RLock()
	defer td.mu.RUnlock()

	summary := StreamSummary{
		Total:    len(td.streams),
		ByCaste:  make(map[string]int),
		ByStatus: make(map[StreamStatus]int),
	}

	for _, stream := range td.streams {
		status := stream.GetStatus()
		summary.ByStatus[status]++
		summary.ByCaste[stream.Caste]++
		summary.TotalTokens += stream.TokenCount

		switch status {
		case StatusActive, StatusPending:
			summary.Active++
		case StatusCompleted:
			summary.Completed++
		case StatusFailed:
			summary.Failed++
		}
	}

	return summary
}

// Close cleans up the TokenDisplay and stops background goroutines.
func (td *TokenDisplay) Close() error {
	close(td.done)
	return nil
}
