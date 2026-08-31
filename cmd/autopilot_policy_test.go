package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

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

	result := buildRunDryRunResult(state, runCompatibilityOptions{})
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
	result := buildRunDryRunResult(state, runCompatibilityOptions{})

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
