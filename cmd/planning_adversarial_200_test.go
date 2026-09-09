package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	planningAdversarialHelperEnv    = "AETHER_PLANNING_ADVERSARIAL_200_HELPER"
	planningAdversarialRootEnv      = "AETHER_PLANNING_ADVERSARIAL_200_ROOT"
	planningAdversarialArgsEnv      = "AETHER_PLANNING_ADVERSARIAL_200_ARGS"
	planningAdversarialNowEnv       = "AETHER_PLANNING_ADVERSARIAL_200_NOW"
	planningAdversarialResultPrefix = "AETHER_PLANNING_ADVERSARIAL_200_RESULT="
)

type planningAdversarialCobraResult200 struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Error  string `json:"error,omitempty"`
	Code   int    `json:"code"`
}

// TestPlanningAdversarial200 composes the five independently repaired gaps at
// real command and process boundaries. Each hostile case starts only after a
// positive control has proved that the same public boundary is reachable.
func TestPlanningAdversarial200(t *testing.T) {
	if os.Getenv(planningAdversarialHelperEnv) == "1" {
		planningAdversarialCobraHelper200(t)
		return
	}

	t.Run("specification body and approval forgery", func(t *testing.T) {
		for _, attack := range []struct {
			name   string
			field  string
			mutate func(*colony.SpecRevision)
		}{
			{
				name: "typed body", field: "requirements",
				mutate: func(revision *colony.SpecRevision) {
					revision.Requirements[0].Description += " forged while copied hashes remain"
				},
			},
			{
				name: "approval receipt", field: "approval",
				mutate: func(revision *colony.SpecRevision) {
					revision.Approval.ApprovedBy += ":forged"
				},
			},
		} {
			t.Run(attack.name, func(t *testing.T) {
				root, candidate := planCandidateTestPending(t)
				planningAdversarialGitInit200(t, root)
				control := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(time.Minute), "plan", "--candidate")
				if control.Error != "" || !strings.Contains(control.Stdout, candidate.ID) || !strings.Contains(control.Stdout, "--accept-candidate") {
					t.Fatalf("positive candidate review = %+v", control)
				}

				state := mustReadSpecificationTestState(t, root)
				revision := &state.Specification.Revisions[len(state.Specification.Revisions)-1]
				attack.mutate(revision)
				specificationIntegrity200WriteState(t, root, state)
				before := planningRealRepo200Snapshot(t, root)
				refused := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(time.Minute), "plan", "--candidate")
				if refused.Error == "" || !strings.Contains(strings.ToLower(refused.Stderr+refused.Error), attack.field) {
					t.Fatalf("%s forgery was not diagnosed at Cobra boundary: %+v", attack.name, refused)
				}
				if after := planningRealRepo200Snapshot(t, root); !reflect.DeepEqual(before, after) {
					t.Fatalf("%s refusal changed state, stages, candidate, receipts, attempts, pointers, or manifests", attack.name)
				}
			})
		}
	})

	t.Run("candidate semantic delta and impact forgery", func(t *testing.T) {
		root, candidate := planCandidateTestPending(t)
		planningAdversarialGitInit200(t, root)
		control := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(time.Minute), planningAdversarialAcceptanceArgs200(candidate)...)
		if control.Error != "" || !strings.Contains(control.Stdout, string(planCandidateOperationAccept)) {
			t.Fatalf("positive exact acceptance = %+v", control)
		}

		for _, attack := range []struct {
			name   string
			mutate func(*colony.PlanCandidate)
		}{
			{
				name: "nested semantic delta",
				mutate: func(value *colony.PlanCandidate) {
					change := colony.PlanningSemanticChange{
						SemanticID: "task:forged-nested-delta", Kind: colony.PlanningSemanticChangeAdded,
						AfterHash: strings.Repeat("a", 64), EvidenceIDs: []string{value.Recommendation.EvidenceIDs[0]},
					}
					if err := colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionTasks, &change); err != nil {
						t.Fatal(err)
					}
					value.SemanticDelta.Tasks = append(value.SemanticDelta.Tasks, change)
				},
			},
			{
				name: "false authority impact",
				mutate: func(value *colony.PlanCandidate) {
					impact := colony.PlanningAuthorityImpact{
						Kind: colony.PlanningAuthorityCandidateStatus, SourceID: "forged-impact",
						AffectedSemanticIDs: []string{value.Proposal.SemanticID},
						Rationale:           "A candidate-authored assertion cannot grant authority.",
					}
					if err := colony.AddressPlanningAuthorityImpact(&impact); err != nil {
						t.Fatal(err)
					}
					value.SemanticDelta.AuthorityImpacts = append(value.SemanticDelta.AuthorityImpacts, impact)
				},
			},
		} {
			t.Run(attack.name, func(t *testing.T) {
				root, candidate := planCandidateTestPending(t)
				planningAdversarialGitInit200(t, root)
				attack.mutate(&candidate)
				if err := colony.AddressPlanningSemanticDelta(&candidate.SemanticDelta); err != nil {
					t.Fatal(err)
				}
				if err := addressPlanCandidateReviewPayload(&candidate); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, filepath.FromSlash(planningRouteCandidateRepositoryPath(candidate.Timeline.RunID)))
				planCandidateSemanticIntegrity200Write(t, path, candidate)
				planCandidateAcceptance200PrimeSessionLocks(t, root, candidate)
				before := planCandidateAcceptance200Inventory(t, root)
				refused := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(time.Minute), planningAdversarialAcceptanceArgs200(candidate)...)
				if refused.Error == "" || !strings.Contains(strings.ToLower(refused.Stderr+refused.Error), "derived") {
					t.Fatalf("%s forgery was not independently rejected: %+v", attack.name, refused)
				}
				planCandidateAcceptance200AssertInventory(t, root, before)
			})
		}
	})

	t.Run("intermediate link and component swap containment", func(t *testing.T) {
		for _, proof := range []string{
			`^TestRepositoryBootstrapContainment200$/^allowed_mutation_creates_only_verified_repository_data_and_locks$`,
			`^TestRepositoryBootstrapContainment200$/^intermediate_aether_link_is_refused_without_mutation$`,
			`^TestRepositoryBootstrapContainment200$/^component_swap_after_validation_fails_closed$`,
		} {
			planningAdversarialRunProof200(t, proof)
		}
	})

	t.Run("stale specification timeline acceptance and build writers", func(t *testing.T) {
		for _, proof := range []string{
			`^TestPlanningWriterConcurrentProcesses200$`,
			`^TestPlanningTimelineConcurrentProcesses200Migrated$`,
			`^TestPlanCandidateAcceptanceConcurrentProcesses200$`,
			`^TestBuildStartConcurrentProcesses200$`,
		} {
			planningAdversarialRunProof200(t, proof)
		}
	})

	t.Run("expiry boundary and early stale recovery", func(t *testing.T) {
		for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
			root, candidate := planCandidateTestPending(t)
			planningAdversarialGitInit200(t, root)
			activeBefore := planningExpiryActivePlanBytes200(t, mustReadSpecificationTestState(t, root))
			result := planningAdversarialRunCobra200(t, root, candidate.ExpiresAt.Add(offset), planningAdversarialAcceptanceArgs200(candidate)...)
			if offset < 0 {
				if result.Error != "" {
					t.Fatalf("just-before-expiry acceptance failed: %+v", result)
				}
				continue
			}
			if result.Error == "" || !strings.Contains(result.Stderr, "candidate_expired") ||
				strings.Contains(result.Stderr, "--acceptance-token") {
				t.Fatalf("expiry offset %s did not fail safely: %+v", offset, result)
			}
			after := mustReadSpecificationTestState(t, root)
			if !bytes.Equal(activeBefore, planningExpiryActivePlanBytes200(t, after)) {
				t.Fatalf("expiry offset %s changed active plan authority", offset)
			}
			persisted := planCandidateExpiry200ReadCandidate(t, root, candidate)
			if persisted.Status != colony.PlanCandidateExpired || persisted.Acceptance != nil {
				t.Fatalf("expiry offset %s persisted %+v", offset, persisted)
			}
		}

		root, candidate := planCandidateTestPending(t)
		planningAdversarialGitInit200(t, root)
		planningAdversarialAdvanceSpecification200(t, root, candidate.CreatedAt.Add(time.Minute))
		before := planningRealRepo200Snapshot(t, root)
		review := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(2*time.Minute), "plan", "--candidate")
		if review.Error != "" || !strings.Contains(review.Stdout, planCandidateRefreshCommand) ||
			!strings.Contains(review.Stdout, "specification_changed") || strings.Contains(review.Stdout, "--acceptance-token") {
			t.Fatalf("early stale review did not expose refresh-only recovery: %+v", review)
		}
		if after := planningRealRepo200Snapshot(t, root); !reflect.DeepEqual(before, after) {
			t.Fatal("early stale review mutated repository authority")
		}
		refused := planningAdversarialRunCobra200(t, root, candidate.CreatedAt.Add(2*time.Minute), planningAdversarialAcceptanceArgs200(candidate)...)
		if refused.Error == "" || !strings.Contains(refused.Stderr, "specification_changed") ||
			!strings.Contains(refused.Stderr, planCandidateRefreshCommand) {
			t.Fatalf("early stale acceptance did not fail with exact recovery: %+v", refused)
		}
		if after := planningRealRepo200Snapshot(t, root); !reflect.DeepEqual(before, after) {
			t.Fatal("early stale acceptance mutated repository authority")
		}
	})
}

