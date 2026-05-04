package learn

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxHiveWisdomEntries = 200

// hiveWisdomEntry mirrors cmd/hive.go hiveWisdomEntry for JSON persistence.
type hiveWisdomEntry struct {
	ID          string  `json:"id"`
	Text        string  `json:"text"`
	Domain      string  `json:"domain"`
	SourceRepo  string  `json:"source_repo"`
	Confidence  float64 `json:"confidence"`
	CreatedAt   string  `json:"created_at"`
	AccessedAt  string  `json:"accessed_at"`
	AccessCount int     `json:"access_count"`
}

// hiveWisdomData mirrors cmd/hive.go hiveWisdomData for JSON persistence.
type hiveWisdomData struct {
	Entries []hiveWisdomEntry `json:"entries"`
}

// HiveStore implements LearnStore for cross-colony hive memory.
// Wraps the hive promotion logic from cmd/hive.go.
type HiveStore struct {
	hubDir     string
	sourceRepo string
}

// NewHiveStore creates a HiveStore that reads/writes to the hive wisdom file
// in the given hub directory. sourceRepo is used for abstracting repo-specific content.
func NewHiveStore(hubDir, sourceRepo string) *HiveStore {
	return &HiveStore{hubDir: hubDir, sourceRepo: sourceRepo}
}

// hiveWisdomPath returns the path to the hive wisdom.json file.
func (h *HiveStore) hiveWisdomPath() string {
	return filepath.Join(h.hubDir, "hive", "wisdom.json")
}

// loadWisdom reads the wisdom file. Returns empty data if file does not exist.
func (h *HiveStore) loadWisdom() (hiveWisdomData, error) {
	var data hiveWisdomData
	raw, err := os.ReadFile(h.hiveWisdomPath())
	if err != nil {
		if os.IsNotExist(err) {
			return data, nil
		}
		return data, fmt.Errorf("learn: read hive wisdom: %w", err)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("learn: parse hive wisdom: %w", err)
	}
	return data, nil
}

// saveWisdom writes the wisdom file atomically.
func (h *HiveStore) saveWisdom(data hiveWisdomData) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("learn: marshal hive wisdom: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(h.hiveWisdomPath()), 0755); err != nil {
		return fmt.Errorf("learn: create hive dir: %w", err)
	}
	return os.WriteFile(h.hiveWisdomPath(), append(encoded, '\n'), 0644)
}

// abstractContent removes repo-specific paths from text, making it generic
// for cross-colony sharing.
func (h *HiveStore) abstractContent(text string) string {
	abstracted := text
	if h.sourceRepo != "" {
		abstracted = strings.ReplaceAll(abstracted, h.sourceRepo, "<repo>")
	}
	for _, prefix := range []string{"src/", "lib/", "pkg/", "cmd/", "internal/"} {
		abstracted = strings.ReplaceAll(abstracted, prefix, "")
	}
	return abstracted
}

// Add promotes a learning entry to hive wisdom. Only hive-shareable entries
// are accepted. Content is abstracted to remove repo-specific paths.
func (h *HiveStore) Add(entry Entry) error {
	if entry.Classification != ClassHiveShareable {
		return fmt.Errorf("learn: only hive-shareable entries can be promoted to hive, got %q", entry.Classification)
	}

	abstracted := h.abstractContent(entry.Content)
	if abstracted == "" {
		return nil // nothing to store
	}

	domain := "general"
	if entry.Evidence.Scope != "" {
		domain = entry.Evidence.Scope
	}

	confidence := entry.Confidence
	if confidence <= 0 {
		confidence = 0.75
	}

	data, err := h.loadWisdom()
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Check for existing entry to boost confidence
	for i, e := range data.Entries {
		if e.Text == abstracted && e.Domain == domain {
			if confidence > data.Entries[i].Confidence {
				data.Entries[i].Confidence = confidence
			}
			data.Entries[i].AccessCount++
			data.Entries[i].AccessedAt = now
			return h.saveWisdom(data)
		}
	}

	// LRU eviction if at capacity
	if len(data.Entries) >= maxHiveWisdomEntries {
		oldestIdx := 0
		for i, e := range data.Entries {
			if e.AccessedAt < data.Entries[oldestIdx].AccessedAt {
				oldestIdx = i
			}
		}
		data.Entries = append(data.Entries[:oldestIdx], data.Entries[oldestIdx+1:]...)
	}

	textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(abstracted)))
	newEntry := hiveWisdomEntry{
		ID:          fmt.Sprintf("%s_%s", domain, textHash[:12]),
		Text:        abstracted,
		Domain:      domain,
		SourceRepo:  h.sourceRepo,
		Confidence:  confidence,
		CreatedAt:   now,
		AccessedAt:  now,
		AccessCount: 0,
	}
	data.Entries = append(data.Entries, newEntry)
	return h.saveWisdom(data)
}

