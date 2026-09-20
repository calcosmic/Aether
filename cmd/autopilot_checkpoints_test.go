package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func checkpointTestDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func checkpointTestExecutionBinding(suffix string) codex.ExecutionBinding {
	attemptID := "attempt-checkpoint-" + suffix
	return codex.ExecutionBinding{
		SchemaVersion:        codex.ExecutionBindingSchemaVersion,
		RunID:                "run-checkpoint-" + suffix,
		AttemptID:            attemptID,
		ManifestSHA256:       checkpointTestDigest("manifest-" + suffix),
		WorkspaceFingerprint: checkpointTestDigest("workspace-" + suffix),
		ExecutionOwner:       "checkpoint-test-owner",
	}
}

func checkpointTestGeneration(t *testing.T, suffix, evidence string) autopilotCheckpointGeneration {
	t.Helper()
	binding := checkpointTestExecutionBinding(suffix)
	generation, err := newAutopilotCheckpointGeneration(binding.AttemptID, &binding, checkpointTestDigest(evidence))
	if err != nil {
		t.Fatalf("create checkpoint generation fixture: %v", err)
	}
	return generation
}

func checkpointRuntimeManifest(t *testing.T, phaseID int, suffix string) codexBuildManifest {
	t.Helper()
	binding := checkpointTestExecutionBinding(suffix)
	manifest := codexBuildManifest{
		Phase:            phaseID,
		AttemptID:        binding.AttemptID,
		ExecutionOwner:   binding.ExecutionOwner,
		ExecutionBinding: &binding,
	}
	digest, err := buildManifestSHA256(manifest)
	if err != nil {
		t.Fatalf("hash runtime checkpoint manifest fixture: %v", err)
	}
	binding.ManifestSHA256 = digest
	manifest.ExecutionBinding = &binding
	return manifest
}

type visualCheckpointTestFixture struct {
	BuildResult map[string]interface{}
	AttemptRel  string
	ClaimsRel   string
}

func checkpointVisualFixture(t *testing.T, phaseID int, suffix string, claims codexBuildClaims) visualCheckpointTestFixture {
	t.Helper()
	claims.BuildPhase = phaseID
	claimsRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseID), suffix+"-claims.json"))
	if err := store.SaveJSON(claimsRel, claims); err != nil {
		t.Fatalf("save visual checkpoint claims: %v", err)
	}

	binding := checkpointTestExecutionBinding(suffix)
	attemptRel := buildAttemptPathForID(phaseID, binding.AttemptID)
	manifest := codexBuildManifest{
		Phase:            phaseID,
		AttemptID:        binding.AttemptID,
		AttemptPath:      displayDataPath(attemptRel),
		ClaimsPath:       displayDataPath(claimsRel),
		ExecutionOwner:   binding.ExecutionOwner,
		ExecutionBinding: &binding,
	}
	manifestDigest, err := buildManifestSHA256(manifest)
	if err != nil {
		t.Fatalf("hash visual checkpoint manifest: %v", err)
	}
	binding.ManifestSHA256 = manifestDigest
	manifest.ExecutionBinding = &binding
	record := buildAttemptRecord{
		SchemaVersion:   buildAttemptSchemaVersion,
		ID:              binding.AttemptID,
		Phase:           phaseID,
		Status:          buildAttemptBuilt,
		ExecutionOwner:  binding.ExecutionOwner,
		RunID:           binding.RunID,
		WorkspaceSHA256: binding.WorkspaceFingerprint,
		ManifestSHA256:  binding.ManifestSHA256,
		ClaimsPath:      displayDataPath(claimsRel),
		PlanManifest:    &manifest,
		Claims:          &claims,
	}
	if err := store.SaveJSON(attemptRel, record); err != nil {
		t.Fatalf("save visual checkpoint attempt: %v", err)
	}
	return visualCheckpointTestFixture{
		AttemptRel: attemptRel,
		ClaimsRel:  claimsRel,
		BuildResult: map[string]interface{}{
			"attempt":     displayDataPath(attemptRel),
			"claims_path": displayDataPath(claimsRel),
		},
	}
}

func checkpointTestState(t *testing.T, phase colony.Phase, stateValue colony.State) colony.ColonyState {
	t.Helper()
	goal := "Durable owner checkpoints"
	session := "checkpoint-test-session"
	now := time.Now().UTC()
	state := colony.ColonyState{
		Goal:           &goal,
		SessionID:      &session,
		State:          stateValue,
		CurrentPhase:   phase.ID,
		BuildStartedAt: &now,
		Plan:           colony.Plan{Phases: []colony.Phase{phase}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}
	return state
}

func loadCheckpointDecisions(t *testing.T) []PendingDecision {
	t.Helper()
	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		t.Fatalf("load checkpoint decisions: %v", err)
	}
	return file.Decisions
}

func unresolvedCheckpoints(decisions []PendingDecision) []PendingDecision {
	var result []PendingDecision
	for _, decision := range decisions {
		if isAutopilotCheckpointType(decision.Type) && !decision.Resolved {
			result = append(result, decision)
		}
	}
	return result
}

func checkpointCapabilityFromReference(t *testing.T, ref autopilotCheckpointReference) string {
	t.Helper()
	const marker = "--checkpoint-capability '"
	start := strings.LastIndex(ref.RecoveryCommand, marker)
	if start < 0 {
		t.Fatalf("checkpoint recovery command has no capability: %q", ref.RecoveryCommand)
	}
	capability := strings.TrimSuffix(ref.RecoveryCommand[start+len(marker):], "'")
	if strings.TrimSpace(capability) == "" {
		t.Fatalf("checkpoint recovery command has an empty capability: %q", ref.RecoveryCommand)
	}
	return capability
}

func persistedCheckpointCapabilityHashes(decision PendingDecision) []string {
	hashes := []string{}
	if hash := strings.TrimSpace(decision.CheckpointCapabilitySHA256); hash != "" {
		hashes = append(hashes, hash)
	}
	for _, hash := range decision.CheckpointCapabilitySHA256s {
		if hash = strings.TrimSpace(hash); hash != "" {
			hashes = append(hashes, hash)
		}
	}
	return hashes
}

func pendingDecisionBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(store.BasePath(), pendingDecisionsFile))
	if err != nil {
		t.Fatalf("read pending decisions: %v", err)
	}
	return data
}

