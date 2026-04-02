package storage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ResolveAetherRoot resolves the Aether root directory using a 3-tier fallback:
//  1. AETHER_ROOT environment variable (if set)
//  2. Git repository root (via git rev-parse --show-toplevel)
//  3. Current working directory (fallback)
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
func ResolveAetherRoot() string {
	if root := os.Getenv("AETHER_ROOT"); root != "" {
		return root
	}
	// Try git root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if out, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fallback to cwd
	dir, _ := os.Getwd()
	return dir
}

// ResolveDataDir resolves the colony data directory.
// If COLONY_DATA_DIR is set, it is returned directly.
// Otherwise, the default path AETHER_ROOT/.aether/data/ is returned.
//
// This matches the COLONY_DATA_DIR override logic from the shell codebase
// where per-colony data directories are resolved via environment variable.
func ResolveDataDir() string {
	if dir := os.Getenv("COLONY_DATA_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(ResolveAetherRoot(), ".aether", "data")
}
