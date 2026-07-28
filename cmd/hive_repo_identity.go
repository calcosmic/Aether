package cmd

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// stableRepoIdentity returns a stable cross-machine identity for a repository,
// derived from its normalized git remote URL when available and from its
// absolute path otherwise. The display name of a repository is deliberately
// not used: the same repository must produce the same identity no matter what
// directory name or path it is checked out under, and different repositories
// must never share one.
func stableRepoIdentity(repoRoot string) string {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return ""
	}
	if remote := repoRemoteURL(repoRoot); remote != "" {
		return "repo_" + hashRepoIdentity(normalizeRepoRemoteURL(remote))
	}
	if abs, err := filepath.Abs(repoRoot); err == nil {
		return "path_" + hashRepoIdentity(filepath.Clean(abs))
	}
	return ""
}

func repoRemoteURL(repoRoot string) string {
	out, err := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// normalizeRepoRemoteURL reduces a git remote URL to a canonical host/path
// form so that equivalent remotes (https vs ssh, with or without .git) share
// one identity.
func normalizeRepoRemoteURL(remote string) string {
	remote = strings.ToLower(strings.TrimSpace(remote))
	remote = strings.TrimSuffix(remote, "/")
	switch {
	case strings.HasPrefix(remote, "git@"):
		// git@host:owner/repo -> host/owner/repo
		remote = strings.TrimPrefix(remote, "git@")
		if idx := strings.Index(remote, ":"); idx >= 0 {
			remote = remote[:idx] + "/" + remote[idx+1:]
		}
	case strings.HasPrefix(remote, "ssh://git@"):
		remote = strings.TrimPrefix(remote, "ssh://git@")
	case strings.HasPrefix(remote, "https://"):
		remote = strings.TrimPrefix(remote, "https://")
	case strings.HasPrefix(remote, "http://"):
		remote = strings.TrimPrefix(remote, "http://")
	}
	remote = strings.TrimSuffix(remote, ".git")
	remote = strings.TrimSuffix(remote, "/")
	return remote
}

func hashRepoIdentity(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum)[:12]
}

// currentRepoIdentity resolves the stable identity of the repository the
// process is running in. Returns "" when the directory is not identifiable.
func currentRepoIdentity() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return stableRepoIdentity(cwd)
}
