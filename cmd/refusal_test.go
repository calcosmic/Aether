package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestEveryRefusalRowNamesANextCommand is the guard on the real,
// checked-in registry -- every row must carry a next command, and the
// enumeration itself must not be empty.
func TestEveryRefusalRowNamesANextCommand(t *testing.T) {
	if problems := refusalRegistryProblems(refusalRegistry); len(problems) > 0 {
		t.Fatalf("refusal register has unnamed next commands or is empty:\n%s", strings.Join(problems, "\n"))
	}
}

// TestRefusalRegisterIsSortedAndUnique proves the real registry is written
// in ascending id order with no duplicate ids, and that looking a row up
// twice returns the same row.
func TestRefusalRegisterIsSortedAndUnique(t *testing.T) {
	if problems := refusalRegistryProblems(refusalRegistry); len(problems) > 0 {
		t.Fatalf("refusal register is not sorted/unique:\n%s", strings.Join(problems, "\n"))
	}
	if len(refusalRegistry) == 0 {
		t.Fatal("refusalRegistry is empty")
	}
	id := refusalRegistry[0].ID
	first, ok1 := refusalForID(id)
	second, ok2 := refusalForID(id)
	if !ok1 || !ok2 {
		t.Fatalf("refusalForID(%q) did not find a row on repeat lookup", id)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("refusalForID(%q) returned different rows across calls:\n%+v\n%+v", id, first, second)
	}
}

// TestRefusalCheckCanFail is the guard on the guard: it feeds
// refusalRegistryProblems an isolated, fabricated slice carrying a row with
// an empty NextCommand, and asserts the check actually reports it -- proof
// the check can fail rather than passing no matter what it is given.
func TestRefusalCheckCanFail(t *testing.T) {
	broken := []refusalRow{
		{ID: "a-fine-row", NextCommand: "aether status"},
		{ID: "z-broken-row", NextCommand: "   "},
	}
	problems := refusalRegistryProblems(broken)
	if len(problems) == 0 {
		t.Fatal("expected refusalRegistryProblems to report the row with an empty next_command, got none")
	}
	found := false
	for _, p := range problems {
		if strings.Contains(p, "z-broken-row") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a problem naming z-broken-row, got: %v", problems)
	}

	if problems := refusalRegistryProblems(nil); len(problems) == 0 {
		t.Fatal("expected refusalRegistryProblems(nil) to report the broken enumeration, got none")
	}
}

// TestRefusalCheckCatchesAnUnsubstitutedPlaceholder is CR-01's own guard on
// the guard: a row whose NextCommand still carries a template placeholder
// like `<command>` is not a real, runnable command -- a copy-pasted
// `aether <command> --help` fails in a shell with an unknown-command error,
// not help text (208-REVIEW.md CR-01). This proves refusalRegistryProblems
// actually catches that shape, on a fabricated slice isolated from the real
// registry, the same discipline TestRefusalCheckCanFail already uses for a
// blank next_command.
func TestRefusalCheckCatchesAnUnsubstitutedPlaceholder(t *testing.T) {
	broken := []refusalRow{
		{ID: "a-fine-row", NextCommand: "aether status"},
		{ID: "z-placeholder-row", NextCommand: "aether <command> --help"},
	}
	problems := refusalRegistryProblems(broken)
	found := false
	for _, p := range problems {
		if strings.Contains(p, "z-placeholder-row") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a problem naming z-placeholder-row's unsubstituted placeholder, got: %v", problems)
	}

	// The real, checked-in registry must never regress to this shape --
	// this is the assertion that actually would have caught CR-01.
	for _, row := range refusalRegistry {
		if strings.ContainsAny(row.NextCommand, "<>") {
			t.Errorf("refusal row %q's next_command %q is not a real, runnable command", row.ID, row.NextCommand)
		}
	}
}

// TestTwoRefusalsSharingANextCommandStaySeparate uses two real registered
// rows that deliberately share the same next command (both freshness
// refusals point at the same rerun command) and proves refuse/renderRefusal
// never merge them into one row.
func TestTwoRefusalsSharingANextCommandStaySeparate(t *testing.T) {
	a := refuse("colonize-finalize-missing-timestamp")
	b := refuse("colonize-finalize-invalid-timestamp")

	if a.NextCommand != b.NextCommand {
		t.Fatalf("test fixture assumption broken: expected a shared next command, got %q and %q", a.NextCommand, b.NextCommand)
	}
	if a.ID == b.ID {
		t.Fatalf("expected different ids, got both %q", a.ID)
	}
	if a.What == b.What {
		t.Fatalf("expected different What text, got both %q", a.What)
	}
	if renderRefusal(a) == renderRefusal(b) {
		t.Fatal("expected renderRefusal to produce two different blocks for two different refusals")
	}
}

// TestRefusalScreenTellsTheReaderToReportIt asserts the rendered block names
// `aether report`, tells the reader not to edit Aether's own program files,
// and uses no untranslated repo-invented word.
func TestRefusalScreenTellsTheReaderToReportIt(t *testing.T) {
	block := renderRefusal(refuse("colonize-finalize-missing-timestamp"))
	if !strings.Contains(block, "aether report") {
		t.Errorf("expected refusal block to name `aether report`, got:\n%s", block)
	}
	if !strings.Contains(strings.ToLower(block), "not") || !strings.Contains(strings.ToLower(block), "edit") {
		t.Errorf("expected refusal block to tell the reader not to edit Aether's own files, got:\n%s", block)
	}
	if violations := untranslatedRepoWords(block); len(violations) > 0 {
		t.Errorf("refusal block used repo-invented word(s) without explaining them: %v\n%s", violations, block)
	}
}

