package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestPlanningEvidenceCollectAllKindsInStableOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	observedAt := time.Date(2026, time.September, 7, 12, 30, 0, 0, time.UTC)
	kinds := []colony.PlanningEvidenceKind{
		colony.PlanningEvidenceSpecification,
		colony.PlanningEvidenceSurvey,
		colony.PlanningEvidenceCharter,
		colony.PlanningEvidenceDecision,
		colony.PlanningEvidenceContext,
		colony.PlanningEvidenceResearch,
		colony.PlanningEvidenceHive,
		colony.PlanningEvidenceOutcome,
	}
	sources := make([]planningEvidenceSource, 0, len(kinds))
	for _, kind := range kinds {
		sources = append(sources, planningEvidenceSource{
			Kind:                 kind,
			Origin:               "fixture:" + string(kind),
			Content:              []byte("Grounded evidence for " + string(kind)),
			Scope:                planningEvidenceScopeFixture(),
			SourceRevision:       "source-revision-1",
			ObservedAt:           observedAt,
			ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
		})
	}

	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		Sources:        sources,
	})
	if err != nil {
		t.Fatalf("collectPlanningEvidence returned error: %v", err)
	}
	if len(records) != len(kinds) {
		t.Fatalf("records = %d, want all %d evidence kinds", len(records), len(kinds))
	}

	ids := make([]string, 0, len(records))
	seenKinds := make(map[colony.PlanningEvidenceKind]bool, len(kinds))
	for _, record := range records {
		ref := record.Reference
		ids = append(ids, ref.ID)
		seenKinds[ref.Kind] = true
		if ref.SchemaVersion != colony.PlanningEvidenceSchemaVersion {
			t.Fatalf("%s schema = %q", ref.Kind, ref.SchemaVersion)
		}
		if len(ref.ContentHash) != 64 || len(ref.ExcerptDigest) != 64 {
			t.Fatalf("%s lacks full content or excerpt digest: %+v", ref.Kind, ref)
		}
		if !ref.Fresh || !ref.Admissible {
			t.Fatalf("new current %s evidence is not fresh and admissible: %+v", ref.Kind, ref)
		}
		if record.Summary == "" || record.Locator.Origin != ref.Origin || record.Locator.SourceRevision != ref.SourceRevision {
			t.Fatalf("%s lost safe summary or locator metadata: %+v", ref.Kind, record)
		}
	}
	for _, kind := range kinds {
		if !seenKinds[kind] {
			t.Errorf("kind %q was not collected", kind)
		}
	}
	if !sort.StringsAreSorted(ids) {
		t.Fatalf("evidence IDs are not stably sorted: %v", ids)
	}

	reversed := append([]planningEvidenceSource(nil), sources...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	again, err := collectPlanningEvidence(planningEvidenceCollectionRequest{RepositoryRoot: root, Sources: reversed})
	if err != nil {
		t.Fatalf("collect reversed evidence: %v", err)
	}
	if got := planningEvidenceRecordIDs(again); !reflect.DeepEqual(got, ids) {
		t.Fatalf("input order changed catalogue IDs:\n got: %v\nwant: %v", got, ids)
	}
}

func TestPlanningEvidenceCollectCanonicalizesRepositoryPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := filepath.Join(root, "evidence", "spec.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("approved specification\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		ApprovedRoots:  []string{"evidence"},
		Sources: []planningEvidenceSource{{
			Kind:                 colony.PlanningEvidenceSpecification,
			Origin:               "ignored-for-repository-source",
			RepositoryPath:       "evidence/notes/../spec.md",
			Scope:                planningEvidenceScopeFixture(),
			SourceRevision:       "spec-revision-1",
			ObservedAt:           time.Date(2026, time.September, 7, 12, 31, 0, 0, time.UTC),
			ApplicableDimensions: colony.PlanningDimensions(),
		}},
	})
	if err != nil {
		t.Fatalf("collect repository evidence: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	ref := records[0].Reference
	if ref.Origin != "evidence/spec.md" || ref.RepositoryPath != "evidence/spec.md" {
		t.Fatalf("repository origin was not canonicalized: %+v", ref)
	}
}

func TestPlanningEvidenceCollectExactDuplicateDoesNotBecomeFreshAgain(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	originalTime := time.Date(2026, time.September, 7, 9, 0, 0, 0, time.UTC)
	source := planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceResearch,
		Origin:               "research:phase-200",
		Content:              []byte("One stable finding"),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       "research-revision-1",
		ObservedAt:           originalTime,
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	}
	first, err := collectPlanningEvidence(planningEvidenceCollectionRequest{RepositoryRoot: root, Sources: []planningEvidenceSource{source}})
	if err != nil {
		t.Fatal(err)
	}
	first[0].Reference.Fresh = false
	first[0].Reference.AdmissibilityReason = "already cited at card frontier"

	reread := source
	reread.ObservedAt = originalTime.Add(3 * time.Hour)
	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		Existing:       first,
		Sources:        []planningEvidenceSource{reread, reread},
	})
	if err != nil {
		t.Fatalf("collect duplicate evidence: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("duplicate evidence produced %d records, want 1", len(records))
	}
	if records[0].Reference.ID != first[0].Reference.ID {
		t.Fatalf("duplicate changed ID: got %q want %q", records[0].Reference.ID, first[0].Reference.ID)
	}
	if !records[0].Reference.ObservedAt.Equal(originalTime) || records[0].Reference.Fresh {
		t.Fatalf("duplicate became fresh or changed original observation: %+v", records[0].Reference)
	}
}

