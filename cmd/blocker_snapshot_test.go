package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func callBlockerSnapshot(t *testing.T, s *storage.Store) (blockerSnapshot, error) {
	t.Helper()
	reader := reflect.ValueOf(readBlockerSnapshot)
	if reader.Type().NumOut() != 2 {
		t.Fatalf("readBlockerSnapshot must return (blockerSnapshot, error), got %d results", reader.Type().NumOut())
	}
	results := reader.Call([]reflect.Value{reflect.ValueOf(s)})
	snapshot := results[0].Interface().(blockerSnapshot)
	if results[1].IsNil() {
		return snapshot, nil
	}
	return snapshot, results[1].Interface().(error)
}

func TestBlockerSnapshotReportsStorageAvailability(t *testing.T) {
	t.Run("both files absent is an available empty snapshot", func(t *testing.T) {
		s, _ := newTestStore(t)
		got, err := callBlockerSnapshot(t, s)
		if err != nil {
			t.Fatalf("read empty snapshot: %v", err)
		}
		want := blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("snapshot = %#v, want %#v", got, want)
		}
	})

	t.Run("nil store is unavailable", func(t *testing.T) {
		_, err := callBlockerSnapshot(t, nil)
		if err == nil {
			t.Fatal("nil store reported available blocker truth")
		}
	})

	t.Run("malformed current file never falls back", func(t *testing.T) {
		s, root := newTestStore(t)
		if err := s.SaveJSON("pending-decisions.json", colony.FlagsFile{Version: "1.0", Decisions: []colony.FlagEntry{{ID: "legacy", Type: "blocker"}}}); err != nil {
			t.Fatalf("write current fixture: %v", err)
		}
		if err := os.Rename(filepath.Join(root, ".aether/data/pending-decisions.json"), filepath.Join(root, ".aether/data/flags.json")); err != nil {
			t.Fatalf("move fixture to legacy path: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, ".aether/data/pending-decisions.json"), []byte("{"), 0o600); err != nil {
			t.Fatalf("write malformed current file: %v", err)
		}
		_, err := callBlockerSnapshot(t, s)
		if err == nil {
			t.Fatal("malformed current blocker truth was hidden by legacy fallback")
		}
		if !strings.Contains(err.Error(), "pending-decisions.json") {
			t.Fatalf("error lacks data-relative context: %v", err)
		}
	})

	t.Run("missing current uses valid legacy file", func(t *testing.T) {
		s, root := newTestStore(t)
		if err := s.SaveJSON("pending-decisions.json", colony.FlagsFile{Version: "1.0", Decisions: []colony.FlagEntry{{ID: "legacy", Type: "blocker"}}}); err != nil {
			t.Fatalf("write current fixture: %v", err)
		}
		if err := os.Rename(filepath.Join(root, ".aether/data/pending-decisions.json"), filepath.Join(root, ".aether/data/flags.json")); err != nil {
			t.Fatalf("move fixture to legacy path: %v", err)
		}
		got, err := callBlockerSnapshot(t, s)
		if err != nil {
			t.Fatalf("read legacy snapshot: %v", err)
		}
		if got.Count != 1 || !reflect.DeepEqual(got.IDs, []string{"legacy"}) {
			t.Fatalf("legacy snapshot = %+v", got)
		}
	})

	t.Run("missing current with malformed legacy is unavailable", func(t *testing.T) {
		s, root := newTestStore(t)
		if err := os.WriteFile(filepath.Join(root, ".aether/data/flags.json"), []byte("{"), 0o600); err != nil {
			t.Fatalf("write malformed legacy file: %v", err)
		}
		_, err := callBlockerSnapshot(t, s)
		if err == nil {
			t.Fatal("malformed legacy blocker truth reported empty")
		}
		if !strings.Contains(err.Error(), "flags.json") {
			t.Fatalf("error lacks legacy data-relative context: %v", err)
		}
	})

	t.Run("non-regular current file is unavailable", func(t *testing.T) {
		s, root := newTestStore(t)
		if err := os.Mkdir(filepath.Join(root, ".aether/data/pending-decisions.json"), 0o700); err != nil {
			t.Fatalf("create non-regular current path: %v", err)
		}
		_, err := callBlockerSnapshot(t, s)
		if err == nil {
			t.Fatal("non-regular blocker truth reported empty")
		}
	})
}

func TestStatusReportsUnavailableBlockerTruth(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	store = s
	path := filepath.Join(root, ".aether/data/pending-decisions.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatalf("write malformed blocker truth: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	result := buildStatusResult(state, s)
	if result["blocker_snapshot_available"] != false {
		t.Fatalf("availability = %#v, want false", result["blocker_snapshot_available"])
	}
	if result["blockers"] != nil || result["blocker_ids"] != nil || result["escalated_blockers"] != nil {
		t.Fatalf("status invented blocker facts: %#v", result)
	}
	detail, _ := result["blocker_snapshot_error"].(string)
	if detail == "" || !strings.Contains(detail, "pending-decisions.json") || strings.Contains(detail, root) {
		t.Fatalf("unsafe or unhelpful diagnostic %q", detail)
	}
	visual := renderDashboard(state, s, result)
	if !strings.Contains(visual, "Blocker truth: unavailable") {
		t.Fatalf("visual status hides unavailable truth:\n%s", visual)
	}
	if strings.Contains(visual, "Flags: 0 blockers") || strings.Contains(visual, "Existing blocker work: 0 active") {
		t.Fatalf("visual status converted unavailable truth to zero:\n%s", visual)
	}
}

func TestFlagCheckBlockersFailsWhenTruthUnavailable(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)
	s, root := setupTestStore(t)
	t.Setenv("AETHER_ROOT", root)
	store = s
	if err := os.WriteFile(filepath.Join(root, ".aether/data/pending-decisions.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write malformed blocker truth: %v", err)
	}
	var out, errOut bytes.Buffer
	stdout, stderr = &out, &errOut
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"flag-check-blockers"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute diagnostic: %v", err)
	}
	if code := renderedCommandExitCode.Load(); code == 0 {
		t.Fatalf("diagnostic succeeded with unavailable truth: stdout=%s stderr=%s", out.String(), errOut.String())
	}
	envelope := parseEnvelope(t, errOut.String())
	if envelope["ok"] != false {
		t.Fatalf("error envelope = %#v", envelope)
	}
	details, _ := envelope["details"].(map[string]interface{})
	if details["blocker_snapshot_available"] != false {
		t.Fatalf("availability details = %#v", details)
	}
	for _, forbidden := range []string{"has_blockers", "blockers", "issues", "notes"} {
		if _, exists := details[forbidden]; exists {
			t.Errorf("unavailable diagnostic invented %s: %#v", forbidden, details)
		}
	}
}

