package cmd

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestBlockerSnapshot(t *testing.T) {
	saveGlobals(t)
	tests := []struct {
		name  string
		write bool
		flags []colony.FlagEntry
		want  blockerSnapshot
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
			name:  "nil store",
			write: false,
			want:  blockerSnapshot{IDs: []string{}, EscalatedIDs: []string{}},
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

			if got := readBlockerSnapshot(s); !reflect.DeepEqual(got, tt.want) {
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
