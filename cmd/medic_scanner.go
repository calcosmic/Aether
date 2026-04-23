package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ScannerResult holds the output of a full colony health scan.
type ScannerResult struct {
	Healthy      bool
	Issues       []HealthIssue
	FilesChecked int
	FilesHealthy int
	Duration     time.Duration
}

// fileChecker wraps file loading with error-to-HealthIssue conversion.
type fileChecker struct {
	basePath     string // .aether/data/ directory
	repoRoot     string // repository root (derived from basePath)
	filesChecked int
	filesHealthy int
	issues       []HealthIssue
}

// newFileChecker creates a fileChecker rooted at the .aether/data/ path.
// repoRoot is derived as two levels up from basePath.
func newFileChecker(basePath string) *fileChecker {
	repoRoot := filepath.Dir(filepath.Dir(basePath)) // .aether/data/ -> repo root
	return &fileChecker{basePath: basePath, repoRoot: repoRoot}
}

// checkJSONFile attempts to load and parse a JSON file.
// Returns nil, false if the file is missing or corrupted (issues are recorded).
// Returns the raw bytes, true if the file was loaded successfully.
func (fc *fileChecker) checkJSONFile(filename, description string) ([]byte, bool) {
	fc.filesChecked++
	fullPath := filepath.Join(fc.basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			fc.issues = append(fc.issues, issueInfo("file", filename,
				fmt.Sprintf("%s not found", description)))
			return nil, false
		}
		fc.issues = append(fc.issues, issueCritical("file", filename,
			fmt.Sprintf("Failed to read %s: %v", description, err)))
		return nil, false
	}

	if !json.Valid(data) {
		fc.issues = append(fc.issues, fixableIssue(issueCritical("file", filename,
			fmt.Sprintf("%s is corrupted: invalid JSON", description))))
		return nil, false
	}

	fc.filesHealthy++
	return data, true
}

// checkJSONLFile attempts to read a JSONL file and count valid/malformed lines.
// Returns the raw bytes, total lines, and malformed line count.
func (fc *fileChecker) checkJSONLFile(filename, description string) ([]byte, int, int) {
	fc.filesChecked++
	fullPath := filepath.Join(fc.basePath, filename)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			fc.issues = append(fc.issues, issueInfo("file", filename,
				fmt.Sprintf("%s not found", description)))
			return nil, 0, 0
		}
		fc.issues = append(fc.issues, issueCritical("file", filename,
			fmt.Sprintf("Failed to read %s: %v", description, err)))
		return nil, 0, 0
	}

	lines := strings.Split(string(data), "\n")
	total := 0
	malformed := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		total++
		if !json.Valid([]byte(trimmed)) {
			malformed++
		}
	}

	if total > 0 {
		fc.filesHealthy++
	}
	return data, total, malformed
}

// allIssues returns all accumulated issues.
func (fc *fileChecker) allIssues() []HealthIssue {
	return fc.issues
}

// issueCritical creates a critical HealthIssue.
func issueCritical(category, file, message string) HealthIssue {
	return HealthIssue{
		Severity: "critical",
		Category: category,
		Message:  message,
		File:     file,
		Fixable:  false,
	}
}

// issueWarning creates a warning HealthIssue.
func issueWarning(category, file, message string) HealthIssue {
	return HealthIssue{
		Severity: "warning",
		Category: category,
		Message:  message,
		File:     file,
		Fixable:  false,
	}
}

// issueInfo creates an info HealthIssue.
func issueInfo(category, file, message string) HealthIssue {
	return HealthIssue{
		Severity: "info",
		Category: category,
		Message:  message,
		File:     file,
		Fixable:  false,
	}
}

// fixableIssue marks an issue as fixable.
func fixableIssue(issue HealthIssue) HealthIssue {
	issue.Fixable = true
	return issue
}

