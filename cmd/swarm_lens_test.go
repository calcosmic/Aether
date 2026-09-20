package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
)

// swarmWriteResponseFile writes payload as the JSON body of a lens worker's
// response file -- the same shape a real worker writes.
func swarmWriteResponseFile(t *testing.T, path string, payload map[string]interface{}) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir response dir: %v", err)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		t.Fatalf("write response file: %v", err)
	}
}

// swarmExecutionFromRawResponse builds a swarmWorkerExecution fixture by
// writing a raw response file and loading it back through
// loadSwarmWorkerResponse -- the runtime's own real response-loading path --
// rather than hand-constructing a swarmWorkerResponse struct literal.
func swarmExecutionFromRawResponse(t *testing.T, dir, name, caste string, payload map[string]interface{}) swarmWorkerExecution {
	t.Helper()
	path := filepath.Join(dir, name+".json")
	swarmWriteResponseFile(t, path, payload)
	response, err := loadSwarmWorkerResponse(path, caste)
	if err != nil {
		t.Fatalf("loadSwarmWorkerResponse(%s): %v", path, err)
	}
	return swarmWorkerExecution{
		Name:         name,
		Caste:        caste,
		Role:         caste,
		Status:       response.Status,
		Summary:      response.Summary,
		Response:     response,
		ResponsePath: path,
	}
}

// TestFourSwarmLensesProduceDistinctEvidence proves the four declared lenses
// are genuinely distinct -- by identifier, caste and brief -- and that four
// real fixture responses (built through loadSwarmWorkerResponse) map to four
// hypotheses whose evidence kinds differ.
func TestFourSwarmLensesProduceDistinctEvidence(t *testing.T) {
	if len(swarmLenses) != 4 {
		t.Fatalf("expected 4 declared lenses, got %d: %+v", len(swarmLenses), swarmLenses)
	}
	if issues := distinctSwarmLensViolations(swarmLenses); len(issues) != 0 {
		t.Fatalf("declared swarm lenses are not distinct: %v", issues)
	}
	wantIDs := map[string]bool{
		swarmLensErrorPath:        true,
		swarmLensPattern:          true,
		swarmLensHistory:          true,
		swarmLensExternalEvidence: true,
	}
	for _, lens := range swarmLenses {
		if !wantIDs[lens.ID] {
			t.Fatalf("unexpected lens id %q, want one of %v", lens.ID, wantIDs)
		}
	}

	// The distinctness check itself must be able to fail: two lenses
	// sharing a brief must be flagged.
	duplicateBrief := []swarmLensDef{
		{ID: "a", Label: "A", Caste: "tracker", Brief: "identical brief text"},
		{ID: "b", Label: "B", Caste: "scout", Brief: "identical brief text"},
	}
	if issues := distinctSwarmLensViolations(duplicateBrief); len(issues) == 0 {
		t.Fatal("expected distinctSwarmLensViolations to fail two lenses sharing a brief")
	}

	dir := t.TempDir()
	fixtures := map[string]map[string]interface{}{
		"tracker": {
			"role":       "tracker",
			"status":     "completed",
			"root_cause": "nil pointer in the auth handler",
			"evidence":   []string{"pkg/auth/handler.go:42"},
		},
		"scout": {
			"role":     "scout",
			"status":   "completed",
			"summary":  "found the matching test pattern in a sibling module",
			"evidence": []string{"pkg/auth/handler_test.go"},
		},
		"archaeologist": {
			"role":     "archaeologist",
			"status":   "completed",
			"summary":  "a recent cleanup commit removed the nil guard",
			"evidence": []string{"git log -- pkg/auth/handler.go"},
		},
		"oracle": {
			"role":     "oracle",
			"status":   "completed",
			"summary":  "the framework's own docs require a nil check before session use",
			"evidence": []string{"https://example.com/docs/session-handling"},
		},
	}

	var runs []swarmWorkerExecution
	for _, lens := range swarmLenses {
		runs = append(runs, swarmExecutionFromRawResponse(t, dir, lens.Caste+"-worker", lens.Caste, fixtures[lens.Caste]))
	}

	hypotheses, missing := hypothesesFromSwarmRuns(runs)
	if len(hypotheses) != 4 {
		t.Fatalf("expected 4 hypotheses, got %d: %+v", len(hypotheses), hypotheses)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no missing lenses, got %+v", missing)
	}

	kinds := map[string]bool{}
	for _, h := range hypotheses {
		if len(h.Evidence) == 0 {
			t.Fatalf("hypothesis for lens %s has no evidence", h.Lens)
		}
		kind := h.Evidence[0].Kind
		if kinds[kind] {
			t.Fatalf("evidence kind %q reused across lenses -- lenses do not produce distinct evidence: %+v", kind, hypotheses)
		}
		kinds[kind] = true
	}
}

