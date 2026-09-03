package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const lifecyclePhase199State = `{
  "version":"1","goal":"Restore focused orientation","colony_name":"Atlas","state":"BUILT","current_phase":2,
  "milestone":"v1.28","initialized_at":"2026-09-03T08:00:00Z","build_started_at":"2026-09-03T10:00:00Z",
  "plan":{"active_revision_id":"plan-revision-199","phases":[
    {"id":1,"name":"Foundation","description":"Establish lifecycle facts","status":"completed","tasks":[{"id":"1.1","goal":"Load facts once","status":"completed","success_criteria":["Facts are immutable"]}],"success_criteria":["The fact bundle is durable"]},
    {"id":2,"name":"Focused views","description":"Give each orientation command one honest closeout","status":"in_progress","tasks":[{"id":"2.1","goal":"Render phase detail","status":"in_progress","depends_on":["1.1"],"success_criteria":["Focused detail is complete"]},{"id":"2.2","goal":"Prove selector order","status":"pending","depends_on":["2.1"],"success_criteria":["List order is deterministic"]}],"success_criteria":["Phase and status agree","Missing evidence is named"]},
    {"id":3,"name":"Agreement","description":"Ratchet shared orientation semantics","status":"pending","tasks":[{"id":"3.1","goal":"Compare views","status":"pending","depends_on":["2.2"]}],"success_criteria":["Every view shares one answer"]}
  ]},
  "events":["2026-09-03T10:00:00Z|plan|accepted"],
  "gate_results":[{"name":"focused-tests","passed":true,"timestamp":"2026-09-03T11:00:00Z","detail":"focused projection passed"}],
  "lifecycle_receipt":{"schema_version":"lifecycle/v1","receipt_id":"receipt-phase-199","command":"build","outcome_kind":"in_progress","state_effect":"committed","transaction":{"id":"tx-phase-199","stage":"committed"},"evidence":[{"id":"evidence-phase-199","source":"build","summary":"attempt evidence recorded","provenance":"confirmed"}],"provenance":"confirmed"},
  "recovery_provenance":"confirmed"
}`

type lifecyclePhase199Harness struct {
	root    string
	watched []string
}

func newLifecyclePhase199Harness(t *testing.T, fixture string) lifecyclePhase199Harness {
	t.Helper()
	root, factStore, watched := seedLifecycleFactsFixture(t, fixture)
	if fixture == "valid" {
		if err := os.WriteFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"), []byte(lifecyclePhase199State), 0o644); err != nil {
			t.Fatalf("write phase fixture: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".aether", "data", "pending-decisions.json"), []byte(`{"version":"1","decisions":[]}`), 0o644); err != nil {
			t.Fatalf("clear phase fixture blockers: %v", err)
		}
	}
	saveGlobals(t)
	resetRootCmd(t)
	resetFlags(rootCmd)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	store = factStore
	phaseNumber = 0
	phaseJSON = false
	return lifecyclePhase199Harness{root: root, watched: watched}
}

func (h lifecyclePhase199Harness) run(t *testing.T, mode string, args ...string) (string, string, error) {
	t.Helper()
	resetFlags(rootCmd)
	phaseNumber = 0
	phaseJSON = false
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	var out bytes.Buffer
	var errOut bytes.Buffer
	stdout = &out
	stderr = &errOut
	rootCmd.SetArgs(append([]string{"phase"}, args...))
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return out.String(), errOut.String(), err
}

func lifecyclePhase199Envelope(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &envelope); err != nil {
		t.Fatalf("decode phase envelope: %v\n%s", err, raw)
	}
	return envelope
}

func lifecyclePhase199Map(t *testing.T, value interface{}, label string) map[string]interface{} {
	t.Helper()
	result, ok := value.(map[string]interface{})
	if !ok {
		t.Fatalf("%s = %T, want object", label, value)
	}
	return result
}

func lifecyclePhase199Slice(t *testing.T, value interface{}, label string) []interface{} {
	t.Helper()
	result, ok := value.([]interface{})
	if !ok {
		t.Fatalf("%s = %T, want array", label, value)
	}
	return result
}

func lifecyclePhase199FactValue(t *testing.T, object map[string]interface{}, key string) interface{} {
	t.Helper()
	fact := lifecyclePhase199Map(t, object[key], key)
	if _, ok := fact["source"]; !ok {
		t.Fatalf("%s does not expose provenance: %#v", key, fact)
	}
	return fact["value"]
}

