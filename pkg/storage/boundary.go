package storage

import (
	"fmt"
	"path/filepath"
	"strings"
)

// protectedPaths lists paths that require explicit authorization before writing.
// Prefix matches are supported (e.g., "checkpoints/" blocks checkpoints/subdir/file.json).
var protectedPaths = []string{
	"COLONY_STATE.json",
	"session.json",
	"checkpoints/",
	"midden/",
}

// BoundaryGuard protects sensitive colony state files from unauthorized writes.
// It wraps a Store and enforces that protected paths can only be written to
// after an explicit Allow() call.
type BoundaryGuard struct {
	store   *Store
	allowed map[string]bool
}

// NewBoundaryGuard creates a BoundaryGuard that protects the standard set of
// sensitive colony paths.
func NewBoundaryGuard(s *Store) *BoundaryGuard {
	return &BoundaryGuard{
		store:   s,
		allowed: make(map[string]bool),
	}
}

// Allow grants write permission to a specific path.
// The path is cleaned using filepath.Clean to prevent traversal attacks.
func (bg *BoundaryGuard) Allow(path string) {
	cleaned := filepath.Clean(path)
	bg.allowed[cleaned] = true
}

// CheckWrite verifies whether writing to the given path is authorized.
// Returns an error if the path is protected and has not been explicitly allowed.
func (bg *BoundaryGuard) CheckWrite(path string) error {
	cleaned := filepath.Clean(path)

	// Check if explicitly allowed
	if bg.allowed[cleaned] {
		return nil
	}

	// Check against protected paths
	for _, pp := range protectedPaths {
		cleanedPP := strings.TrimRight(pp, "/")
		if cleaned == cleanedPP || strings.HasPrefix(cleaned, cleanedPP+string(filepath.Separator)) {
			return fmt.Errorf("boundary: write to protected path %q requires explicit authorization", path)
		}
	}

	return nil
}