func TestPlanningEvidenceNormalizeContentAddressChangesOnlyWithMeaningfulInput(t *testing.T) {
	t.Parallel()

	source := planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceContext,
		Origin:               "context:capsule",
		Content:              []byte("line one  \r\nline two\r\n"),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       "capsule-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 11, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	}
	first, err := normalizePlanningEvidence(source)
	if err != nil {
		t.Fatalf("normalize first source: %v", err)
	}

	formatOnly := source
	formatOnly.Content = []byte("line one\nline two\n")
	formatOnly.ObservedAt = source.ObservedAt.Add(time.Hour)
	second, err := normalizePlanningEvidence(formatOnly)
	if err != nil {
		t.Fatalf("normalize equivalent source: %v", err)
	}
	if first.Reference.ID != second.Reference.ID || first.Reference.ContentHash != second.Reference.ContentHash {
		t.Fatalf("line-ending normalization changed identity:\nfirst=%+v\nsecond=%+v", first.Reference, second.Reference)
	}

	changed := source
	changed.Content = []byte("line one\nline two changed\n")
	third, err := normalizePlanningEvidence(changed)
	if err != nil {
		t.Fatalf("normalize changed source: %v", err)
	}
	if third.Reference.ID == first.Reference.ID || third.Reference.ContentHash == first.Reference.ContentHash {
		t.Fatalf("changed content reused stale identity:\nold=%+v\nnew=%+v", first.Reference, third.Reference)
	}

	changed.ExpectedContentHash = first.Reference.ContentHash
	_, err = normalizePlanningEvidence(changed)
	var refusal *planningEvidenceRefusal
	if !errors.As(err, &refusal) || refusal.Code != planningEvidenceRefusalHashMismatch {
		t.Fatalf("changed content against old claim error = %v, want typed hash mismatch", err)
	}
}

func TestPlanningEvidenceCollectRejectsTraversalBeforeRead(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	readCalls := 0
	_, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		ApprovedRoots:  []string{".planning"},
		ReadFile: func(path string) ([]byte, error) {
			readCalls++
			return []byte("must not be read"), nil
		},
		Sources: []planningEvidenceSource{{
			Kind:                 colony.PlanningEvidenceResearch,
			RepositoryPath:       ".planning/../../outside-secret.md",
			Scope:                planningEvidenceScopeFixture(),
			SourceRevision:       "research-revision-1",
			ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
			ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
		}},
	})
	var refusal *planningEvidenceRefusal
	if !errors.As(err, &refusal) || refusal.Code != planningEvidenceRefusalPathOutsideApprovedRoots {
		t.Fatalf("traversal error = %v, want typed path refusal", err)
	}
	if readCalls != 0 {
		t.Fatalf("traversal path was read %d time(s), want zero", readCalls)
	}
}

