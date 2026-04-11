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
	Timestamp  string // Field 1: ISO 8601 UTC (2006-01-02T15:04:05Z)
	ParentName string // Field 2: parent agent name
	Caste      string // Field 3: caste name (builder, watcher, etc.)
	AgentName  string // Field 4: agent name
	Task       string // Field 5: task description
	Depth      int    // Field 6: spawn depth
	Status     string // Field 7: "spawned", "active", "completed", "failed", "blocked"
	Summary    string // Completion summary (set via spawn-complete --summary)
}

// completionLine represents a status update line in the spawn tree file.
// Format: timestamp|name|status|summary (summary is empty for backward compat)
type completionLine struct {
	Timestamp string
	Name      string
	Status    string
	Summary   string
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
	// Load existing entries (graceful: empty if file missing)
	entries, completions, _ := st.parseFile()
	st.entries = entries
	st.completions = completions
	return st
}

// RecordSpawn creates a new spawn entry and persists it to the file.
func (st *SpawnTree) RecordSpawn(parent, caste, name, task string, depth int) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	entry := SpawnEntry{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		ParentName: parent,
		Caste:      caste,
		AgentName:  name,
		Task:       task,
		Depth:      depth,
		Status:     "spawned",
	}
	st.entries = append(st.entries, entry)
	return st.persistLocked()
}

// UpdateStatus finds an entry by agent name and updates its status.
// It adds a completion line to the file matching the shell's second awk rule.
// The summary is an optional description stored alongside the completion.
func (st *SpawnTree) UpdateStatus(name string, status string, summary string) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	// Find and update the entry
	found := false
	for i := range st.entries {
		if st.entries[i].AgentName == name {
			st.entries[i].Status = status
			st.entries[i].Summary = summary
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("spawn_tree: agent %q not found", name)
	}

	// Add completion line
	st.completions = append(st.completions, completionLine{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Name:      name,
		Status:    status,
		Summary:   summary,
	})

	return st.persistLocked()
}

// Persist writes all entries to the file via store.AtomicWrite.
func (st *SpawnTree) Persist() error {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.persistLocked()
}

// persistLocked writes all entries to file. Caller must hold the mutex.
func (st *SpawnTree) persistLocked() error {
	var lines []string

	// Spawned lines: 7 fields timestamp|parent|caste|name|task|depth|status
	for _, e := range st.entries {
		line := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
			e.Timestamp, e.ParentName, e.Caste, e.AgentName, e.Task, e.Depth, e.Status)
		lines = append(lines, line)
	}

	// Completion lines: timestamp|name|status|summary
	// When summary is empty, format is timestamp|name|status| (backward compat)
	for _, c := range st.completions {
		line := fmt.Sprintf("%s|%s|%s|%s", c.Timestamp, c.Name, c.Status, c.Summary)
		lines = append(lines, line)
	}

	data := []byte(strings.Join(lines, "\n") + "\n")
	return st.store.AtomicWrite(st.filePath, data)
}

// parseFile reads and parses the spawn tree file.
// Returns spawn entries, completion lines, and any error.
func (st *SpawnTree) parseFile() ([]SpawnEntry, []completionLine, error) {
	data, err := st.store.ReadFile(st.filePath)
	if err != nil {
		// File doesn't exist -- return empty
		return nil, nil, nil
	}

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

		fields := strings.Split(line, "|")
		fieldCount := len(fields)

		if fieldCount == 7 {
			// Spawn entry: timestamp|parent|caste|name|task|depth|status
			depth, err := strconv.Atoi(fields[5])
			if err != nil {
				continue // skip malformed
			}
			entry := SpawnEntry{
				Timestamp:  fields[0],
				ParentName: fields[1],
				Caste:      fields[2],
				AgentName:  fields[3],
				Task:       fields[4],
				Depth:      depth,
				Status:     fields[6],
			}
			nameToIdx[fields[3]] = len(entries)
			entries = append(entries, entry)
		} else if fieldCount >= 4 {
			// Completion line: timestamp|name|status|summary
			// Old format (no summary): timestamp|name|status|  -> field[3] = ""
			// New format (with summary): timestamp|name|status|summary -> field[3] = summary
			status := fields[2]
			if status == "completed" || status == "failed" || status == "blocked" {
				summary := fields[3]
				completions = append(completions, completionLine{
					Timestamp: fields[0],
					Name:      fields[1],
					Status:    status,
					Summary:   summary,
				})
				// Merge status and summary into the matching spawn entry
				if idx, ok := nameToIdx[fields[1]]; ok {
					entries[idx].Status = status
					if summary != "" {
						entries[idx].Summary = summary
					}
				}
			}
		}
	}

	return entries, completions, nil
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

	var active []SpawnEntry
	for _, e := range st.entries {
		if e.Status == "spawned" || e.Status == "active" {
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
		TotalCount     int `json:"total_count"`
		ActiveCount    int `json:"active_count"`
		CompletedCount int `json:"completed_count"`
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

		if e.Status == "spawned" || e.Status == "active" {
			activeCount++
		} else if e.Status == "completed" || e.Status == "failed" || e.Status == "blocked" {
			completedCount++
		}
	}

	result := spawnTreeJSON{
		Spawns: spawns,
	}
	result.Metadata.TotalCount = len(st.entries)
	result.Metadata.ActiveCount = activeCount
	result.Metadata.CompletedCount = completedCount

	return json.MarshalIndent(result, "", "  ")
}