func TestVisualCheckpointPathClassifier(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"web/components/StatusCard.tsx", true},
		{"web/styles/checkpoint.css", true},
		{"templates/dashboard.html", true},
		{"web/components/status-card.test.tsx", false},
		{"web/__tests__/dashboard.jsx", false},
		{"docs/dashboard.md", false},
		{"cmd/autopilot.go", false},
		{"../outside/app.tsx", false},
		{"/tmp/forged.css", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isUICheckpointPath(tt.path); got != tt.want {
				t.Fatalf("isUICheckpointPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestVisualCheckpointMaterializesFromPersistedClaims(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 2, Name: "UI result", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)

	claims := codexBuildClaims{
		BuildPhase:    phase.ID,
		FilesCreated:  []string{"web/components/StatusCard.tsx", "../outside/forged.tsx"},
		FilesModified: []string{"web/styles/status.css", "cmd/status.go", "docs/status.md", "web/components/status-card.test.tsx"},
		TestsWritten:  []string{"web/components/status-card.test.tsx"},
	}
	fixture := checkpointVisualFixture(t, phase.ID, "persisted-claims", claims)
	fixture.BuildResult["ui_touched"] = false
	refs, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, fixture.BuildResult)
	delete(fixture.BuildResult, "ui_touched")
	if err != nil {
		t.Fatalf("materialize visual checkpoint: %v", err)
	}
	if len(refs) != 1 || refs[0].Type != autopilotCheckpointTypeVisual {
		t.Fatalf("visual references = %#v, want one visual checkpoint", refs)
	}
	decisions := unresolvedCheckpoints(loadCheckpointDecisions(t))
	if len(decisions) != 1 {
		t.Fatalf("pending checkpoints = %d, want 1: %#v", len(decisions), decisions)
	}
	wantPaths := []string{"web/components/StatusCard.tsx", "web/styles/status.css"}
	if !reflect.DeepEqual(decisions[0].SourcePaths, wantPaths) {
		t.Fatalf("visual evidence paths = %#v, want %#v", decisions[0].SourcePaths, wantPaths)
	}
	if decisions[0].CheckpointAttemptID == "" || decisions[0].CheckpointExecutionBindingSHA256 == "" || decisions[0].CheckpointEvidenceSHA256 == "" || decisions[0].WorkGeneration == "" {
		t.Fatalf("visual checkpoint omitted generation provenance: %#v", decisions[0])
	}
	if decisions[0].CheckpointCompatibilityKey == "" || decisions[0].CheckpointCompatibilityKey == decisions[0].CheckpointKey {
		t.Fatalf("visual checkpoint did not separate compatibility and generation keys: %#v", decisions[0])
	}

	second, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, fixture.BuildResult)
	if err != nil {
		t.Fatalf("replay visual checkpoint: %v", err)
	}
	if len(second) != 1 || second[0].ID != refs[0].ID {
		t.Fatalf("visual replay changed checkpoint identity: first=%#v second=%#v", refs, second)
	}
	if got := len(unresolvedCheckpoints(loadCheckpointDecisions(t))); got != 1 {
		t.Fatalf("visual replay duplicated pending decisions: got %d", got)
	}

	// Unknown JSON fields such as ui_touched are ignored by the claims type;
	// with no created/modified UI path they cannot manufacture a decision.
	boolOnly := checkpointVisualFixture(t, 3, "boolean-only", codexBuildClaims{})
	boolOnly.BuildResult["ui_touched"] = true
	if got, err := materializeVisualCheckpointFromBuildResult(root, 3, boolOnly.BuildResult); err != nil || len(got) != 0 {
		t.Fatalf("worker boolean manufactured visual work: refs=%#v err=%v", got, err)
	}
}

func TestVisualCheckpointWorkGenerationRequiresFreshApproval(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 2, Name: "Fresh visual approval", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	claims := codexBuildClaims{FilesModified: []string{"web/components/StatusCard.tsx"}}

	generationA := checkpointVisualFixture(t, phase.ID, "generation-a", claims)
	first, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, generationA.BuildResult)
	if err != nil || len(first) != 1 {
		t.Fatalf("materialize generation A: refs=%#v err=%v", first, err)
	}
	replayed, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, generationA.BuildResult)
	if err != nil || len(replayed) != 1 || replayed[0].ID != first[0].ID {
		t.Fatalf("exact generation A replay drifted: first=%#v replay=%#v err=%v", first, replayed, err)
	}
	if got := len(loadCheckpointDecisions(t)); got != 1 {
		t.Fatalf("exact generation A replay created %d rows, want 1", got)
	}
	if _, found, err := resolveAutopilotCheckpointPendingDecision(
		first[0].Question,
		"generation A looks correct",
		phase.ID,
		checkpointCapabilityFromReference(t, first[0]),
	); err != nil || !found {
		t.Fatalf("resolve generation A: found=%v err=%v", found, err)
	}
	if exactResolvedReplay, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, generationA.BuildResult); err != nil || len(exactResolvedReplay) != 0 {
		t.Fatalf("resolved exact replay reopened owner work: refs=%#v err=%v", exactResolvedReplay, err)
	}

	generationB := checkpointVisualFixture(t, phase.ID, "generation-b", claims)
	changedAttempt, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, generationB.BuildResult)
	if err != nil || len(changedAttempt) != 1 {
		t.Fatalf("materialize changed attempt: refs=%#v err=%v", changedAttempt, err)
	}
	if changedAttempt[0].ID == first[0].ID {
		t.Fatalf("changed attempt inherited generation A row: A=%s B=%s", first[0].ID, changedAttempt[0].ID)
	}

	changedClaims := codexBuildClaims{
		BuildPhase:    phase.ID,
		FilesModified: []string{"web/components/StatusCard.tsx", "web/styles/status.css"},
	}
	if err := store.SaveJSON(generationB.ClaimsRel, changedClaims); err != nil {
		t.Fatalf("replace generation B claims bytes: %v", err)
	}
	beforeTamperCheck := snapshotProjectDataTree(t, store.BasePath())
	changedEvidence, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, generationB.BuildResult)
	if err == nil || len(changedEvidence) != 0 {
		t.Fatalf("post-build claims tampering was accepted: refs=%#v err=%v", changedEvidence, err)
	}
	if afterTamperCheck := snapshotProjectDataTree(t, store.BasePath()); !reflect.DeepEqual(beforeTamperCheck, afterTamperCheck) {
		t.Fatalf("post-build claims rejection mutated durable state\nbefore: %#v\nafter:  %#v", beforeTamperCheck, afterTamperCheck)
	}

	decisions := loadCheckpointDecisions(t)
	if len(decisions) != 2 || !decisions[0].Resolved || decisions[1].Resolved {
		t.Fatalf("generation rows = %#v, want one resolved and one fresh unresolved row", decisions)
	}
	wantCompatibility := decisions[0].CheckpointCompatibilityKey
	seenRows := map[string]bool{}
	seenGenerations := map[string]bool{}
	for _, decision := range decisions {
		if wantCompatibility == "" || decision.CheckpointCompatibilityKey != wantCompatibility {
			t.Fatalf("generation changed compatibility identity: %#v", decisions)
		}
		if decision.CheckpointKey == "" || seenRows[decision.CheckpointKey] {
			t.Fatalf("generation row identity was empty or reused: %#v", decisions)
		}
		if decision.WorkGeneration == "" || seenGenerations[decision.WorkGeneration] {
			t.Fatalf("work generation was empty or reused: %#v", decisions)
		}
		seenRows[decision.CheckpointKey] = true
		seenGenerations[decision.WorkGeneration] = true
	}
}

