package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const lifecycleHistory199State = `{
  "version":"1","goal":"Restore focused orientation","colony_name":"Atlas","state":"BUILT","current_phase":1,
  "milestone":"v1.28","initialized_at":"2026-09-03T08:00:00Z","build_started_at":"2026-09-03T10:00:00Z",
  "plan":{"active_revision_id":"plan-revision-199","phases":[
    {"id":1,"name":"Focused views","description":"Give each orientation command one honest closeout","status":"in_progress","tasks":[{"id":"1.1","goal":"Project history","status":"in_progress"}],"success_criteria":["History and status agree"]}
  ]},
  "events":[
    "2026-09-03T11:00:00Z|phase_completed|continue|Phase 1 checks completed",
    "2026-09-03T12:00:00Z|mystery_kind|custom-source|Opaque result retained",
    "2026-09-03T09:00:00Z|initialized|init|Colony initialized",
    "2026-09-03T12:00:00Z|build_completed|build|Build packet completed"
  ],
  "gate_results":[{"name":"history-tests","passed":true,"timestamp":"2026-09-03T11:30:00Z","detail":"history projection passed"}],
  "lifecycle_receipt":{"schema_version":"lifecycle/v1","receipt_id":"receipt-history-199","command":"plan","outcome_kind":"completed","projection_revision":"lifecycle-projection/v1","state_effect":"committed","transaction":{"id":"tx-history-199","stage":"committed"},"evidence":[{"id":"evidence-history-199","source":"plan","summary":"accepted plan recorded"}],"provenance":"confirmed"},
  "recovery_provenance":"confirmed"
}`

type lifecycleHistory199Harness struct {
	root    string
	watched []string
}

func newLifecycleHistory199Harness(t *testing.T, fixture string) lifecycleHistory199Harness {
	t.Helper()
	root, factStore, watched := seedLifecycleFactsFixture(t, fixture)
	if fixture == "valid" {
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "data", "COLONY_STATE.json"), lifecycleHistory199State)
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "data", "spawn-tree.txt"), strings.Join([]string{
			"2026-09-03T10:30:00Z|queen|builder|Mason-live|project history|1|running",
			"2026-09-03T10:20:00Z|queen|watcher|Keen-done|verify history|1|spawned",
			"2026-09-03T10:45:00Z|Keen-done|completed|history checks passed",
		}, "\n")+"\n")
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "data", "pending-decisions.json"), `{"version":"1","decisions":[]}`)
	}
	saveGlobals(t)
	resetRootCmd(t)
	resetFlags(rootCmd)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	store = factStore
	historyLimit = 20
	historyFilter = ""
	historyJSON = false
	return lifecycleHistory199Harness{root: root, watched: watched}
}

func (h lifecycleHistory199Harness) run(t *testing.T, mode string, args ...string) (string, string, error) {
	t.Helper()
	resetFlags(rootCmd)
	historyLimit = 20
	historyFilter = ""
	historyJSON = false
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	var out bytes.Buffer
	var errOut bytes.Buffer
	stdout = &out
	stderr = &errOut
	rootCmd.SetArgs(append([]string{"history"}, args...))
	err := rootCmd.Execute()
	rootCmd.SetArgs(nil)
	return out.String(), errOut.String(), err
}

func lifecycleHistory199Result(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	envelope := lifecyclePhase199Envelope(t, raw)
	if envelope["ok"] != true {
		t.Fatalf("history envelope = %#v", envelope)
	}
	return lifecyclePhase199Map(t, envelope["result"], "result")
}

func lifecycleHistory199Rows(t *testing.T, result map[string]interface{}, key string) []map[string]interface{} {
	t.Helper()
	raw := lifecyclePhase199Slice(t, result[key], key)
	rows := make([]map[string]interface{}, 0, len(raw))
	for _, value := range raw {
		rows = append(rows, lifecyclePhase199Map(t, value, key+" row"))
	}
	return rows
}

func TestLifecycleHistory199Order(t *testing.T) {
	h := newLifecycleHistory199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--json")
	if err != nil {
		t.Fatalf("history returned error: %v\nstderr: %s", err, errOut)
	}
	first := lifecycleHistory199Result(t, out)
	out, errOut, err = h.run(t, "json", "--json")
	if err != nil {
		t.Fatalf("second history returned error: %v\nstderr: %s", err, errOut)
	}
	second := lifecycleHistory199Result(t, out)
	if !reflect.DeepEqual(first["events"], second["events"]) {
		t.Fatalf("history ordering is nondeterministic\nfirst: %#v\nsecond: %#v", first["events"], second["events"])
	}

	rows := lifecycleHistory199Rows(t, first, "events")
	wantTimestamps := []string{
		"2026-09-03T12:00:00Z",
		"2026-09-03T12:00:00Z",
		"2026-09-03T11:00:00Z",
		"2026-09-03T10:45:00Z",
		"2026-09-03T10:30:00Z",
		"2026-09-03T09:00:00Z",
		"",
	}
	if len(rows) != len(wantTimestamps) {
		t.Fatalf("history rows = %d, want %d: %#v", len(rows), len(wantTimestamps), rows)
	}
	for index, want := range wantTimestamps {
		if got := rows[index]["timestamp"]; got != want {
			t.Fatalf("row %d timestamp = %v, want %q: %#v", index, got, want, rows)
		}
	}
	if rows[0]["event"] != "build completed" || rows[1]["event"] != "unknown event evidence: mystery_kind" {
		t.Fatalf("same-time tie-break changed: %#v", rows[:2])
	}
	if rows[len(rows)-1]["evidence_kind"] != "receipt" || rows[len(rows)-1]["receipt_id"] != "receipt-history-199" {
		t.Fatalf("receipt without an authoritative timestamp was lost or invented: %#v", rows[len(rows)-1])
	}
	if len(lifecycleHistory199Rows(t, first, "active_work")) != 1 {
		t.Fatalf("active work was not kept separate: %#v", first["active_work"])
	}
	if len(lifecycleHistory199Rows(t, first, "recent_outcomes")) != 2 {
		t.Fatalf("completed actor and receipt outcomes were not kept separate: %#v", first["recent_outcomes"])
	}
}

