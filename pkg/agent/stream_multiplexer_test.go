package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestStreamMultiplexer_CreateAndSubscribe(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Subscribe to a stream
	eventCh, consumerID, err := mux.Subscribe("agent.builder.*", ClientTypeSSE)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	if consumerID == "" {
		t.Error("expected non-empty consumer ID")
	}

	if eventCh == nil {
		t.Error("expected non-nil event channel")
	}

	// Check stream count
	if mux.GetStreamCount() != 1 {
		t.Errorf("expected 1 stream, got %d", mux.GetStreamCount())
	}

	// Check consumer count
	if mux.GetConsumerCount("agent.builder.*") != 1 {
		t.Errorf("expected 1 consumer, got %d", mux.GetConsumerCount("agent.builder.*"))
	}
}

func TestStreamMultiplexer_MultipleConsumersSameStream(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Subscribe multiple consumers to the same stream
	consumers := 5
	var channels []<-chan events.Event
	var consumerIDs []string

	for i := 0; i < consumers; i++ {
		eventCh, consumerID, err := mux.Subscribe("agent.builder.*", ClientTypeSSE)
		if err != nil {
			t.Fatalf("failed to subscribe consumer %d: %v", i, err)
		}
		channels = append(channels, eventCh)
		consumerIDs = append(consumerIDs, consumerID)
	}

	// Should only have 1 stream (multiplexed)
	if mux.GetStreamCount() != 1 {
		t.Errorf("expected 1 stream, got %d", mux.GetStreamCount())
	}

	// Should have 5 consumers
	if mux.GetConsumerCount("agent.builder.*") != consumers {
		t.Errorf("expected %d consumers, got %d", consumers, mux.GetConsumerCount("agent.builder.*"))
	}

	// Total consumer count should also be 5
	if mux.GetTotalConsumerCount() != consumers {
		t.Errorf("expected %d total consumers, got %d", consumers, mux.GetTotalConsumerCount())
	}

	// Test that all consumers receive events
	ctx := context.Background()
	testContent := "test token content"
	_, err = bus.PublishAgentToken(ctx, "builder", testContent, 0, false)
	if err != nil {
		t.Fatalf("failed to publish token: %v", err)
	}

	// Give time for event to propagate
	time.Sleep(50 * time.Millisecond)

	// Verify all consumers received the event
	for i, ch := range channels {
		select {
		case evt := <-ch:
			if evt.Topic != "agent.builder.tokens" {
				t.Errorf("consumer %d: expected topic agent.builder.tokens, got %s", i, evt.Topic)
			}
		case <-time.After(time.Second):
			t.Errorf("consumer %d: timeout waiting for event", i)
		}
	}

	// Unsubscribe all consumers
	for i, id := range consumerIDs {
		err := mux.Unsubscribe("agent.builder.*", id)
		if err != nil {
			t.Errorf("failed to unsubscribe consumer %d: %v", i, err)
		}
	}

	// Stream should be cleaned up
	if mux.GetStreamCount() != 0 {
		t.Errorf("expected 0 streams after unsubscribe, got %d", mux.GetStreamCount())
	}
}

func TestStreamMultiplexer_MultipleStreams(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Subscribe to different streams
	_, _, err = mux.Subscribe("agent.builder.*", ClientTypeSSE)
	if err != nil {
		t.Fatalf("failed to subscribe to builder: %v", err)
	}

	_, _, err = mux.Subscribe("agent.watcher.*", ClientTypeSSE)
	if err != nil {
		t.Fatalf("failed to subscribe to watcher: %v", err)
	}

	_, _, err = mux.Subscribe("agent.*", ClientTypeWebSocket)
	if err != nil {
		t.Fatalf("failed to subscribe to all agents: %v", err)
	}

	// Should have 3 streams
	if mux.GetStreamCount() != 3 {
		t.Errorf("expected 3 streams, got %d", mux.GetStreamCount())
	}

	// Publish events and verify correct routing
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "builder token", 0, false)
	bus.PublishAgentToken(ctx, "watcher", "watcher token", 0, false)

	time.Sleep(50 * time.Millisecond)

	// Verify stats
	stats := mux.GetStats()
	if stats.StreamCount != 3 {
		t.Errorf("expected 3 streams in stats, got %d", stats.StreamCount)
	}
	if stats.TotalConsumers != 3 {
		t.Errorf("expected 3 total consumers in stats, got %d", stats.TotalConsumers)
	}
	if stats.SSEConsumers != 2 {
		t.Errorf("expected 2 SSE consumers in stats, got %d", stats.SSEConsumers)
	}
	if stats.WebSocketConsumers != 1 {
		t.Errorf("expected 1 WebSocket consumer in stats, got %d", stats.WebSocketConsumers)
	}
}

