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

// jobSizeSkipDirNames are directory names gatherJobSizeFacts never descends
// into -- large, machine-generated trees that would swamp the walk without
// ever being what a sentence names.
var jobSizeSkipDirNames = map[string]bool{
	".git":         true,
	"node_modules": true,
}

// smallAttemptFacts are facts about a small-route attempt that already
// ran -- all of them independently measured (the disk-measured
// changed-file list and the project's own check outcome), never anything a
// helper wrote about itself. This plan defines the shape; plan 02's
// escalation branch is the first reader of it, and resolveJobSizeRoute
// deliberately leaves a non-nil Attempt unread here.
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

// resolveJobSizeRoute is the one route authority for `/ant-go`. Priority
// order, exactly like resolveVerificationDepth (cmd/review_depth.go):
//
//  1. an accepted plan with outstanding work always means the big route,
//  2. zero matched paths means the big route (nothing in the project
//     matches what the sentence named, so this reads as new work rather
//     than a change to something that already exists),
//  3. more matched paths than smallJobFileBudget means the big route,
//  4. otherwise the small route.
//
// facts.Attempt is read by a later escalation branch, not by this
// function -- a non-nil Attempt is deliberately ignored here.
func resolveJobSizeRoute(facts jobSizeFacts) jobSizeDecision {
	if facts.PlanAcceptedUnbuilt {
		return jobSizeDecision{
			Route: jobSizeRouteBig,
			Reason: "an accepted plan with work still to do already exists for this project, " +
				"so this job joins the planning route instead of skipping ahead of it",
		}
	}
	matched := len(facts.MatchedPaths)
	if matched == 0 {
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
// skipping jobSizeSkipDirNames and .aether/data) and returns every real
// path whose base name, base name without extension, or containing
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
			if jobSizeSkipDirNames[name] {
				return filepath.SkipDir
			}
			if relSlash == ".aether/data" || strings.HasPrefix(relSlash, ".aether/data/") {
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
