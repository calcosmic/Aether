package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestScoutNewEvidenceNeedsNoFingerprintFromTheHelper proves the actual bug
// fix on the real two-pass planning flow: a pass-2 Scout result carrying
// source-shaped new evidence with NO content_hash, id, or excerpt_digest is
// accepted, the evidence is catalogued with a program-derived hash/ID/digest
// and the current run's own scope, and a plan revision citing that derived
// evidence is allowed. At HEAD (before the fix) this is red: the Scout helper
// cannot compute a SHA-256 content hash, so the submission is refused.
func TestScoutNewEvidenceNeedsNoFingerprintFromTheHelper(t *testing.T) {
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil {
		t.Fatal("first Route pass did not authorize the next Scout")
	}
	scoutManifest := first.ScoutDispatch.Manifest

	const excerpt = "A later Scout pass found evidence it cannot fingerprint itself."
	bare := planningEvidenceRecord{
		Reference: colony.PlanningEvidenceRef{
			Kind:                 colony.PlanningEvidenceResearch,
			Origin:               "scout:pass-2:no-fingerprint-finding",
			ApplicableDimensions: colony.PlanningDimensions(),
		},
		Summary: excerpt,
		Locator: planningEvidenceLocator{Kind: colony.PlanningEvidenceResearch, Origin: "scout:pass-2:no-fingerprint-finding"},
	}
	gap := colony.PlanningGap{
		SchemaVersion: colony.PlanningSchemaVersion, ID: "gap-no-fingerprint", ContentHash: planningStageTestHash("7"),
		Dimension: colony.PlanningDimensionKnowledge, Materiality: colony.PlanningGapNonMaterial, Severity: 15,
		Description:             "The evidence the second Scout pass found still needs a fingerprint before Route-Setter can act.",
		EvidenceThatWouldChange: "A confirmed, program-derived address for the newly submitted evidence.",
	}
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		Findings: []planningScoutStageFinding{{
			StableID: "no-fingerprint-finding", Summary: "A later pass observed new evidence without computing its own hash.",
			Unknown: true, UnknownReason: "the exact evidence address is derived by the program, not the Scout",
		}},
		NewEvidence:    []planningEvidenceRecord{bare},
		UnresolvedGaps: []colony.PlanningGap{gap},
	}

	coordinated, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err != nil {
		t.Fatalf("pass-2 Scout evidence with no fingerprint = %v, want acceptance", err)
	}
	if len(coordinated.Scout.Result.NewEvidence) != 1 {
		t.Fatalf("Scout completion = %+v, want exactly one derived evidence record", coordinated.Scout.Result)
	}
	derived := coordinated.Scout.Result.NewEvidence[0].Reference
	if derived.ContentHash == "" || !strings.HasSuffix(derived.ID, "-"+derived.ContentHash[:12]) {
		t.Fatalf("derived evidence reference = %+v, want a program-computed content address", derived)
	}
	if derived.GoalID != "goal-200" || derived.SessionID != "session-200" ||
		derived.SpecificationRevisionID != scoutManifest.Specification.RevisionID || derived.PlanRevisionID != scoutManifest.BasePlanRevisionID {
		t.Fatalf("derived evidence scope = %+v, want the current run authority", derived)
	}
	if !derived.Fresh || !derived.Admissible {
		t.Fatalf("derived evidence = %+v, want fresh and admissible", derived)
	}
	if coordinated.RouteDispatch == nil {
		t.Fatal("Scout completion did not authorize the next Route-Setter")
	}

	// A plan revision may now cite the program-derived evidence.
	routeManifest := coordinated.RouteDispatch.Manifest
	secondResult := planningRouteStageResult{
		ResultType: planningStageResultRouteSetter, ManifestID: routeManifest.ID, ManifestHash: routeManifest.ContentHash,
		RunID: routeManifest.RunID, Pass: routeManifest.Pass, Caste: planningStageCasteRouteSetter,
		Specification: routeManifest.Specification, BasePlanRevisionID: routeManifest.BasePlanRevisionID, BasePlanRevisionHash: routeManifest.BasePlanRevisionHash,
		PriorCardHash: routeManifest.PriorCardHash, InputFrontierHash: routeManifest.InputFrontierHash,
		ScoutReceipt: *routeManifest.ScoutReceipt, CandidateSnapshotHash: routeManifest.CandidateSnapshotHash,
		Proposal: firstResult.Proposal, ProposalEvidenceIDs: []string{derived.ID},
	}
	for index, dimension := range colony.PlanningDimensions() {
		before := first.Route.Card.DimensionAssessments[index].After
		secondResult.DimensionAssessments = append(secondResult.DimensionAssessments, colony.PlanningDimensionAssessment{
			SchemaVersion: colony.PlanningSchemaVersion, ID: "no-fingerprint-assessment-" + string(dimension), ContentHash: planningStageTestHash(string(rune('a' + index))),
			Dimension: dimension, Before: before, After: before,
			FreshEvidenceIDs:  []string{derived.ID},
			RemainingGap:      planningRouteStageGap("no-fingerprint-route-gap-"+string(dimension), dimension, derived.ID, colony.PlanningGapNonMaterial, 12+index),
			Rationale:         "The program-derived evidence leaves this dimension unchanged.",
			ProducerReceiptID: routeManifest.ID,
		})
	}
	second, err := coordinatePlanningRouteStage(root, routeManifest, planningRouteStageTestBytes(t, secondResult))
	if err != nil {
		t.Fatalf("plan revision citing the program-derived evidence = %v, want acceptance", err)
	}
	if second.Route.Card.ContentHash == "" {
		t.Fatal("Route completion did not produce a card citing the program-derived evidence")
	}
}

