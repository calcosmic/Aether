package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/codegraph"
)

// Post-init proposals — the runtime computes ranked next moves from what the
// repo actually contains, and the wrapper asks the user to choose. Init used
// to end with the SAME static three lines for every repo on earth while
// init-research had already computed every discriminator needed; the data
// was gathered pre-init and thrown away. This recomputes a cheap subset at
// the moment of proposing (milliseconds, honest, no stale JSON crossing the
// wrapper boundary).

type initProposal struct {
	Command string `json:"command"`
	Label   string `json:"label"`
	Reason  string `json:"reason"`
}

// initProposalSourceFileThreshold is how many source files make a repo
// "existing code worth mapping first". Deliberately low: even a small real
// codebase benefits from colonize before planning.
const initProposalSourceFileThreshold = 10

// initProposalWalkCap bounds the discriminator walk — proposals need "is
// there real code here", not an exact census.
const initProposalWalkCap = 500

// computeInitProposals ranks the sensible next moves after init:
// colonize-first for existing code, discuss-first for broad goals, plan
// otherwise. Always returns all three, ranked, each with a plain-English
// reason. Any error degrades to the generic ranking — a proposal must never
// fail an init.
func computeInitProposals(repoRoot, goal string, priorColony bool) []initProposal {
	detected := ""
	sourceCount := 0
	fileCount := 0

	if entries, err := os.ReadDir(repoRoot); err == nil {
		entryNames := make(map[string]bool)
		for _, e := range entries {
			if !e.IsDir() {
				entryNames[e.Name()] = true
			}
		}
		for _, det := range projectDetectors {
			if entryNames[det.file] && detected == "" {
				detected = det.typ
			}
		}
		_ = filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if codegraph.ShouldSkipDir(d.Name()) && path != repoRoot {
					return filepath.SkipDir
				}
				return nil
			}
			fileCount++
			if sourceFileExtensions[strings.ToLower(filepath.Ext(d.Name()))] {
				sourceCount++
			}
			if fileCount >= initProposalWalkCap {
				return filepath.SkipAll
			}
			return nil
		})
	}

	colonize := initProposal{
		Command: "aether colonize",
		Label:   "Map the codebase first",
		Reason:  "Surveys the existing code so the plan is grounded in what is actually here.",
	}
	discuss := initProposal{
		Command: "aether discuss",
		Label:   "Clarify the goal first",
		Reason:  "A few clarifying choices now make the first plan sharper.",
	}
	plan := initProposal{
		Command: "aether plan",
		Label:   "Straight to planning",
		Reason:  "The goal is clear and the folder is fresh — generate the phase map now.",
	}

	hasExistingCode := sourceCount >= initProposalSourceFileThreshold || detected != ""
	goalIsBroad := len(strings.Fields(goal)) < 5

	var ranked []initProposal
	switch {
	case hasExistingCode:
		what := "source files"
		if detected != "" {
			what = detected + " code"
		}
		countNote := fmt.Sprintf("~%d source files", sourceCount)
		if fileCount >= initProposalWalkCap {
			countNote = fmt.Sprintf("%d+ files", initProposalWalkCap)
		}
		colonize.Reason = fmt.Sprintf("This folder already holds %s (%s) — mapping it first gives the colony a real picture before planning. (recommended)", what, countNote)
		ranked = []initProposal{colonize, discuss, plan}
	case goalIsBroad:
		discuss.Reason = "The goal is broad — a few clarifying choices now will make the plan much sharper. (recommended)"
		ranked = []initProposal{discuss, plan, colonize}
	default:
		plan.Reason = "The goal is clear and the folder is fresh — go straight to the phase map. (recommended)"
		ranked = []initProposal{plan, discuss, colonize}
	}

	if priorColony {
		ranked[0].Reason += " A previous colony's record was backed up during init — its wisdom (instincts, signals) carries forward automatically."
	}
	return ranked
}

// renderInitProposals is the consent-framed Next Moves section: ranked
// options with reasons, nothing runs until the user chooses.
func renderInitProposals(proposals []initProposal) string {
	if len(proposals) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n🧭 Next Moves — nothing runs until you choose:\n")
	for i, p := range proposals {
		fmt.Fprintf(&b, "  %d. %s — `%s`\n", i+1, p.Label, p.Command)
		fmt.Fprintf(&b, "     └── %s\n", p.Reason)
	}
	return b.String()
}
