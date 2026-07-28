package memory

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxInstinctContentChars is a hard ceiling, not a truncation point.
//
// The previous behaviour truncated at 100 characters mid-word, producing hub
// entries like `"— the pending, apply observed "`. A mangled rule is worse than
// no rule: it still occupies prompt budget, still carries a confidence score,
// and cannot be acted on. Over-length content is now rejected so the caller can
// fix the observation rather than silently ship a fragment.
const MaxInstinctContentChars = 240

// MinInstinctContentChars filters fragments too short to carry a condition.
const MinInstinctContentChars = 20

var (
	// A path-like token: at least one slash and a file extension, or a
	// recognisable source directory prefix.
	pathPattern = regexp.MustCompile(`[\w./-]+\.[a-zA-Z]{1,5}\b|\b(?:src|pkg|cmd|lib|internal|app|test)/[\w./-]+`)

	// A shell or tool invocation.
	commandPattern = regexp.MustCompile(`\b(?:go|npm|npx|make|git|cargo|python3?|pytest|pnpm|yarn|docker|tsc|eslint|aether)\s+[a-z][\w:-]*`)

	// An error or failure signal worth remembering.
	errorPattern = regexp.MustCompile(`(?i)\b(?:error|panic|exception|failed|failure|exit code|timeout|segfault|nil pointer|undefined|cannot|refused)\b`)

	// Content that merely narrates lifecycle progress carries no information.
	// These killed the previous store: the entire learned memory of the Aether
	// repo after 167 phases was one entry reading "Phase 6 completed successfully".
	narrationPattern = regexp.MustCompile(`(?i)^(?:phase|plan|task|milestone|wave|step)\s+[\w.-]*\s*(?:completed?|finished|done|succeeded|passed|shipped)`)
)

// IsAdmissibleInstinctContent reports whether an observation is concrete enough
// to become durable, injectable memory.
//
// The bar is deliberately mechanical: the content must name a file, a command,
// or an error. Prose that names none of those cannot be checked against a
// repository later, so it can never be invalidated — and memory that cannot be
// invalidated accumulates forever. Every one of the 67 entries found in the hub
// during the July 2026 audit failed this test.
//
// Returns a reason when rejected, so callers can log why rather than silently
// dropping the observation.
func IsAdmissibleInstinctContent(content string) (bool, string) {
	trimmed := strings.TrimSpace(content)

	if len(trimmed) < MinInstinctContentChars {
		return false, fmt.Sprintf("content is %d chars; minimum is %d", len(trimmed), MinInstinctContentChars)
	}
	if len(trimmed) > MaxInstinctContentChars {
		return false, fmt.Sprintf("content is %d chars; maximum is %d (rejected rather than truncated)", len(trimmed), MaxInstinctContentChars)
	}
	if narrationPattern.MatchString(trimmed) {
		return false, "content narrates lifecycle progress rather than describing a reusable pattern"
	}
	if pathPattern.MatchString(trimmed) {
		return true, ""
	}
	if commandPattern.MatchString(trimmed) {
		return true, ""
	}
	if errorPattern.MatchString(trimmed) {
		return true, ""
	}

	return false, "content names no file, command, or error, so it cannot be verified or invalidated later"
}
