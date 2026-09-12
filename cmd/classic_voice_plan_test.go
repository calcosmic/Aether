package cmd

// Phase "Classic Visual Voice" plan 06 -- the planning screens (the primary
// path in cmd/planning_visuals.go, wave 3) carry their own symbols.
//
// These seven cards are the busiest set in the program and the last flat
// ones: the planning iteration card, the planning decision card, the plan
// candidate review, the preset choice, the planning stage card, the planning
// stop card and the plan-accepted card. This file registers each into the
// shared voice-screen corpus (plan 01) and locks the facts each fixture
// supplies as unchanged by the restyle.
//
// Task 2 (cmd/codex_visuals.go's legacy/repair plan-screen branches) adds its
// own registrations to this same file, in its own section below.

import (
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Fixtures -- one per registered card, each carrying at least one figure, one
// identifier, and (where the card has one) an evidence-that-would-change
// sentence, so TestPlanningCardsKeepEveryFigureAndIdentifier has something
// concrete to assert against.
// ---------------------------------------------------------------------------

func classicVoicePlanDecisionFixture() planningDecisionCard {
	return planningDecisionCard{
		ID: "decision-card-01", DecisionID: "DECISION-01", Decision: "Choose the storage boundary",
		WhyNow:              "The choice changes recovery behavior",
		QueenRecommendation: "Keep authority local because the approved recovery contract requires offline operation",
		Evidence:            []colony.PlanningEvidenceRef{{ID: "EVIDENCE-02"}},
		Choices: []planningDecisionChoice{
			{ID: "CHOICE-01", Label: "Local durable state", Consequence: "No network dependency"},
			{ID: "CHOICE-02", Label: "Remote durable state", Consequence: "Survives machine loss"},
		},
		AffectedSemanticIDs: []string{"REQ-01", "REC-01"}, PriorAnswer: "Remote state",
		Revalidation: "The recovery requirement changed", PlanningResumes: "Scout pass 2",
	}
}

func classicVoicePlanStageManifestFixture() planningStageManifest {
	return planningStageManifest{
		SchemaVersion: "1", ID: "STAGE-01", Pass: 2,
		ExpectedCaste:    planningStageCasteScout,
		EvidenceFrontier: []planningStageEvidenceBinding{{ID: "EVIDENCE-03", ContentHash: strings.Repeat("d", 64)}},
		WeakestGap:       &colony.PlanningGap{ID: "GAP-02", Description: "Dependency ownership is unverified"},
	}
}

func classicVoicePlanStopDecisionFixture() (colony.PlanningStopDecision, []colony.PlanningGap) {
	decision := colony.PlanningStopDecision{
		Reason: colony.PlanningStopDiminishingReturns, Rationale: "One more pass can resolve the gap",
		EvidenceThatWouldChange: "A verified dependency contract",
	}
	gaps := []colony.PlanningGap{
		{ID: "GAP-03", Dimension: colony.PlanningDimensionRisks, Materiality: colony.PlanningGapNonMaterial,
			Description: "Dependency ownership is unverified", EvidenceThatWouldChange: "A verified dependency contract"},
	}
	return decision, gaps
}

func classicVoicePlanAcceptanceFixture() (colony.PlanCandidate, colony.PlanRevision, colony.PlanAcceptanceReceipt) {
	candidate := colony.PlanCandidate{
		ID: "CANDIDATE-02", ContentHash: strings.Repeat("e", 64),
		Timeline: colony.PlanningTimelineBinding{CardIDs: []string{"ITERATION-01", "ITERATION-02"}},
	}
	revision := colony.PlanRevision{ID: "PLAN-REV-03", Phases: []colony.Phase{{ID: 3, Name: "Render planning truth"}}}
	receipt := colony.PlanAcceptanceReceipt{
		CandidateID: candidate.ID, CandidateContentHash: candidate.ContentHash,
		SpecificationRevisionID: "SPEC-REV-02", TimelineID: "TIMELINE-02", TimelineDigest: strings.Repeat("f", 64),
	}
	return candidate, revision, receipt
}

func classicVoicePlanPresetSelectionFixture() planningPresetSelection {
	return planningPresetSelection{PresetRequired: true, Options: append([]planningPresetPolicy(nil), planningPresetPolicies...)}
}

// ---------------------------------------------------------------------------
// Corpus registration -- one init per screen, in this screen's own test file.
// ---------------------------------------------------------------------------

func init() {
	registerVoiceScreen("plan-iteration", func(t *testing.T) string {
		t.Helper()
		return renderPlanningIterationVisual(planningVisualIterationFixture(colony.PlanningStopContinue), planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-decision", func(t *testing.T) string {
		t.Helper()
		return renderPlanningDecisionVisual(classicVoicePlanDecisionFixture(), planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-candidate-review", func(t *testing.T) string {
		t.Helper()
		return renderPlanningCandidateVisual(planningVisualCandidateFixture(), planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-preset", func(t *testing.T) string {
		t.Helper()
		return renderPlanningPresetVisual("Ship the billing rewrite", "SPEC-REV-01", classicVoicePlanPresetSelectionFixture(), planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-stage", func(t *testing.T) string {
		t.Helper()
		return renderPlanningStageVisual(classicVoicePlanStageManifestFixture(), "started", "Scout-12", false, 3, 2, planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-stop", func(t *testing.T) string {
		t.Helper()
		decision, gaps := classicVoicePlanStopDecisionFixture()
		return renderPlanningStopVisual(decision, gaps, 72, 90, "balanced", planningVisualOptions{Width: 96})
	})
	registerVoiceScreen("plan-acceptance", func(t *testing.T) string {
		t.Helper()
		candidate, revision, receipt := classicVoicePlanAcceptanceFixture()
		return renderPlanningAcceptanceVisual(candidate, revision, receipt, false, planningVisualOptions{Width: 96})
	})
}

// ---------------------------------------------------------------------------
// Task 1 -- density, fact preservation, and narrow-width symbol safety for
// the seven primary planning cards.
// ---------------------------------------------------------------------------

// classicVoicePlanCardNames is the seven Task-1 registrations, named once so
// every test below iterates the same fixed set rather than re-deriving it
// from the full (shared, growing) corpus.
var classicVoicePlanCardNames = []string{
	"plan-iteration", "plan-decision", "plan-candidate-review", "plan-preset",
	"plan-stage", "plan-stop", "plan-acceptance",
}

func classicVoicePlanCardRender(t *testing.T, name string) string {
	t.Helper()
	for _, screen := range voiceScreenRegistry {
		if screen.Name == name {
			return screen.Render(t)
		}
	}
	t.Fatalf("screen %q is not registered in the voice screen corpus", name)
	return ""
}

// TestPlanScreenMeetsTheReferenceDensity asserts each of the seven primary
// planning cards measures at or above the figure derived from the reference
// commits, as its own subtest. Task 2 (below, added by a later commit in this
// same plan) extends the set this test iterates to ten.
func TestPlanScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	for _, name := range classicVoicePlanCardNames {
		name := name
		t.Run(name, func(t *testing.T) {
			rendered := classicVoicePlanCardRender(t, name)
			led, total, ratio := voiceDensity(rendered)
			if total == 0 {
				t.Fatalf("%s produced no content lines to measure:\n%s", name, rendered)
			}
			if ratio < reference {
				t.Errorf("%s measures %v (led=%d total=%d), below the reference figure %v:\n%s",
					name, ratio, led, total, reference, rendered)
			}
		})
	}
}

// TestPlanningCardsKeepEveryFigureAndIdentifier asserts every numeric figure,
// identifier and evidence sentence each fixture supplies still appears in its
// card's rendering, unchanged -- the restyle is presentation-only.
func TestPlanningCardsKeepEveryFigureAndIdentifier(t *testing.T) {
	cases := map[string][]string{
		"plan-iteration": {
			"EVIDENCE-01", "A verified dependency contract",
			"70", "50", "GAP-01",
		},
		"plan-decision": {
			"DECISION-01", "EVIDENCE-02", "CHOICE-01", "CHOICE-02",
			"REQ-01", "REC-01", "Remote state", "Scout pass 2",
		},
		"plan-candidate-review": {
			"CANDIDATE-01", "SPEC-REV-01", "PLAN-REV-01",
			"EVIDENCE-01", "A verified dependency contract", "82", "80",
		},
		"plan-preset": {
			"Ship the billing rewrite", "SPEC-REV-01", "Fast", "Balanced", "Deep", "Exhaustive",
			"80", "90", "95", "99",
		},
		"plan-stage": {
			"Dependency ownership is unverified", "EVIDENCE-03", "Scout-12",
		},
		"plan-stop": {
			"GAP-03", "A verified dependency contract", "72", "90", "balanced",
		},
		"plan-acceptance": {
			"CANDIDATE-02", "PLAN-REV-03", "SPEC-REV-02", "ffffffffffff",
		},
	}
	for name, wants := range cases {
		name, wants := name, wants
		t.Run(name, func(t *testing.T) {
			rendered := classicVoicePlanCardRender(t, name)
			for _, want := range wants {
				if !strings.Contains(rendered, want) {
					t.Errorf("%s lost fixture value %q:\n%s", name, want, rendered)
				}
			}
		})
	}
}

// classicVoiceLineStartsInsideASymbol reports whether line, once leading
// whitespace is stripped, begins with a rune that only ever makes sense
// attached to a preceding rune (a variation selector, the zero-width joiner,
// or a Unicode combining mark) -- exactly what a mid-symbol wrap would
// produce, since voiceLine always emits a complete glyph before any other
// content.
func classicVoiceLineStartsInsideASymbol(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(trimmed)
	return r == 0xFE0F || r == 0xFE0E || r == 0x200D || unicode.Is(unicode.Mn, r)
}

// TestPlanningCardsFitTheirBandWithSymbols renders each of the seven primary
// planning cards at the narrowest supported width and asserts every line is
// valid UTF-8, within the band (or explicitly exempted as an indivisible
// identifier), and never begins mid-symbol -- the width machinery must still
// hold once every card is glyph-led.
func TestPlanningCardsFitTheirBandWithSymbols(t *testing.T) {
	const narrowest = 24
	for _, name := range classicVoicePlanCardNames {
		name := name
		t.Run(name, func(t *testing.T) {
			rendered := classicVoicePlanCardRender(t, name)
			var narrow string
			switch name {
			case "plan-iteration":
				narrow = renderPlanningIterationVisual(planningVisualIterationFixture(colony.PlanningStopContinue), planningVisualOptions{Width: narrowest})
			case "plan-decision":
				narrow = renderPlanningDecisionVisual(classicVoicePlanDecisionFixture(), planningVisualOptions{Width: narrowest})
			case "plan-candidate-review":
				narrow = renderPlanningCandidateVisual(planningVisualCandidateFixture(), planningVisualOptions{Width: narrowest})
			case "plan-preset":
				narrow = renderPlanningPresetVisual("Ship the billing rewrite", "SPEC-REV-01", classicVoicePlanPresetSelectionFixture(), planningVisualOptions{Width: narrowest})
			case "plan-stage":
				narrow = renderPlanningStageVisual(classicVoicePlanStageManifestFixture(), "started", "Scout-12", false, 3, 2, planningVisualOptions{Width: narrowest})
			case "plan-stop":
				decision, gaps := classicVoicePlanStopDecisionFixture()
				narrow = renderPlanningStopVisual(decision, gaps, 72, 90, "balanced", planningVisualOptions{Width: narrowest})
			case "plan-acceptance":
				candidate, revision, receipt := classicVoicePlanAcceptanceFixture()
				narrow = renderPlanningAcceptanceVisual(candidate, revision, receipt, false, planningVisualOptions{Width: narrowest})
			default:
				t.Fatalf("no narrow-width renderer wired for %q", name)
			}
			_ = rendered
			for number, line := range strings.Split(strings.TrimSuffix(narrow, "\n"), "\n") {
				if !utf8.ValidString(line) {
					t.Errorf("%s line %d is not valid UTF-8: %q", name, number+1, line)
				}
				if got := planningVisibleWidth(line); got > narrowest && !planningVisualLineMayOverflow(line) {
					t.Errorf("%s line %d width = %d, want <= %d: %q", name, number+1, got, narrowest, line)
				}
				if classicVoiceLineStartsInsideASymbol(line) {
					t.Errorf("%s line %d begins mid-symbol (a wrap cut a glyph in two): %q", name, number+1, line)
				}
			}
		})
	}
}
