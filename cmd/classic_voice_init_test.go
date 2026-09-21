package cmd

// Phase "the owner sees Aether's screens" Part D -- the first screen a new
// project ever draws joins the guarded voice corpus.
//
// This corpus already caught one screen that had drifted off its command's
// path (see cmd/classic_voice_status_dashboard_test.go): a green density
// test measuring a renderer nothing calls. The same check applies here.
// codex_visuals.go's renderInitVisual has exactly one caller and it is a
// _test.go file (cmd/lifecycle_card_agreement_test.go) -- `aether init`
// itself renders through renderFrontDoorInitVisual in cmd/init_cmd.go. This
// file measures the renderer the command actually runs, not the one the
// plan's file inventory happened to name.
//
// renderFrontDoorInitVisual is exercised directly (as
// cmd/classic_voice_status_dashboard_test.go exercises renderDashboard
// directly), not through a full `rootCmd.Execute()` of `aether init`. A
// full command run also appends the widely shared lifecycle-closeout card
// (cmd/lifecycle_closeout.go, used by every lifecycle command -- init,
// colonize, build, continue, seal) after this screen's own content, and
// that shared card is not actually part of renderFrontDoorInitVisual
// itself.

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// classicVoiceInitRender calls the real production renderer with a
// realistic, fully populated set of arguments -- a goal, an accepted
// charter, seeded hive wisdom, a stale-territory result, and ranked next
// moves -- so this fixture measures the screen's own prose at the density
// a real first run actually carries, not its thinnest possible shape.
func classicVoiceInitRender(t *testing.T) string {
	t.Helper()
	goal := "Ship the reporting feature end to end"
	charter := colony.Charter{
		Intent:      goal,
		Vision:      "Owners see every screen the program draws, every time",
		Governance:  "Program decides, chat shows",
		Goals:       "Screens reach the owner unchanged",
		TechStack:   "Go, Claude Code, OpenCode",
		KeyRisks:    "A folded reply hides the finished work",
		Constraints: "No second, competing symbol table",
	}
	state := colony.ColonyState{
		Goal:    &goal,
		Charter: &charter,
		AcceptedCharter: &colony.AcceptedCharter{
			EpisodeID:  "ship_1790012787",
			Goal:       goal,
			Provenance: "owner-provided",
			AcceptedAt: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
			Charter:    &charter,
		},
	}
	territory := SurveyFreshnessResult{Freshness: colony.SurveyFreshnessStale}
	proposals := computeInitProposals(".", goal, false)
	return renderFrontDoorInitVisual(state, "Bootstrapped", territory, "/Users/owner/repos/reporting-service/.aether/data", 2, proposals)
}

func init() {
	registerVoiceScreen("init-start", classicVoiceInitRender)
}

// TestInitScreenMeetsTheReferenceDensity holds the real `aether init`
// screen to the same Classic-voice bar every other registered screen meets.
func TestInitScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	rendered := classicVoiceInitRender(t)
	led, total, ratio := voiceDensity(rendered)
	if total == 0 {
		t.Fatalf("the init screen produced no content lines to measure")
	}
	if ratio < reference {
		t.Errorf("the init screen `aether init` draws measures %v (led=%d total=%d), below the reference figure %v.\n%s",
			ratio, led, total, reference, rendered)
	}
}

// TestTheInitCommandRendersTheScreenTheCorpusMeasures guards against the
// exact class of drift documented at the top of this file: the corpus must
// measure a renderer `aether init` genuinely calls.
func TestTheInitCommandRendersTheScreenTheCorpusMeasures(t *testing.T) {
	callers := productionCallersOf(t, "renderFrontDoorInitVisual")
	if len(callers) == 0 {
		t.Fatalf("renderFrontDoorInitVisual has no production caller, so the \"init-start\" corpus entry would measure code no command runs")
	}
}

// classicVoiceCharterRender exercises renderCharterFields (shared by the
// init birth ceremony and the standalone `aether phase`/charter display)
// with a fully populated charter, so the field-list itself is held to the
// same bar independent of which screen embeds it.
func classicVoiceCharterRender(t *testing.T) string {
	t.Helper()
	ch := colony.Charter{
		Intent:      "Ship the reporting feature end to end",
		Vision:      "Owners see every screen the program draws, every time",
		Governance:  "Program decides, chat shows",
		Goals:       "Screens reach the owner unchanged",
		TechStack:   "Go, Claude Code, OpenCode",
		KeyRisks:    "A folded reply hides the finished work",
		Constraints: "No second, competing symbol table",
	}
	return renderCharterFields(ch)
}

func init() {
	registerVoiceScreen("init-charter", classicVoiceCharterRender)
}

func TestInitCharterMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	rendered := classicVoiceCharterRender(t)
	led, total, ratio := voiceDensity(rendered)
	if total == 0 {
		t.Fatalf("the charter block produced no content lines to measure")
	}
	if ratio < reference {
		t.Errorf("the charter block measures %v (led=%d total=%d), below the reference figure %v.\n%s",
			ratio, led, total, reference, rendered)
	}
}

func TestTheCharterFieldsAreCalledByInit(t *testing.T) {
	callers := productionCallersOf(t, "renderCharterFields")
	if len(callers) == 0 {
		t.Fatalf("renderCharterFields has no production caller")
	}
	found := false
	for _, c := range callers {
		if c == "init_cmd.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("renderCharterFields is not called from init_cmd.go; callers: %v", callers)
	}
}
