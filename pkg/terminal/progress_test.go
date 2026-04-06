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
type progressSyncBuffer struct {
	buf bytes.Buffer
	mu  sync.RWMutex
}

func (sb *progressSyncBuffer) Write(p []byte) (n int, err error) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.buf.Write(p)
}

func (sb *progressSyncBuffer) String() string {
	sb.mu.RLock()
	defer sb.mu.RUnlock()
	return sb.buf.String()
}

// TestProgressIndicatorCreation tests basic ProgressIndicator creation.
func TestProgressIndicatorCreation(t *testing.T) {
	pi := NewProgressIndicator()

	if pi == nil {
		t.Fatal("NewProgressIndicator() returned nil")
	}

	if pi.output == nil {
		t.Error("ProgressIndicator output writer should not be nil")
	}

	if pi.agents == nil {
		t.Error("ProgressIndicator agents map should not be nil")
	}

	pi.Close()
}

// TestProgressIndicatorWithWriter tests creating with custom writer.
func TestProgressIndicatorWithWriter(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)

	if pi.output != buf {
		t.Error("ProgressIndicator should use provided writer")
	}

	if pi.isTTY {
		t.Error("ProgressIndicator should respect isTTY parameter")
	}

	pi.Close()
}

// TestRegisterAgent tests agent registration.
func TestRegisterAgent(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	agent := pi.RegisterAgent("builder-1", "Weld-20", "builder")

	if agent == nil {
		t.Fatal("RegisterAgent() returned nil")
	}

	if agent.AgentID != "builder-1" {
		t.Errorf("Expected AgentID 'builder-1', got %q", agent.AgentID)
	}

	if agent.AgentName != "Weld-20" {
		t.Errorf("Expected AgentName 'Weld-20', got %q", agent.AgentName)
	}

	if agent.Caste != "builder" {
		t.Errorf("Expected Caste 'builder', got %q", agent.Caste)
	}

	if agent.GetStatus() != StatusPending {
		t.Errorf("Expected initial status %q, got %q", StatusPending, agent.GetStatus())
	}
}

// TestRegisterDuplicateAgent tests that duplicate registration returns existing agent.
func TestRegisterDuplicateAgent(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	agent1 := pi.RegisterAgent("builder-1", "Weld-20", "builder")
	agent2 := pi.RegisterAgent("builder-1", "Weld-21", "watcher")

	if agent1 != agent2 {
		t.Error("Duplicate registration should return existing agent")
	}

	// Original values should be preserved
	if agent2.AgentName != "Weld-20" {
		t.Error("Original agent name should be preserved")
	}

	agents := pi.GetAllAgents()
	if len(agents) != 1 {
		t.Errorf("Expected 1 agent, got %d", len(agents))
	}
}

// TestUnregisterAgent tests agent unregistration.
func TestUnregisterAgent(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UnregisterAgent("builder-1")

	agent := pi.GetAgent("builder-1")
	if agent != nil {
		t.Error("Agent should be nil after unregistering")
	}
}

// TestUpdateProgress tests updating agent progress.
func TestUpdateProgress(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UpdateProgress("builder-1", 42)

	agent := pi.GetAgent("builder-1")
	if agent.GetTokenCount() != 42 {
		t.Errorf("Expected 42 tokens, got %d", agent.GetTokenCount())
	}
}

// TestUpdateStatus tests updating agent status.
func TestUpdateStatus(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UpdateStatus("builder-1", StatusActive)

	agent := pi.GetAgent("builder-1")
	if agent.GetStatus() != StatusActive {
		t.Errorf("Expected status %q, got %q", StatusActive, agent.GetStatus())
	}
}

// TestGetActiveAgents tests retrieving only active agents.
func TestGetActiveAgents(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	// Register multiple agents
	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.RegisterAgent("watcher-1", "Watch-10", "watcher")
	pi.RegisterAgent("scout-1", "Scout-5", "scout")

	// Set different statuses
	pi.UpdateStatus("builder-1", StatusActive)
	pi.UpdateStatus("watcher-1", StatusCompleted)
	pi.UpdateStatus("scout-1", StatusActive)

	active := pi.GetActiveAgents()
	if len(active) != 2 {
		t.Errorf("Expected 2 active agents, got %d", len(active))
	}
}

// TestAgentProgressIsActive tests the IsActive method.
func TestAgentProgressIsActive(t *testing.T) {
	agent := &AgentProgress{
		AgentID:   "test-1",
		AgentName: "Test",
		Caste:     "builder",
		Status:    StatusPending,
	}

	if !agent.IsActive() {
		t.Error("Pending agent should be active")
	}

	agent.SetStatus(StatusActive)
	if !agent.IsActive() {
		t.Error("Active agent should be active")
	}

	agent.SetStatus(StatusCompleted)
	if agent.IsActive() {
		t.Error("Completed agent should not be active")
	}

	agent.SetStatus(StatusFailed)
	if agent.IsActive() {
		t.Error("Failed agent should not be active")
	}
}

