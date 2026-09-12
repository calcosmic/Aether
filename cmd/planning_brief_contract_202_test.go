package cmd

import (
	"reflect"
	"strings"
	"testing"
)

// briefContractTestStageManifest is a Scout stage manifest in the shape the
// runtime produces, for asserting on the brief a staged worker receives.
func briefContractTestStageManifest(caste planningStageWorkerCaste) planningStageManifest {
	return planningStageManifest{
		ID:            "planning-stage-manifest-" + strings.Repeat("a", 16),
		ContentHash:   strings.Repeat("b", 64),
		RunID:         "plan-briefcontract",
		Pass:          2,
		ExpectedCaste: caste,
	}
}

// TestStagedBriefNamesTheSchemaTheFinalizerAccepts is the regression test for a
// brief that instructed a worker to fail. The staged brief used to describe a
// `scout_report` payload with `confidence` and `study_files`; the finalizer
// decodes `planning-scout-result/v1` strictly with unknown fields refused, so a
// Scout that followed its own brief was rejected outright.
func TestStagedBriefNamesTheSchemaTheFinalizerAccepts(t *testing.T) {
	for _, test := range []struct {
		caste      planningStageWorkerCaste
		spec       string
		wantSchema planningStageResultType
	}{
		{planningStageCasteScout, "scout", planningStageResultScout},
		{planningStageCasteRouteSetter, "route_setter", planningStageResultRouteSetter},
	} {
		t.Run(string(test.caste), func(t *testing.T) {
			spec, ok := planningStageResumeWorkerSpec(test.spec)
			if !ok {
				t.Fatalf("no worker spec for %q", test.spec)
			}
			stage := briefContractTestStageManifest(test.caste)
			brief := renderPlanningWorkerBrief(t.TempDir(), codexSurveyContext{}, spec, &stage)

			if !strings.Contains(brief, string(test.wantSchema)) {
				t.Fatalf("staged %s brief never names the schema %q the finalizer requires", test.caste, test.wantSchema)
			}
			if !strings.Contains(brief, "Unknown fields are REFUSED") {
				t.Fatalf("staged %s brief does not warn that unknown fields fail the whole submission", test.caste)
			}
			// The legacy instructions are the ones that got a compliant worker
			// rejected; none may survive on the staged lane.
			for _, forbidden := range []string{
				"scout_report",
				"study_files",
				"Write planning outputs directly into the repository",
				"phase-plan.json",
			} {
				if strings.Contains(brief, forbidden) {
					t.Fatalf("staged %s brief still carries the legacy instruction %q, which the staged finalizer refuses", test.caste, forbidden)
				}
			}
			if !strings.Contains(brief, stage.ID) || !strings.Contains(brief, stage.RunID) {
				t.Fatalf("staged %s brief does not bind the worker to its exact manifest", test.caste)
			}
		})
	}
}

// TestStagedBriefFieldListMatchesTheDecoder proves the brief's field list is
// derived from the result struct rather than retyped, by checking it against
// the struct's own JSON tags. A brief that drifts from the decoder is the
// defect this whole contract exists to prevent.
func TestStagedBriefFieldListMatchesTheDecoder(t *testing.T) {
	for _, test := range []struct {
		caste  planningStageWorkerCaste
		spec   string
		sample interface{}
	}{
		{planningStageCasteScout, "scout", planningScoutStageResult{}},
		{planningStageCasteRouteSetter, "route_setter", planningRouteStageResult{}},
	} {
		t.Run(string(test.caste), func(t *testing.T) {
			spec, _ := planningStageResumeWorkerSpec(test.spec)
			stage := briefContractTestStageManifest(test.caste)
			brief := renderPlanningWorkerBrief(t.TempDir(), codexSurveyContext{}, spec, &stage)

			// Derive the expected field list independently here, straight
			// from the struct tags, rather than by calling the helper under
			// test -- otherwise this would only prove the helper agrees with
			// itself. A zero-value struct cannot be marshalled (its status
			// field validates on encode), so reflection is the honest route.
			rt := reflect.TypeOf(test.sample)
			for i := 0; i < rt.NumField(); i++ {
				tag := rt.Field(i).Tag.Get("json")
				if tag == "" || tag == "-" {
					continue
				}
				field := strings.TrimSpace(strings.Split(tag, ",")[0])
				if field == "" {
					continue
				}
				if !strings.Contains(brief, field) {
					t.Fatalf("staged %s brief omits %q, a field the decoder expects", test.caste, field)
				}
			}
			required, optional := jsonFieldNames(test.sample)
			if len(required) == 0 {
				t.Fatalf("no required fields derived for %s; the brief would list nothing", test.caste)
			}
			// An omitempty field must never be presented as required.
			for _, name := range optional {
				idx := strings.Index(brief, "- Required top-level fields: ")
				end := strings.Index(brief[idx:], "\n")
				if idx >= 0 && end > 0 && strings.Contains(brief[idx:idx+end], name) {
					t.Fatalf("optional field %q is listed as required in the staged %s brief", name, test.caste)
				}
			}
		})
	}
}

// TestLegacyBriefKeepsItsRepositoryContract is the other lane. The in-process
// whole-chain dispatch genuinely does write ROUTE-SETTER.md and phase-plan.json,
// so fixing the staged brief must not strip the instructions that lane needs.
func TestLegacyBriefKeepsItsRepositoryContract(t *testing.T) {
	spec, _ := planningStageResumeWorkerSpec("route_setter")
	brief := renderPlanningWorkerBrief(t.TempDir(), codexSurveyContext{}, spec, nil)

	for _, want := range []string{
		"Write planning outputs directly into the repository",
		"phase-plan.json",
	} {
		if !strings.Contains(brief, want) {
			t.Fatalf("legacy whole-chain brief lost %q, which that lane still requires", want)
		}
	}
	if strings.Contains(brief, "Unknown fields are REFUSED") {
		t.Fatal("legacy brief carries the staged contract, which does not apply to that lane")
	}
}
