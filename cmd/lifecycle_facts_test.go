package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const lifecycleFactsNow = "2026-09-03T12:00:00Z"

func TestLifecycleFactsReadOnly(t *testing.T) {
	assertLifecycleFactsLoaderHasNoWriteCalls(t)

	for _, fixture := range []string{"valid", "missing", "malformed"} {
		t.Run(fixture, func(t *testing.T) {
			root, factStore, watched := seedLifecycleFactsFixture(t, fixture)
			before := fingerprintLifecycleFactSurfaces(t, root, watched)

			now, err := time.Parse(time.RFC3339, lifecycleFactsNow)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := loadLifecycleFacts(root, factStore, now); err != nil {
				t.Fatalf("load lifecycle facts from %s fixture: %v", fixture, err)
			}

			after := fingerprintLifecycleFactSurfaces(t, root, watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("orientation read mutated %s fixture\nbefore: %#v\nafter:  %#v", fixture, before, after)
			}
		})
	}
}

func TestLifecycleFactsProvenance(t *testing.T) {
	root, factStore, _ := seedLifecycleFactsFixture(t, "valid")
	now, _ := time.Parse(time.RFC3339, lifecycleFactsNow)
	facts, err := loadLifecycleFacts(root, factStore, now)
	if err != nil {
		t.Fatalf("load valid lifecycle facts: %v", err)
	}

	if got := facts.Identity.Value.Goal; got != "Ship the classic front door" {
		t.Fatalf("goal = %q, want fixture goal", got)
	}
	if got := facts.Identity.Value.Standing; got != "READY" {
		t.Fatalf("standing = %q, want READY", got)
	}
	if got := facts.Progress.Value.CurrentPhase; got != 1 || len(facts.Progress.Value.Phases) != 1 || len(facts.Progress.Value.Phases[0].Tasks) != 1 {
		t.Fatalf("phase/task facts incomplete: %#v", facts.Progress.Value)
	}
	if len(facts.Actors.Value) != 1 || facts.Actors.Value[0].Caste != "builder" || facts.Actors.Value[0].Parent != "queen" {
		t.Fatalf("actor lineage facts incomplete: %#v", facts.Actors.Value)
	}
	if len(facts.Signals.Value) != 1 || len(facts.Research.Value.Dreams) != 1 || len(facts.Research.Value.Territory) != 1 {
		t.Fatalf("signals/research facts incomplete: signals=%#v research=%#v", facts.Signals.Value, facts.Research.Value)
	}
	if len(facts.Memory.Value.Findings) != 1 || len(facts.Memory.Value.Observations) != 1 || len(facts.Memory.Value.Instincts) != 1 {
		t.Fatalf("memory/findings facts incomplete: %#v", facts.Memory.Value)
	}
	if len(facts.Verification.Value.Gates) != 1 || len(facts.Verification.Value.Artifacts) != 1 {
		t.Fatalf("verification facts incomplete: %#v", facts.Verification.Value)
	}
	if facts.Timing.Value.Elapsed != 2*time.Hour || facts.ReportedCost.Value.TotalTokens != 1500 {
		t.Fatalf("timing/cost facts incomplete: timing=%#v cost=%#v", facts.Timing.Value, facts.ReportedCost.Value)
	}
	if len(facts.History.Value) != 1 || len(facts.Blockers.Value) != 1 || facts.Evidence.Value.Receipt == nil || facts.Evidence.Value.Recovery == nil {
		t.Fatalf("history/blocker/evidence facts incomplete: history=%#v blockers=%#v evidence=%#v", facts.History.Value, facts.Blockers.Value, facts.Evidence.Value)
	}

	for _, source := range facts.Sources() {
		if strings.TrimSpace(source.Domain) == "" || strings.TrimSpace(source.Path) == "" {
			t.Fatalf("fact source lacks domain/path: %#v", source)
		}
		if source.Provenance != LifecycleFactConfirmed {
			t.Fatalf("valid %s provenance = %q (%s), want %q", source.Domain, source.Provenance, source.Diagnostic, LifecycleFactConfirmed)
		}
	}

	t.Run("legacy normalization is in-memory and diagnosed", func(t *testing.T) {
		legacyRoot, legacyStore, watched := seedLifecycleFactsFixture(t, "legacy")
		before := fingerprintLifecycleFactSurfaces(t, legacyRoot, watched)
		legacy, err := loadLifecycleFacts(legacyRoot, legacyStore, now)
		if err != nil {
			t.Fatalf("load legacy lifecycle facts: %v", err)
		}
		if legacy.Progress.Value.CurrentPhase != 2 {
			t.Fatalf("legacy current phase = %d, want normalized 2", legacy.Progress.Value.CurrentPhase)
		}
		if legacy.State.Source.Provenance != LifecycleFactConfirmed || !strings.Contains(strings.ToLower(legacy.State.Source.Diagnostic), "legacy") {
			t.Fatalf("legacy provenance/diagnostic = %#v", legacy.State.Source)
		}
		after := fingerprintLifecycleFactSurfaces(t, legacyRoot, watched)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("legacy compatibility read persisted its repair\nbefore: %#v\nafter:  %#v", before, after)
		}
	})
}

