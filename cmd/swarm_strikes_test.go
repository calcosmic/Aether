package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestSwarmTargetFingerprint(t *testing.T) {
	t.Parallel()

	canonical := swarmTargetFingerprint("Fix the AUTH session bug")
	if canonical == "" {
		t.Fatal("fingerprint is empty for a real target")
	}
	if len(canonical) != 64 {
		t.Fatalf("fingerprint length = %d, want a full SHA-256 digest", len(canonical))
	}

	equivalent := []string{
		"fix the auth session bug",
		"  FIX   THE\tAUTH\nSESSION BUG  ",
		"...Fix the AUTH session bug!!!",
		"\"Fix the AUTH session bug\"",
	}
	for _, target := range equivalent {
		if got := swarmTargetFingerprint(target); got != canonical {
			t.Errorf("fingerprint(%q) = %q, want %q", target, got, canonical)
		}
	}

	for _, distinct := range []string{
		"Fix the authorization session bug",
		"Fix the AUTH session timeout",
		"Fix the AUTH-session bug",
	} {
		if got := swarmTargetFingerprint(distinct); got == canonical {
			t.Errorf("materially different target %q shared fingerprint %q", distinct, got)
		}
	}

	if got := swarmTargetFingerprint(" ... !!! "); got != "" {
		t.Fatalf("punctuation-only target fingerprint = %q, want empty", got)
	}
}

func TestSwarmStrikeHistory(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		records    []swarmStrikeFixture
		target     string
		wantCount  int
		wantNext   int
		wantIDs    []string
		wantStatus []string
	}{
		{
			name:      "zero prior results",
			target:    "auth panic",
			wantCount: 0,
			wantNext:  1,
		},
		{
			name: "one failure",
			records: []swarmStrikeFixture{
				{id: "swarm-1", target: "auth panic", status: "failed", at: base},
			},
			target:     "AUTH   PANIC",
			wantCount:  1,
			wantNext:   2,
			wantIDs:    []string{"swarm-1"},
			wantStatus: []string{"failed"},
		},
		{
			name: "two failures",
			records: []swarmStrikeFixture{
				{id: "swarm-1", target: "auth panic", status: "failed", at: base},
				{id: "swarm-2", target: "auth panic", status: "failed", at: base.Add(time.Minute)},
			},
			target:     "auth panic",
			wantCount:  2,
			wantNext:   3,
			wantIDs:    []string{"swarm-1", "swarm-2"},
			wantStatus: []string{"failed", "failed"},
		},
		{
			name: "three failures including blocked",
			records: []swarmStrikeFixture{
				{id: "swarm-1", target: "auth panic", status: "failed", at: base},
				{id: "swarm-2", target: "auth panic", status: "blocked", at: base.Add(time.Minute)},
				{id: "swarm-3", target: "auth panic", status: "failed", at: base.Add(2 * time.Minute)},
			},
			target:     "auth panic",
			wantCount:  3,
			wantNext:   4,
			wantIDs:    []string{"swarm-1", "swarm-2", "swarm-3"},
			wantStatus: []string{"failed", "blocked", "failed"},
		},
		{
			name: "completed result resets its own target",
			records: []swarmStrikeFixture{
				{id: "swarm-1", target: "auth panic", status: "failed", at: base},
				{id: "swarm-other-success", target: "database panic", status: "completed", at: base.Add(time.Minute)},
				{id: "swarm-2", target: "auth panic", status: "completed", at: base.Add(2 * time.Minute)},
				{id: "swarm-3", target: "auth panic", status: "failed", at: base.Add(3 * time.Minute)},
			},
			target:     "auth panic",
			wantCount:  1,
			wantNext:   2,
			wantIDs:    []string{"swarm-3"},
			wantStatus: []string{"failed"},
		},
		{
			name: "different targets do not share strikes",
			records: []swarmStrikeFixture{
				{id: "swarm-a1", target: "auth panic", status: "failed", at: base},
				{id: "swarm-db1", target: "database panic", status: "failed", at: base.Add(time.Minute)},
				{id: "swarm-a2", target: "auth panic", status: "blocked", at: base.Add(2 * time.Minute)},
				{id: "swarm-db2", target: "database panic", status: "failed", at: base.Add(3 * time.Minute)},
			},
			target:     "auth panic",
			wantCount:  2,
			wantNext:   3,
			wantIDs:    []string{"swarm-a1", "swarm-a2"},
			wantStatus: []string{"failed", "blocked"},
		},
		{
			name: "cancelled and non-terminal records do not invent strikes or resets",
			records: []swarmStrikeFixture{
				{id: "swarm-1", target: "auth panic", status: "failed", at: base},
				{id: "swarm-2", target: "auth panic", status: "cancelled", at: base.Add(time.Minute)},
				{id: "swarm-3", target: "auth panic", status: "running", at: base.Add(2 * time.Minute)},
				{id: "swarm-4", target: "auth panic", status: "blocked", at: base.Add(3 * time.Minute)},
			},
			target:     "auth panic",
			wantCount:  2,
			wantNext:   3,
			wantIDs:    []string{"swarm-1", "swarm-4"},
			wantStatus: []string{"failed", "blocked"},
		},
		{
			name: "legacy result without stored fingerprint derives it from target",
			records: []swarmStrikeFixture{
				{id: "swarm-legacy", target: "auth panic", status: "failed", at: base, legacy: true},
			},
			target:     "auth panic",
			wantCount:  1,
			wantNext:   2,
			wantIDs:    []string{"swarm-legacy"},
			wantStatus: []string{"failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestStore(t)
			for _, record := range tt.records {
				writeSwarmStrikeFixture(t, s, record)
			}

			history, err := evaluateSwarmStrikeHistory(s, tt.target)
			if err != nil {
				t.Fatalf("evaluateSwarmStrikeHistory: %v", err)
			}
			if history.StrikeCount != tt.wantCount {
				t.Fatalf("StrikeCount = %d, want %d: %+v", history.StrikeCount, tt.wantCount, history)
			}
			if history.NextAttempt != tt.wantNext {
				t.Fatalf("NextAttempt = %d, want %d: %+v", history.NextAttempt, tt.wantNext, history)
			}
			if got := swarmStrikeEvidenceIDs(history.Evidence); !reflect.DeepEqual(got, tt.wantIDs) {
				t.Fatalf("evidence IDs = %v, want %v", got, tt.wantIDs)
			}
			if got := swarmStrikeEvidenceStatuses(history.Evidence); !reflect.DeepEqual(got, tt.wantStatus) {
				t.Fatalf("evidence statuses = %v, want %v", got, tt.wantStatus)
			}
		})
	}
}

