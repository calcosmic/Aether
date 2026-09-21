package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// scoutMinimalEvidenceEntryJSON is the smallest new_evidence entry the Scout's
// own brief tells it to send (renderScoutNewEvidenceRule): no fingerprint, no
// id, no locator, no timestamp, no scope. It is literal JSON on purpose. A
// research helper produces text, not Go structs, and a struct-built fixture
// silently carries zero-valued fields a real helper never sends.
const scoutMinimalEvidenceEntryJSON = `{"reference":{"kind":"research","origin":"https://docs.example.test/webhooks/signing","applicable_dimensions":["risks","dependencies"]},"summary":"The payment provider documents that webhook deliveries can be replayed unless the receiver records each delivery id."}`

// TestScoutMinimalWireEvidenceIsAcceptedEndToEnd drives the real pass-2 Scout
// stage with the literal JSON above in place of new_evidence. This is the
// exact situation that dead-ended a downstream plan on 2026-09-21.
func TestScoutMinimalWireEvidenceIsAcceptedEndToEnd(t *testing.T) {
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil {
		t.Fatal("first Route pass did not authorize the next Scout")
	}
	scoutManifest := first.ScoutDispatch.Manifest

	gap := colony.PlanningGap{
		SchemaVersion: colony.PlanningSchemaVersion, ID: "gap-wire-evidence", ContentHash: planningStageTestHash("7"),
		Dimension: colony.PlanningDimensionRisks, Materiality: colony.PlanningGapNonMaterial, Severity: 15,
		Description:             "Replay protection ordering still needs confirming against the provider documentation.",
		EvidenceThatWouldChange: "The provider's own statement on delivery replays.",
	}
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID: "wire-evidence-finding", Summary: "Provider documentation describes replayable deliveries.",
			Unknown: true, UnknownReason: "the evidence address is derived by the program, not the Scout",
		}},
		UnresolvedGaps: []colony.PlanningGap{gap},
	}

	// Marshal the envelope, then splice the literal entry in as raw JSON so
	// nothing about the entry passes through a Go struct on the way in.
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(planningScoutStageTestBytes(t, scoutResult), &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["new_evidence"] = json.RawMessage("[" + scoutMinimalEvidenceEntryJSON + "]")
	wire, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}

	coordinated, err := coordinatePlanningScoutStage(root, scoutManifest, wire)
	if err != nil {
		t.Fatalf("minimal wire-shaped Scout evidence = %v, want acceptance", err)
	}
	if got := len(coordinated.Scout.Result.NewEvidence); got != 1 {
		t.Fatalf("catalogued %d evidence records, want 1", got)
	}
	ref := coordinated.Scout.Result.NewEvidence[0].Reference
	if ref.ContentHash == "" || !strings.HasSuffix(ref.ID, "-"+ref.ContentHash[:12]) || ref.ExcerptDigest == "" {
		t.Fatalf("evidence reference = %+v, want program-derived address and digest", ref)
	}
	if !ref.Fresh || !ref.Admissible {
		t.Fatalf("evidence reference = %+v, want fresh and admissible", ref)
	}
	if coordinated.RouteDispatch == nil {
		t.Fatal("accepted Scout evidence did not authorize the next planning pass")
	}
}

// TestScoutBriefNamesTheEvidenceShapeItAccepts ties the instruction to the
// decoder: every field name the brief tells the Scout to send must be a field
// the literal entry above uses, and the brief must tell it not to fingerprint.
func TestScoutBriefNamesTheEvidenceShapeItAccepts(t *testing.T) {
	rule := renderScoutNewEvidenceRule()
	for _, name := range []string{"reference", "kind", "origin", "repository_path", "applicable_dimensions", "summary"} {
		if !strings.Contains(rule, "`"+name+"`") {
			t.Errorf("Scout evidence rule does not name %q:\n%s", name, rule)
		}
	}
	for _, name := range []string{"reference", "kind", "origin", "applicable_dimensions", "summary"} {
		if !strings.Contains(scoutMinimalEvidenceEntryJSON, `"`+name+`"`) {
			t.Errorf("the accepted wire fixture does not use %q, so the rule and the proof have drifted", name)
		}
	}
	if !strings.Contains(rule, "content_hash") || !strings.Contains(strings.ToLower(rule), "do not") {
		t.Errorf("Scout evidence rule must tell the helper not to compute fingerprints:\n%s", rule)
	}
	stage := planningStageManifest{ExpectedCaste: planningStageCasteScout}
	if !strings.Contains(renderStagedResultContract(stage, "planning"), rule) {
		t.Error("the Scout's staged result contract does not carry the evidence rule")
	}
	route := planningStageManifest{ExpectedCaste: planningStageCasteRouteSetter}
	if strings.Contains(renderStagedResultContract(route, "planning"), rule) {
		t.Error("the Route-Setter contract must not carry the Scout's evidence rule")
	}
}
