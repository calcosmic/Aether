package agent

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

// StreamMultiplexer manages multiple consumers subscribing to the same agent streams.
// It uses a pub/sub pattern to broadcast events to all connected clients.
type StreamMultiplexer struct {
	bus           *events.Bus
	mu            sync.RWMutex
	streams       map[string]*multiplexedStream // key: streamID (e.g., "agent.builder" or "agent.*")
	consumerCount int
}

// multiplexedStream represents a single stream that multiple consumers can subscribe to.
type multiplexedStream struct {
	streamID    string
	topicPattern string
	sub         <-chan events.Event
	consumers   map[string]*streamConsumer
	mu          sync.RWMutex
	stopCh      chan struct{}
	stopped     bool
}

// streamConsumer represents a single consumer of a multiplexed stream.
type streamConsumer struct {
	id       string
	eventCh  chan events.Event
	doneCh   chan struct{}
	clientType ClientType
}

// ClientType indicates the type of client connection.
type ClientType string

const (
	ClientTypeSSE       ClientType = "sse"
	ClientTypeWebSocket ClientType = "websocket"
)

// NewStreamMultiplexer creates a new stream multiplexer for the given event bus.
func NewStreamMultiplexer(bus *events.Bus) *StreamMultiplexer {
	return &StreamMultiplexer{
		bus:     bus,
		streams: make(map[string]*multiplexedStream),
	}
}

// Subscribe registers a new consumer for a stream matching the given topic pattern.
// Returns a channel that receives all events for this stream and an ID for unsubscribing.
func (sm *StreamMultiplexer) Subscribe(topicPattern string, clientType ClientType) (<-chan events.Event, string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate unique consumer ID
	consumerID := fmt.Sprintf("consumer_%d_%d", time.Now().UnixNano(), sm.consumerCount)
	sm.consumerCount++

	// Get or create the multiplexed stream
	stream, exists := sm.streams[topicPattern]
	if !exists {
		// Create new multiplexed stream
		sub, err := sm.bus.Subscribe(topicPattern)
		if err != nil {
			return nil, "", fmt.Errorf("failed to subscribe to bus: %w", err)
		}

		stream = &multiplexedStream{
			streamID:     fmt.Sprintf("stream_%d", time.Now().UnixNano()),
			topicPattern: topicPattern,
			sub:          sub,
			consumers:    make(map[string]*streamConsumer),
			stopCh:       make(chan struct{}),
		}
		sm.streams[topicPattern] = stream

		// Start the broadcast goroutine for this stream
		go stream.broadcast()
	}

	// Create consumer
	consumer := &streamConsumer{
		id:         consumerID,
		eventCh:    make(chan events.Event, 256), // Buffered to avoid blocking
		doneCh:     make(chan struct{}),
		clientType: clientType,
	}

	stream.mu.Lock()
	if stream.stopped {
		stream.mu.Unlock()
		return nil, "", fmt.Errorf("stream has been stopped")
	}
	stream.consumers[consumerID] = consumer
	stream.mu.Unlock()

	return consumer.eventCh, consumerID, nil
}

// Unsubscribe removes a consumer from a stream and cleans up if no consumers remain.
func (sm *StreamMultiplexer) Unsubscribe(topicPattern, consumerID string) error {
	sm.mu.Lock()
	stream, exists := sm.streams[topicPattern]
	sm.mu.Unlock()

	if !exists {
		return fmt.Errorf("stream not found for pattern: %s", topicPattern)
	}

	stream.mu.Lock()
	consumer, exists := stream.consumers[consumerID]
	if !exists {
		stream.mu.Unlock()
		return fmt.Errorf("consumer not found: %s", consumerID)
	}

	// Remove consumer and close its channel
	delete(stream.consumers, consumerID)
	close(consumer.doneCh)
	// Don't close eventCh here — broadcast goroutine may still be sending to it.
	// The channel will be GC'd when all references are dropped.

	// Check if this was the last consumer
	shouldStop := len(stream.consumers) == 0
	if shouldStop {
		stream.stopped = true
	}
	stream.mu.Unlock()

	// If no more consumers, clean up the stream
	if shouldStop {
		close(stream.stopCh)
		sm.bus.Unsubscribe(topicPattern, stream.sub)

		sm.mu.Lock()
		delete(sm.streams, topicPattern)
		sm.mu.Unlock()
	}

	return nil
}

// broadcast continuously reads from the subscription and broadcasts to all consumers.
func (ms *multiplexedStream) broadcast() {
	for {
		select {
		case <-ms.stopCh:
			return

		case evt, ok := <-ms.sub:
			if !ok {
				// Subscription closed, signal all consumers
				ms.mu.Lock()
				for _, consumer := range ms.consumers {
					close(consumer.doneCh)
					// Don't close eventCh — broadcast is exiting so no more sends will occur
				}
				ms.consumers = make(map[string]*streamConsumer)
				ms.stopped = true
				ms.mu.Unlock()
				return
			}

			// Broadcast to all consumers
			ms.mu.RLock()
			consumers := make([]*streamConsumer, 0, len(ms.consumers))
			for _, c := range ms.consumers {
				consumers = append(consumers, c)
			}
			ms.mu.RUnlock()

			for _, consumer := range consumers {
				select {
				case consumer.eventCh <- evt:
					// Event sent successfully
				case <-consumer.doneCh:
					// Consumer is done, skip
				case <-ms.stopCh:
					return
				default:
					// Channel full - drop event for this consumer
					// This prevents slow consumers from blocking the broadcaster
				}
			}
		}
	}
}

// GetConsumerCount returns the number of active consumers for a stream.
func (sm *StreamMultiplexer) GetConsumerCount(topicPattern string) int {
	sm.mu.RLock()
	stream, exists := sm.streams[topicPattern]
	sm.mu.RUnlock()

	if !exists {
		return 0
	}

	stream.mu.RLock()
	defer stream.mu.RUnlock()
	return len(stream.consumers)
}