func TestLifecycleFactsMissingAndMalformed(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, lifecycleFactsNow)
	for _, tc := range []struct {
		fixture string
		want    LifecycleFactProvenance
	}{
		{fixture: "missing", want: LifecycleFactMissing},
		{fixture: "malformed", want: LifecycleFactMalformed},
	} {
		t.Run(tc.fixture, func(t *testing.T) {
			root, factStore, _ := seedLifecycleFactsFixture(t, tc.fixture)
			facts, err := loadLifecycleFacts(root, factStore, now)
			if err != nil {
				t.Fatalf("load %s lifecycle facts: %v", tc.fixture, err)
			}
			for _, source := range facts.Sources() {
				if source.Provenance != tc.want {
					t.Fatalf("%s provenance = %q, want %q (%s)", source.Domain, source.Provenance, tc.want, source.Diagnostic)
				}
				if strings.TrimSpace(source.Diagnostic) == "" {
					t.Fatalf("%s %s source has no diagnostic", tc.fixture, source.Domain)
				}
			}
		})
	}

	t.Run("unavailable store", func(t *testing.T) {
		root := t.TempDir()
		facts, err := loadLifecycleFacts(root, nil, now)
		if err != nil {
			t.Fatalf("nil store should be represented as unavailable facts, got %v", err)
		}
		for _, source := range facts.Sources() {
			if source.Provenance != LifecycleFactUnavailable || source.Diagnostic == "" {
				t.Fatalf("unavailable %s source = %#v", source.Domain, source)
			}
		}
	})
}