// TestPlanningGapEdgeAccounting200 maps each spec-less truth to the named Go
// proof that owns it. The child test binary executes behavior, so this cannot
// pass by counting tags in a plan or source file.
func TestPlanningGapEdgeAccounting200(t *testing.T) {
	mappings := []struct {
		name  string
		proof string
	}{
		{"CEC-03/adjacency", `^TestRepositoryBootstrapContainment200$/^adjacent_prefix_is_not_repository_containment$`},
		{"CEC-03/empty-degenerate", `^TestRepositoryBootstrapContainment200$/^explicit_empty_data_root_is_refused$`},
		{"CEC-03/ordering-stability", `^TestPlanningTimelineConcurrentProcesses200Migrated$`},
		{"PLAN-02/adjacency", `^TestPlanCandidateSemanticIntegrity200NestedAdjacencyEmptyAndOrdering$`},
		{"PLAN-02/empty-degenerate", `^TestPlanCandidateSemanticIntegrity200NestedAdjacencyEmptyAndOrdering$`},
		{"PLAN-02/ordering-stability", `^TestPlanCandidateSemanticIntegrity200NestedAdjacencyEmptyAndOrdering$`},
		{"PLAN-04/idempotency", `^TestPlanCandidateAcceptanceIntegrity200$/^accepted_replay_is_exact_and_read_only$`},
		{"PLAN-04/concurrency-effect-ordering", `^TestPlanCandidateAcceptanceConcurrentProcesses200$`},
		{"PLAN-04/boundary-values", `^TestPlanningNumericBoundaries200$/^PLAN-04$/^boundary-values$`},
		{"PLAN-04/precision-overflow", `^TestPlanningNumericBoundaries200$/^PLAN-04$/^precision-overflow$`},
		{"PLAN-01/boundary-values", `^TestPlanningNumericBoundaries200$/^PLAN-01$/^boundary-values$`},
		{"PLAN-01/precision-overflow", `^TestPlanningNumericBoundaries200$/^PLAN-01$/^precision-overflow$`},
		{"PLAN-06/idempotency", `^TestBuildStartTransaction200ExactReplayIsReadOnly$`},
		{"PLAN-06/concurrency-effect-ordering", `^TestBuildStartConcurrentProcesses200$`},
	}
	if len(mappings) != 14 {
		t.Fatalf("edge mapping count = %d, want 14", len(mappings))
	}
	for _, mapping := range mappings {
		mapping := mapping
		t.Run(mapping.name, func(t *testing.T) {
			planningAdversarialRunProof200(t, mapping.proof)
		})
	}
}

