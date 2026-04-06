// Package terminal provides terminal UI components for displaying streaming content.
package terminal

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// StreamFlag represents the streaming display configuration.
type StreamFlag struct {
	Enabled    bool   // Whether streaming display is enabled
	AutoDetect bool   // Whether TTY was auto-detected
	Output     io.Writer // Output writer (defaults to os.Stdout)
}

// DefaultStreamFlag returns a StreamFlag with auto-detection enabled.
// It automatically enables streaming if stdout is a TTY.
func DefaultStreamFlag() StreamFlag {
	return StreamFlag{
		Enabled:    isTerminalWriter(os.Stdout),
		AutoDetect: true,
		Output:     os.Stdout,
	}
}

// NewStreamFlag creates a StreamFlag with explicit enable/disable setting.
func NewStreamFlag(enabled bool) StreamFlag {
	return StreamFlag{
		Enabled:    enabled,
		AutoDetect: false,
		Output:     os.Stdout,
	}
}

// WithOutput sets a custom output writer.
func (sf StreamFlag) WithOutput(output io.Writer) StreamFlag {
	sf.Output = output
	return sf
}

// IsStreaming returns true if streaming display is active.
func (sf StreamFlag) IsStreaming() bool {
	return sf.Enabled
}

// String returns a human-readable description of the stream flag state.
func (sf StreamFlag) String() string {
	if !sf.Enabled {
		return "streaming disabled"
	}
	if sf.AutoDetect {
		return "streaming enabled (auto-detected TTY)"
	}
	return "streaming enabled (manual)"
}

// BuildDisplayContext manages the streaming display context for build commands.
type BuildDisplayContext struct {
	flag          StreamFlag
	tokenDisplay  *TokenDisplay
	progress      *ProgressIndicator
	mu            sync.RWMutex
	activeStreams map[string]bool
}

// NewBuildDisplayContext creates a new build display context.
// If streaming is not enabled, it returns a context with nil displays
// that can still be called safely (no-op).
func NewBuildDisplayContext(flag StreamFlag) *BuildDisplayContext {
	ctx := &BuildDisplayContext{
		flag:          flag,
		activeStreams: make(map[string]bool),
	}

	if flag.Enabled {
		ctx.tokenDisplay = NewTokenDisplayWithWriter(flag.Output, isTerminalWriter(flag.Output))
		ctx.progress = NewProgressIndicatorWithWriter(flag.Output, isTerminalWriter(flag.Output))
	}

	return ctx
}

// isTerminalWriter checks if a writer is a terminal.
func isTerminalWriter(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		stat, err := f.Stat()
		if err != nil {
			return false
		}
		return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
	}
	return false
}

// RegisterAgent registers an agent for streaming display.
// Safe to call even when streaming is disabled (becomes no-op).
func (bdc *BuildDisplayContext) RegisterAgent(agentID, agentName, caste string) {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	bdc.mu.Lock()
	defer bdc.mu.Unlock()

	bdc.activeStreams[agentID] = true

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.RegisterStream(agentID, caste)
		bdc.tokenDisplay.UpdateStatus(agentID, StatusActive)
	}

	if bdc.progress != nil {
		bdc.progress.RegisterAgent(agentID, agentName, caste)
		bdc.progress.UpdateStatus(agentID, StatusActive)
	}
}

// AddToken adds a token to an agent's stream.
// Safe to call even when streaming is disabled (becomes no-op).
func (bdc *BuildDisplayContext) AddToken(agentID, token string) {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.AddToken(agentID, token)
	}
}

// UpdateProgress updates an agent's progress.
// Safe to call even when streaming is disabled (becomes no-op).
func (bdc *BuildDisplayContext) UpdateProgress(agentID string, tokenCount int) {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	if bdc.progress != nil {
		bdc.progress.UpdateProgress(agentID, tokenCount)
	}
}

// CompleteAgent marks an agent as completed.
// Safe to call even when streaming is disabled (becomes no-op).
func (bdc *BuildDisplayContext) CompleteAgent(agentID string) {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	bdc.mu.Lock()
	defer bdc.mu.Unlock()

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.CompleteStream(agentID)
	}

	if bdc.progress != nil {
		bdc.progress.UpdateStatus(agentID, StatusCompleted)
	}

	delete(bdc.activeStreams, agentID)
}

// FailAgent marks an agent as failed.
// Safe to call even when streaming is disabled (becomes no-op).
func (bdc *BuildDisplayContext) FailAgent(agentID string, err error) {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	bdc.mu.Lock()
	defer bdc.mu.Unlock()

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.FailStream(agentID, err)
	}

	if bdc.progress != nil {
		bdc.progress.UpdateStatus(agentID, StatusFailed)
	}

	delete(bdc.activeStreams, agentID)
}

