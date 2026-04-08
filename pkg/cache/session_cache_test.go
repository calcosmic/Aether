package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// helper: write a JSON file and return its path and mtime
func writeJSON(t *testing.T, dir, name string, data interface{}) string {
	t.Helper()
	path := filepath.Join(dir, name)
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal test data: %v", err)
	}
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

// helper: get a fresh temp dir
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "session_cache_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func TestNewSessionCache(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)
	if sc == nil {
		t.Fatal("NewSessionCache returned nil")
	}
	if sc.dataDir != dir {
		t.Errorf("dataDir = %q, want %q", sc.dataDir, dir)
	}
}

func TestGet_MissNoCacheFile(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	val, ok := sc.Get("/nonexistent/file.json")
	if ok {
		t.Error("Get should return false for missing cache")
	}
	if val != nil {
		t.Errorf("Get should return nil, got %v", val)
	}
}

func TestSetAndGet_CacheHit(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create a source file so mtime check works
	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "test.json", map[string]string{"key": "value"})

	data := map[string]interface{}{"cached": true}
	err := sc.Set(srcPath, data)
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, ok := sc.Get(srcPath)
	if !ok {
		t.Fatal("Get should return true after Set")
	}
	got, ok := val.(map[string]interface{})
	if !ok {
		t.Fatal("cached value is not map[string]interface{}")
	}
	if got["cached"] != true {
		t.Errorf("cached value = %v, want cached=true", got["cached"])
	}
}

func TestGet_CacheInvalidation(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create a source file
	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "test.json", map[string]string{"v": "1"})

	// Cache it
	err := sc.Set(srcPath, map[string]interface{}{"v": "1"})
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Modify the file to change mtime
	time.Sleep(10 * time.Millisecond)
	writeJSON(t, srcDir, "test.json", map[string]string{"v": "2"})

	// Cache should be stale now
	_, ok := sc.Get(srcPath)
	if ok {
		t.Error("Get should return false after source file changed (different mtime)")
	}
}

func TestLoad_ReadThroughCache(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "data.json", map[string]string{"hello": "world"})

	// First Load should read from file
	var result map[string]string
	err := sc.Load(srcPath, &result)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if result["hello"] != "world" {
		t.Errorf("result = %v, want hello=world", result)
	}

	// Second Load should hit cache (source file unchanged)
	var cached map[string]string
	err = sc.Load(srcPath, &cached)
	if err != nil {
		t.Fatalf("Load from cache failed: %v", err)
	}
	if cached["hello"] != "world" {
		t.Errorf("cached result = %v, want hello=world", cached)
	}

	// Deleting source file makes cache stale -- Load returns error
	os.Remove(srcPath)
	var stale map[string]string
	err = sc.Load(srcPath, &stale)
	if err == nil {
		t.Error("Load should return error when source file is deleted (stale cache)")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	var result map[string]string
	err := sc.Load("/nonexistent/file.json", &result)
	if err == nil {
		t.Error("Load should return error for missing file")
	}
}

func TestConcurrentAccess(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "concurrent.json", map[string]int{"n": 42})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_ = sc.Set(srcPath, map[string]int{"n": i})
			} else {
				sc.Get(srcPath)
			}
		}(i)
	}
	wg.Wait()
}

func TestClear(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "clear.json", map[string]string{"x": "y"})

	_ = sc.Set(srcPath, map[string]string{"x": "y"})
	count, err := sc.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if count < 1 {
		t.Errorf("Clear returned count %d, want >= 1", count)
	}

	_, ok := sc.Get(srcPath)
	if ok {
		t.Error("Get should return false after Clear")
	}
}

func TestClear_ReturnsCorrectCount(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create multiple cache files via Set
	srcDir := tempDir(t)
	writeJSON(t, srcDir, "a.json", map[string]string{"a": "1"})
	writeJSON(t, srcDir, "b.json", map[string]string{"b": "2"})
	writeJSON(t, srcDir, "c.json", map[string]string{"c": "3"})

	_ = sc.Set(filepath.Join(srcDir, "a.json"), map[string]string{"a": "1"})
	_ = sc.Set(filepath.Join(srcDir, "b.json"), map[string]string{"b": "2"})
	_ = sc.Set(filepath.Join(srcDir, "c.json"), map[string]string{"c": "3"})

	count, err := sc.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("Clear count = %d, want 3", count)
	}

	// Verify no .cache_* files remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), cacheFilePrefix) {
			t.Errorf("found remaining cache file: %s", e.Name())
		}
	}
}

func TestClear_EmptyDirectory(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	count, err := sc.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Clear count = %d, want 0", count)
	}
}