// TestHelperSuppliedFingerprintIsNeverTrusted proves a fingerprint supplied by
// the untrusted helper is never honoured: a wrong hash/id is recomputed, a
// claimed repository path is always read from disk (content supplied by the
// helper for a repository path is ignored), and a path outside the approved
// evidence roots is refused by name.
func TestHelperSuppliedFingerprintIsNeverTrusted(t *testing.T) {
	t.Run("wrong hash is recomputed, not honoured", func(t *testing.T) {
		root, firstManifest, firstResult := planningRouteStageTestFixture(t)
		first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
		if err != nil {
			t.Fatal(err)
		}
		scoutManifest := first.ScoutDispatch.Manifest

		const excerpt = "A later Scout pass claims a fingerprint it cannot actually compute."
		const origin = "scout:pass-2:fabricated-fingerprint"
		wrongHash := planningStageTestHash("f")
		fabricated := planningEvidenceRecord{
			Reference: colony.PlanningEvidenceRef{
				SchemaVersion:        colony.PlanningEvidenceSchemaVersion,
				ID:                   "evidence-research-" + wrongHash[:12],
				ContentHash:          wrongHash,
				Kind:                 colony.PlanningEvidenceResearch,
				Origin:               origin,
				ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
				ExcerptDigest:        planningStageTestHash("e"),
			},
			Summary: excerpt,
			Locator: planningEvidenceLocator{Kind: colony.PlanningEvidenceResearch, Origin: origin},
		}
		scoutResult := planningScoutStageResult{
			ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
			RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
			Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
			InputFrontierHash: scoutManifest.InputFrontierHash,
			NewEvidence:       []planningEvidenceRecord{fabricated},
		}
		coordinated, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
		if err != nil {
			t.Fatalf("fabricated-fingerprint Scout evidence = %v, want acceptance with the fingerprint ignored", err)
		}
		derived := coordinated.Scout.Result.NewEvidence[0].Reference
		if derived.ContentHash == wrongHash || derived.ID == fabricated.Reference.ID {
			t.Fatalf("derived evidence = %+v, want the fabricated fingerprint %q/%q discarded", derived, fabricated.Reference.ID, wrongHash)
		}

		scope := planningEvidenceScope{
			GoalID: "goal-200", SessionID: "session-200",
			SpecificationRevisionID: scoutManifest.Specification.RevisionID, PlanRevisionID: scoutManifest.BasePlanRevisionID,
		}
		expected, err := normalizePlanningEvidence(planningEvidenceSource{
			Kind: colony.PlanningEvidenceResearch, Origin: origin, Content: []byte(excerpt),
			Scope:                scope,
			SourceRevision:       planningEvidenceSourceRevision("scout-evidence", []byte(excerpt)),
			ObservedAt:           time.Now().UTC(),
			ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
		})
		if err != nil {
			t.Fatal(err)
		}
		if derived.ContentHash != expected.Reference.ContentHash || derived.ID != expected.Reference.ID {
			t.Fatalf("derived evidence = %+v, want the honestly recomputed address %+v", derived, expected.Reference)
		}
	})

	t.Run("repository path is read from disk, submitted content is ignored", func(t *testing.T) {
		root, firstManifest, firstResult := planningRouteStageTestFixture(t)
		first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
		if err != nil {
			t.Fatal(err)
		}
		scoutManifest := first.ScoutDispatch.Manifest

		const realContent = "The real file on disk is the only content the program will ever hash."
		relPath := filepath.ToSlash(filepath.Join("docs", "pass2-note.md"))
		if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(relPath)), []byte(realContent), 0o644); err != nil {
			t.Fatal(err)
		}

		misleadingHash := planningStageTestHash("b")
		repoSubmission := planningEvidenceRecord{
			Reference: colony.PlanningEvidenceRef{
				SchemaVersion:        colony.PlanningEvidenceSchemaVersion,
				ID:                   "evidence-research-" + misleadingHash[:12],
				ContentHash:          misleadingHash,
				Kind:                 colony.PlanningEvidenceResearch,
				RepositoryPath:       relPath,
				ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
			},
			Summary: "This misleading excerpt is not what the program should hash.",
			Locator: planningEvidenceLocator{Kind: colony.PlanningEvidenceResearch, RepositoryPath: relPath},
		}
		scoutResult := planningScoutStageResult{
			ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
			RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
			Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
			InputFrontierHash: scoutManifest.InputFrontierHash,
			NewEvidence:       []planningEvidenceRecord{repoSubmission},
		}
		coordinated, err := coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
		if err != nil {
			t.Fatalf("repository-path Scout evidence = %v, want acceptance reading the real file", err)
		}
		derived := coordinated.Scout.Result.NewEvidence[0]
		if derived.Reference.ContentHash == misleadingHash {
			t.Fatalf("derived evidence = %+v, want the misleading claimed hash %q discarded", derived.Reference, misleadingHash)
		}
		if derived.Summary != realContent {
			t.Fatalf("derived summary = %q, want the real file content %q, never the submitted excerpt", derived.Summary, realContent)
		}
	})

	t.Run("a path outside the approved roots is refused by name", func(t *testing.T) {
		root, firstManifest, firstResult := planningRouteStageTestFixture(t)
		first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
		if err != nil {
			t.Fatal(err)
		}
		scoutManifest := first.ScoutDispatch.Manifest

		outside := planningEvidenceRecord{
			Reference: colony.PlanningEvidenceRef{
				Kind:                 colony.PlanningEvidenceResearch,
				RepositoryPath:       "../outside-the-repository.md",
				ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
			},
			Summary: "This path claims to live outside the repository entirely.",
			Locator: planningEvidenceLocator{Kind: colony.PlanningEvidenceResearch, RepositoryPath: "../outside-the-repository.md"},
		}
		scoutResult := planningScoutStageResult{
			ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
			RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
			Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
			InputFrontierHash: scoutManifest.InputFrontierHash,
			NewEvidence:       []planningEvidenceRecord{outside},
		}
		_, err = coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
		if err == nil || !strings.Contains(strings.ToLower(err.Error()), "escapes its root") {
			t.Fatalf("outside-root Scout evidence error = %v, want a named path-outside-approved-roots refusal", err)
		}
	})
}