func TestLifecycleFactsSpecAndPlanAuthorityStates(t *testing.T) {
	accepted, _ := validCurrentPlanningState(t)

	draft := accepted
	draft.Plan = colony.Plan{}
	draft.CurrentPhase = 0
	draft.Specification = cloneLifecycleTestSpecification(accepted.Specification)
	draft.Specification.Revisions[0].Status = colony.SpecStatusDraft
	draft.Specification.Revisions[0].Approval = nil

	approved := draft
	approved.Specification = cloneLifecycleTestSpecification(accepted.Specification)

	candidateReady := accepted
	pending := candidateReady.Plan.Candidates[0]
	pendingHash := planningStateTestDigest("lifecycle-pending-candidate")
	pending.ID = planningStateTestAddress("plan-candidate", pendingHash)
	pending.ContentHash = pendingHash
	pending.Status = colony.PlanCandidatePendingReview
	pending.Acceptance = nil
	pending.Recommendation.CandidateID = pending.ID
	candidateReady.Plan.Candidates = append(append([]colony.PlanCandidate(nil), candidateReady.Plan.Candidates...), pending)
	candidateReady.Plan.PendingCandidateID = pending.ID

	affected := accepted
	affected.Plan.Revisions = append([]colony.PlanRevision(nil), accepted.Plan.Revisions...)
	affected.Plan.Revisions[len(affected.Plan.Revisions)-1].AffectedSemanticIDs = []string{"task:affected"}

	legacyTaskID := "legacy-task"
	legacy := colony.ColonyState{
		Goal: fixtureGoal("Keep the old plan buildable"), State: colony.StateREADY, CurrentPhase: 1,
		Plan: colony.Plan{
			AcceptancePolicy: colony.PlanAcceptanceLegacyUnbound,
			Phases:           []colony.Phase{{ID: 1, Name: "Legacy", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &legacyTaskID, Goal: "Build it", Status: colony.TaskPending}}}},
		},
	}

	tests := []struct {
		name          string
		state         colony.ColonyState
		wantSpec      colony.SpecRevisionStatus
		wantApproved  bool
		wantCandidate colony.PlanCandidateStatus
		wantStop      colony.PlanningStopReason
		wantBinding   LifecyclePlanAcceptanceBindingStatus
		wantAccepted  bool
		wantLegacy    bool
		wantAffected  []string
	}{
		{name: "draft", state: draft, wantSpec: colony.SpecStatusDraft, wantBinding: LifecyclePlanBindingAbsent},
		{name: "approved", state: approved, wantSpec: colony.SpecStatusApproved, wantApproved: true, wantBinding: LifecyclePlanBindingAbsent},
		{name: "candidate ready", state: candidateReady, wantSpec: colony.SpecStatusApproved, wantApproved: true, wantCandidate: colony.PlanCandidatePendingReview, wantStop: pending.StopDecision.Reason, wantBinding: LifecyclePlanBindingAccepted, wantAccepted: true},
		{name: "accepted", state: accepted, wantSpec: colony.SpecStatusApproved, wantApproved: true, wantBinding: LifecyclePlanBindingAccepted, wantAccepted: true},
		{name: "affected revision", state: affected, wantSpec: colony.SpecStatusApproved, wantApproved: true, wantBinding: LifecyclePlanBindingAffected, wantAccepted: true, wantAffected: []string{"task:affected"}},
		{name: "legacy", state: legacy, wantBinding: LifecyclePlanBindingLegacyUnbound, wantLegacy: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			facts := lifecycleFactsFromStateSnapshot(tc.state, false, time.Time{})
			specification := facts.Specification.Value
			planning := facts.Planning.Value
			if specification.Status != tc.wantSpec || specification.Approved != tc.wantApproved {
				t.Fatalf("specification facts = %+v, want status=%q approved=%t", specification, tc.wantSpec, tc.wantApproved)
			}
			if planning.PendingCandidateStatus != tc.wantCandidate || planning.PendingCandidateStopReason != tc.wantStop {
				t.Fatalf("candidate facts = %+v, want status=%q stop=%q", planning, tc.wantCandidate, tc.wantStop)
			}
			if planning.AcceptanceBindingStatus != tc.wantBinding || planning.AcceptedPlan != tc.wantAccepted || planning.LegacyUnbound != tc.wantLegacy {
				t.Fatalf("acceptance facts = %+v, want binding=%q accepted=%t legacy=%t", planning, tc.wantBinding, tc.wantAccepted, tc.wantLegacy)
			}
			if !reflect.DeepEqual(planning.AffectedUnresolvedSemanticIDs, tc.wantAffected) {
				t.Fatalf("affected IDs = %v, want %v", planning.AffectedUnresolvedSemanticIDs, tc.wantAffected)
			}
		})
	}
}