// TestSwarmHypothesisPreservesUnstatedConfidence proves a response with no
// stated confidence maps to a hypothesis whose confidence reads as unstated
// (nil), distinguishable from a hypothesis whose confidence was explicitly
// stated as zero.
func TestSwarmHypothesisPreservesUnstatedConfidence(t *testing.T) {
	dir := t.TempDir()
	unstated := swarmExecutionFromRawResponse(t, dir, "tracker-unstated", "tracker", map[string]interface{}{
		"role": "tracker", "status": "completed", "root_cause": "unstated confidence claim",
	})
	stated := swarmExecutionFromRawResponse(t, dir, "tracker-stated", "tracker", map[string]interface{}{
		"role": "tracker", "status": "completed", "root_cause": "stated zero confidence claim", "confidence": 0,
	})

	hypUnstated, _ := hypothesesFromSwarmRuns([]swarmWorkerExecution{unstated})
	if len(hypUnstated) != 1 {
		t.Fatalf("expected 1 hypothesis, got %d", len(hypUnstated))
	}
	if hypUnstated[0].Confidence != nil {
		t.Fatalf("expected unstated confidence to be nil, got %d", *hypUnstated[0].Confidence)
	}

	hypStated, _ := hypothesesFromSwarmRuns([]swarmWorkerExecution{stated})
	if len(hypStated) != 1 {
		t.Fatalf("expected 1 hypothesis, got %d", len(hypStated))
	}
	if hypStated[0].Confidence == nil {
		t.Fatal("expected a stated zero confidence to be non-nil")
	}
	if *hypStated[0].Confidence != 0 {
		t.Fatalf("confidence = %d, want 0", *hypStated[0].Confidence)
	}
}

// TestSwarmLensWithoutClaimIsRecordedAsNotReporting proves a lens response
// with no usable claim (no root cause, no summary) produces no hypothesis
// and is recorded as a non-reporting lens with a reason -- and that lenses
// which never ran at all are recorded the same way.
func TestSwarmLensWithoutClaimIsRecordedAsNotReporting(t *testing.T) {
	dir := t.TempDir()
	blank := swarmExecutionFromRawResponse(t, dir, "scout-blank", "scout", map[string]interface{}{
		"role": "scout", "status": "blocked",
	})

	hypotheses, missing := hypothesesFromSwarmRuns([]swarmWorkerExecution{blank})
	if len(hypotheses) != 0 {
		t.Fatalf("expected no hypothesis for a claim-less response, got %+v", hypotheses)
	}
	if len(missing) != len(swarmLenses) {
		t.Fatalf("expected all %d lenses recorded (1 no-claim + %d never-ran), got %d: %+v",
			len(swarmLenses), len(swarmLenses)-1, len(missing), missing)
	}
	found := false
	for _, m := range missing {
		if m.Lens == swarmLensPattern {
			found = true
			if strings.TrimSpace(m.Reason) == "" {
				t.Fatal("expected a non-empty reason for the non-reporting lens")
			}
		}
	}
	if !found {
		t.Fatalf("expected the pattern lens to be recorded as non-reporting, got %+v", missing)
	}
}