func TestSwarmStrikeHistoryIgnoresMalformedOrSpoofedRecords(t *testing.T) {
	t.Parallel()

	s, _ := newTestStore(t)
	base := time.Date(2026, time.August, 31, 11, 0, 0, 0, time.UTC)
	writeSwarmStrikeFixture(t, s, swarmStrikeFixture{
		id: "swarm-good", target: "auth panic", status: "failed", at: base,
	})

	writeRawSwarmResult(t, s, "swarm-bad-json", []byte(`{"swarm_id":`))
	writeRawSwarmResult(t, s, "swarm-no-target", mustJSONFixture(t, map[string]interface{}{
		"swarm_id": "swarm-no-target", "status": "failed", "completed_at": base.Add(time.Minute).Format(time.RFC3339),
	}))
	writeRawSwarmResult(t, s, "swarm-bad-time", mustJSONFixture(t, map[string]interface{}{
		"swarm_id": "swarm-bad-time", "target": "auth panic", "status": "failed", "completed_at": "not-a-time",
	}))
	writeRawSwarmResult(t, s, "swarm-spoofed-fingerprint", mustJSONFixture(t, map[string]interface{}{
		"swarm_id":           "swarm-spoofed-fingerprint",
		"target":             "database panic",
		"target_fingerprint": swarmTargetFingerprint("auth panic"),
		"status":             "failed",
		"completed_at":       base.Add(2 * time.Minute).Format(time.RFC3339),
	}))
	writeRawSwarmResult(t, s, "swarm-id-mismatch", mustJSONFixture(t, map[string]interface{}{
		"swarm_id": "some-other-id", "target": "auth panic", "status": "failed", "completed_at": base.Add(3 * time.Minute).Format(time.RFC3339),
	}))

	history, err := evaluateSwarmStrikeHistory(s, "auth panic")
	if err != nil {
		t.Fatalf("evaluateSwarmStrikeHistory: %v", err)
	}
	if history.StrikeCount != 1 || !reflect.DeepEqual(swarmStrikeEvidenceIDs(history.Evidence), []string{"swarm-good"}) {
		t.Fatalf("malformed or spoofed rows affected history: %+v", history)
	}
}

