package cmd

import (
	"fmt"
	"strings"
)

// printPlanningBriefs answers "will the research I pointed at actually reach
// the planners?" before a plan run is paid for.
//
// It deliberately does not require an existing plan, unlike the build
// equivalent: the case that matters most is a fresh colony's first plan, which
// is exactly when someone hands over research and has no phases yet.
// It reads state and writes nothing.
func printPlanningBriefs(root string, full bool) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}
	state, err := loadActiveColonyState()
	if err != nil {
		return fmt.Errorf("%s", colonyStateLoadMessage(err))
	}

	survey, _ := loadCodexSurveyContext(root)

	var b strings.Builder
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n  PLANNING CONTEXT CHECKLIST\n")
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n")
	fmt.Fprintf(&b, "  Goal: %s\n", truncateString(strings.TrimSpace(ptrStr(state.Goal)), 68))
	fmt.Fprintf(&b, "  Phases so far: %d\n\n", len(state.Plan.Phases))

	for _, spec := range planningWorkerSpecs {
		brief := renderPlanningWorkerBrief(root, survey, spec)
		fmt.Fprintf(&b, "  %s\n", strings.ToUpper(spec.Caste))
		for _, row := range []briefChecklistRow{
			checklistRowFor(brief, "Territory Survey", "- Primary survey source:"),
			checklistRowFor(brief, "Colony Research", "## Colony Research"),
		} {
			status := "ABSENT"
			if row.Present {
				status = "present"
			}
			fmt.Fprintf(&b, "    %-20s %-8s %6d\n", row.Label, status, row.Chars)
		}
		fmt.Fprintf(&b, "    %-20s %-8s %6d\n", "Total brief", "—", len(brief))
		b.WriteString("\n")
		if full {
			b.WriteString(brief)
			b.WriteString("\n\n")
		}
	}

	if len(state.ResearchDocs) == 0 {
		b.WriteString("  No research attached. Point this colony at a saved research document with:\n")
		b.WriteString("    aether plan --research <path>       (see `aether research list`)\n")
	} else {
		fmt.Fprintf(&b, "  Research attached: %s\n", strings.Join(state.ResearchDocs, ", "))
	}

	writeVisualOutput(stdout, b.String())
	return nil
}