// TestSwarmComparisonSurfacesSharedCausesAndContradictions proves
// compareSwarmHypotheses groups matching claims into one shared-cause entry
// naming every corroborating lens, and surfaces a self-reported
// contradiction between two lenses naming both lenses and both claims.
func TestSwarmComparisonSurfacesSharedCausesAndContradictions(t *testing.T) {
	t.Run("shared cause", func(t *testing.T) {
		hyps := []swarmHypothesis{
			{Lens: swarmLensErrorPath, LensLabel: "Error Path", Claim: "missing nil guard in the auth handler"},
			{Lens: swarmLensPattern, LensLabel: "Repository Pattern", Claim: "missing nil guard in the auth handler"},
			{Lens: swarmLensHistory, LensLabel: "Git History", Claim: "missing nil guard in the auth handler"},
			{Lens: swarmLensExternalEvidence, LensLabel: "External Evidence", Claim: "missing nil guard in the auth handler"},
		}
		comparison := compareSwarmHypotheses(hyps, nil)
		if len(comparison.SharedCauses) != 1 {
			t.Fatalf("expected 1 shared cause, got %d: %+v", len(comparison.SharedCauses), comparison.SharedCauses)
		}
		if len(comparison.SharedCauses[0].Lenses) != 4 {
			t.Fatalf("expected all 4 lenses named in the shared cause, got %+v", comparison.SharedCauses[0].Lenses)
		}
	})

	t.Run("contradiction", func(t *testing.T) {
		hyps := []swarmHypothesis{
			{
				Lens: swarmLensPattern, LensLabel: "Repository Pattern",
				Claim:          "a config bug caused the failure",
				Contradictions: []string{"pattern claims a config bug, but error-path claims a race condition"},
			},
			{
				Lens: swarmLensErrorPath, LensLabel: "Error Path",
				Claim: "a race condition caused the failure",
			},
		}
		comparison := compareSwarmHypotheses(hyps, nil)
		if len(comparison.Contradictions) != 1 {
			t.Fatalf("expected 1 contradiction, got %d: %+v", len(comparison.Contradictions), comparison.Contradictions)
		}
		c := comparison.Contradictions[0]
		gotLenses := map[string]bool{c.LensA: true, c.LensB: true}
		if !gotLenses[swarmLensPattern] || !gotLenses[swarmLensErrorPath] {
			t.Fatalf("expected contradiction naming both lenses, got %+v", c)
		}
		gotClaims := map[string]bool{c.ClaimA: true, c.ClaimB: true}
		if !gotClaims["a config bug caused the failure"] || !gotClaims["a race condition caused the failure"] {
			t.Fatalf("expected contradiction naming both claims, got %+v", c)
		}
	})
}

// TestSwarmRepairRankingPrefersCorroboration proves ranking orders candidate
// repairs by cross-lens corroboration first, then by whether a confidence
// was stated at all, and that the selection reason names the runner-up.
func TestSwarmRepairRankingPrefersCorroboration(t *testing.T) {
	t.Run("more lenses wins with equal confidence", func(t *testing.T) {
		hyps := []swarmHypothesis{
			{Lens: swarmLensErrorPath, Claim: "cause A", ProposedRepair: "repair A", Confidence: intPtr(70)},
			{Lens: swarmLensPattern, Claim: "cause A", ProposedRepair: "repair A", Confidence: intPtr(70)},
			{Lens: swarmLensHistory, Claim: "cause B", ProposedRepair: "repair B", Confidence: intPtr(90)},
		}
		comparison := compareSwarmHypotheses(hyps, nil)
		if comparison.Selected == nil {
			t.Fatal("expected a selected repair")
		}
		if comparison.Selected.Repair != "repair A" {
			t.Fatalf("selected = %q, want repair A (corroborated by 2 lenses)", comparison.Selected.Repair)
		}
		if !strings.Contains(comparison.SelectionReason, "repair B") {
			t.Fatalf("selection reason should name what it beat: %q", comparison.SelectionReason)
		}
	})

	t.Run("stated confidence outranks unstated with equal corroboration", func(t *testing.T) {
		hyps := []swarmHypothesis{
			{Lens: swarmLensErrorPath, Claim: "cause A", ProposedRepair: "repair A"},
			{Lens: swarmLensPattern, Claim: "cause B", ProposedRepair: "repair B", Confidence: intPtr(50)},
		}
		comparison := compareSwarmHypotheses(hyps, nil)
		if comparison.Selected == nil || comparison.Selected.Repair != "repair B" {
			t.Fatalf("expected stated-confidence repair B to win, got %+v", comparison.Selected)
		}
	})
}