func TestBlockerSnapshot(t *testing.T) {
	saveGlobals(t)
	tests := []struct {
		name    string
		write   bool
		flags   []colony.FlagEntry
		want    blockerSnapshot
		wantErr bool
	}{
		{
			name: "missing flags file",
			want: blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}},
		},
		{
			name:  "empty flags file",
			write: true,
			want:  blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}},
		},
		{
			name:    "nil store",
			write:   false,
			want:    blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}},
			wantErr: true,
		},
		{
			name:  "ordinary and escalated blockers are sorted",
			write: true,
			flags: []colony.FlagEntry{
				{ID: "blocker-z", Type: "blocker"},
				{ID: "blocker-a", Type: "blocker", Source: "escalation"},
				{ID: "blocker-m", Type: "blocker"},
			},
			want: blockerSnapshot{
				Count:          3,
				IDs:            []string{"blocker-a", "blocker-m", "blocker-z"},
				EscalatedCount: 1,
				EscalatedIDs:   []string{"blocker-a"},
			},
		},
		{
			name:  "resolved and neighboring records are excluded",
			write: true,
			flags: []colony.FlagEntry{
				{ID: "resolved", Type: "blocker", Source: "escalation", Resolved: true},
				{ID: "issue", Type: "issue"},
				{ID: "note", Type: "note"},
				{ID: "owner-checkpoint", Type: "owner_confirmation"},
				{ID: "case-variant", Type: "Blocker", Source: "escalation"},
			},
			want: blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}},
		},
		{
			name:  "duplicate and blank IDs do not destabilize evidence",
			write: true,
			flags: []colony.FlagEntry{
				{ID: "duplicate", Type: "blocker"},
				{ID: "duplicate", Type: "blocker"},
				{ID: "", Type: "blocker", Source: "escalation"},
			},
			want: blockerSnapshot{
				Count:          3,
				IDs:            []string{"duplicate"},
				EscalatedCount: 1,
				EscalatedIDs:   []string{},
			},
		},
		{
			name:  "source matching is exact",
			write: true,
			flags: []colony.FlagEntry{
				{ID: "ordinary", Type: "blocker", Source: "Escalation"},
			},
			want: blockerSnapshot{
				Count:        1,
				IDs:          []string{"ordinary"},
				EscalatedIDs: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s *storage.Store
			if tt.name != "nil store" {
				s, _ = newTestStore(t)
				if tt.write {
					store = s
					writeTestFlags(t, tt.flags...)
				}
			}

			got, err := readBlockerSnapshot(s)
			if (err != nil) != tt.wantErr {
				t.Fatalf("readBlockerSnapshot() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("readBlockerSnapshot() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBlockerSnapshotComparison(t *testing.T) {
	tests := []struct {
		name   string
		before blockerSnapshot
		after  blockerSnapshot
		want   blockerSnapshotComparison
	}{
		{
			name:   "same-count ordinary ID replacement",
			before: blockerSnapshot{Count: 1, IDs: []string{"old"}, EscalatedIDs: []string{}},
			after:  blockerSnapshot{Count: 1, IDs: []string{"replacement"}, EscalatedIDs: []string{}},
			want:   blockerSnapshotComparison{AddedEscalatedIDs: []string{}},
		},
		{
			name:   "ordinary blocker added",
			before: blockerSnapshot{Count: 1, IDs: []string{"old"}, EscalatedIDs: []string{}},
			after:  blockerSnapshot{Count: 2, IDs: []string{"new", "old"}, EscalatedIDs: []string{}},
			want: blockerSnapshotComparison{
				CountIncreased:    true,
				AddedEscalatedIDs: []string{},
			},
		},
		{
			name:   "existing blocker escalated",
			before: blockerSnapshot{Count: 1, IDs: []string{"old"}, EscalatedIDs: []string{}},
			after: blockerSnapshot{
				Count:          1,
				IDs:            []string{"old"},
				EscalatedCount: 1,
				EscalatedIDs:   []string{"old"},
			},
			want: blockerSnapshotComparison{
				EscalationAdded:   true,
				AddedEscalatedIDs: []string{"old"},
			},
		},
		{
			name:   "unidentified escalation count added",
			before: blockerSnapshot{EscalatedIDs: []string{}},
			after:  blockerSnapshot{Count: 1, EscalatedCount: 1, EscalatedIDs: []string{}},
			want: blockerSnapshotComparison{
				CountIncreased:    true,
				EscalationAdded:   true,
				AddedEscalatedIDs: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareBlockerSnapshots(tt.before, tt.after); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("compareBlockerSnapshots() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestBlockerSnapshotObservesSwarmEscalation(t *testing.T) {
	s, _ := newTestStore(t)
	base := time.Date(2026, time.August, 31, 15, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		if _, err := persistSwarmResultOutcome(s, swarmResultRecord{
			SwarmID:     []string{"swarm-1", "swarm-2", "swarm-3"}[i],
			Target:      "auth panic",
			Status:      "failed",
			CompletedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("persist strike %d: %v", i+1, err)
		}
	}

	flags := activeSwarmEscalationFlags(s)
	if len(flags) != 1 {
		t.Fatalf("active escalation flags = %d, want 1: %+v", len(flags), flags)
	}
	snapshot, err := readBlockerSnapshot(s)
	if err != nil {
		t.Fatalf("read blocker snapshot: %v", err)
	}
	if snapshot.Count != 1 || snapshot.EscalatedCount != 1 {
		t.Fatalf("snapshot = %+v, want one escalated blocker", snapshot)
	}
	if !reflect.DeepEqual(snapshot.IDs, []string{flags[0].ID}) || !reflect.DeepEqual(snapshot.EscalatedIDs, []string{flags[0].ID}) {
		t.Fatalf("snapshot IDs = %+v, want escalation %q", snapshot, flags[0].ID)
	}
}

func TestFlagCheckBlockersAndStatusShareSnapshot(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	t.Cleanup(func() { store = nil })
	t.Setenv("AETHER_ROOT", root)
	store = s
	writeTestFlags(t, blockerStatusFixture()...)

	var buf bytes.Buffer
	stdout = &buf
	rootCmd.SetArgs([]string{"flag-check-blockers"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-check-blockers: %v", err)
	}
	diagnostic := parseEnvelope(t, buf.String())["result"].(map[string]interface{})

	buf.Reset()
	rootCmd.SetArgs([]string{"status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("status: %v", err)
	}
	status := parseEnvelope(t, buf.String())["result"].(map[string]interface{})

	for _, field := range []string{"blockers", "blocker_ids", "escalated_blockers", "escalated_blocker_ids"} {
		if !reflect.DeepEqual(status[field], diagnostic[field]) {
			t.Errorf("status[%q] = %#v, diagnostic = %#v", field, status[field], diagnostic[field])
		}
	}
	if diagnostic["blockers"] != float64(2) || diagnostic["escalated_blockers"] != float64(1) {
		t.Fatalf("diagnostic snapshot = %#v, want 2 blockers and 1 escalation", diagnostic)
	}
	// The legacy diagnostic counts unknown non-blocker rows (including the
	// owner-checkpoint-shaped fixture) as issues. That compatibility behavior
	// remains separate from blocker-snapshot membership.
	if diagnostic["issues"] != float64(2) || diagnostic["notes"] != float64(0) || diagnostic["has_blockers"] != true {
		t.Fatalf("compatibility fields changed: %#v", diagnostic)
	}
}

func TestStatusEscalatedBlockersVisual(t *testing.T) {
	saveGlobals(t)
	s, root := setupTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	writeTestFlags(t, blockerStatusFixture()...)

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	result := buildStatusResult(state, s)
	output := renderDashboard(state, s, result)

	if !strings.Contains(output, "Existing blocker work: 2 active (1 escalated)") {
		t.Fatalf("status must name existing and escalated blocker work plainly:\n%s", output)
	}
	if strings.Contains(strings.ToLower(output), "automatic stop") {
		t.Fatalf("existing ordinary blockers must not be described as an automatic stop:\n%s", output)
	}
}

func blockerStatusFixture() []colony.FlagEntry {
	return []colony.FlagEntry{
		{ID: "blocker-b", Type: "blocker"},
		{ID: "blocker-a", Type: "blocker", Source: "escalation"},
		{ID: "resolved", Type: "blocker", Source: "escalation", Resolved: true},
		{ID: "issue", Type: "issue"},
		{ID: "owner-checkpoint", Type: "owner_confirmation"},
	}
}