func TestVisualCheckpointRejectsPostBuildClaimsTamperingWithoutMutation(t *testing.T) {
	tests := []struct {
		name     string
		terminal codexBuildClaims
		tampered codexBuildClaims
	}{
		{
			name:     "removing terminal UI work cannot suppress owner review",
			terminal: codexBuildClaims{FilesModified: []string{"web/components/StatusCard.tsx"}},
			tampered: codexBuildClaims{FilesModified: []string{"cmd/status.go"}},
		},
		{
			name:     "inventing UI work cannot manufacture owner review",
			terminal: codexBuildClaims{FilesModified: []string{"cmd/status.go"}},
			tampered: codexBuildClaims{FilesModified: []string{"web/components/ForgedCard.tsx"}},
		},
		{
			name:     "replacing terminal UI paths cannot mint a generation",
			terminal: codexBuildClaims{FilesModified: []string{"web/components/StatusCard.tsx"}},
			tampered: codexBuildClaims{FilesModified: []string{"web/styles/forged.css"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			s, root := newTestStore(t)
			store = s
			phase := colony.Phase{ID: 4, Name: "Immutable visual claims", Status: colony.PhaseInProgress}
			checkpointTestState(t, phase, colony.StateBUILT)
			fixture := checkpointVisualFixture(t, phase.ID, strings.ReplaceAll(tt.name, " ", "-"), tt.terminal)
			tampered := tt.tampered
			tampered.BuildPhase = phase.ID
			if err := store.SaveJSON(fixture.ClaimsRel, tampered); err != nil {
				t.Fatalf("tamper persisted claims: %v", err)
			}

			before := snapshotProjectDataTree(t, store.BasePath())
			refs, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, fixture.BuildResult)
			if err == nil || len(refs) != 0 {
				t.Fatalf("tampered visual claims were accepted: refs=%#v err=%v", refs, err)
			}
			if !strings.Contains(strings.ToLower(err.Error()), "terminal claims") {
				t.Fatalf("tamper rejection did not name terminal claims: %v", err)
			}
			after := snapshotProjectDataTree(t, store.BasePath())
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("tamper rejection mutated durable state\nbefore: %#v\nafter:  %#v", before, after)
			}
		})
	}
}

func TestVisualCheckpointRejectsMissingOrMismatchedGenerationEvidence(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(fixture visualCheckpointTestFixture) (int, map[string]interface{})
	}{
		{
			name: "missing attempt path",
			mutate: func(fixture visualCheckpointTestFixture) (int, map[string]interface{}) {
				delete(fixture.BuildResult, "attempt")
				return 5, fixture.BuildResult
			},
		},
		{
			name: "missing claims path",
			mutate: func(fixture visualCheckpointTestFixture) (int, map[string]interface{}) {
				delete(fixture.BuildResult, "claims_path")
				return 5, fixture.BuildResult
			},
		},
		{
			name: "phase mismatch",
			mutate: func(fixture visualCheckpointTestFixture) (int, map[string]interface{}) {
				return 6, fixture.BuildResult
			},
		},
		{
			name: "attempt claims path mismatch",
			mutate: func(fixture visualCheckpointTestFixture) (int, map[string]interface{}) {
				fixture.BuildResult["claims_path"] = displayDataPath("build/phase-5/other-claims.json")
				return 5, fixture.BuildResult
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobals(t)
			s, root := newTestStore(t)
			store = s
			phase := colony.Phase{ID: 5, Name: "Reject stale visual provenance", Status: colony.PhaseInProgress}
			checkpointTestState(t, phase, colony.StateBUILT)
			fixture := checkpointVisualFixture(t, phase.ID, strings.ReplaceAll(tt.name, " ", "-"), codexBuildClaims{
				FilesModified: []string{"web/components/StatusCard.tsx"},
			})
			phaseID, result := tt.mutate(fixture)
			if refs, err := materializeVisualCheckpointFromBuildResult(root, phaseID, result); err == nil {
				t.Fatalf("mismatched generation materialized refs=%#v", refs)
			}
			if _, err := os.Stat(filepath.Join(store.BasePath(), pendingDecisionsFile)); !os.IsNotExist(err) {
				t.Fatalf("mismatched generation mutated %s: stat err=%v", pendingDecisionsFile, err)
			}
		})
	}
}

