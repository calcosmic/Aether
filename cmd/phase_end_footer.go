package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The classic end-of-phase footer, restored. The shell era (v2.0.0→v5.4.0)
// ended every phase with the colony's whole picture — open flags, active
// steering signals, where you are, whether it's safe to clear context, and
// the next command. The Go port kept only the last two (and degraded
// signals to a bare count). This module restores the missing sections in
// today's house style: headed emoji lines with `└──` detail — the classic
// bordered tables are forbidden by the SEE-12 display standard.

// renderPhaseEndFlagsSection shows the flag triage state at phase end:
// blockers (stop the line), issues (parkable), notes (parked). Flags that
// nobody can see at the moment decisions are made might as well not exist —
// the classic footer showed them at every phase boundary.
func renderPhaseEndFlagsSection() string {
	if store == nil {
		return ""
	}
	var ff colony.FlagsFile
	if err := store.LoadJSON("pending-decisions.json", &ff); err != nil {
		if err2 := store.LoadJSON("flags.json", &ff); err2 != nil {
			return "🚩 Flags: none open\n"
		}
	}

	groups := map[string][]colony.FlagEntry{}
	for _, flag := range ff.Decisions {
		if flag.Resolved {
			continue
		}
		flagType := strings.ToLower(strings.TrimSpace(flag.Type))
		switch flagType {
		case "blocker", "issue", "note":
		default:
			// Clarifications and other pending-decision types are not
			// flags; the discuss surface owns them.
			continue
		}
		groups[flagType] = append(groups[flagType], flag)
	}
	total := len(groups["blocker"]) + len(groups["issue"]) + len(groups["note"])
	if total == 0 {
		return "🚩 Flags: none open\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "🚩 Flags: %d blocker(s), %d issue(s), %d note(s)\n",
		len(groups["blocker"]), len(groups["issue"]), len(groups["note"]))
	const maxRowsPerGroup = 3
	for _, flagType := range []string{"blocker", "issue", "note"} {
		entries := groups[flagType]
		shown := entries
		if len(shown) > maxRowsPerGroup {
			shown = shown[:maxRowsPerGroup]
		}
		for _, flag := range shown {
			desc := strings.TrimSpace(flag.Description)
			if desc == "" {
				desc = flag.ID
			}
			marker := ""
			if flag.Acknowledged {
				marker = " [parked]"
			}
			fmt.Fprintf(&b, "   └── %s %s%s", flagTypeIcon(flagType), desc, marker)
			detail := flag.ID
			if flag.Source != "" {
				detail += ", from " + flag.Source
			}
			fmt.Fprintf(&b, " (%s)\n", detail)
		}
		if extra := len(entries) - len(shown); extra > 0 {
			fmt.Fprintf(&b, "   └── (+%d more %ss — /ant-flags)\n", extra, flagType)
		}
	}
	if len(groups["blocker"]) > 0 {
		b.WriteString("   └── blockers stop advancement: resolve with /ant-flags --resolve <id>, or /ant-unblock dispatches the Fixer\n")
	}
	return b.String()
}

func flagTypeIcon(flagType string) string {
	switch flagType {
	case "blocker":
		return "🚫"
	case "issue":
		return "⚠️"
	default:
		return "📝"
	}
}

// renderPhaseEndProgressSection is the classic "where you're up to" beat:
// the phase bar plus this phase's task count.
func renderPhaseEndProgressSection(state colony.ColonyState, phaseID int) string {
	var b strings.Builder
	b.WriteString("📍 ")
	b.WriteString(renderProgressSummary(phaseID, len(state.Plan.Phases)))
	b.WriteString("\n")
	if phaseID >= 1 && phaseID <= len(state.Plan.Phases) {
		phase := state.Plan.Phases[phaseID-1]
		if total := len(phase.Tasks); total > 0 {
			done := 0
			for _, task := range phase.Tasks {
				if task.Status == colony.TaskCompleted {
					done++
				}
			}
			fmt.Fprintf(&b, "   └── Tasks %s %d/%d in phase %d\n", generateProgressBar(done, total, 16), done, total, phaseID)
		}
	}
	return b.String()
}

// renderPhaseEndFooter composes the restored footer: flags, active steering
// signals (content and strength, not a bare count), and progress. The
// clear-context line and Next Up stay with the callers — both already exist;
// what was missing is everything above them.
func renderPhaseEndFooter(state colony.ColonyState, phaseID int) string {
	var b strings.Builder
	b.WriteString(renderPhaseEndFlagsSection())
	b.WriteString(renderSteeringSignals())
	b.WriteString(renderPhaseEndProgressSection(state, phaseID))
	return b.String()
}

// phaseHandoffRecordsExist reports whether at least one persisted worker
// handoff exists for the phase — the build closeout's half of the
// "handoff really saved" check (the other half is .aether/HANDOFF.md).
func phaseHandoffRecordsExist(phaseID int) bool {
	records, err := loadWorkerHandoffRecords()
	if err != nil {
		return false
	}
	for _, record := range records {
		if record.Phase == phaseID {
			return true
		}
	}
	return false
}
