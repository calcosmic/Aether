package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

func TestSSEServerCommandExists(t *testing.T) {
	// Verify the serve command is registered
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			found = true
			break
		}
	}
	if !found {
		t.Error("serve command not found in root commands")
	}
}

func TestSSEServerFlags(t *testing.T) {
	var serveCmdFound *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			serveCmdFound = cmd
			break
		}
	}
	if serveCmdFound == nil {
		t.Fatal("serve command not found")
	}

	// Check port flag exists
	portFlag := serveCmdFound.Flags().Lookup("port")
	if portFlag == nil {
		t.Error("port flag not found")
	} else if portFlag.DefValue != "8080" {
		t.Errorf("expected default port 8080, got %q", portFlag.DefValue)
	}

	// Check host flag exists
	hostFlag := serveCmdFound.Flags().Lookup("host")
	if hostFlag == nil {
		t.Error("host flag not found")
	}

	// Check keepalive flag exists
	keepaliveFlag := serveCmdFound.Flags().Lookup("keepalive")
	if keepaliveFlag == nil {
		t.Error("keepalive flag not found")
	}
}

func TestSSEServerHealthEndpoint(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var response map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status healthy, got %q", response["status"])
	}
}

func TestSSEServerHealthMethodNotAllowed(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rr := httptest.NewRecorder()

	server.handleHealth(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestSSEServerSpecificAgentRequiresName(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	// Path without agent name
	req := httptest.NewRequest(http.MethodGet, "/sse/agents/", nil)
	rr := httptest.NewRecorder()

	server.handleSpecificAgent(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", rr.Code)
	}
}

func TestSSEServerSpecificAgentSetsCorrectHeaders(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	// Run handler in goroutine since it blocks
	done := make(chan bool)
	go func() {
		server.handleSpecificAgent(rr, req)
		done <- true
	}()

	// Give handler time to set up
	time.Sleep(50 * time.Millisecond)

	// Check headers were set before closing
	headers := rr.Header()
	if headers.Get("Content-Type") != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %q", headers.Get("Content-Type"))
	}
	if headers.Get("Cache-Control") != "no-cache" {
		t.Errorf("expected Cache-Control no-cache, got %q", headers.Get("Cache-Control"))
	}
	if headers.Get("Connection") != "keep-alive" {
		t.Errorf("expected Connection keep-alive, got %q", headers.Get("Connection"))
	}
	if headers.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected Access-Control-Allow-Origin *, got %q", headers.Get("Access-Control-Allow-Origin"))
	}

	// Cleanup - need to cancel the request context
	bus.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}
}

func TestSSEServerReceivesAgentTokenEvents(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	// Run handler in goroutine
	done := make(chan bool)
	go func() {
		server.handleSpecificAgent(rr, req)
		done <- true
	}()

	// Give handler time to subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish a token event
	ctx := req.Context()
	_, err := bus.PublishAgentToken(ctx, "builder", "Hello", 0, false)
	if err != nil {
		t.Fatalf("failed to publish token: %v", err)
	}

	// Give handler time to receive and write
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	bus.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}

	// Check response body contains SSE formatted event
	body := rr.Body.String()
	if !strings.Contains(body, "event: connected") {
		t.Errorf("expected connected event, got body: %q", body)
	}
	if !strings.Contains(body, "data: {") {
		t.Errorf("expected SSE data prefix, got body: %q", body)
	}
	if !strings.Contains(body, "\n\n") {
		t.Errorf("expected double newline SSE terminator, got body: %q", body)
	}
}

func TestSSEServerAllAgentsReceivesEvents(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents", nil)
	rr := httptest.NewRecorder()

	// Run handler in goroutine
	done := make(chan bool)
	go func() {
		server.handleAllAgents(rr, req)
		done <- true
	}()

	// Give handler time to subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish token events for different agents
	ctx := req.Context()
	bus.PublishAgentToken(ctx, "builder", "builder token", 0, false)
	bus.PublishAgentToken(ctx, "watcher", "watcher token", 0, false)

	// Give handler time to receive and write
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	bus.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}

	// Check response body contains both events
	body := rr.Body.String()
	if !strings.Contains(body, "builder token") {
		t.Errorf("expected builder token in body, got: %q", body)
	}
	if !strings.Contains(body, "watcher token") {
		t.Errorf("expected watcher token in body, got: %q", body)
	}
}