// TestSwarmComparisonWithMissingLensNamesIt proves a comparison built from
// three reporting lenses names the fourth, missing one by identifier with
// its reason, and still selects a repair from the three that reported.
func TestSwarmComparisonWithMissingLensNamesIt(t *testing.T) {
	hyps := []swarmHypothesis{
		{Lens: swarmLensErrorPath, LensLabel: "Error Path", Claim: "cause A", ProposedRepair: "repair A"},
		{Lens: swarmLensPattern, LensLabel: "Repository Pattern", Claim: "cause A", ProposedRepair: "repair A"},
		{Lens: swarmLensHistory, LensLabel: "Git History", Claim: "cause A", ProposedRepair: "repair A"},
	}
	missing := []swarmNonReportingLens{
		{Lens: swarmLensExternalEvidence, Label: "External Evidence", Reason: "external-evidence lens did not run in this investigation wave"},
	}
	comparison := compareSwarmHypotheses(hyps, missing)
	if len(comparison.MissingLenses) != 1 || comparison.MissingLenses[0].Lens != swarmLensExternalEvidence {
		t.Fatalf("expected the missing lens to be named, got %+v", comparison.MissingLenses)
	}
	if comparison.Selected == nil {
		t.Fatal("expected ranking to still select a repair from the three reporting lenses")
	}

	card := renderSwarmHypothesisCard(comparison)
	if !strings.Contains(card, "External Evidence") {
		t.Fatalf("card does not name the missing lens:\n%s", card)
	}
	if !strings.Contains(card, missing[0].Reason) {
		t.Fatalf("card does not carry the missing lens's reason:\n%s", card)
	}
}

// TestSwarmComparisonWithNoEvidenceRanksNothing proves that when zero lenses
// report, the comparison has an empty ranked list, no selected repair, and
// the card says plainly that no evidence was gathered.
func TestSwarmComparisonWithNoEvidenceRanksNothing(t *testing.T) {
	var missing []swarmNonReportingLens
	for _, lens := range swarmLenses {
		missing = append(missing, swarmNonReportingLens{Lens: lens.ID, Label: lens.Label, Reason: lens.Label + " did not run"})
	}
	comparison := compareSwarmHypotheses(nil, missing)
	if len(comparison.Ranked) != 0 {
		t.Fatalf("expected no ranked repairs, got %+v", comparison.Ranked)
	}
	if comparison.Selected != nil {
		t.Fatalf("expected no selected repair, got %+v", comparison.Selected)
	}
	card := renderSwarmHypothesisCard(comparison)
	if !strings.Contains(card, "No lens produced usable evidence") {
		t.Fatalf("card does not say plainly that no evidence was gathered:\n%s", card)
	}
}

// TestSwarmCardUsesSharedCasteIdentity proves the card's caste glyphs come
// from the shared caste-identity functions in cmd/codex_visuals.go, not a
// separately maintained map: changing one entry in the shared map changes
// the rendered card.
func TestSwarmCardUsesSharedCasteIdentity(t *testing.T) {
	original := casteEmojiMap["tracker"]
	casteEmojiMap["tracker"] = "\U0001F984" // unicorn -- guaranteed not to collide with any real caste glyph
	t.Cleanup(func() { casteEmojiMap["tracker"] = original })

	comparison := compareSwarmHypotheses(nil, nil)
	card := renderSwarmHypothesisCard(comparison)
	if !strings.Contains(card, "\U0001F984") {
		t.Fatalf("card did not reflect the changed shared caste emoji map:\n%s", card)
	}
}

