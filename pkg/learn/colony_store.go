package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// entriesFile is the relative path within the store's base directory
// where learning entries are persisted (D-06: data in .aether/data/learn/).
const entriesFile = "entries.json"

// ColonyStore implements LearnStore with repo-isolated JSON persistence.
type ColonyStore struct {
	store  *storage.Store
	nextID atomic.Int64
}

// NewColonyStore creates a ColonyStore backed by the given store.
func NewColonyStore(store *storage.Store) *ColonyStore {
	return &ColonyStore{store: store}
}

// loadEntries reads the entries from disk. Returns empty slice if file does not exist.
func (c *ColonyStore) loadEntries() ([]Entry, error) {
	var entries []Entry
	err := c.store.LoadJSON(entriesFile, &entries)
	if err != nil {
		if _, statErr := os.Stat(filepath.Join(c.store.BasePath(), entriesFile)); os.IsNotExist(statErr) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("learn: load entries: %w", err)
	}
	return entries, nil
}

// generateID creates a unique ID for a learning entry.
func (c *ColonyStore) generateID() string {
	seq := c.nextID.Add(1)
	return fmt.Sprintf("lrn_%s_%d", time.Now().Format("20060102"), seq)
}

// updateEntries atomically reads, mutates, and writes the entries slice.
// It reads the existing file content directly (bypassing LoadJSON to avoid
// lock conflicts with UpdateFile's write lock), mutates via the callback,
// and writes back atomically.
func (c *ColonyStore) updateEntries(mutate func([]Entry) ([]Entry, error)) error {
	return c.store.UpdateFile(entriesFile, func(existing []byte) ([]byte, error) {
		var current []Entry
		if len(existing) > 0 {
			if err := json.Unmarshal(existing, &current); err != nil {
				return nil, fmt.Errorf("learn: unmarshal entries: %w", err)
			}
		}

		updated, err := mutate(current)
		if err != nil {
			return nil, err
		}
		if updated == nil {
			updated = []Entry{}
		}

		data, err := json.MarshalIndent(updated, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("learn: marshal entries: %w", err)
		}
		return append(data, '\n'), nil
	})
}

// Add writes an entry to the store and assigns an ID if empty.
func (c *ColonyStore) Add(entry Entry) error {
	var assignedID string
	err := c.updateEntries(func(entries []Entry) ([]Entry, error) {
		if entry.ID == "" {
			entry.ID = c.generateID()
		}
		assignedID = entry.ID
		return append(entries, entry), nil
	})
	if err == nil && assignedID != "" {
		entry.ID = assignedID
	}
	return err
}

// Get retrieves an entry by ID. Returns nil if not found.
func (c *ColonyStore) Get(id string) (*Entry, error) {
	entries, err := c.loadEntries()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// List returns entries matching the given filter. Returns empty slice if none match.
func (c *ColonyStore) List(filter EntryFilter) ([]Entry, error) {
	entries, err := c.loadEntries()
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, e := range entries {
		if filter.Phase != 0 && e.Phase != filter.Phase {
			continue
		}
		if filter.Classification != "" && e.Classification != filter.Classification {
			continue
		}
		if filter.MinConfidence > 0 && e.Confidence < filter.MinConfidence {
			continue
		}
		result = append(result, e)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	if result == nil {
		result = []Entry{}
	}
	return result, nil
}

// Replace updates an existing entry by ID. Returns error if not found.
func (c *ColonyStore) Replace(id string, entry Entry) error {
	return c.updateEntries(func(entries []Entry) ([]Entry, error) {
		found := false
		for i := range entries {
			if entries[i].ID == id {
				entries[i] = entry
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("learn: entry %q not found", id)
		}
		return entries, nil
	})
}

// Remove deletes an entry by ID. Returns error if not found.
func (c *ColonyStore) Remove(id string) error {
	return c.updateEntries(func(entries []Entry) ([]Entry, error) {
		found := false
		filtered := make([]Entry, 0, len(entries))
		for _, e := range entries {
			if e.ID == id {
				found = true
				continue
			}
			filtered = append(filtered, e)
		}
		if !found {
			return nil, fmt.Errorf("learn: entry %q not found", id)
		}
		return filtered, nil
	})
}

// Compact removes lowest-confidence entries until total content length fits
// within the budget. Higher-confidence entries are kept. Budget is measured in
// characters of Content.
func (c *ColonyStore) Compact(budget int) error {
	return c.updateEntries(func(entries []Entry) ([]Entry, error) {
		// Sort by confidence descending (highest first)
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Confidence > entries[j].Confidence
		})

		// Keep entries until cumulative content length exceeds budget
		var totalLen int
		var kept []Entry
		for _, e := range entries {
			if totalLen+len(e.Content) > budget {
				continue // skip this entry (lowest remaining)
			}
			totalLen += len(e.Content)
			kept = append(kept, e)
		}

		return kept, nil
	})
}