// performHealthScan runs all health scanners against the colony data directory.
func performHealthScan(opts MedicOptions) (*ScannerResult, error) {
	start := time.Now()

	dataDir := filepath.Join(resolveAetherRoot(), ".aether", "data")
	fc := newFileChecker(dataDir)

	var allIssues []HealthIssue

	allIssues = append(allIssues, scanColonyState(fc)...)
	allIssues = append(allIssues, scanSession(fc)...)
	allIssues = append(allIssues, scanPheromones(fc)...)
	allIssues = append(allIssues, scanDataFiles(fc)...)
	allIssues = append(allIssues, scanJSONL(fc)...)

	// Deep scan: wrapper parity, hub publish integrity, ceremony, and release integrity checks.
	if opts.Deep {
		allIssues = append(allIssues, scanWrapperParity(fc)...)
		allIssues = append(allIssues, scanHubPublishIntegrity()...)
		allIssues = append(allIssues, scanCeremonyIntegrity(fc)...)
		allIssues = append(allIssues, scanIntegrity()...)
	}

	// Merge fileChecker issues (file-level issues from checkJSONFile/checkJSONLFile)
	allIssues = append(allIssues, fc.allIssues()...)

	result := &ScannerResult{
		Issues:       allIssues,
		FilesChecked: fc.filesChecked,
		FilesHealthy: fc.filesHealthy,
		Duration:     time.Since(start),
	}

	// Healthy if no critical issues
	for _, issue := range allIssues {
		if issue.Severity == "critical" {
			result.Healthy = false
			return result, nil
		}
	}
	result.Healthy = true
	return result, nil
}

