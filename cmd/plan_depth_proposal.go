package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// depthKnobOption is a single selectable value within a depthKnob.
type depthKnobOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Recommended bool   `json:"recommended"`
}

// depthKnob is one of the three depth controls (granularity, planning
// depth, verification depth) presented together in a single proposal card.
type depthKnob struct {
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Options     []depthKnobOption `json:"options"`
	Recommended string            `json:"recommended"`
	Reason      string            `json:"reason"`
}

// depthProposal bundles all three depth knobs the Queen proposes together
// at plan time (RESEARCH-09, RESEARCH-10) instead of asking twice, cold,
// with no reasoning and no verification-depth option at all.
type depthProposal struct {
	Knobs []depthKnob `json:"knobs"`
}

// granularityOrder is the fixed display order for granularity options,
// coarsest phase count first.
var granularityOrder = []colony.PlanGranularity{
	colony.GranularitySprint,
	colony.GranularityMilestone,
	colony.GranularityQuarter,
	colony.GranularityMajor,
}

// granularityReasons is the lookup table for renderGranularityReason,
// following the getSmartDefaultReason lookup-table shape from
// cmd/review_depth.go. "existing_route" is a format string; the rest are
// used verbatim.
var granularityReasons = map[string]string{
	"existing_route":  "matching the %d phases already in the route",
	"narrow_goal":     "goal is a single focused change",
	"multi_subsystem": "goal spans several subsystems",
	"broad_goal":      "goal covers a broad body of work",
	"program_goal":    "goal describes a program-scale body of work",
}

// renderGranularityReason produces a plain-English reason for the
// recommended granularity. When phases already exist in the plan, the
// reason names the current phase count; otherwise it buckets the goal's
// word count. The goal text itself is only measured, never interpolated
// (T-164-10).
func renderGranularityReason(state colony.ColonyState, g colony.PlanGranularity) string {
	if len(state.Plan.Phases) > 0 {
		return fmt.Sprintf(granularityReasons["existing_route"], len(state.Plan.Phases))
	}

	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	words := len(strings.Fields(goal))

	switch {
	case words <= 12:
		return granularityReasons["narrow_goal"]
	case words <= 30:
		return granularityReasons["multi_subsystem"]
	case words <= 60:
		return granularityReasons["broad_goal"]
	default:
		return granularityReasons["program_goal"]
	}
}

// depthProposalGranularityLabel formats one granularity option's label
// using colony.GranularityRange as the single source of truth for
// phase-count ranges — never hard-code the ranges here.
func depthProposalGranularityLabel(g colony.PlanGranularity) string {
	min, max := colony.GranularityRange(g)
	return fmt.Sprintf("%s — %d-%d phases", g, min, max)
}

func buildGranularityKnob(state colony.ColonyState, granularity colony.PlanGranularity) depthKnob {
	options := make([]depthKnobOption, 0, len(granularityOrder))
	for _, g := range granularityOrder {
		options = append(options, depthKnobOption{
			Value:       string(g),
			Label:       depthProposalGranularityLabel(g),
			Recommended: g == granularity,
		})
	}
	return depthKnob{
		Key:         "granularity",
		Title:       "Granularity — how many phases the plan should have",
		Options:     options,
		Recommended: string(granularity),
		Reason:      renderGranularityReason(state, granularity),
	}
}

// smartOrExplicitReason returns the smart-default reason from
// renderSmartDepthReason when smartDefault is true, otherwise a reason
// stating the value was chosen explicitly on the command line.
func smartOrExplicitReason(state colony.ColonyState, smartDefault bool) string {
	if smartDefault {
		return renderSmartDepthReason(colony.Phase{ID: 1}, len(state.Plan.Phases))
	}
	return "selected explicitly on the command line"
}

func buildPlanningDepthKnob(state colony.ColonyState, planningDepth string, smartDefault bool) depthKnob {
	options := []depthKnobOption{
		{Value: "light", Label: "light — coarse tasks, 1-3 per plan", Recommended: planningDepth == "light"},
		{Value: "standard", Label: "standard — normal task breakdown", Recommended: planningDepth == "standard"},
		{Value: "deep", Label: "deep — granular subtasks with edge cases and test coverage", Recommended: planningDepth == "deep"},
	}
	return depthKnob{
		Key:         "planning_depth",
		Title:       "Task decomposition depth — how granular each plan's tasks are",
		Options:     options,
		Recommended: planningDepth,
		Reason:      smartOrExplicitReason(state, smartDefault),
	}
}

func buildVerificationDepthKnob(state colony.ColonyState, verificationDepth string, smartDefault bool) depthKnob {
	options := []depthKnobOption{
		{Value: "light", Label: "light — Watcher only", Recommended: verificationDepth == "light"},
		{Value: "standard", Label: "standard — Watcher plus Probe", Recommended: verificationDepth == "standard"},
		{Value: "heavy", Label: "heavy — adds Gatekeeper and Auditor", Recommended: verificationDepth == "heavy"},
	}
	return depthKnob{
		Key:         "verification_depth",
		Title:       "Verification depth — how much safety review each phase gets",
		Options:     options,
		Recommended: verificationDepth,
		Reason:      smartOrExplicitReason(state, smartDefault),
	}
}

// computeDepthProposal computes all three existing depth controls --
// granularity, task decomposition (planning) depth, and verification
// depth -- together in a single proposal, each knob carrying a pre-marked
// recommendation and a plain-English reason (RESEARCH-09, RESEARCH-10).
func computeDepthProposal(state colony.ColonyState, granularity colony.PlanGranularity,
	planningDepth, verificationDepth string,
	planningSmartDefault, verificationSmartDefault bool) depthProposal {
	return depthProposal{
		Knobs: []depthKnob{
			buildGranularityKnob(state, granularity),
			buildPlanningDepthKnob(state, planningDepth, planningSmartDefault),
			buildVerificationDepthKnob(state, verificationDepth, verificationSmartDefault),
		},
	}
}

// depthProposalKnobRecommended returns the recommended value for the knob
// with the given key, or the empty string when no such knob exists.
func depthProposalKnobRecommended(p depthProposal, key string) string {
	for _, k := range p.Knobs {
		if k.Key == key {
			return k.Recommended
		}
	}
	return ""
}

// renderDepthProposalCard renders the Queen's three-knob depth proposal as
// a single selection-only card (D-13's hard constraint): every option is a
// numbered line, the recommended option is visibly marked, and the card
// closes with exactly one accept line (all three recommended values
// already filled in, zero typing required) and one change instruction
// phrased as selecting a knob and an option number. No free-text prompt
// ever appears in the card.
func renderDepthProposalCard(p depthProposal) string {
	if len(p.Knobs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("Queen's depth proposal — three knobs, one card:\n")
	for _, k := range p.Knobs {
		fmt.Fprintf(&b, "\n%s\n", k.Title)
		for i, opt := range k.Options {
			marker := "  "
			if opt.Recommended {
				marker = "> "
			}
			fmt.Fprintf(&b, "%s%d. %s\n", marker, i+1, opt.Label)
		}
		fmt.Fprintf(&b, "  Reason: %s\n", k.Reason)
	}

	granularity := depthProposalKnobRecommended(p, "granularity")
	planning := depthProposalKnobRecommended(p, "planning_depth")
	verification := depthProposalKnobRecommended(p, "verification_depth")

	fmt.Fprintf(&b, "\nAccept all: aether host plan --depth %s --planning-depth %s --verification-depth %s\n",
		granularity, planning, verification)
	b.WriteString("Change one: reply with the knob name and the option number (for example: verification-depth 3)\n")

	return b.String()
}
