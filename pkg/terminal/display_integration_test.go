package terminal

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestStreamFlag_DefaultStreamFlag(t *testing.T) {
	sf := DefaultStreamFlag()

	// AutoDetect should be true for default
	if !sf.AutoDetect {
		t.Error("DefaultStreamFlag should have AutoDetect=true")
	}

	// Output should be set
	if sf.Output == nil {
		t.Error("DefaultStreamFlag should have Output set")
	}
}

func TestStreamFlag_NewStreamFlag(t *testing.T) {
	// Test enabled
	sf := NewStreamFlag(true)
	if !sf.Enabled {
		t.Error("NewStreamFlag(true) should have Enabled=true")
	}
	if sf.AutoDetect {
		t.Error("NewStreamFlag should have AutoDetect=false")
	}

	// Test disabled
	sf = NewStreamFlag(false)
	if sf.Enabled {
		t.Error("NewStreamFlag(false) should have Enabled=false")
	}
}

func TestStreamFlag_WithOutput(t *testing.T) {
	var buf bytes.Buffer
	sf := NewStreamFlag(true).WithOutput(&buf)

	if sf.Output != &buf {
		t.Error("WithOutput should set the output writer")
	}
}

func TestStreamFlag_IsStreaming(t *testing.T) {
	sf := NewStreamFlag(true)
	if !sf.IsStreaming() {
		t.Error("IsStreaming should return true when enabled")
	}

	sf = NewStreamFlag(false)
	if sf.IsStreaming() {
		t.Error("IsStreaming should return false when disabled")
	}
}

func TestStreamFlag_String(t *testing.T) {
	tests := []struct {
		name     string
		flag     StreamFlag
		expected string
	}{
		{
			name:     "disabled",
			flag:     StreamFlag{Enabled: false},
			expected: "streaming disabled",
		},
		{
			name:     "enabled auto",
			flag:     StreamFlag{Enabled: true, AutoDetect: true},
			expected: "streaming enabled (auto-detected TTY)",
		},
		{
			name:     "enabled manual",
			flag:     StreamFlag{Enabled: true, AutoDetect: false},
			expected: "streaming enabled (manual)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flag.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestBuildDisplayContext_New(t *testing.T) {
	var buf bytes.Buffer

	// Test with streaming enabled
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)

	if ctx == nil {
		t.Fatal("NewBuildDisplayContext should not return nil")
	}

	if !ctx.IsStreaming() {
		t.Error("Context should be streaming when enabled")
	}

	if ctx.tokenDisplay == nil {
		t.Error("tokenDisplay should be set when streaming is enabled")
	}

	if ctx.progress == nil {
		t.Error("progress should be set when streaming is enabled")
	}

	ctx.Close()

	// Test with streaming disabled
	flag = NewStreamFlag(false)
	ctx = NewBuildDisplayContext(flag)

	if ctx == nil {
		t.Fatal("NewBuildDisplayContext should not return nil")
	}

	if ctx.IsStreaming() {
		t.Error("Context should not be streaming when disabled")
	}

	// Displays should be nil when disabled
	if ctx.tokenDisplay != nil {
		t.Error("tokenDisplay should be nil when streaming is disabled")
	}

	if ctx.progress != nil {
		t.Error("progress should be nil when streaming is disabled")
	}

	ctx.Close()
}

func TestBuildDisplayContext_RegisterAgent(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")

	if ctx.GetActiveCount() != 1 {
		t.Errorf("GetActiveCount() = %d, want 1", ctx.GetActiveCount())
	}

	// Register another agent
	ctx.RegisterAgent("agent-2", "Watcher-1", "watcher")

	if ctx.GetActiveCount() != 2 {
		t.Errorf("GetActiveCount() = %d, want 2", ctx.GetActiveCount())
	}
}

func TestBuildDisplayContext_RegisterAgent_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.RegisterAgent("agent-1", "Builder-1", "builder")

	if ctx.GetActiveCount() != 0 {
		t.Error("GetActiveCount should be 0 when streaming is disabled")
	}
}

func TestBuildDisplayContext_CompleteAgent(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.CompleteAgent("agent-1")

	if ctx.GetActiveCount() != 0 {
		t.Errorf("GetActiveCount() = %d, want 0 after completion", ctx.GetActiveCount())
	}
}

func TestBuildDisplayContext_FailAgent(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.FailAgent("agent-1", errors.New("test error"))

	if ctx.GetActiveCount() != 0 {
		t.Errorf("GetActiveCount() = %d, want 0 after failure", ctx.GetActiveCount())
	}
}

