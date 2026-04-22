package agent

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// SpawnEntry represents a single agent spawn record, matching the shell
// spawn-tree.txt format with exactly 7 pipe-delimited fields plus an
// optional summary from completion lines.
type SpawnEntry struct {
	Timestamp         string // Field 1: ISO 8601 UTC (2006-01-02T15:04:05Z)
	ActivityTimestamp string // Latest activity timestamp (spawn or completion) for run filtering.
	ParentName        string // Field 2: parent agent name
	Caste             string // Field 3: caste name (builder, watcher, etc.)
	AgentName         string // Field 4: agent name
	Task              string // Field 5: task description
	Depth             int    // Field 6: spawn depth
	Status            string // Field 7: "spawned", "active", "completed", "failed", "blocked"
	Summary           string // Completion summary (set via spawn-complete --summary)
}

// SpawnRun tracks one logical dispatch or workflow run for current-run filtering.
type SpawnRun struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	Status    string `json:"status"`
}

// completionLine represents a status update line in the spawn tree file.
// Format: timestamp|name|status|summary (summary is empty for backward compat)
type completionLine struct {
	Timestamp string
	Name      string
	Status    string
	Summary   string
}

type spawnRunState struct {
	CurrentRunID string     `json:"current_run_id,omitempty"`
	Runs         []SpawnRun `json:"runs,omitempty"`
}

// SpawnTree tracks running agents in the same pipe-delimited format as the
// shell spawn-tree.txt, enabling Go and shell to coexist.
type SpawnTree struct {
	store       *storage.Store
	mu          sync.Mutex
	entries     []SpawnEntry
	completions []completionLine
	filePath    string
}

const (
	defaultSpawnRunFile  = "spawn-runs.json"
	spawnRunHistoryLimit = 12
	spawnRunStatusActive = "active"
	spawnRunStatusDone   = "completed"
	spawnRunStatusFailed = "failed"
	spawnRunStatusStale  = "superseded"
)

// NewSpawnTree creates a spawn tree backed by the given store.
// filePath defaults to "spawn-tree.txt" if empty.
// Existing entries are loaded from the file on creation (graceful: empty if missing).
func NewSpawnTree(store *storage.Store, filePath string) *SpawnTree {
	if filePath == "" {
		filePath = "spawn-tree.txt"
	}
	st := &SpawnTree{
		store:    store,
		filePath: filePath,
	}
	if store == nil {
		return st
	}
	// Load existing entries (graceful: empty if file missing)
	entries, completions, _ := st.parseFile()
	st.entries = entries
	st.completions = completions
	return st
}

// BeginRun records a new logical runtime run and makes it the current run.
// Older active runs are superseded so stale activity stops poisoning later commands.
func (st *SpawnTree) BeginRun(command string, startedAt time.Time) (SpawnRun, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	runState, err := st.loadRunStateLocked()
	if err != nil {
		return SpawnRun{}, err
	}

	now := startedAt.UTC().Format(time.RFC3339)
	for i := range runState.Runs {
		if runState.Runs[i].Status == spawnRunStatusActive {
			runState.Runs[i].Status = spawnRunStatusStale
			if strings.TrimSpace(runState.Runs[i].EndedAt) == "" {
				runState.Runs[i].EndedAt = now
			}
		}
	}

	run := SpawnRun{
		ID:        newSpawnRunID(command, startedAt.UTC()),
		Command:   sanitizeSpawnField(command),
		StartedAt: now,
		Status:    spawnRunStatusActive,
	}
	runState.CurrentRunID = run.ID
	runState.Runs = append(runState.Runs, run)
	runState.Runs = trimSpawnRuns(runState.Runs)

	if err := st.saveRunStateLocked(runState); err != nil {
		return SpawnRun{}, err
	}
	return run, nil
}