func TestPlanningEvidenceNormalizeReturnsTypedRefusals(t *testing.T) {
	t.Parallel()

	valid := planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceResearch,
		Origin:               "research:phase-200",
		Content:              []byte("grounded finding"),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       "research-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionKnowledge},
	}

	tests := []struct {
		name string
		edit func(*planningEvidenceSource)
		code planningEvidenceRefusalCode
	}{
		{name: "unsupported kind", edit: func(source *planningEvidenceSource) { source.Kind = "future-source" }, code: planningEvidenceRefusalUnsupportedKind},
		{name: "missing content", edit: func(source *planningEvidenceSource) { source.Content = nil }, code: planningEvidenceRefusalMissingContent},
		{name: "hash mismatch", edit: func(source *planningEvidenceSource) { source.ExpectedContentHash = strings.Repeat("0", 64) }, code: planningEvidenceRefusalHashMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := valid
			tt.edit(&source)
			_, err := normalizePlanningEvidence(source)
			var refusal *planningEvidenceRefusal
			if !errors.As(err, &refusal) || refusal.Code != tt.code {
				t.Fatalf("error = %v, want typed refusal %q", err, tt.code)
			}
		})
	}
}

func TestPlanningEvidenceCollectStoresRedactedSummaryAndLocatorOnly(t *testing.T) {
	t.Parallel()

	const secret = "supersecret123"
	source := planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceDecision,
		Origin:               "decision:owner-1",
		Content:              []byte("Use the safe path. password = \"" + secret + "\""),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       "decision-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionRequirements},
	}
	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{RepositoryRoot: t.TempDir(), Sources: []planningEvidenceSource{source}})
	if err != nil {
		t.Fatalf("collect sensitive evidence: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	encoded := records[0].Summary + " " + records[0].Locator.Origin
	if strings.Contains(encoded, secret) || strings.Contains(encoded, string(source.Content)) {
		t.Fatalf("catalogue exposed unrestricted source content: %q", encoded)
	}
	if !strings.Contains(strings.ToLower(records[0].Summary), "redacted") {
		t.Fatalf("sensitive summary did not disclose redaction: %q", records[0].Summary)
	}
	if records[0].Locator.Origin != source.Origin || records[0].Locator.SourceRevision != source.SourceRevision {
		t.Fatalf("locator metadata was lost: %+v", records[0].Locator)
	}
}

func TestPlanningEvidenceCollectRejectsTamperedExistingProjection(t *testing.T) {
	t.Parallel()

	record := planningEvidenceRecordFixture(t, colony.PlanningEvidenceResearch, "research:trusted", "trusted finding", []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
	})
	tests := []struct {
		name string
		edit func(*planningEvidenceRecord)
		code planningEvidenceRefusalCode
	}{
		{
			name: "summary digest changed",
			edit: func(value *planningEvidenceRecord) { value.Summary = "tampered finding" },
			code: planningEvidenceRefusalHashMismatch,
		},
		{
			name: "locator changed",
			edit: func(value *planningEvidenceRecord) { value.Locator.Origin = "research:other" },
			code: planningEvidenceRefusalInvalidMetadata,
		},
		{
			name: "unsafe summary with matching digest",
			edit: func(value *planningEvidenceRecord) {
				value.Summary = `password = "secret-password"`
				value.Reference.ExcerptDigest = planningEvidenceSHA256([]byte(value.Summary))
			},
			code: planningEvidenceRefusalInvalidMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tampered := record
			tt.edit(&tampered)
			_, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
				RepositoryRoot: t.TempDir(),
				Existing:       []planningEvidenceRecord{tampered},
			})
			var refusal *planningEvidenceRefusal
			if !errors.As(err, &refusal) || refusal.Code != tt.code {
				t.Fatalf("tampered projection error = %v, want %q", err, tt.code)
			}
		})
	}
}