func TestLifecycleFactsSpecAndPlanUseOneStateSnapshot(t *testing.T) {
	root, factStore, _ := seedLifecycleFactsFixture(t, "valid")
	state, _ := validCurrentPlanningState(t)
	state.Plan.Revisions[len(state.Plan.Revisions)-1].PlanningRunID = "planning-run-200"
	stateBytes, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleFixtureFile(t, filepath.Join(factStore.BasePath(), "COLONY_STATE.json"), string(stateBytes))

	pending := PendingDecisionFile{Decisions: []PendingDecision{{
		ID: "decision-1", Type: clarificationDecisionType, Description: "Choose behavior", Source: "discuss:behavior:test",
		SessionID: "session-200", GoalHash: pendingDecisionGoalHash(lifecycleString(state.Goal)), CreatedAt: lifecycleFactsNow,
	}}}
	pendingBytes, err := json.Marshal(pending)
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleFixtureFile(t, filepath.Join(factStore.BasePath(), pendingDecisionsFile), string(pendingBytes))

	stage := planningStageTestState(planningStageOwnerDecision)
	stage.RunID = "planning-run-200"
	stage.Pass = 2
	stage.Preset = planningStagePresetDeep
	currentSpec, _ := currentSpecificationRevision(*state.Specification)
	stage.Specification = planningStageSpecificationBinding{
		RevisionID: currentSpec.ID, ContentHash: currentSpec.ContentHash, Status: currentSpec.Status,
		ApprovalReceiptID: currentSpec.Approval.ID, ApprovalReceiptHash: planningStateTestDigest("approval-receipt"),
	}
	active := state.Plan.Revisions[len(state.Plan.Revisions)-1]
	stage.BasePlanRevisionID = active.ID
	stage.BasePlanRevisionHash = active.PlanHash
	stage.PendingAffectedSemanticIDs = []string{"requirement:owner-choice"}
	stageBytes, err := json.Marshal(stage)
	if err != nil {
		t.Fatal(err)
	}
	writeLifecycleFixtureFile(t, filepath.Join(root, filepath.FromSlash(planningStageStateRepositoryPath(stage.RunID))), string(stageBytes))

	loads := 0
	now, _ := time.Parse(time.RFC3339, lifecycleFactsNow)
	facts, err := loadLifecycleFactsWithStateReader(root, factStore, now, func(path string) (colony.ColonyState, LifecycleFactSource) {
		loads++
		return readLifecycleState(path)
	})
	if err != nil {
		t.Fatalf("load authority-aware facts: %v", err)
	}
	if loads != 1 {
		t.Fatalf("state loads = %d, want exactly 1", loads)
	}
	if facts.Intent.Value.UnresolvedDiscussionCount != 1 {
		t.Fatalf("unresolved discussion count = %d, want 1", facts.Intent.Value.UnresolvedDiscussionCount)
	}
	if facts.Planning.Value.RunID != stage.RunID || facts.Planning.Value.Stage != string(stage.Stage) || facts.Planning.Value.Preset != string(stage.Preset) || facts.Planning.Value.Pass != stage.Pass {
		t.Fatalf("planning stage facts = %+v, want run/stage/preset/pass from validated stage state", facts.Planning.Value)
	}

	terminal := projectLifecycle(facts, LifecycleViewVisual, "codex")
	machine := projectLifecycle(facts, LifecycleViewJSON, "codex")
	if loads != 1 {
		t.Fatalf("projection performed another state load: %d total", loads)
	}
	if !reflect.DeepEqual(terminal.Intent, machine.Intent) || !reflect.DeepEqual(terminal.Specification, machine.Specification) || !reflect.DeepEqual(terminal.Planning, machine.Planning) {
		t.Fatalf("terminal and JSON projections diverged\nterminal=%+v\nmachine=%+v", terminal, machine)
	}
}

func cloneLifecycleTestSpecification(specification *colony.Specification) *colony.Specification {
	if specification == nil {
		return nil
	}
	clone := *specification
	clone.Revisions = append([]colony.SpecRevision(nil), specification.Revisions...)
	return &clone
}

type lifecycleSurfaceFingerprint struct {
	Trees map[string][]string
	Files map[string]string
	Refs  string
}

