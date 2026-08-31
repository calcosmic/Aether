package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

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

	claimsPath := "build/phase-2/claims.json"
	claims := codexBuildClaims{
		BuildPhase:    phase.ID,
		FilesCreated:  []string{"web/components/StatusCard.tsx", "../outside/forged.tsx"},
		FilesModified: []string{"web/styles/status.css", "cmd/status.go", "docs/status.md", "web/components/status-card.test.tsx"},
		TestsWritten:  []string{"web/components/status-card.test.tsx"},
	}
	if err := store.SaveJSON(claimsPath, claims); err != nil {
		t.Fatalf("save build claims: %v", err)
	}

	refs, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, map[string]interface{}{
		"claims_path": displayDataPath(claimsPath),
		// This untrusted legacy-style assertion must have no effect; paths
		// in the persisted runtime claims are the sole evidence source.
		"ui_touched": false,
	})
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

	second, err := materializeVisualCheckpointFromBuildResult(root, phase.ID, map[string]interface{}{"claims_path": displayDataPath(claimsPath)})
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
	boolOnlyPath := "build/phase-3/claims.json"
	if err := store.SaveJSON(boolOnlyPath, map[string]interface{}{"build_phase": 3, "files_created": []string{}, "files_modified": []string{}, "ui_touched": true}); err != nil {
		t.Fatalf("save boolean-only claims: %v", err)
	}
	if got, err := materializeVisualCheckpointFromBuildResult(root, 3, map[string]interface{}{"claims_path": displayDataPath(boolOnlyPath), "ui_touched": true}); err != nil || len(got) != 0 {
		t.Fatalf("worker boolean manufactured visual work: refs=%#v err=%v", got, err)
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

	first, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria)
	if err != nil {
		t.Fatalf("materialize runtime checkpoints: %v", err)
	}
	second, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria)
	if err != nil {
		t.Fatalf("replay runtime checkpoints: %v", err)
	}
	if len(first) != 2 || !reflect.DeepEqual(first, second) {
		t.Fatalf("runtime checkpoint replay drifted: first=%#v second=%#v", first, second)
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

func TestCheckpointAnswerResolvesOriginalRow(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Resolve checkpoint", Status: colony.PhaseInProgress}
	checkpointTestState(t, phase, colony.StateBUILT)
	criteria := []codexCriterionVerification{
		{TaskID: "1.1", Criterion: "The first interaction feels right", State: criterionStateNeedsOwnerConfirmation},
		{TaskID: "1.2", Criterion: "The first interaction feels right on mobile", State: criterionStateNeedsOwnerConfirmation},
	}
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria)
	if err != nil || len(refs) != 2 {
		t.Fatalf("materialize checkpoints: refs=%#v err=%v", refs, err)
	}
	targetQuestion := ownerConfirmationQuestionText(phase.ID, criteria[0].TaskID, criteria[0].Criterion)
	resolved, err := recordDecisionAnswer(targetQuestion, "confirmed", phase.ID, "owner")
	if err != nil {
		t.Fatalf("record checkpoint answer: %v", err)
	}
	if resolved.ID != refs[0].ID {
		t.Fatalf("answer appended/replaced the row instead of resolving its stable ID: got %s want %s", resolved.ID, refs[0].ID)
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
	replayed, err := materializeRuntimeVerificationCheckpoints(phase.ID, criteria)
	if err != nil {
		t.Fatalf("re-materialize after answer: %v", err)
	}
	if len(replayed) != 1 || replayed[0].ID != refs[1].ID {
		t.Fatalf("resolved checkpoint reopened or remained in signals: %#v", replayed)
	}
}

func TestFinalPhaseAdvancesWithOwnerCheckpointAndSealBlocks(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	phase := colony.Phase{ID: 1, Name: "Final owner boundary", Status: colony.PhaseInProgress}
	state := checkpointTestState(t, phase, colony.StateBUILT)
	criterion := codexCriterionVerification{
		TaskID: "1.1", Criterion: "The final screen looks polished", State: criterionStateNeedsOwnerConfirmation, Passed: true,
	}
	refs, err := materializeRuntimeVerificationCheckpoints(phase.ID, []codexCriterionVerification{criterion})
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
	wantCommand := checkpointDecisionAnswerCommand(loadCheckpointDecisions(t)[0])
	if blockers[0].RecoveryCommand != wantCommand {
		t.Fatalf("seal recovery command = %q, want exact %q", blockers[0].RecoveryCommand, wantCommand)
	}
	if _, _, err := validateSealReady(false); err == nil {
		t.Fatal("seal accepted unresolved owner checkpoint")
	} else if !strings.Contains(err.Error(), refs[0].ID) || !strings.Contains(err.Error(), wantCommand) {
		t.Fatalf("seal refusal omitted stable ID or exact answer command: %v", err)
	}

	question, _ := parseClarificationDescription(loadCheckpointDecisions(t)[0].Description)
	if _, err := recordDecisionAnswer(question, "confirmed", phase.ID, "owner"); err != nil {
		t.Fatalf("resolve final checkpoint: %v", err)
	}
	if _, _, err := validateSealReady(false); err != nil {
		t.Fatalf("seal remained blocked after exact answer: %v", err)
	}
}