func TestPlanningEvidenceFreshRejectsRestatementAndSupersededSource(t *testing.T) {
	t.Parallel()

	original := planningEvidenceRecordFixture(t, colony.PlanningEvidenceResearch, "research:phase-200", "first grounded finding", []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
	})
	frontier := planningEvidenceFrontierFixture(original.Reference)

	result := isFreshPlanningEvidence(original.Reference, frontier)
	if !result.Allowed || result.Reason == "" {
		t.Fatalf("current uncited evidence was not fresh: %+v", result)
	}

	frontier.CitedEvidenceIDs = []string{original.Reference.ID}
	result = isFreshPlanningEvidence(original.Reference, frontier)
	if result.Allowed || result.Code != planningEvidencePolicyAlreadyCited || !strings.Contains(result.Reason, "already") {
		t.Fatalf("restated evidence result = %+v, want already-cited refusal", result)
	}

	changed := planningEvidenceRecordFixture(t, colony.PlanningEvidenceResearch, "research:phase-200", "changed grounded finding", []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
	})
	if changed.Reference.ID == original.Reference.ID {
		t.Fatal("fixture content change did not change evidence identity")
	}
	frontier = planningEvidenceFrontierFixture(changed.Reference)
	if result = isFreshPlanningEvidence(changed.Reference, frontier); !result.Allowed {
		t.Fatalf("changed current evidence was not fresh: %+v", result)
	}
	frontier.CurrentSourceHashes[planningEvidenceSourceKey(original.Reference)] = changed.Reference.ContentHash
	if result = isFreshPlanningEvidence(original.Reference, frontier); result.Allowed || result.Code != planningEvidencePolicySourceSuperseded {
		t.Fatalf("old content after source change result = %+v, want superseded refusal", result)
	}
}

func TestPlanningEvidenceScopeRequiresExactPlanningFrontier(t *testing.T) {
	t.Parallel()

	record := planningEvidenceRecordFixture(t, colony.PlanningEvidenceContext, "context:capsule", "scoped context", []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
	})
	base := planningEvidenceFrontierFixture(record.Reference)

	tests := []struct {
		name string
		edit func(*planningEvidenceFrontier)
		code planningEvidencePolicyCode
	}{
		{name: "goal drift", edit: func(frontier *planningEvidenceFrontier) { frontier.Scope.GoalID = "goal-changed" }, code: planningEvidencePolicyGoalMismatch},
		{name: "session drift", edit: func(frontier *planningEvidenceFrontier) { frontier.Scope.SessionID = "session-changed" }, code: planningEvidencePolicySessionMismatch},
		{name: "specification drift", edit: func(frontier *planningEvidenceFrontier) { frontier.Scope.SpecificationRevisionID = "spec-rev-changed" }, code: planningEvidencePolicySpecificationMismatch},
		{name: "base plan drift", edit: func(frontier *planningEvidenceFrontier) { frontier.Scope.PlanRevisionID = "plan-rev-changed" }, code: planningEvidencePolicyPlanMismatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frontier := base
			frontier.CurrentSourceHashes = clonePlanningEvidenceHashMap(base.CurrentSourceHashes)
			tt.edit(&frontier)
			result := isFreshPlanningEvidence(record.Reference, frontier)
			if result.Allowed || result.Code != tt.code || result.Reason == "" {
				t.Fatalf("scope result = %+v, want refusal %q with reason", result, tt.code)
			}
		})
	}
}

func TestPlanningEvidenceFreshRejectsInactiveHiveStates(t *testing.T) {
	t.Parallel()

	record := planningEvidenceRecordFixture(t, colony.PlanningEvidenceHive, "hive:w-200", "cross-colony finding", []colony.PlanningDimension{
		colony.PlanningDimensionRisks,
	})
	base := planningEvidenceFrontierFixture(record.Reference)

	tests := []struct {
		state planningEvidenceSourceState
		code  planningEvidencePolicyCode
	}{
		{state: planningEvidenceSourceRevoked, code: planningEvidencePolicySourceRevoked},
		{state: planningEvidenceSourceQuarantined, code: planningEvidencePolicySourceQuarantined},
		{state: planningEvidenceSourceDormant, code: planningEvidencePolicySourceDormant},
		{state: planningEvidenceSourceSuperseded, code: planningEvidencePolicySourceSuperseded},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			frontier := base
			frontier.CurrentSourceHashes = clonePlanningEvidenceHashMap(base.CurrentSourceHashes)
			frontier.SourceStates = map[string]planningEvidenceSourceState{
				planningEvidenceSourceKey(record.Reference): tt.state,
			}
			result := isFreshPlanningEvidence(record.Reference, frontier)
			if result.Allowed || result.Code != tt.code || !strings.Contains(result.Reason, string(tt.state)) {
				t.Fatalf("%s Hive result = %+v, want surfaced refusal %q", tt.state, result, tt.code)
			}
		})
	}
}

