package learn

import (
	"regexp"
	"strings"
)

// PrivacyScanResult mirrors cmd/security_cmds.go PrivacyScanResult.
// Re-declared here to avoid importing cmd/ from pkg/.
type PrivacyScanResult struct {
	Blocked  bool
	Clean    string
	Findings []string
}

// secretPattern detects API keys and similar secret patterns that should
// block classification. Used by ClassifyEntry for content that wasn't
// pre-scanned by privacyScan (e.g., in tests).
var secretPattern = regexp.MustCompile(`(?i)(?:sk-|key-|token-)[a-zA-Z0-9]{10,}`)

// ClassifyEntry determines the classification of learning content.
// Per D-10, D-11: automatic classification with no user involvement.
//
// Classification rules:
//   - Blocked: privacy scan flagged secrets (blocked=true)
//   - RepoLocal: privacy scan redacted paths (clean != content)
//   - HiveShareable: clean content passes through unchanged AND is generic
//   - NeedsApproval: clean content passes through but contains repo-specific patterns
func ClassifyEntry(content string, scanResult PrivacyScanResult) Classification {
	if scanResult.Blocked {
		return ClassBlocked
	}
	if scanResult.Clean != content {
		// Content was redacted (paths removed by privacy scan)
		return ClassRepoLocal
	}
	if IsGeneric(content) {
		return ClassHiveShareable
	}
	return ClassNeedsApproval
}

// IsGeneric returns true if content contains no repo-specific patterns:
// no file paths (containing /), no file extensions (e.g., .go, .ts, .json).
// Generic content is safe for hive sharing (D-10, D-11).
func IsGeneric(content string) bool {
	// No forward slashes (indicates file paths)
	if strings.Contains(content, "/") {
		return false
	}
	// No file extensions (e.g., .go, .ts, .json) -- up to 4 chars after dot
	extPattern := regexp.MustCompile(`\.\w{1,4}\b`)
	if extPattern.MatchString(content) {
		return false
	}
	return true
}