func TestRuntimeVerificationDecisionIsIdempotent(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Runtime owner checks", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	criteria := []codexCriterionVerification{
		{TaskID: "1.1", Criterion: "VoiceOver order feels natural", State: criterionStateNeedsOwnerConfirmation, Evidence: []string{"automated accessibility probe cannot judge reading flow"}},
		{TaskID: "1.2", Criterion: "Animation timing feels calm", State: criterionStateNeedsOwnerConfirmation, Evidence: []string{"requires hands-on playback"}},
		{TaskID: "1.3", Criterion: "Unit tests pass", State: "", Evidence: []string{"go test passed"}},
	}
	generation := checkpointTestGeneration(t, "runtime-idempotent", "runtime-idempotent-evidence")

	first, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generation)
	if err != nil {
		t.Fatalf("materialize runtime checkpoints: %v", err)
	}
	second, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generation)
	if err != nil {
		t.Fatalf("replay runtime checkpoints: %v", err)
	}
	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("runtime checkpoint replay drifted: first=%#v second=%#v", first, second)
	}
	for i := range first {
		if first[i].ID != second[i].ID || first[i].Type != second[i].Type || first[i].Question != second[i].Question {
			t.Fatalf("runtime checkpoint replay changed stable identity: first=%#v second=%#v", first[i], second[i])
		}
		if firstCapability, secondCapability := checkpointCapabilityFromReference(t, first[i]), checkpointCapabilityFromReference(t, second[i]); firstCapability == secondCapability {
			t.Fatalf("runtime checkpoint replay reused a capability for %s", first[i].ID)
		}
	}
	if got := len(unresolvedCheckpoints(loadCheckpointDecisions(t))); got != 2 {
		t.Fatalf("runtime replay created %d unresolved decisions, want 2", got)
	}
	signals := continueReviewAutopilotSignals(nil, first)
	if !reflect.DeepEqual(signals.Checkpoints, first) {
		t.Fatalf("continue autopilot signals omitted decision IDs/types: got %#v want %#v", signals.Checkpoints, first)
	}
	runtimeActive := false
	for _, evaluation := range signals.Evaluations {
		if evaluation.Spec.Code == autopilotTriggerRuntimeVerificationNeeded {
			runtimeActive = evaluation.Active
		}
	}
	if !runtimeActive {
		t.Fatal("runtime-verification decisions did not activate the typed autopilot signal")
	}
}

func TestRuntimeCheckpointWorkGenerationRequiresFreshApproval(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 2, Name: "Runtime generation freshness", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	criteria := []codexCriterionVerification{{
		TaskID: "2.1", Criterion: "The playback feels natural", State: criterionStateNeedsOwnerConfirmation,
		Policy: criterionEvidencePolicyBoundV1, Enforced: true, Passed: true,
		Evidence:          []string{"owner must judge pacing", "automated checks passed"},
		RequiredArtifacts: []string{"ui/player.tsx", "ui/timeline.tsx"},
		Summary:           "No deterministic check can judge the final feel.",
	}}
	manifestA := checkpointRuntimeManifest(t, phase.ID, "runtime-generation-a")
	generationA, err := runtimeCheckpointGenerationFromManifest(manifestA, criteria)
	if err != nil {
		t.Fatalf("derive generation A: %v", err)
	}
	first, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generationA)
	if err != nil || len(first) != 1 {
		t.Fatalf("materialize generation A: refs=%#v err=%v", first, err)
	}
	capabilityA := checkpointCapabilityFromReference(t, first[0])
	if _, found, resolveErr := resolveAutopilotCheckpointPendingDecision(first[0].Question, "confirmed", phase.ID, capabilityA); resolveErr != nil || !found {
		t.Fatalf("resolve generation A: found=%v err=%v", found, resolveErr)
	}

	reordered := append([]codexCriterionVerification(nil), criteria...)
	reordered[0].Evidence = []string{"automated checks passed", "owner must judge pacing", "automated checks passed"}
	reordered[0].RequiredArtifacts = []string{"ui/timeline.tsx", "ui/player.tsx", "ui/player.tsx"}
	replayGeneration, err := runtimeCheckpointGenerationFromManifest(manifestA, reordered)
	if err != nil {
		t.Fatalf("derive exact replay generation: %v", err)
	}
	if replayGeneration.WorkGeneration != generationA.WorkGeneration {
		t.Fatalf("set/order-only changes altered generation: first=%s replay=%s", generationA.WorkGeneration, replayGeneration.WorkGeneration)
	}
	replayed, err := materializeRuntimeVerificationCheckpoints(phase.ID, reordered, replayGeneration)
	if err != nil || len(replayed) != 0 {
		t.Fatalf("resolved exact replay reopened owner work: refs=%#v err=%v", replayed, err)
	}

	manifestB := checkpointRuntimeManifest(t, phase.ID, "runtime-generation-b")
	generationB, err := runtimeCheckpointGenerationFromManifest(manifestB, criteria)
	if err != nil {
		t.Fatalf("derive changed-attempt generation: %v", err)
	}
	second, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generationB)
	if err != nil || len(second) != 1 || second[0].ID == first[0].ID {
		t.Fatalf("changed attempt did not create fresh owner work: first=%#v second=%#v err=%v", first, second, err)
	}

	changedEvidence := append([]codexCriterionVerification(nil), criteria...)
	changedEvidence[0].Evidence = append(append([]string(nil), criteria[0].Evidence...), "owner observed a timing regression")
	changedGeneration, err := runtimeCheckpointGenerationFromManifest(manifestA, changedEvidence)
	if err != nil {
		t.Fatalf("derive changed-evidence generation: %v", err)
	}
	third, err := materializeRuntimeVerificationCheckpoints(phase.ID, changedEvidence, changedGeneration)
	if err != nil || len(third) != 1 || third[0].ID == first[0].ID || third[0].ID == second[0].ID {
		t.Fatalf("changed verification evidence did not create fresh owner work: first=%#v second=%#v third=%#v err=%v", first, second, third, err)
	}
	decisions := loadCheckpointDecisions(t)
	if len(decisions) != 3 || !decisions[0].Resolved || decisions[1].Resolved || decisions[2].Resolved {
		t.Fatalf("runtime generations did not preserve resolved history plus fresh unresolved rows: %#v", decisions)
	}
}

