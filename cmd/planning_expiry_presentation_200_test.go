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

func TestPlanningExpiryPresentation200(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	authority := planCandidateExpiry200Authority(t, root, candidate)
	current, err := reviewPlanCandidateAt(root, candidate.ExpiresAt.Add(-time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}

	staleAuthority := authority
	staleAuthority.SpecificationRevisionID += "-successor"
	stale := planningExpiryReview200(current, candidate, assessPlanCandidateStanding(candidate, staleAuthority, candidate.ExpiresAt.Add(-time.Nanosecond)))
	expired := planningExpiryReview200(current, candidate, assessPlanCandidateStanding(candidate, authority, candidate.ExpiresAt))

	acceptedRoot, acceptedCandidate := planCandidateTestPending(t)
	acceptedResult, err := acceptPlanCandidate(acceptedRoot, planCandidateTestAcceptanceRequest(acceptedCandidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:presentation-200",
		AcceptedAt: acceptedCandidate.ExpiresAt.Add(-time.Nanosecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted := current
	accepted.Candidate = acceptedResult.Candidate
	accepted.Standing = planCandidateStandingAccepted
	accepted.Acceptance = planCandidateAcceptanceRequest{}
	accepted.AcceptanceCommand = ""
	accepted.Refusal = nil

	cases := []struct {
		name             string
		review           planCandidateReview
		standing         planCandidateStanding
		next             string
		why              string
		stateEffect      planCandidateStateEffect
		acceptance       bool
		required         []string
		forbidden        []string
		executionActions []string
	}{
		{
			name: "current", review: current, standing: planCandidateStandingCurrent,
			next: current.AcceptanceCommand, stateEffect: planCandidateStateEffectUnchanged, acceptance: true,
			required:  []string{"CANDIDATE — NOT ACTIVE", "Accept this candidate?", current.AcceptanceCommand},
			forbidden: []string{"aether build", "aether run", planCandidateRefreshCommand},
		},
		{
			name: "specification stale", review: stale, standing: planCandidateStandingStale,
			next: planCandidateRefreshCommand, why: "specification_changed", stateEffect: planCandidateStateEffectUnchanged,
			required:  []string{"CANDIDATE — NOT ACTIVE", "Why unavailable: specification_changed", "State: unchanged", "Next: " + planCandidateRefreshCommand},
			forbidden: []string{"--acceptance-token", "Accept this candidate?", "aether build", "aether run"},
		},
		{
			name: "expired", review: expired, standing: planCandidateStandingExpired,
			next: planCandidateRefreshCommand, why: "candidate_expired", stateEffect: planCandidateStateEffectUnchanged,
			required:  []string{"CANDIDATE — NOT ACTIVE", "Why unavailable: candidate_expired", "State: unchanged", "Next: " + planCandidateRefreshCommand},
			forbidden: []string{"--acceptance-token", "Accept this candidate?", "aether build", "aether run"},
		},
		{
			name: "accepted", review: accepted, standing: planCandidateStandingAccepted,
			next: "aether build | aether run", stateEffect: planCandidateStateEffectUnchanged,
			required:         []string{"ACCEPTED PLAN — ACTIVE", "Standing: accepted", "aether build", "aether run"},
			forbidden:        []string{"--acceptance-token", "Accept this candidate?", planCandidateRefreshCommand},
			executionActions: []string{"aether build", "aether run"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			projection := projectPlanningCandidate(tc.review)
			if projection.Standing != tc.standing || projection.ExpiresAt != tc.review.Candidate.ExpiresAt.UTC() ||
				projection.WhyUnavailable != tc.why || projection.StateEffect != tc.stateEffect ||
				projection.ActivePlanEffect != planCandidateActivePlanEffectUnchanged ||
				projection.AcceptanceAvailable != tc.acceptance || projection.Next != tc.next ||
				!reflect.DeepEqual(projection.ExecutionActions, tc.executionActions) {
				t.Fatalf("projection authority = %+v", projection)
			}
			if tc.acceptance && projection.AcceptanceCommand != current.AcceptanceCommand {
				t.Fatalf("current acceptance command = %q, want %q", projection.AcceptanceCommand, current.AcceptanceCommand)
			}

			encoded, marshalErr := json.Marshal(projection)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			jsonBody := string(encoded)
			nextJSON, marshalErr := json.Marshal(tc.next)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			for _, want := range []string{
				`"standing":"` + string(tc.standing) + `"`,
				`"expires_at":"` + tc.review.Candidate.ExpiresAt.UTC().Format(time.RFC3339Nano) + `"`,
				`"state_effect":"` + string(tc.stateEffect) + `"`,
				`"active_plan_effect":"unchanged"`,
				`"acceptance_available":` + fmt.Sprint(tc.acceptance),
				`"next":` + string(nextJSON),
			} {
				if !strings.Contains(jsonBody, want) {
					t.Errorf("JSON projection missing %q: %s", want, jsonBody)
				}
			}
			if tc.why != "" && !strings.Contains(jsonBody, `"why_unavailable":"`+tc.why+`"`) {
				t.Errorf("JSON projection missing refusal reason: %s", jsonBody)
			}
			if !tc.acceptance && strings.Contains(jsonBody, `"acceptance_command"`) {
				t.Errorf("unavailable projection exposed acceptance_command: %s", jsonBody)
			}

			for _, width := range []int{32, 47, 48, 63, 64, 80} {
				visual := renderPlanningCandidateVisual(tc.review, planningVisualOptions{Width: width})
				for _, want := range append([]string{
					"Candidate: " + tc.review.Candidate.ID,
					"Candidate hash: " + tc.review.Candidate.ContentHash,
					"SPEC hash: " + tc.review.Candidate.SpecificationRevisionHash,
					"Proposal hash: " + tc.review.Candidate.ProposalHash,
					"Standing: " + string(tc.standing),
					"Expires: " + tc.review.Candidate.ExpiresAt.UTC().Format(time.RFC3339Nano),
					"Active plan: unchanged",
				}, tc.required...) {
					if !strings.Contains(visual, want) {
						t.Errorf("width %d missing exact %q:\n%s", width, want, visual)
					}
				}
				for _, forbidden := range tc.forbidden {
					if strings.Contains(visual, forbidden) {
						t.Errorf("width %d exposed forbidden %q:\n%s", width, forbidden, visual)
					}
				}
			}
			for _, command := range append(append([]string(nil), projection.ExecutionActions...), projection.Next) {
				for _, candidateCommand := range strings.Split(command, " | ") {
					if candidateCommand == "" {
						continue
					}
					if _, ok := availableCommand(candidateCommand); !ok {
						t.Errorf("printed command does not resolve against live Cobra tree: %q", candidateCommand)
					}
				}
			}
		})
	}

	for _, reason := range []string{
		"specification_changed", "base_plan_changed", "proposal_changed", "timeline_changed",
		"candidate_body_changed", "planning_stage_changed", "clock_before_candidate_creation",
	} {
		t.Run("early stale "+reason, func(t *testing.T) {
			assessment := planCandidateStandingAssessment{
				Standing: planCandidateStandingStale, WhyUnavailable: reason,
				Evidence: []string{"binding=" + reason}, AcceptanceAvailable: false,
				StateEffect: planCandidateStateEffectUnchanged, ActivePlanEffect: planCandidateActivePlanEffectUnchanged,
				RecoveryCommand: planCandidateRefreshCommand,
			}
			review := planningExpiryReview200(current, candidate, assessment)
			projection := projectPlanningCandidate(review)
			if projection.WhyUnavailable != reason || projection.Next != planCandidateRefreshCommand || projection.AcceptanceCommand != "" {
				t.Fatalf("stale reason %s projected unsafely: %+v", reason, projection)
			}
		})
	}

	t.Run("color and no color preserve identical semantics", func(t *testing.T) {
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		t.Setenv("NO_COLOR", "")
		t.Setenv("AETHER_FORCE_COLOR", "1")
		colored := renderPlanningCandidateVisual(expired, planningVisualOptions{Width: 64})
		t.Setenv("NO_COLOR", "1")
		t.Setenv("AETHER_FORCE_COLOR", "")
		plain := renderPlanningCandidateVisual(expired, planningVisualOptions{Width: 64})
		if stripped := planningStripANSI(colored); stripped != plain {
			t.Fatalf("color changed candidate semantics\ncolor-stripped:\n%s\nplain:\n%s", stripped, plain)
		}
	})

	t.Run("at_deadline_cobra_json_refusal", func(t *testing.T) {
		result := runPlanningExpiryCobraRefusal200(t, "json")
		var envelope struct {
			OK      bool                        `json:"ok"`
			Code    int                         `json:"code"`
			Details planCandidateRefusalDetails `json:"details"`
		}
		if err := json.Unmarshal([]byte(result.stderr), &envelope); err != nil {
			t.Fatalf("decode JSON refusal: %v\n%s", err, result.stderr)
		}
		if envelope.OK || envelope.Code != 1 {
			t.Fatalf("JSON refusal envelope = %+v", envelope)
		}
		assertPlanningExpiryRefusalDetails200(t, envelope.Details, result.candidate)
	})

	t.Run("at_deadline_cobra_visual_refusal", func(t *testing.T) {
		result := runPlanningExpiryCobraRefusal200(t, "visual")
		for _, want := range []string{
			"Candidate status: expired", "Standing: expired", "Expires: " + result.candidate.ExpiresAt.UTC().Format(time.RFC3339Nano),
			"Why unavailable: candidate_expired", "State: candidate marked expired", "Active plan: unchanged",
			"Next: " + planCandidateRefreshCommand,
		} {
			if !strings.Contains(result.stderr, want) {
				t.Errorf("visual refusal missing %q:\n%s", want, result.stderr)
			}
		}
		if strings.Contains(result.stderr, "--acceptance-token") || strings.TrimSpace(result.stdout) != "" {
			t.Fatalf("visual refusal leaked acceptance/success output\nstdout:\n%s\nstderr:\n%s", result.stdout, result.stderr)
		}
	})
}

func TestLifecycleFactsPlanningCandidateStanding200(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	factStore := planningExpiryFactStore200(t, root)
	currentAt := candidate.ExpiresAt.Add(-time.Nanosecond)
	before := planCandidateTestSnapshot(t, root)

	currentFacts, err := loadLifecycleFacts(root, factStore, currentAt)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleCandidateStanding200(t, currentFacts, candidate, planCandidateStandingCurrent, "", true, planCandidateStateEffectUnchanged, "")
	exactAcceptance := planCandidateAcceptanceCommand(planCandidateTestAcceptanceRequest(candidate))
	if currentFacts.Planning.Value.PendingCandidateAcceptanceCommand != exactAcceptance {
		t.Fatalf("current lifecycle acceptance = %q, want %q", currentFacts.Planning.Value.PendingCandidateAcceptanceCommand, exactAcceptance)
	}
	if after := planCandidateTestSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("current lifecycle fact load mutated the candidate repository")
	}

	deadlineFacts, err := loadLifecycleFacts(root, factStore, candidate.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleCandidateStanding200(t, deadlineFacts, candidate, planCandidateStandingExpired, "candidate_expired", false, planCandidateStateEffectUnchanged, planCandidateRefreshCommand)
	if deadlineFacts.Planning.Value.PendingCandidateStatus != colony.PlanCandidatePendingReview {
		t.Fatalf("read-only deadline load changed pending status: %+v", deadlineFacts.Planning.Value)
	}
	if after := planCandidateTestSnapshot(t, root); !reflect.DeepEqual(before, after) {
		t.Fatal("deadline lifecycle fact load mutated the candidate repository")
	}

	if _, acceptErr := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:lifecycle-facts-200", AcceptedAt: candidate.ExpiresAt,
	}); acceptErr == nil {
		t.Fatal("at-deadline setup unexpectedly accepted candidate")
	}
	afterExpiry := planCandidateTestSnapshot(t, root)
	persistedFacts, err := loadLifecycleFacts(root, factStore, candidate.ExpiresAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleCandidateStanding200(t, persistedFacts, candidate, planCandidateStandingExpired, "candidate_expired", false, planCandidateStateEffectUnchanged, planCandidateRefreshCommand)
	if persistedFacts.Planning.Value.PendingCandidateID != "" {
		t.Fatalf("persisted expiry invented a pending candidate: %+v", persistedFacts.Planning.Value)
	}
	if after := planCandidateTestSnapshot(t, root); !reflect.DeepEqual(afterExpiry, after) {
		t.Fatal("later expired lifecycle review performed another mutation")
	}

	staleRoot, staleCandidate := planCandidateTestPending(t)
	staleStore := planningExpiryFactStore200(t, staleRoot)
	staleState, err := loadSpecificationColonyState(staleRoot)
	if err != nil {
		t.Fatal(err)
	}
	if staleState.Specification == nil {
		t.Fatal("stale lifecycle fixture has no specification")
	}
	currentRevision, ok := currentSpecificationRevision(*staleState.Specification)
	if !ok {
		t.Fatal("stale lifecycle fixture has no current specification revision")
	}
	for index := range staleState.Specification.Revisions {
		if staleState.Specification.Revisions[index].ID == currentRevision.ID {
			staleState.Specification.Revisions[index].ContentHash = strings.Repeat("0", 64)
		}
	}
	staleBytes, err := json.Marshal(staleState)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staleStore.BasePath(), "COLONY_STATE.json"), staleBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	staleFacts, err := loadLifecycleFacts(staleRoot, staleStore, staleCandidate.ExpiresAt.Add(-time.Nanosecond))
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleCandidateStanding200(t, staleFacts, staleCandidate, planCandidateStandingStale, "specification_changed", false, planCandidateStateEffectUnchanged, planCandidateRefreshCommand)

	acceptedRoot, acceptedCandidate := planCandidateTestPending(t)
	acceptedAt := acceptedCandidate.ExpiresAt.Add(-time.Hour)
	acceptedResult, err := acceptPlanCandidate(acceptedRoot, planCandidateTestAcceptanceRequest(acceptedCandidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:lifecycle-facts-200", AcceptedAt: acceptedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	acceptedStore := planningExpiryFactStore200(t, acceptedRoot)
	acceptedBefore := planCandidateTestSnapshot(t, acceptedRoot)
	acceptedFacts, err := loadLifecycleFacts(acceptedRoot, acceptedStore, acceptedCandidate.ExpiresAt.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	assertLifecycleCandidateStanding200(t, acceptedFacts, acceptedResult.Candidate, planCandidateStandingAccepted, "", false, planCandidateStateEffectUnchanged, "")
	if after := planCandidateTestSnapshot(t, acceptedRoot); !reflect.DeepEqual(acceptedBefore, after) {
		t.Fatal("accepted lifecycle fact load mutated the candidate repository")
	}
}

func TestPlanningExpiryNextAction200(t *testing.T) {
	root, candidate := planCandidateTestPending(t)
	factStore := planningExpiryFactStore200(t, root)
	currentAt := candidate.ExpiresAt.Add(-time.Nanosecond)
	facts, err := loadLifecycleFacts(root, factStore, currentAt)
	if err != nil {
		t.Fatal(err)
	}
	review, err := reviewPlanCandidateAt(root, currentAt)
	if err != nil {
		t.Fatal(err)
	}
	wantAcceptance := projectPlanningCandidate(review).Next
	if wantAcceptance == "" || wantAcceptance != facts.Planning.Value.PendingCandidateAcceptanceCommand {
		t.Fatalf("review and lifecycle facts disagree on acceptance: review=%q facts=%q", wantAcceptance, facts.Planning.Value.PendingCandidateAcceptanceCommand)
	}

	currentBefore := facts
	current := resolveNextAction(nextActionInput{Facts: facts})
	if !reflect.DeepEqual(currentBefore, facts) {
		t.Fatal("current Next Up mutated lifecycle facts")
	}
	assertPlanningExpiryNextAction200(t, current, "aether plan --candidate", []string{wantAcceptance}, []string{"aether build", "aether run", planCandidateRefreshCommand})

	staleFacts := facts
	staleFacts.Planning.Value.PendingCandidateStanding = planCandidateStandingStale
	staleFacts.Planning.Value.PendingCandidateWhyUnavailable = "specification_changed"
	staleFacts.Planning.Value.PendingCandidateAcceptanceAvailable = false
	staleFacts.Planning.Value.PendingCandidateAcceptanceCommand = ""
	staleFacts.Planning.Value.PendingCandidateRecoveryCommand = planCandidateRefreshCommand
	stale := resolveNextAction(nextActionInput{Facts: staleFacts})
	assertPlanningExpiryNextAction200(t, stale, planCandidateRefreshCommand, nil, []string{"--accept-candidate", "aether build", "aether run"})

	expiredFacts, err := loadLifecycleFacts(root, factStore, candidate.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	expiredBefore := expiredFacts
	expired := resolveNextAction(nextActionInput{Facts: expiredFacts})
	if !reflect.DeepEqual(expiredBefore, expiredFacts) {
		t.Fatal("expired Next Up mutated lifecycle facts")
	}
	assertPlanningExpiryNextAction200(t, expired, planCandidateRefreshCommand, nil, []string{"--accept-candidate", "aether build", "aether run"})

	acceptedRoot, acceptedCandidate := planCandidateTestPending(t)
	acceptedAt := acceptedCandidate.ExpiresAt.Add(-time.Nanosecond)
	if _, err := acceptPlanCandidate(acceptedRoot, planCandidateTestAcceptanceRequest(acceptedCandidate), planCandidateAcceptanceOptions{
		AcceptedBy: "owner:next-action-200", AcceptedAt: acceptedAt,
	}); err != nil {
		t.Fatal(err)
	}
	acceptedFacts, err := loadLifecycleFacts(acceptedRoot, planningExpiryFactStore200(t, acceptedRoot), acceptedAt)
	if err != nil {
		t.Fatal(err)
	}
	accepted := resolveNextAction(nextActionInput{Facts: acceptedFacts})
	assertPlanningExpiryNextAction200(t, accepted, "", []string{"aether build 1", "aether run"}, []string{"--accept-candidate", planCandidateRefreshCommand})
}

func planningExpiryFactStore200(t *testing.T, root string) *storage.Store {
	t.Helper()
	factStore, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatal(err)
	}
	return factStore
}

func assertLifecycleCandidateStanding200(t *testing.T, facts LifecycleFacts, candidate colony.PlanCandidate, standing planCandidateStanding, why string, acceptance bool, stateEffect planCandidateStateEffect, recovery string) {
	t.Helper()
	planning := facts.Planning.Value
	if !facts.CapturedAt.Equal(facts.Timing.Value.CapturedAt) ||
		planning.PendingCandidateStanding != standing || !planning.PendingCandidateExpiresAt.Equal(candidate.ExpiresAt.UTC()) ||
		planning.PendingCandidateWhyUnavailable != why || planning.PendingCandidateAcceptanceAvailable != acceptance ||
		planning.PendingCandidateStateEffect != stateEffect || planning.PendingCandidateActivePlanEffect != planCandidateActivePlanEffectUnchanged ||
		planning.PendingCandidateRecoveryCommand != recovery {
		t.Fatalf("lifecycle candidate standing = %+v at %s", planning, facts.CapturedAt)
	}
	if why != "" && len(planning.PendingCandidateEvidence) == 0 {
		t.Fatalf("lifecycle candidate refusal omitted evidence: %+v", planning)
	}
	encoded, err := json.Marshal(planning)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"pending_candidate_standing", "pending_candidate_expires_at", "pending_candidate_state_effect", "pending_candidate_active_plan_effect", "pending_candidate_acceptance_available"} {
		if !bytes.Contains(encoded, []byte(`"`+key+`"`)) {
			t.Errorf("lifecycle JSON omitted %s: %s", key, encoded)
		}
	}
}

func assertPlanningExpiryNextAction200(t *testing.T, action nextAction, primary string, wanted, forbidden []string) {
	t.Helper()
	if action.Projection == nil {
		t.Fatal("Next Up omitted lifecycle projection")
	}
	commands := []string{action.Command}
	for _, choice := range action.Projection.NextAction.Choices {
		commands = append(commands, choice.RuntimeCommand)
	}
	for _, alternative := range action.Alternatives {
		commands = append(commands, alternative.Command)
	}
	if action.Command != primary {
		t.Fatalf("Next Up primary = %q, want %q; all=%v", action.Command, primary, commands)
	}
	for _, want := range wanted {
		if !containsString(commands, want) {
			t.Errorf("Next Up commands %v omit %q", commands, want)
		}
		if _, ok := availableCommand(want); !ok {
			t.Errorf("Next Up command does not resolve against live Cobra tree: %q", want)
		}
	}
	encoded, err := json.Marshal(action)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range forbidden {
		if bytes.Contains(encoded, []byte(value)) {
			t.Errorf("Next Up recursively exposed forbidden %q: %s", value, encoded)
		}
	}
}

func planningExpiryReview200(base planCandidateReview, candidate colony.PlanCandidate, assessment planCandidateStandingAssessment) planCandidateReview {
	review := base
	review.Candidate = candidate
	review.Standing = assessment.Standing
	review.Acceptance = planCandidateAcceptanceRequest{}
	review.AcceptanceCommand = ""
	refusal := planCandidateRefusal(candidate, assessment)
	review.Refusal = &refusal
	return review
}

type planningExpiryCobraResult200 struct {
	stdout    string
	stderr    string
	candidate colony.PlanCandidate
}

func runPlanningExpiryCobraRefusal200(t *testing.T, mode string) planningExpiryCobraResult200 {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	root, candidate := planCandidateTestPending(t)
	withWorkingDir(t, root)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	t.Setenv("NO_COLOR", "1")

	previousClock := planCandidateNow
	planCandidateNow = func() time.Time { return candidate.ExpiresAt }
	t.Cleanup(func() { planCandidateNow = previousClock })

	before, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	beforeAuthority := planningExpiryActivePlanBytes200(t, before)
	request := planCandidateTestAcceptanceRequest(candidate)
	args := []string{
		"plan", "--accept-candidate", request.CandidateID,
		"--spec-revision", request.SpecificationRevisionID,
		"--spec-hash", request.SpecificationRevisionHash,
		"--base-plan-revision", request.BasePlanRevisionID,
		"--timeline-digest", request.TimelineDigest,
		"--proposal-hash", request.ProposalHash,
		"--acceptance-token", request.AcceptanceToken,
	}
	var out, errOut bytes.Buffer
	stdout, stderr = &out, &errOut
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs(args)
	commandErr := Execute()
	var renderedErr renderedCommandError
	if !errors.As(commandErr, &renderedErr) || renderedErr.code != 1 {
		t.Fatalf("Cobra expiry error = %#v, want rendered exit 1\nstdout:\n%s\nstderr:\n%s", commandErr, out.String(), errOut.String())
	}

	after, err := loadSpecificationColonyState(root)
	if err != nil {
		t.Fatal(err)
	}
	if afterAuthority := planningExpiryActivePlanBytes200(t, after); !bytes.Equal(beforeAuthority, afterAuthority) {
		t.Fatalf("expiry refusal changed active plan bytes\nbefore: %s\nafter: %s", beforeAuthority, afterAuthority)
	}
	persisted := planCandidateExpiry200ReadCandidate(t, root, candidate)
	if persisted.Status != colony.PlanCandidateExpired || persisted.Acceptance != nil {
		t.Fatalf("persisted candidate = %+v, want expired without acceptance", persisted)
	}

	return planningExpiryCobraResult200{stdout: out.String(), stderr: errOut.String(), candidate: persisted}
}

func planningExpiryActivePlanBytes200(t *testing.T, state colony.ColonyState) []byte {
	t.Helper()
	value := struct {
		ActiveRevisionID string                `json:"active_revision_id"`
		Revisions        []colony.PlanRevision `json:"revisions"`
		Phases           []colony.Phase        `json:"phases"`
	}{
		ActiveRevisionID: state.Plan.ActiveRevisionID,
		Revisions:        state.Plan.Revisions,
		Phases:           state.Plan.Phases,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func assertPlanningExpiryRefusalDetails200(t *testing.T, got planCandidateRefusalDetails, candidate colony.PlanCandidate) {
	t.Helper()
	if got.CandidateID != candidate.ID || got.CandidateStatus != colony.PlanCandidateExpired ||
		got.Standing != planCandidateStandingExpired || !got.ExpiresAt.Equal(candidate.ExpiresAt) ||
		got.WhyUnavailable != "candidate_expired" || len(got.Evidence) == 0 ||
		got.StateEffect != planCandidateStateEffectMarkedExpired ||
		got.ActivePlanEffect != planCandidateActivePlanEffectUnchanged || got.RecoveryCommand != planCandidateRefreshCommand {
		t.Fatalf("typed refusal = %+v", got)
	}
}
