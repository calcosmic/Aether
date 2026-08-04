package cmd

import (
	"os"
	"strings"
	"testing"
)

// This file guards the documentation-claim corollary from CLAUDE.md's
// "Definition of Done": the learning pipeline (pkg/memory's consolidation
// stack) has been declared "restored" in v1.10, v1.11, v1.13, and v1.23
// without ever being invoked by a lifecycle command. v1.25 Phase 162 finally
// wired real runtime callers and corrected three documents that described the
// pipeline as unwired or misdescribed its phase-end ant list. These tests
// exist so those corrected claims cannot silently rot back the way they did
// across four prior milestones.
//
// Every banned substring below is chosen so it would only appear as part of
// a genuine claim being reintroduced into the markdown files under test --
// none of it should ever need to appear in those files' own prose again, so
// a match here is always a real regression, never a false positive from this
// test's own documentation (which lives here, in Go comments, not in the
// markdown files being asserted against).

// learningDocsUnderTest are read relative to the cmd/ package directory,
// matching the convention already used by TestCLAUDEMDVerificationDepthClaims
// in cmd/claudemd_verification_depth_test.go.
var learningDocsUnderTest = []string{
	"../CLAUDE.md",
	"../AGENTS.md",
	"../.aether/docs/structural-learning-stack.md",
}

// retiredLearningDocClaims are substrings that described the learning
// pipeline as unwired, or misdescribed what phase-end consolidation actually
// executes, before v1.25 Phase 162 corrected them. None of these should ever
// reappear in the three files above.
var retiredLearningDocClaims = []string{
	"Phase 162 wires this",
	"no lifecycle command invokes",
	"do not invoke either one yet",
	"three ants only",
	"nurse → herald → janitor",
	"hive-opt-in",
	"opt-in, twice over",
}

// TestLearningDocsDoNotClaimUnwiredConsolidation fails if any of the three
// documents that describe the learning pipeline reintroduces a retired claim
// -- either that consolidation is unwired, or the pre-existing false
// three-ant phase-end description this phase also fixed.
func TestLearningDocsDoNotClaimUnwiredConsolidation(t *testing.T) {
	for _, path := range learningDocsUnderTest {
		path := path
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				t.Skipf("skipping %s: file not found relative to package directory (unusual working directory or vendored package)", path)
				return
			}
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			content := string(data)

			for _, banned := range retiredLearningDocClaims {
				if strings.Contains(content, banned) {
					t.Errorf("%s still contains retired claim %q -- see .aether/docs/learning-system-authority.md for the corrected text", path, banned)
				}
			}
		})
	}
}

// TestLearningDecisionRecordExists asserts the LEARN-03/LEARN-04 decision
// record exists, is substantive (not a stub), and mentions both pkg/memory
// (the authority decision) and AETHER_HIVE_POLICY (the Hive Brain default
// decision) -- so deleting the record, or gutting it to a placeholder, fails
// this test rather than silently losing the documented decision.
//
// Unlike TestLearningDocsDoNotClaimUnwiredConsolidation, a missing file here
// is always a hard failure, never a skip: this test's entire purpose is to
// catch the record being deleted or renamed, so treating "not found" as a
// pass-through would defeat it.
func TestLearningDecisionRecordExists(t *testing.T) {
	const path = "../.aether/docs/learning-system-authority.md"

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s not found or unreadable (%v) -- the LEARN-03/LEARN-04 decision record must exist", path, err)
	}
	content := string(data)

	const minBytes = 500
	if len(content) < minBytes {
		t.Errorf("%s is only %d bytes; expected at least %d -- looks gutted to a stub", path, len(content), minBytes)
	}
	if !strings.Contains(content, "pkg/memory") {
		t.Errorf("%s does not mention pkg/memory -- the authority decision must name which system is authoritative", path)
	}
	if !strings.Contains(content, "AETHER_HIVE_POLICY") {
		t.Errorf("%s does not mention AETHER_HIVE_POLICY -- the Hive Brain default decision must name the control surface", path)
	}
}
