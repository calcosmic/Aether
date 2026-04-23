package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

type codexArtifactSnapshot struct {
	Existed bool
	ModTime time.Time
}

func snapshotRelativeFiles(root string, relDirs ...string) map[string]codexArtifactSnapshot {
	snapshots := make(map[string]codexArtifactSnapshot)
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return snapshots
	}

	for _, relDir := range relDirs {
		relDir = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relDir)))
		if relDir == "" || relDir == "." {
			continue
		}
		absDir := filepath.Join(root, filepath.FromSlash(relDir))
		info, err := os.Stat(absDir)
		if err != nil || !info.IsDir() {
			continue
		}

		_ = filepath.WalkDir(absDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			relPath, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			relPath = filepath.ToSlash(relPath)
			snapshots[relPath] = codexArtifactSnapshot{
				Existed: true,
				ModTime: info.ModTime(),
			}
			return nil
		})
	}

	return snapshots
}

func claimedArtifactSet(claimedFiles []string) map[string]bool {
	set := make(map[string]bool, len(claimedFiles))
	for _, file := range claimedFiles {
		file = filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
		if file == "" || file == "." {
			continue
		}
		set[file] = true
	}
	return set
}

func shouldPreserveWorkerArtifact(root string, relPath string, before map[string]codexArtifactSnapshot, claimed map[string]bool) bool {
	relPath = filepath.ToSlash(filepath.Clean(strings.TrimSpace(relPath)))
	if relPath == "" || relPath == "." {
		return false
	}
	if claimed[relPath] {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relPath))); err == nil {
			return true
		}
	}

	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(relPath)))
	if err != nil || info.IsDir() {
		return false
	}

	// If a fallback marker exists, only preserve artifacts newer than the marker.
	// This allows real workers to overwrite stale fallback artifacts.
	fallbackMarkerPath := filepath.Join(root, ".aether", "data", "planning", ".fallback-marker")
	if markerInfo, mErr := os.Stat(fallbackMarkerPath); mErr == nil {
		return info.ModTime().After(markerInfo.ModTime())
	}

	snapshot, existed := before[relPath]
	if !existed || !snapshot.Existed {
		return true
	}
	return info.ModTime().After(snapshot.ModTime)
}