// EndRun marks the specified logical runtime run as finished.
func (st *SpawnTree) EndRun(runID, status string, endedAt time.Time) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	runState, err := st.loadRunStateLocked()
	if err != nil {
		return err
	}
	status = normalizeSpawnRunStatus(status)
	if status == "" {
		status = spawnRunStatusDone
	}
	if endedAt.IsZero() {
		endedAt = time.Now().UTC()
	}

	for i := range runState.Runs {
		if runState.Runs[i].ID != runID {
			continue
		}
		runState.Runs[i].Status = status
		runState.Runs[i].EndedAt = endedAt.UTC().Format(time.RFC3339)
		return st.saveRunStateLocked(runState)
	}
	return fmt.Errorf("spawn_tree: run %q not found", runID)
}

// CurrentRun returns the current or most recent tracked run.
func (st *SpawnTree) CurrentRun() (SpawnRun, bool, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	runState, err := st.loadRunStateLocked()
	if err != nil {
		return SpawnRun{}, false, err
	}
	if run, ok := findSpawnRun(runState, runState.CurrentRunID); ok {
		return run, true, nil
	}
	if len(runState.Runs) == 0 {
		return SpawnRun{}, false, nil
	}
	return runState.Runs[len(runState.Runs)-1], true, nil
}

// EntriesForRun returns spawn entries that belong to the given run's time window.
func (st *SpawnTree) EntriesForRun(runID string) ([]SpawnEntry, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	entries, _, err := st.parseFile()
	if err != nil {
		return nil, err
	}
	runState, err := st.loadRunStateLocked()
	if err != nil {
		return nil, err
	}
	run, ok := findSpawnRun(runState, runID)
	if !ok {
		return nil, nil
	}
	return filterEntriesForRun(entries, runState.Runs, run), nil
}

// RecordSpawn creates a new spawn entry and persists it to the file.
func (st *SpawnTree) RecordSpawn(parent, caste, name, task string, depth int) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.store == nil {
		return fmt.Errorf("spawn tree: store is nil")
	}

	if err := st.store.UpdateFile(st.filePath, func(existing []byte) ([]byte, error) {
		entries, completions := parseSpawnTreeBytes(existing)
		entry := SpawnEntry{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			ParentName: sanitizeSpawnField(parent),
			Caste:      sanitizeSpawnField(caste),
			AgentName:  sanitizeSpawnField(name),
			Task:       sanitizeSpawnField(task),
			Depth:      depth,
			Status:     "spawned",
		}
		entries = append(entries, entry)
		st.entries = entries
		st.completions = completions
		return formatSpawnTreeLines(entries, completions), nil
	}); err != nil {
		return err
	}
	return st.reloadLocked()
}

// UpdateStatus finds an entry by agent name and updates its status.
// It adds a completion line to the file matching the shell's second awk rule.
// The summary is an optional description stored alongside the completion.
func (st *SpawnTree) UpdateStatus(name string, status string, summary string) error {
	return st.updateStatus(name, status, summary, false)
}

// UpdateStatusPreserveActivity updates status without re-attributing the entry
// to the current run window. This is useful when a later command reconciles an
// older worker record but should not make that worker appear as fresh activity.
func (st *SpawnTree) UpdateStatusPreserveActivity(name string, status string, summary string) error {
	return st.updateStatus(name, status, summary, true)
}