func TestRuntimeCheckpointGenerationDirectExternalParity(t *testing.T) {
	manifest := checkpointRuntimeManifest(t, 7, "runtime-parity")
	directCriteria := []codexCriterionVerification{
		{
			TaskID: "7.2", Criterion: "The fallback feels understandable", State: criterionStateNeedsOwnerConfirmation,
			Policy: criterionEvidencePolicyBoundV1, Enforced: true, Passed: true,
			RequiredArtifacts: []string{"ui/fallback.tsx", "ui/shared.tsx"}, RequiredChecks: []string{"watcher", "tests"},
			Evidence: []string{"manual judgment remains", "tests passed"}, BlockingIssues: []string{"reviewer unavailable"}, Summary: "Owner review is required.",
		},
		{
			TaskID: "7.1", Criterion: "The primary flow passes", Policy: criterionEvidencePolicyBoundV1,
			Enforced: true, Passed: true, RequiredArtifacts: []string{"ui/primary.tsx"}, Evidence: []string{"go test passed"}, Summary: "Deterministic proof recorded.",
		},
	}
	direct, err := runtimeCheckpointGenerationFromManifest(manifest, directCriteria)
	if err != nil {
		t.Fatalf("derive direct generation: %v", err)
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal external manifest fixture: %v", err)
	}
	criteriaBytes, err := json.Marshal(directCriteria)
	if err != nil {
		t.Fatalf("marshal external criteria fixture: %v", err)
	}
	var externalManifest codexBuildManifest
	var externalCriteria []codexCriterionVerification
	if err := json.Unmarshal(manifestBytes, &externalManifest); err != nil {
		t.Fatalf("decode external manifest fixture: %v", err)
	}
	if err := json.Unmarshal(criteriaBytes, &externalCriteria); err != nil {
		t.Fatalf("decode external criteria fixture: %v", err)
	}
	externalCriteria[0], externalCriteria[1] = externalCriteria[1], externalCriteria[0]
	externalCriteria[1].Evidence = []string{"tests passed", "manual judgment remains", "tests passed"}
	externalCriteria[1].RequiredArtifacts = []string{"ui/shared.tsx", "ui/fallback.tsx"}
	externalCriteria[1].RequiredChecks = []string{"tests", "watcher"}
	external, err := runtimeCheckpointGenerationFromManifest(externalManifest, externalCriteria)
	if err != nil {
		t.Fatalf("derive external generation: %v", err)
	}
	if external.WorkGeneration != direct.WorkGeneration || external.EvidenceSHA256 != direct.EvidenceSHA256 {
		t.Fatalf("direct/external generation drifted: direct=%#v external=%#v", direct, external)
	}

	mutations := []struct {
		name   string
		mutate func([]codexCriterionVerification)
	}{
		{name: "status", mutate: func(criteria []codexCriterionVerification) { criteria[0].State = "reviewed" }},
		{name: "result", mutate: func(criteria []codexCriterionVerification) { criteria[0].Passed = false }},
		{name: "evidence", mutate: func(criteria []codexCriterionVerification) {
			criteria[0].Evidence = append(criteria[0].Evidence, "new observation")
		}},
		{name: "source paths", mutate: func(criteria []codexCriterionVerification) {
			criteria[0].RequiredArtifacts = append(criteria[0].RequiredArtifacts, "ui/new.tsx")
		}},
	}
	for _, tt := range mutations {
		t.Run(tt.name, func(t *testing.T) {
			changed := append([]codexCriterionVerification(nil), directCriteria...)
			changed[0].Evidence = append([]string(nil), directCriteria[0].Evidence...)
			changed[0].RequiredArtifacts = append([]string(nil), directCriteria[0].RequiredArtifacts...)
			tt.mutate(changed)
			generation, generationErr := runtimeCheckpointGenerationFromManifest(manifest, changed)
			if generationErr != nil {
				t.Fatalf("derive changed generation: %v", generationErr)
			}
			if generation.EvidenceSHA256 == direct.EvidenceSHA256 || generation.WorkGeneration == direct.WorkGeneration {
				t.Fatalf("%s change did not alter runtime generation", tt.name)
			}
		})
	}
}

func TestRuntimeCheckpointGenerationAcceptsJournalBoundDirectFinalProjection(t *testing.T) {
	saveGlobals(t)
	startedAt := time.Now().UTC().Truncate(time.Second)
	claimsRel := filepath.ToSlash(filepath.Join("build", "phase-1", "claims.json"))
	fixture := commitTestBuildStart(t, testBuildStartOptions{
		GeneratedAt: startedAt, ExecutionOwner: "go-runtime", ClaimsPath: claimsRel,
		SelectedTasks: []string{"1.1"},
		Dispatches:    []codexBuildDispatch{{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-checkpoint", TaskID: "1.1", Status: "planned"}},
		MakeLatest:    testBuildStartBool(true),
	})
	phase, attemptRel := fixture.Phase, fixture.AttemptPath
	state := fixture.State
	state.State = colony.StateBUILT
	bound := *fixture.Manifest
	manifestRel := fixture.Request.Effects.ManifestPath
	if err := transitionBuildAttempt(attemptRel, buildAttemptBuilt, "fixture built", nil, &codexBuildClaims{BuildPhase: phase.ID}, "real", nil); err != nil {
		t.Fatalf("complete direct build attempt fixture: %v", err)
	}

	finalProjection := bound
	finalProjection.AttemptID = ""
	finalProjection.AttemptPath = ""
	finalProjection.ExecutionBinding = nil
	criteria := []codexCriterionVerification{{
		TaskID: "1.1", Criterion: "The owner confirms the result", State: criterionStateNeedsOwnerConfirmation, Passed: true,
	}}
	generation, err := validatedRuntimeCheckpointGeneration(codexContinueManifest{Present: true, Path: manifestRel, Data: finalProjection}, state, criteria)
	if err != nil {
		t.Fatalf("derive journal-backed direct generation: %v", err)
	}
	want, err := runtimeCheckpointGenerationFromManifest(bound, criteria)
	if err != nil {
		t.Fatalf("derive bound generation fixture: %v", err)
	}
	if generation.WorkGeneration != want.WorkGeneration {
		t.Fatalf("final projection generation = %s, durable bound generation = %s", generation.WorkGeneration, want.WorkGeneration)
	}

	tampered := finalProjection
	tampered.ClaimsPath = displayDataPath("build/phase-4/tampered-claims.json")
	if _, err := validatedRuntimeCheckpointGeneration(codexContinueManifest{Present: true, Path: manifestRel, Data: tampered}, state, criteria); err == nil {
		t.Fatal("tampered direct final projection inherited durable attempt provenance")
	}
	if _, statErr := os.Stat(filepath.Join(store.BasePath(), pendingDecisionsFile)); !os.IsNotExist(statErr) {
		t.Fatalf("rejected final projection mutated pending decisions: stat err=%v", statErr)
	}
}

func TestCheckpointCapabilityRotatesWithoutChangingIdentityOrPersistingRawTokens(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 4, Name: "Capability rotation", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	criterion := codexCriterionVerification{
		TaskID: "4.1", Criterion: "The interaction feels deliberate", State: criterionStateNeedsOwnerConfirmation,
	}
	generation := checkpointTestGeneration(t, "capability-rotation", "capability-rotation-evidence")

	first, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{criterion}, generation)
	if err != nil || len(first) != 1 {
		t.Fatalf("materialize first checkpoint: refs=%#v err=%v", first, err)
	}
	firstCapability := checkpointCapabilityFromReference(t, first[0])
	firstDecision := loadCheckpointDecisions(t)[0]
	firstKey := firstDecision.CheckpointKey
	if firstDecision.ID == "" || firstKey == "" {
		t.Fatalf("checkpoint has no stable identity: %#v", firstDecision)
	}
	if firstDecision.CheckpointCapability != "" {
		t.Fatalf("raw capability survived a persistence round trip: %q", firstDecision.CheckpointCapability)
	}
	if hashes := persistedCheckpointCapabilityHashes(firstDecision); len(hashes) != 1 {
		t.Fatalf("first checkpoint persisted %d capability hashes, want 1: %#v", len(hashes), hashes)
	}
	raw := pendingDecisionBytes(t)
	if bytes.Contains(raw, []byte(firstCapability)) {
		t.Fatalf("pending-decisions.json persisted the raw checkpoint capability: %s", raw)
	}
	if !bytes.Contains(raw, []byte("checkpoint_capability_sha256")) {
		t.Fatalf("pending-decisions.json did not persist a checkpoint capability hash: %s", raw)
	}
	field, ok := reflect.TypeOf(PendingDecision{}).FieldByName("CheckpointCapability")
	if !ok || field.Tag.Get("json") != "-" {
		t.Fatalf("CheckpointCapability must be transient with json:\"-\"; field=%#v", field)
	}

	second, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{criterion}, generation)
	if err != nil || len(second) != 1 {
		t.Fatalf("rotate checkpoint capability: refs=%#v err=%v", second, err)
	}
	secondCapability := checkpointCapabilityFromReference(t, second[0])
	if secondCapability == firstCapability {
		t.Fatal("capability rotation reused the raw token")
	}
	secondDecision := loadCheckpointDecisions(t)[0]
	if secondDecision.ID != firstDecision.ID || secondDecision.CheckpointKey != firstKey {
		t.Fatalf("capability rotation changed checkpoint identity: first=%#v second=%#v", firstDecision, secondDecision)
	}
	hashes := persistedCheckpointCapabilityHashes(secondDecision)
	if len(hashes) != 2 || hashes[0] == hashes[1] {
		t.Fatalf("capability rotation hashes = %#v, want two distinct durable hashes", hashes)
	}
	raw = pendingDecisionBytes(t)
	if bytes.Contains(raw, []byte(firstCapability)) || bytes.Contains(raw, []byte(secondCapability)) {
		t.Fatalf("pending-decisions.json persisted a raw capability after rotation: %s", raw)
	}
	if resolved, found, err := resolveAutopilotCheckpointPendingDecision(
		second[0].Question, "owner used the first displayed command", phase.ID, firstCapability,
	); err != nil || !found || resolved.ID != firstDecision.ID {
		t.Fatalf("rotation invalidated the first displayed capability: resolved=%#v found=%v err=%v", resolved, found, err)
	}
}

