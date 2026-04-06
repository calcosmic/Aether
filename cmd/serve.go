// Package cmd implements the Aether CLI commands using Cobra.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

// serveCmd represents the serve command for starting an SSE/WebSocket server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an SSE/WebSocket server for agent stream consumption",
	Long: `Start an HTTP server that exposes Server-Sent Events (SSE) and WebSocket endpoints
for consuming agent streams in real-time.

The server provides endpoints for subscribing to agent events:
  GET  /sse/agents          - Subscribe to all agent events (SSE)
  GET  /sse/agents/{name}   - Subscribe to events from a specific agent (SSE)
  GET  /ws/agents           - WebSocket for all agent events
  GET  /ws/agents/{name}    - WebSocket for specific agent events
  GET  /health              - Health check endpoint

Events are streamed as SSE with proper headers and formatting.
WebSocket connections provide the same events with lower latency.
The server only starts when explicitly requested and runs until interrupted.`,
	RunE: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntP("port", "p", 8080, "Port to listen on")
	serveCmd.Flags().StringP("host", "H", "localhost", "Host to bind to")
	serveCmd.Flags().Int("keepalive", 30, "Keepalive interval in seconds")
	serveCmd.Flags().Bool("ws-local-only", true, "Only allow WebSocket connections from localhost")
}

// runServe starts the SSE/WebSocket server.
func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	host, _ := cmd.Flags().GetString("host")
	keepalive, _ := cmd.Flags().GetInt("keepalive")
	wsLocalOnly, _ := cmd.Flags().GetBool("ws-local-only")

	if store == nil {
		return fmt.Errorf("store not initialized")
	}

	// Create event bus
	bus := events.NewBus(store, events.DefaultConfig())
	defer bus.Close()

	// Create server
	server := newStreamingServer(bus, keepalive, wsLocalOnly)

	// Setup routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/sse/agents", server.handleAllAgents)
	mux.HandleFunc("/sse/agents/", server.handleSpecificAgent)
	mux.HandleFunc("/ws/agents", server.handleWebSocketAllAgents)
	mux.HandleFunc("/ws/agents/", server.handleWebSocketSpecificAgent)

	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Fprintf(stdout, "Starting streaming server on http://%s\n", addr)
	fmt.Fprintf(stdout, "SSE Endpoints:\n")
	fmt.Fprintf(stdout, "  GET /sse/agents        - Subscribe to all agent events\n")
	fmt.Fprintf(stdout, "  GET /sse/agents/{name} - Subscribe to specific agent events\n")
	fmt.Fprintf(stdout, "WebSocket Endpoints:\n")
	fmt.Fprintf(stdout, "  GET /ws/agents         - WebSocket for all agent events\n")
	fmt.Fprintf(stdout, "  GET /ws/agents/{name}  - WebSocket for specific agent events\n")
	fmt.Fprintf(stdout, "  GET /health            - Health check\n")
	fmt.Fprintf(stdout, "\nPress Ctrl+C to stop\n")

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // No timeout for SSE/WebSocket connections
		IdleTimeout:  60 * time.Second,
	}

	return srv.ListenAndServe()
}

// streamingServer handles SSE and WebSocket connections and event streaming.
type streamingServer struct {
	bus           *events.Bus
	keepalive     time.Duration
	wsLocalOnly   bool
	upgrader      websocket.Upgrader
	wsConnections sync.Map // tracks active WebSocket connections for cleanup
}

// newStreamingServer creates a new streaming server instance.
func newStreamingServer(bus *events.Bus, keepaliveSecs int, wsLocalOnly bool) *streamingServer {
	if keepaliveSecs <= 0 {
		keepaliveSecs = 30
	}
	return &streamingServer{
		bus:         bus,
		keepalive:   time.Duration(keepaliveSecs) * time.Second,
		wsLocalOnly: wsLocalOnly,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for local development
				// In production, this should be restricted
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

// sseEvent represents an event formatted for SSE transmission.
type sseEvent struct {
	ID    string          `json:"id"`
	Topic string          `json:"topic"`
	Data  json.RawMessage `json:"data"`
	Time  string          `json:"time"`
}

// handleHealth responds to health check requests.
func (s *streamingServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

// handleAllAgents handles SSE connections for all agent events.
func (s *streamingServer) handleAllAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Subscribe to all agent events
	sub, err := s.bus.Subscribe("agent.*")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to subscribe: %v", err), http.StatusInternalServerError)
		return
	}
	defer s.bus.Unsubscribe("agent.*", sub)

	s.serveSSE(w, r, sub, "all agents")
}

// handleSpecificAgent handles SSE connections for a specific agent.
func (s *streamingServer) handleSpecificAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract agent name from path
	// Path format: /sse/agents/{name}
	path := strings.TrimPrefix(r.URL.Path, "/sse/agents/")
	if path == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	agentName := strings.Split(path, "/")[0]
	if agentName == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	// Subscribe to specific agent events
	topic := fmt.Sprintf("agent.%s.*", agentName)
	sub, err := s.bus.Subscribe(topic)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to subscribe: %v", err), http.StatusInternalServerError)
		return
	}
	defer s.bus.Unsubscribe(topic, sub)

	s.serveSSE(w, r, sub, agentName)
}