func (st *SpawnTree) updateStatus(name string, status string, summary string, preserveActivity bool) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	name = sanitizeSpawnField(name)
	status = normalizeSpawnStatus(status)
	summary = sanitizeSpawnField(summary)
	updatedAt := time.Now().UTC().Format(time.RFC3339)
	if st.store == nil {
		return fmt.Errorf("spawn tree: store is nil")
	}

	if err := st.store.UpdateFile(st.filePath, func(existing []byte) ([]byte, error) {
		entries, completions := parseSpawnTreeBytes(existing)

		found := false
		completionTimestamp := updatedAt
		for i := len(entries) - 1; i >= 0; i-- {
			if entries[i].AgentName == name {
				entries[i].Status = status
				entries[i].Summary = summary
				if preserveActivity {
					completionTimestamp = strings.TrimSpace(entries[i].ActivityTimestamp)
					if completionTimestamp == "" {
						completionTimestamp = strings.TrimSpace(entries[i].Timestamp)
					}
				} else {
					entries[i].ActivityTimestamp = updatedAt
				}
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("spawn_tree: agent %q not found", name)
		}
		if strings.TrimSpace(completionTimestamp) == "" {
			completionTimestamp = updatedAt
		}

		completions = append(completions, completionLine{
			Timestamp: completionTimestamp,
			Name:      name,
			Status:    status,
			Summary:   summary,
		})

		st.entries = entries
		st.completions = completions
		return formatSpawnTreeLines(entries, completions), nil
	}); err != nil {
		return err
	}
	return st.reloadLocked()
}

// Persist writes all entries to the file via store.AtomicWrite.
func (st *SpawnTree) Persist() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.persistLocked()
}

// persistLocked writes all entries to file. Caller must hold the mutex.
func (st *SpawnTree) persistLocked() error {
	if st.store == nil {
		return nil
	}
	return st.store.AtomicWrite(st.filePath, formatSpawnTreeLines(st.entries, st.completions))
}

// parseFile reads and parses the spawn tree file.
// Returns spawn entries, completion lines, and any error.
func (st *SpawnTree) parseFile() ([]SpawnEntry, []completionLine, error) {
	if st.store == nil {
		return nil, nil, nil
	}
	data, err := st.store.ReadFile(st.filePath)
	if err != nil {
		// File doesn't exist -- return empty
		return nil, nil, nil
	}
	entries, completions := parseSpawnTreeBytes(data)
	return entries, completions, nil
}

func parseSpawnTreeBytes(data []byte) ([]SpawnEntry, []completionLine) {
	var entries []SpawnEntry
	var completions []completionLine

	// Build index from agent name to entry for status merging
	nameToIdx := make(map[string]int)

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.SplitN(line, "|", 7)
		fieldCount := len(fields)

		if fieldCount == 7 {
			// Spawn entry: timestamp|parent|caste|name|task|depth|status
			depth, err := strconv.Atoi(fields[5])
			if err != nil {
				continue // skip malformed
			}
			entry := SpawnEntry{
				Timestamp:         fields[0],
				ActivityTimestamp: fields[0],
				ParentName:        fields[1],
				Caste:             fields[2],
				AgentName:         fields[3],
				Task:              fields[4],
				Depth:             depth,
				Status:            normalizeSpawnStatus(fields[6]),
			}
			nameToIdx[fields[3]] = len(entries)
			entries = append(entries, entry)
			continue
		}

		completionFields := strings.SplitN(line, "|", 4)
		if len(completionFields) >= 4 {
			// Completion line: timestamp|name|status|summary
			// Old format (no summary): timestamp|name|status|  -> field[3] = ""
			// New format (with summary): timestamp|name|status|summary -> field[3] = summary
			status := normalizeSpawnStatus(completionFields[2])
			if status != "" {
				summary := completionFields[3]
				completions = append(completions, completionLine{
					Timestamp: completionFields[0],
					Name:      completionFields[1],
					Status:    status,
					Summary:   summary,
				})
				// Merge status and summary into the matching spawn entry
				if idx, ok := nameToIdx[completionFields[1]]; ok {
					entries[idx].Status = status
					entries[idx].ActivityTimestamp = completionFields[0]
					if summary != "" {
						entries[idx].Summary = summary
					}
				}
			}
		}
	}

	return entries, completions
}

