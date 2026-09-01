package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func seedRunDryRunReplanFixture(t *testing.T) string {
	t.Helper()
	_, root := seedRunFixture(t, 3)
	boundary := time.Date(2026, time.September, 1, 8, 0, 0, 0, time.UTC)
	mutateRunFixtureState(t, func(state *colony.ColonyState) {
		state.State = colony.StateREADY
		state.CurrentPhase = 1
		state.Plan = autopilotLessonPlan(boundary, "revision-dry-run")
		state.Plan.Phases = []colony.Phase{
			{ID: 1, Name: "First", Status: colony.PhaseReady},
			{ID: 2, Name: "Second", Status: colony.PhasePending},
			{ID: 3, Name: "Third", Status: colony.PhasePending},
		}
	})
	return root
}

func snapshotDryRunDurableFiles(t *testing.T) map[string][]byte {
	t.Helper()
	snapshot := map[string][]byte{}
	err := filepath.Walk(store.BasePath(), func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(store.BasePath(), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot durable files: %v", err)
	}
	return snapshot
}

func dryRunReplanPreviewStep(result map[string]interface{}) map[string]interface{} {
	steps, _ := result["steps"].([]map[string]interface{})
	for _, step := range steps {
		if stringValue(step["trigger_code"]) == string(autopilotTriggerReplanDue) {
			return step
		}
	}
	return nil
}

func TestRunDryRunLessonAwareReplanNeedsConfirmedLessons(t *testing.T) {
	saveGlobals(t)
	root := seedRunDryRunReplanFixture(t)
	installAutopilotRunTestDeps(t)
	loadCalls := 0
	runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) {
		loadCalls++
		return nil, nil
	}

	before := snapshotDryRunDurableFiles(t)
	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{DryRun: true, ReplanInterval: 2})
	if err != nil {
		t.Fatalf("dry-run without lessons: %v", err)
	}
	after := snapshotDryRunDurableFiles(t)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("dry-run without lessons mutated durable files:\nbefore=%v\nafter=%v", before, after)
	}
	if loadCalls == 0 {
		t.Fatal("dry-run never read the live confirmed-lesson selector")
	}
	if stringValue(result["stopped_reason"]) != "completed" || intValue(result["phases_planned"]) != 3 {
		t.Fatalf("cadence without lessons fabricated a stop: %+v", result)
	}
	if preview := dryRunReplanPreviewStep(result); preview != nil {
		t.Fatalf("cadence without lessons fabricated replan_due: %+v", preview)
	}
	if _, err := os.Stat(filepath.Join(store.BasePath(), pendingDecisionsFile)); !os.IsNotExist(err) {
		t.Fatalf("dry-run created pending decisions: %v", err)
	}
}

