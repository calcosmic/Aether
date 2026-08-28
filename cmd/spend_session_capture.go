package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// spendSessionRel is the store-relative path where the current session's
// platform-delivered transcript location is durably recorded. Deliberately
// NOT added to sanctionedDataWritePrefixes (cmd/hook_cmds.go) -- this
// directory is written by the Go runtime only, and no worker's Write tool
// should ever be allowed to aim at the spend ledger.
const spendSessionRel = "spend/session.json"

// spendSessionSchemaVersion guards loadSpendSessionRecord against reading a
// record shaped by a future, incompatible version of this file.
const spendSessionSchemaVersion = 1

// spendSessionRecord is the durable, per-run record of the platform-delivered
// transcript path (SPEND-02's whole honesty argument, D-06): the Go runtime
// reads this artifact itself instead of trusting a token figure an
// orchestrating LLM might relay through a completion packet.
type spendSessionRecord struct {
	SchemaVersion  int    `json:"schema_version"`
	Platform       string `json:"platform"`
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Cwd            string `json:"cwd"`
	CapturedAt     string `json:"captured_at"`
}

// validateSpendTranscriptPath is the T-174-01 boundary check. It mirrors
// validateAndNormalizeClaimPathToRoot (cmd/codex_build_finalize.go:1629)
// structurally: reject the empty string, reject any path containing a null
// byte, require an absolute path, clean it, resolve the expected root as
// $HOME/.claude/projects, evaluate symlinks on both the root and the
// candidate (falling back to the un-evaluated absolute path when
// EvalSymlinks fails, exactly as the claim-path validator does for its own
// root), and require containment decided by filepath.Rel with a leading-".."
// segment rejection -- never a bare lexical prefix match on the raw strings,
// which a crafted sibling directory name could defeat (e.g.
// "/home/user/.claude/projects-evil").
func validateSpendTranscriptPath(claimed string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	root := filepath.Join(home, ".claude", "projects")

	return validateSpendContainedPath(root, claimed, "transcript path")
}

// validateSpendContainedPath is the ONE containment boundary for every path the
// spend subsystem opens, whichever platform's store it belongs to. Both callers
// -- validateSpendTranscriptPath (Claude Code's projects directory) and
// validateOpenCodeStoragePath (OpenCode's local storage tree) -- go through it.
//
// It was two copies until Phase 196 plan 05. Two copies of a security boundary
// is one copy too many: the salvaged OpenCode reader restated the same rule and
// added its own symlink helper without refactoring the original to use it, so a
// later fix to one copy would silently have left the other wrong.
// TestSpendPathContainmentHasOneImplementation fails if a second copy appears.
//
// The rule, unchanged: reject an empty path, a null byte and a relative path;
// filepath.Clean; resolve symlinks on BOTH root and candidate; then decide
// containment with filepath.Rel and a leading ".." rejection -- never a bare
// lexical prefix match on the raw strings, which a crafted sibling directory
// name such as "<root>-evil" would defeat.
//
// label names the thing being validated in plain English, so the diagnostic a
// caller surfaces says what was refused rather than quoting an internal name.
func validateSpendContainedPath(root, claimed, label string) (string, error) {
	claimed = strings.TrimSpace(claimed)
	if claimed == "" {
		return "", fmt.Errorf("%s is empty", label)
	}
	if strings.ContainsRune(claimed, 0) {
		return "", fmt.Errorf("%s contains a null byte", label)
	}
	if !filepath.IsAbs(claimed) {
		return "", fmt.Errorf("%s %q must be absolute", label, claimed)
	}
	candidate := filepath.Clean(claimed)

	rootEval := evalSpendPathSymlinks(root)
	candidateEval := evalSpendPathSymlinks(candidate)

	if candidateEval == rootEval {
		return candidate, nil
	}
	rel, err := filepath.Rel(rootEval, candidateEval)
	if err != nil {
		return "", fmt.Errorf("%s %q does not resolve under %q", label, claimed, root)
	}
	if strings.SplitN(rel, string(filepath.Separator), 2)[0] == ".." {
		return "", fmt.Errorf("%s %q escapes %q", label, claimed, root)
	}

	return candidate, nil
}

// evalSpendPathSymlinks resolves symlinks in path, and — where path does not
// exist yet — resolves symlinks in its deepest EXISTING ancestor and rejoins
// the remainder.
//
// The plain filepath.EvalSymlinks fallback this replaced returned the
// un-evaluated path whenever the target was absent, which made containment
// compare an evaluated root against an unevaluated candidate. On macOS, where
// /var is a symlink to /private/var, that rejected a perfectly legitimate
// transcript path purely because the file did not exist yet — the containment
// answer depended on whether the file happened to have been written, which is
// not what containment means. Found by Phase 196 plan 03, whose reader must
// answer "no rows, no error" for a session that never wrote a transcript.
//
// It cannot loosen containment: every segment that exists is still resolved,
// filepath.Clean has already collapsed any "..", and the caller still decides
// containment with filepath.Rel rather than a lexical prefix match.
func evalSpendPathSymlinks(path string) string {
	if evaluated, err := filepath.EvalSymlinks(path); err == nil {
		return evaluated
	}
	remainder := ""
	current := path
	for {
		parent := filepath.Dir(current)
		if parent == current {
			return path
		}
		remainder = filepath.Join(filepath.Base(current), remainder)
		if evaluated, err := filepath.EvalSymlinks(parent); err == nil {
			return filepath.Join(evaluated, remainder)
		}
		current = parent
	}
}

// recordSpendSessionFromHook is the fail-soft entry point called on every
// PreToolUse hook fire (T-173-02's non-blocking discipline, matching
// captureRawHookPayload). It has no return value and every error path
// returns silently -- a capture failure must never change the hook's
// allow/deny answer for a dispatch (T-174-09).
func recordSpendSessionFromHook(in claudeHookInput) {
	if store == nil {
		return
	}
	if strings.TrimSpace(in.TranscriptPath) == "" {
		return
	}
	transcriptPath, err := validateSpendTranscriptPath(in.TranscriptPath)
	if err != nil {
		return
	}

	var existing spendSessionRecord
	if loadErr := store.LoadJSON(spendSessionRel, &existing); loadErr == nil {
		if existing.TranscriptPath == transcriptPath && existing.SessionID == in.SessionID {
			// Repeated dispatches in the same session leave the file
			// untouched -- idempotent on repeat.
			return
		}
	}

	record := spendSessionRecord{
		SchemaVersion:  spendSessionSchemaVersion,
		Platform:       "claude-code",
		SessionID:      in.SessionID,
		TranscriptPath: transcriptPath,
		Cwd:            in.Cwd,
		CapturedAt:     time.Now().UTC().Format(time.RFC3339),
	}
	_ = store.SaveJSON(spendSessionRel, record)
}

// loadSpendSessionRecord is the read-side helper used by plan 174-04's
// parser. ok is false when the store is nil, the file is absent, the JSON is
// malformed, or schema_version does not match spendSessionSchemaVersion.
func loadSpendSessionRecord() (spendSessionRecord, bool) {
	if store == nil {
		return spendSessionRecord{}, false
	}
	var record spendSessionRecord
	if err := store.LoadJSON(spendSessionRel, &record); err != nil {
		return spendSessionRecord{}, false
	}
	if record.SchemaVersion != spendSessionSchemaVersion {
		return spendSessionRecord{}, false
	}
	return record, true
}
