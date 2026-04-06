package llm

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

// EventBusStreamHandler implements StreamHandler and publishes tokens
// to the event bus with batching and backpressure handling.
type EventBusStreamHandler struct {
	agentName string
	bus       *events.Bus
	ctx       context.Context

	// Batching state
	mu          sync.Mutex
	buffer      []string
	sequence    int
	ticker      *time.Ticker
	stopCh      chan struct{}
	stopped     bool
	isComplete  bool
}

// EventBusStreamHandlerConfig configures the handler's behavior.
type EventBusStreamHandlerConfig struct {
	// AgentName identifies the agent producing tokens
	AgentName string
	// Bus is the event bus to publish to
	Bus *events.Bus
	// BatchInterval is the time interval for batching tokens (default 50ms)
	BatchInterval time.Duration
}

// NewEventBusStreamHandler creates a new handler that publishes tokens to the event bus.
// Tokens are batched and published at regular intervals to reduce event volume.
func NewEventBusStreamHandler(ctx context.Context, config EventBusStreamHandlerConfig) *EventBusStreamHandler {
	batchInterval := config.BatchInterval
	if batchInterval <= 0 {
		batchInterval = 50 * time.Millisecond
	}

	h := &EventBusStreamHandler{
		agentName: config.AgentName,
		bus:       config.Bus,
		ctx:       ctx,
		buffer:    make([]string, 0, 64),
		ticker:    time.NewTicker(batchInterval),
		stopCh:    make(chan struct{}),
	}

	// Start background batch publisher
	go h.batchLoop()

	return h
}

// batchLoop runs in a goroutine and publishes batched tokens at regular intervals.
func (h *EventBusStreamHandler) batchLoop() {
	for {
		select {
		case <-h.ticker.C:
			h.flush()
		case <-h.stopCh:
			h.flush()
			return
		case <-h.ctx.Done():
			h.flush()
			return
		}
	}
}

// flush publishes any buffered tokens to the event bus.
// Must be called with h.mu unlocked (it acquires the lock internally).
func (h *EventBusStreamHandler) flush() {
	h.mu.Lock()
	if len(h.buffer) == 0 {
		h.mu.Unlock()
		return
	}

	// Take ownership of buffer and reset
	content := h.concatenateBuffer()
	isComplete := h.isComplete
	seq := h.sequence
	h.buffer = h.buffer[:0]
	h.mu.Unlock()

	// Publish outside the lock to avoid blocking
	h.publish(content, seq, isComplete)

	// Increment sequence after successful publish planning
	h.mu.Lock()
	h.sequence++
	h.mu.Unlock()
}

// concatenateBuffer joins all buffered tokens into a single string.
// Must be called with h.mu held.
func (h *EventBusStreamHandler) concatenateBuffer() string {
	if len(h.buffer) == 0 {
		return ""
	}
	if len(h.buffer) == 1 {
		return h.buffer[0]
	}

	// Calculate total length
	totalLen := 0
	for _, t := range h.buffer {
		totalLen += len(t)
	}

	// Build concatenated string
	result := make([]byte, 0, totalLen)
	for _, t := range h.buffer {
		result = append(result, t...)
	}
	return string(result)
}

// publish sends a token event to the event bus.
// Handles backpressure by dropping events if the channel is full.
func (h *EventBusStreamHandler) publish(content string, sequence int, isComplete bool) {
	if content == "" && !isComplete {
		return
	}

	_, err := h.bus.PublishAgentToken(h.ctx, h.agentName, content, sequence, isComplete)
	if err != nil {
		// Log warning but don't block - backpressure handling
		log.Printf("[EventBusStreamHandler] Dropped token event for agent %s (seq %d): %v",
			h.agentName, sequence, err)
	}
}

// OnToken is called for each text token received from the stream.
// Tokens are buffered and published in batches.
func (h *EventBusStreamHandler) OnToken(token string) {
	if token == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.stopped {
		return
	}

	h.buffer = append(h.buffer, token)
}

// OnToolStart is called when a tool use block begins.
// Publishes immediately (not batched) for responsiveness.
func (h *EventBusStreamHandler) OnToolStart(toolName string, toolID string) {
	// Tool start events are important for UX - publish immediately
	content := "[Tool: " + toolName + "]"
	h.mu.Lock()
	seq := h.sequence
	h.sequence++
	h.mu.Unlock()

	h.publish(content, seq, false)
}

// OnToolEnd is called when a tool use block ends.
// Publishes immediately (not batched) for responsiveness.
func (h *EventBusStreamHandler) OnToolEnd(toolName string, toolID string, result string) {
	// Tool end events are important for UX - publish immediately
	content := "[/Tool: " + toolName + "]"
	h.mu.Lock()
	seq := h.sequence
	h.sequence++
	h.mu.Unlock()

	h.publish(content, seq, false)
}

// OnComplete is called when the stream completes successfully.
// Flushes any remaining buffered tokens and marks the stream as complete.
func (h *EventBusStreamHandler) OnComplete(result *StreamResult) {
	h.mu.Lock()
	h.isComplete = true
	h.stopped = true
	h.mu.Unlock()

	// Stop the ticker and trigger final flush
	h.ticker.Stop()
	close(h.stopCh)
}

// OnError is called when an error occurs during streaming.
// Stops the handler and flushes any remaining tokens.
func (h *EventBusStreamHandler) OnError(err error) {
	h.mu.Lock()
	h.stopped = true
	h.mu.Unlock()

	// Stop the ticker and trigger final flush
	h.ticker.Stop()
	close(h.stopCh)

	// Log the error
	log.Printf("[EventBusStreamHandler] Stream error for agent %s: %v", h.agentName, err)
}

// Stop gracefully stops the handler and flushes any remaining tokens.
func (h *EventBusStreamHandler) Stop() {
	h.mu.Lock()
	if h.stopped {
		h.mu.Unlock()
		return
	}
	h.stopped = true
	h.mu.Unlock()

	h.ticker.Stop()
	close(h.stopCh)
}

// Sequence returns the current sequence number (for testing).
func (h *EventBusStreamHandler) Sequence() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sequence
}