func TestSSEServerSpecificAgentFiltersEvents(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	// Run handler in goroutine
	done := make(chan bool)
	go func() {
		server.handleSpecificAgent(rr, req)
		done <- true
	}()

	// Give handler time to subscribe
	time.Sleep(100 * time.Millisecond)

	// Publish token events for different agents
	ctx := req.Context()
	bus.PublishAgentToken(ctx, "builder", "builder token", 0, false)
	bus.PublishAgentToken(ctx, "watcher", "watcher token", 0, false)

	// Give handler time to receive and write
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	bus.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}

	// Check response body only contains builder events
	body := rr.Body.String()
	if !strings.Contains(body, "builder token") {
		t.Errorf("expected builder token in body, got: %q", body)
	}
	if strings.Contains(body, "watcher token") {
		t.Errorf("should not contain watcher token, got: %q", body)
	}
}

func TestSSEServerMultipleEvents(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	done := make(chan bool)
	go func() {
		server.handleSpecificAgent(rr, req)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish multiple events
	ctx := req.Context()
	for i := 0; i < 3; i++ {
		_, err := bus.PublishAgentToken(ctx, "builder", fmt.Sprintf("token%d", i), i, i == 2)
		if err != nil {
			t.Fatalf("failed to publish token %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	bus.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}

	body := rr.Body.String()
	// Count data: prefixes (should be 3 + 1 for connected event)
	count := strings.Count(body, "data: {")
	if count < 3 {
		t.Errorf("expected at least 3 data events, got %d\nBody: %q", count, body)
	}
}

func TestSSEServerMethodNotAllowed(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodPost, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	server.handleSpecificAgent(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", rr.Code)
	}
}

func TestSSEServerWriteEventFormat(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	server := newStreamingServer(bus, 30, false)

	req := httptest.NewRequest(http.MethodGet, "/sse/agents/builder", nil)
	rr := httptest.NewRecorder()

	// Set up the response writer for SSE
	rr.Header().Set("Content-Type", "text/event-stream")
	flusher := rr.Result().Body
	_ = flusher

	// Create a test event
	tokenEvt := events.AgentTokenEvent{
		AgentName:  "builder",
		Content:    "test content",
		Timestamp:  "2026-04-06T12:00:00Z",
		IsComplete: false,
		Sequence:   1,
	}
	_ = tokenEvt

	// Write event through the server
	done := make(chan bool)
	go func() {
		// We can't easily test writeEvent in isolation due to flusher requirement,
		// but we can test the full flow
		server.handleSpecificAgent(rr, req)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish the event
	ctx := req.Context()
	bus.PublishAgentToken(ctx, "builder", "test content", 1, false)

	time.Sleep(100 * time.Millisecond)
	bus.Close()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Error("handler did not complete")
	}

	body := rr.Body.String()
	// Verify SSE format elements
	if !strings.Contains(body, "event:") {
		t.Errorf("expected event: field in SSE output, got: %q", body)
	}
	if !strings.Contains(body, "id:") {
		t.Errorf("expected id: field in SSE output, got: %q", body)
	}
	if !strings.Contains(body, "data: {") {
		t.Errorf("expected data: field in SSE output, got: %q", body)
	}
}

func TestMustJSON(t *testing.T) {
	// Test valid JSON
	result := mustJSON(map[string]string{"key": "value"})
	if !strings.Contains(result, `"key"`) {
		t.Errorf("expected JSON with key field, got %q", result)
	}

	// Test with nested structure
	result2 := mustJSON(map[string]interface{}{
		"nested": map[string]string{"inner": "data"},
	})
	if !strings.Contains(result2, `"nested"`) {
		t.Errorf("expected JSON with nested field, got %q", result2)
	}
}


// ===================== WebSocket Tests =====================

// setupWebSocketTestServer is a test helper that creates a WebSocket server for testing
func setupWebSocketTestServer(t *testing.T) (*streamingServer, *events.Bus, func()) {
	dir := t.TempDir()
	store, err := storage.NewStore(dir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	bus := events.NewBus(store, events.DefaultConfig())
	server := newStreamingServer(bus, 30, false)

	cleanup := func() {
		bus.Close()
	}

	return server, bus, cleanup
}

// TestWebSocketCommandFlags verifies WebSocket-related flags exist
func TestWebSocketCommandFlags(t *testing.T) {
	var serveCmdFound *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "serve" {
			serveCmdFound = cmd
			break
		}
	}
	if serveCmdFound == nil {
		t.Fatal("serve command not found")
	}

	// Check ws-local-only flag exists
	wsLocalOnlyFlag := serveCmdFound.Flags().Lookup("ws-local-only")
	if wsLocalOnlyFlag == nil {
		t.Error("ws-local-only flag not found")
	} else if wsLocalOnlyFlag.DefValue != "true" {
		t.Errorf("expected default ws-local-only true, got %q", wsLocalOnlyFlag.DefValue)
	}
}

// TestWebSocketAllAgentsConnection tests WebSocket connection for all agents
func TestWebSocketAllAgentsConnection(t *testing.T) {
	server, _, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server with WebSocket handler
	ts := httptest.NewServer(http.HandlerFunc(server.handleWebSocketAllAgents))
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v (status: %d)", err, resp.StatusCode)
	}
	defer conn.Close()

	// Read initial connected event
	var msg wsEvent
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read connected event: %v", err)
	}

	if msg.Type != "connected" {
		t.Errorf("expected type 'connected', got %q", msg.Type)
	}
	if msg.Topic != "system" {
		t.Errorf("expected topic 'system', got %q", msg.Topic)
	}
}

// TestWebSocketSpecificAgentConnection tests WebSocket connection for specific agent
func TestWebSocketSpecificAgentConnection(t *testing.T) {
	server, _, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server with WebSocket handler
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract agent name from path for test
		r.URL.Path = "/ws/agents/builder"
		server.handleWebSocketSpecificAgent(w, r)
	}))
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v (status: %d)", err, resp.StatusCode)
	}
	defer conn.Close()

	// Read initial connected event
	var msg wsEvent
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read connected event: %v", err)
	}

	if msg.Type != "connected" {
		t.Errorf("expected type 'connected', got %q", msg.Type)
	}
}

