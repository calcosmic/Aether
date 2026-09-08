package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
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
)

// TestPlanCandidateAcceptanceIntegrity200 proves that content addressing is
// not treated as truth. Every mutation below is re-addressed into a perfectly
// self-consistent candidate before acceptance; only independent derivation
// from the locked base/specification/proposal tuple can reject it.
func TestPlanCandidateAcceptanceIntegrity200(t *testing.T) {
	seedRoot, seedCandidate := planCandidateTestPending(t)
	evidenceID := seedCandidate.Recommendation.EvidenceIDs[0]
	digest := func(value string) string {
		return strings.Repeat(value, 64)
	}
	change := func(section colony.PlanningSemanticSection, semanticID string, kind colony.PlanningSemanticChangeKind) colony.PlanningSemanticChange {
		value := colony.PlanningSemanticChange{SemanticID: semanticID, Kind: kind, EvidenceIDs: []string{evidenceID}}
		switch kind {
		case colony.PlanningSemanticChangeAdded:
			value.AfterHash = digest("a")
		case colony.PlanningSemanticChangeModified:
			value.BeforeHash, value.AfterHash = digest("b"), digest("c")
		case colony.PlanningSemanticChangeRemoved:
			value.BeforeHash = digest("d")
		}
		if err := colony.AddressPlanningSemanticChange(section, &value); err != nil {
			t.Fatalf("address forged %s change: %v", section, err)
		}
		return value
	}

	mutations := []struct {
		name   string
		mutate func(*colony.PlanCandidate)
	}{
		{name: "added phase", mutate: func(candidate *colony.PlanCandidate) {
			candidate.SemanticDelta.Phases = append(candidate.SemanticDelta.Phases, change(colony.PlanningSemanticSectionPhases, "phase-forged", colony.PlanningSemanticChangeAdded))
		}},
		{name: "changed task", mutate: func(candidate *colony.PlanCandidate) {
			candidate.SemanticDelta.Tasks = append(candidate.SemanticDelta.Tasks, change(colony.PlanningSemanticSectionTasks, "task-forged", colony.PlanningSemanticChangeModified))
		}},
		{name: "removed dependency", mutate: func(candidate *colony.PlanCandidate) {
			candidate.SemanticDelta.Dependencies = append(candidate.SemanticDelta.Dependencies, change(colony.PlanningSemanticSectionDependencies, "dependency-forged", colony.PlanningSemanticChangeRemoved))
		}},
		{name: "changed acceptance check", mutate: func(candidate *colony.PlanCandidate) {
			candidate.SemanticDelta.AcceptanceChecks = append(candidate.SemanticDelta.AcceptanceChecks, change(colony.PlanningSemanticSectionAcceptanceChecks, "acceptance-forged", colony.PlanningSemanticChangeModified))
		}},
		{name: "removed recovery", mutate: func(candidate *colony.PlanCandidate) {
			candidate.SemanticDelta.RecoveryExpectations = append(candidate.SemanticDelta.RecoveryExpectations, change(colony.PlanningSemanticSectionRecoveryExpectations, "recovery-forged", colony.PlanningSemanticChangeRemoved))
		}},
		{name: "false authority impact", mutate: func(candidate *colony.PlanCandidate) {
			impact := colony.PlanningAuthorityImpact{
				Kind: colony.PlanningAuthorityCandidateStatus, SourceID: "forged-authority",
				AffectedSemanticIDs: []string{candidate.Proposal.SemanticID},
				Rationale:           "A self-authored claim cannot create acceptance authority.",
			}
			if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
				t.Fatalf("address forged authority impact: %v", err)
			}
			candidate.SemanticDelta.AuthorityImpacts = append(candidate.SemanticDelta.AuthorityImpacts, impact)
		}},
		{name: "overbroad affected scope", mutate: func(candidate *colony.PlanCandidate) {
			candidate.Proposal.AffectedSemanticIDs = append(candidate.Proposal.AffectedSemanticIDs, "semantic-forged-overbroad")
		}},
		{name: "missing affected scope", mutate: func(candidate *colony.PlanCandidate) {
			candidate.Proposal.AffectedSemanticIDs = nil
		}},
	}

	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			root := planCandidateAcceptance200CloneRepository(t, seedRoot)
			candidate := planCandidateSemanticIntegrity200Clone(t, seedCandidate)
			test.mutate(&candidate)
			if err := colony.AddressPlanningSemanticDelta(&candidate.SemanticDelta); err != nil {
				t.Fatalf("re-address forged delta: %v", err)
			}
			if err := addressPlanCandidateReviewPayload(&candidate); err != nil {
				t.Fatalf("re-address forged candidate: %v", err)
			}
			planCandidateSemanticIntegrity200Write(t, filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID))), candidate)
			before := planCandidateAcceptance200Inventory(t, root)
			if _, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
				AcceptedBy: "owner:plan-31-integrity", AcceptedAt: time.Date(2026, time.September, 9, 9, 0, 0, 0, time.UTC),
			}); err == nil || !strings.Contains(err.Error(), "derived") {
				t.Fatalf("self-consistent forged candidate acceptance error = %v, want derived-authority refusal", err)
			}
			planCandidateAcceptance200AssertInventory(t, root, before)
		})
	}

	t.Run("all transaction apply faults roll back", func(t *testing.T) {
		for _, point := range []string{
			"after_validation", "after_stage:target-0001", "after_stage:target-0002",
			"after_stage:target-0003", "after_stage:target-0004", "after_intent",
			"after_target_commit:target-0001", "after_target_commit:target-0002",
			"after_target_commit:target-0003", "after_target_commit:target-0004",
			"after_root_commit:data", "after_global_verification",
		} {
			t.Run(strings.ReplaceAll(point, ":", "_"), func(t *testing.T) {
				root := planCandidateAcceptance200CloneRepository(t, seedRoot)
				candidate := planCandidateSemanticIntegrity200Clone(t, seedCandidate)
				before := planCandidateAcceptance200AuthorityInventory(t, root)
				injected := errors.New("plan-31 injected acceptance fault")
				_, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
					AcceptedBy: "owner:plan-31-rollback", AcceptedAt: time.Date(2026, time.September, 9, 9, 5, 0, 0, time.UTC),
					Fault: func(got string) error {
						if got == point {
							return injected
						}
						return nil
					},
				})
				if !errors.Is(err, injected) {
					t.Fatalf("fault %q error = %v, want injected error", point, err)
				}
				planCandidateAcceptance200AssertAuthorityInventory(t, root, before)
			})
		}
	})

	t.Run("accepted replay is exact and read only", func(t *testing.T) {
		root := planCandidateAcceptance200CloneRepository(t, seedRoot)
		candidate := planCandidateSemanticIntegrity200Clone(t, seedCandidate)
		request := planCandidateTestAcceptanceRequest(candidate)
		accepted, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{
			AcceptedBy: "owner:plan-31-replay", AcceptedAt: time.Date(2026, time.September, 9, 9, 10, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("accept exact candidate: %v", err)
		}
		before := planCandidateAcceptance200Inventory(t, root)
		replayed, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{
			AcceptedBy: "different-clock-independent-reader", AcceptedAt: time.Date(2036, time.September, 9, 9, 10, 0, 0, time.UTC),
		})
		if err != nil {
			t.Fatalf("replay exact candidate: %v", err)
		}
		if !replayed.Replayed || !reflect.DeepEqual(replayed.Candidate, accepted.Candidate) ||
			!reflect.DeepEqual(replayed.Revision, accepted.Revision) || !reflect.DeepEqual(replayed.Receipt, accepted.Receipt) {
			t.Fatalf("replay diverged from original acceptance:\nfirst=%+v\nreplay=%+v", accepted, replayed)
		}
		planCandidateAcceptance200AssertInventory(t, root, before)
	})
}