func TestPlanningEvidenceFreshOwnerAnswerRequiresExactEquivalence(t *testing.T) {
	t.Parallel()

	equivalence := planningOwnerAnswerEquivalenceFixture()
	key, err := planningOwnerAnswerEquivalenceKey(equivalence)
	if err != nil {
		t.Fatalf("compute owner-answer equivalence key: %v", err)
	}
	sameKey, err := planningOwnerAnswerEquivalenceKey(equivalence)
	if err != nil || sameKey != key {
		t.Fatalf("identical equivalence scope changed key: got %q want %q err=%v", sameKey, key, err)
	}

	source := planningEvidenceSource{
		Kind:                 colony.PlanningEvidenceDecision,
		Origin:               "decision:owner-answer-1",
		Content:              []byte("Keep the current owner-selected behavior"),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       key,
		ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ApplicableDimensions: []colony.PlanningDimension{colony.PlanningDimensionRequirements},
	}
	record, err := normalizePlanningEvidence(source)
	if err != nil {
		t.Fatalf("normalize owner answer: %v", err)
	}
	frontier := planningEvidenceFrontierFixture(record.Reference)
	frontier.OwnerAnswerEquivalenceKey = key
	if result := isFreshPlanningEvidence(record.Reference, frontier); !result.Allowed {
		t.Fatalf("identical owner-answer equivalence was not reusable: %+v", result)
	}

	changes := []struct {
		name string
		edit func(*planningOwnerAnswerEquivalence)
	}{
		{name: "goal", edit: func(value *planningOwnerAnswerEquivalence) { value.GoalID = "goal-changed" }},
		{name: "session", edit: func(value *planningOwnerAnswerEquivalence) { value.SessionID = "session-changed" }},
		{name: "specification", edit: func(value *planningOwnerAnswerEquivalence) { value.SpecificationRevisionID = "spec-changed" }},
		{name: "base plan", edit: func(value *planningOwnerAnswerEquivalence) { value.BasePlanRevisionID = "plan-changed" }},
		{name: "meaning", edit: func(value *planningOwnerAnswerEquivalence) { value.MeaningHash = strings.Repeat("1", 64) }},
		{name: "behavior", edit: func(value *planningOwnerAnswerEquivalence) { value.BehaviorHash = strings.Repeat("2", 64) }},
		{name: "impact", edit: func(value *planningOwnerAnswerEquivalence) { value.ImpactHash = strings.Repeat("3", 64) }},
		{name: "risk", edit: func(value *planningOwnerAnswerEquivalence) { value.RiskHash = strings.Repeat("4", 64) }},
		{name: "acceptance", edit: func(value *planningOwnerAnswerEquivalence) { value.AcceptanceHash = strings.Repeat("5", 64) }},
	}
	for _, tt := range changes {
		t.Run(tt.name, func(t *testing.T) {
			changed := equivalence
			tt.edit(&changed)
			changedKey, keyErr := planningOwnerAnswerEquivalenceKey(changed)
			if keyErr != nil {
				t.Fatalf("compute changed key: %v", keyErr)
			}
			if changedKey == key {
				t.Fatalf("%s change retained owner-answer equivalence key", tt.name)
			}
			changedFrontier := frontier
			changedFrontier.CurrentSourceHashes = clonePlanningEvidenceHashMap(frontier.CurrentSourceHashes)
			changedFrontier.OwnerAnswerEquivalenceKey = changedKey
			result := isFreshPlanningEvidence(record.Reference, changedFrontier)
			if result.Allowed || result.Code != planningEvidencePolicyOwnerAnswerChanged || !strings.Contains(result.Reason, "revalidation") {
				t.Fatalf("%s change result = %+v, want owner-answer revalidation", tt.name, result)
			}
		})
	}
}

func TestPlanningEvidenceAppliesOnlyToExplicitClaimedDimension(t *testing.T) {
	t.Parallel()

	riskOnly := planningEvidenceRecordFixture(t, colony.PlanningEvidenceResearch, "research:risk-only", "risk finding", []colony.PlanningDimension{
		colony.PlanningDimensionRisks,
	}).Reference
	if result := evidenceAppliesToDimension(riskOnly, colony.PlanningDimensionRisks); !result.Allowed {
		t.Fatalf("risk claim did not apply to Risks: %+v", result)
	}
	for _, dimension := range []colony.PlanningDimension{colony.PlanningDimensionRequirements, colony.PlanningDimensionEffort} {
		result := evidenceAppliesToDimension(riskOnly, dimension)
		if result.Allowed || result.Code != planningEvidencePolicyDimensionNotClaimed || result.Reason == "" {
			t.Fatalf("risk-only evidence applied to %s: %+v", dimension, result)
		}
	}

	multiClaim := riskOnly
	multiClaim.ApplicableDimensions = []colony.PlanningDimension{
		colony.PlanningDimensionRequirements,
		colony.PlanningDimensionRisks,
		colony.PlanningDimensionEffort,
	}
	for _, dimension := range multiClaim.ApplicableDimensions {
		if result := evidenceAppliesToDimension(multiClaim, dimension); !result.Allowed {
			t.Fatalf("explicit multi-claim did not apply to %s: %+v", dimension, result)
		}
	}
}

