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

// SpinnerFrames contains the animation frames for the spinner.
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ProgressStyle defines the visual style for progress indicators.
type ProgressStyle struct {
	ShowSpinner     bool
	ShowTokenCount  bool
	ShowAgentName   bool
	ShowCaste       bool
	ShowProgressBar bool
	CompactMode     bool
}

// DefaultProgressStyle returns the default progress indicator style.
func DefaultProgressStyle() ProgressStyle {
	return ProgressStyle{
		ShowSpinner:     true,
		ShowTokenCount:  true,
		ShowAgentName:   true,
		ShowCaste:       true,
		ShowProgressBar: false,
		CompactMode:     false,
	}
}

// CompactProgressStyle returns a compact style for limited space.
func CompactProgressStyle() ProgressStyle {
	return ProgressStyle{
		ShowSpinner:     true,
		ShowTokenCount:  true,
		ShowAgentName:   true,
		ShowCaste:       false,
		ShowProgressBar: false,
		CompactMode:     true,
	}
}

// AgentProgress tracks the progress of a single agent.
type AgentProgress struct {
	AgentID       string
	AgentName     string
	Caste         string
	Status        StreamStatus
	TokenCount    int
	EstimatedTotal int // Estimated total tokens (0 = unknown)
	StartedAt     time.Time
	LastActivity  time.Time
	mu            sync.RWMutex
}

// UpdateTokens updates the token count and last activity time.
func (ap *AgentProgress) UpdateTokens(count int) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.TokenCount = count
	ap.LastActivity = time.Now()
}

// GetTokenCount returns the current token count thread-safely.
func (ap *AgentProgress) GetTokenCount() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.TokenCount
}

// SetStatus updates the agent status.
func (ap *AgentProgress) SetStatus(status StreamStatus) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.Status = status
	ap.LastActivity = time.Now()
}

// GetStatus returns the current status thread-safely.
func (ap *AgentProgress) GetStatus() StreamStatus {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	return ap.Status
}

// GetProgressPercent returns the progress as a percentage (0-100).
// Returns -1 if total is unknown.
func (ap *AgentProgress) GetProgressPercent() int {
	ap.mu.RLock()
	defer ap.mu.RUnlock()
	if ap.EstimatedTotal <= 0 {
		return -1
	}
	percent := (ap.TokenCount * 100) / ap.EstimatedTotal
	if percent > 100 {
		return 100
	}
	return percent
}

// IsActive returns true if the agent is currently active.
func (ap *AgentProgress) IsActive() bool {
	status := ap.GetStatus()
	return status == StatusActive || status == StatusPending
}

// ProgressIndicator manages the display of progress for multiple agents.
type ProgressIndicator struct {
	output      io.Writer
	agents      map[string]*AgentProgress
	mu          sync.RWMutex
	isTTY       bool
	style       ProgressStyle
	spinnerIdx  int
	spinnerMu   sync.Mutex
	ticker      *time.Ticker
	done        chan struct{}
	paused      bool
	pausedMu    sync.RWMutex
}

// NewProgressIndicator creates a new ProgressIndicator writing to stdout.
func NewProgressIndicator() *ProgressIndicator {
	return NewProgressIndicatorWithWriter(os.Stdout, isTerminal(os.Stdout))
}

// NewProgressIndicatorWithWriter creates a ProgressIndicator with custom writer.
func NewProgressIndicatorWithWriter(output io.Writer, isTTY bool) *ProgressIndicator {
	pi := &ProgressIndicator{
		output: output,
		agents: make(map[string]*AgentProgress),
		isTTY:  isTTY,
		style:  DefaultProgressStyle(),
		done:   make(chan struct{}),
	}

	// Start spinner animation
	if isTTY {
		pi.startSpinner()
	}

	return pi
}

// SetStyle updates the display style.
func (pi *ProgressIndicator) SetStyle(style ProgressStyle) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	pi.style = style
}

// RegisterAgent registers a new agent for progress tracking.
func (pi *ProgressIndicator) RegisterAgent(agentID, agentName, caste string) *AgentProgress {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if existing, ok := pi.agents[agentID]; ok {
		return existing
	}

	now := time.Now()
	agent := &AgentProgress{
		AgentID:      agentID,
		AgentName:    agentName,
		Caste:        caste,
		Status:       StatusPending,
		StartedAt:    now,
		LastActivity: now,
	}

	pi.agents[agentID] = agent
	return agent
}