func TestClear_OnlyRemovesCacheFiles(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create a non-cache file in the data directory
	otherFile := filepath.Join(dir, "COLONY_STATE.json")
	if err := os.WriteFile(otherFile, []byte(`{"goal":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a cache file
	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "x.json", map[string]string{"x": "1"})
	_ = sc.Set(srcPath, map[string]string{"x": "1"})

	count, err := sc.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("Clear count = %d, want 1", count)
	}

	// Non-cache file must still exist
	if _, err := os.Stat(otherFile); os.IsNotExist(err) {
		t.Error("Clear should not remove non-cache files")
	}
}

func TestClear_RemovesOrphanedCacheFiles(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create an orphaned cache file not tracked in memory
	orphan := filepath.Join(dir, ".cache_orphan.json")
	if err := os.WriteFile(orphan, []byte(`{"mtime":"0001-01-01T00:00:00Z","data":null}`), 0644); err != nil {
		t.Fatal(err)
	}

	count, err := sc.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("Clear count = %d, want 1 (orphaned file)", count)
	}

	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Error("orphaned cache file should be removed")
	}
}

func TestClearStale_OnlyRemovesOldFiles(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create a "fresh" cache file (just written, mtime is now)
	fresh := filepath.Join(dir, ".cache_fresh.json")
	if err := os.WriteFile(fresh, []byte(`{"mtime":"0001-01-01T00:00:00Z","data":null}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a "stale" cache file with old mtime
	stale := filepath.Join(dir, ".cache_stale.json")
	if err := os.WriteFile(stale, []byte(`{"mtime":"0001-01-01T00:00:00Z","data":null}`), 0644); err != nil {
		t.Fatal(err)
	}
	// Backdate the stale file to 2 hours ago
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, twoHoursAgo, twoHoursAgo); err != nil {
		t.Fatal(err)
	}

	// Clear files older than 1 hour
	count, err := sc.ClearStale(1 * time.Hour)
	if err != nil {
		t.Fatalf("ClearStale returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("ClearStale count = %d, want 1", count)
	}

	// Fresh file should still exist
	if _, err := os.Stat(fresh); os.IsNotExist(err) {
		t.Error("fresh cache file should not be removed by ClearStale")
	}

	// Stale file should be gone
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale cache file should be removed by ClearStale")
	}
}

func TestClearStale_EmptyDirectory(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	count, err := sc.ClearStale(1 * time.Hour)
	if err != nil {
		t.Fatalf("ClearStale returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("ClearStale count = %d, want 0", count)
	}
}

func TestClearStale_OnlyRemovesCacheFiles(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Create a non-cache file with old mtime
	otherFile := filepath.Join(dir, "COLONY_STATE.json")
	if err := os.WriteFile(otherFile, []byte(`{"goal":"test"}`), 0644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(otherFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create a stale cache file
	stale := filepath.Join(dir, ".cache_old.json")
	if err := os.WriteFile(stale, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(stale, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	count, err := sc.ClearStale(1 * time.Hour)
	if err != nil {
		t.Fatalf("ClearStale returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("ClearStale count = %d, want 1", count)
	}

	// Non-cache file must still exist
	if _, err := os.Stat(otherFile); os.IsNotExist(err) {
		t.Error("ClearStale should not remove non-cache files")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	path := filepath.Join(srcDir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	var result map[string]string
	err := sc.Load(path, &result)
	if err == nil {
		t.Error("Load should return error for invalid JSON")
	}
}

func TestIsStale_FreshCache(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "fresh.json", map[string]string{"v": "1"})

	_ = sc.Set(srcPath, map[string]string{"v": "1"})

	if sc.IsStale(srcPath) {
		t.Error("IsStale should return false for fresh cache entry")
	}
}

func TestIsStale_NoCacheEntry(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "missing.json", map[string]string{"v": "1"})

	// No cache entry set
	if !sc.IsStale(srcPath) {
		t.Error("IsStale should return true when no cache entry exists")
	}
}

func TestIsStale_FileDeleted(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "deleted.json", map[string]string{"v": "1"})

	_ = sc.Set(srcPath, map[string]string{"v": "1"})

	// Delete the source file
	os.Remove(srcPath)

	if !sc.IsStale(srcPath) {
		t.Error("IsStale should return true when source file is deleted")
	}
}

func TestIsStale_FileModified(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "modified.json", map[string]string{"v": "1"})

	_ = sc.Set(srcPath, map[string]string{"v": "1"})

	// Modify the file (change mtime)
	time.Sleep(10 * time.Millisecond)
	writeJSON(t, srcDir, "modified.json", map[string]string{"v": "2"})

	if !sc.IsStale(srcPath) {
		t.Error("IsStale should return true after source file is modified")
	}
}

func TestInvalidate_RemovesEntry(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "inv.json", map[string]string{"v": "1"})

	_ = sc.Set(srcPath, map[string]string{"v": "1"})

	// Verify it's cached
	val, ok := sc.Get(srcPath)
	if !ok || val == nil {
		t.Fatal("precondition: entry should be cached")
	}

	// Invalidate
	err := sc.Invalidate(srcPath)
	if err != nil {
		t.Fatalf("Invalidate failed: %v", err)
	}

	// Verify it's gone from memory
	_, ok = sc.Get(srcPath)
	if ok {
		t.Error("Get should return false after Invalidate")
	}

	// Verify disk cache file is removed
	cachePath := sc.cacheFilePath(srcPath)
	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Errorf("cache file %q should be removed after Invalidate", cachePath)
	}
}

func TestInvalidate_NoEntry(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	// Invalidating a non-existent entry should not error
	err := sc.Invalidate("/nonexistent/file.json")
	if err != nil {
		t.Fatalf("Invalidate should not error for missing entry: %v", err)
	}
}

func TestLoad_InvalidatesStaleCache(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "reload.json", map[string]string{"v": "original"})

	// First load populates cache
	var result1 map[string]string
	err := sc.Load(srcPath, &result1)
	if err != nil {
		t.Fatalf("first Load failed: %v", err)
	}
	if result1["v"] != "original" {
		t.Fatalf("first Load result = %v, want v=original", result1)
	}

	// Modify the source file
	time.Sleep(10 * time.Millisecond)
	writeJSON(t, srcDir, "reload.json", map[string]string{"v": "updated"})

	// Second load should detect stale cache and re-read from disk
	var result2 map[string]string
	err = sc.Load(srcPath, &result2)
	if err != nil {
		t.Fatalf("second Load failed: %v", err)
	}
	if result2["v"] != "updated" {
		t.Errorf("second Load result = %v, want v=updated (stale cache should be invalidated)", result2)
	}
}

func TestAutoCleanup_RemovesStaleFilesOnStartup(t *testing.T) {
	// This test verifies the auto-cleanup pattern used in context assembly commands:
	// sc := cache.NewSessionCache(store.BasePath())
	// sc.ClearStale(24 * time.Hour)
	//
	// Stale files (>24h old) should be removed; fresh files should remain.
	dir := tempDir(t)

	// Create a fresh cache file (just written, mtime is now)
	fresh := filepath.Join(dir, ".cache_COLONY_STATE.json")
	if err := os.WriteFile(fresh, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a stale cache file (>24h old)
	stale := filepath.Join(dir, ".cache_pheromones.json")
	if err := os.WriteFile(stale, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	staleTime := time.Now().Add(-25 * time.Hour)
	if err := os.Chtimes(stale, staleTime, staleTime); err != nil {
		t.Fatal(err)
	}

	// Create another stale file just under the threshold (23h)
	almostStale := filepath.Join(dir, ".cache_instincts.json")
	if err := os.WriteFile(almostStale, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	almostTime := time.Now().Add(-23 * time.Hour)
	if err := os.Chtimes(almostStale, almostTime, almostTime); err != nil {
		t.Fatal(err)
	}

	// Simulate what context assembly commands do: create cache and run auto-cleanup
	sc := NewSessionCache(dir)
	sc.ClearStale(24 * time.Hour) // fire-and-forget, errors silently ignored

	// Verify stale file was removed
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("stale cache file (>24h) should be removed by auto-cleanup")
	}

	// Verify fresh file still exists
	if _, err := os.Stat(fresh); os.IsNotExist(err) {
		t.Error("fresh cache file should not be removed by auto-cleanup")
	}

	// Verify almost-stale file (23h) still exists
	if _, err := os.Stat(almostStale); os.IsNotExist(err) {
		t.Error("cache file <24h old should not be removed by auto-cleanup")
	}

	// Verify the cache is still usable after cleanup
	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "test.json", map[string]string{"key": "val"})
	var result map[string]string
	if err := sc.Load(srcPath, &result); err != nil {
		t.Errorf("cache Load should still work after cleanup: %v", err)
	}
}

func TestAutoCleanup_NonBlockingOnEmptyDir(t *testing.T) {
	// Verify auto-cleanup is safe on an empty directory (no stale files to clean)
	dir := tempDir(t)
	sc := NewSessionCache(dir)
	count, _ := sc.ClearStale(24 * time.Hour)
	if count != 0 {
		t.Errorf("expected 0 files cleaned, got %d", count)
	}
}

func TestAutoCleanup_SilentlyIgnoresErrors(t *testing.T) {
	// Verify that ClearStale on a non-existent directory doesn't panic or error
	// when called in fire-and-forget mode (errors discarded)
	sc := NewSessionCache("/nonexistent/path/that/does/not/exist")
	// Fire-and-forget: errors silently ignored
	count, _ := sc.ClearStale(24 * time.Hour)
	// Should return 0, not panic
	if count != 0 {
		t.Errorf("expected 0 files cleaned for non-existent dir, got %d", count)
	}
}

func TestCacheFileNaming(t *testing.T) {
	dir := tempDir(t)
	sc := NewSessionCache(dir)

	srcDir := tempDir(t)
	srcPath := writeJSON(t, srcDir, "pheromones.json", map[string]string{"type": "focus"})

	_ = sc.Set(srcPath, map[string]string{"type": "focus"})

	// A cache file should exist with .cache_ prefix
	expectedPrefix := ".cache_pheromones.json"
	found := false
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	for _, e := range entries {
		if e.Name() == expectedPrefix {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected cache file %q not found in %q", expectedPrefix, dir)
	}
}