// TestRestatedEvidenceIsStillRefused proves the frontier-restatement refusal
// still holds once new evidence is derived, not merely trusted: evidence
// whose program-derived address equals a frontier entry the current run
// already knows about is refused.
func TestRestatedEvidenceIsStillRefused(t *testing.T) {
	root, firstManifest, firstResult := planningRouteStageTestFixture(t)
	first, err := coordinatePlanningRouteStage(root, firstManifest, planningRouteStageTestBytes(t, firstResult))
	if err != nil {
		t.Fatal(err)
	}
	if first.ScoutDispatch == nil {
		t.Fatal("first Route pass did not authorize the next Scout")
	}
	scoutManifest := first.ScoutDispatch.Manifest

	var priorOrigin string
	for _, ref := range first.ScoutDispatch.Evidence {
		if ref.Origin == "route-fresh" {
			priorOrigin = ref.Origin
			break
		}
	}
	if priorOrigin == "" {
		t.Fatal("the first Route pass's own cited evidence is not carried into the next Scout frontier")
	}

	restated := planningEvidenceRecord{
		Reference: colony.PlanningEvidenceRef{
			Kind:                 colony.PlanningEvidenceResearch,
			Origin:               priorOrigin,
			SourceRevision:       priorOrigin + "-revision",
			ApplicableDimensions: colony.PlanningDimensions(),
		},
		Summary: "Fresh evidence supports the complete Route proposal.",
		Locator: planningEvidenceLocator{Kind: colony.PlanningEvidenceResearch, Origin: priorOrigin, SourceRevision: priorOrigin + "-revision"},
	}
	scoutResult := planningScoutStageResult{
		ResultType: planningStageResultScout, ManifestID: scoutManifest.ID, ManifestHash: scoutManifest.ContentHash,
		RunID: scoutManifest.RunID, Pass: scoutManifest.Pass, Caste: planningStageCasteScout,
		Specification: scoutManifest.Specification, BasePlanRevisionID: scoutManifest.BasePlanRevisionID, BasePlanRevisionHash: scoutManifest.BasePlanRevisionHash,
		InputFrontierHash: scoutManifest.InputFrontierHash,
		NewEvidence:       []planningEvidenceRecord{restated},
	}
	_, err = coordinatePlanningScoutStage(root, scoutManifest, planningScoutStageTestBytes(t, scoutResult))
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "restates the prior frontier") {
		t.Fatalf("restated evidence error = %v, want a restates-the-prior-frontier refusal", err)
	}
}
