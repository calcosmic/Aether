package cmd

import (
	"strings"
	"testing"
	"time"

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

// TestTheRecruitInstructionHasOneSource proves there is exactly one place the
// invitation is written, so a second, drifting copy cannot appear in a
// different lane's brief -- the failure this phase spent five plans avoiding
// everywhere else.
func TestTheRecruitInstructionHasOneSource(t *testing.T) {
	callers := productionCallersOf(t, "renderRecruitmentInvitation")
	if len(callers) == 0 {
		t.Fatal("renderRecruitmentInvitation has no production caller: the invitation is written but never reaches a brief")
	}
	if len(callers) > 1 {
		t.Errorf("the recruit invitation is emitted from %d places (%v); it must have exactly one source or the lanes will drift", len(callers), callers)
	}
}