const (
	planningAdversarialTimelineHelperEnv      = "AETHER_PLANNING_TIMELINE_200_MIGRATED_HELPER"
	planningAdversarialTimelineRootEnv        = "AETHER_PLANNING_TIMELINE_200_MIGRATED_ROOT"
	planningAdversarialTimelineCardEnv        = "AETHER_PLANNING_TIMELINE_200_MIGRATED_CARD"
	planningAdversarialTimelinePredecessorEnv = "AETHER_PLANNING_TIMELINE_200_MIGRATED_PREDECESSOR"
	planningAdversarialTimelineReceiptEnv     = "AETHER_PLANNING_TIMELINE_200_MIGRATED_RECEIPT"
	planningAdversarialTimelineHoldEnv        = "AETHER_PLANNING_TIMELINE_200_MIGRATED_HOLD"
)

// TestPlanningTimelineConcurrentProcesses200Migrated is the Plan 39 fixture
// migration for TestPlanningTimelineConcurrentProcesses200. The original test
// predates nested canonical record hashes; this proof retains its exact two-OS-
// process ordering contract with freshly addressed cards and pipe barriers.
func TestPlanningTimelineConcurrentProcesses200Migrated(t *testing.T) {
	if os.Getenv(planningAdversarialTimelineHelperEnv) == "1" {
		planningAdversarialTimelineChild200(t)
		return
	}

	root := t.TempDir()
	planningAdversarialGitInit200(t, root)
	outside := t.TempDir()
	mustWritePlanningMutationFile(t, filepath.Join(outside, "sentinel.txt"), []byte("outside-timeline-migrated"))
	outsideBefore := snapshotPlanningMutationTree(t, outside)
	first := planningAdversarialCanonicalTimelineCard200(t, 1, "planning-session-process-race-migrated", time.Date(2026, time.September, 9, 14, 0, 0, 0, time.UTC))
	second := planningAdversarialCanonicalTimelineCard200(t, 2, first.RunID, time.Date(2026, time.September, 9, 14, 1, 0, 0, time.UTC))

	firstChild := planningAdversarialStartTimelineChild200(t, root, "route-process-one", first, colony.PlanningIterationCard{}, true)
	secondChild := planningAdversarialStartTimelineChild200(t, root, "route-process-two", second, first, false)
	awaitPlanningMutationPipe(t, firstChild.ready, "first migrated timeline child ready")
	awaitPlanningMutationPipe(t, secondChild.ready, "second migrated timeline child ready")
	releasePlanningMutationPipe(t, firstChild.release)
	awaitPlanningMutationPipe(t, firstChild.holdReady, "first migrated timeline child holds session")
	releasePlanningMutationPipe(t, secondChild.release)
	releasePlanningMutationPipe(t, firstChild.holdRelease)

	firstResult := awaitPlanningMutationChild(t, firstChild)
	secondResult := awaitPlanningMutationChild(t, secondChild)
	if firstResult.Error != "" || secondResult.Error != "" || firstResult.Receipt == nil || secondResult.Receipt == nil {
		t.Fatalf("migrated timeline children failed: first=%+v second=%+v", firstResult, secondResult)
	}
	loaded, err := loadPlanningTimeline(root, first.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Cards) != 2 || loaded.Index == nil ||
		loaded.Cards[0].ID != firstResult.Receipt.CardID || loaded.Cards[1].ID != secondResult.Receipt.CardID ||
		loaded.Index.Entries[0].PreviousCardHash != "" || loaded.Index.Entries[1].PreviousCardHash != first.ContentHash ||
		loaded.Index.LastCardHash != secondResult.Receipt.CardHash || loaded.Index.TimelineDigest != secondResult.Receipt.TimelineDigest {
		t.Fatalf("migrated serialized timeline = %+v; first=%+v second=%+v", loaded, firstResult, secondResult)
	}
	if after := snapshotPlanningMutationTree(t, outside); !equalPlanningMutationSnapshot(outsideBefore, after) {
		t.Fatalf("migrated timeline race changed outside tree\nbefore: %#v\nafter: %#v", outsideBefore, after)
	}
}