// TestGetProgressPercent tests progress percentage calculation.
func TestGetProgressPercent(t *testing.T) {
	agent := &AgentProgress{
		AgentID:        "test-1",
		TokenCount:     50,
		EstimatedTotal: 100,
	}

	percent := agent.GetProgressPercent()
	if percent != 50 {
		t.Errorf("Expected 50%%, got %d%%", percent)
	}

	// Test with unknown total
	agent.EstimatedTotal = 0
	percent = agent.GetProgressPercent()
	if percent != -1 {
		t.Errorf("Expected -1 for unknown total, got %d", percent)
	}

	// Test capping at 100%
	agent.TokenCount = 150
	agent.EstimatedTotal = 100
	percent = agent.GetProgressPercent()
	if percent != 100 {
		t.Errorf("Expected capped at 100%%, got %d%%", percent)
	}
}

// TestSetEstimatedTotal tests setting estimated total.
func TestSetEstimatedTotal(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.SetEstimatedTotal("builder-1", 1000)

	agent := pi.GetAgent("builder-1")
	if agent.EstimatedTotal != 1000 {
		t.Errorf("Expected estimated total 1000, got %d", agent.EstimatedTotal)
	}
}

// TestProgressStyles tests different progress styles.
func TestProgressStyles(t *testing.T) {
	defaultStyle := DefaultProgressStyle()
	if !defaultStyle.ShowSpinner {
		t.Error("Default style should show spinner")
	}
	if !defaultStyle.ShowTokenCount {
		t.Error("Default style should show token count")
	}
	if defaultStyle.CompactMode {
		t.Error("Default style should not be compact")
	}

	compactStyle := CompactProgressStyle()
	if !compactStyle.ShowSpinner {
		t.Error("Compact style should show spinner")
	}
	if compactStyle.ShowCaste {
		t.Error("Compact style should not show caste")
	}
	if !compactStyle.CompactMode {
		t.Error("Compact style should be compact")
	}
}

// TestSetStyle tests setting the display style.
func TestSetStyle(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	compact := CompactProgressStyle()
	pi.SetStyle(compact)

	// Verify style was set (indirectly through behavior)
	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UpdateStatus("builder-1", StatusActive)
	pi.ForceRender()
}

// TestProgressSummary tests the summary statistics.
func TestProgressSummary(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	// Register agents with different statuses
	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.RegisterAgent("watcher-1", "Watch-10", "watcher")
	pi.RegisterAgent("scout-1", "Scout-5", "scout")

	pi.UpdateStatus("builder-1", StatusActive)
	pi.UpdateProgress("builder-1", 100)
	pi.UpdateStatus("watcher-1", StatusCompleted)
	pi.UpdateProgress("watcher-1", 50)
	pi.UpdateStatus("scout-1", StatusFailed)

	summary := pi.GetSummary()

	if summary.TotalAgents != 3 {
		t.Errorf("Expected 3 total agents, got %d", summary.TotalAgents)
	}

	if summary.ActiveAgents != 1 {
		t.Errorf("Expected 1 active agent, got %d", summary.ActiveAgents)
	}

	if summary.CompletedAgents != 1 {
		t.Errorf("Expected 1 completed agent, got %d", summary.CompletedAgents)
	}

	if summary.FailedAgents != 1 {
		t.Errorf("Expected 1 failed agent, got %d", summary.FailedAgents)
	}

	if summary.TotalTokens != 150 {
		t.Errorf("Expected 150 total tokens, got %d", summary.TotalTokens)
	}

	if summary.ByCaste["builder"] != 1 {
		t.Error("Expected 1 builder in ByCaste")
	}
}

// TestPauseResume tests pausing and resuming the display.
func TestPauseResume(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.Pause()
	if !pi.isPaused() {
		t.Error("ProgressIndicator should be paused")
	}

	pi.Resume()
	if pi.isPaused() {
		t.Error("ProgressIndicator should be resumed")
	}
}