func formatSpawnTreeLines(entries []SpawnEntry, completions []completionLine) []byte {
	var lines []string

	for _, e := range entries {
		line := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
			e.Timestamp, e.ParentName, e.Caste, e.AgentName, e.Task, e.Depth, e.Status)
		lines = append(lines, line)
	}

	for _, c := range completions {
		line := fmt.Sprintf("%s|%s|%s|%s", c.Timestamp, c.Name, c.Status, c.Summary)
		lines = append(lines, line)
	}

	return []byte(strings.Join(lines, "\n") + "\n")
}

func (st *SpawnTree) reloadLocked() error {
	entries, completions, err := st.parseFile()
	if err != nil {
		return err
	}
	st.entries = entries
	st.completions = completions
	return nil
}

func sanitizeSpawnField(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "¦")
	return strings.TrimSpace(value)
}

func normalizeSpawnStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "timed_out":
		return "timeout"
	case "manual", "manually_reconciled":
		return "manually-reconciled"
	}
	return sanitizeSpawnField(status)
}

func normalizeSpawnRunStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "", "complete", "completed", "done":
		return spawnRunStatusDone
	case "failed", "error", "timeout":
		return spawnRunStatusFailed
	case "superseded", "stale":
		return spawnRunStatusStale
	default:
		return sanitizeSpawnField(status)
	}
}

func newSpawnRunID(command string, startedAt time.Time) string {
	label := sanitizeSpawnField(strings.ToLower(strings.ReplaceAll(command, " ", "-")))
	if label == "" {
		label = "run"
	}
	return fmt.Sprintf("%s-%d", label, startedAt.UnixNano())
}

func trimSpawnRuns(runs []SpawnRun) []SpawnRun {
	if len(runs) <= spawnRunHistoryLimit {
		return runs
	}
	return append([]SpawnRun{}, runs[len(runs)-spawnRunHistoryLimit:]...)
}

func findSpawnRun(runState spawnRunState, runID string) (SpawnRun, bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return SpawnRun{}, false
	}
	for _, run := range runState.Runs {
		if run.ID == runID {
			return run, true
		}
	}
	return SpawnRun{}, false
}

func parseSpawnRunTime(raw string) time.Time {
	ts, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return ts
}