// serveSSE handles the SSE connection lifecycle and event streaming.
func (s *streamingServer) serveSSE(w http.ResponseWriter, r *http.Request, sub <-chan events.Event, agentDesc string) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\n")
	fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]string{
		"agent":   agentDesc,
		"message": "Connected to agent event stream",
		"time":    time.Now().UTC().Format(time.RFC3339),
	}))
	flusher.Flush()

	// Create keepalive ticker
	keepalive := time.NewTicker(s.keepalive)
	defer keepalive.Stop()

	// Stream events
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return

		case evt, ok := <-sub:
			if !ok {
				// Subscription closed
				fmt.Fprintf(w, "event: disconnected\n")
				fmt.Fprintf(w, "data: %s\n\n", mustJSON(map[string]string{
					"message": "Subscription closed",
					"time":    time.Now().UTC().Format(time.RFC3339),
				}))
				flusher.Flush()
				return
			}

			// Format and send event
			if err := s.writeEvent(w, flusher, evt); err != nil {
				// Client likely disconnected
				return
			}

		case <-keepalive.C:
			// Send keepalive comment
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// writeEvent formats and writes an event to the SSE stream.
func (s *streamingServer) writeEvent(w http.ResponseWriter, flusher http.Flusher, evt events.Event) error {
	// Build SSE event
	// Format: event: <type>\nid: <id>\ndata: <json>\n\n

	// Extract event type from topic (e.g., "agent.builder.token" -> "token")
	eventType := "message"
	if strings.HasPrefix(evt.Topic, "agent.") {
		parts := strings.Split(evt.Topic, ".")
		if len(parts) >= 3 {
			eventType = parts[len(parts)-1]
		}
	}

	// Create event data
	data := sseEvent{
		ID:    evt.ID,
		Topic: evt.Topic,
		Data:  evt.Payload,
		Time:  evt.Timestamp,
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// Write SSE format
	fmt.Fprintf(w, "event: %s\n", eventType)
	fmt.Fprintf(w, "id: %s\n", evt.ID)
	fmt.Fprintf(w, "data: %s\n\n", string(dataJSON))

	flusher.Flush()
	return nil
}

// mustJSON marshals v to JSON or returns empty object on error.
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// sseClient is a helper for connecting to SSE endpoints.
type sseClient struct {
	baseURL string
	client  *http.Client
}

// newSSEClient creates a new SSE client.
func newSSEClient(baseURL string) *sseClient {
	return &sseClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 0, // No timeout for SSE
		},
	}
}

// connect establishes an SSE connection and returns the response.
func (c *sseClient) connect(ctx context.Context, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	return c.client.Do(req)
}

// ===================== WebSocket Support =====================

// wsEvent represents an event formatted for WebSocket transmission.
type wsEvent struct {
	Type      string          `json:"type"`      // Event type (token, connected, disconnected, etc.)
	ID        string          `json:"id"`        // Event ID
	Topic     string          `json:"topic"`     // Event topic
	Data      json.RawMessage `json:"data"`      // Event payload
	Timestamp string          `json:"timestamp"` // Event timestamp
}

// isLocalhost checks if the request is from localhost.
func isLocalhost(r *http.Request) bool {
	host := r.Host
	if host == "" {
		host = r.RemoteAddr
	}
	// Check for localhost variants
	return strings.HasPrefix(host, "localhost:") ||
		strings.HasPrefix(host, "127.0.0.1:") ||
		strings.HasPrefix(host, "[::1]:") ||
		strings.Contains(host, "127.0.0.1:") ||
		strings.Contains(host, "[::1]:")
}

// handleWebSocketAllAgents handles WebSocket connections for all agent events.
func (s *streamingServer) handleWebSocketAllAgents(w http.ResponseWriter, r *http.Request) {
	// Check local-only restriction
	if s.wsLocalOnly && !isLocalhost(r) {
		http.Error(w, "WebSocket connections only allowed from localhost", http.StatusForbidden)
		return
	}

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote error response
		return
	}

	// Track connection for cleanup
	connID := fmt.Sprintf("ws-all-%d", time.Now().UnixNano())
	s.wsConnections.Store(connID, conn)
	defer s.wsConnections.Delete(connID)
	defer conn.Close()

	// Subscribe to all agent events
	sub, err := s.bus.Subscribe("agent.*")
	if err != nil {
		s.writeWSError(conn, fmt.Sprintf("Failed to subscribe: %v", err))
		return
	}
	defer s.bus.Unsubscribe("agent.*", sub)

	s.serveWebSocket(conn, sub, "all agents")
}