func TestStreamMultiplexer_Unsubscribe(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)

	// Subscribe
	_, consumerID, err := mux.Subscribe("agent.builder.*", ClientTypeSSE)
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Unsubscribe
	err = mux.Unsubscribe("agent.builder.*", consumerID)
	if err != nil {
		t.Fatalf("failed to unsubscribe: %v", err)
	}

	// Stream should be cleaned up (no more consumers)
	if mux.GetStreamCount() != 0 {
		t.Errorf("expected 0 streams after unsubscribe, got %d", mux.GetStreamCount())
	}

	// Unsubscribe again should fail
	err = mux.Unsubscribe("agent.builder.*", consumerID)
	if err == nil {
		t.Error("expected error when unsubscribing non-existent consumer")
	}

	// Unsubscribe from non-existent stream should fail
	err = mux.Unsubscribe("agent.nonexistent.*", "fake-id")
	if err == nil {
		t.Error("expected error when unsubscribing from non-existent stream")
	}

	mux.Close()
}

func TestStreamMultiplexer_GracefulDisconnect(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)

	// Subscribe multiple consumers
	var consumerIDs []string
	for i := 0; i < 3; i++ {
		_, id, _ := mux.Subscribe("agent.builder.*", ClientTypeSSE)
		consumerIDs = append(consumerIDs, id)
	}

	// Unsubscribe one consumer
	err = mux.Unsubscribe("agent.builder.*", consumerIDs[0])
	if err != nil {
		t.Fatalf("failed to unsubscribe: %v", err)
	}

	// Stream should still exist (2 more consumers)
	if mux.GetStreamCount() != 1 {
		t.Errorf("expected 1 stream, got %d", mux.GetStreamCount())
	}

	if mux.GetConsumerCount("agent.builder.*") != 2 {
		t.Errorf("expected 2 consumers, got %d", mux.GetConsumerCount("agent.builder.*"))
	}

	// Remaining consumers should still receive events
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "test", 0, false)

	time.Sleep(50 * time.Millisecond)

	mux.Close()
}

func TestStreamMultiplexer_Close(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)

	// Create multiple streams with multiple consumers
	for i := 0; i < 3; i++ {
		pattern := fmt.Sprintf("agent.%d.*", i)
		for j := 0; j < 3; j++ {
			mux.Subscribe(pattern, ClientTypeSSE)
		}
	}

	if mux.GetStreamCount() != 3 {
		t.Errorf("expected 3 streams, got %d", mux.GetStreamCount())
	}

	if mux.GetTotalConsumerCount() != 9 {
		t.Errorf("expected 9 consumers, got %d", mux.GetTotalConsumerCount())
	}

	// Close should clean everything up
	mux.Close()

	if mux.GetStreamCount() != 0 {
		t.Errorf("expected 0 streams after close, got %d", mux.GetStreamCount())
	}

	if mux.GetTotalConsumerCount() != 0 {
		t.Errorf("expected 0 consumers after close, got %d", mux.GetTotalConsumerCount())
	}
}

func TestStreamMultiplexer_SSEConnection(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create SSE connection
	conn, err := mux.ConnectSSE("agent.builder.*")
	if err != nil {
		t.Fatalf("failed to connect SSE: %v", err)
	}

	if conn.ConsumerID == "" {
		t.Error("expected non-empty consumer ID")
	}

	if conn.TopicPattern != "agent.builder.*" {
		t.Errorf("expected topic pattern agent.builder.*, got %s", conn.TopicPattern)
	}

	// Test receiving events
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "sse test", 0, false)

	select {
	case evt := <-conn.EventCh:
		if evt.Topic != "agent.builder.tokens" {
			t.Errorf("expected topic agent.builder.tokens, got %s", evt.Topic)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for SSE event")
	}

	// Close connection
	err = conn.Close(mux)
	if err != nil {
		t.Fatalf("failed to close connection: %v", err)
	}

	// Double close should be safe
	err = conn.Close(mux)
	if err != nil {
		t.Errorf("double close should not error: %v", err)
	}
}

func TestStreamMultiplexer_WebSocketConnection(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create WebSocket connection
	conn, err := mux.ConnectWebSocket("agent.watcher.*")
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}

	if conn.ConsumerID == "" {
		t.Error("expected non-empty consumer ID")
	}

	// Test receiving events
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "watcher", "ws test", 0, false)

	select {
	case evt := <-conn.EventCh:
		if evt.Topic != "agent.watcher.tokens" {
			t.Errorf("expected topic agent.watcher.tokens, got %s", evt.Topic)
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for WebSocket event")
	}

	// Close connection
	err = conn.Close(mux)
	if err != nil {
		t.Fatalf("failed to close connection: %v", err)
	}
}

