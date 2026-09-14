package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
)

// ---------------------------------------------------------------------------
// Phase 198.2 plan 04 (WIRE-06, D-15, D-16): the memory pack every helper
// receives may never carry a heading nothing fills. This file enumerates
// every part buildColonyPrimeOutput can build, records who actually writes
// each one, and fails by name the moment a part has neither a writer nor a
// justified, shrink-only exception.
// ---------------------------------------------------------------------------

// memoryPackPartNames walks cmd/colony_prime_context.go's AST and returns
// the deduplicated set of every colonyPrimeSection composite literal's
// "name:" field value -- mechanically, never from a hand-typed list, because
// a hand-typed list is a second surface that can silently go stale (this
// repo's own documented failure mode). The same AST-walk shape is already
// used for a different invariant in
// cmd/memory_feed_test.go:TestEveryBuildLaneFeedsMemoryThroughOneBoundary.
func memoryPackPartNames(t *testing.T) []string {
	t.Helper()
	fset := token.NewFileSet()
	path := filepath.Join(".", "colony_prime_context.go")
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	seen := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		ident, ok := lit.Type.(*ast.Ident)
		if !ok || ident.Name != "colonyPrimeSection" {
			return true
		}
		for _, elt := range lit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := kv.Key.(*ast.Ident)
			if !ok || key.Name != "name" {
				continue
			}
			basicLit, ok := kv.Value.(*ast.BasicLit)
			if !ok || basicLit.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(basicLit.Value)
			if err != nil || value == "" {
				continue
			}
			seen[value] = true
		}
		return true
	})

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// memoryPackPartWriter documents the concrete runtime function that fills a
// memory-pack part, and the command(s) that exercise that writer in
// ordinary colony operation.
//
// A note on "reachedBy": 198.2-CONTEXT.md D-16 names five moments a
// WORKER receives this content -- a build, a check (/ant-continue), a plan,
// a survey (/ant-colonize), or an owner answering a question
// (decision-answer). Those five are the READ lanes: they are the commands
// that actually call buildColonyPrimeOutput and hand the assembled pack to
// a helper. They are not a restriction on which command may WRITE the
// underlying data -- a part such as "charter" is written once, at
// /ant-init, and then READ by every one of the five; a part such as
// "pheromones" is written by /ant-focus/redirect/feedback and READ the same
// way. Both halves (a real write path, and a real read lane that surfaces
// it) are what "a live runtime writer reachable from ordinary operation"
// means for this invariant.
type memoryPackPartWriter struct {
	writer    string // function / file:line that performs the write
	reachedBy string // command(s) that exercise the writer or read the result
}

