package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
)

// safeIdentifierSegment validates a user-supplied identifier that is about to
// become a single directory name, and returns the contained absolute path.
//
// Swarm ids and chamber names arrive from `--id` / `--name` flags and were
// joined straight onto a base directory, one of them immediately before an
// os.RemoveAll. An id of `../../..` therefore selected a directory far outside
// the colony and deleted it. The identifier is meant to name ONE directory, so
// it may not contain separators or traversal at all — this rejects rather than
// sanitises, because a silently-rewritten id would delete or create the wrong
// thing just as surely.
func safeIdentifierSegment(baseDir, kind, identifier string) (string, error) {
	id := strings.TrimSpace(identifier)
	if id == "" {
		return "", fmt.Errorf("%s is required", kind)
	}
	if id == "." || id == ".." {
		return "", fmt.Errorf("%s %q is not a valid name", kind, identifier)
	}
	if strings.ContainsAny(id, `/\`) {
		return "", fmt.Errorf("%s %q must name a single directory, not a path", kind, identifier)
	}
	if strings.Contains(id, "..") {
		return "", fmt.Errorf("%s %q may not contain path traversal", kind, identifier)
	}
	if strings.ContainsRune(id, 0) {
		return "", fmt.Errorf("%s contains an invalid character", kind)
	}
	// Windows drive-qualified names ("C:evil") are absolute on some hosts.
	if len(id) >= 2 && id[1] == ':' {
		return "", fmt.Errorf("%s %q must not be drive-qualified", kind, identifier)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("resolve %s base directory: %w", kind, err)
	}
	target := filepath.Join(absBase, id)
	rel, err := filepath.Rel(absBase, target)
	if err != nil || rel != id {
		// Belt and braces: after the checks above this cannot normally fire,
		// but a containment guard that trusts its own preconditions is not a
		// containment guard.
		return "", fmt.Errorf("%s %q does not resolve inside %s", kind, identifier, baseDir)
	}
	return target, nil
}