// TestPlanCandidateAcceptanceConcurrentProcesses200 runs two real OS
// processes against one candidate. The repository lock must be acquired before
// either process reloads authority, yielding exactly one activation and one
// byte-identical replay.
func TestPlanCandidateAcceptanceConcurrentProcesses200(t *testing.T) {
	if os.Getenv("AETHER_PLAN_ACCEPTANCE_200_HELPER") == "1" {
		planCandidateAcceptance200ProcessHelper(t)
		return
	}
	root, candidate := planCandidateTestPending(t)
	requestBytes, err := json.Marshal(planCandidateTestAcceptanceRequest(candidate))
	if err != nil {
		t.Fatal(err)
	}
	gate := filepath.Join(t.TempDir(), "start")
	type process struct {
		command *exec.Cmd
		output  bytes.Buffer
	}
	processes := make([]process, 2)
	for index := range processes {
		command := exec.Command(os.Args[0], "-test.run=^TestPlanCandidateAcceptanceConcurrentProcesses200$")
		command.Env = append(os.Environ(),
			"AETHER_PLAN_ACCEPTANCE_200_HELPER=1",
			"AETHER_PLAN_ACCEPTANCE_200_ROOT="+root,
			"AETHER_PLAN_ACCEPTANCE_200_GATE="+gate,
			"AETHER_PLAN_ACCEPTANCE_200_REQUEST="+string(requestBytes),
		)
		command.Stdout, command.Stderr = &processes[index].output, &processes[index].output
		processes[index].command = command
		if err := command.Start(); err != nil {
			t.Fatalf("start acceptance helper %d: %v", index, err)
		}
	}
	if err := os.WriteFile(gate, []byte("go\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	results := make([]planCandidateAcceptance200ProcessResult, len(processes))
	for index := range processes {
		if err := processes[index].command.Wait(); err != nil {
			t.Fatalf("acceptance helper %d: %v\n%s", index, err, processes[index].output.String())
		}
		var encoded string
		for _, line := range strings.Split(strings.TrimSpace(processes[index].output.String()), "\n") {
			if strings.HasPrefix(line, "{") {
				encoded = line
				break
			}
		}
		if err := json.Unmarshal([]byte(encoded), &results[index]); err != nil {
			t.Fatalf("decode helper %d output %q: %v", index, processes[index].output.String(), err)
		}
		if results[index].Error != "" {
			t.Fatalf("acceptance helper %d refused exact replay: %s", index, results[index].Error)
		}
	}
	if results[0].Result.Replayed == results[1].Result.Replayed {
		t.Fatalf("process replay flags = %t/%t, want one activation and one replay", results[0].Result.Replayed, results[1].Result.Replayed)
	}
	if !reflect.DeepEqual(results[0].Result.Candidate, results[1].Result.Candidate) ||
		!reflect.DeepEqual(results[0].Result.Revision, results[1].Result.Revision) ||
		!reflect.DeepEqual(results[0].Result.Receipt, results[1].Result.Receipt) {
		t.Fatalf("serialized process results diverged:\nfirst=%+v\nsecond=%+v", results[0], results[1])
	}
	state, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Plan.Revisions) != 1 || len(state.Plan.Candidates) != 1 {
		t.Fatalf("concurrent acceptance wrote duplicate lineage: revisions=%d candidates=%d", len(state.Plan.Revisions), len(state.Plan.Candidates))
	}
}

type planCandidateAcceptance200ProcessResult struct {
	Result planCandidateAcceptanceResult `json:"result"`
	Error  string                        `json:"error,omitempty"`
}

func planCandidateAcceptance200ProcessHelper(t *testing.T) {
	root := os.Getenv("AETHER_PLAN_ACCEPTANCE_200_ROOT")
	gate := os.Getenv("AETHER_PLAN_ACCEPTANCE_200_GATE")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(gate); err == nil {
			break
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for process start gate")
		}
		time.Sleep(time.Millisecond)
	}
	var request planCandidateAcceptanceRequest
	if err := json.Unmarshal([]byte(os.Getenv("AETHER_PLAN_ACCEPTANCE_200_REQUEST")), &request); err != nil {
		t.Fatal(err)
	}
	result, err := acceptPlanCandidate(root, request, planCandidateAcceptanceOptions{
		AcceptedBy: "owner:plan-31-process", AcceptedAt: time.Date(2026, time.September, 9, 9, 15, 0, 0, time.UTC),
	})
	response := planCandidateAcceptance200ProcessResult{Result: result}
	if err != nil {
		response.Error = err.Error()
	}
	content, marshalErr := json.Marshal(response)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	fmt.Println(string(content))
}

