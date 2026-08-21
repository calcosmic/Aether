package cmd

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// dispatchContractPlanningFallbackExceptionFields are the two dispatch
// contract fields where 191-02 deliberately did NOT fold
// colony/policies/dispatch-contract.yaml's (now-deleted) stale planning
// value into the Go-compiled default. See the doc comments on
// fallbackPlanningFallbackBehavior and fallbackPlanningFallbackVisibility
// (cmd/codex_dispatch_contract.go) and 191-02-SUMMARY.md for the full
// evidence trail: three independent, pre-existing, committed sources
// (TestPlanIncludesDispatchContract, TestPlanVisualOutputShowsDispatchContractDetails,
// cmd/testdata/golden_plan.txt) all pin the CURRENT Go fallback -- a
// deliberate fail-closed design -- as correct, proving the YAML file was
// itself stale, not the Go code.
var dispatchContractPlanningFallbackExceptionFields = []string{"fallback_behavior", "fallback_visibility"}

// assertDispatchContractFieldsIdentical compares two dispatch contract maps
// field by field, skipping any field named in exceptKeys. Skipping (not
// ignoring the whole map) means every OTHER field still gets the full
// byte-identical proof 191-02's fold-then-delete process requires --
// the exception is scoped as narrowly as the evidence supports.
func assertDispatchContractFieldsIdentical(t *testing.T, label string, before, after map[string]interface{}, exceptKeys ...string) {
	t.Helper()
	except := make(map[string]bool, len(exceptKeys))
	for _, k := range exceptKeys {
		except[k] = true
	}
	seen := make(map[string]bool)
	for k, bv := range before {
		seen[k] = true
		if except[k] {
			continue
		}
		av, ok := after[k]
		if !ok {
			t.Errorf("%s: field %q present before deletion, missing after", label, k)
			continue
		}
		if !reflect.DeepEqual(bv, av) {
			t.Errorf("%s: field %q is NOT identical before vs after deletion\nBEFORE: %#v\nAFTER:  %#v", label, k, bv, av)
		}
	}
	for k := range after {
		if !seen[k] && !except[k] {
			t.Errorf("%s: field %q present after deletion, missing before", label, k)
		}
	}
}

