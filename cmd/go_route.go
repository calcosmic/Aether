package cmd

// Phase 209 plan 01, Task 1/2 -- the one route authority for `/ant-go`.
//
// This file holds the size decision and nothing else. `cmd/go_cmd.go`
// dispatches from resolveJobSizeRoute's answer alone and computes no route
// of its own -- TestGoRouteHasOneAuthority's AST half in
// cmd/go_route_test.go enforces that structurally, the same way
// TestStatusLineComesFromTheSharedDecision (cmd/status_line_test.go)
// enforces it for the permanent status line.

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/calcosmic/Aether/pkg/codegraph"
	"github.com/calcosmic/Aether/pkg/colony"
)

// jobSizeRoute is the two possible answers `/ant-go`'s size decision can
// give: do the job now with one helper (the existing quick-job path), or
// hand it to the planning route instead.
type jobSizeRoute string

const (
	jobSizeRouteSmall jobSizeRoute = "small"
	jobSizeRouteBig   jobSizeRoute = "big"
)

// smallJobFileBudget is the one place the small-job size threshold is
// written. A sentence whose concrete referents resolve to more files than
// this in the real working tree is too big for the one-helper quick route.
const smallJobFileBudget = 3

// jobSizeWalkBudget bounds how many filesystem entries gatherJobSizeFacts
// will visit before giving up. It is a read-only safety valve against a
// pathological working tree, never a size signal itself.
const jobSizeWalkBudget = 20000

// gatherJobSizeFacts never maintains its own skip list -- it reuses
// codegraph.ShouldSkipDir, the one canonical set of noise directories every
// tree-walk in this repository must use (pkg/codegraph/scan_filter.go),
// so a divergent local list can never quietly re-appear here
// (TestSkipListDivergence).

// smallAttemptFacts are facts about a small-route attempt that already
// ran -- all of them independently measured (the disk-measured
// changed-file list and the project's own check outcome), never anything a
// helper wrote about itself. resolveJobSizeRoute's attempt branch (plan
// 02) is the reader of it.
type smallAttemptFacts struct {
	FilesChanged int
	ChecksStatus string
	Verdict      colony.WorkOutcome
}

// jobSizeFacts are the read-only, independently-gathered facts
// resolveJobSizeRoute decides from.
type jobSizeFacts struct {
	Job                 string
	MatchedPaths        []string
	ColonyActive        bool
	PlanAcceptedUnbuilt bool
	Attempt             *smallAttemptFacts
}

// jobSizeDecision is what resolveJobSizeRoute returns. Reason is a
// finished plain-English sentence and is never empty.
type jobSizeDecision struct {
	Route     jobSizeRoute
	Reason    string
	Escalated bool
}

// resolveJobSizeRoute is the one route authority for `/ant-go`. It has two
// layers:
//
//  1. resolveJobSizeRouteBase decides from the pre-attempt facts alone, in
//     priority order exactly like resolveVerificationDepth
//     (cmd/review_depth.go).
//  2. When facts.Attempt is non-nil (the small route's own attempt has
//     already run), this is the highest-priority branch: a job already
//     sized big before the attempt never comes back down (D-03 is a
//     one-way move only, so the base decision wins unchanged and Escalated
//     stays false); otherwise the attempt's own measured facts -- and
//     nothing a helper wrote about itself -- decide whether the job is
//     moved up to the big route, with Escalated set to true when it is.
func resolveJobSizeRoute(facts jobSizeFacts) jobSizeDecision {
	base := resolveJobSizeRouteBase(facts)
	if facts.Attempt == nil {
		return base
	}
	if base.Route == jobSizeRouteBig {
		// Already big before the attempt: this is not a fresh escalation,
		// and nothing ever moves a job back down to small.
		return base
	}
	if reason, escalate := jobSizeAttemptEscalationReason(*facts.Attempt); escalate {
		return jobSizeDecision{
			Route:     jobSizeRouteBig,
			Reason:    reason,
			Escalated: true,
		}
	}
	return base
}

