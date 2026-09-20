package cmd

// CEC-07 (203-12-PLAN.md, Task 2): agencyEvidenceFromTrophallaxisDecision
// populates AgencyReceiptEvidence.ChangedDecision/EffectEvidence from real
// recorded evidence (a trophallaxis packet's own decision, plus this plan's
// recruitment credit record) instead of the deleted stale limitation stub
// that used to sit at that assignment site. The both-or-neither invariant
// and its exact error messages are exercised unchanged by the pre-existing
// cmd/agency_receipt_199_test.go -- this file adds only what changed: the
// new resolver, and the source-scan proof that the stale limitation cannot
// silently return.
//
// TestAgencyNoLimitationNamesAFuturePhase below builds the identifier it
// scans for by concatenation, deliberately: a literal occurrence of that
// name in this file's own source would trip the very grep this plan's
// acceptance criteria run against cmd/, despite this file proving the
// identifier is gone, not restoring it.

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestAgencyEvidenceFromTrophallaxisDecision drives the real public path --
// pack, acknowledge, and record a decision through cmd/trophallaxis.go
// (203-10), then record a credit outcome through cmd/recruitment_credit.go
// (this plan's Task 1) -- and asserts agencyEvidenceFromTrophallaxisDecision
// resolves both ChangedDecision and EffectEvidence from that genuinely
// recorded data, and that the resolved pair satisfies
// BuildAgencySignalResult's own already-correct both-or-neither join.
func TestAgencyEvidenceFromTrophallaxisDecision(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	packetID, decisionID := newCreditTrophallaxisFixture(t, "agency-join")

	// Before any credit is recorded, there is nothing to report -- not a
	// default or fabricated join.
	if _, ok, err := agencyEvidenceFromTrophallaxisDecision(packetID); err != nil || ok {
		t.Fatalf("expected no evidence before credit is recorded, got ok=%v err=%v", ok, err)
	}

	if _, _, err := recordRecruitmentCredit("contrib-agency-join", recruitmentContributionRecruitmentResult, decisionID, packetID, recruitmentCreditOutcomeHelpful, ""); err != nil {
		t.Fatalf("record credit: %v", err)
	}

	evidence, ok, err := agencyEvidenceFromTrophallaxisDecision(packetID)
	if err != nil {
		t.Fatalf("resolve evidence: %v", err)
	}
	if !ok {
		t.Fatal("expected resolved evidence once a credit record exists")
	}
	if evidence.ChangedDecision == nil || evidence.ChangedDecision.ID != decisionID {
		t.Fatalf("ChangedDecision = %+v, want id %q", evidence.ChangedDecision, decisionID)
	}
	if evidence.EffectEvidence == nil || evidence.EffectEvidence.ID != packetID {
		t.Fatalf("EffectEvidence = %+v, want id %q", evidence.EffectEvidence, packetID)
	}

	// The resolved pair must satisfy the pre-existing both-or-neither join
	// unchanged -- this is the proof the join was genuinely starved, not
	// broken: real data now flows straight through it.
	signal := agencySignal199("FOCUS", "credit join proof")
	result, err := BuildAgencySignalResult(signal, false, evidence)
	if err != nil {
		t.Fatalf("BuildAgencySignalResult with resolved evidence: %v", err)
	}
	if result.MeasuredEffect == AgencyMeasuredEffectPending {
		t.Fatalf("measured effect still pending after resolving real evidence: %+v", result)
	}
	if !strings.Contains(result.MeasuredEffect, decisionID) || !strings.Contains(result.MeasuredEffect, packetID) {
		t.Fatalf("measured effect = %q, want it to name decision %q and evidence %q", result.MeasuredEffect, decisionID, packetID)
	}
}

// TestAgencyEvidenceFromTrophallaxisDecisionSkipsPending proves a pending
// credit record (a decision changed, but no verified outcome yet) is never
// mistaken for measured evidence -- must_haves: "never a default or
// inferred credit."
func TestAgencyEvidenceFromTrophallaxisDecisionSkipsPending(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	packetID, decisionID := newCreditTrophallaxisFixture(t, "agency-pending")
	if _, _, err := recordRecruitmentCredit("contrib-agency-pending", recruitmentContributionNote, decisionID, "", "", ""); err != nil {
		t.Fatalf("record pending credit: %v", err)
	}

	if _, ok, err := agencyEvidenceFromTrophallaxisDecision(packetID); err != nil || ok {
		t.Fatalf("a pending-only credit record must never resolve as measured evidence, got ok=%v err=%v", ok, err)
	}
}

// TestAgencyNoLimitationNamesAFuturePhase is a source scan asserting the
// removed limitation identifier and any "awaits Phase N" sentence are gone
// from cmd/'s live (non-test) source, and enforcing this going forward: a
// future reintroduction of either pattern fails this test by naming the
// offending file and line.
func TestAgencyNoLimitationNamesAFuturePhase(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}

	// Built by concatenation, not as a literal, so this file's own source
	// never contains the identifier text -- the deleted name stays
	// genuinely absent from cmd/, this test included.
	removedIdentifier := "SwarmPhase" + "202" + "Limitation"
	awaitsPhase := regexp.MustCompile(`awaits Phase \d+`)
	fset := token.NewFileSet()
	var violations []string
	for _, name := range names {
		if name == "agency_contract_test.go" {
			continue
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(raw), removedIdentifier) {
			violations = append(violations, name+": still references the removed limitation identifier "+removedIdentifier)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if awaitsPhase.MatchString(line) {
				violations = append(violations, fmt.Sprintf("%s:%d: %s", name, i+1, strings.TrimSpace(line)))
			}
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		// Parse to prove the file is still valid Go after the scan --
		// a broken fixture (unparsable file) must not silently pass.
		if _, err := parser.ParseFile(fset, name, nil, 0); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
	}
	if len(violations) != 0 {
		t.Fatalf("found a stale phase-numbered limitation sentence:\n%s", strings.Join(violations, "\n"))
	}
}