func fingerprintLifecycleFactSurfaces(t *testing.T, root string, watched []string) lifecycleSurfaceFingerprint {
	t.Helper()
	result := lifecycleSurfaceFingerprint{Trees: map[string][]string{}, Files: map[string]string{}}
	for _, path := range watched {
		info, err := os.Stat(path)
		if err != nil {
			result.Files[path] = "missing:" + err.Error()
			continue
		}
		if info.IsDir() {
			var entries []string
			err := filepath.WalkDir(path, func(current string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				stat, err := entry.Info()
				if err != nil {
					return err
				}
				rel, _ := filepath.Rel(path, current)
				line := fmt.Sprintf("%s|%s|%d|%d", filepath.ToSlash(rel), stat.Mode(), stat.Size(), stat.ModTime().UnixNano())
				if stat.Mode().IsRegular() {
					data, err := os.ReadFile(current)
					if err != nil {
						return err
					}
					digest := sha256.Sum256(data)
					line += "|" + hex.EncodeToString(digest[:])
				}
				entries = append(entries, line)
				return nil
			})
			if err != nil {
				t.Fatalf("fingerprint %s: %v", path, err)
			}
			sort.Strings(entries)
			result.Trees[path] = entries
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("fingerprint file %s: %v", path, err)
		}
		digest := sha256.Sum256(data)
		result.Files[path] = fmt.Sprintf("%s|%d|%d|%s", info.Mode(), info.Size(), info.ModTime().UnixNano(), hex.EncodeToString(digest[:]))
	}
	refs, err := exec.Command("git", "-C", root, "show-ref").CombinedOutput()
	if err != nil && len(refs) == 0 {
		t.Fatalf("read git refs: %v", err)
	}
	result.Refs = string(refs)
	return result
}

