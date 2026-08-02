package cmd

import (
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestDepthProposalCardKnobs(t *testing.T) {
	state := colony.ColonyState{Goal: strPtr("Add a login form")}
	p := computeDepthProposal(state, colony.GranularityMilestone, "standard", "heavy", false, false)

	if len(p.Knobs) != 3 {
		t.Fatalf("expected 3 knobs, got %d", len(p.Knobs))
	}

	wantKeys := []string{"granularity", "planning_depth", "verification_depth"}
	for i, want := range wantKeys {
		if p.Knobs[i].Key != want {
			t.Errorf("knob %d: expected key %q, got %q", i, want, p.Knobs[i].Key)
		}
	}

	granKnob := p.Knobs[0]
	if len(granKnob.Options) != 4 {
		t.Fatalf("expected 4 granularity options, got %d", len(granKnob.Options))
	}
	wantValues := []colony.PlanGranularity{
		colony.GranularitySprint, colony.GranularityMilestone,
		colony.GranularityQuarter, colony.GranularityMajor,
	}
	recommendedCount := 0
	for i, opt := range granKnob.Options {
		if opt.Value != string(wantValues[i]) {
			t.Errorf("granularity option %d: expected value %q, got %q", i, wantValues[i], opt.Value)
		}
		min, max := colony.GranularityRange(wantValues[i])
		wantLabelFragment := "-"
		if !strings.Contains(opt.Label, wantLabelFragment) {
			t.Errorf("granularity option %d label %q missing range separator", i, opt.Label)
		}
		if !strings.Contains(opt.Label, strconv.Itoa(min)) || !strings.Contains(opt.Label, strconv.Itoa(max)) {
			t.Errorf("granularity option %d label %q missing range %d-%d", i, opt.Label, min, max)
		}
		if opt.Recommended {
			recommendedCount++
			if opt.Value != string(colony.GranularityMilestone) {
				t.Errorf("expected milestone to be recommended, got %q", opt.Value)
			}
		}
	}
	if recommendedCount != 1 {
		t.Errorf("expected exactly 1 recommended granularity option, got %d", recommendedCount)
	}
	if granKnob.Recommended != string(colony.GranularityMilestone) {
		t.Errorf("expected knob.Recommended = milestone, got %q", granKnob.Recommended)
	}
	if granKnob.Reason == "" {
		t.Error("expected non-empty granularity reason")
	}

	planKnob := p.Knobs[1]
	if len(planKnob.Options) != 3 {
		t.Fatalf("expected 3 planning_depth options, got %d", len(planKnob.Options))
	}
	wantPlanValues := []string{"light", "standard", "deep"}
	planRecommended := 0
	for i, opt := range planKnob.Options {
		if opt.Value != wantPlanValues[i] {
			t.Errorf("planning_depth option %d: expected %q, got %q", i, wantPlanValues[i], opt.Value)
		}
		if opt.Recommended {
			planRecommended++
			if opt.Value != "standard" {
				t.Errorf("expected standard to be recommended, got %q", opt.Value)
			}
		}
	}
	if planRecommended != 1 {
		t.Errorf("expected exactly 1 recommended planning_depth option, got %d", planRecommended)
	}
	if planKnob.Reason == "" {
		t.Error("expected non-empty planning_depth reason")
	}

	verifyKnob := p.Knobs[2]
	if len(verifyKnob.Options) != 3 {
		t.Fatalf("expected 3 verification_depth options, got %d", len(verifyKnob.Options))
	}
	wantVerifyValues := []string{"light", "standard", "heavy"}
	verifyRecommended := 0
	for i, opt := range verifyKnob.Options {
		if opt.Value != wantVerifyValues[i] {
			t.Errorf("verification_depth option %d: expected %q, got %q", i, wantVerifyValues[i], opt.Value)
		}
		if opt.Recommended {
			verifyRecommended++
			if opt.Value != "heavy" {
				t.Errorf("expected heavy to be recommended, got %q", opt.Value)
			}
		}
	}
	if verifyRecommended != 1 {
		t.Errorf("expected exactly 1 recommended verification_depth option, got %d", verifyRecommended)
	}
	if verifyKnob.Reason == "" {
		t.Error("expected non-empty verification_depth reason")
	}
}

func TestDepthProposalReasons(t *testing.T) {
	shortGoalState := colony.ColonyState{Goal: strPtr("Fix the login bug")}
	longGoalState := colony.ColonyState{Goal: strPtr(strings.Repeat("subsystem ", 20) + "goal spanning several distinct areas of the codebase all at once")}

	shortReason := renderGranularityReason(shortGoalState, colony.GranularitySprint)
	longReason := renderGranularityReason(longGoalState, colony.GranularityQuarter)

	if shortReason == "" || longReason == "" {
		t.Fatal("expected non-empty reasons for both goals")
	}
	if shortReason == longReason {
		t.Errorf("expected short and long goal reasons to differ, both were %q", shortReason)
	}

	existingPhasesState := colony.ColonyState{
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}, {ID: 6}, {ID: 7}},
		},
	}
	existingReason := renderGranularityReason(existingPhasesState, colony.GranularityMilestone)
	if !strings.Contains(existingReason, "7") {
		t.Errorf("expected reason to name existing phase count 7, got %q", existingReason)
	}

	// Smart-default vs explicit: computeDepthProposal must route through
	// renderSmartDepthReason when smartDefault is true, and use an
	// "explicit" reason string when the user supplied the flag.
	state := colony.ColonyState{Goal: strPtr("goal"), Plan: colony.Plan{Phases: []colony.Phase{{ID: 1}}}}
	smartProposal := computeDepthProposal(state, colony.GranularitySprint, "standard", "standard", true, true)
	explicitProposal := computeDepthProposal(state, colony.GranularitySprint, "standard", "standard", false, false)

	if smartProposal.Knobs[1].Reason == explicitProposal.Knobs[1].Reason {
		t.Error("expected planning_depth reason to differ between smart-default and explicit selection")
	}
	if smartProposal.Knobs[2].Reason == explicitProposal.Knobs[2].Reason {
		t.Error("expected verification_depth reason to differ between smart-default and explicit selection")
	}
	if !strings.Contains(explicitProposal.Knobs[1].Reason, "explicitly") {
		t.Errorf("expected explicit planning_depth reason to mention explicit selection, got %q", explicitProposal.Knobs[1].Reason)
	}
	if !strings.Contains(explicitProposal.Knobs[2].Reason, "explicitly") {
		t.Errorf("expected explicit verification_depth reason to mention explicit selection, got %q", explicitProposal.Knobs[2].Reason)
	}
}