func TestBuildDisplayContext_AddToken(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.AddToken("agent-1", "Hello")
	ctx.AddToken("agent-1", " World")

	// Should not panic and should buffer tokens
	summary := ctx.GetStreamSummary()
	if summary.Total != 1 {
		t.Errorf("StreamSummary.Total = %d, want 1", summary.Total)
	}
}

func TestBuildDisplayContext_AddToken_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.AddToken("agent-1", "Hello")
}

func TestBuildDisplayContext_UpdateProgress(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.UpdateProgress("agent-1", 100)

	summary := ctx.GetProgressSummary()
	if summary.TotalAgents != 1 {
		t.Errorf("ProgressSummary.TotalAgents = %d, want 1", summary.TotalAgents)
	}
	if summary.TotalTokens != 100 {
		t.Errorf("ProgressSummary.TotalTokens = %d, want 100", summary.TotalTokens)
	}
}

func TestBuildDisplayContext_UpdateProgress_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.UpdateProgress("agent-1", 100)
}

func TestBuildDisplayContext_PauseResume(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.Pause()
	ctx.Resume()

	// Should not panic
}

func TestBuildDisplayContext_PauseResume_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.Pause()
	ctx.Resume()
}

func TestBuildDisplayContext_Flush(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.AddToken("agent-1", "test")
	ctx.Flush()

	// Output should have been written
	if buf.Len() == 0 {
		t.Error("Flush should write output")
	}
}

func TestBuildDisplayContext_Flush_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.Flush()
}

func TestBuildDisplayContext_GetStreamSummary_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	summary := ctx.GetStreamSummary()
	if summary.Total != 0 {
		t.Error("GetStreamSummary should return empty summary when disabled")
	}
}

func TestBuildDisplayContext_GetProgressSummary_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	summary := ctx.GetProgressSummary()
	if summary.TotalAgents != 0 {
		t.Error("GetProgressSummary should return empty summary when disabled")
	}
}

func TestBuildDisplayContext_NilReceiver(t *testing.T) {
	var ctx *BuildDisplayContext

	// All methods should handle nil receiver gracefully
	ctx.RegisterAgent("agent-1", "Builder-1", "builder")
	ctx.AddToken("agent-1", "test")
	ctx.UpdateProgress("agent-1", 100)
	ctx.CompleteAgent("agent-1")
	ctx.FailAgent("agent-1", errors.New("test"))
	ctx.Pause()
	ctx.Resume()
	ctx.Flush()

	if ctx.GetActiveCount() != 0 {
		t.Error("GetActiveCount should return 0 for nil context")
	}

	if ctx.IsStreaming() {
		t.Error("IsStreaming should return false for nil context")
	}

	summary := ctx.GetStreamSummary()
	if summary.Total != 0 {
		t.Error("GetStreamSummary should return empty for nil context")
	}

	progressSummary := ctx.GetProgressSummary()
	if progressSummary.TotalAgents != 0 {
		t.Error("GetProgressSummary should return empty for nil context")
	}

	err := ctx.Close()
	if err != nil {
		t.Errorf("Close should not error for nil context: %v", err)
	}
}

func TestContinueDisplayContext_New(t *testing.T) {
	var buf bytes.Buffer

	// Test with streaming enabled
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewContinueDisplayContext(flag)

	if ctx == nil {
		t.Fatal("NewContinueDisplayContext should not return nil")
	}

	if !ctx.IsStreaming() {
		t.Error("Context should be streaming when enabled")
	}

	ctx.Close()

	// Test with streaming disabled
	flag = NewStreamFlag(false)
	ctx = NewContinueDisplayContext(flag)

	if ctx == nil {
		t.Fatal("NewContinueDisplayContext should not return nil")
	}

	if ctx.IsStreaming() {
		t.Error("Context should not be streaming when disabled")
	}

	ctx.Close()
}

func TestContinueDisplayContext_RegisterVerification(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewContinueDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterVerification("verify-1", "Build Check")

	summary := ctx.GetSummary()
	if summary.TotalAgents != 1 {
		t.Errorf("GetSummary().TotalAgents = %d, want 1", summary.TotalAgents)
	}
}

func TestContinueDisplayContext_RegisterVerification_NoStreaming(t *testing.T) {
	flag := NewStreamFlag(false)
	ctx := NewContinueDisplayContext(flag)
	defer ctx.Close()

	// Should not panic when streaming is disabled
	ctx.RegisterVerification("verify-1", "Build Check")
}