// resolveJobSizeRouteBase is resolveJobSizeRoute's pre-attempt priority
// chain:
//
//  1. an accepted plan with outstanding work always means the big route,
//  2. zero matched paths means the big route (nothing in the project
//     matches what the sentence named, so this reads as new work rather
//     than a change to something that already exists) -- when no project
//     is recorded at all, the reason says plainly that this job is being
//     started as a piece of planned work under that sentence as its goal,
//  3. more matched paths than smallJobFileBudget means the big route,
//  4. otherwise the small route.
func resolveJobSizeRouteBase(facts jobSizeFacts) jobSizeDecision {
	if facts.PlanAcceptedUnbuilt {
		return jobSizeDecision{
			Route: jobSizeRouteBig,
			Reason: "an accepted plan with work still to do already exists for this project, " +
				"so this job joins the planning route instead of skipping ahead of it",
		}
	}
	matched := len(facts.MatchedPaths)
	if matched == 0 {
		if !facts.ColonyActive {
			return jobSizeDecision{
				Route: jobSizeRouteBig,
				Reason: fmt.Sprintf("no project is set up in this folder yet, so this job is being started "+
					"as a piece of planned work under %q as its goal", facts.Job),
			}
		}
		return jobSizeDecision{
			Route: jobSizeRouteBig,
			Reason: "nothing in this project matches what the sentence named, so this reads as " +
				"new work rather than a change to something that already exists",
		}
	}
	if matched > smallJobFileBudget {
		return jobSizeDecision{
			Route: jobSizeRouteBig,
			Reason: fmt.Sprintf("the sentence names %d files already in this project, more than the "+
				"%d-file budget for a job one helper can finish in one pass", matched, smallJobFileBudget),
		}
	}
	return jobSizeDecision{
		Route: jobSizeRouteSmall,
		Reason: fmt.Sprintf("the sentence names %d file(s) already in this project, small enough for "+
			"one helper to finish in one pass", matched),
	}
}

// jobSizeAttemptEscalationReason is D-03's escalation rule for a small
// attempt that already ran: escalate when it measurably changed more
// files than the small-job budget, or when the project's own checks
// genuinely FAILED on it (a blocker verdict). Both signals are
// independently measured -- the disk-measured changed-file count and the
// project's own check outcome -- never a helper's own account of its work.
//
// An attempt whose checks could not be run at all (a partial verdict --
// no verification command resolved, or it ran out of time) is deliberately
// excluded here. That is not evidence the job was bigger than it looked;
// it is the ordinary state of a brand-new project with no check command
// configured. The quick path already keeps this distinction on purpose and
// says so in its own words (cmd/command_truth.go, next to
// quickChecksPassed/quickChecksFailed/quickChecksNotResolved): "not
// checked" is never folded into either passed or failed. Defect register
// entry 61 was exactly that fold happening here, one level up -- this rule
// restores the existing distinction rather than inventing a new one.
//
// Returns ("", false) when neither signal fires.
func jobSizeAttemptEscalationReason(attempt smallAttemptFacts) (string, bool) {
	if attempt.FilesChanged > smallJobFileBudget {
		return fmt.Sprintf("the quick attempt actually changed %d file(s), more than the %d-file budget for "+
			"a job one helper can finish in one pass", attempt.FilesChanged, smallJobFileBudget), true
	}
	if attempt.Verdict == colony.WorkOutcomeBlocker {
		status := strings.TrimSpace(attempt.ChecksStatus)
		if status == "" {
			status = quickChecksFailed
		}
		return fmt.Sprintf("the project's own checks did not pass on the quick attempt (status: %s)", status), true
	}
	return "", false
}

// smallAttemptFactsFromQuickResult reads only the independently measured
// facts out of runQuickJob's own result map -- the disk-measured changed
// file list ("files") and the project's own check outcome ("checks_status",
// "work_outcome") -- and never a helper's own account of its work such as
// "summary" or "raw_output".
func smallAttemptFactsFromQuickResult(result map[string]interface{}) smallAttemptFacts {
	var filesChanged int
	if files, ok := result["files"].([]string); ok {
		filesChanged = len(files)
	}
	verdict, _ := result["work_outcome"].(colony.WorkOutcome)
	return smallAttemptFacts{
		FilesChanged: filesChanged,
		ChecksStatus: stringValue(result["checks_status"]),
		Verdict:      verdict,
	}
}