// TestWebSocketReceivesAgentTokenEvents tests that WebSocket receives token events
func TestWebSocketReceivesAgentTokenEvents(t *testing.T) {
	server, bus, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/ws/agents/builder"
		server.handleWebSocketSpecificAgent(w, r)
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read connected event
	var connectedMsg wsEvent
	if err := conn.ReadJSON(&connectedMsg); err != nil {
		t.Fatalf("failed to read connected event: %v", err)
	}

	// Publish a token event
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = bus.PublishAgentToken(ctx, "builder", "Hello WebSocket", 0, false)
	if err != nil {
		t.Fatalf("failed to publish token: %v", err)
	}

	// Read token event
	var tokenMsg wsEvent
	if err := conn.ReadJSON(&tokenMsg); err != nil {
		t.Fatalf("failed to read token event: %v", err)
	}

	if tokenMsg.Type != "tokens" {
		t.Errorf("expected type 'tokens', got %q", tokenMsg.Type)
	}
	if tokenMsg.Topic != "agent.builder.tokens" {
		t.Errorf("expected topic 'agent.builder.tokens', got %q", tokenMsg.Topic)
	}

	// Verify data contains the token content
	var tokenData events.AgentTokenEvent
	if err := json.Unmarshal(tokenMsg.Data, &tokenData); err != nil {
		t.Fatalf("failed to unmarshal token data: %v", err)
	}
	if tokenData.Content != "Hello WebSocket" {
		t.Errorf("expected content 'Hello WebSocket', got %q", tokenData.Content)
	}
}

// TestWebSocketAllAgentsReceivesMultipleAgentEvents tests that /ws/agents receives events from all agents
func TestWebSocketAllAgentsReceivesMultipleAgentEvents(t *testing.T) {
	server, bus, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(server.handleWebSocketAllAgents))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read connected event
	var connectedMsg wsEvent
	conn.ReadJSON(&connectedMsg)

	// Publish token events for different agents
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "builder token", 0, false)
	bus.PublishAgentToken(ctx, "watcher", "watcher token", 0, false)

	// Read both events
	receivedBuilders := 0
	receivedWatchers := 0
	for i := 0; i < 2; i++ {
		var msg wsEvent
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("failed to read event %d: %v", i, err)
		}

		var tokenData events.AgentTokenEvent
		json.Unmarshal(msg.Data, &tokenData)

		if tokenData.AgentName == "builder" {
			receivedBuilders++
		}
		if tokenData.AgentName == "watcher" {
			receivedWatchers++
		}
	}

	if receivedBuilders != 1 {
		t.Errorf("expected 1 builder event, got %d", receivedBuilders)
	}
	if receivedWatchers != 1 {
		t.Errorf("expected 1 watcher event, got %d", receivedWatchers)
	}
}

