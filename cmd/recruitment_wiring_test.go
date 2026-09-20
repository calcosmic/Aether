package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// TestEveryDispatchedWorkerIsToldHowToAskForHelp is the wiring proof for the
// whole of Phase 203.
//
// Five plans built `aether recruit` and nothing invoked it. The orphan check
// reported that correctly for four waves. It was NOT closed by widening an
// allowlist, and it must not be closed by documenting the command somewhere
// nobody reads: .aether/workers.md is named by no agent definition and loaded
// by no runtime code, which is the "a doc mention is not an execution" case
// the reachability rules already exclude playbooks for.
//
// This asserts the real thing instead: the brief the runtime composes for a
// dispatched worker -- the text that lands in its prompt -- tells it the
// command and how a refusal behaves. Asserting on the composed brief rather
// than on the source line means a future refactor that drops the section
// fails here, whatever the line looks like.
//
// This covers exactly ONE lane: composeBuildManifestBrief, the Claude/OpenCode
// wrapper's plan-only build lane. 203-REVIEW.md's CR-05 found that lane was
// not the whole picture -- see TestNativeBuildLaneWorkerIsToldHowToAskForHelp
// and TestContinueLaneWorkersAreToldHowToAskForHelp below for the two lanes
// this one cannot see.
func TestEveryDispatchedWorkerIsToldHowToAskForHelp(t *testing.T) {
	saveGlobals(t)
	_, root := setupTestStore(t)
	t.Setenv("AETHER_ROOT", root)

	phase := colony.Phase{ID: 1, Name: "Wiring proof"}
	for _, caste := range []string{"builder", "watcher", "scout"} {
		t.Run(caste, func(t *testing.T) {
			brief := composeBuildManifestBrief(root, phase,
				codexBuildDispatch{Name: "Helper-1", Caste: caste, Task: "do the work"},
				time.Date(2026, time.September, 13, 12, 0, 0, 0, time.UTC), true)

			if !strings.Contains(brief, "aether recruit") {
				t.Fatalf("a dispatched %s worker is never told the command that lets it ask for help; the mechanism is unreachable from its own prompt", caste)
			}
			// A refusal being a NORMAL outcome is the half workers get wrong:
			// without it they retry, or treat a denial as a failed task.
			if !strings.Contains(brief, "refusal") {
				t.Errorf("the %s brief names the command but never says a refusal is a normal answer, so a refused worker will read a denial as an error", caste)
			}
		})
	}
}

// TestNativeBuildLaneWorkerIsToldHowToAskForHelp is 203-REVIEW.md CR-05's fix
// for gap (a): composeBuildManifestBrief above covers only the Claude/OpenCode
// wrapper's plan-only build lane (attachBuildDispatchContext). Autopilot
// (`aether run`) and a direct `aether build <phase>` dispatch every worker
// through executeCodexBuildDispatches -> buildCodexWorkerDispatches instead,
// which never went through composeBuildManifestBrief and so never carried the
// invitation -- confirmed absent before this fix (see the fix's commit for the
// red-test evidence). This calls the exact function that builds the
// codex.WorkerDispatch the runtime hands to the invoker on that lane, so a
// regression here means a real dispatched worker loses the instruction.
func TestNativeBuildLaneWorkerIsToldHowToAskForHelp(t *testing.T) {
	saveGlobals(t)
	_, root := setupTestStore(t)
	t.Setenv("AETHER_ROOT", root)

	phase := colony.Phase{ID: 1, Name: "Wiring proof"}
	dispatches := []codexBuildDispatch{{Name: "Helper-1", Caste: "builder", Task: "do the work"}}

	workerDispatches, err := buildCodexWorkerDispatches(root, phase, dispatches, time.Now().UTC(), &codex.FakeInvoker{}, "", time.Minute, nil)
	if err != nil {
		t.Fatalf("buildCodexWorkerDispatches: %v", err)
	}
	if len(workerDispatches) != 1 {
		t.Fatalf("got %d worker dispatches, want 1", len(workerDispatches))
	}

	brief := workerDispatches[0].TaskBrief
	if !strings.Contains(brief, "aether recruit") {
		t.Fatal("a worker dispatched on the native/direct build lane (executeCodexBuildDispatches, which autopilot's `aether run` and a direct `aether build <phase>` both use) is never told the command that lets it ask for help")
	}
	if !strings.Contains(brief, "refusal") {
		t.Error("the native/direct build lane's brief names the command but never says a refusal is a normal answer")
	}
}