// gatherJobSizeFacts is the read-only, independent fact gatherer
// resolveJobSizeRoute decides from. It creates no directory and no lock: a
// plain walk of the real working tree, plus a plain read of the recorded
// project state the way frontDoorLifecycleProjection (cmd/root.go) reads
// it -- never storage.NewStore.
func gatherJobSizeFacts(root, job string) jobSizeFacts {
	tokens := jobSizeSentenceTokens(job)
	matched := jobSizeMatchedReferents(root, tokens)

	colonyActive := false
	planAcceptedUnbuilt := false
	statePath := filepath.Join(root, ".aether", "data", "COLONY_STATE.json")
	if state, source := readLifecycleState(statePath); source.Provenance == LifecycleFactConfirmed {
		colonyActive = true
		planAcceptedUnbuilt = jobSizePlanHasOutstandingWork(state)
	}

	return jobSizeFacts{
		Job:                 job,
		MatchedPaths:        matched,
		ColonyActive:        colonyActive,
		PlanAcceptedUnbuilt: planAcceptedUnbuilt,
	}
}

// jobSizePlanHasOutstandingWork reports whether the recorded project has an
// accepted plan (at least one phase) with at least one phase not yet
// completed.
func jobSizePlanHasOutstandingWork(state colony.ColonyState) bool {
	if len(state.Plan.Phases) == 0 {
		return false
	}
	for _, phase := range state.Plan.Phases {
		if phase.Status != colony.PhaseCompleted {
			return true
		}
	}
	return false
}

// jobSizeSentenceTokens splits a sentence into its whitespace/punctuation
// delimited words, lowercased. "." "_" and "-" are kept inside a token (not
// treated as delimiters) so a referent like "README.md" or "go_route.go"
// survives as one token that can equal a real file's base name. Tokens
// shorter than 3 characters are dropped -- too short to usefully match a
// real path segment, and the surface most likely to produce an accidental
// match on a common English word.
func jobSizeSentenceTokens(sentence string) []string {
	var tokens []string
	var b strings.Builder
	flush := func() {
		if b.Len() > 0 {
			word := b.String()
			if len(word) >= 3 {
				tokens = append(tokens, word)
			}
			b.Reset()
		}
	}
	for _, r := range sentence {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
			b.WriteRune(unicode.ToLower(r))
		} else {
			flush()
		}
	}
	flush()
	return tokens
}

// jobSizeMatchedReferents walks root once (bounded by jobSizeWalkBudget,
// skipping codegraph.ShouldSkipDir's canonical noise directories) and
// returns every real path whose base name, base name without extension, or
// containing
// directory's name equals one of tokens, case-insensitively. Matching a
// directory expands to every file underneath it, so a sentence naming a
// directory is sized by how many files that directory actually holds, not
// by counting the directory as one match.
func jobSizeMatchedReferents(root string, tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	tokenSet := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = true
	}

	var allFiles []string
	var matchedDirs []string
	visited := 0
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil || rel == "." {
			return nil
		}
		visited++
		if visited > jobSizeWalkBudget {
			return filepath.SkipAll
		}
		relSlash := filepath.ToSlash(rel)
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if codegraph.ShouldSkipDir(name) {
				return filepath.SkipDir
			}
			if tokenSet[name] {
				matchedDirs = append(matchedDirs, relSlash)
			}
			return nil
		}
		allFiles = append(allFiles, relSlash)
		return nil
	})

	matched := map[string]bool{}
	for _, f := range allFiles {
		name := strings.ToLower(filepath.Base(f))
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if tokenSet[name] || (base != "" && tokenSet[base]) {
			matched[f] = true
			continue
		}
		for _, dir := range matchedDirs {
			if f == dir || strings.HasPrefix(f, dir+"/") {
				matched[f] = true
				break
			}
		}
	}
	out := make([]string, 0, len(matched))
	for f := range matched {
		out = append(out, f)
	}
	return uniqueSortedStrings(out)
}