// swarmLensTestInvoker is a stub codex.WorkerInvoker driving the real
// runSwarmDestroy path end to end, with per-caste response payloads the
// test controls directly -- Task 3's tests exercise the real production
// pipeline, not a hand-built swarmHypothesis fixture.
type swarmLensTestInvoker struct {
	responses map[string]map[string]interface{}
	configs   []codex.WorkerConfig
}

func (i *swarmLensTestInvoker) Invoke(_ context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.configs = append(i.configs, cfg)

	payload, ok := i.responses[cfg.Caste]
	if !ok {
		payload = map[string]interface{}{
			"role":    cfg.Caste,
			"status":  "completed",
			"summary": cfg.Caste + " completed the swarm pass.",
		}
	}
	status, _ := payload["status"].(string)
	if strings.TrimSpace(status) == "" {
		status = "completed"
	}
	summary, _ := payload["summary"].(string)
	result := codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     status,
		Summary:    summary,
		Duration:   time.Second,
	}
	if cfg.Caste == "builder" {
		result.FilesModified = []string{"pkg/auth/handler.go"}
		result.TestsWritten = []string{"pkg/auth/handler_test.go"}
	}

	if strings.TrimSpace(cfg.ResponsePath) == "" {
		return codex.WorkerResult{}, context.Canceled
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ResponsePath), 0755); err != nil {
		return codex.WorkerResult{}, err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(cfg.ResponsePath, append(data, '\n'), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return result, nil
}

func (i *swarmLensTestInvoker) IsAvailable(_ context.Context) bool { return true }
func (i *swarmLensTestInvoker) ValidateAgent(_ string) error       { return nil }

// TestSwarmInvestigationRunsFourLenses drives a real fixture Swarm run and
// proves the investigation wave dispatches exactly the four declared lenses,
// each carrying its lens identifier on its persisted live events, and that
// a hypothesis-formed event and a contradiction-found event appear for a
// fixture set containing a contradiction.
func TestSwarmInvestigationRunsFourLenses(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Destroy a stubborn bug"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
	})

	invoker := &swarmLensTestInvoker{
		responses: map[string]map[string]interface{}{
			"tracker": {
				"role": "tracker", "status": "completed",
				"root_cause":     "missing nil guard in the auth handler",
				"proposed_fix":   "add a nil guard before dereferencing the session",
				"confidence":     80,
				"contradictions": []string{"tracker says this is a nil-guard bug, but scout's pattern lens claims a permissions bug"},
			},
			"scout": {
				"role": "scout", "status": "completed",
				"summary": "the failure looks like a permissions bug, not a nil-guard issue",
			},
			"archaeologist": {
				"role": "archaeologist", "status": "completed",
				"root_cause": "a recent cleanup commit removed the nil guard",
			},
			"oracle": {
				"role": "oracle", "status": "completed",
				"summary": "framework docs require a nil check before session use",
			},
		},
	}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"swarm", "Session dereference panic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	workers, _ := result["workers"].([]interface{})

	gotLenses := map[string]bool{}
	for _, raw := range workers {
		w, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		if lens, ok := swarmLensForCaste(stringValue(w["caste"])); ok {
			gotLenses[lens.ID] = true
		}
	}
	for _, lens := range swarmLenses {
		if !gotLenses[lens.ID] {
			t.Fatalf("investigation wave did not dispatch the %s lens; got workers %+v", lens.ID, workers)
		}
	}

	liveEvts := liveEventsSince(t)
	lensByCaste := map[string]string{}
	hypothesisFormed := false
	contradictionFound := false
	for _, evt := range liveEvts {
		switch evt.Topic {
		case events.LiveTopicWorkerStarted:
			if evt.Payload.Wave == 1 {
				lensByCaste[evt.Payload.Caste] = evt.Payload.Lens
			}
		case events.LiveTopicFindingRecorded:
			if evt.Payload.Status == "hypothesis_formed" {
				hypothesisFormed = true
			}
		case events.LiveTopicContradictionFound:
			if evt.Payload.Status == "contradiction_found" {
				contradictionFound = true
			}
		}
	}
	for _, lens := range swarmLenses {
		if lensByCaste[lens.Caste] != lens.ID {
			t.Fatalf("worker-started event for caste %s carries lens %q, want %q", lens.Caste, lensByCaste[lens.Caste], lens.ID)
		}
	}
	if !hypothesisFormed {
		t.Fatal("expected at least one hypothesis-formed live event")
	}
	if !contradictionFound {
		t.Fatal("expected a contradiction-found live event for the fixture's contradicting lenses")
	}
}

