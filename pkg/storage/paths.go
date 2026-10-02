package storage

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// resolveTimeout is the timeout used for git commands when resolving paths.
const resolveTimeout = 30 * time.Second

// ResolveAetherRoot resolves the Aether root directory using a 4-tier fallback:
//  1. AETHER_ROOT environment variable (if set)
//  2. Git repository root (via git rev-parse --show-toplevel)
//  3. The nearest folder at or above the working directory holding .aether
//  4. Current working directory (fallback)
//
// This matches the shell AETHER_ROOT resolution in atomic-write.sh:
//
//	if [[ -z "${AETHER_ROOT:-}" ]]; then
//	    if git rev-parse --show-toplevel >/dev/null 2>&1; then
//	        AETHER_ROOT="$(git rev-parse --show-toplevel)"
//	    else
//	        AETHER_ROOT="$(pwd)"
//	    fi
//	fi
func ResolveAetherRoot(ctx context.Context) string {
	if root := os.Getenv("AETHER_ROOT"); root != "" {
		return root
	}
	// Try git root
	gitCtx, cancel := context.WithTimeout(ctx, resolveTimeout)
	defer cancel()
	cmd := exec.CommandContext(gitCtx, "git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Outside git, the nearest folder at or above the working directory that
	// already holds a project is the project. Falling straight back to the
	// working directory let a helper that stepped into a subfolder turn that
	// subfolder into a stray second project (Phase 210 blocker 14).
	dir, _ := os.Getwd()
	if project, ok := nearestAetherProject(dir); ok {
		return project
	}
	return dir
}

// nearestAetherProject walks up from start and returns the first folder that
// holds a .aether folder. The home folder is never a project: ~/.aether is the
// shared install every project reads from, not a project of its own, and the
// same goes for a hub moved elsewhere with AETHER_HUB_DIR.
func nearestAetherProject(start string) (string, bool) {
	if start == "" {
		return "", false
	}
	home, _ := os.UserHomeDir()
	hub := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR"))
	for dir := filepath.Clean(start); ; {
		marker := filepath.Join(dir, ".aether")
		if info, err := os.Stat(marker); err == nil && info.IsDir() &&
			!samePathOnDisk(dir, home) && !samePathOnDisk(marker, hub) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func samePathOnDisk(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ai, bi)
}

// ResolveDataDir resolves the colony data directory.
// If COLONY_DATA_DIR is set, it is returned directly.
// Otherwise, the default path AETHER_ROOT/.aether/data/ is returned.
//
// This matches the COLONY_DATA_DIR override logic from the shell codebase
// where per-colony data directories are resolved via environment variable.
func ResolveDataDir(ctx context.Context) string {
	if dir := os.Getenv("COLONY_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(ResolveAetherRoot(ctx), ".aether", "data")
}