func planningAdversarialCanonicalTimelineCard200(t *testing.T, iteration int, runID string, createdAt time.Time) colony.PlanningIterationCard {
	t.Helper()
	card := validPlanningIterationCardForTest(t, iteration, createdAt)
	card.RunID = runID
	for index := range card.DimensionAssessments {
		if err := colony.AddressPlanningDimensionAssessment(&card.DimensionAssessments[index]); err != nil {
			t.Fatal(err)
		}
	}
	card.WeakestGap = card.DimensionAssessments[2].RemainingGap
	for index := range card.SemanticDelta.Phases {
		if err := colony.AddressPlanningSemanticChange(colony.PlanningSemanticSectionPhases, &card.SemanticDelta.Phases[index]); err != nil {
			t.Fatal(err)
		}
	}
	for index := range card.SemanticDelta.AuthorityImpacts {
		if err := colony.AddressPlanningAuthorityImpact(&card.SemanticDelta.AuthorityImpacts[index]); err != nil {
			t.Fatal(err)
		}
	}
	if err := colony.AddressPlanningSemanticDelta(&card.SemanticDelta); err != nil {
		t.Fatal(err)
	}
	card.Decision.SelectedGapID = card.WeakestGap.ID
	card.Decision.ResidualGapIDs = []string{card.WeakestGap.ID}
	if err := addressPlanningStopDecision(&card.Decision); err != nil {
		t.Fatal(err)
	}
	canonical, _, err := canonicalPlanningTimelineCard(card)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func planningAdversarialStartTimelineChild200(t *testing.T, root, receiptID string, card, predecessor colony.PlanningIterationCard, hold bool) *planningMutationChildProcess {
	t.Helper()
	readyRead, readyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	releaseRead, releaseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	resultRead, resultWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	holdReadyRead, holdReadyWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	holdReleaseRead, holdReleaseWrite, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	cardJSON, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	predecessorJSON := []byte("{}")
	if card.Iteration > 1 {
		predecessorJSON, err = json.Marshal(predecessor)
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPlanningTimelineConcurrentProcesses200Migrated$", "-test.count=1")
	command.Env = replaceProcessEnv(os.Environ(), map[string]string{
		planningAdversarialTimelineHelperEnv:      "1",
		planningAdversarialTimelineRootEnv:        root,
		planningAdversarialTimelineCardEnv:        string(cardJSON),
		planningAdversarialTimelinePredecessorEnv: string(predecessorJSON),
		planningAdversarialTimelineReceiptEnv:     receiptID,
		planningAdversarialTimelineHoldEnv:        map[bool]string{false: "0", true: "1"}[hold],
	})
	command.ExtraFiles = []*os.File{readyWrite, releaseRead, resultWrite, holdReadyWrite, holdReleaseRead}
	output := &bytes.Buffer{}
	command.Stdout, command.Stderr = output, output
	if err := command.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	_ = readyWrite.Close()
	_ = releaseRead.Close()
	_ = resultWrite.Close()
	_ = holdReadyWrite.Close()
	_ = holdReleaseRead.Close()
	return &planningMutationChildProcess{
		command: command, output: output, ready: readyRead, release: releaseWrite, result: resultRead,
		holdReady: holdReadyRead, holdRelease: holdReleaseWrite, cancel: cancel,
	}
}

func planningAdversarialTimelineChild200(t *testing.T) {
	t.Helper()
	ready := os.NewFile(3, "planning-adversarial-timeline-ready")
	release := os.NewFile(4, "planning-adversarial-timeline-release")
	result := os.NewFile(5, "planning-adversarial-timeline-result")
	holdReady := os.NewFile(6, "planning-adversarial-timeline-hold-ready")
	holdRelease := os.NewFile(7, "planning-adversarial-timeline-hold-release")
	defer ready.Close()
	defer release.Close()
	defer result.Close()
	defer holdReady.Close()
	defer holdRelease.Close()
	if _, err := ready.WriteString("ready\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadFull(release, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	var card colony.PlanningIterationCard
	if err := json.Unmarshal([]byte(os.Getenv(planningAdversarialTimelineCardEnv)), &card); err != nil {
		t.Fatal(err)
	}
	opts := planningTimelineAppendOptions{ReceiptID: os.Getenv(planningAdversarialTimelineReceiptEnv)}
	if card.Iteration > 1 {
		var predecessor colony.PlanningIterationCard
		if err := json.Unmarshal([]byte(os.Getenv(planningAdversarialTimelinePredecessorEnv)), &predecessor); err != nil {
			t.Fatal(err)
		}
		opts.PreviousCardHash = predecessor.ContentHash
	}
	if os.Getenv(planningAdversarialTimelineHoldEnv) == "1" {
		opts.Fault = func(point string) error {
			if point != "after_validation" {
				return nil
			}
			if _, err := holdReady.WriteString("held\n"); err != nil {
				return err
			}
			_, err := io.ReadFull(holdRelease, make([]byte, 1))
			return err
		}
	}
	receipt, appendErr := appendPlanningIterationCard(os.Getenv(planningAdversarialTimelineRootEnv), card, opts)
	response := planningMutationProcessResult{}
	if appendErr != nil {
		response.Error = appendErr.Error()
	} else {
		response.Receipt = &receipt
	}
	if err := json.NewEncoder(result).Encode(response); err != nil {
		t.Fatal(err)
	}
}

func planningAdversarialRunCobra200(t *testing.T, root string, now time.Time, args ...string) planningAdversarialCobraResult200 {
	t.Helper()
	encodedArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	command := exec.Command(os.Args[0], "-test.run=^TestPlanningAdversarial200$", "-test.count=1")
	command.Dir = root
	command.Env = replaceProcessEnv(os.Environ(), map[string]string{
		planningAdversarialHelperEnv: "1",
		planningAdversarialRootEnv:   root,
		planningAdversarialArgsEnv:   string(encodedArgs),
		planningAdversarialNowEnv:    now.UTC().Format(time.RFC3339Nano),
		"AETHER_ROOT":                root,
		"COLONY_DATA_DIR":            filepath.Join(root, ".aether", "data"),
		"AETHER_OUTPUT_MODE":         "json",
		"NO_COLOR":                   "1",
		"HOME":                       home,
		"USERPROFILE":                home,
	})
	output, runErr := command.CombinedOutput()
	if runErr != nil {
		t.Fatalf("Cobra helper process failed: %v\n%s", runErr, output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, planningAdversarialResultPrefix) {
			continue
		}
		var result planningAdversarialCobraResult200
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, planningAdversarialResultPrefix)), &result); err != nil {
			t.Fatalf("decode Cobra helper result: %v\n%s", err, output)
		}
		return result
	}
	t.Fatalf("Cobra helper returned no result:\n%s", output)
	return planningAdversarialCobraResult200{}
}