func TestSwarmStrikeHistoryIsIndependentOfDirectoryOrder(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	records := []swarmStrikeFixture{
		{id: "swarm-z", target: "auth panic", status: "failed", at: base},
		{id: "swarm-a", target: "auth panic", status: "blocked", at: base.Add(time.Minute)},
		{id: "swarm-m", target: "auth panic", status: "failed", at: base.Add(2 * time.Minute)},
	}

	evaluate := func(order []int) swarmStrikeHistory {
		s, _ := newTestStore(t)
		for _, index := range order {
			writeSwarmStrikeFixture(t, s, records[index])
		}
		history, err := evaluateSwarmStrikeHistory(s, "auth panic")
		if err != nil {
			t.Fatalf("evaluateSwarmStrikeHistory: %v", err)
		}
		return history
	}

	forward := evaluate([]int{0, 1, 2})
	shuffled := evaluate([]int{2, 0, 1})
	if !reflect.DeepEqual(forward, shuffled) {
		t.Fatalf("history changed with creation order:\nforward: %+v\nshuffled: %+v", forward, shuffled)
	}
	if got := swarmStrikeEvidenceIDs(forward.Evidence); !reflect.DeepEqual(got, []string{"swarm-z", "swarm-a", "swarm-m"}) {
		t.Fatalf("evidence order = %v, want durable timestamp order", got)
	}
}

func TestSwarmStrikeHistoryDoesNotCreateCounterArtifacts(t *testing.T) {
	t.Parallel()

	s, _ := newTestStore(t)
	writeSwarmStrikeFixture(t, s, swarmStrikeFixture{
		id: "swarm-only", target: "auth panic", status: "failed", at: time.Now().UTC(),
	})
	before := regularFilesUnder(t, s.BasePath())

	if _, err := evaluateSwarmStrikeHistory(s, "auth panic"); err != nil {
		t.Fatalf("evaluateSwarmStrikeHistory: %v", err)
	}

	after := regularFilesUnder(t, s.BasePath())
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("history evaluation mutated storage:\nbefore: %v\nafter:  %v", before, after)
	}
	for _, path := range after {
		name := strings.ToLower(filepath.Base(path))
		if strings.Contains(name, "strike") || strings.Contains(name, "counter") {
			t.Fatalf("unexpected mutable strike artifact %q", path)
		}
	}
}

func TestSwarmStrikeDirectExternalParity(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.August, 31, 13, 0, 0, 0, time.UTC)
	type normalizedOutcome struct {
		History swarmStrikeHistory
		Flags   []colony.FlagEntry
	}
	run := func(dispatchMode string) normalizedOutcome {
		s, _ := newTestStore(t)
		for i, status := range []string{"failed", "blocked", "failed"} {
			record := swarmResultRecord{
				SwarmID:      []string{"swarm-1", "swarm-2", "swarm-3"}[i],
				Target:       "auth panic",
				Status:       status,
				CompletedAt:  base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
				DispatchMode: dispatchMode,
			}
			if _, err := persistSwarmResultOutcome(s, record); err != nil {
				t.Fatalf("persist %s result %d: %v", dispatchMode, i+1, err)
			}
		}
		history, err := evaluateSwarmStrikeHistory(s, "auth panic")
		if err != nil {
			t.Fatalf("evaluate %s history: %v", dispatchMode, err)
		}
		return normalizedOutcome{History: history, Flags: activeSwarmEscalationFlags(s)}
	}

	direct := run("")
	external := run("external-task")
	if !reflect.DeepEqual(direct, external) {
		t.Fatalf("direct/external escalation drift:\ndirect:   %+v\nexternal: %+v", direct, external)
	}
}