// TestContinueLaneWorkersAreToldHowToAskForHelp is 203-REVIEW.md CR-05/WR-08's
// fix for gap (c): every worker dispatched during `aether continue` (Watcher,
// Gatekeeper, Auditor, Probe) was composed by one of two brief functions that
// never called renderRecruitmentInvitation at all, so none of them could ever
// have been told about `aether recruit`.
//
// Watcher (renderCodexContinueWatcherBrief) and Probe -- a genuine review
// caste dispatched via renderCodexContinueReviewBrief -- both carry a Bash
// tool and must receive the invitation. Gatekeeper and Auditor deliberately
// must NOT: codexContinueReviewSpecs' own doc comment and
// TestReviewSpecsDoNotInstructBashlessCastes
// (cmd/review_findings_persist_test.go) already establish that these two
// castes have no Bash tool by explicit design and must never be instructed to
// run a CLI command -- doing so here would repeat exactly the field failure
// that rule exists to prevent, just with a different command.
func TestContinueLaneWorkersAreToldHowToAskForHelp(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	phase := colony.Phase{ID: 1, Name: "Wiring proof"}

	t.Run("watcher", func(t *testing.T) {
		brief := renderCodexContinueWatcherBrief(root, phase, codexContinueManifest{}, nil, codexClaimVerification{}, codexWatcherVerification{}, time.Minute)
		if !strings.Contains(brief, "aether recruit") {
			t.Fatal("the continue lane's Watcher -- the most expensive single worker in the flow -- is never told the command that lets it ask for help")
		}
		if !strings.Contains(brief, "refusal") {
			t.Error("the continue lane's Watcher brief names the command but never says a refusal is a normal answer")
		}
	})

	t.Run("probe_review_caste", func(t *testing.T) {
		spec, ok := continueReviewSpecForCaste("probe")
		if !ok {
			t.Fatal("no continue review spec for probe")
		}
		brief := renderCodexContinueReviewBrief(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)
		if !strings.Contains(brief, "aether recruit") {
			t.Fatal("the continue lane's Probe (a review caste with a Bash tool) is never told the command that lets it ask for help")
		}
		if !strings.Contains(brief, "refusal") {
			t.Error("the continue lane's Probe brief names the command but never says a refusal is a normal answer")
		}
	})

	t.Run("bashless_castes_excluded", func(t *testing.T) {
		for _, caste := range []string{"gatekeeper", "auditor"} {
			spec, ok := continueReviewSpecForCaste(caste)
			if !ok {
				t.Fatalf("no continue review spec for %s", caste)
			}
			brief := renderCodexContinueReviewBrief(root, phase, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, spec)
			if strings.Contains(brief, "aether recruit") {
				t.Errorf("%s continue brief instructs a CLI command this Bash-less caste cannot run", caste)
			}
		}
	})
}

