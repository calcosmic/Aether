package cmd

import (
	"encoding/json"
	"fmt"
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
	if swarmTargetFingerprint("C++ crash") == swarmTargetFingerprint("C crash") {
		t.Fatal("material language punctuation was stripped from C++ target")
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

func TestSwarmRecoveryEpoch(t *testing.T) {
	t.Parallel()

	s, _ := newTestStore(t)
	target := "auth panic"
	base := time.Date(2026, time.September, 1, 8, 0, 0, 0, time.UTC)
	history := seedSwarmRecoveryStrikes(t, s, target, base)
	before, after, inserted := swarmRecoveryPlans()
	saveSwarmRecoveryPlan(t, s, before)

	record, err := buildSwarmRecoveryRecord(
		target,
		history,
		colony.FlagEntry{
			ID:        swarmEscalationFlagID(history.TargetFingerprint),
			Type:      "blocker",
			Source:    "escalation",
			Resolved:  false,
			CreatedAt: history.Evidence[len(history.Evidence)-1].CompletedAt,
		},
		before,
		after,
		inserted,
		base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("buildSwarmRecoveryRecord: %v", err)
	}
	if err := saveSwarmResultRecord(s, record); err != nil {
		t.Fatalf("save recovery result: %v", err)
	}

	uncommitted, err := evaluateSwarmStrikeHistory(s, "  AUTH PANIC!!! ")
	if err != nil {
		t.Fatalf("evaluate uncommitted recovery: %v", err)
	}
	if uncommitted.StrikeCount != 3 || uncommitted.LatestRecovery != nil {
		t.Fatalf("recovery staged before state commit became active: %+v", uncommitted)
	}

	saveSwarmRecoveryPlan(t, s, after)
	recovered, err := evaluateSwarmStrikeHistory(s, "  AUTH PANIC!!! ")
	if err != nil {
		t.Fatalf("evaluate committed recovery: %v", err)
	}
	if recovered.StrikeCount != 0 || recovered.NextAttempt != 1 {
		t.Fatalf("committed recovery did not open a new epoch: %+v", recovered)
	}
	if recovered.LatestRecovery == nil {
		t.Fatal("verified recovery evidence was not returned")
	}
	if recovered.LatestRecovery.SwarmID != record.SwarmID ||
		recovered.LatestRecovery.InsertedPhaseID != inserted.ID ||
		recovered.LatestRecovery.TargetFingerprint != history.TargetFingerprint {
		t.Fatalf("latest recovery = %+v, want event/phase/target identity", recovered.LatestRecovery)
	}

	if err := saveSwarmResultRecord(s, swarmResultRecord{
		SwarmID:     "swarm-after-recovery-1",
		Target:      target,
		Status:      "failed",
		CompletedAt: base.Add(4 * time.Minute).Format(time.RFC3339Nano),
	}); err != nil {
		t.Fatalf("save post-recovery failure: %v", err)
	}
	nextEpoch, err := evaluateSwarmStrikeHistory(s, target)
	if err != nil {
		t.Fatalf("evaluate post-recovery failure: %v", err)
	}
	if nextEpoch.StrikeCount != 1 || nextEpoch.NextAttempt != 2 ||
		!reflect.DeepEqual(swarmStrikeEvidenceIDs(nextEpoch.Evidence), []string{"swarm-after-recovery-1"}) {
		t.Fatalf("post-recovery failures did not count from zero: %+v", nextEpoch)
	}

	data, err := os.ReadFile(filepath.Join(s.BasePath(), "swarms", record.SwarmID, "result.json"))
	if err != nil {
		t.Fatalf("read recovery result: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode recovery result: %v", err)
	}
	for _, forbidden := range []string{"workers", "files", "tests", "blockers", "root_cause", "solution", "recommendation"} {
		if _, exists := raw[forbidden]; exists {
			t.Errorf("recovery event contains swarm outcome field %q: %s", forbidden, data)
		}
	}
	if raw["status"] != "recovered" {
		t.Fatalf("recovery status = %v, want recovered", raw["status"])
	}
}

func TestSwarmRecoveryEpochRejectsMalformedSpoofedAndCrossTargetRecords(t *testing.T) {
	t.Parallel()

	target := "auth panic"
	base := time.Date(2026, time.September, 1, 9, 0, 0, 0, time.UTC)
	before, after, inserted := swarmRecoveryPlans()

	tests := []struct {
		name        string
		mutate      func(map[string]interface{})
		livePlan    colony.Plan
		rawJSON     []byte
		queryTarget string
	}{
		{
			name:    "malformed json",
			rawJSON: []byte("{\"swarm_id\":"),
		},
		{
			name: "wrong authorization source",
			mutate: func(raw map[string]interface{}) {
				raw["recovery"].(map[string]interface{})["authorization_source"] = "manual-reset"
			},
		},
		{
			name: "mismatched metadata target fingerprint",
			mutate: func(raw map[string]interface{}) {
				raw["recovery"].(map[string]interface{})["target_fingerprint"] = swarmTargetFingerprint("database panic")
			},
		},
		{
			name: "cross-target row",
			mutate: func(raw map[string]interface{}) {
				raw["target"] = "database panic"
				raw["target_fingerprint"] = swarmTargetFingerprint("database panic")
			},
		},
		{
			name: "identical before and after definitions",
			mutate: func(raw map[string]interface{}) {
				recovery := raw["recovery"].(map[string]interface{})
				recovery["before_plan_definition_hash"] = recovery["after_plan_definition_hash"]
			},
		},
		{
			name: "missing corrective phase fingerprint",
			mutate: func(raw map[string]interface{}) {
				delete(raw["recovery"].(map[string]interface{}), "inserted_phase_definition_fingerprint")
			},
		},
		{
			name:     "absent corrective phase",
			livePlan: before,
		},
		{
			name: "mismatched committed plan definition",
			mutate: func(raw map[string]interface{}) {
				raw["recovery"].(map[string]interface{})["after_plan_definition_hash"] = strings.Repeat("a", 64)
			},
		},
		{
			name: "recovery carries fabricated worker evidence",
			mutate: func(raw map[string]interface{}) {
				raw["workers"] = []interface{}{map[string]interface{}{"name": "fake"}}
			},
		},
		{
			name:        "different queried target remains isolated",
			queryTarget: "database panic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, _ := newTestStore(t)
			history := seedSwarmRecoveryStrikes(t, s, target, base)
			livePlan := tt.livePlan
			if len(livePlan.Phases) == 0 {
				livePlan = after
			}
			saveSwarmRecoveryPlan(t, s, livePlan)

			record, err := buildSwarmRecoveryRecord(
				target,
				history,
				colony.FlagEntry{
					ID:        swarmEscalationFlagID(history.TargetFingerprint),
					Type:      "blocker",
					Source:    "escalation",
					CreatedAt: history.Evidence[len(history.Evidence)-1].CompletedAt,
				},
				before,
				after,
				inserted,
				base.Add(3*time.Minute),
			)
			if err != nil {
				t.Fatalf("build valid recovery fixture: %v", err)
			}

			data := tt.rawJSON
			if data == nil {
				var raw map[string]interface{}
				encoded, err := json.Marshal(record)
				if err != nil {
					t.Fatalf("marshal valid recovery fixture: %v", err)
				}
				if err := json.Unmarshal(encoded, &raw); err != nil {
					t.Fatalf("decode valid recovery fixture: %v", err)
				}
				if tt.mutate != nil {
					tt.mutate(raw)
				}
				data = mustJSONFixture(t, raw)
			}
			writeRawSwarmResult(t, s, record.SwarmID, data)

			queryTarget := tt.queryTarget
			if queryTarget == "" {
				queryTarget = target
			}
			got, err := evaluateSwarmStrikeHistory(s, queryTarget)
			if err != nil {
				t.Fatalf("evaluate history: %v", err)
			}
			if queryTarget == target {
				if got.StrikeCount != 3 || got.LatestRecovery != nil {
					t.Fatalf("invalid recovery changed original epoch: %+v", got)
				}
			} else if got.StrikeCount != 0 || got.LatestRecovery != nil {
				t.Fatalf("cross-target recovery affected %q: %+v", queryTarget, got)
			}
		})
	}
}

func TestSwarmRecoveryEpochRequiresCompleteTypedMetadata(t *testing.T) {
	t.Parallel()

	s, _ := newTestStore(t)
	target := "auth panic"
	base := time.Date(2026, time.September, 1, 10, 0, 0, 0, time.UTC)
	history := seedSwarmRecoveryStrikes(t, s, target, base)
	before, after, inserted := swarmRecoveryPlans()
	record, err := buildSwarmRecoveryRecord(
		target,
		history,
		colony.FlagEntry{
			ID:        swarmEscalationFlagID(history.TargetFingerprint),
			Type:      "blocker",
			Source:    "escalation",
			CreatedAt: history.Evidence[len(history.Evidence)-1].CompletedAt,
		},
		before,
		after,
		inserted,
		base.Add(3*time.Minute),
	)
	if err != nil {
		t.Fatalf("build valid recovery fixture: %v", err)
	}

	missing := record
	missing.Recovery = nil
	if err := saveSwarmResultRecord(s, missing); err == nil {
		t.Fatal("recovered status without metadata was accepted")
	}

	withOutcome := record
	withOutcome.SwarmID = "swarm-recovery-with-outcome"
	withOutcome.Workers = []swarmWorkerExecution{{Name: "fabricated"}}
	if err := saveSwarmResultRecord(s, withOutcome); err == nil {
		t.Fatal("recovery event carrying worker evidence was accepted")
	}

	wrongStatus := record
	wrongStatus.SwarmID = "swarm-recovery-wrong-status"
	wrongStatus.Status = "failed"
	if err := saveSwarmResultRecord(s, wrongStatus); err == nil {
		t.Fatal("non-recovery status carrying recovery metadata was accepted")
	}

	if err := saveSwarmResultRecord(s, record); err != nil {
		t.Fatalf("complete typed recovery was rejected: %v", err)
	}
}

func seedSwarmRecoveryStrikes(t *testing.T, s *storage.Store, target string, base time.Time) swarmStrikeHistory {
	t.Helper()
	for i, status := range []string{"failed", "blocked", "failed"} {
		if _, err := persistSwarmResultOutcome(s, swarmResultRecord{
			SwarmID:     fmt.Sprintf("swarm-recovery-seed-%d", i+1),
			Target:      target,
			Status:      status,
			CompletedAt: base.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano),
		}); err != nil {
			t.Fatalf("persist recovery seed %d: %v", i+1, err)
		}
	}
	history, err := evaluateSwarmStrikeHistory(s, target)
	if err != nil {
		t.Fatalf("evaluate recovery seed: %v", err)
	}
	return history
}

func swarmRecoveryPlans() (colony.Plan, colony.Plan, colony.Phase) {
	beforePhase := colony.Phase{
		ID:          1,
		Name:        "Build authentication",
		Description: "Implement the authentication flow",
		Status:      colony.PhaseReady,
		Tasks: []colony.Task{{
			Goal:   "Implement authentication",
			Status: colony.TaskPending,
		}},
		SuccessCriteria: []string{"Authentication works"},
	}
	inserted := colony.Phase{
		ID:          2,
		Name:        "Stabilize auth panic",
		Description: "Correct the repeated swarm failure",
		Status:      colony.PhasePending,
		Tasks: []colony.Task{{
			Goal:   "Diagnose the repeated failure",
			Status: colony.TaskPending,
		}},
		SuccessCriteria: []string{"The original target can be retried"},
	}
	before := colony.Plan{Phases: []colony.Phase{beforePhase}}
	after := colony.Plan{Phases: []colony.Phase{beforePhase, inserted}}
	return before, after, inserted
}

func saveSwarmRecoveryPlan(t *testing.T, s *storage.Store, plan colony.Plan) {
	t.Helper()
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{Plan: plan}); err != nil {
		t.Fatalf("save recovery plan state: %v", err)
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