func TestRunDryRunReplanModeParityAndContinueBypass(t *testing.T) {
	tests := []struct {
		name       string
		headless   bool
		bypass     bool
		wantReason string
		wantTop    autopilotDisposition
		wantQueued bool
	}{
		{name: "interactive pauses", wantReason: "replan_due", wantTop: autopilotDispositionPause},
		{name: "headless queues and continues", headless: true, wantReason: "completed", wantQueued: true},
		{name: "continue bypasses due event", headless: true, bypass: true, wantReason: "completed"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			root := seedRunDryRunReplanFixture(t)
			installAutopilotRunTestDeps(t)
			runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) {
				return []confirmedAutopilotLesson{{
					EntryID: "lesson-dry-run", Content: "Keep the live and preview policy together.",
					Phase: 1, PlanRevisionID: "revision-dry-run",
				}}, nil
			}

			before := snapshotDryRunDurableFiles(t)
			result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{
				DryRun: true, ReplanInterval: 2, Headless: tc.headless, ContinueWithoutReplan: tc.bypass,
			})
			if err != nil {
				t.Fatalf("dry-run replan preview: %v", err)
			}
			after := snapshotDryRunDurableFiles(t)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("dry-run replan preview mutated durable files:\nbefore=%v\nafter=%v", before, after)
			}
			if got := stringValue(result["stopped_reason"]); got != tc.wantReason {
				t.Fatalf("stopped_reason = %q, want %q: %+v", got, tc.wantReason, result)
			}

			preview := dryRunReplanPreviewStep(result)
			if tc.wantQueued {
				if preview == nil || !boolValue(preview["preview_only"]) ||
					stringValue(preview["disposition"]) != string(autopilotDispositionQueueAndContinue) ||
					intValue(preview["lesson_count"]) != 1 ||
					stringValue(preview["plan_revision_id"]) != "revision-dry-run" {
					t.Fatalf("headless preview did not queue canonical evidence: %+v", preview)
				}
			} else if preview != nil {
				t.Fatalf("unexpected replan preview step: %+v", preview)
			}
			if tc.wantTop != "" {
				if stringValue(result["trigger_code"]) != string(autopilotTriggerReplanDue) ||
					stringValue(result["disposition"]) != string(tc.wantTop) ||
					intValue(result["lesson_count"]) != 1 ||
					stringValue(result["plan_revision_id"]) != "revision-dry-run" {
					t.Fatalf("interactive preview lost canonical decision evidence: %+v", result)
				}
			}
			if _, err := os.Stat(filepath.Join(store.BasePath(), pendingDecisionsFile)); !os.IsNotExist(err) {
				t.Fatalf("dry-run created pending decisions: %v", err)
			}
		})
	}
}

func TestRunDryRunLessonLoadCorruptionIsReadOnly(t *testing.T) {
	saveGlobals(t)
	root := seedRunDryRunReplanFixture(t)
	installAutopilotRunTestDeps(t)
	runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) {
		return nil, errors.New("corrupt typed lesson catalogue")
	}
	before := snapshotDryRunDurableFiles(t)
	result, err := runCompatibilityAutopilot(root, runCompatibilityOptions{DryRun: true, ReplanInterval: 2, Headless: true})
	after := snapshotDryRunDurableFiles(t)
	if err == nil || !strings.Contains(err.Error(), "corrupt typed lesson catalogue") {
		t.Fatalf("lesson corruption result=%+v err=%v, want read-only preview failure", result, err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("failed dry-run lesson read mutated durable files:\nbefore=%v\nafter=%v", before, after)
	}
	if _, statErr := os.Stat(filepath.Join(store.BasePath(), pendingDecisionsFile)); !os.IsNotExist(statErr) {
		t.Fatalf("failed dry-run created pending decisions: %v", statErr)
	}
}

func expectedAutopilotTriggerCodes() []autopilotTriggerCode {
	return []autopilotTriggerCode{
		autopilotTriggerDeterministicVerificationFailed,
		autopilotTriggerAuditorScoreBelowFloor,
		autopilotTriggerCriticalReviewFinding,
		autopilotTriggerBlockerCountIncreased,
		autopilotTriggerBlockerEscalated,
		autopilotTriggerColonyNotRunnable,
		autopilotTriggerProviderUnavailable,
		autopilotTriggerRuntimeVerificationNeeded,
		autopilotTriggerVisualCheckpointNeeded,
		autopilotTriggerReplanDue,
		autopilotTriggerCancelled,
		autopilotTriggerWorkerTimeout,
		autopilotTriggerMaxPhasesReached,
		autopilotTriggerColonyComplete,
	}
}