func TestPlanningEvidenceAppliesHonorsKindMatrixAndAdmissibility(t *testing.T) {
	t.Parallel()

	survey := planningEvidenceRecordFixture(t, colony.PlanningEvidenceSurvey, "survey:repo", "repository inventory", []colony.PlanningDimension{
		colony.PlanningDimensionKnowledge,
	}).Reference
	survey.ApplicableDimensions = append(survey.ApplicableDimensions, colony.PlanningDimensionRequirements)
	result := evidenceAppliesToDimension(survey, colony.PlanningDimensionRequirements)
	if result.Allowed || result.Code != planningEvidencePolicyKindNotApplicable {
		t.Fatalf("survey bypassed kind applicability matrix: %+v", result)
	}

	survey.Admissible = false
	survey.AdmissibilityReason = "scope-mismatched context"
	result = evidenceAppliesToDimension(survey, colony.PlanningDimensionKnowledge)
	if result.Allowed || result.Code != planningEvidencePolicyInadmissible || !strings.Contains(result.Reason, "scope-mismatched") {
		t.Fatalf("inadmissible evidence reason was not surfaced: %+v", result)
	}

	invalidDimension := evidenceAppliesToDimension(survey, colony.PlanningDimension("overall"))
	if invalidDimension.Allowed || invalidDimension.Code != planningEvidencePolicyInvalidDimension {
		t.Fatalf("invalid dimension result = %+v", invalidDimension)
	}
}

func planningEvidenceRecordFixture(t *testing.T, kind colony.PlanningEvidenceKind, origin, content string, dimensions []colony.PlanningDimension) planningEvidenceRecord {
	t.Helper()
	record, err := normalizePlanningEvidence(planningEvidenceSource{
		Kind:                 kind,
		Origin:               origin,
		Content:              []byte(content),
		Scope:                planningEvidenceScopeFixture(),
		SourceRevision:       "source-revision-1",
		ObservedAt:           time.Date(2026, time.September, 7, 12, 0, 0, 0, time.UTC),
		ApplicableDimensions: dimensions,
	})
	if err != nil {
		t.Fatalf("normalize fixture: %v", err)
	}
	return record
}

func planningEvidenceFrontierFixture(ref colony.PlanningEvidenceRef) planningEvidenceFrontier {
	return planningEvidenceFrontier{
		Scope: planningEvidenceScope{
			GoalID:                  ref.GoalID,
			SessionID:               ref.SessionID,
			SpecificationRevisionID: ref.SpecificationRevisionID,
			PlanRevisionID:          ref.PlanRevisionID,
		},
		CurrentSourceHashes: map[string]string{planningEvidenceSourceKey(ref): ref.ContentHash},
	}
}

func planningOwnerAnswerEquivalenceFixture() planningOwnerAnswerEquivalence {
	return planningOwnerAnswerEquivalence{
		GoalID:                  "goal-200",
		SessionID:               "session-200",
		SpecificationRevisionID: "spec-rev-1",
		BasePlanRevisionID:      "plan-rev-1",
		MeaningHash:             strings.Repeat("a", 64),
		BehaviorHash:            strings.Repeat("b", 64),
		ImpactHash:              strings.Repeat("c", 64),
		RiskHash:                strings.Repeat("d", 64),
		AcceptanceHash:          strings.Repeat("e", 64),
	}
}

func clonePlanningEvidenceHashMap(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func planningEvidenceScopeFixture() planningEvidenceScope {
	return planningEvidenceScope{
		GoalID:                  "goal-200",
		SessionID:               "session-200",
		SpecificationRevisionID: "spec-rev-1",
		PlanRevisionID:          "plan-rev-1",
	}
}

func planningEvidenceRecordIDs(records []planningEvidenceRecord) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.Reference.ID)
	}
	return ids
}