// GetActiveCount returns the number of active streams.
func (bdc *BuildDisplayContext) GetActiveCount() int {
	if bdc == nil {
		return 0
	}

	bdc.mu.RLock()
	defer bdc.mu.RUnlock()

	return len(bdc.activeStreams)
}

// IsStreaming returns true if streaming is enabled.
func (bdc *BuildDisplayContext) IsStreaming() bool {
	if bdc == nil {
		return false
	}
	return bdc.flag.Enabled
}

// Pause pauses the display updates.
func (bdc *BuildDisplayContext) Pause() {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.Pause()
	}

	if bdc.progress != nil {
		bdc.progress.Pause()
	}
}

// Resume resumes the display updates.
func (bdc *BuildDisplayContext) Resume() {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.Resume()
	}

	if bdc.progress != nil {
		bdc.progress.Resume()
	}
}

// Flush forces an immediate display update.
func (bdc *BuildDisplayContext) Flush() {
	if bdc == nil || !bdc.flag.Enabled {
		return
	}

	if bdc.tokenDisplay != nil {
		bdc.tokenDisplay.Flush()
	}

	if bdc.progress != nil {
		bdc.progress.ForceRender()
	}
}

// Close cleans up resources.
func (bdc *BuildDisplayContext) Close() error {
	if bdc == nil || !bdc.flag.Enabled {
		return nil
	}

	var errs []error

	if bdc.tokenDisplay != nil {
		if err := bdc.tokenDisplay.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if bdc.progress != nil {
		if err := bdc.progress.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing display context: %v", errs)
	}

	return nil
}

// GetStreamSummary returns a summary of all streams.
// Returns empty summary when streaming is disabled.
func (bdc *BuildDisplayContext) GetStreamSummary() StreamSummary {
	if bdc == nil || !bdc.flag.Enabled || bdc.tokenDisplay == nil {
		return StreamSummary{}
	}

	return bdc.tokenDisplay.GetStreamSummary()
}

// GetProgressSummary returns a summary of all agent progress.
// Returns empty summary when streaming is disabled.
func (bdc *BuildDisplayContext) GetProgressSummary() ProgressSummary {
	if bdc == nil || !bdc.flag.Enabled || bdc.progress == nil {
		return ProgressSummary{}
	}

	return bdc.progress.GetSummary()
}

// ContinueDisplayContext manages the streaming display context for continue commands.
// This is a lighter-weight version focused on verification progress.
type ContinueDisplayContext struct {
	flag     StreamFlag
	progress *ProgressIndicator
	mu       sync.RWMutex
}

// NewContinueDisplayContext creates a new continue display context.
func NewContinueDisplayContext(flag StreamFlag) *ContinueDisplayContext {
	ctx := &ContinueDisplayContext{
		flag: flag,
	}

	if flag.Enabled {
		ctx.progress = NewProgressIndicatorWithWriter(flag.Output, isTerminalWriter(flag.Output))
	}

	return ctx
}

// RegisterVerification registers a verification step for display.
func (cdc *ContinueDisplayContext) RegisterVerification(stepID, stepName string) {
	if cdc == nil || !cdc.flag.Enabled || cdc.progress == nil {
		return
	}

	cdc.progress.RegisterAgent(stepID, stepName, "verification")
	cdc.progress.UpdateStatus(stepID, StatusActive)
}

// CompleteVerification marks a verification step as completed.
func (cdc *ContinueDisplayContext) CompleteVerification(stepID string) {
	if cdc == nil || !cdc.flag.Enabled || cdc.progress == nil {
		return
	}

	cdc.progress.UpdateStatus(stepID, StatusCompleted)
}

// FailVerification marks a verification step as failed.
func (cdc *ContinueDisplayContext) FailVerification(stepID string) {
	if cdc == nil || !cdc.flag.Enabled || cdc.progress == nil {
		return
	}

	cdc.progress.UpdateStatus(stepID, StatusFailed)
}

// IsStreaming returns true if streaming is enabled.
func (cdc *ContinueDisplayContext) IsStreaming() bool {
	if cdc == nil {
		return false
	}
	return cdc.flag.Enabled
}

// Close cleans up resources.
func (cdc *ContinueDisplayContext) Close() error {
	if cdc == nil || !cdc.flag.Enabled || cdc.progress == nil {
		return nil
	}

	return cdc.progress.Close()
}

// GetSummary returns progress summary.
func (cdc *ContinueDisplayContext) GetSummary() ProgressSummary {
	if cdc == nil || !cdc.flag.Enabled || cdc.progress == nil {
		return ProgressSummary{}
	}

	return cdc.progress.GetSummary()
}

