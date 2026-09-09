package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
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
			for _, want := range []string{
				`"standing":"` + string(tc.standing) + `"`,
				`"expires_at":"` + tc.review.Candidate.ExpiresAt.UTC().Format(time.RFC3339Nano) + `"`,
				`"state_effect":"` + string(tc.stateEffect) + `"`,
				`"active_plan_effect":"unchanged"`,
				`"acceptance_available":` + fmt.Sprint(tc.acceptance),
				`"next":"` + tc.next + `"`,
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