func TestAutopilotCheckpointSealStorageErrorsFailClosed(t *testing.T) {
	t.Run("absent pending decisions is the only empty success", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		state := checkpointTestState(t, colony.Phase{ID: 1, Name: "No owner work", Status: colony.PhaseCompleted}, colony.StateCOMPLETED)

		blockers, err := autopilotCheckpointSealBlockers(state)
		if err != nil || len(blockers) != 0 {
			t.Fatalf("absent pending decisions returned blockers=%#v err=%v, want empty success", blockers, err)
		}
	})

	t.Run("malformed pending decisions", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		state := checkpointTestState(t, colony.Phase{ID: 1, Name: "Corrupt owner work", Status: colony.PhaseCompleted}, colony.StateCOMPLETED)
		if err := os.WriteFile(filepath.Join(store.BasePath(), pendingDecisionsFile), []byte("{not-json"), 0o644); err != nil {
			t.Fatalf("seed malformed pending decisions: %v", err)
		}

		blockers, err := autopilotCheckpointSealBlockers(state)
		if err == nil || len(blockers) != 0 {
			t.Fatalf("malformed pending decisions returned blockers=%#v err=%v, want explicit error and no blockers", blockers, err)
		}
	})

	t.Run("directory at pending decisions path", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		state := checkpointTestState(t, colony.Phase{ID: 1, Name: "Directory owner work", Status: colony.PhaseCompleted}, colony.StateCOMPLETED)
		if err := os.Mkdir(filepath.Join(store.BasePath(), pendingDecisionsFile), 0o755); err != nil {
			t.Fatalf("seed directory-backed pending decisions: %v", err)
		}

		blockers, err := autopilotCheckpointSealBlockers(state)
		if err == nil || len(blockers) != 0 {
			t.Fatalf("directory-backed pending decisions returned blockers=%#v err=%v, want explicit error and no blockers", blockers, err)
		}
	})

	t.Run("capability hash cannot be persisted", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		phase := colony.Phase{ID: 1, Name: "Unwritable owner work", Status: colony.PhaseCompleted}
		state := checkpointTestState(t, phase, colony.StateCOMPLETED)
		refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{{
			TaskID: "1.1", Criterion: "The owner experience feels right", State: criterionStateNeedsOwnerConfirmation,
		}}, checkpointTestGeneration(t, "unwritable-owner-work", "unwritable-owner-work-evidence"))
		if err != nil || len(refs) != 1 {
			t.Fatalf("seed checkpoint: refs=%#v err=%v", refs, err)
		}
		before := append([]byte(nil), pendingDecisionBytes(t)...)
		if err := os.Chmod(store.BasePath(), 0o555); err != nil {
			t.Fatalf("make data directory unwritable: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(store.BasePath(), 0o755) })

		blockers, err := autopilotCheckpointSealBlockers(state)
		if err == nil || len(blockers) != 0 {
			t.Fatalf("unwritable pending decisions returned blockers=%#v err=%v, want explicit error and no authorization command", blockers, err)
		}
		if err := os.Chmod(store.BasePath(), 0o755); err != nil {
			t.Fatalf("restore data directory permissions: %v", err)
		}
		if after := pendingDecisionBytes(t); !bytes.Equal(before, after) {
			t.Fatalf("failed capability write changed pending decisions:\nbefore=%s\nafter=%s", before, after)
		}
	})
}