func filterEntriesForRun(entries []SpawnEntry, runs []SpawnRun, run SpawnRun) []SpawnEntry {
	start := parseSpawnRunTime(run.StartedAt)
	if start.IsZero() {
		return entries
	}

	end := parseSpawnRunTime(run.EndedAt)
	for _, candidate := range runs {
		candidateStart := parseSpawnRunTime(candidate.StartedAt)
		if candidateStart.IsZero() || !candidateStart.After(start) {
			continue
		}
		if end.IsZero() || candidateStart.Before(end) {
			end = candidateStart
		}
	}

	filtered := make([]SpawnEntry, 0, len(entries))
	for _, entry := range entries {
		ts := spawnEntryActivityTime(entry)
		if ts.IsZero() {
			continue
		}
		if ts.Before(start) {
			continue
		}
		if !end.IsZero() && ts.After(end) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func spawnEntryActivityTime(entry SpawnEntry) time.Time {
	if ts := parseSpawnRunTime(entry.ActivityTimestamp); !ts.IsZero() {
		return ts
	}
	return parseSpawnRunTime(entry.Timestamp)
}

func (st *SpawnTree) loadRunStateLocked() (spawnRunState, error) {
	if st.store == nil {
		return spawnRunState{}, nil
	}

	data, err := st.store.ReadFile(defaultSpawnRunFile)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		return spawnRunState{}, nil
	}

	var state spawnRunState
	if err := json.Unmarshal(data, &state); err != nil {
		return spawnRunState{}, err
	}
	state.Runs = trimSpawnRuns(state.Runs)
	return state, nil
}

func (st *SpawnTree) saveRunStateLocked(state spawnRunState) error {
	if st.store == nil {
		return nil
	}
	state.Runs = trimSpawnRuns(state.Runs)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return st.store.AtomicWrite(defaultSpawnRunFile, append(data, '\n'))
}

// Parse reads the file and returns all spawn entries with statuses merged
// from completion lines.
func (st *SpawnTree) Parse() ([]SpawnEntry, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	entries, _, err := st.parseFile()
	return entries, err
}

// Active returns entries with Status "spawned" or "active".
func (st *SpawnTree) Active() []SpawnEntry {
	st.mu.Lock()
	defer st.mu.Unlock()

	entries, _, _ := st.parseFile()
	if runState, err := st.loadRunStateLocked(); err == nil {
		if run, ok := findSpawnRun(runState, runState.CurrentRunID); ok {
			entries = filterEntriesForRun(entries, runState.Runs, run)
		}
	}

	var active []SpawnEntry
	for _, e := range entries {
		if IsLiveSpawnStatus(e.Status) {
			active = append(active, e)
		}
	}
	if active == nil {
		active = []SpawnEntry{}
	}
	return active
}

// spawnTreeJSON matches the shell parse_spawn_tree output format.
type spawnTreeJSON struct {
	Spawns   []spawnEntryJSON `json:"spawns"`
	Metadata struct {
		TotalCount     int    `json:"total_count"`
		ActiveCount    int    `json:"active_count"`
		CompletedCount int    `json:"completed_count"`
		CurrentRunID   string `json:"current_run_id,omitempty"`
	} `json:"metadata"`
}

// spawnEntryJSON is a single spawn in the JSON output.
type spawnEntryJSON struct {
	Name        string `json:"name"`
	Parent      string `json:"parent"`
	Caste       string `json:"caste"`
	Task        string `json:"task"`
	Status      string `json:"status"`
	SpawnedAt   string `json:"spawned_at"`
	CompletedAt string `json:"completed_at"`
}

// ToJSON returns JSON matching the shell parse_spawn_tree output format.
func (st *SpawnTree) ToJSON() ([]byte, error) {
	st.mu.Lock()
	defer st.mu.Unlock()

	activeCount := 0
	completedCount := 0

	// Build completion lookup: agent name -> completion timestamp
	completionTimes := make(map[string]string)
	for _, c := range st.completions {
		completionTimes[c.Name] = c.Timestamp
	}

	spawns := make([]spawnEntryJSON, 0, len(st.entries))
	for _, e := range st.entries {
		completedAt := ""
		if ts, ok := completionTimes[e.AgentName]; ok {
			completedAt = ts
		}

		spawns = append(spawns, spawnEntryJSON{
			Name:        e.AgentName,
			Parent:      e.ParentName,
			Caste:       e.Caste,
			Task:        e.Task,
			Status:      e.Status,
			SpawnedAt:   e.Timestamp,
			CompletedAt: completedAt,
		})

		if IsLiveSpawnStatus(e.Status) {
			activeCount++
		} else if IsTerminalSpawnStatus(e.Status) {
			completedCount++
		}
	}

	result := spawnTreeJSON{
		Spawns: spawns,
	}
	result.Metadata.TotalCount = len(st.entries)
	result.Metadata.ActiveCount = activeCount
	result.Metadata.CompletedCount = completedCount
	if runState, err := st.loadRunStateLocked(); err == nil {
		result.Metadata.CurrentRunID = strings.TrimSpace(runState.CurrentRunID)
	}

	return json.MarshalIndent(result, "", "  ")
}

// IsLiveSpawnStatus reports whether a worker status should be treated as in-flight.
func IsLiveSpawnStatus(status string) bool {
	switch normalizeSpawnStatus(status) {
	case "spawned", "starting", "active", "running":
		return true
	default:
		return false
	}
}

// IsTerminalSpawnStatus reports whether a worker status should be treated as finished.
func IsTerminalSpawnStatus(status string) bool {
	switch normalizeSpawnStatus(status) {
	case "completed", "failed", "blocked", "timeout", "superseded", "manually-reconciled":
		return true
	default:
		return false
	}
}
