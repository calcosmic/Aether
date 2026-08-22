package cmd

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// checkFailureIndexMaxExcerpts and checkFailureIndexMaxExcerptChars bound how
// much of a failing check's raw output this index ever carries. D-02: a
// failed check reaches the fix builder as a compact index, not the whole
// log (spec section 5.1's large-tool-output-externalisation idea, narrowed
// to exactly what this task needs -- the general facility spec section 5.1
// describes is deliberately NOT built here, see buildCheckFailureIndex's doc
// comment).
const (
	checkFailureIndexMaxExcerpts     = 5
	checkFailureIndexMaxExcerptChars = 200
)

// checkFailureIndex is the compact, bounded summary of one failing
// verification step a fix-attempt builder receives instead of the whole
// command log (D-02). It is deliberately small: the check's name and
// command, whether it timed out, a handful of deduplicated failure
// locations, and which task(s) this phase's own claimed changed files tie
// the failure to.
type checkFailureIndex struct {
	Check             string   `json:"check"`
	Command           string   `json:"command,omitempty"`
	ExitCode          int      `json:"exit_code,omitempty"`
	TimedOut          bool     `json:"timed_out,omitempty"`
	Excerpts          []string `json:"excerpts,omitempty"`
	ImplicatedTaskIDs []string `json:"implicated_task_ids,omitempty"`
	Truncated         bool     `json:"truncated,omitempty"`
}

// buildCheckFailureIndex builds the compact failure index for one failing
// verification step. It never carries the whole command output -- only the
// first few distinct failure lines, capped in count and per-line length,
// with Truncated set whenever anything was dropped.
//
// Deliberate scope boundary: this is NOT a general large-tool-output
// externalisation facility (the wider design in the priority spec's section
// 5.1). It builds exactly the index D-02's single bounded fix attempt needs
// -- a general facility, if one is ever built, is later-stage work outside
// this phase (193-CONTEXT.md, Deferred Ideas).
func buildCheckFailureIndex(step codexVerificationStep, claims codexBuildClaims, phase colony.Phase) checkFailureIndex {
	excerpts, truncated := compactFailureExcerpts(step.Output, step.Summary)
	return checkFailureIndex{
		Check:             step.Name,
		Command:           step.Command,
		ExitCode:          step.ExitCode,
		TimedOut:          step.TimedOut,
		Excerpts:          excerpts,
		ImplicatedTaskIDs: implicatedTaskIDsFromExcerpts(excerpts, claims, phase),
		Truncated:         truncated,
	}
}

// compactFailureExcerpts extracts the first few distinct, non-empty lines
// from a failing step's raw output, deduplicated in first-seen order and
// bounded in both count (checkFailureIndexMaxExcerpts) and per-line length
// (checkFailureIndexMaxExcerptChars). When the output produced nothing
// usable (a blocked step with no command output, for example), it falls
// back to the step's own one-line Summary so the index is never empty for a
// genuine failure. Truncated is set whenever excerpts were dropped by count
// or a line was cut short by length -- never when nothing needed dropping.
func compactFailureExcerpts(output, summary string) ([]string, bool) {
	var distinct []string
	seen := map[string]bool{}
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		distinct = append(distinct, line)
	}

	truncated := len(distinct) > checkFailureIndexMaxExcerpts
	if truncated {
		distinct = distinct[:checkFailureIndexMaxExcerpts]
	}

	excerpts := make([]string, 0, len(distinct))
	for _, line := range distinct {
		if len(line) > checkFailureIndexMaxExcerptChars {
			line = line[:checkFailureIndexMaxExcerptChars] + "…"
			truncated = true
		}
		excerpts = append(excerpts, line)
	}

	if len(excerpts) == 0 {
		if summary := strings.TrimSpace(summary); summary != "" {
			excerpts = []string{summary}
		}
	}
	return excerpts, truncated
}

// failureExcerptPathPattern matches a source-file reference inside a
// failure line -- either a repository-relative path ("cmd/foo.go") or a bare
// filename the way Go's own test runner reports it ("foo_test.go:42:").
var failureExcerptPathPattern = regexp.MustCompile(`[\w./-]+\.(?:go|ts|tsx|js|jsx|py|rs|rb)\b`)

// implicatedTaskIDsFromExcerpts intersects the file references found in a
// failing check's excerpts against each task's reported changed files
// (criterionClaimSets, the same normalization every other claim/criterion
// check already uses), matched either by the full normalized path or by
// filename alone -- Go's own test failure lines report only the bare
// filename, not the repository-relative path. When nothing intersects, it
// returns nil rather than guessing which task caused the failure.
func implicatedTaskIDsFromExcerpts(excerpts []string, claims codexBuildClaims, phase colony.Phase) []string {
	var refs []string
	for _, excerpt := range excerpts {
		refs = append(refs, failureExcerptPathPattern.FindAllString(excerpt, -1)...)
	}
	if len(refs) == 0 {
		return nil
	}

	validTaskIDs := map[string]bool{}
	for idx, task := range phase.Tasks {
		validTaskIDs[buildTaskID(task, idx)] = true
	}

	implicated := map[string]bool{}
	for taskID, claimedPaths := range criterionClaimSets(claims) {
		if taskID == "" {
			continue
		}
		if len(validTaskIDs) > 0 && !validTaskIDs[taskID] {
			continue
		}
		for claimedPath := range claimedPaths {
			base := path.Base(claimedPath)
			for _, ref := range refs {
				if ref == claimedPath || path.Base(ref) == base {
					implicated[taskID] = true
				}
			}
		}
	}
	if len(implicated) == 0 {
		return nil
	}
	ids := make([]string, 0, len(implicated))
	for id := range implicated {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