// TestCodexAgentDefinitionsCarryTheRecruitInvitation is 203-REVIEW.md CR-05's
// fix for gap (b): the persistent Codex agent definitions -- loaded as
// developer_instructions, playing the same role the Claude/OpenCode .md agent
// definitions play as a dispatched worker's own system instructions -- never
// mentioned `aether recruit` on any of the 27 .codex/agents/*.toml files,
// while their Claude and OpenCode counterparts for these three castes already
// did.
func TestCodexAgentDefinitionsCarryTheRecruitInvitation(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	for _, name := range []string{"aether-builder", "aether-watcher", "aether-scout"} {
		path := filepath.Join(repoRoot, ".codex", "agents", name+".toml")
		instructions, err := codex.LoadAgentInstructions(path)
		if err != nil {
			t.Fatalf("load %s: %v", path, err)
		}
		if !strings.Contains(instructions, "aether recruit") {
			t.Errorf("%s never mentions `aether recruit`, though its Claude/OpenCode counterpart does", path)
		}
		if !strings.Contains(instructions, "refusal") {
			t.Errorf("%s names the command but never says a refusal is a normal answer", path)
		}
	}
}

// TestTheRecruitInstructionHasOneSource proves the recruit invitation has
// exactly one TEXT source (renderRecruitmentInvitation) and enumerates, by
// name, every function that is allowed -- and required -- to call it. Each of
// the four lanes composes its worker's brief independently (this codebase's
// established pattern: build's wrapper and native lanes, and continue's
// review and watcher lanes, each carry their own steering/handoff channels
// rather than sharing one composer), so a single "exactly one caller" count
// cannot express this any more -- a lane silently losing its call, or a new
// lane copying the text instead of calling the shared function, must fail by
// name instead of just by a raw count changing.
func TestTheRecruitInstructionHasOneSource(t *testing.T) {
	want := map[string]bool{
		"composeBuildManifestBrief":       true, // build: Claude/OpenCode plan-only wrapper lane
		"buildCodexWorkerDispatches":      true, // build: native/direct lane (autopilot `aether run`)
		"renderCodexContinueReviewBrief":  true, // continue: queen-selected/forced review castes
		"renderCodexContinueWatcherBrief": true, // continue: the Watcher's own verification pass
	}

	got := productionCallerFunctionsOf(t, "renderRecruitmentInvitation")
	if len(got) == 0 {
		t.Fatal("renderRecruitmentInvitation has no production caller: the invitation is written but never reaches a brief")
	}
	for name := range want {
		if !got[name] {
			t.Errorf("%s no longer calls renderRecruitmentInvitation -- that lane's dispatched workers would stop being told they can ask for help", name)
		}
	}
	for name := range got {
		if !want[name] {
			t.Errorf("unexpected new caller %s of renderRecruitmentInvitation -- if this is a genuine new lane, add it to this test's enumerated set by name; if it duplicates the text some other way, call the shared function instead of writing a second copy", name)
		}
	}
}

// productionCallerFunctionsOf returns the set of top-level function (or
// var-declared, e.g. cobra command) names, across every non-test .go file
// under cmd/, whose body contains at least one call to name.
//
// Unlike productionCallersOf (cmd/classic_voice_status_dashboard_test.go),
// which is file-level, this is function-level: it is what lets a test
// enumerate the exact set of allowed callers by name rather than merely
// counting file hits -- two call sites landing in the same file (as
// composeBuildManifestBrief and buildCodexWorkerDispatches both do, in
// cmd/codex_build.go) must not collapse into what looks like "one caller".
func productionCallerFunctionsOf(t *testing.T, name string) map[string]bool {
	t.Helper()
	dir := "../cmd"
	entries, err := os.ReadDir(dir)
	if err != nil {
		dir = "."
		entries, err = os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read cmd dir: %v", err)
		}
	}

	callers := map[string]bool{}
	fset := token.NewFileSet()
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		path := n
		if _, statErr := os.Stat(path); statErr != nil {
			path = filepath.Join(dir, n)
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			continue
		}
		for _, decl := range file.Decls {
			declName, body := declNameAndBody(decl)
			if body == nil || declName == "" {
				continue
			}
			found := false
			ast.Inspect(body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if id, ok := call.Fun.(*ast.Ident); ok && id.Name == name {
					found = true
				}
				return true
			})
			if found {
				callers[declName] = true
			}
		}
	}
	return callers
}