// TestWebSocketSpecificAgentFiltersEvents tests that specific agent endpoint filters events
func TestWebSocketSpecificAgentFiltersEvents(t *testing.T) {
	server, bus, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/ws/agents/builder"
		server.handleWebSocketSpecificAgent(w, r)
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read connected event
	var connectedMsg wsEvent
	conn.ReadJSON(&connectedMsg)

	// Publish token events for different agents
	ctx := context.Background()
	bus.PublishAgentToken(ctx, "builder", "builder token", 0, false)
	bus.PublishAgentToken(ctx, "watcher", "watcher token", 0, false)

	// Should only receive builder event
	var msg wsEvent
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	var tokenData events.AgentTokenEvent
	json.Unmarshal(msg.Data, &tokenData)

	if tokenData.AgentName != "builder" {
		t.Errorf("expected builder event, got %q", tokenData.AgentName)
	}

	// Set read deadline to check if more messages come (shouldn't)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var extraMsg wsEvent
	extraErr := conn.ReadJSON(&extraMsg)
	if extraErr == nil {
		t.Error("expected no more messages after builder event, but received one")
	}
}

// TestWebSocketLocalhostRestriction tests that WebSocket connections can be restricted to localhost
// TestWebSocketLocalhostRestriction tests that WebSocket connections can be restricted to localhost
func TestWebSocketLocalhostRestriction(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewStore(dir)
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	// Create server with localhost-only restriction
	server := newStreamingServer(bus, 30, true)

	// Test that a request from a non-localhost IP is rejected
	req := httptest.NewRequest(http.MethodGet, "/ws/agents", nil)
	req.Host = "192.168.1.1:8080" // Simulate non-localhost request
	rr := httptest.NewRecorder()

	server.handleWebSocketAllAgents(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected status 403 for non-localhost request, got %d", rr.Code)
	}
}
func TestWebSocketConnectionClosesOnBusClose(t *testing.T) {
	server, bus, _ := setupWebSocketTestServer(t)

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(server.handleWebSocketAllAgents))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read connected event
	var connectedMsg wsEvent
	conn.ReadJSON(&connectedMsg)

	// Close the bus
	bus.Close()

	// Connection should receive disconnected event or close
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var msg wsEvent
	err = conn.ReadJSON(&msg)
	// We expect an error or disconnected message
	if err == nil && msg.Type != "disconnected" {
		t.Errorf("expected disconnected event or error, got type %q", msg.Type)
	}
}

// TestWebSocketEventFormat tests that WebSocket events have correct format
func TestWebSocketEventFormat(t *testing.T) {
	server, _, cleanup := setupWebSocketTestServer(t)
	defer cleanup()

	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/ws/agents/builder"
		server.handleWebSocketSpecificAgent(w, r)
	}))
	defer ts.Close()

	wsURL := strings.Replace(ts.URL, "http://", "ws://", 1)

	// Connect to WebSocket
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	// Read connected event and verify format
	var msg wsEvent
	if err := conn.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read event: %v", err)
	}

	// Verify all required fields are present
	if msg.Type == "" {
		t.Error("expected Type field to be set")
	}
	if msg.ID == "" {
		t.Error("expected ID field to be set")
	}
	if msg.Topic == "" {
		t.Error("expected Topic field to be set")
	}
	if msg.Timestamp == "" {
		t.Error("expected Timestamp field to be set")
	}

	// Verify timestamp is valid RFC3339
	_, err = time.Parse(time.RFC3339, msg.Timestamp)
	if err != nil {
		t.Errorf("expected valid RFC3339 timestamp, got %q: %v", msg.Timestamp, err)
	}
}

// TestWebSocketClientHelper tests the WebSocket client helper
func TestWebSocketClientHelper(t *testing.T) {
	// Test that newWSClient creates a client with correct settings
	client := newWSClient("http://localhost:8080")
	if client == nil {
		t.Fatal("expected client to be created")
	}
	if client.baseURL != "ws://localhost:8080" {
		t.Errorf("expected baseURL ws://localhost:8080, got %q", client.baseURL)
	}
	if client.dialer == nil {
		t.Error("expected dialer to be set")
	}
	if client.dialer.HandshakeTimeout != 10*time.Second {
		t.Errorf("expected handshake timeout 10s, got %v", client.dialer.HandshakeTimeout)
	}
}

// TestIsLocalhost tests the localhost detection function
func TestIsLocalhost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected bool
	}{
		{"localhost with port", "localhost:8080", true},
		{"127.0.0.1 with port", "127.0.0.1:8080", true},
		{"IPv6 localhost", "[::1]:8080", true},
		{"remote host", "example.com:8080", false},
		{"remote IP", "192.168.1.1:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Host = tt.host
			result := isLocalhost(req)
			if result != tt.expected {
				t.Errorf("isLocalhost(%q) = %v, expected %v", tt.host, result, tt.expected)
			}
		})
	}
}