func TestAutopilotCheckpointSealCapabilityIsFreshStableAndTransient(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 6, Name: "Seal capability", Status: colony.PhaseCompleted}
	state := checkpointTestState(t, phase, colony.StateCOMPLETED)
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{{
		TaskID: "6.1", Criterion: "The final interaction feels right", State: criterionStateNeedsOwnerConfirmation,
	}}, checkpointTestGeneration(t, "seal-capability", "seal-capability-evidence"))
	if err != nil || len(refs) != 1 {
		t.Fatalf("seed checkpoint: refs=%#v err=%v", refs, err)
	}
	initialCapability := checkpointCapabilityFromReference(t, refs[0])
	before := loadCheckpointDecisions(t)[0]

	first, err := autopilotCheckpointSealBlockers(state)
	if err != nil || len(first) != 1 {
		t.Fatalf("first seal blocker load: blockers=%#v err=%v", first, err)
	}
	second, err := autopilotCheckpointSealBlockers(state)
	if err != nil || len(second) != 1 {
		t.Fatalf("second seal blocker load: blockers=%#v err=%v", second, err)
	}
	firstCapability := checkpointCapabilityFromReference(t, autopilotCheckpointReference{RecoveryCommand: first[0].RecoveryCommand})
	secondCapability := checkpointCapabilityFromReference(t, autopilotCheckpointReference{RecoveryCommand: second[0].RecoveryCommand})
	if firstCapability == initialCapability || secondCapability == initialCapability || secondCapability == firstCapability {
		t.Fatalf("seal did not issue a fresh capability each time: initial=%q first=%q second=%q", initialCapability, firstCapability, secondCapability)
	}
	after := loadCheckpointDecisions(t)[0]
	if first[0].ID != before.ID || second[0].ID != before.ID || after.ID != before.ID || after.CheckpointKey != before.CheckpointKey {
		t.Fatalf("seal capability rotation changed stable identity: before=%#v after=%#v first=%#v second=%#v", before, after, first[0], second[0])
	}
	raw := pendingDecisionBytes(t)
	for _, capability := range []string{initialCapability, firstCapability, secondCapability} {
		if bytes.Contains(raw, []byte(capability)) {
			t.Fatalf("pending decisions persisted raw seal capability %q: %s", capability, raw)
		}
	}
	if hashes := persistedCheckpointCapabilityHashes(after); len(hashes) != 3 {
		t.Fatalf("seal capability rotations persisted %d hashes, want 3: %#v", len(hashes), hashes)
	}
	resolved, found, err := resolveAutopilotCheckpointPendingDecision(
		checkpointDecisionQuestion(after), "owner confirmed after seal", phase.ID, secondCapability,
	)
	if err != nil || !found || resolved.ID != before.ID {
		t.Fatalf("fresh seal capability was not valid: resolved=%#v found=%v err=%v", resolved, found, err)
	}
}

func TestCheckpointCapabilityIsScopedSingleUseAndCannotCrossRows(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 5, Name: "Capability scope", Status: colony.PhaseInProgress}
	state := checkpointTestState(t, phase, colony.StateBUILT)
	criteria := []codexCriterionVerification{
		{TaskID: "5.1", Criterion: "The desktop flow feels right", State: criterionStateNeedsOwnerConfirmation},
		{TaskID: "5.2", Criterion: "The mobile flow feels right", State: criterionStateNeedsOwnerConfirmation},
	}
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, checkpointTestGeneration(t, "capability-scope", "capability-scope-evidence"))
	if err != nil || len(refs) != 2 {
		t.Fatalf("materialize checkpoints: refs=%#v err=%v", refs, err)
	}
	firstCapability := checkpointCapabilityFromReference(t, refs[0])
	secondCapability := checkpointCapabilityFromReference(t, refs[1])
	firstQuestion := refs[0].Question

	assertRejectedWithoutMutation := func(t *testing.T, question, capability string, answerPhase int) {
		t.Helper()
		before := pendingDecisionBytes(t)
		if _, found, err := resolveAutopilotCheckpointPendingDecision(question, "confirmed", answerPhase, capability); err != nil {
			t.Fatalf("rejected checkpoint answer returned error: %v", err)
		} else if found {
			t.Fatalf("checkpoint answer unexpectedly resolved for phase=%d capability=%q", answerPhase, capability)
		}
		after := pendingDecisionBytes(t)
		if !bytes.Equal(before, after) {
			t.Fatalf("rejected checkpoint answer mutated pending decisions:\nbefore=%s\nafter=%s", before, after)
		}
	}

	assertRejectedWithoutMutation(t, firstQuestion, "", phase.ID)
	assertRejectedWithoutMutation(t, firstQuestion, "not-the-capability", phase.ID)
	assertRejectedWithoutMutation(t, firstQuestion, secondCapability, phase.ID)
	assertRejectedWithoutMutation(t, firstQuestion, firstCapability, phase.ID+1)

	staleSession := "checkpoint-stale-session"
	state.SessionID = &staleSession
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save stale checkpoint scope: %v", err)
	}
	assertRejectedWithoutMutation(t, firstQuestion, firstCapability, phase.ID)
	state = checkpointTestState(t, phase, colony.StateBUILT)

	resolved, found, err := resolveAutopilotCheckpointPendingDecision(firstQuestion, "confirmed by owner", phase.ID, firstCapability)
	if err != nil || !found {
		t.Fatalf("correct checkpoint capability was refused: found=%v err=%v", found, err)
	}
	if resolved.ID != refs[0].ID || !resolved.Resolved || resolved.Resolution != "confirmed by owner" {
		t.Fatalf("resolved checkpoint = %#v, want the exact first row", resolved)
	}
	decisions := loadCheckpointDecisions(t)
	if len(decisions) != 2 || !decisions[0].Resolved || decisions[1].Resolved {
		t.Fatalf("correct capability did not resolve exactly one row: %#v", decisions)
	}

	replayBaseline := pendingDecisionBytes(t)
	if _, found, err := resolveAutopilotCheckpointPendingDecision(firstQuestion, "replayed", phase.ID, firstCapability); err != nil {
		t.Fatalf("replay returned error: %v", err)
	} else if found {
		t.Fatal("single-use checkpoint capability was replayed")
	}
	if after := pendingDecisionBytes(t); !bytes.Equal(replayBaseline, after) {
		t.Fatalf("replayed capability mutated the resolved row:\nbefore=%s\nafter=%s", replayBaseline, after)
	}
}