// TestDispatchContractYamlDeletionProducesByteIdenticalOutput is 191-02's
// Layer 1 (Go-level) proof for loadDispatchContractPolicy(): colony/policies/
// dispatch-contract.yaml's real, pre-deletion content -- captured verbatim in
// cmd/testdata/191-02-dispatch-contract-original.yaml before this task
// touched anything -- must produce the same survey and planning dispatch
// contracts as the post-deletion state (dispatchContractPathOverride pointed
// at a path that does not exist, forcing every field onto its Go-compiled
// fallback constant) for every field EXCEPT the two documented,
// deliberate exceptions in dispatchContractPlanningFallbackExceptionFields.
//
// resetDispatchPolicyState (cmd/dispatch_contract_test.go) is reused
// unchanged -- both files are part of the same "cmd" test binary, so no
// second reset helper is needed (191-02-PLAN.md's read_first flagged this as
// something to confirm, not assume).
func TestDispatchContractYamlDeletionProducesByteIdenticalOutput(t *testing.T) {
	originalPath, err := filepath.Abs(filepath.Join("testdata", "191-02-dispatch-contract-original.yaml"))
	if err != nil {
		t.Fatalf("resolve testdata path: %v", err)
	}

	// Render 1: the real, original dispatch-contract.yaml content, frozen in
	// testdata so this test does not depend on the live file's continued
	// existence -- it must still compile and pass once that file is gone.
	resetDispatchPolicyState()
	dispatchContractPathOverride = originalPath
	defer func() { dispatchContractPathOverride = "" }()

	beforeSurvey := surveyDispatchContractWithTimeout(0)
	beforePlanning := planningDispatchContractWithTimeout(0)
	beforeExtended := planningDispatchContractForDispatches([]codexPlanningDispatch{{}, {}, {}}, 0)

	if renderDispatchContract(beforeSurvey) == "" || renderDispatchContract(beforePlanning) == "" || renderDispatchContract(beforeExtended) == "" {
		t.Fatal("before-render produced empty rendered contract text -- fixture did not exercise the loader")
	}

	// Confirm the fixture actually contains the stale YAML text this test
	// exists to document -- otherwise the "except" list below would be
	// silently vacuous (excusing a field that never actually diverged).
	if got := stringValue(beforePlanning["fallback_behavior"]); !strings.Contains(got, "synthesize planning artifacts locally") {
		t.Fatalf("testdata fixture's planning fallback_behavior = %q, expected the stale colony/policies/dispatch-contract.yaml text this test documents as superseded", got)
	}

	// Render 2: simulate dispatch-contract.yaml having been deleted -- every
	// field falls back to its Go-compiled default.
	resetDispatchPolicyState()
	dispatchContractPathOverride = filepath.Join(t.TempDir(), "nonexistent-dispatch-contract.yaml")

	afterSurvey := surveyDispatchContractWithTimeout(0)
	afterPlanning := planningDispatchContractWithTimeout(0)
	afterExtended := planningDispatchContractForDispatches([]codexPlanningDispatch{{}, {}, {}}, 0)

	if renderDispatchContract(afterSurvey) == "" || renderDispatchContract(afterPlanning) == "" || renderDispatchContract(afterExtended) == "" {
		t.Fatal("after-render produced empty rendered contract text -- fixture did not exercise the loader")
	}

	// Survey has no documented exceptions -- full byte-identical proof.
	assertDispatchContractFieldsIdentical(t, "survey", beforeSurvey, afterSurvey)

	// Planning: every field except the two documented exceptions must be
	// byte-identical. planningDispatchContractForDispatches's "extended"
	// path recomputes execution_model/dependency_behavior but leaves
	// fallback_behavior/fallback_visibility exactly as
	// planningDispatchContractWithTimeout set them, so the same exception
	// list applies there too.
	assertDispatchContractFieldsIdentical(t, "planning", beforePlanning, afterPlanning, dispatchContractPlanningFallbackExceptionFields...)
	assertDispatchContractFieldsIdentical(t, "planning (extended)", beforeExtended, afterExtended, dispatchContractPlanningFallbackExceptionFields...)

	// The two exception fields: assert the AFTER (post-deletion, permanent)
	// value is the tested, golden-fixture-locked fail-closed text/list --
	// not merely "different from before," but specifically the value
	// TestPlanIncludesDispatchContract, TestPlanVisualOutputShowsDispatchContractDetails
	// and cmd/testdata/golden_plan.txt already require.
	afterBehavior := stringValue(afterPlanning["fallback_behavior"])
	if !strings.Contains(afterBehavior, "does not fall back to local synthesis") || !strings.Contains(afterBehavior, "aether plan --synthetic") {
		t.Errorf("after-deletion planning fallback_behavior = %q, want the fail-closed synthetic guidance TestPlanIncludesDispatchContract and the golden fixture require", afterBehavior)
	}
	afterVisibility := stringSliceValue(afterPlanning["fallback_visibility"])
	for _, want := range []string{"synthetic", "synthetic_warning"} {
		if !containsString(afterVisibility, want) {
			t.Errorf("after-deletion planning fallback_visibility = %v, missing required entry %q", afterVisibility, want)
		}
	}

	// And confirm the BEFORE value was indeed the stale, different text --
	// documenting the exception is real, not a copy-paste no-op.
	if strings.Contains(stringValue(beforePlanning["fallback_behavior"]), "does not fall back to local synthesis") {
		t.Error("before-deletion (real YAML file) planning fallback_behavior unexpectedly already matches the fail-closed text -- the documented exception may no longer be necessary; re-verify against cmd/testdata/191-02-dispatch-contract-original.yaml and remove the exception if so")
	}
}