// memoryPackPartWriters is the writer map validated by
// TestEveryMemoryPackPartHasALiveWriter. "decisions" and "learnings" are
// deliberately ABSENT: they were the two parts this plan's Task 1 run
// found with no writer (state.Memory.Decisions / state.Memory.PhaseLearnings
// are only ever emptied, at cmd/init_cmd.go:284 and cmd/entomb_cmd.go:669,
// never filled), and Task 2 removed their section builders rather than
// adding them here (198.2-CONTEXT.md D-15).
var memoryPackPartWriters = map[string]memoryPackPartWriter{
	"state": {
		writer:    `store.SaveJSON("COLONY_STATE.json", ...) -- written by every phase-advancing command`,
		reachedBy: "/ant-init, /ant-build, /ant-continue, /ant-plan, /ant-colonize",
	},
	"review_depth": {
		writer:    "state.VerificationDepth, set in cmd/codex_plan.go and cmd/codex_build.go / cmd/queen_judgement.go",
		reachedBy: "/ant-plan, /ant-build",
	},
	"charter": {
		writer:    "state.Charter, set once by generateCharter, cmd/init_cmd.go:297",
		reachedBy: "/ant-init (the predecessor every colony runs before build/continue/plan/colonize)",
	},
	"pheromones": {
		writer:    "pheromones.json, written by cmd/pheromone_write.go",
		reachedBy: "/ant-focus, /ant-redirect, /ant-feedback",
	},
	"instincts": {
		writer:    "instincts.json, written by pkg/memory/consolidate.go phase-end consolidation and cmd/instinct.go",
		reachedBy: "/ant-continue (automatic phase-end consolidation)",
	},
	"worker_handoffs": {
		writer:    "worker-handoffs.json, written by persistDispatchWorkerHandoff via cmd/memory_feed.go's recordDispatchWorkerOutcome boundary",
		reachedBy: "/ant-build, /ant-continue",
	},
	"hive_wisdom": {
		writer:    "~/.aether/hive/wisdom.json, written by phase-end and seal-time hive promotion",
		reachedBy: "/ant-continue (every check), /ant-seal",
	},
	"learned_memory": {
		writer:    "entries.json via learn.NewColonyStore(store).Add(...), written in cmd/codex_continue_finalize.go and the build-finalize memory-feed path",
		reachedBy: "/ant-build, /ant-continue",
	},
	// LEARN-01 (204-02-PLAN.md Task 2, ruling (b)): the same entries.json
	// writer as "learned_memory" above -- a fresh entry captureContinueLearning
	// writes is genuinely a hypothesis, and this section is where it is
	// honestly shown until (if ever) it earns learn.StatusValidated and
	// moves to the section above.
	"learned_memory_unverified": {
		writer:    "entries.json via learn.NewColonyStore(store).Add(...), written in cmd/codex_continue_finalize.go and the build-finalize memory-feed path",
		reachedBy: "/ant-build, /ant-continue",
	},
	"global_queen_md": {
		writer:    `hub QUEEN.md, written by "/ant-preferences" and phase-end consolidation's instinct promotion`,
		reachedBy: "/ant-preferences, /ant-continue",
	},
	"user_preferences": {
		writer:    `hub/local QUEEN.md's "## User Preferences" section, written by /ant-preferences and /ant-profile`,
		reachedBy: "/ant-preferences, /ant-profile",
	},
	"prior_reviews": {
		writer:    "reviews/<domain>/ledger.json, written by review workers (Gatekeeper/Auditor/Probe) during a check",
		reachedBy: "/ant-continue",
	},
	"local_queen_wisdom": {
		writer:    "repo-local QUEEN.md, written by /ant-preferences and phase-end consolidation",
		reachedBy: "/ant-preferences, /ant-continue",
	},
	"clarified_intent": {
		writer:    "pending-decisions.json, written by recordDecisionAnswer, cmd/handoff_decisions_cmd.go:102",
		reachedBy: "decision-answer",
	},
	"blockers": {
		writer:    "pending-decisions.json / flags.json, written by /ant-flag and auto-created blocker flags",
		reachedBy: "/ant-flag, /ant-continue",
	},
	"midden": {
		writer:    "midden.json, written by appendMiddenEntry, cmd/midden_shared.go, on build/check failures",
		reachedBy: "/ant-build, /ant-continue",
	},
	"medic_health": {
		writer:    "medic-last-scan.json, written by the medic scan (cmd/medic_auto_spawn.go, auto-spawned during a check) and /ant-medic",
		reachedBy: "/ant-continue (auto-spawn), /ant-medic",
	},
	"activity_tail": {
		writer:    "activity.log, appended by the shared activity-logging helper across commands",
		reachedBy: "every command (ask-mode only surfaces it)",
	},
}

// memoryPackWriterExceptions is the shrink-only allowlist for a memory-pack
// part that legitimately has no writer today. Every entry must carry a
// one-sentence reason. It is empty as of 198.2-04 -- the only two parts
// this invariant ever found without a writer (decisions, learnings) were
// removed rather than excepted.
var memoryPackWriterExceptions = map[string]string{}

// memoryPackWriterExceptionFloor pins the maximum tolerated size of
// memoryPackWriterExceptions. Widening this map requires widening this
// constant in the SAME reviewed change -- see
// TestMemoryPackWriterExceptionsOnlyShrink.
const memoryPackWriterExceptionFloor = 0

// TestEveryMemoryPackPartHasALiveWriter is the D-16 invariant: every part
// buildColonyPrimeOutput can build must have either a writer in
// memoryPackPartWriters, or a justified entry in
// memoryPackWriterExceptions. It fails in both directions -- an
// undiscovered part is named, and so is an orphaned writer-map entry for a
// part that no longer exists -- so the map can never quietly rot.
func TestEveryMemoryPackPartHasALiveWriter(t *testing.T) {
	discovered := memoryPackPartNames(t)

	var missing []string
	for _, name := range discovered {
		if _, ok := memoryPackPartWriters[name]; ok {
			continue
		}
		if _, ok := memoryPackWriterExceptions[name]; ok {
			continue
		}
		missing = append(missing, name)
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("memory-pack part(s) with no writer and no exception-list entry: %s -- every part a helper receives must be filled by something in the running program (198.2-CONTEXT.md D-15/D-16); give it a writer, remove it, or add a justified exception", strings.Join(missing, ", "))
	}

	discoveredSet := map[string]bool{}
	for _, name := range discovered {
		discoveredSet[name] = true
	}
	var orphaned []string
	for name := range memoryPackPartWriters {
		if !discoveredSet[name] {
			orphaned = append(orphaned, name)
		}
	}
	if len(orphaned) > 0 {
		sort.Strings(orphaned)
		t.Errorf("writer-map entries for part(s) that no longer exist in cmd/colony_prime_context.go: %s -- the writer map has gone stale, delete these entries", strings.Join(orphaned, ", "))
	}
}