// handleWebSocketSpecificAgent handles WebSocket connections for a specific agent.
func (s *streamingServer) handleWebSocketSpecificAgent(w http.ResponseWriter, r *http.Request) {
	// Check local-only restriction
	if s.wsLocalOnly && !isLocalhost(r) {
		http.Error(w, "WebSocket connections only allowed from localhost", http.StatusForbidden)
		return
	}

	// Extract agent name from path
	// Path format: /ws/agents/{name}
	path := strings.TrimPrefix(r.URL.Path, "/ws/agents/")
	if path == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	agentName := strings.Split(path, "/")[0]
	if agentName == "" {
		http.Error(w, "Agent name required", http.StatusBadRequest)
		return
	}

	// Upgrade to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote error response
		return
	}

	// Track connection for cleanup
	connID := fmt.Sprintf("ws-%s-%d", agentName, time.Now().UnixNano())
	s.wsConnections.Store(connID, conn)
	defer s.wsConnections.Delete(connID)
	defer conn.Close()

	// Subscribe to specific agent events
	topic := fmt.Sprintf("agent.%s.*", agentName)
	sub, err := s.bus.Subscribe(topic)
	if err != nil {
		s.writeWSError(conn, fmt.Sprintf("Failed to subscribe: %v", err))
		return
	}
	defer s.bus.Unsubscribe(topic, sub)

	s.serveWebSocket(conn, sub, agentName)
}

// serveWebSocket handles the WebSocket connection lifecycle and event streaming.
func (s *streamingServer) serveWebSocket(conn *websocket.Conn, sub <-chan events.Event, agentDesc string) {
	// Send initial connection event
	if err := conn.WriteJSON(wsEvent{
		Type:      "connected",
		ID:        fmt.Sprintf("evt_%d_0000", time.Now().Unix()),
		Topic:     "system",
		Data:      json.RawMessage(mustJSON(map[string]string{"agent": agentDesc, "message": "Connected to agent event stream"})),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		// Client likely disconnected
		return
	}

	// Create done channel for signaling disconnect
	done := make(chan struct{})
	defer close(done)

	// Start goroutine to handle client disconnect
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// Client disconnected or error
				return
			}
		}
	}()

	// Stream events
	for {
		select {
		case <-done:
			// Client disconnected
			return

		case evt, ok := <-sub:
			if !ok {
				// Subscription closed
				conn.WriteJSON(wsEvent{
					Type:      "disconnected",
					ID:        fmt.Sprintf("evt_%d_0001", time.Now().Unix()),
					Topic:     "system",
					Data:      json.RawMessage(mustJSON(map[string]string{"message": "Subscription closed"})),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
				})
				return
			}

			// Format and send event
			if err := s.writeWebSocketEvent(conn, evt); err != nil {
				// Client likely disconnected
				return
			}
		}
	}
}

// writeWebSocketEvent formats and writes an event to the WebSocket connection.
func (s *streamingServer) writeWebSocketEvent(conn *websocket.Conn, evt events.Event) error {
	// Extract event type from topic (e.g., "agent.builder.token" -> "token")
	eventType := "message"
	if strings.HasPrefix(evt.Topic, "agent.") {
		parts := strings.Split(evt.Topic, ".")
		if len(parts) >= 3 {
			eventType = parts[len(parts)-1]
		}
	}

	wsEvt := wsEvent{
		Type:      eventType,
		ID:        evt.ID,
		Topic:     evt.Topic,
		Data:      evt.Payload,
		Timestamp: evt.Timestamp,
	}

	return conn.WriteJSON(wsEvt)
}

// writeWSError writes an error message to the WebSocket connection.
func (s *streamingServer) writeWSError(conn *websocket.Conn, message string) {
	conn.WriteJSON(wsEvent{
		Type:      "error",
		ID:        fmt.Sprintf("evt_%d_0000", time.Now().Unix()),
		Topic:     "system",
		Data:      json.RawMessage(mustJSON(map[string]string{"message": message})),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// closeAllWebSockets closes all active WebSocket connections.
// This should be called when the server is shutting down.
func (s *streamingServer) closeAllWebSockets() {
	s.wsConnections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*websocket.Conn); ok {
			conn.WriteJSON(wsEvent{
				Type:      "disconnected",
				ID:        fmt.Sprintf("evt_%d_9999", time.Now().Unix()),
				Topic:     "system",
				Data:      json.RawMessage(mustJSON(map[string]string{"message": "Server shutting down"})),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
			})
			conn.Close()
		}
		return true
	})
}

// ===================== WebSocket Client =====================

// wsClient is a helper for connecting to WebSocket endpoints.
type wsClient struct {
	baseURL string
	dialer  *websocket.Dialer
}

// newWSClient creates a new WebSocket client.
func newWSClient(baseURL string) *wsClient {
	return &wsClient{
		baseURL: strings.Replace(baseURL, "http://", "ws://", 1),
		dialer: &websocket.Dialer{
			HandshakeTimeout: 10 * time.Second,
		},
	}
}

// connect establishes a WebSocket connection.
func (c *wsClient) connect(ctx context.Context, path string) (*websocket.Conn, *http.Response, error) {
	wsURL := c.baseURL + path
	return c.dialer.DialContext(ctx, wsURL, nil)
}