type planCandidateAcceptance200PathState struct {
	Mode os.FileMode
	Data []byte
}

func planCandidateAcceptance200Inventory(t *testing.T, root string) map[string]planCandidateAcceptance200PathState {
	t.Helper()
	result := make(map[string]planCandidateAcceptance200PathState)
	start := filepath.Join(root, ".aether")
	if err := filepath.WalkDir(start, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		state := planCandidateAcceptance200PathState{Mode: info.Mode()}
		if info.Mode().IsRegular() {
			state.Data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, readErr := os.Readlink(path)
			if readErr != nil {
				return readErr
			}
			state.Data = []byte(target)
		}
		result[filepath.ToSlash(relative)] = state
		return nil
	}); err != nil {
		t.Fatalf("inventory acceptance repository: %v", err)
	}
	return result
}

func planCandidateAcceptance200AssertInventory(t *testing.T, root string, want map[string]planCandidateAcceptance200PathState) {
	t.Helper()
	got := planCandidateAcceptance200Inventory(t, root)
	if !reflect.DeepEqual(got, want) {
		keys := make([]string, 0, len(got)+len(want))
		seen := make(map[string]struct{})
		for key := range got {
			seen[key] = struct{}{}
		}
		for key := range want {
			seen[key] = struct{}{}
		}
		for key := range seen {
			if !reflect.DeepEqual(got[key], want[key]) {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		t.Fatalf("acceptance refusal changed repository paths: %s", strings.Join(keys, ", "))
	}
}

func planCandidateAcceptance200AuthorityInventory(t *testing.T, root string) map[string]planCandidateAcceptance200PathState {
	t.Helper()
	result := planCandidateAcceptance200Inventory(t, root)
	for path := range result {
		if strings.HasPrefix(path, ".aether/data/.aether-transactions/") ||
			strings.HasPrefix(path, ".aether/data/transactions/") ||
			path == ".aether/data/.aether-transactions" || path == ".aether/data/transactions" {
			delete(result, path)
		}
	}
	return result
}

func planCandidateAcceptance200AssertAuthorityInventory(t *testing.T, root string, want map[string]planCandidateAcceptance200PathState) {
	t.Helper()
	got := planCandidateAcceptance200AuthorityInventory(t, root)
	if !reflect.DeepEqual(got, want) {
		keys := make([]string, 0, len(got)+len(want))
		seen := make(map[string]struct{})
		for key := range got {
			seen[key] = struct{}{}
		}
		for key := range want {
			seen[key] = struct{}{}
		}
		for key := range seen {
			if !reflect.DeepEqual(got[key], want[key]) {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		t.Fatalf("acceptance fault changed authority paths: %s", strings.Join(keys, ", "))
	}
}

func planCandidateAcceptance200CloneRepository(t *testing.T, source string) string {
	t.Helper()
	destination := t.TempDir()
	if err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}
		target := filepath.Join(destination, relative)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch {
		case info.IsDir():
			return os.Mkdir(target, info.Mode().Perm())
		case info.Mode().IsRegular():
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(target, content, info.Mode().Perm())
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		default:
			return fmt.Errorf("unsupported fixture path %s with mode %s", relative, info.Mode())
		}
	}); err != nil {
		t.Fatalf("clone acceptance repository: %v", err)
	}
	return destination
}