// GetTotalConsumerCount returns the total number of active consumers across all streams.
func (sm *StreamMultiplexer) GetTotalConsumerCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	total := 0
	for _, stream := range sm.streams {
		stream.mu.RLock()
		total += len(stream.consumers)
		stream.mu.RUnlock()
	}
	return total
}

// GetStreamCount returns the number of active multiplexed streams.
func (sm *StreamMultiplexer) GetStreamCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.streams)
}

// GetStreamInfo returns information about all active streams.
func (sm *StreamMultiplexer) GetStreamInfo() []StreamInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	info := make([]StreamInfo, 0, len(sm.streams))
	for topicPattern, stream := range sm.streams {
		stream.mu.RLock()
		consumers := make([]ConsumerInfo, 0, len(stream.consumers))
		for id, c := range stream.consumers {
			consumers = append(consumers, ConsumerInfo{
				ID:         id,
				ClientType: string(c.clientType),
			})
		}
		stream.mu.RUnlock()

		info = append(info, StreamInfo{
			TopicPattern:  topicPattern,
			ConsumerCount: len(consumers),
			Consumers:     consumers,
		})
	}
	return info
}

// StreamInfo provides information about a multiplexed stream.
type StreamInfo struct {
	TopicPattern  string         `json:"topicPattern"`
	ConsumerCount int            `json:"consumerCount"`
	Consumers     []ConsumerInfo `json:"consumers"`
}

// ConsumerInfo provides information about a stream consumer.
type ConsumerInfo struct {
	ID         string `json:"id"`
	ClientType string `json:"clientType"`
}

// Close stops all multiplexed streams and cleans up resources.
func (sm *StreamMultiplexer) Close() {
	sm.mu.Lock()
	streams := make([]*multiplexedStream, 0, len(sm.streams))
	for _, stream := range sm.streams {
		streams = append(streams, stream)
	}
	sm.streams = make(map[string]*multiplexedStream)
	sm.mu.Unlock()

	// Stop all streams
	for _, stream := range streams {
		close(stream.stopCh)
		sm.bus.Unsubscribe(stream.topicPattern, stream.sub)

		stream.mu.Lock()
		for _, consumer := range stream.consumers {
			close(consumer.doneCh)
			// Don't close eventCh — stopCh is already closed so broadcast will exit
		}
		stream.consumers = make(map[string]*streamConsumer)
		stream.stopped = true
		stream.mu.Unlock()
	}
}

// SSEConnection represents an active SSE connection managed by the multiplexer.
type SSEConnection struct {
	ConsumerID   string
	TopicPattern string
	EventCh      <-chan events.Event
	DoneCh       chan struct{}
	mu           sync.Mutex
	closed       bool
}

// Close closes the SSE connection and unsubscribes from the multiplexer.
func (c *SSEConnection) Close(sm *StreamMultiplexer) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.DoneCh)
	return sm.Unsubscribe(c.TopicPattern, c.ConsumerID)
}

// WebSocketConnection represents an active WebSocket connection managed by the multiplexer.
type WebSocketConnection struct {
	ConsumerID   string
	TopicPattern string
	EventCh      <-chan events.Event
	DoneCh       chan struct{}
	mu           sync.Mutex
	closed       bool
}

// Close closes the WebSocket connection and unsubscribes from the multiplexer.
func (c *WebSocketConnection) Close(sm *StreamMultiplexer) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	close(c.DoneCh)
	return sm.Unsubscribe(c.TopicPattern, c.ConsumerID)
}

// ConnectSSE creates a new SSE connection through the multiplexer.
func (sm *StreamMultiplexer) ConnectSSE(topicPattern string) (*SSEConnection, error) {
	eventCh, consumerID, err := sm.Subscribe(topicPattern, ClientTypeSSE)
	if err != nil {
		return nil, err
	}

	return &SSEConnection{
		ConsumerID:   consumerID,
		TopicPattern: topicPattern,
		EventCh:      eventCh,
		DoneCh:       make(chan struct{}),
	}, nil
}

// ConnectWebSocket creates a new WebSocket connection through the multiplexer.
func (sm *StreamMultiplexer) ConnectWebSocket(topicPattern string) (*WebSocketConnection, error) {
	eventCh, consumerID, err := sm.Subscribe(topicPattern, ClientTypeWebSocket)
	if err != nil {
		return nil, err
	}

	return &WebSocketConnection{
		ConsumerID:   consumerID,
		TopicPattern: topicPattern,
		EventCh:      eventCh,
		DoneCh:       make(chan struct{}),
	}, nil
}

// MultiplexerStats provides statistics about the multiplexer.
type MultiplexerStats struct {
	StreamCount       int `json:"streamCount"`
	TotalConsumers    int `json:"totalConsumers"`
	SSEConsumers      int `json:"sseConsumers"`
	WebSocketConsumers int `json:"webSocketConsumers"`
}

// GetStats returns current statistics about the multiplexer.
func (sm *StreamMultiplexer) GetStats() MultiplexerStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := MultiplexerStats{
		StreamCount: len(sm.streams),
	}

	for _, stream := range sm.streams {
		stream.mu.RLock()
		for _, consumer := range stream.consumers {
			stats.TotalConsumers++
			if consumer.clientType == ClientTypeSSE {
				stats.SSEConsumers++
			} else {
				stats.WebSocketConsumers++
			}
		}
		stream.mu.RUnlock()
	}

	return stats
}

// MarshalJSON implements json.Marshaler for stats.
func (s MultiplexerStats) MarshalJSON() ([]byte, error) {
	type Alias MultiplexerStats
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(&s),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