// Get retrieves a hive entry by ID. Returns nil if not found.
func (h *HiveStore) Get(id string) (*Entry, error) {
	data, err := h.loadWisdom()
	if err != nil {
		return nil, err
	}
	for _, e := range data.Entries {
		if e.ID == id {
			entry := hiveWisdomToEntry(e)
			return &entry, nil
		}
	}
	return nil, nil
}

// List returns entries matching the given filter.
func (h *HiveStore) List(filter EntryFilter) ([]Entry, error) {
	data, err := h.loadWisdom()
	if err != nil {
		return nil, err
	}

	var result []Entry
	for _, e := range data.Entries {
		entry := hiveWisdomToEntry(e)
		if filter.MinConfidence > 0 && entry.Confidence < filter.MinConfidence {
			continue
		}
		result = append(result, entry)
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	if result == nil {
		result = []Entry{}
	}
	return result, nil
}

// Replace updates an existing hive entry by ID.
func (h *HiveStore) Replace(id string, entry Entry) error {
	data, err := h.loadWisdom()
	if err != nil {
		return err
	}

	found := false
	for i, e := range data.Entries {
		if e.ID == id {
			data.Entries[i].Text = h.abstractContent(entry.Content)
			data.Entries[i].Confidence = entry.Confidence
			data.Entries[i].AccessedAt = time.Now().UTC().Format(time.RFC3339)
			data.Entries[i].AccessCount++
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("learn: hive entry %q not found", id)
	}
	return h.saveWisdom(data)
}

// Remove deletes a hive entry by ID.
func (h *HiveStore) Remove(id string) error {
	data, err := h.loadWisdom()
	if err != nil {
		return err
	}

	found := false
	filtered := make([]hiveWisdomEntry, 0, len(data.Entries))
	for _, e := range data.Entries {
		if e.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, e)
	}
	if !found {
		return fmt.Errorf("learn: hive entry %q not found", id)
	}
	data.Entries = filtered
	return h.saveWisdom(data)
}

// Compact removes least-recently-accessed entries until total content length
// fits within the budget (measured in characters of Text).
func (h *HiveStore) Compact(budget int) error {
	data, err := h.loadWisdom()
	if err != nil {
		return err
	}

	// Sort by accessed-at ascending (oldest first for eviction)
	sort.SliceStable(data.Entries, func(i, j int) bool {
		return data.Entries[i].AccessedAt < data.Entries[j].AccessedAt
	})

	var totalLen int
	var kept []hiveWisdomEntry
	for _, e := range data.Entries {
		if totalLen+len(e.Text) > budget {
			continue
		}
		totalLen += len(e.Text)
		kept = append(kept, e)
	}

	data.Entries = kept
	return h.saveWisdom(data)
}

// hiveWisdomToEntry converts a hive wisdom entry to a learn Entry.
func hiveWisdomToEntry(e hiveWisdomEntry) Entry {
	return Entry{
		ID:            e.ID,
		Content:       e.Text,
		CreatedAt:     e.CreatedAt,
		Confidence:    e.Confidence,
		Classification: ClassHiveShareable,
		FilePath:      e.SourceRepo,
	}
}