// TestMemoryPackWriterExceptionsOnlyShrink is the ratchet: the exception
// list may only shrink, and every entry must carry a written reason.
// Widening memoryPackWriterExceptionFloor is a deliberate, reviewed act,
// not something that can happen silently.
func TestMemoryPackWriterExceptionsOnlyShrink(t *testing.T) {
	if len(memoryPackWriterExceptions) > memoryPackWriterExceptionFloor {
		t.Errorf("memoryPackWriterExceptions has grown to %d entries, exceeding the recorded floor of %d -- the exception list may only shrink; give the new part(s) a real writer, or widen the floor in the SAME reviewed change with a written reason for each new entry", len(memoryPackWriterExceptions), memoryPackWriterExceptionFloor)
	}
	for name, reason := range memoryPackWriterExceptions {
		if strings.TrimSpace(reason) == "" {
			t.Errorf("memoryPackWriterExceptions[%q] has no written reason", name)
		}
	}
}

// TestMemoryPackHasNoEmptyHeadings assembles the pack on a colony with
// nothing stored and asserts no "## " heading is followed by zero content
// before the next heading (or the end of the output) -- an absent part
// must be absent, not rendered as an empty heading (WIRE-06, D-16).
func TestMemoryPackHasNoEmptyHeadings(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	output := buildColonyPrimeOutput(false)
	lines := strings.Split(output.Context, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "## ") {
			continue
		}
		hasContent := false
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if strings.HasPrefix(strings.TrimSpace(next), "## ") {
				break
			}
			if strings.TrimSpace(next) != "" {
				hasContent = true
				break
			}
		}
		if !hasContent {
			t.Errorf("heading %q at line %d has no content before the next heading or end of output -- an absent part must be absent, not rendered as an empty heading:\n%s", strings.TrimSpace(line), i, output.Context)
		}
	}
}

// TestMemoryPackIsStableAcrossRuns assembles the pack twice against
// unchanged state and asserts byte-equality, so two parts whose ranking
// scores are exactly equal never silently reorder between runs (WIRE-06).
func TestMemoryPackIsStableAcrossRuns(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "stability test"
	state := colony.ColonyState{
		Version: "1.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Phase One", Status: colony.PhaseReady}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	first := buildColonyPrimeOutput(false)
	second := buildColonyPrimeOutput(false)
	if first.Context != second.Context {
		t.Fatalf("colony-prime output changed across two runs against unchanged state -- equal-scoring parts must keep a deterministic order\n--- run 1 ---\n%s\n--- run 2 ---\n%s", first.Context, second.Context)
	}
}

// ---------------------------------------------------------------------------
// Task 2 (198.2-04): proving nothing that "decisions" / "learnings" used to
// carry has stopped arriving, and that it arrives exactly once.
// ---------------------------------------------------------------------------

// sectionBoundsInContext returns [start, end) for the block that begins at
// the given heading (inclusive of the heading line) and ends immediately
// before the next "## " heading, or at the end of the string.
func sectionBoundsInContext(t *testing.T, context, heading string) (int, int) {
	t.Helper()
	start := strings.Index(context, heading)
	if start < 0 {
		t.Fatalf("heading %q not found in:\n%s", heading, context)
	}
	rest := context[start+len(heading):]
	end := len(context)
	if idx := strings.Index(rest, "\n## "); idx >= 0 {
		end = start + len(heading) + idx
	}
	return start, end
}

