package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// maxBackups is the maximum number of backup files to retain per base file.
// Matches the shell MAX_BACKUPS=3 from atomic-write.sh.
const maxBackups = 3

// CreateBackup copies the file at path to a timestamped backup file.
// If the original file does not exist, it returns nil (nothing to back up).
// Backup files are named: path + ".bak.{timestamp}" matching the shell pattern.
func (s *Store) CreateBackup(path string) error {
	fullPath := s.resolvePath(path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // nothing to back up
	}
	backupPath := fmt.Sprintf("%s.bak.%s", fullPath, time.Now().Format("20060102-150405"))
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("storage: read for backup %q: %w", fullPath, err)
	}
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("storage: write backup %q: %w", backupPath, err)
	}
	return nil
}

// RotateBackups removes the oldest backup files beyond maxBackups.
// Backup files are identified by the glob pattern: path + ".bak.*"
// Files are sorted by modification time (newest first) and the oldest
// entries beyond the limit are deleted.
func (s *Store) RotateBackups(basePath string) error {
	fullPath := s.resolvePath(basePath)
	pattern := fullPath + ".bak.*"
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("storage: glob backups for %q: %w", fullPath, err)
	}
	if len(matches) <= maxBackups {
		return nil
	}
	// Sort by modification time, newest first
	sort.Slice(matches, func(i, j int) bool {
		si, _ := os.Stat(matches[i])
		sj, _ := os.Stat(matches[j])
		return si.ModTime().After(sj.ModTime())
	})
	// Delete oldest beyond maxBackups
	for _, f := range matches[maxBackups:] {
		os.Remove(f)
	}
	return nil
}