// UnregisterAgent removes an agent from tracking.
func (pi *ProgressIndicator) UnregisterAgent(agentID string) {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	delete(pi.agents, agentID)
}

// GetAgent retrieves an agent's progress by ID.
func (pi *ProgressIndicator) GetAgent(agentID string) *AgentProgress {
	pi.mu.RLock()
	defer pi.mu.RUnlock()
	return pi.agents[agentID]
}

// GetAllAgents returns all registered agents.
func (pi *ProgressIndicator) GetAllAgents() []*AgentProgress {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	result := make([]*AgentProgress, 0, len(pi.agents))
	for _, agent := range pi.agents {
		result = append(result, agent)
	}
	return result
}

// GetActiveAgents returns only currently active agents.
func (pi *ProgressIndicator) GetActiveAgents() []*AgentProgress {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	result := make([]*AgentProgress, 0)
	for _, agent := range pi.agents {
		if agent.IsActive() {
			result = append(result, agent)
		}
	}
	return result
}

// UpdateProgress updates an agent's token count.
func (pi *ProgressIndicator) UpdateProgress(agentID string, tokenCount int) {
	pi.mu.RLock()
	agent, ok := pi.agents[agentID]
	pi.mu.RUnlock()

	if !ok {
		return
	}

	agent.UpdateTokens(tokenCount)
	pi.render()
}

// UpdateStatus updates an agent's status.
func (pi *ProgressIndicator) UpdateStatus(agentID string, status StreamStatus) {
	pi.mu.RLock()
	agent, ok := pi.agents[agentID]
	pi.mu.RUnlock()

	if !ok {
		return
	}

	agent.SetStatus(status)
	pi.render()
}

// SetEstimatedTotal sets the estimated total tokens for an agent.
func (pi *ProgressIndicator) SetEstimatedTotal(agentID string, total int) {
	pi.mu.Lock()
	defer pi.mu.Unlock()

	if agent, ok := pi.agents[agentID]; ok {
		agent.EstimatedTotal = total
	}
}

// startSpinner starts the spinner animation loop.
func (pi *ProgressIndicator) startSpinner() {
	pi.ticker = time.NewTicker(100 * time.Millisecond)
	go pi.spinnerLoop()
}

// spinnerLoop runs the spinner animation.
func (pi *ProgressIndicator) spinnerLoop() {
	for {
		select {
		case <-pi.ticker.C:
			pi.spinnerMu.Lock()
			pi.spinnerIdx = (pi.spinnerIdx + 1) % len(SpinnerFrames)
			pi.spinnerMu.Unlock()
			if !pi.isPaused() {
				pi.render()
			}
		case <-pi.done:
			return
		}
	}
}

// getCurrentSpinnerFrame returns the current spinner frame.
func (pi *ProgressIndicator) getCurrentSpinnerFrame() string {
	pi.spinnerMu.Lock()
	defer pi.spinnerMu.Unlock()
	return SpinnerFrames[pi.spinnerIdx]
}

// Pause temporarily pauses the progress display.
func (pi *ProgressIndicator) Pause() {
	pi.pausedMu.Lock()
	defer pi.pausedMu.Unlock()
	pi.paused = true
}

// Resume resumes the progress display.
func (pi *ProgressIndicator) Resume() {
	pi.pausedMu.Lock()
	defer pi.pausedMu.Unlock()
	pi.paused = false
}

// isPaused returns true if the display is paused.
func (pi *ProgressIndicator) isPaused() bool {
	pi.pausedMu.RLock()
	defer pi.pausedMu.RUnlock()
	return pi.paused
}

// render draws the current progress state.
func (pi *ProgressIndicator) render() {
	if pi.isPaused() {
		return
	}

	agents := pi.GetActiveAgents()
	if len(agents) == 0 {
		return
	}

	if pi.isTTY {
		pi.renderTTY(agents)
	} else {
		pi.renderNonTTY(agents)
	}
}

