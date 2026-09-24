package cmd

// Phase 209 plan 06 -- defect register entry 60. 209-TIMING.md's real,
// unattended run 4 measured `/ant-go`'s planning hand-off naming a
// command (`aether plan --preset fast`) that starts no worker and
// produces no dispatch manifest -- so the live assistant reading that
// hand-off had nothing to act on and stopped to ask the owner instead.
// TestGoalReachesBuiltWorkWithNoExtraSteps (cmd/go_default_path_test.go)
// cannot see this: it drives the Go runtime directly with the planning
// stage faked already-accepted and never opens a wrapper file. This test
// reads the real, on-disk wrapper text for all four hand-maintained
// /ant-go sources and fails by name on whichever file's hand-off cannot
// actually reach real planning dispatch.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goPlanningHandoffDispatchCommand is the real runtime command
// /ant-plan's own wrapper (.claude/commands/ant/plan.md, "Choose Planning
// Preset") uses to obtain a dispatch manifest and start a real Scout
// worker.
const goPlanningHandoffDispatchCommand = "aether host plan"

// goPlanningHandoffInertCommand is what the pre-fix /ant-go hand-off
// named instead. It starts no worker and produces no dispatch manifest --
// naming only this gives a live assistant nothing to act on.
const goPlanningHandoffInertCommand = "aether plan --preset fast"

// goPlanningHandoffBigRouteMarker opens the big-route ("planning route")
// hand-off bullet in every /ant-go wrapper source.
const goPlanningHandoffBigRouteMarker = "If the screen says the planning route:"

// goPlanningHandoffBigRouteTerminator closes the big-route hand-off
// bullet -- it is the next sentence common to every wrapper source
// regardless of whether that source uses a numbered list (the YAML
// spec) or a markdown bullet list (the three managed projections).
const goPlanningHandoffBigRouteTerminator = "This command never refuses the job."

func TestGoPlanningHandoffCanActuallyStartPlanning(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	files := []string{
		".aether/commands/go.yaml",
		".claude/commands/ant-go.md",
		".claude/commands/ant/go.md",
		".opencode/commands/ant/go.md",
	}

	for _, rel := range files {
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(root, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", rel, err)
			}
			section, err := goPlanningHandoffBigRouteSection(string(data))
			if err != nil {
				t.Fatalf("%s: %v", rel, err)
			}

			if !goPlanningHandoffReachesRealDispatch(section) {
				t.Fatalf(
					"%s's planning-route hand-off names no way to actually start planning. "+
						"It must either call the real dispatch entry point (%q -- the command "+
						"/ant-plan's own wrapper uses to obtain a dispatch manifest and start a "+
						"real Scout worker) or explicitly delegate to /ant-plan's own flow by name "+
						"as the thing to carry out. Naming only %q gives a live assistant nothing to "+
						"act on, and an unattended session will stop and ask the owner instead "+
						"(see 209-TIMING.md run 4 and defect register entry 60).\n\nSection read:\n%s",
					rel, goPlanningHandoffDispatchCommand, goPlanningHandoffInertCommand, section,
				)
			}

			if !goPlanningHandoffForbidsAskingTheOwner(section) {
				t.Fatalf(
					"%s's planning-route hand-off does not plainly tell the assistant it must not "+
						"stop anywhere in this hand-off to put the choice of how to proceed back to "+
						"the owner -- the single door already settled that.\n\nSection read:\n%s",
					rel, section,
				)
			}
		})
	}
}

// goPlanningHandoffBigRouteSection extracts the "If the screen says the
// planning route:" bullet -- the big-route hand-off -- from raw wrapper
// text. This mirrors how cmd/platform_doc_hygiene_test.go and
// cmd/planning_public_paths_200_test.go read and scope wrapper markdown
// directly from disk rather than inventing a second way to locate
// wrapper sources.
func goPlanningHandoffBigRouteSection(text string) (string, error) {
	start := strings.Index(text, goPlanningHandoffBigRouteMarker)
	if start == -1 {
		return "", fmt.Errorf("no %q bullet found", goPlanningHandoffBigRouteMarker)
	}
	rest := text[start:]
	if end := strings.Index(rest, goPlanningHandoffBigRouteTerminator); end != -1 {
		rest = rest[:end]
	}
	return rest, nil
}

// goPlanningHandoffReachesRealDispatch is true when the hand-off either
// names the real dispatch entry point or explicitly delegates to
// /ant-plan's own flow by name -- the two honest shapes named in
// 209-06-PLAN.md's Task 1. Naming only the inert preset command is
// neither.
func goPlanningHandoffReachesRealDispatch(section string) bool {
	if strings.Contains(section, goPlanningHandoffDispatchCommand) {
		return true
	}
	return goPlanningHandoffDelegatesToPlanFlowByName(section)
}

// goPlanningHandoffDelegatesToPlanFlowByName is true when the hand-off
// names /ant-plan explicitly and tells the assistant to carry out its
// own flow (rather than merely mentioning /ant-plan exists, the way the
// pre-fix text mentions /ant-discuss and /ant-spec as still available).
func goPlanningHandoffDelegatesToPlanFlowByName(section string) bool {
	if !strings.Contains(section, "/ant-plan") {
		return false
	}
	lowered := strings.ToLower(section)
	if !strings.Contains(lowered, "flow") {
		return false
	}
	for _, verb := range []string{"carry out", "follow", "starting at"} {
		if strings.Contains(lowered, verb) {
			return true
		}
	}
	return false
}

// goPlanningHandoffForbidsAskingTheOwner is true when the hand-off
// plainly says the assistant must not stop to hand the choice of how to
// proceed back to the owner. The pre-fix text only forbids stopping to
// "clarify intent, draft a specification by hand, or approve one" -- a
// narrower prohibition that does not cover the real, measured failure
// (stopping to ask *how to dispatch*), so this check requires the
// broader "back to the owner" phrasing, not merely the word "ask".
func goPlanningHandoffForbidsAskingTheOwner(section string) bool {
	lowered := strings.ToLower(section)
	stopsAsking := strings.Contains(lowered, "never stop") ||
		strings.Contains(lowered, "must not stop") ||
		strings.Contains(lowered, "does not stop")
	return stopsAsking && strings.Contains(lowered, "back to the owner")
}