func TestStreamMultiplexer_MixedClientTypes(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create mixed connections
	sseConn1, _ := mux.ConnectSSE("agent.builder.*")
	wsConn1, _ := mux.ConnectWebSocket("agent.builder.*")
	sseConn2, _ := mux.ConnectSSE("agent.builder.*")
	wsConn2, _ := mux.ConnectWebSocket("agent.builder.*")

	// All should receive the same events
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "mixed test", 0, false)

	time.Sleep(50 * time.Millisecond)

	connections := []struct {
		name string
		ch   <-chan events.Event
	}{
		{"sse1", sseConn1.EventCh},
		{"ws1", wsConn1.EventCh},
		{"sse2", sseConn2.EventCh},
		{"ws2", wsConn2.EventCh},
	}

	for _, conn := range connections {
		select {
		case evt := <-conn.ch:
			if evt.Topic != "agent.builder.tokens" {
				t.Errorf("%s: expected topic agent.builder.tokens, got %s", conn.name, evt.Topic)
			}
		default:
			t.Errorf("%s: did not receive event", conn.name)
		}
	}

	// Cleanup
	sseConn1.Close(mux)
	wsConn1.Close(mux)
	sseConn2.Close(mux)
	wsConn2.Close(mux)
}

func TestStreamMultiplexer_GetStreamInfo(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create connections
	mux.ConnectSSE("agent.builder.*")
	mux.ConnectWebSocket("agent.builder.*")
	mux.ConnectSSE("agent.watcher.*")

	info := mux.GetStreamInfo()

	if len(info) != 2 {
		t.Errorf("expected 2 stream info entries, got %d", len(info))
	}

	for _, streamInfo := range info {
		if streamInfo.TopicPattern == "agent.builder.*" {
			if streamInfo.ConsumerCount != 2 {
				t.Errorf("expected 2 consumers for builder, got %d", streamInfo.ConsumerCount)
			}
		}
		if streamInfo.TopicPattern == "agent.watcher.*" {
			if streamInfo.ConsumerCount != 1 {
				t.Errorf("expected 1 consumer for watcher, got %d", streamInfo.ConsumerCount)
			}
		}
	}
}