// TestPhaseLessonsStillReachAHelperUnderLearnedMemory seeds a verified phase
// lesson exactly the way continue-finalize's learn-store write does
// (learn.CollectEvidence + learn.ClassifyEntry + learn.NewColonyStore.Add,
// mirroring cmd/codex_continue_finalize.go's evidence/entry construction --
// never a hand-typed Entry literal), then asserts the lesson arrives
// exactly once, under Learned Memory, and that the removed "Key Decisions"
// / "Phase Learnings" headings never resurface.
func TestPhaseLessonsStillReachAHelperUnderLearnedMemory(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	const sentinel = "sentinel phase lesson 198-2-04: never skip the RED gate on a TDD plan"

	goal := "phase lessons still reach a helper"
	state := colony.ColonyState{
		Version: "1.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Phase One", Status: colony.PhaseReady}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	evidence := learn.CollectEvidence("run_198_2_04", 1,
		[]learn.WorkerResult{{Name: "Builder-1", Caste: "builder", Status: "completed"}},
		learn.GateResult{Passed: 3, Total: 3}, "repo-local")
	scanResult := privacyScan(sentinel)
	classification := learn.ClassifyEntry(sentinel, learn.PrivacyScanResult{
		Blocked:  scanResult.Blocked,
		Clean:    scanResult.Clean,
		Findings: scanResult.Findings,
	})
	entry := learn.Entry{
		Content:        scanResult.Clean,
		Evidence:       evidence,
		Classification: classification,
		Phase:          1,
		Confidence:     evidence.Confidence,
		Status:         learn.StatusHypothesis,
	}
	learnStore := learn.NewColonyStore(s)
	if err := learnStore.Add(entry); err != nil {
		t.Fatalf("seed learn entry: %v", err)
	}

	output := buildColonyPrimeOutput(false)

	if count := strings.Count(output.Context, sentinel); count != 1 {
		t.Fatalf("expected the phase lesson to arrive exactly once, found %d occurrence(s):\n%s", count, output.Context)
	}
	if strings.Contains(output.Context, "## Key Decisions") || strings.Contains(output.Context, "## Phase Learnings") {
		t.Fatalf("removed heading resurfaced in output:\n%s", output.Context)
	}

	start, end := sectionBoundsInContext(t, output.Context, "## LEARNED MEMORY")
	sentinelIdx := strings.Index(output.Context, sentinel)
	if sentinelIdx < start || sentinelIdx >= end {
		t.Fatalf("sentinel landed outside the Learned Memory block (block=[%d,%d), sentinel=%d):\n%s", start, end, sentinelIdx, output.Context)
	}
}

// TestOwnerAnswersStillReachAHelperUnderClarifiedIntent seeds an owner
// answer through recordDecisionAnswer -- the exact function the
// decision-answer command calls -- and asserts it arrives exactly once,
// under Clarified Intent, with the removed "Key Decisions" heading never
// resurfacing.
func TestOwnerAnswersStillReachAHelperUnderClarifiedIntent(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "owner answers still reach a helper"
	state := colony.ColonyState{
		Version: "1.0", Goal: &goal, State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Phase One", Status: colony.PhaseReady}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	const question = "Which surface should own the sentinel feature?"
	const sentinelAnswer = "sentinel owner answer 198-2-04: use the admin surface"
	if _, err := recordDecisionAnswer(question, sentinelAnswer, 1, "worker-handoff"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	output := buildColonyPrimeOutput(false)

	// recordDecisionAnswer also emits a FEEDBACK pheromone signal carrying
	// the same answer text (see emitDecisionFeedback) -- a second, distinct
	// sentence in its own section, not a duplicate of Clarified Intent's
	// content. The exact-once assertion below therefore checks the
	// distinctively-formatted "{question} => {answer}" line Clarified
	// Intent renders (see clarifiedIntentPromptRenderResult /
	// TestColonyPrime_IncludesResolvedClarifiedIntent), which only that
	// section produces.
	clarifiedLine := question + " => " + sentinelAnswer
	if count := strings.Count(output.Context, clarifiedLine); count != 1 {
		t.Fatalf("expected the owner's answer to arrive exactly once under Clarified Intent, found %d occurrence(s) of %q:\n%s", count, clarifiedLine, output.Context)
	}
	if strings.Contains(output.Context, "## Key Decisions") || strings.Contains(output.Context, "## Phase Learnings") {
		t.Fatalf("removed heading resurfaced in output:\n%s", output.Context)
	}

	start, end := sectionBoundsInContext(t, output.Context, "## CLARIFIED INTENT")
	sentinelIdx := strings.Index(output.Context, clarifiedLine)
	if sentinelIdx < start || sentinelIdx >= end {
		t.Fatalf("sentinel landed outside the Clarified Intent block (block=[%d,%d), sentinel=%d):\n%s", start, end, sentinelIdx, output.Context)
	}
}