func TestLifecycleHistory199UnknownEvidence(t *testing.T) {
	h := newLifecycleHistory199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--json")
	if err != nil {
		t.Fatalf("history returned error: %v\nstderr: %s", err, errOut)
	}
	rows := lifecycleHistory199Rows(t, lifecycleHistory199Result(t, out), "events")
	for _, row := range rows {
		if row["event"] != "unknown event evidence: mystery_kind" {
			continue
		}
		if row["known"] != false || row["category"] != "unknown_evidence" {
			t.Fatalf("unknown event was presented as known/successful: %#v", row)
		}
		if row["actor"] != "Unknown" || row["evidence_source"] != "custom-source" {
			t.Fatalf("history inferred an actor from a source command: %#v", row)
		}
		if row["result"] != "Opaque result retained" || row["raw"] != "2026-09-03T12:00:00Z|mystery_kind|custom-source|Opaque result retained" {
			t.Fatalf("unknown evidence lost its human result or JSON detail: %#v", row)
		}
		return
	}
	t.Fatalf("unknown event disappeared: %#v", rows)
}

func TestLifecycleHistory199FocusedProjection(t *testing.T) {
	h := newLifecycleHistory199Harness(t, "valid")
	out, errOut, err := h.run(t, "json", "--json")
	if err != nil {
		t.Fatalf("history returned error: %v\nstderr: %s", err, errOut)
	}
	result := lifecycleHistory199Result(t, out)
	for key, want := range map[string]string{
		"schema_version":      LifecycleResultSchemaVersion,
		"command":             "history",
		"projection_revision": LifecycleProjectionRevision,
	} {
		if got := result[key]; got != want {
			t.Errorf("%s = %v, want %q", key, got, want)
		}
	}
	identity := lifecyclePhase199Map(t, lifecyclePhase199FactValue(t, result, "identity"), "identity.value")
	if identity["name"] != "Atlas" || lifecyclePhase199FactValue(t, result, "goal") != "Restore focused orientation" || lifecyclePhase199FactValue(t, result, "standing") != "BUILT" {
		t.Fatalf("shared identity/goal/standing changed: %#v", result)
	}
	for _, key := range []string{"phase", "blockers", "owner_decisions", "next_action", "alternatives", "state_effect", "history_source"} {
		if _, ok := result[key]; !ok {
			t.Errorf("focused history missing %q: %#v", key, result)
		}
	}
	next := lifecyclePhase199Map(t, result["next_action"], "next_action")
	if next["runtime_command"] != "aether continue" || next["reason"] != "The current phase has work that must be checked and advanced." {
		t.Fatalf("history changed shared Next Up: %#v", next)
	}
	if result["state_effect"] != "none" {
		t.Fatalf("read-only history claimed mutation: %#v", result)
	}

	visual, visualErr, err := h.run(t, "visual")
	if err != nil {
		t.Fatalf("history visual returned error: %v\nstderr: %s", err, visualErr)
	}
	plain := stripANSI(visual)
	for _, heading := range []string{"Colony", "Activity", "Active work", "Recent completed outcomes"} {
		if !strings.Contains(plain, heading) {
			t.Errorf("history visual missing %q\n%s", heading, plain)
		}
	}
	if !strings.Contains(plain, "aether continue") {
		t.Errorf("history visual missing the shared Next Up command\n%s", plain)
	}
	for _, forbidden := range []string{
		`{"ok":true`, `"schema_version"`,
		"2026-09-03T12:00:00Z|mystery_kind|custom-source|Opaque result retained",
	} {
		if strings.Contains(plain, forbidden) {
			t.Fatalf("history visual leaked raw envelope/detail %q\n%s", forbidden, plain)
		}
	}
}

func TestLifecycleHistory199ReadOnly(t *testing.T) {
	for _, fixture := range []string{"valid", "missing", "malformed"} {
		t.Run(fixture, func(t *testing.T) {
			h := newLifecycleHistory199Harness(t, fixture)
			before := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			_, _, _ = h.run(t, "json", "--json")
			_, _, _ = h.run(t, "visual")
			after := fingerprintLifecycleFactSurfaces(t, h.root, h.watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s history invocation mutated workspace or hub\nbefore: %#v\nafter: %#v", fixture, before, after)
			}
		})
	}
}

func TestLifecycleHistory199FixtureHasNoAccidentalFileRemoval(t *testing.T) {
	// Guard the helper itself: its direct writes must not remove the store that
	// the read-only fingerprint is intended to observe.
	h := newLifecycleHistory199Harness(t, "valid")
	if _, err := os.Stat(filepath.Join(h.root, ".aether", "data", "COLONY_STATE.json")); err != nil {
		t.Fatal(err)
	}
}
