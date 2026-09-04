package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

type sealTransaction199Fixture struct {
	Root      string
	DataRoot  string
	HubRoot   string
	Store     *storage.Store
	Input     SealTransactionInput
	Baselines map[string][]byte
}

func newSealTransaction199Fixture(t *testing.T, forced bool) sealTransaction199Fixture {
	t.Helper()
	root := t.TempDir()
	dataRoot := filepath.Join(root, ".aether", "data")
	hubRoot := filepath.Join(root, "hub")
	for _, dir := range []string{dataRoot, hubRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s, err := storage.NewStore(dataRoot)
	if err != nil {
		t.Fatal(err)
	}
	facts := sealOutcome199Facts()
	facts.Root = root
	facts.State.Source.Path = filepath.Join(dataRoot, "COLONY_STATE.json")
	facts.Progress.Source.Path = facts.State.Source.Path
	facts.Memory.Source.Path = filepath.Join(dataRoot, "instincts.json")
	facts.Signals.Source.Path = filepath.Join(dataRoot, "pheromones.json")
	facts.Blockers.Source.Path = filepath.Join(dataRoot, "pending-decisions.json")
	facts.Evidence.Source.Path = facts.State.Source.Path
	facts.Session.Source.Path = filepath.Join(dataRoot, "session.json")
	facts.Verification.Source.Path = filepath.Join(dataRoot, "seal", "final-review.json")
	facts.Memory.Value.Instincts = []colony.InstinctEntry{
		{ID: "public-pattern", Action: "Keep .aether/data/seal/receipt.json linked to its lifecycle transaction so replay can verify the closure", Domain: "lifecycle", Confidence: 0.95, Provenance: colony.InstinctProvenance{Source: "phase-199", Evidence: "gate:tests"}},
		{ID: "secret-pattern", Action: "password=never-export-this", Domain: "security", Confidence: 0.99, Provenance: colony.InstinctProvenance{Source: "private-review", Evidence: "finding:secret"}},
	}
	facts.State.Value.Memory.PhaseLearnings = []colony.PhaseLearning{{
		ID: "learning-199", Phase: 2, PhaseName: "Closure", Timestamp: "2026-09-04T10:02:00Z",
		Learnings: []colony.Learning{{Claim: "Retain active state after closure", Status: "validated", Tested: true, Evidence: "test:retained-state"}},
	}}
	facts.State.Value.Memory.Decisions = []colony.Decision{{ID: "decision-199", Phase: 2, Claim: "Status remains primary", Rationale: "Sealed state stays reviewable", Timestamp: "2026-09-04T10:02:00Z"}}
	strength := 1.0
	version := "2.0"
	facts.Signals.Value = []colony.PheromoneSignal{
		{ID: "focus-1", Type: "FOCUS", Active: true, Strength: &strength, Content: json.RawMessage(`{"text":"finish the current phase"}`)},
		{ID: "redirect-1", Type: "REDIRECT", Active: true, Strength: &strength, Content: json.RawMessage(`{"text":"do not erase retained state"}`)},
	}
	state := facts.State.Value
	session := colony.SessionFile{SessionID: "session-199", LastCommand: "continue", CurrentPhase: 2, CurrentMilestone: "Working", SuggestedNext: "aether seal", Summary: "Ready for closure"}
	pheromones := colony.PheromoneFile{Version: &version, Signals: facts.Signals.Value}
	for path, value := range map[string]any{
		"COLONY_STATE.json":      state,
		"session.json":           session,
		"pheromones.json":        pheromones,
		"pending-decisions.json": colony.FlagsFile{Version: "1", Decisions: facts.Blockers.Value},
		"instincts.json":         colony.InstinctsFile{Version: "1.0", Instincts: facts.Memory.Value.Instincts},
		"seal/final-review.json": sealFinalReviewReport{Phase: 2, PhaseName: "Closure", GeneratedAt: "2026-09-04T10:02:00Z", Source: "test", Passed: true},
	} {
		if err := s.SaveJSON(path, value); err != nil {
			t.Fatalf("seed %s: %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".aether", "QUEEN.md"), []byte("# Queen\n\n## Wisdom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(hubRoot, "registry"), 0o755); err != nil {
		t.Fatal(err)
	}
	registry := registryData{Colonies: []registryEntry{{RepoPath: root, Active: true, RegisteredAt: "2026-09-01T00:00:00Z"}}}
	registryBytes, _ := json.MarshalIndent(registry, "", "  ")
	if err := os.WriteFile(filepath.Join(hubRoot, "registry", "registry.json"), append(registryBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	request := SealPreflightRequest{Caller: SealCallerDirectOwner}
	if forced {
		facts.State.Value.Plan.Phases[1].Status = colony.PhaseInProgress
		facts.State.Value.Plan.Phases[1].Tasks[0].Status = colony.TaskPending
		facts.Progress.Value.Phases = facts.State.Value.Plan.Phases
		request.Force = true
		request.Reason = "owner records unfinished work for later"
	}
	preflight, err := BuildSealPreflight(facts, request)
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	input := SealTransactionInput{
		Root:                       root,
		DataRoot:                   dataRoot,
		HubRoot:                    hubRoot,
		Facts:                      facts,
		State:                      facts.State.Value,
		Session:                    session,
		Preflight:                  preflight,
		FinalReview:                sealFinalReviewReport{Phase: 2, PhaseName: "Closure", GeneratedAt: "2026-09-04T10:02:00Z", Source: "test", Passed: !forced},
		Now:                        time.Date(2026, 9, 4, 10, 3, 0, 0, time.UTC),
		DisablePostCommitPromotion: true,
	}
	return sealTransaction199Fixture{Root: root, DataRoot: dataRoot, HubRoot: hubRoot, Store: s, Input: input, Baselines: sealTransaction199Snapshot(t, root, hubRoot)}
}

func sealTransaction199Snapshot(t *testing.T, roots ...string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || strings.Contains(path, string(filepath.Separator)+"transactions"+string(filepath.Separator)) || strings.Contains(path, string(filepath.Separator)+lifecycleTransactionDirectory+string(filepath.Separator)) || strings.Contains(path, string(filepath.Separator)+"locks"+string(filepath.Separator)) || strings.HasPrefix(info.Name(), ".cache_") {
				return nil
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[path] = content
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func TestSealTransaction199Atomic(t *testing.T) {
	for target := 1; target <= 15; target++ {
		t.Run(fmt.Sprintf("target-%02d", target), func(t *testing.T) {
			fixture := newSealTransaction199Fixture(t, false)
			point := fmt.Sprintf("after_target_commit:target-%04d", target)
			fixture.Input.Fault = func(got string) error {
				if got == point {
					return errors.New("simulated crash")
				}
				return nil
			}
			_, err := CommitSealTransaction(fixture.Input)
			if err == nil {
				t.Fatalf("fault %s was not reached", point)
			}
			intent := filepath.Join(fixture.DataRoot, "transactions", SealTransactionID(fixture.Input), "intent.json")
			if _, statErr := os.Stat(intent); statErr != nil {
				after := sealTransaction199Snapshot(t, fixture.Root, fixture.HubRoot)
				if !reflect.DeepEqual(after, fixture.Baselines) {
					t.Fatalf("fault %s neither retained a journal nor restored exact bytes", point)
				}
				return
			}
			fixture.Input.Fault = nil
			result, retryErr := CommitSealTransaction(fixture.Input)
			if retryErr != nil {
				t.Fatalf("resume %s: %v", point, retryErr)
			}
			if result.Receipt.StateEffect != colony.LifecycleStateEffectCommitted || result.Outcome.Transaction.Stage != colony.TransactionStageVerified {
				t.Fatalf("resume result = %+v", result)
			}
		})
	}
}

func TestSealTransaction199ReplayExactlyOnce(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	t.Setenv("AETHER_HUB_DIR", fixture.HubRoot)
	t.Setenv(hivePolicyEnv, "promote")
	resetHivePolicyWarnOnceForTest()
	fixture.Input.DisablePostCommitPromotion = false
	if err := os.MkdirAll(filepath.Join(fixture.HubRoot, "hive"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture.HubRoot, hiveWisdomPath), []byte("{\"version\":2,\"entries\":[]}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome.OutcomeID != second.Outcome.OutcomeID || first.Receipt.ReceiptID != second.Receipt.ReceiptID {
		t.Fatalf("replay changed identity: first=%+v second=%+v", first, second)
	}
	eventsBytes, err := os.ReadFile(filepath.Join(fixture.DataRoot, "event-bus.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(eventsBytes, []byte(first.Outcome.OutcomeID)); got != 1 {
		t.Fatalf("seal event count = %d, want 1\n%s", got, eventsBytes)
	}
	var registry registryData
	readJSON199(t, filepath.Join(fixture.HubRoot, "registry", "registry.json"), &registry)
	if len(registry.Colonies) != 1 || registry.Colonies[0].Active {
		t.Fatalf("registry transition replayed incorrectly: %+v", registry)
	}
	if len(first.Promotions) != 1 || !first.Promotions[0].Promoted || !reflect.DeepEqual(first.Promotions, second.Promotions) {
		t.Fatalf("promotion receipts were not replayed exactly once: first=%+v second=%+v", first.Promotions, second.Promotions)
	}
	var wisdom hiveWisdomData
	readJSON199(t, filepath.Join(fixture.HubRoot, hiveWisdomPath), &wisdom)
	if len(wisdom.Entries) != 1 || len(wisdom.Entries[0].Evidence) != 1 {
		t.Fatalf("Hive promotion replayed instead of reusing its linked receipt: %+v", wisdom)
	}
}

func TestSealTransaction199ForcedTruth(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, true)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome.Disposition != colony.SealDispositionForcedIncomplete || result.Outcome.OutcomeKind != colony.OutcomeKindForcedIncompleteClosure || result.Outcome.OwnerReason != fixture.Input.Preflight.OwnerReason {
		t.Fatalf("forced outcome lost truth: %+v", result.Outcome)
	}
	for _, path := range []string{"COLONY_STATE.json", "session.json", "seal/outcome.json", "seal/receipt.json", "seal/closure-evidence.json", "seal/rollback.json"} {
		content, err := os.ReadFile(filepath.Join(fixture.DataRoot, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !bytes.Contains(content, []byte("forced_incomplete")) || !bytes.Contains(content, []byte(fixture.Input.Preflight.OwnerReason)) {
			t.Fatalf("%s lost forced discriminator/reason:\n%s", path, content)
		}
	}
	for _, mode := range []struct {
		name       string
		forceColor string
		noColor    string
	}{
		{name: "visual", forceColor: "1"},
		{name: "no-color", noColor: "1"},
	} {
		t.Run(mode.name, func(t *testing.T) {
			t.Setenv("AETHER_FORCE_COLOR", mode.forceColor)
			t.Setenv("NO_COLOR", mode.noColor)
			output := RenderSealOutcome(result)
			if !strings.Contains(output, "⛔ FORCED SEAL — COMPLETION NOT VERIFIED") || !strings.Contains(output, "The colony was force-sealed for recordkeeping. Completion was not verified.") {
				t.Fatalf("forced render missing exact truth copy:\n%s", output)
			}
			lower := strings.ToLower(output)
			for _, forbidden := range []string{"crowned anthill", "all phases completed", "goal achieved", "final form", "✅"} {
				if strings.Contains(lower, strings.ToLower(forbidden)) {
					t.Fatalf("forced render contains forbidden success discriminator %q:\n%s", forbidden, output)
				}
			}
		})
	}
}

func TestSealTransaction199RetainsActiveState(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	var state colony.ColonyState
	readJSON199(t, filepath.Join(fixture.DataRoot, "COLONY_STATE.json"), &state)
	if state.State != colony.StateCOMPLETED || state.SealOutcome == nil || state.SealOutcome.OutcomeID != result.Outcome.OutcomeID || !state.IsVerifiedCompletion() {
		t.Fatalf("retained state = %+v", state)
	}
	if _, err := os.Stat(filepath.Join(fixture.Root, ".aether", "CROWNED-ANTHILL.md")); err != nil {
		t.Fatalf("Crowned record not retained beside active state: %v", err)
	}
}

func TestSealTransaction199MemoryProvenance(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence.Memory) < 4 {
		t.Fatalf("memory provenance too small: %+v", result.Evidence.Memory)
	}
	for _, memory := range result.Evidence.Memory {
		if memory.Source == "" || memory.Store == "" || memory.RepositoryIdentity == "" || memory.Scope == "" || !strings.HasPrefix(memory.Digest, "sha256:") || memory.TransactionID != result.TransactionID || len(memory.EvidenceIDs)+len(memory.DecisionIDs) == 0 || memory.Sanitization == "" {
			t.Fatalf("incomplete memory provenance: %+v", memory)
		}
	}
	var roundTrip SealClosureEvidence
	readJSON199(t, filepath.Join(fixture.DataRoot, "seal", "closure-evidence.json"), &roundTrip)
	if !reflect.DeepEqual(roundTrip.Memory, result.Evidence.Memory) {
		t.Fatalf("memory provenance did not round trip")
	}
}

func TestSealTransaction199SignalRetention(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence.Signals) != 2 || result.Evidence.Signals[0].Classification != "expired_at_closure" || result.Evidence.Signals[1].Classification != "retained" {
		t.Fatalf("signal decisions = %+v", result.Evidence.Signals)
	}
	var file colony.PheromoneFile
	readJSON199(t, filepath.Join(fixture.DataRoot, "pheromones.json"), &file)
	if file.Signals[0].Active || !file.Signals[1].Active {
		t.Fatalf("signal retention state = %+v", file.Signals)
	}
}

func TestSealTransaction199WisdomPrivacy(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	decisions := map[string]SealWisdomDecision{}
	for _, decision := range result.Evidence.Wisdom {
		decisions[decision.ID] = decision
	}
	if !decisions["public-pattern"].Eligible || decisions["public-pattern"].Sensitive || decisions["public-pattern"].Sanitization != "passed" {
		t.Fatalf("public wisdom decision = %+v", decisions["public-pattern"])
	}
	if decisions["secret-pattern"].Eligible || !decisions["secret-pattern"].Sensitive || decisions["secret-pattern"].Promoted {
		t.Fatalf("private wisdom escaped filter: %+v", decisions["secret-pattern"])
	}
}

func TestSealTransaction199HivePolicy(t *testing.T) {
	for _, tc := range []struct {
		name   string
		policy string
		forced bool
		allow  bool
	}{
		{name: "read", policy: "read"},
		{name: "off", policy: "off"},
		{name: "invalid", policy: "typo"},
		{name: "forced", policy: "promote", forced: true},
		{name: "promote", policy: "promote", allow: true},
		{name: "unset", policy: "", allow: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(hivePolicyEnv, tc.policy)
			resetHivePolicyWarnOnceForTest()
			fixture := newSealTransaction199Fixture(t, tc.forced)
			result, err := CommitSealTransaction(fixture.Input)
			if err != nil {
				t.Fatal(err)
			}
			var public SealWisdomDecision
			for _, decision := range result.Evidence.Wisdom {
				if decision.ID == "public-pattern" {
					public = decision
				}
			}
			if public.Promoted {
				t.Fatalf("pre/post commit promotion disabled in fixture but decision claims promotion: %+v", public)
			}
			if public.PromotionAllowed != tc.allow {
				t.Fatalf("policy %q forced=%v allowed=%v, want %v: %+v", tc.policy, tc.forced, public.PromotionAllowed, tc.allow, public)
			}
			if !tc.allow && public.PromotionReceiptID != "" {
				t.Fatalf("prohibited policy minted promotion receipt: %+v", public)
			}
		})
	}
}

func TestSealTransaction199NextActions(t *testing.T) {
	fixture := newSealTransaction199Fixture(t, false)
	result, err := CommitSealTransaction(fixture.Input)
	if err != nil {
		t.Fatal(err)
	}
	output := RenderSealOutcome(result)
	status := strings.Index(output, "aether status")
	entomb := strings.Index(output, "aether entomb")
	if status < 0 || entomb < 0 || status > entomb || !strings.Contains(output, "Optional") {
		t.Fatalf("status is not primary and entomb optional:\n%s", output)
	}
	if result.PrimaryNext != "aether status" || result.OptionalNext != "aether entomb" {
		t.Fatalf("structured next actions = %q / %q", result.PrimaryNext, result.OptionalNext)
	}
}

func readJSON199(t *testing.T, path string, destination any) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(content, destination); err != nil {
		t.Fatalf("decode %s: %v\n%s", path, err, content)
	}
}