func TestCheckpointAnswerResolvesOriginalRow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Resolve checkpoint", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	criteria := []codexCriterionVerification{
		{TaskID: "1.1", Criterion: "The first interaction feels right", State: criterionStateNeedsOwnerConfirmation},
		{TaskID: "1.2", Criterion: "The first interaction feels right on mobile", State: criterionStateNeedsOwnerConfirmation},
	}
	generation := checkpointTestGeneration(t, "answer-original", "answer-original-evidence")
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generation)
	if err != nil || len(refs) != 2 {
		t.Fatalf("materialize checkpoints: refs=%#v err=%v", refs, err)
	}
	capability := checkpointCapabilityFromReference(t, refs[0])
	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	answerArgs := []string{
		"decision-answer",
		"--question", refs[0].Question,
		"--answer", "confirmed",
		"--phase", "1",
		"--checkpoint-capability", capability,
	}
	rootCmd.SetArgs(answerArgs)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("decision-answer returned Cobra error: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("decision-answer rejected the displayed checkpoint command: %s", errBuf.String())
	}
	result := parseEnvelope(t, outBuf.String())["result"].(map[string]interface{})
	if result["id"] != refs[0].ID {
		t.Fatalf("answer appended/replaced the row instead of resolving its stable ID: got %v want %s", result["id"], refs[0].ID)
	}
	decisions := loadCheckpointDecisions(t)
	if len(decisions) != 2 {
		t.Fatalf("answer appended a third record: %#v", decisions)
	}
	unresolved := unresolvedCheckpoints(decisions)
	if len(unresolved) != 1 || unresolved[0].ID != refs[1].ID {
		t.Fatalf("exact match resolved the wrong similar question: %#v", unresolved)
	}

	// Re-materializing the same criteria must preserve the resolved original
	// and return only the still-open checkpoint to unattended orchestration.
	replayed, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria, generation)
	if err != nil {
		t.Fatalf("re-materialize after answer: %v", err)
	}
	if len(replayed) != 1 || replayed[0].ID != refs[1].ID {
		t.Fatalf("resolved checkpoint reopened or remained in signals: %#v", replayed)
	}

	// A consumed capability must remain reserved for the checkpoint path. It
	// may not fall through to an ordinary clarification and append a new row.
	replayBaseline := pendingDecisionBytes(t)
	outBuf.Reset()
	errBuf.Reset()
	rootCmd.SetArgs(answerArgs)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("replayed decision-answer returned Cobra error: %v", err)
	}
	if errBuf.Len() == 0 {
		t.Fatalf("replayed checkpoint capability was accepted: %s", outBuf.String())
	}
	if envelope := parseEnvelope(t, errBuf.String()); envelope["ok"] != false {
		t.Fatalf("replay did not return an error envelope: %#v", envelope)
	}
	if after := pendingDecisionBytes(t); !bytes.Equal(replayBaseline, after) {
		t.Fatalf("replayed checkpoint command mutated pending decisions:\nbefore=%s\nafter=%s", replayBaseline, after)
	}
}

func TestFinalPhaseAdvancesWithOwnerCheckpointAndSealBlocks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Final owner boundary", Status: colony.PhaseInProgress}
	state := checkpointTestState(t, phase, colony.StateBUILT)
	criterion := codexCriterionVerification{
		TaskID: "1.1", Criterion: "The final screen looks polished", State: criterionStateNeedsOwnerConfirmation, Passed: true,
	}
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{criterion}, checkpointTestGeneration(t, "final-owner-boundary", "final-owner-boundary-evidence"))
	if err != nil || len(refs) != 1 {
		t.Fatalf("materialize final checkpoint: refs=%#v err=%v", refs, err)
	}

	verification := codexContinueVerificationReport{Phase: phase.ID, ChecksPassed: true, Passed: true, Criteria: []codexCriterionVerification{criterion}}
	assessment := codexContinueAssessment{PositiveEvidence: true, Passed: true}
	gates := runCodexContinueGates(phase, codexContinueManifest{}, verification, assessment, time.Now().UTC(), nil)
	foundOwnerGate := false
	for _, check := range gates.Checks {
		if check.Name == "owner_confirmation_pending" {
			foundOwnerGate = true
			if !check.Passed {
				t.Fatalf("final-phase owner work still blocks advancement: %#v", check)
			}
		}
	}
	if !foundOwnerGate {
		t.Fatal("owner_confirmation_pending gate was not evaluated")
	}

	advanced, err := advancePhase(advancePhaseParams{
		PhaseID: phase.ID, ExpectedBuildStartedAt: state.BuildStartedAt,
		AllowedStates: []colony.State{colony.StateBUILT}, Source: "checkpoint-test", Now: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("advance final phase: %v", err)
	}
	if !advanced.Final || advanced.Updated.State != colony.StateCOMPLETED || advanced.Updated.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("final phase did not reach completed state: %#v", advanced)
	}

	blockers, _ := checkSealBlockers(store, advanced.Updated)
	if len(blockers) != 1 || blockers[0].ID != refs[0].ID {
		t.Fatalf("seal blockers do not name the durable checkpoint ID: %#v", blockers)
	}
	wantCommand := blockers[0].RecoveryCommand
	_ = checkpointCapabilityFromReference(t, autopilotCheckpointReference{RecoveryCommand: wantCommand})
	if _, _, err := validateSealReady(false); err == nil {
		t.Fatal("seal accepted unresolved owner checkpoint")
	} else if !strings.Contains(err.Error(), refs[0].ID) || !strings.Contains(err.Error(), "--checkpoint-capability") {
		t.Fatalf("seal refusal omitted stable ID or a freshly issued capability command: %v", err)
	}

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	rootCmd.SetArgs([]string{
		"decision-answer",
		"--question", refs[0].Question,
		"--answer", "confirmed",
		"--phase", "1",
		"--checkpoint-capability", checkpointCapabilityFromReference(t, refs[0]),
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resolve final checkpoint: %v", err)
	}
	if errBuf.Len() != 0 {
		t.Fatalf("public checkpoint answer failed: %s", errBuf.String())
	}
	if _, _, err := validateSealReady(false); err != nil {
		t.Fatalf("seal remained blocked after exact answer: %v", err)
	}
}