func TestDepthProposalCardIsSelectionOnly(t *testing.T) {
	state := colony.ColonyState{Goal: strPtr("Add a login form")}
	p := computeDepthProposal(state, colony.GranularityMilestone, "standard", "heavy", false, false)
	card := renderDepthProposalCard(p)

	if card == "" {
		t.Fatal("expected non-empty card for a three-knob proposal")
	}

	acceptCount := strings.Count(card, "Accept all:")
	if acceptCount != 1 {
		t.Errorf("expected exactly 1 'Accept all:' line, got %d", acceptCount)
	}
	changeCount := strings.Count(card, "Change one:")
	if changeCount != 1 {
		t.Errorf("expected exactly 1 'Change one:' line, got %d", changeCount)
	}

	var acceptLine string
	for _, line := range strings.Split(card, "\n") {
		if strings.Contains(line, "Accept all:") {
			acceptLine = line
		}
	}
	if acceptLine == "" {
		t.Fatal("could not find Accept all: line")
	}
	for _, flag := range []string{"--depth", "--planning-depth", "--verification-depth"} {
		if !strings.Contains(acceptLine, flag) {
			t.Errorf("expected accept line to contain %q, got %q", flag, acceptLine)
		}
	}
	if !strings.Contains(acceptLine, "milestone") {
		t.Errorf("expected accept line to contain concrete granularity value 'milestone', got %q", acceptLine)
	}
	if !strings.Contains(acceptLine, "standard") {
		t.Errorf("expected accept line to contain concrete planning-depth value 'standard', got %q", acceptLine)
	}
	if !strings.Contains(acceptLine, "heavy") {
		t.Errorf("expected accept line to contain concrete verification-depth value 'heavy', got %q", acceptLine)
	}

	reasonCount := strings.Count(card, "Reason:")
	if reasonCount != 3 {
		t.Errorf("expected exactly 3 'Reason:' lines for a three-knob proposal, got %d", reasonCount)
	}

	markedCount := strings.Count(card, "> ")
	if markedCount != len(p.Knobs) {
		t.Errorf("expected %d marked-recommended lines, got %d", len(p.Knobs), markedCount)
	}

	lowerCard := strings.ToLower(card)
	for _, forbidden := range []string{"enter a value", "type a", "describe", "free text"} {
		if strings.Contains(lowerCard, forbidden) {
			t.Errorf("card must not contain free-text prompt language, found %q", forbidden)
		}
	}

	empty := renderDepthProposalCard(depthProposal{})
	if empty != "" {
		t.Errorf("expected empty proposal to render as empty string, got %q", empty)
	}
}