// TestRefusalCarriesItsNextCommandOnBothLanes drives the same refusal id
// through ExitWithError's render path (renderRefusalToExitWriter, factored
// out of the os.Exit call so it is testable) and through outputRefusal's
// visual path, and fails if the backticked next command differs between
// the two.
func TestRefusalCarriesItsNextCommandOnBothLanes(t *testing.T) {
	saveGlobals(t)
	r := refuse("colonize-finalize-missing-timestamp")

	// Both lanes must be compared under the SAME rendering environment --
	// writeVisualOutput translates command names per platform only when
	// shouldRenderVisualOutput(w) is true for that writer/environment, so
	// comparing a translated run against an untranslated one would fail for
	// a reason that has nothing to do with the two lanes disagreeing.
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	var exitBuf bytes.Buffer
	stderr = &exitBuf
	renderRefusalToExitWriter(r)
	exitOutput := exitBuf.String()

	var visualBuf bytes.Buffer
	stderr = &visualBuf
	outputRefusal(r)
	visualOutput := visualBuf.String()

	exitNext := backtickedNextCommandLine(exitOutput)
	visualNext := backtickedNextCommandLine(visualOutput)
	if exitNext == "" || visualNext == "" {
		t.Fatalf("expected both lanes to render a `Next:` line, got exit=%q visual=%q\nexit output:\n%s\nvisual output:\n%s", exitNext, visualNext, exitOutput, visualOutput)
	}
	if exitNext != visualNext {
		t.Errorf("the two lanes named different next commands for the same refusal:\nExitWithError lane: %s\noutputRefusal lane: %s", exitNext, visualNext)
	}
	if !strings.Contains(exitOutput, r.What) || !strings.Contains(visualOutput, r.What) {
		t.Errorf("expected both lanes to carry the same What text %q:\nexit: %s\nvisual: %s", r.What, exitOutput, visualOutput)
	}
}

// backtickedNextCommandLine extracts the "Next: `...`" line a rendered
// refusal block carries, so a test can compare what the two lanes actually
// printed rather than assuming neither lane ever translates the command.
func backtickedNextCommandLine(rendered string) string {
	for _, line := range strings.Split(rendered, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Next:") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// TestColonizeFinalizeRecoversAMissingTimestamp runs the real
// runCodexColonizeFinalize against a completion packet whose
// colonize_manifest.generated_at is the empty string, with a matching
// receipt on disk (written by the real runCodexColonizePlanOnly, never a
// hand-typed fixture), and asserts the call succeeds and the territory
// snapshot is published.
func TestColonizeFinalizeRecoversAMissingTimestamp(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-colonize-recover-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	goal := "Recover a missing survey timestamp"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	result, err := runCodexColonizePlanOnly(root, codexColonizeOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexColonizePlanOnly: %v", err)
	}
	manifest := result["colonize_manifest"].(codexColonizeManifest)
	manifest.GeneratedAt = ""
	dispatches := make([]codexSurveyorDispatch, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		dispatch.Status = "completed"
		dispatch.Summary = "survey complete"
		dispatches = append(dispatches, dispatch)
	}

	finalizeResult, err := runCodexColonizeFinalize(root, codexExternalColonizeCompletion{
		ColonizeManifest: &manifest,
		Dispatches:       dispatches,
	})
	if err != nil {
		t.Fatalf("expected recovery to succeed, got error: %v", err)
	}
	surveyDir, _ := finalizeResult["survey_dir"].(string)
	if surveyDir == "" {
		t.Fatal("expected survey_dir in finalize result")
	}
	if _, err := os.Stat(surveyDir); err != nil {
		t.Fatalf("expected territory snapshot published at %s: %v", surveyDir, err)
	}
	if recovered, _ := finalizeResult["generated_at_recovered"].(bool); !recovered {
		t.Errorf("expected generated_at_recovered=true in finalize result, got: %v", finalizeResult["generated_at_recovered"])
	}
}

// TestColonizeFinalizeRefusalNamesTheWayPast runs the same flow with no
// receipt on disk and asserts the returned error is a refusal whose
// NextCommand is `aether colonize --plan-only --force-resurvey` and whose
// ProtectsWork is false.
func TestColonizeFinalizeRefusalNamesTheWayPast(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-colonize-norecovery-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	goal := "Refuse a missing survey timestamp with no receipt"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	result, err := runCodexColonizePlanOnly(root, codexColonizeOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("runCodexColonizePlanOnly: %v", err)
	}
	manifest := result["colonize_manifest"].(codexColonizeManifest)
	manifest.GeneratedAt = ""
	// Remove the receipt the plan-only run just wrote, so no recovery is
	// possible -- this is the "no way past it except rerunning" case.
	if err := os.Remove(filepath.Join(dataDir, colonizeManifestReceiptPath)); err != nil {
		t.Fatalf("remove receipt: %v", err)
	}
	dispatches := make([]codexSurveyorDispatch, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		dispatch.Status = "completed"
		dispatch.Summary = "survey complete"
		dispatches = append(dispatches, dispatch)
	}

	_, err = runCodexColonizeFinalize(root, codexExternalColonizeCompletion{
		ColonizeManifest: &manifest,
		Dispatches:       dispatches,
	})
	if err == nil {
		t.Fatal("expected colonize-finalize to refuse with no receipt to recover from")
	}
	var r refusal
	if !errors.As(err, &r) {
		t.Fatalf("expected a refusal, got: %v", err)
	}
	if r.NextCommand != "aether colonize --plan-only --force-resurvey" {
		t.Errorf("unexpected next command: %q", r.NextCommand)
	}
	if r.ProtectsWork {
		t.Error("expected ProtectsWork=false: rerunning the survey plan loses nothing")
	}
}