func seedLifecycleFactsFixture(t *testing.T, fixture string) (string, *storage.Store, []string) {
	t.Helper()
	root := t.TempDir()
	runFixtureGit(t, root, "init", "-q")
	runFixtureGit(t, root, "config", "user.email", "facts@example.test")
	runFixtureGit(t, root, "config", "user.name", "Lifecycle Facts")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("facts fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, root, "add", "README.md")
	runFixtureGit(t, root, "commit", "-q", "-m", "fixture")

	dataDir := filepath.Join(root, ".aether", "data")
	factStore, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create fixture store: %v", err)
	}
	hub := filepath.Join(root, "fake-hub")
	if err := os.MkdirAll(hub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AETHER_HOME", hub)
	registry := filepath.Join(hub, "registry.json")
	if err := os.WriteFile(registry, []byte(`{"colonies":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(dataDir, "COLONY_STATE.json")
	sessionPath := filepath.Join(dataDir, "session.json")
	if fixture == "valid" || fixture == "legacy" {
		phase := "1"
		if fixture == "legacy" {
			phase = `"2"`
		}
		writeLifecycleFixtureFile(t, statePath, fmt.Sprintf(`{
  "version":"1","goal":"Ship the classic front door","colony_name":"Atlas","state":"READY","current_phase":%s,
  "milestone":"v2","initialized_at":"2026-09-03T08:00:00Z","build_started_at":"2026-09-03T10:00:00Z",
  "plan":{"generated_at":"2026-09-03T09:00:00Z","phases":[{"id":1,"name":"Front door","status":"ready","tasks":[{"id":"1-1","goal":"Load facts","status":"pending"}]}]},
  "memory":{"phase_learnings":[{"id":"learning-1","phase":1,"phase_name":"Front door","learnings":[],"timestamp":"2026-09-03T10:00:00Z"}],"decisions":[],"instincts":[]},
  "events":["2026-09-03T10:00:00Z|plan|accepted"],
  "gate_results":[{"name":"tests","passed":true,"timestamp":"2026-09-03T11:00:00Z"}],
  "lifecycle_receipt":{"schema_version":"lifecycle/v1","receipt_id":"receipt-1","command":"plan","outcome_kind":"completed","state_effect":"committed","transaction":{"id":"tx-1","stage":"committed"},"provenance":"confirmed"},
  "recovery_provenance":"confirmed"
}`, phase))
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "pheromones.json"), `{"signals":[{"id":"sig-1","type":"FOCUS","active":true,"content":{"text":"read only"}}]}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "spawn-tree.txt"), "2026-09-03T10:00:00Z|queen|builder|Mason-1|load facts|1|completed\n")
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "pending-decisions.json"), `{"version":"1","decisions":[{"id":"flag-1","type":"blocker","description":"owner input","source":"test","created_at":"2026-09-03T10:00:00Z","resolved":false}]}`)
		writeLifecycleFixtureFile(t, sessionPath, `{"session_id":"session-1","started_at":"2026-09-03T08:00:00Z","last_command":"aether plan","current_phase":1,"lifecycle_receipt":{"schema_version":"lifecycle/v1","receipt_id":"receipt-1","command":"plan","outcome_kind":"completed","state_effect":"committed","transaction":{"id":"tx-1","stage":"committed"},"provenance":"confirmed"},"recovery_provenance":"confirmed"}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "instincts.json"), `{"version":"1","instincts":[{"id":"instinct-1","trigger":"orientation","action":"read facts","domain":"workflow","trust_score":0.9,"trust_tier":"trusted","confidence":0.9,"provenance":{},"application_history":[],"related_instincts":[]}]}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "learning-observations.json"), `{"observations":[{"id":"observation-1","content":"reads stay read-only"}]}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "reviews", "security", "ledger.json"), `{"entries":[{"id":"finding-1","phase":1,"agent":"gatekeeper","generated_at":"2026-09-03T11:00:00Z","status":"open","severity":"HIGH","description":"No writes"}]}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "build", "phase-1", "verification.json"), `{"passed":true}`)
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "spend", "phase-1-build.json"), `{"schema_version":1,"phase":1,"workflow":"build","recorded_at":"2026-09-03T11:00:00Z","rows":[{"name":"Mason-1","caste":"builder","task":"load facts","status":"completed","usage":{"input_tokens":1000,"output_tokens":500,"total_tokens":1500}}]}`)
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "dreams", "orientation.md"), "# Orientation\n")
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "research", "front-door.md"), "# Front door research\n")
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "survey", "territory.json"), `{"language":"go"}`)
	} else if fixture == "malformed" {
		for _, path := range []string{
			statePath,
			filepath.Join(dataDir, "pheromones.json"),
			filepath.Join(dataDir, "spawn-tree.txt"),
			filepath.Join(dataDir, "pending-decisions.json"),
			sessionPath,
			filepath.Join(dataDir, "instincts.json"),
			filepath.Join(dataDir, "learning-observations.json"),
			filepath.Join(dataDir, "reviews", "security", "ledger.json"),
			filepath.Join(dataDir, "build", "phase-1", "verification.json"),
			filepath.Join(dataDir, "spend", "phase-1-build.json"),
		} {
			writeLifecycleFixtureFile(t, path, "{not-valid")
		}
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "dreams"), "not a directory")
		writeLifecycleFixtureFile(t, filepath.Join(root, ".aether", "research"), "not a directory")
		writeLifecycleFixtureFile(t, filepath.Join(dataDir, "survey"), "not a directory")
	}

	return root, factStore, []string{filepath.Join(root, ".aether"), hub, sessionPath, registry}
}

func writeLifecycleFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runFixtureGit(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func assertLifecycleFactsLoaderHasNoWriteCalls(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("lifecycle_facts.go")
	if err != nil {
		t.Fatalf("read lifecycle_facts.go: %v", err)
	}
	source := string(data)
	start := strings.Index(source, "func loadLifecycleFacts(")
	if start < 0 {
		t.Fatal("loadLifecycleFacts declaration not found")
	}
	loader := source[start:]
	for _, forbidden := range []string{
		"SaveJSON(", "AtomicWrite(", "MkdirAll(", "WriteFile(", "Remove(",
		"ensureLegacySessionMirror(", "registryUpsert", "exec.Command(",
		"loadActiveColonyState(", "loadColonyStateWithCompatibilityRepair()",
	} {
		if strings.Contains(loader, forbidden) {
			t.Fatalf("loadLifecycleFacts reaches write-capable helper %q", forbidden)
		}
	}
}