func TestLifecyclePhase199FocusedProjection(t *testing.T) {
	h := newLifecyclePhase199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--json")
	if err != nil {
		t.Fatalf("phase detail returned error: %v\nstderr: %s", err, errOut)
	}
	envelope := lifecyclePhase199Envelope(t, out)
	if envelope["ok"] != true {
		t.Fatalf("phase detail envelope = %#v", envelope)
	}
	result := lifecyclePhase199Map(t, envelope["result"], "result")
	for key, want := range map[string]string{
		"schema_version":      LifecycleResultSchemaVersion,
		"command":             "phase",
		"projection_revision": LifecycleProjectionRevision,
		"selection":           "detail",
	} {
		if got := result[key]; got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	identity := lifecyclePhase199Map(t, lifecyclePhase199FactValue(t, result, "identity"), "identity.value")
	if identity["name"] != "Atlas" || lifecyclePhase199FactValue(t, result, "goal") != "Restore focused orientation" || lifecyclePhase199FactValue(t, result, "standing") != "BUILT" {
		t.Fatalf("shared identity/goal/standing changed: %#v", result)
	}
	detail := lifecyclePhase199Map(t, result["phase"], "phase")
	if detail["number"] != float64(2) || detail["name"] != "Focused views" || detail["objective"] != "Give each orientation command one honest closeout" {
		t.Fatalf("focused phase identity = %#v", detail)
	}
	dependencies := lifecyclePhase199Slice(t, lifecyclePhase199FactValue(t, detail, "dependencies"), "dependencies.value")
	if !reflect.DeepEqual(dependencies, []interface{}{"1.1", "2.1"}) {
		t.Fatalf("dependencies = %#v, want accepted-plan task dependencies", dependencies)
	}
	criteria := lifecyclePhase199Slice(t, lifecyclePhase199FactValue(t, detail, "success_criteria"), "success_criteria.value")
	if len(criteria) != 2 {
		t.Fatalf("success criteria = %#v", criteria)
	}
	if len(lifecyclePhase199Slice(t, lifecyclePhase199FactValue(t, detail, "tasks"), "tasks.value")) != 2 {
		t.Fatalf("tasks missing from focused detail: %#v", detail)
	}
	for _, key := range []string{"attempt", "actors", "verification", "evidence", "blockers"} {
		if _, ok := detail[key]; !ok {
			t.Errorf("focused phase missing %q: %#v", key, detail)
		}
	}
	next := lifecyclePhase199Map(t, result["next_action"], "next_action")
	if next["runtime_command"] != "aether continue" || next["reason"] != "The current phase has work that must be checked and advanced." {
		t.Fatalf("phase changed shared Next Up: %#v", next)
	}

	visual, visualErr, err := h.run(t, "visual")
	if err != nil {
		t.Fatalf("phase visual returned error: %v\nstderr: %s", err, visualErr)
	}
	plain := stripANSI(visual)
	for _, heading := range []string{"Colony", "Objective", "Dependencies", "Tasks", "Success criteria", "Current attempt & ants", "Verification & evidence", "Blockers", "Next Up"} {
		if !strings.Contains(plain, heading) {
			t.Errorf("focused phase visual missing %q\n%s", heading, plain)
		}
	}
	for _, forbidden := range []string{"Pheromones", "Territory & Notes", "Memory, Findings & Gates", "Elapsed & Reported Cost", "Recent History"} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("focused phase expanded into status section %q\n%s", forbidden, plain)
		}
	}
}

func TestLifecyclePhase199List(t *testing.T) {
	h := newLifecyclePhase199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--list", "--json")
	if err != nil {
		t.Fatalf("phase --list returned error: %v\nstderr: %s", err, errOut)
	}
	result := lifecyclePhase199Map(t, lifecyclePhase199Envelope(t, out)["result"], "result")
	if result["selection"] != "list" {
		t.Fatalf("selection = %v, want list", result["selection"])
	}
	rows := lifecyclePhase199Slice(t, result["phases"], "phases")
	if len(rows) != 3 {
		t.Fatalf("list rows = %d, want 3", len(rows))
	}
	for index, raw := range rows {
		row := lifecyclePhase199Map(t, raw, "phase row")
		if row["number"] != float64(index+1) {
			t.Fatalf("list order = %#v", rows)
		}
		for _, key := range []string{"name", "state", "blocked", "verification", "attempt"} {
			if _, ok := row[key]; !ok {
				t.Errorf("phase row missing %q: %#v", key, row)
			}
		}
	}
}