// TestSwarmFixWaveConsumesTheComparison proves the fix wave's task brief
// carries the structured comparison's selected repair and selection reason
// -- not renderSwarmFindingSummary's free-text concatenation.
func TestSwarmFixWaveConsumesTheComparison(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Destroy a stubborn bug"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
	})

	invoker := &swarmLensTestInvoker{
		responses: map[string]map[string]interface{}{
			"tracker": {
				"role": "tracker", "status": "completed",
				"root_cause":   "missing nil guard in the auth handler",
				"proposed_fix": "add a nil guard before dereferencing the session",
				"confidence":   85,
			},
		},
	}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"swarm", "Session dereference panic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	var fixBrief string
	for _, cfg := range invoker.configs {
		if cfg.Caste == "builder" {
			fixBrief = cfg.TaskBrief
		}
	}
	if fixBrief == "" {
		t.Fatal("expected a builder dispatch with a task brief")
	}
	if !strings.Contains(fixBrief, "add a nil guard before dereferencing the session") {
		t.Fatalf("fix wave brief does not contain the comparison's selected repair:\n%s", fixBrief)
	}
	if !strings.Contains(fixBrief, "Selected repair (from the four-lens hypothesis comparison):") {
		t.Fatalf("fix wave brief does not carry the structured comparison marker (looks like renderSwarmFindingSummary instead):\n%s", fixBrief)
	}
	if !strings.Contains(fixBrief, "Why this repair was selected:") {
		t.Fatalf("fix wave brief does not carry the selection reasoning:\n%s", fixBrief)
	}
}

// TestSwarmWithNoUsableEvidenceDispatchesNoFixWave proves a run whose four
// lenses all return unusable responses dispatches zero fix-wave and zero
// verification-wave workers and completes with a stated no-evidence
// outcome.
func TestSwarmWithNoUsableEvidenceDispatchesNoFixWave(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Destroy a stubborn bug"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
	})

	invoker := &swarmLensTestInvoker{
		responses: map[string]map[string]interface{}{
			"tracker":       {"role": "tracker", "status": "blocked"},
			"scout":         {"role": "scout", "status": "blocked"},
			"archaeologist": {"role": "archaeologist", "status": "blocked"},
			"oracle":        {"role": "oracle", "status": "blocked"},
		},
	}
	originalInvoker := newSwarmWorkerInvoker
	newSwarmWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newSwarmWorkerInvoker = originalInvoker })

	rootCmd.SetArgs([]string{"swarm", "Session dereference panic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if got := result["status"]; got != "failed" {
		t.Fatalf("status = %v, want failed", got)
	}
	if got, _ := result["no_evidence"].(bool); !got {
		t.Fatalf("expected no_evidence: true in result, got %v", result["no_evidence"])
	}
	for _, cfg := range invoker.configs {
		if cfg.Caste == "builder" || cfg.Caste == "watcher" {
			t.Fatalf("expected no fix/verification wave dispatch, but caste %s was invoked", cfg.Caste)
		}
	}
}