func planningAdversarialCobraHelper200(t *testing.T) {
	t.Helper()
	root := os.Getenv(planningAdversarialRootEnv)
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv(planningAdversarialArgsEnv)), &args); err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339Nano, os.Getenv(planningAdversarialNowEnv))
	if err != nil {
		t.Fatal(err)
	}
	saveGlobals(t)
	resetRootCmd(t)
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	planCandidateNow = func() time.Time { return now }
	var out, errOut bytes.Buffer
	stdout, stderr = &out, &errOut
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	commandErr := Execute()
	result := planningAdversarialCobraResult200{Stdout: out.String(), Stderr: errOut.String()}
	if commandErr != nil {
		result.Error = commandErr.Error()
		var rendered renderedCommandError
		if errors.As(commandErr, &rendered) {
			result.Code = rendered.code
		} else {
			result.Code = 1
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(planningAdversarialResultPrefix + string(encoded))
}

func planningAdversarialRunProof200(t *testing.T, pattern string) {
	t.Helper()
	command := exec.Command(os.Args[0], "-test.run="+pattern, "-test.count=1")
	command.Env = replaceProcessEnv(os.Environ(), map[string]string{
		planningAdversarialHelperEnv: "",
	})
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("mapped proof %s failed: %v\n%s", pattern, err, output)
	}
}