func TestAutopilotTriggerCatalogue(t *testing.T) {
	specs := autopilotTriggerSpecs()
	if err := validateAutopilotTriggerSpecs(specs); err != nil {
		t.Fatalf("canonical trigger catalogue is invalid: %v", err)
	}

	gotCodes := make([]autopilotTriggerCode, 0, len(specs))
	for _, spec := range specs {
		gotCodes = append(gotCodes, spec.Code)
		if strings.TrimSpace(spec.Label) == "" {
			t.Errorf("trigger %q has a blank label", spec.Code)
		}
		if strings.TrimSpace(spec.Detection) == "" {
			t.Errorf("trigger %q has blank detection text", spec.Code)
		}
		if strings.TrimSpace(spec.NextActionTemplate) == "" {
			t.Errorf("trigger %q has a blank next-action template", spec.Code)
		}
	}
	if want := expectedAutopilotTriggerCodes(); !reflect.DeepEqual(gotCodes, want) {
		t.Fatalf("trigger codes/order = %v, want %v", gotCodes, want)
	}

	auditor, ok := autopilotTriggerSpecByCode(autopilotTriggerAuditorScoreBelowFloor)
	if !ok {
		t.Fatal("Auditor score trigger missing")
	}
	auditorDetection := strings.ToLower(auditor.Detection)
	if !strings.Contains(auditorDetection, "below 60") {
		t.Errorf("Auditor score-floor text must say below 60, got %q", auditor.Detection)
	}
	if !strings.Contains(auditorDetection, "actually ran") {
		t.Errorf("Auditor trigger must be conditional on an Auditor that actually ran, got %q", auditor.Detection)
	}
	if strings.Contains(auditorDetection, "dispatch") || strings.Contains(auditorDetection, "send an auditor") {
		t.Errorf("Auditor trigger must not imply an Auditor is dispatched to obtain a score, got %q", auditor.Detection)
	}
}

func TestAutopilotTriggerCatalogueRejectsInvalidRows(t *testing.T) {
	tests := []struct {
		name   string
		mutate func([]autopilotTriggerSpec) []autopilotTriggerSpec
	}{
		{
			name: "blank code",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[0].Code = ""
				return specs
			},
		},
		{
			name: "duplicate code",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[1].Code = specs[0].Code
				return specs
			},
		},
		{
			name: "unknown headless disposition",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[0].HeadlessDisposition = autopilotDisposition("invented")
				return specs
			},
		},
		{
			name: "unknown interactive disposition",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[0].InteractiveDisposition = autopilotDisposition("invented")
				return specs
			},
		},
		{
			name: "blank detection",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[0].Detection = "  "
				return specs
			},
		},
		{
			name: "blank next action",
			mutate: func(specs []autopilotTriggerSpec) []autopilotTriggerSpec {
				specs[0].NextActionTemplate = ""
				return specs
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			specs := append([]autopilotTriggerSpec(nil), autopilotTriggerSpecs()...)
			if err := validateAutopilotTriggerSpecs(tc.mutate(specs)); err == nil {
				t.Fatal("validation unexpectedly accepted invalid trigger catalogue")
			}
		})
	}
}