// renderTTY renders progress for terminal display.
func (pi *ProgressIndicator) renderTTY(agents []*AgentProgress) {
	pi.mu.RLock()
	style := pi.style
	pi.mu.RUnlock()

	var output strings.Builder

	// Clear previous lines
	for i := 0; i < len(agents); i++ {
		output.WriteString("\033[2K\r")
		if i < len(agents)-1 {
			output.WriteString("\033[A")
		}
	}

	// Render each active agent
	for i, agent := range agents {
		line := pi.formatAgentLine(agent, style)
		output.WriteString(line)
		if i < len(agents)-1 {
			output.WriteString("\n")
		}
	}

	// Move cursor back to start
	for i := 0; i < len(agents)-1; i++ {
		output.WriteString("\033[A")
	}
	output.WriteString("\r")

	fmt.Fprint(pi.output, output.String())
}

// renderNonTTY renders progress for non-terminal environments.
func (pi *ProgressIndicator) renderNonTTY(agents []*AgentProgress) {
	pi.mu.RLock()
	style := pi.style
	pi.mu.RUnlock()

	var output strings.Builder

	for _, agent := range agents {
		line := pi.formatAgentLine(agent, style)
		output.WriteString(line)
		output.WriteString("\n")
	}

	fmt.Fprint(pi.output, output.String())
}

// formatAgentLine formats a single agent's progress line.
func (pi *ProgressIndicator) formatAgentLine(agent *AgentProgress, style ProgressStyle) string {
	var parts []string

	// Spinner
	if style.ShowSpinner && agent.IsActive() {
		if agent.GetStatus() == StatusActive {
			parts = append(parts, pi.getCurrentSpinnerFrame())
		} else {
			parts = append(parts, "⏳")
		}
	}

	// Caste
	if style.ShowCaste {
		parts = append(parts, fmt.Sprintf("[%s]", agent.Caste))
	}

	// Agent name
	if style.ShowAgentName {
		if style.CompactMode {
			parts = append(parts, truncateString(agent.AgentName, 12))
		} else {
			parts = append(parts, agent.AgentName)
		}
	}

	// Token count
	if style.ShowTokenCount {
		tokenStr := fmt.Sprintf("%d tokens", agent.GetTokenCount())

		// Add progress percentage if estimated total is known
		percent := agent.GetProgressPercent()
		if percent >= 0 {
			tokenStr = fmt.Sprintf("%s (%d%%)", tokenStr, percent)
		}

		parts = append(parts, tokenStr)
	}

	// Progress bar
	if style.ShowProgressBar {
		percent := agent.GetProgressPercent()
		if percent >= 0 {
			parts = append(parts, formatProgressBar(percent, 20))
		}
	}

	return strings.Join(parts, " ")
}

// truncateString truncates a string to max length with ellipsis.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// formatProgressBar creates a visual progress bar.
func formatProgressBar(percent, width int) string {
	if width < 3 {
		width = 3
	}

	filled := (percent * (width - 2)) / 100
	if filled < 0 {
		filled = 0
	}
	if filled > width-2 {
		filled = width - 2
	}

	var bar strings.Builder
	bar.WriteString("[")
	for i := 0; i < width-2; i++ {
		if i < filled {
			bar.WriteString("=")
		} else if i == filled {
			bar.WriteString(">")
		} else {
			bar.WriteString(" ")
		}
	}
	bar.WriteString("]")

	return bar.String()
}

// ProgressSummary provides aggregate statistics about all agents.
type ProgressSummary struct {
	TotalAgents    int
	ActiveAgents   int
	PendingAgents  int
	CompletedAgents int
	FailedAgents   int
	TotalTokens    int
	ByCaste        map[string]int
}

// GetSummary returns a summary of all agent progress.
func (pi *ProgressIndicator) GetSummary() ProgressSummary {
	pi.mu.RLock()
	defer pi.mu.RUnlock()

	summary := ProgressSummary{
		TotalAgents: len(pi.agents),
		ByCaste:     make(map[string]int),
	}

	for _, agent := range pi.agents {
		summary.ByCaste[agent.Caste]++
		summary.TotalTokens += agent.TokenCount

		switch agent.GetStatus() {
		case StatusActive:
			summary.ActiveAgents++
		case StatusPending:
			summary.PendingAgents++
		case StatusCompleted:
			summary.CompletedAgents++
		case StatusFailed:
			summary.FailedAgents++
		}
	}

	return summary
}

// Close cleans up the ProgressIndicator.
func (pi *ProgressIndicator) Close() error {
	if pi.ticker != nil {
		pi.ticker.Stop()
	}
	close(pi.done)
	return nil
}

// ForceRender forces an immediate display update.
func (pi *ProgressIndicator) ForceRender() {
	pi.render()
}