func TestContinueDisplayContext_CompleteVerification(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewContinueDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterVerification("verify-1", "Build Check")
	ctx.CompleteVerification("verify-1")

	summary := ctx.GetSummary()
	if summary.CompletedAgents != 1 {
		t.Errorf("GetSummary().CompletedAgents = %d, want 1", summary.CompletedAgents)
	}
}

func TestContinueDisplayContext_FailVerification(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewContinueDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterVerification("verify-1", "Build Check")
	ctx.FailVerification("verify-1")

	summary := ctx.GetSummary()
	if summary.FailedAgents != 1 {
		t.Errorf("GetSummary().FailedAgents = %d, want 1", summary.FailedAgents)
	}
}

func TestContinueDisplayContext_NilReceiver(t *testing.T) {
	var ctx *ContinueDisplayContext

	// All methods should handle nil receiver gracefully
	ctx.RegisterVerification("verify-1", "Build Check")
	ctx.CompleteVerification("verify-1")
	ctx.FailVerification("verify-1")

	if ctx.IsStreaming() {
		t.Error("IsStreaming should return false for nil context")
	}

	summary := ctx.GetSummary()
	if summary.TotalAgents != 0 {
		t.Error("GetSummary should return empty for nil context")
	}

	err := ctx.Close()
	if err != nil {
		t.Errorf("Close should not error for nil context: %v", err)
	}
}

func TestBuildDisplayContext_MultipleAgents(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	// Register multiple agents
	agents := []struct {
		id    string
		name  string
		caste string
	}{
		{"agent-1", "Builder-1", "builder"},
		{"agent-2", "Builder-2", "builder"},
		{"agent-3", "Watcher-1", "watcher"},
	}

	for _, a := range agents {
		ctx.RegisterAgent(a.id, a.name, a.caste)
	}

	if ctx.GetActiveCount() != 3 {
		t.Errorf("GetActiveCount() = %d, want 3", ctx.GetActiveCount())
	}

	// Complete some agents
	ctx.CompleteAgent("agent-1")
	ctx.CompleteAgent("agent-2")

	if ctx.GetActiveCount() != 1 {
		t.Errorf("GetActiveCount() = %d, want 1 after completions", ctx.GetActiveCount())
	}

	// Fail the last one
	ctx.FailAgent("agent-3", errors.New("verification failed"))

	if ctx.GetActiveCount() != 0 {
		t.Errorf("GetActiveCount() = %d, want 0 after all done", ctx.GetActiveCount())
	}
}

func TestBuildDisplayContext_TokenAccumulation(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	ctx.RegisterAgent("agent-1", "Builder-1", "builder")

	// Add multiple tokens
	tokens := []string{"Hello", " ", "World", "!"}
	for _, token := range tokens {
		ctx.AddToken("agent-1", token)
	}

	summary := ctx.GetStreamSummary()
	if summary.TotalTokens != len(tokens) {
		t.Errorf("TotalTokens = %d, want %d", summary.TotalTokens, len(tokens))
	}
}

func TestIntegration_WithBufferOutput(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(true).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)

	// Simulate a build with streaming
	ctx.RegisterAgent("builder-1", "Hammer-42", "builder")
	ctx.AddToken("builder-1", "Building")
	ctx.AddToken("builder-1", " phase")
	ctx.AddToken("builder-1", " 1...")
	ctx.Flush()

	// Output should contain the agent name
	output := buf.String()
	if !strings.Contains(output, "Hammer-42") {
		t.Error("Output should contain agent name")
	}

	// Complete the agent
	ctx.CompleteAgent("builder-1")
	ctx.Close()
}

func TestIntegration_DisabledStreaming(t *testing.T) {
	var buf bytes.Buffer
	flag := NewStreamFlag(false).WithOutput(&buf)
	ctx := NewBuildDisplayContext(flag)

	// Simulate a build without streaming
	ctx.RegisterAgent("builder-1", "Hammer-42", "builder")
	ctx.AddToken("builder-1", "Building")
	ctx.UpdateProgress("builder-1", 100)
	ctx.CompleteAgent("builder-1")
	ctx.Close()

	// Output should be empty since streaming is disabled
	if buf.Len() != 0 {
		t.Errorf("Output should be empty when streaming is disabled, got: %q", buf.String())
	}
}

// Benchmark tests
func BenchmarkBuildDisplayContext_RegisterAgent(b *testing.B) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	defer ctx.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.RegisterAgent("agent", "Name", "builder")
	}
}

func BenchmarkBuildDisplayContext_AddToken(b *testing.B) {
	flag := NewStreamFlag(false)
	ctx := NewBuildDisplayContext(flag)
	ctx.RegisterAgent("agent", "Name", "builder")
	defer ctx.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.AddToken("agent", "token")
	}
}