func TestAutopilotDispositionMatrix(t *testing.T) {
	want := map[autopilotTriggerCode][2]autopilotDisposition{
		autopilotTriggerDeterministicVerificationFailed: {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerAuditorScoreBelowFloor:          {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerCriticalReviewFinding:           {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerBlockerCountIncreased:           {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerBlockerEscalated:                {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerColonyNotRunnable:               {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerProviderUnavailable:             {autopilotDispositionStop, autopilotDispositionStop},
		autopilotTriggerRuntimeVerificationNeeded:       {autopilotDispositionQueueAndContinue, autopilotDispositionPause},
		autopilotTriggerVisualCheckpointNeeded:          {autopilotDispositionQueueAndContinue, autopilotDispositionPause},
		autopilotTriggerReplanDue:                       {autopilotDispositionQueueAndContinue, autopilotDispositionPause},
		autopilotTriggerCancelled:                       {autopilotDispositionNormalStop, autopilotDispositionNormalStop},
		autopilotTriggerWorkerTimeout:                   {autopilotDispositionNormalStop, autopilotDispositionNormalStop},
		autopilotTriggerMaxPhasesReached:                {autopilotDispositionNormalStop, autopilotDispositionNormalStop},
		autopilotTriggerColonyComplete:                  {autopilotDispositionNormalStop, autopilotDispositionNormalStop},
	}

	for _, spec := range autopilotTriggerSpecs() {
		dispositions, ok := want[spec.Code]
		if !ok {
			t.Errorf("unexpected trigger %q in disposition matrix", spec.Code)
			continue
		}
		if spec.HeadlessDisposition != dispositions[0] || spec.InteractiveDisposition != dispositions[1] {
			t.Errorf("%s dispositions = (%s, %s), want (%s, %s)", spec.Code, spec.HeadlessDisposition, spec.InteractiveDisposition, dispositions[0], dispositions[1])
		}
		delete(want, spec.Code)
	}
	if len(want) != 0 {
		t.Fatalf("catalogue omitted disposition rows: %v", want)
	}
}

func TestRunDryRunUsesCanonicalTriggerCatalogue(t *testing.T) {
	state := colony.ColonyState{
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Catalogue fixture", Status: colony.PhasePending},
		}},
	}

	originalLessons := runAutopilotLoadLessons
	runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) { return nil, nil }
	t.Cleanup(func() { runAutopilotLoadLessons = originalLessons })
	result, err := buildRunDryRunResult(state, runCompatibilityOptions{})
	if err != nil {
		t.Fatalf("build dry-run result: %v", err)
	}
	dryRunSpecs, ok := result["trigger_catalogue"].([]autopilotTriggerSpec)
	if !ok {
		t.Fatalf("dry-run trigger_catalogue type = %T, want []autopilotTriggerSpec", result["trigger_catalogue"])
	}

	gotCodes := make([]autopilotTriggerCode, 0, len(dryRunSpecs))
	for _, spec := range dryRunSpecs {
		gotCodes = append(gotCodes, spec.Code)
	}
	wantSpecs := autopilotTriggerSpecs()
	wantCodes := make([]autopilotTriggerCode, 0, len(wantSpecs))
	for _, spec := range wantSpecs {
		wantCodes = append(wantCodes, spec.Code)
	}
	if !reflect.DeepEqual(gotCodes, wantCodes) {
		t.Fatalf("dry-run codes = %v, canonical codes = %v", gotCodes, wantCodes)
	}
	if !reflect.DeepEqual(dryRunSpecs, wantSpecs) {
		t.Fatal("dry-run contract must project the canonical trigger specs without hand-written substitutions")
	}
}

func TestAutopilotTriggerCatalogueRendersBothModeDispositions(t *testing.T) {
	state := colony.ColonyState{
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Rendered catalogue fixture", Status: colony.PhasePending},
		}},
	}
	originalLessons := runAutopilotLoadLessons
	runAutopilotLoadLessons = func(colony.Plan) ([]confirmedAutopilotLesson, error) { return nil, nil }
	t.Cleanup(func() { runAutopilotLoadLessons = originalLessons })
	result, err := buildRunDryRunResult(state, runCompatibilityOptions{})
	if err != nil {
		t.Fatalf("build dry-run result: %v", err)
	}

	// Exercise the JSON-round-tripped shape used by hosted callers as well as
	// the direct typed result used by the CLI.
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal dry-run result: %v", err)
	}
	var roundTripped map[string]interface{}
	if err := json.Unmarshal(encoded, &roundTripped); err != nil {
		t.Fatalf("unmarshal dry-run result: %v", err)
	}
	rendered := renderRunCompatibilityVisual(roundTripped)

	for _, spec := range autopilotTriggerSpecs() {
		for _, want := range []string{
			string(spec.Code),
			spec.Label,
			"Headless: " + string(spec.HeadlessDisposition),
			"Interactive: " + string(spec.InteractiveDisposition),
		} {
			if !strings.Contains(rendered, want) {
				t.Errorf("rendered dry-run catalogue missing %q", want)
			}
		}
	}
}