func TestSwarmThreeStrikeReplayKeepsOneEscalation(t *testing.T) {
	t.Parallel()

	s, _ := newTestStore(t)
	base := time.Date(2026, time.August, 31, 14, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		if _, err := persistSwarmResultOutcome(s, swarmResultRecord{
			SwarmID:     []string{"swarm-1", "swarm-2"}[i],
			Target:      "auth panic",
			Status:      "failed",
			CompletedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("persist prior strike %d: %v", i+1, err)
		}
	}
	third := swarmResultRecord{
		SwarmID:      "swarm-3",
		Target:       "auth panic",
		Status:       "blocked",
		CompletedAt:  base.Add(2 * time.Minute).Format(time.RFC3339),
		DispatchMode: "external-task",
	}
	for replay := 1; replay <= 2; replay++ {
		if _, err := persistSwarmResultOutcome(s, third); err != nil {
			t.Fatalf("persist third strike replay %d: %v", replay, err)
		}
		if flags := activeSwarmEscalationFlags(s); len(flags) != 1 {
			t.Fatalf("replay %d active escalation flags = %d, want 1: %+v", replay, len(flags), flags)
		}
	}
}

type swarmStrikeFixture struct {
	id     string
	target string
	status string
	at     time.Time
	legacy bool
}

func writeSwarmStrikeFixture(t *testing.T, s *storage.Store, fixture swarmStrikeFixture) {
	t.Helper()
	record := swarmResultRecord{
		SwarmID:     fixture.id,
		Target:      fixture.target,
		Status:      fixture.status,
		CompletedAt: fixture.at.UTC().Format(time.RFC3339),
	}
	if err := saveSwarmResultRecord(s, record); err != nil {
		t.Fatalf("save swarm result %s: %v", fixture.id, err)
	}
	if !fixture.legacy {
		return
	}
	path := filepath.Join(s.BasePath(), "swarms", fixture.id, "result.json")
	var raw map[string]interface{}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read legacy fixture: %v", err)
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode legacy fixture: %v", err)
	}
	delete(raw, "target_fingerprint")
	writeRawSwarmResult(t, s, fixture.id, mustJSONFixture(t, raw))
}

func writeRawSwarmResult(t *testing.T, s *storage.Store, id string, data []byte) {
	t.Helper()
	dir := filepath.Join(s.BasePath(), "swarms", id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir raw swarm result: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "result.json"), append(data, '\n'), 0644); err != nil {
		t.Fatalf("write raw swarm result: %v", err)
	}
}

func mustJSONFixture(t *testing.T, value interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON fixture: %v", err)
	}
	return data
}

func swarmStrikeEvidenceIDs(evidence []swarmStrikeEvidence) []string {
	if len(evidence) == 0 {
		return nil
	}
	ids := make([]string, 0, len(evidence))
	for _, item := range evidence {
		ids = append(ids, item.SwarmID)
	}
	return ids
}

func swarmStrikeEvidenceStatuses(evidence []swarmStrikeEvidence) []string {
	if len(evidence) == 0 {
		return nil
	}
	statuses := make([]string, 0, len(evidence))
	for _, item := range evidence {
		statuses = append(statuses, item.Status)
	}
	return statuses
}

func regularFilesUnder(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	sort.Strings(files)
	return files
}

func activeSwarmEscalationFlags(s *storage.Store) []colony.FlagEntry {
	flags, ok := loadFlagsFile(s)
	if !ok {
		return nil
	}
	active := make([]colony.FlagEntry, 0, len(flags.Decisions))
	for _, flag := range flags.Decisions {
		if flag.Type == "blocker" && flag.Source == "escalation" && !flag.Resolved {
			active = append(active, flag)
		}
	}
	sort.Slice(active, func(i, j int) bool { return active[i].ID < active[j].ID })
	return active
}