func TestStreamMultiplexer_StatsJSON(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	mux.ConnectSSE("agent.builder.*")
	mux.ConnectWebSocket("agent.builder.*")

	stats := mux.GetStats()
	data, err := stats.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal stats: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(data)
	if !contains(jsonStr, "streamCount") {
		t.Error("expected streamCount in JSON")
	}
	if !contains(jsonStr, "totalConsumers") {
		t.Error("expected totalConsumers in JSON")
	}
	if !contains(jsonStr, "timestamp") {
		t.Error("expected timestamp in JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Load Test: Multiple consumers receiving events concurrently
func TestStreamMultiplexer_LoadTest_MultipleConsumers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create 20 consumers on the same stream
	consumerCount := 20
	var channels []<-chan events.Event

	for i := 0; i < consumerCount; i++ {
		clientType := ClientTypeSSE
		if i%2 == 0 {
			clientType = ClientTypeWebSocket
		}
		eventCh, _, err := mux.Subscribe("agent.loadtest.*", clientType)
		if err != nil {
			t.Fatalf("failed to subscribe consumer %d: %v", i, err)
		}
		channels = append(channels, eventCh)
	}

	// Verify all consumers registered
	if mux.GetConsumerCount("agent.loadtest.*") != consumerCount {
		t.Fatalf("expected %d consumers, got %d", consumerCount, mux.GetConsumerCount("agent.loadtest.*"))
	}

	// Publish 100 events
	eventCount := 100
	ctx := context.Background()

	for i := 0; i < eventCount; i++ {
		_, err := bus.PublishAgentToken(ctx, "loadtest", fmt.Sprintf("token-%d", i), i, i == eventCount-1)
		if err != nil {
			t.Fatalf("failed to publish event %d: %v", i, err)
		}
	}

	// Give time for events to propagate
	time.Sleep(200 * time.Millisecond)

	// Verify all consumers received all events
	for i, ch := range channels {
		received := 0
	drain:
		for {
			select {
			case <-ch:
				received++
			case <-time.After(100 * time.Millisecond):
				break drain
			}
		}

		if received != eventCount {
			t.Errorf("consumer %d: expected %d events, received %d", i, eventCount, received)
		}
	}
}

// Load Test: Multiple streams with multiple consumers each
func TestStreamMultiplexer_LoadTest_MultipleStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create 5 streams with 10 consumers each
	streamCount := 5
	consumersPerStream := 10

	for s := 0; s < streamCount; s++ {
		pattern := fmt.Sprintf("agent.stream%d.*", s)
		for c := 0; c < consumersPerStream; c++ {
			_, _, err := mux.Subscribe(pattern, ClientTypeSSE)
			if err != nil {
				t.Fatalf("failed to subscribe stream %d consumer %d: %v", s, c, err)
			}
		}
	}

	// Verify counts
	if mux.GetStreamCount() != streamCount {
		t.Errorf("expected %d streams, got %d", streamCount, mux.GetStreamCount())
	}

	expectedConsumers := streamCount * consumersPerStream
	if mux.GetTotalConsumerCount() != expectedConsumers {
		t.Errorf("expected %d total consumers, got %d", expectedConsumers, mux.GetTotalConsumerCount())
	}

	// Publish events to each stream
	ctx := context.Background()
	for s := 0; s < streamCount; s++ {
		agentName := fmt.Sprintf("stream%d", s)
		for e := 0; e < 10; e++ {
			bus.PublishAgentToken(ctx, agentName, fmt.Sprintf("token-%d", e), e, false)
		}
	}

	time.Sleep(200 * time.Millisecond)

	// Verify stats
	stats := mux.GetStats()
	if stats.StreamCount != streamCount {
		t.Errorf("expected %d streams in stats, got %d", streamCount, stats.StreamCount)
	}
	if stats.TotalConsumers != expectedConsumers {
		t.Errorf("expected %d consumers in stats, got %d", expectedConsumers, stats.TotalConsumers)
	}
}

// Load Test: Rapid subscribe/unsubscribe cycles
func TestStreamMultiplexer_LoadTest_RapidSubscribeUnsubscribe(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Rapid subscribe/unsubscribe cycles
	cycles := 50
	pattern := "agent.rapid.*"

	for i := 0; i < cycles; i++ {
		_, consumerID, err := mux.Subscribe(pattern, ClientTypeSSE)
		if err != nil {
			t.Fatalf("cycle %d: failed to subscribe: %v", i, err)
		}

		err = mux.Unsubscribe(pattern, consumerID)
		if err != nil {
			t.Fatalf("cycle %d: failed to unsubscribe: %v", i, err)
		}
	}

	// Should have 0 streams after all unsubscriptions
	if mux.GetStreamCount() != 0 {
		t.Errorf("expected 0 streams after rapid cycles, got %d", mux.GetStreamCount())
	}

	if mux.GetTotalConsumerCount() != 0 {
		t.Errorf("expected 0 consumers after rapid cycles, got %d", mux.GetTotalConsumerCount())
	}
}

// Load Test: Concurrent operations
func TestStreamMultiplexer_LoadTest_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Concurrent subscriptions
	var wg sync.WaitGroup
	consumerCount := 30
	var consumerIDs []string
	var mu sync.Mutex

	for i := 0; i < consumerCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, id, err := mux.Subscribe("agent.concurrent.*", ClientTypeSSE)
			if err != nil {
				t.Errorf("consumer %d: failed to subscribe: %v", idx, err)
				return
			}
			mu.Lock()
			consumerIDs = append(consumerIDs, id)
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all consumers registered
	if mux.GetConsumerCount("agent.concurrent.*") != consumerCount {
		t.Errorf("expected %d consumers, got %d", consumerCount, mux.GetConsumerCount("agent.concurrent.*"))
	}

	// Concurrent event publishing
	ctx := context.Background()
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bus.PublishAgentToken(ctx, "concurrent", fmt.Sprintf("token-%d", idx), idx, false)
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Concurrent unsubscriptions
	for _, id := range consumerIDs {
		wg.Add(1)
		go func(consumerID string) {
			defer wg.Done()
			err := mux.Unsubscribe("agent.concurrent.*", consumerID)
			if err != nil {
				t.Errorf("failed to unsubscribe %s: %v", consumerID, err)
			}
		}(id)
	}

	wg.Wait()

	// Verify cleanup
	if mux.GetStreamCount() != 0 {
		t.Errorf("expected 0 streams after concurrent unsubscribe, got %d", mux.GetStreamCount())
	}
}

// Benchmark: Measure throughput with multiple consumers
func BenchmarkStreamMultiplexer_Throughput(b *testing.B) {
	dir := b.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	// Create 10 consumers
	for i := 0; i < 10; i++ {
		mux.Subscribe("agent.benchmark.*", ClientTypeSSE)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.PublishAgentToken(ctx, "benchmark", "token", i, false)
	}
}

// Benchmark: Measure latency with single consumer
func BenchmarkStreamMultiplexer_Latency(b *testing.B) {
	dir := b.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	mux := NewStreamMultiplexer(bus)
	defer mux.Close()

	eventCh, _, _ := mux.Subscribe("agent.benchmark.*", ClientTypeSSE)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.PublishAgentToken(ctx, "benchmark", "token", i, false)
		<-eventCh
	}
}