// resolveAetherRoot returns the Aether root directory for the scanner.
func resolveAetherRoot() string {
	if root := os.Getenv("AETHER_ROOT"); root != "" {
		return root
	}
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

// ---------------------------------------------------------------------------
// ColonyState scanner
// ---------------------------------------------------------------------------

// scanColonyState validates COLONY_STATE.json.
func scanColonyState(fc *fileChecker) []HealthIssue {
	const filename = "COLONY_STATE.json"
	var issues []HealthIssue

	data, ok := fc.checkJSONFile(filename, "COLONY_STATE.json")
	if !ok {
		// If file not found, it's info-level. If corrupted, already critical.
		// Return any issues from fileChecker.
		return nil
	}

	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		issues = append(issues, issueCritical("state", filename,
			fmt.Sprintf("COLONY_STATE.json is corrupted: %v", err)))
		return issues
	}

	// Validate required fields
	if state.Version != "3.0" {
		issues = append(issues, issueWarning("state", filename,
			fmt.Sprintf("State version is '%s', expected '3.0'", state.Version)))
	}

	if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
		issues = append(issues, issueCritical("state", filename,
			"Colony goal is missing"))
	}

	validStates := map[colony.State]bool{
		colony.StateIDLE: true, colony.StateREADY: true,
		colony.StateEXECUTING: true, colony.StateBUILT: true,
		colony.StateCOMPLETED: true,
	}
	if !validStates[state.State] {
		issues = append(issues, issueCritical("state", filename,
			fmt.Sprintf("Invalid state '%s'", state.State)))
	}

	if !state.Scope.Valid() && state.Scope != "" {
		issues = append(issues, issueWarning("state", filename,
			fmt.Sprintf("Invalid scope '%s'", state.Scope)))
	}

	// Validate state consistency
	if state.State == colony.StateEXECUTING && state.CurrentPhase == 0 {
		issues = append(issues, fixableIssue(issueWarning("state", filename,
			"State is EXECUTING but no current phase")))
	}
	if state.State == colony.StateIDLE && state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		issues = append(issues, issueInfo("state", filename,
			"Colony is IDLE but has a goal set"))
	}
	if state.Paused && state.State != colony.StateREADY {
		issues = append(issues, issueWarning("state", filename,
			fmt.Sprintf("Paused flag set but state is %s, not READY", state.State)))
	}

	// Validate parallel mode
	if state.ParallelMode != "" && !state.ParallelMode.Valid() {
		issues = append(issues, issueWarning("state", filename,
			fmt.Sprintf("Invalid parallel_mode '%s'", state.ParallelMode)))
	}

	// Validate worktrees
	orphaned := 0
	validStatuses := map[colony.WorktreeStatus]bool{
		colony.WorktreeAllocated: true, colony.WorktreeInProgress: true,
		colony.WorktreeMerged: true, colony.WorktreeOrphaned: true,
	}
	for _, wt := range state.Worktrees {
		if !validStatuses[wt.Status] && wt.Status != "" {
			issues = append(issues, issueWarning("state", filename,
				fmt.Sprintf("Invalid worktree status '%s' for %s", wt.Status, wt.ID)))
		}
		if wt.Status == colony.WorktreeOrphaned {
			orphaned++
		}
	}
	if orphaned > 0 {
		issues = append(issues, fixableIssue(issueWarning("state", filename,
			fmt.Sprintf("%d orphaned worktree entries", orphaned))))
	}

	// Check deprecated signals field
	if len(state.Signals) > 0 {
		issues = append(issues, fixableIssue(issueWarning("state", filename,
			fmt.Sprintf("Deprecated 'signals' field has %d entries -- should be migrated to pheromones.json", len(state.Signals)))))
	}

	// Validate plan structure
	for i, phase := range state.Plan.Phases {
		validPhaseStatus := map[string]bool{
			colony.PhasePending: true, colony.PhaseReady: true,
			colony.PhaseInProgress: true, colony.PhaseCompleted: true,
		}
		if !validPhaseStatus[phase.Status] && phase.Status != "" {
			issues = append(issues, issueWarning("state", filename,
				fmt.Sprintf("Invalid phase status '%s' at index %d", phase.Status, i)))
		}
		for j, task := range phase.Tasks {
			validTaskStatus := map[string]bool{
				colony.TaskPending: true, colony.TaskCompleted: true,
				colony.TaskInProgress: true,
			}
			if !validTaskStatus[task.Status] && task.Status != "" {
				issues = append(issues, issueWarning("state", filename,
					fmt.Sprintf("Invalid task status '%s' in phase %d, task %d", task.Status, i, j)))
			}
		}
	}

	// Validate events format
	for _, entry := range state.Events {
		segments := strings.Split(entry, "|")
		if len(segments) < 2 {
			issues = append(issues, issueWarning("state", filename,
				fmt.Sprintf("Event entry malformed: %s", entry)))
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// Session scanner
// ---------------------------------------------------------------------------

// scanSession validates session.json and cross-references with COLONY_STATE.json.
func scanSession(fc *fileChecker) []HealthIssue {
	const filename = "session.json"
	var issues []HealthIssue

	data, ok := fc.checkJSONFile(filename, "session.json")
	if !ok {
		return nil
	}

	var session colony.SessionFile
	if err := json.Unmarshal(data, &session); err != nil {
		issues = append(issues, issueCritical("session", filename,
			fmt.Sprintf("session.json is corrupted: %v", err)))
		return issues
	}

	// Validate required fields
	if session.SessionID == "" {
		issues = append(issues, issueWarning("session", filename, "Session ID missing"))
	}

	if session.StartedAt != "" {
		if _, err := time.Parse(time.RFC3339, session.StartedAt); err != nil {
			// Try other common formats
			if _, err2 := time.Parse("2006-01-02T15:04:05Z", session.StartedAt); err2 != nil {
				if _, err3 := time.Parse("2006-01-02T15:04:05-07:00", session.StartedAt); err3 != nil {
					issues = append(issues, issueWarning("session", filename,
						"Invalid started_at timestamp"))
				}
			}
		}
	}

	if session.ColonyGoal == "" {
		issues = append(issues, issueWarning("session", filename,
			"Session has no colony_goal"))
	}

	// Cross-reference with COLONY_STATE.json
	stateData, stateOk := fc.checkJSONFile("COLONY_STATE.json", "COLONY_STATE.json")
	if stateOk {
		var state colony.ColonyState
		if err := json.Unmarshal(stateData, &state); err == nil {
			if session.CurrentPhase != state.CurrentPhase {
				issues = append(issues, fixableIssue(issueWarning("session", filename,
					fmt.Sprintf("session.json current_phase (%d) doesn't match COLONY_STATE (%d)",
						session.CurrentPhase, state.CurrentPhase))))
			}
			if session.ColonyGoal != "" && state.Goal != nil && session.ColonyGoal != *state.Goal {
				issues = append(issues, fixableIssue(issueWarning("session", filename,
					"session.json goal doesn't match COLONY_STATE goal")))
			}
		}
	}

	// Staleness check
	if session.LastCommandAt != "" {
		lastActivity := parseTimestamp(session.LastCommandAt)
		if !lastActivity.IsZero() {
			daysSince := time.Since(lastActivity).Hours() / 24
			if daysSince > 30 {
				issues = append(issues, issueCritical("session", filename,
					fmt.Sprintf("Session critically stale -- last activity %.0f days ago", daysSince)))
			} else if daysSince > 7 {
				issues = append(issues, issueWarning("session", filename,
					fmt.Sprintf("Session is stale -- last activity %.0f days ago", daysSince)))
			}
		}
	}

	return issues
}

// parseTimestamp tries common timestamp formats.
func parseTimestamp(s string) time.Time {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ---------------------------------------------------------------------------
// Pheromone scanner
// ---------------------------------------------------------------------------

// scanPheromones validates pheromones.json.
func scanPheromones(fc *fileChecker) []HealthIssue {
	const filename = "pheromones.json"
	var issues []HealthIssue

	data, ok := fc.checkJSONFile(filename, "pheromones.json")
	if !ok {
		return nil
	}

	var pheromones colony.PheromoneFile
	if err := json.Unmarshal(data, &pheromones); err != nil {
		issues = append(issues, issueCritical("pheromone", filename,
			fmt.Sprintf("pheromones.json is corrupted: %v", err)))
		return issues
	}

	validTypes := map[string]bool{
		"FOCUS": true, "REDIRECT": true, "FEEDBACK": true,
	}
	now := time.Now()

	// Track content hashes for duplicate detection
	seenHashes := make(map[string]string) // hash -> first ID

	for i, sig := range pheromones.Signals {
		if sig.ID == "" {
			issues = append(issues, fixableIssue(issueWarning("pheromone", filename,
				fmt.Sprintf("Signal missing ID at index %d", i))))
		}
		if !validTypes[sig.Type] && sig.Type != "" {
			issues = append(issues, fixableIssue(issueWarning("pheromone", filename,
				fmt.Sprintf("Invalid signal type '%s' at index %d", sig.Type, i))))
		}
		if sig.Content != nil && len(sig.Content) > 0 && !json.Valid(sig.Content) {
			issues = append(issues, issueCritical("pheromone", filename,
				fmt.Sprintf("Signal at index %d has invalid content JSON", i)))
		}
		if sig.CreatedAt != "" {
			if parseTimestamp(sig.CreatedAt).IsZero() {
				issues = append(issues, issueWarning("pheromone", filename,
					fmt.Sprintf("Signal at index %d has invalid created_at timestamp", i)))
			}
		}

		// Check expired-but-active
		if sig.ExpiresAt != nil && *sig.ExpiresAt != "" && sig.Active {
			expiresAt := parseTimestamp(*sig.ExpiresAt)
			if !expiresAt.IsZero() && now.After(expiresAt) {
				issues = append(issues, fixableIssue(issueWarning("pheromone", filename,
					fmt.Sprintf("Signal '%s' has expired but is still active", sig.ID))))
			}
		}

		// Check duplicate content hashes
		if sig.ContentHash != nil && *sig.ContentHash != "" {
			if firstID, exists := seenHashes[*sig.ContentHash]; exists {
				issues = append(issues, issueWarning("pheromone", filename,
					fmt.Sprintf("Duplicate signal content detected (IDs: %s, %s)", firstID, sig.ID)))
			} else {
				seenHashes[*sig.ContentHash] = sig.ID
			}
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// Data file scanner
// ---------------------------------------------------------------------------

// scanDataFiles validates all remaining .aether/data/ JSON files.
func scanDataFiles(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	// Structured data files (have Go structs)
	structuredFiles := []struct {
		filename    string
		description string
	}{
		{"midden/midden.json", "midden.json"},
		{"instincts.json", "instincts.json"},
		{"learning-observations.json", "learning-observations.json"},
		{"assumptions.json", "assumptions.json"},
		{"pending-decisions.json", "pending-decisions.json"},
	}

	for _, sf := range structuredFiles {
		data, ok := fc.checkJSONFile(sf.filename, sf.description)
		if !ok {
			continue
		}
		// Check if file is empty ({})
		trimmed := strings.TrimSpace(string(data))
		if trimmed == "{}" || trimmed == "[]" || trimmed == "null" {
			issues = append(issues, issueInfo("data", sf.filename,
				fmt.Sprintf("%s is empty (expected for new colonies)", sf.description)))
		}
	}

	// Unstructured data files (raw JSON, no Go struct)
	unstructuredFiles := []string{
		"workers.json",
		"spawn-runs.json",
		"last-build-result.json",
		"colony-registry.json",
		"instinct-graph.json",
		"queen-wisdom.json",
		"cost-ledger.json",
	}

	for _, uf := range unstructuredFiles {
		_, ok := fc.checkJSONFile(uf, uf)
		if !ok {
			continue
		}
	}

	// constraints.json: ghost file check
	constraintsPath := filepath.Join(fc.basePath, "constraints.json")
	constraintsData, err := os.ReadFile(constraintsPath)
	if err == nil {
		trimmed := strings.TrimSpace(string(constraintsData))
		if trimmed != "{}" && trimmed != "" {
			issues = append(issues, fixableIssue(issueWarning("data", "constraints.json",
				"constraints.json has content but Go code ignores it (ghost file)")))
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// JSONL scanner
// ---------------------------------------------------------------------------

// scanJSONL validates trace.jsonl, event-bus.jsonl, and spawn-tree.txt.
func scanJSONL(fc *fileChecker) []HealthIssue {
	var issues []HealthIssue

	// trace.jsonl
	_, total, malformed := fc.checkJSONLFile("trace.jsonl", "trace.jsonl")
	if total > 0 && malformed > 0 {
		issues = append(issues, issueWarning("jsonl", "trace.jsonl",
			fmt.Sprintf("trace.jsonl has %d malformed lines out of %d", malformed, total)))
	}
	// Check file size approaching rotation limit (50MB)
	tracePath := filepath.Join(fc.basePath, "trace.jsonl")
	if info, err := os.Stat(tracePath); err == nil {
		sizeMB := float64(info.Size()) / (1024 * 1024)
		if sizeMB > 45 {
			issues = append(issues, issueInfo("jsonl", "trace.jsonl",
				fmt.Sprintf("trace.jsonl is approaching rotation limit (%.1fMB)", sizeMB)))
		}
	}

	// event-bus.jsonl
	_, evtTotal, evtMalformed := fc.checkJSONLFile("event-bus.jsonl", "event-bus.jsonl")
	if evtTotal > 0 && evtMalformed > 0 {
		issues = append(issues, issueWarning("jsonl", "event-bus.jsonl",
			fmt.Sprintf("event-bus.jsonl has %d malformed lines out of %d", evtMalformed, evtTotal)))
	}

	// Check for expired events in event-bus.jsonl
	eventBusPath := filepath.Join(fc.basePath, "event-bus.jsonl")
	if eventData, err := os.ReadFile(eventBusPath); err == nil {
		now := time.Now()
		expiredCount := 0
		for _, line := range strings.Split(string(eventData), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || !json.Valid([]byte(trimmed)) {
				continue
			}
			var evt struct {
				ExpiresAt string `json:"expires_at"`
			}
			if json.Unmarshal([]byte(trimmed), &evt) == nil && evt.ExpiresAt != "" {
				if exp := parseTimestamp(evt.ExpiresAt); !exp.IsZero() && now.After(exp) {
					expiredCount++
				}
			}
		}
		if expiredCount > 0 {
			issues = append(issues, issueWarning("jsonl", "event-bus.jsonl",
				fmt.Sprintf("event-bus.jsonl has %d expired events", expiredCount)))
		}
	}

	// spawn-tree.txt
	spawnTreePath := filepath.Join(fc.basePath, "spawn-tree.txt")
	if _, err := os.Stat(spawnTreePath); err == nil {
		fc.filesChecked++
		content, err := os.ReadFile(spawnTreePath)
		if err != nil {
			issues = append(issues, issueWarning("jsonl", "spawn-tree.txt",
				"Failed to read spawn-tree.txt"))
		} else {
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) > 0 {
				fc.filesHealthy++
			}
		}
	}

	return issues
}

// ---------------------------------------------------------------------------
// Release integrity scanner
// ---------------------------------------------------------------------------

// scanIntegrity checks the version chain (binary vs hub agreement) and stale
// publish detection. Returns HealthIssue entries for medic deep scan.
// Does NOT duplicate companion-file counting — that is scanHubPublishIntegrity's job.
func scanIntegrity() []HealthIssue {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []HealthIssue{{
			Severity: "critical",
			Category: "integrity",
			Message:  "Cannot determine home directory for integrity scan",
			Fixable:  false,
		}}
	}

	channel := resolveRuntimeChannel()
	hubDir := resolveHubPathForHome(homeDir, channel)
	hubVersion := readHubVersionAtPath(hubDir)

	if hubVersion == "" {
		return []HealthIssue{{
			Severity: "critical",
			Category: "integrity",
			Message:  fmt.Sprintf("Hub not installed at %s. Run aether install to populate the hub.", hubDir),
			Fixable:  true,
		}}
	}

	binaryVersion := resolveVersion()
	var issues []HealthIssue

	// Run stale publish detection
	result := checkStalePublish(hubDir, hubVersion, binaryVersion, channel, nil)
	switch result.Classification {
	case staleOK:
		// No issue — everything is consistent
	case staleInfo:
		issues = append(issues, HealthIssue{
			Severity: "warning",
			Category: "integrity",
			Message:  result.Message + " Recovery: " + result.RecoveryCommand,
			Fixable:  true,
		})
	case staleWarning, staleCritical:
		issues = append(issues, HealthIssue{
			Severity: "critical",
			Category: "integrity",
			Message:  result.Message + " Recovery: " + result.RecoveryCommand,
			Fixable:  true,
		})
	}

	// Check version agreement
	if binaryVersion != "unknown" && hubVersion != binaryVersion {
		issues = append(issues, HealthIssue{
			Severity: "critical",
			Category: "integrity",
			Message:  fmt.Sprintf("Binary version (%s) does not match hub version (%s). Run aether publish to synchronize.", binaryVersion, hubVersion),
			Fixable:  true,
		})
	}

	return issues
}
