package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 03 (SYN-201-03/D-01/D-02) -- the verification-boundary
// decision: a pure reconciliation function, a durable attempt-bound record,
// and one read path every consumer must use. See
// cmd/verification_boundary.go for the mechanism this file proves.

// --- Task 1: reconciling a proposal into a validated decision ---

func TestVerificationBoundaryDefaultsToTheCheckStep(t *testing.T) {
	decision := queenApplyVerificationBoundary("", "", colony.Phase{}, colony.ColonyState{})
	if decision.Choice != verificationBoundaryChoiceCheckStep {
		t.Fatalf("Choice = %q, want %q", decision.Choice, verificationBoundaryChoiceCheckStep)
	}
	if decision.Source != verificationBoundarySourceDeterministic {
		t.Fatalf("Source = %q, want %q", decision.Source, verificationBoundarySourceDeterministic)
	}
	if decision.Refused {
		t.Fatalf("expected an empty proposal to not be refused, got %+v", decision)
	}

	t.Run("an explicit check-step proposal is the Queen's own choice, one record only", func(t *testing.T) {
		d := queenApplyVerificationBoundary("check_step", "the phase is low risk", colony.Phase{}, colony.ColonyState{})
		if d.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want %q", d.Choice, verificationBoundaryChoiceCheckStep)
		}
		if d.Source != verificationBoundarySourceQueen {
			t.Fatalf("Source = %q, want %q", d.Source, verificationBoundarySourceQueen)
		}
		if d.Refused {
			t.Fatalf("expected no refusal, got %+v", d)
		}
	})
}

func TestVerificationBoundaryRefusesBuildEndWithoutAReason(t *testing.T) {
	cases := []struct {
		name   string
		reason string
	}{
		{"empty reason", ""},
		{"whitespace-only reason", "   \t  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := queenApplyVerificationBoundary("build_end", tc.reason, colony.Phase{}, colony.ColonyState{})
			if !d.Refused {
				t.Fatalf("expected the proposal to be refused, got %+v", d)
			}
			if strings.TrimSpace(d.RefusedWhy) == "" {
				t.Fatalf("expected a non-empty RefusedWhy, got %+v", d)
			}
			if d.Choice != verificationBoundaryChoiceCheckStep {
				t.Fatalf("Choice = %q, want fallback to %q", d.Choice, verificationBoundaryChoiceCheckStep)
			}
			if d.Source != verificationBoundarySourceDeterministic {
				t.Fatalf("Source = %q, want %q on a refused proposal", d.Source, verificationBoundarySourceDeterministic)
			}
		})
	}

	t.Run("a build-end proposal with a real reason is accepted verbatim", func(t *testing.T) {
		d := queenApplyVerificationBoundary("build_end", "  this phase touches credential handling  ", colony.Phase{}, colony.ColonyState{})
		if d.Refused {
			t.Fatalf("expected no refusal, got %+v", d)
		}
		if d.Choice != verificationBoundaryChoiceBuildEnd {
			t.Fatalf("Choice = %q, want %q", d.Choice, verificationBoundaryChoiceBuildEnd)
		}
		if d.Source != verificationBoundarySourceQueen {
			t.Fatalf("Source = %q, want %q", d.Source, verificationBoundarySourceQueen)
		}
		want := "this phase touches credential handling"
		if d.Reason != want {
			t.Fatalf("Reason = %q, want %q (byte-identical to the input after trimming)", d.Reason, want)
		}
	})

	t.Run("an unrecognised proposal is refused by name and falls back", func(t *testing.T) {
		d := queenApplyVerificationBoundary("sometime-later", "because I said so", colony.Phase{}, colony.ColonyState{})
		if !d.Refused {
			t.Fatalf("expected the proposal to be refused, got %+v", d)
		}
		if !strings.Contains(d.RefusedWhy, "sometime-later") {
			t.Fatalf("RefusedWhy = %q, want it to name the offending value", d.RefusedWhy)
		}
		if d.Choice != verificationBoundaryChoiceCheckStep {
			t.Fatalf("Choice = %q, want fallback to %q", d.Choice, verificationBoundaryChoiceCheckStep)
		}
	})
}

func TestVerificationBoundaryNormalisesProposalExactly(t *testing.T) {
	variants := []string{
		"build_end", " build_end", "build_end ", "BUILD_END", "Build_End", "\tbuild_end\n",
	}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			d := queenApplyVerificationBoundary(variant, "needs a build-end review", colony.Phase{}, colony.ColonyState{})
			if d.Choice != verificationBoundaryChoiceBuildEnd {
				t.Fatalf("proposal %q resolved to Choice=%q, want %q", variant, d.Choice, verificationBoundaryChoiceBuildEnd)
			}
			if d.Refused {
				t.Fatalf("proposal %q was refused: %+v", variant, d)
			}
		})
	}

	t.Run("no fuzzy matching -- a near-miss is refused, not silently resolved", func(t *testing.T) {
		d := queenApplyVerificationBoundary("build-end", "close but not exact", colony.Phase{}, colony.ColonyState{})
		if !d.Refused {
			t.Fatalf("expected a hyphenated near-miss to be refused rather than fuzzily matched, got %+v", d)
		}
	})
}

func TestVerificationBoundarySummaryIsStable(t *testing.T) {
	d := queenApplyVerificationBoundary("build_end", "touches auth", colony.Phase{}, colony.ColonyState{})
	first := d.Summary()
	second := d.Summary()
	if first != second {
		t.Fatalf("two calls to Summary() on the same decision produced different text:\n1: %q\n2: %q", first, second)
	}
	if first == "" {
		t.Fatalf("expected a non-empty summary")
	}

	refused := queenApplyVerificationBoundary("build_end", "", colony.Phase{}, colony.ColonyState{})
	refusedSummary := refused.Summary()
	if !strings.Contains(refusedSummary, "refused") {
		t.Fatalf("Summary() for a refused proposal = %q, want it to say a proposal was refused", refusedSummary)
	}
}