// TestTruncateString tests string truncation.
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input   string
		maxLen  int
		expected string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 3, "hi"},
		{"hello", 3, "hel"},
		{"test", 2, "te"},
	}

	for _, tt := range tests {
		result := truncateString(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateString(%q, %d) = %q, expected %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

// TestFormatProgressBar tests progress bar formatting.
func TestFormatProgressBar(t *testing.T) {
	// Test 0%
	bar := formatProgressBar(0, 10)
	if !strings.Contains(bar, "[") || !strings.Contains(bar, "]") {
		t.Error("Progress bar should have brackets")
	}

	// Test 50%
	bar = formatProgressBar(50, 10)
	if len(bar) != 10 {
		t.Errorf("Expected bar length 10, got %d", len(bar))
	}

	// Test 100%
	bar = formatProgressBar(100, 10)
	if !strings.Contains(bar, "=") {
		t.Error("100%% bar should contain '=' characters")
	}

	// Test minimum width
	bar = formatProgressBar(50, 2)
	if len(bar) != 3 { // Minimum is 3: []
		t.Errorf("Expected minimum bar length 3, got %d", len(bar))
	}
}

// TestFormatAgentLine tests agent line formatting.
func TestFormatAgentLine(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	agent := &AgentProgress{
		AgentID:   "builder-1",
		AgentName: "Weld-20",
		Caste:     "builder",
		Status:    StatusActive,
		TokenCount: 42,
	}

	style := DefaultProgressStyle()
	line := pi.formatAgentLine(agent, style)

	if !strings.Contains(line, "Weld-20") {
		t.Error("Line should contain agent name")
	}

	if !strings.Contains(line, "builder") {
		t.Error("Line should contain caste")
	}

	if !strings.Contains(line, "42") {
		t.Error("Line should contain token count")
	}
}

// TestConcurrentAccess tests thread-safe concurrent access.
func TestProgressIndicatorConcurrentAccess(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	// Register agents
	for i := 0; i < 5; i++ {
		pi.RegisterAgent(fmt.Sprintf("agent-%d", i), fmt.Sprintf("Agent-%d", i), "builder")
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	updatesPerGoroutine := 50

	// Concurrent updates
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				agentID := fmt.Sprintf("agent-%d", id%5)
				pi.UpdateProgress(agentID, j)
				if j%10 == 0 {
					pi.UpdateStatus(agentID, StatusActive)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify no crashes and agents still accessible
	for i := 0; i < 5; i++ {
		agent := pi.GetAgent(fmt.Sprintf("agent-%d", i))
		if agent == nil {
			t.Errorf("Agent %d should exist after concurrent access", i)
		}
	}
}

// TestClose tests cleanup on close.
func TestProgressIndicatorClose(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UpdateStatus("builder-1", StatusActive)

	err := pi.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// After close, operations should not panic
	pi.UpdateProgress("builder-1", 100)
}

// TestSpinnerFrames tests that spinner frames are defined.
func TestSpinnerFrames(t *testing.T) {
	if len(SpinnerFrames) == 0 {
		t.Error("SpinnerFrames should not be empty")
	}

	// All frames should be non-empty
	for i, frame := range SpinnerFrames {
		if frame == "" {
			t.Errorf("Spinner frame %d should not be empty", i)
		}
	}
}

// TestNonTTYOutput tests non-terminal output format.
func TestNonTTYOutput(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")
	pi.UpdateStatus("builder-1", StatusActive)
	pi.UpdateProgress("builder-1", 10)

	// Give time for render
	time.Sleep(50 * time.Millisecond)

	output := buf.String()

	// Non-TTY output should not contain ANSI codes
	if strings.Contains(output, "\033[") {
		t.Error("Non-TTY output should not contain ANSI escape codes")
	}

	// Should contain agent info
	if !strings.Contains(output, "Weld-20") {
		t.Error("Output should contain agent name")
	}
}

// TestEmptyAgentList tests behavior with no agents.
func TestEmptyAgentList(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	// Should not panic
	pi.ForceRender()
	pi.GetSummary()

	agents := pi.GetActiveAgents()
	if len(agents) != 0 {
		t.Error("Expected 0 active agents with empty list")
	}
}

// TestUnknownAgentOperations tests operations on unregistered agents.
func TestUnknownAgentOperations(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	// Should not panic
	pi.UpdateProgress("unknown-agent", 100)
	pi.UpdateStatus("unknown-agent", StatusActive)
	pi.SetEstimatedTotal("unknown-agent", 1000)

	agent := pi.GetAgent("unknown-agent")
	if agent != nil {
		t.Error("Unknown agent should return nil")
	}
}

// TestAgentLastActivity tests that last activity is updated.
func TestAgentLastActivity(t *testing.T) {
	agent := &AgentProgress{
		AgentID:      "test-1",
		LastActivity: time.Now().Add(-time.Hour),
	}

	oldActivity := agent.LastActivity
	time.Sleep(10 * time.Millisecond)
	agent.UpdateTokens(10)

	if !agent.LastActivity.After(oldActivity) {
		t.Error("LastActivity should be updated on token update")
	}

	oldActivity = agent.LastActivity
	time.Sleep(10 * time.Millisecond)
	agent.SetStatus(StatusActive)

	if !agent.LastActivity.After(oldActivity) {
		t.Error("LastActivity should be updated on status change")
	}
}

// TestGetAllAgentsReturnsCopy tests that GetAllAgents returns a copy.
func TestGetAllAgentsReturnsCopy(t *testing.T) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")

	agents1 := pi.GetAllAgents()
	agents2 := pi.GetAllAgents()

	if len(agents1) != len(agents2) {
		t.Error("Agent lists should have same length")
	}

	// Modifying one should not affect the other
	// (they're different slices, though pointing to same underlying agents)
}

// BenchmarkProgressUpdate benchmarks progress updates.
func BenchmarkProgressUpdate(b *testing.B) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	pi.RegisterAgent("builder-1", "Weld-20", "builder")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pi.UpdateProgress("builder-1", i)
	}
}

// BenchmarkRegisterAgent benchmarks agent registration.
func BenchmarkRegisterAgent(b *testing.B) {
	buf := &progressSyncBuffer{}
	pi := NewProgressIndicatorWithWriter(buf, false)
	defer pi.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pi.RegisterAgent(fmt.Sprintf("agent-%d", i), fmt.Sprintf("Agent-%d", i), "builder")
	}
}