func TestLifecyclePhase199All(t *testing.T) {
	h := newLifecyclePhase199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--all", "--json")
	if err != nil {
		t.Fatalf("phase --all returned error: %v\nstderr: %s", err, errOut)
	}
	result := lifecyclePhase199Map(t, lifecyclePhase199Envelope(t, out)["result"], "result")
	if result["selection"] != "all" {
		t.Fatalf("selection = %v, want all", result["selection"])
	}
	details := lifecyclePhase199Slice(t, result["phases"], "phases")
	if len(details) != 3 {
		t.Fatalf("all details = %d, want 3", len(details))
	}
	for index, raw := range details {
		detail := lifecyclePhase199Map(t, raw, "phase detail")
		if detail["number"] != float64(index+1) {
			t.Fatalf("all order = %#v", details)
		}
		for _, key := range []string{"objective", "dependencies", "tasks", "success_criteria", "attempt", "actors", "verification", "evidence", "blockers"} {
			if _, ok := detail[key]; !ok {
				t.Errorf("complete phase detail missing %q: %#v", key, detail)
			}
		}
	}
}

func TestLifecyclePhase199SelectorValidation(t *testing.T) {
	for _, args := range [][]string{
		{"--list", "--all", "--json"},
		{"--number", "2", "--list", "--json"},
		{"--number", "2", "--all", "--json"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			h := newLifecyclePhase199Harness(t, "valid")
			before := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			_, errOut, err := h.run(t, "json", args...)
			if err != nil {
				t.Fatalf("selector validation escaped the command contract: %v", err)
			}
			envelope := lifecyclePhase199Envelope(t, errOut)
			if envelope["ok"] != false || !strings.Contains(strings.ToLower(envelope["error"].(string)), "mutually exclusive") {
				t.Fatalf("selector error = %#v", envelope)
			}
			after := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("selector error mutated workspace or hub\nbefore: %#v\nafter: %#v", before, after)
			}
		})
	}
}

func TestLifecyclePhase199MissingPhase(t *testing.T) {
	t.Run("unknown number", func(t *testing.T) {
		h := newLifecyclePhase199Harness(t, "valid")
		_, errOut, err := h.run(t, "json", "--number", "99", "--json")
		if err != nil {
			t.Fatalf("missing phase escaped command contract: %v", err)
		}
		envelope := lifecyclePhase199Envelope(t, errOut)
		if envelope["ok"] != false || !strings.Contains(envelope["error"].(string), "phase 99 not found") {
			t.Fatalf("missing phase error = %#v", envelope)
		}
		details := lifecyclePhase199Map(t, envelope["details"], "details")
		next := lifecyclePhase199Map(t, details["next_action"], "next_action")
		if next["runtime_command"] != "aether continue" || strings.TrimSpace(next["reason"].(string)) == "" {
			t.Fatalf("missing phase lost canonical recovery action: %#v", details)
		}
	})

	t.Run("no accepted plan", func(t *testing.T) {
		h := newLifecyclePhase199Harness(t, "missing")
		_, errOut, err := h.run(t, "json", "--json")
		if err != nil {
			t.Fatalf("empty phase escaped command contract: %v", err)
		}
		envelope := lifecyclePhase199Envelope(t, errOut)
		if envelope["ok"] != false || !strings.Contains(strings.ToLower(envelope["error"].(string)), "no accepted phases") {
			t.Fatalf("empty phase error = %#v", envelope)
		}
		details := lifecyclePhase199Map(t, envelope["details"], "details")
		next := lifecyclePhase199Map(t, details["next_action"], "next_action")
		if next["runtime_command"] != `aether init "goal"` {
			t.Fatalf("empty phase recovery = %#v, want init", next)
		}
	})
}

func TestLifecyclePhase199ReadOnly(t *testing.T) {
	for _, fixture := range []string{"valid", "missing", "malformed"} {
		t.Run(fixture, func(t *testing.T) {
			h := newLifecyclePhase199Harness(t, fixture)
			before := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			for _, args := range [][]string{{"--json"}, {"--list", "--json"}, {"--all", "--json"}, {"--number", "99", "--json"}} {
				_, _, _ = h.run(t, "json", args...)
			}
			after := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s phase invocation mutated workspace or hub\nbefore: %#v\nafter: %#v", fixture, before, after)
			}
		})
	}
}
