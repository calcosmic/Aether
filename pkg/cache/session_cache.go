// Package cache provides session-level caching for parsed JSON blobs,
// keyed by (filename, mtime) to avoid redundant re-parsing within a
// single CLI invocation.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const cacheFilePrefix = ".cache_"

// cacheEntry holds a cached value alongside the source file's mtime
// at the time it was cached.
type cacheEntry struct {
	Mtime time.Time
	Data  json.RawMessage // use RawMessage to avoid gob/interface{} issues
}

// SessionCache stores parsed JSON blobs keyed by (filename, mtime).
// If the source file's modification time differs from the cached mtime,
// the entry is considered stale.
//
// The in-memory cache is the primary fast path within a single CLI invocation.
// Disk persistence (via JSON) enables cross-invocation caching.
//
// Thread-safe via sync.RWMutex.
type SessionCache struct {
	dataDir string
	mu      sync.RWMutex
	mem     map[string]*cacheEntry
}

// NewSessionCache creates a new SessionCache that stores cache files in dataDir.
func NewSessionCache(dataDir string) *SessionCache {
	return &SessionCache{
		dataDir: dataDir,
		mem:     make(map[string]*cacheEntry),
	}
}

// cacheFilePath returns the on-disk cache file path for a given source filename.
// It uses the .cache_ prefix convention in the data directory.
func (sc *SessionCache) cacheFilePath(filename string) string {
	base := filepath.Base(filename)
	return filepath.Join(sc.dataDir, cacheFilePrefix+base)
}

// sourceMtime returns the modification time of the given file.
func sourceMtime(filename string) (time.Time, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return time.Time{}, fmt.Errorf("stat %q: %w", filename, err)
	}
	return info.ModTime(), nil
}

// getFromMem returns the in-memory cache entry without checking source file mtime.
// This is the fast path for Load within a single session.
func (sc *SessionCache) getFromMem(filename string) (json.RawMessage, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	entry, ok := sc.mem[filename]
	if !ok {
		return nil, false
	}
	return entry.Data, true
}

// Get returns cached data for the given filename if it is still fresh
// (source file mtime matches the cached mtime). Returns (nil, false)
// if the entry is missing, stale, or on any error.
func (sc *SessionCache) Get(filename string) (interface{}, bool) {
	sc.mu.RLock()
	entry, ok := sc.mem[filename]
	sc.mu.RUnlock()

	if !ok {
		return nil, false
	}

	currentMtime, err := sourceMtime(filename)
	if err != nil {
		// Source file gone or unreadable -- cache miss
		return nil, false
	}

	if !currentMtime.Equal(entry.Mtime) {
		return nil, false
	}

	// Unmarshal from raw JSON into a generic interface
	var result interface{}
	if err := json.Unmarshal(entry.Data, &result); err != nil {
		return nil, false
	}
	return result, true
}

// Set stores data in the cache for the given filename, recording the
// current file mtime. The data is persisted to disk via JSON encoding.
func (sc *SessionCache) Set(filename string, data interface{}) error {
	mtime, err := sourceMtime(filename)
	if err != nil {
		return fmt.Errorf("cache Set: %w", err)
	}

	// Serialize data to raw JSON for storage
	raw, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("cache Set: marshal data: %w", err)
	}

	entry := &cacheEntry{Mtime: mtime, Data: raw}

	sc.mu.Lock()
	sc.mem[filename] = entry
	sc.mu.Unlock()

	// Persist to disk via JSON
	cachePath := sc.cacheFilePath(filename)
	entryRaw, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("cache Set: marshal entry: %w", err)
	}
	if err := os.WriteFile(cachePath, entryRaw, 0644); err != nil {
		return fmt.Errorf("cache Set: write cache file: %w", err)
	}

	return nil
}

// Load is a high-level read-through cache method. It checks the in-memory
// cache first; if the entry exists but is stale (source file mtime changed),
// it re-reads from disk. On cache miss, it reads and parses the JSON file,
// stores the result in cache, and returns the parsed data.
//
// The into parameter must be a pointer to the expected data type.
func (sc *SessionCache) Load(filename string, into interface{}) error {
	// Check in-memory cache with mtime validation
	sc.mu.RLock()
	entry, ok := sc.mem[filename]
	sc.mu.RUnlock()

	if ok {
		// Verify the source file hasn't changed since caching
		currentMtime, err := sourceMtime(filename)
		if err == nil && currentMtime.Equal(entry.Mtime) {
			// Fresh cache hit -- unmarshal directly
			return json.Unmarshal(entry.Data, into)
		}
		// Stale: fall through to re-read from disk
	}

	// Cache miss -- read and parse the JSON file
	raw, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("cache Load: read file: %w", err)
	}

	if err := json.Unmarshal(raw, into); err != nil {
		return fmt.Errorf("cache Load: parse JSON: %w", err)
	}

	// Store parsed result in cache (non-fatal on error)
	_ = sc.Set(filename, into)

	return nil
}

// Clear removes all in-memory cache entries and deletes all .cache_* files
// from the data directory (including orphaned files not tracked in memory).
// Returns the count of files removed and any error encountered while
// reading the data directory.
func (sc *SessionCache) Clear() (int, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Scan data directory for all .cache_* files
	entries, err := os.ReadDir(sc.dataDir)
	if err != nil {
		return 0, fmt.Errorf("cache Clear: read data dir: %w", err)
	}

	count := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), cacheFilePrefix) {
			continue
		}
		path := filepath.Join(sc.dataDir, e.Name())
		if err := os.Remove(path); err != nil {
			if !os.IsNotExist(err) {
				return count, fmt.Errorf("cache Clear: remove %q: %w", path, err)
			}
			continue
		}
		count++
	}

	sc.mem = make(map[string]*cacheEntry)
	return count, nil
}

// ClearStale removes .cache_* files from the data directory whose modification
// time is older than maxAge. It does not clear the in-memory map. Returns
// the count of files removed and any error encountered while reading the
// data directory.
func (sc *SessionCache) ClearStale(maxAge time.Duration) (int, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	entries, err := os.ReadDir(sc.dataDir)
	if err != nil {
		return 0, fmt.Errorf("cache ClearStale: read data dir: %w", err)
	}

	cutoff := time.Now().Add(-maxAge)
	count := 0
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), cacheFilePrefix) {
			continue
		}
		path := filepath.Join(sc.dataDir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue // skip files we can't stat
		}
		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err != nil {
				if !os.IsNotExist(err) {
					return count, fmt.Errorf("cache ClearStale: remove %q: %w", path, err)
				}
				continue
			}
			count++
		}
	}

	return count, nil
}

// IsStale reports whether the cache entry for the given filename is stale.
// An entry is stale if:
//   - no cache entry exists in memory, or
//   - the source file no longer exists or is unreadable, or
//   - the source file's mtime differs from the cached mtime.
func (sc *SessionCache) IsStale(filename string) bool {
	sc.mu.RLock()
	entry, ok := sc.mem[filename]
	sc.mu.RUnlock()

	if !ok {
		return true
	}

	currentMtime, err := sourceMtime(filename)
	if err != nil {
		return true
	}

	return !currentMtime.Equal(entry.Mtime)
}

// Invalidate removes a specific cache entry from both memory and disk.
// It is a no-op (returns nil) if no cache entry exists for the filename.
func (sc *SessionCache) Invalidate(filename string) error {
	sc.mu.Lock()
	delete(sc.mem, filename)
	sc.mu.Unlock()

	cachePath := sc.cacheFilePath(filename)
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("cache Invalidate: remove cache file: %w", err)
	}

	return nil
}