func planningAdversarialAcceptanceArgs200(candidate colony.PlanCandidate) []string {
	request := planCandidateTestAcceptanceRequest(candidate)
	return []string{
		"plan", "--accept-candidate", request.CandidateID,
		"--spec-revision", request.SpecificationRevisionID,
		"--spec-hash", request.SpecificationRevisionHash,
		"--base-plan-revision", request.BasePlanRevisionID,
		"--timeline-digest", request.TimelineDigest,
		"--proposal-hash", request.ProposalHash,
		"--acceptance-token", request.AcceptanceToken,
	}
}

func planningAdversarialGitInit200(t *testing.T, root string) {
	t.Helper()
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("initialize real repository: %v\n%s", err, output)
	}
}

func planningAdversarialAdvanceSpecification200(t *testing.T, root string, createdAt time.Time) {
	t.Helper()
	state := mustReadSpecificationTestState(t, root)
	current, ok := currentSpecificationRevision(*state.Specification)
	if !ok {
		t.Fatal("candidate fixture has no current specification")
	}
	successorSpecification, successor, _, err := buildSpecificationSuccessor(*state.Specification, state.Plan, specificationRevisionRequest{
		PredecessorRevisionID:  current.ID,
		PredecessorContentHash: current.ContentHash,
		Scope:                  current.Scope,
		CreatedAt:              createdAt,
		Changes: []specificationRevisionChange{{
			Operation: specificationChangeModify,
			Section:   specificationSectionRequirements,
			TargetID:  current.Requirements[0].ID,
			Item: specificationItemInput{
				Description: "A successor contract makes the older plan candidate stale.",
				EvidenceIDs: []string{"owner:adversarial-successor-200"},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	token := specificationApprovalToken(successorSpecification.ID, successor.ID, successor.ContentHash)
	approval, err := buildSpecificationApprovalReceipt(successorSpecification.ID, successor, specificationApprovalRequest{
		RevisionID: successor.ID, RevisionContentHash: successor.ContentHash, ApprovalToken: token,
		ApprovedBy: "owner:adversarial-successor-200", ApprovedAt: createdAt.Add(time.Nanosecond),
	}, specificationApprovalTokenHash(token))
	if err != nil {
		t.Fatal(err)
	}
	latest := len(successorSpecification.Revisions) - 1
	successorSpecification.Revisions[latest].Status = colony.SpecStatusApproved
	successorSpecification.Revisions[latest].Approval = &approval
	state.Specification = &successorSpecification
	specificationIntegrity200WriteState(t, root, state)
}
