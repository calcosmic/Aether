package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// printWorkerBriefs renders the exact prompt each worker would receive for a
// phase and reports how that prompt is composed, section by section.
//
// This exists because worker prompts were never inspectable. Section budgets
// were tuned, context was added, and playbooks were injected for months without
// anyone being able to see the resulting prompt. The first time one was printed,
// 69% of it turned out to be truncated orchestrator instructions telling the
// worker it was the Queen.
//
// It deliberately does NOT reuse runCodexBuildPlanOnlyWithOptions. That path
// runs pre-build gates, validates state transitions and opens a build attempt —
// it mutates. An inspection command that mutates is the same defect as a
// --dry-run flag that writes to disk, which this codebase already has. This
// function calls only pure readers: load state, resolve the Queen's policy,
// compute the planned dispatches, render the brief.
func printWorkerBriefs(root string, phaseNum int, selectedTaskIDs []string, workerName string, options codexBuildOptions) error {
	if store == nil {
		return fmt.Errorf("no store initialized")
	}

	state, err := loadActiveColonyState()
	if err != nil {
		return fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if len(state.Plan.Phases) == 0 {
		return fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}

	phase := state.Plan.Phases[phaseNum-1]

	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{
		LightFlag:         options.LightFlag,
		HeavyFlag:         options.HeavyFlag,
		VerificationDepth: options.VerificationDepth,
		WorkerTimeout:     options.WorkerTimeout,
	})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)

	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, uniqueSortedStrings(selectedTaskIDs), reviewDepth)
	if len(dispatches) == 0 {
		return fmt.Errorf("phase %d has no planned dispatches", phaseNum)
	}

	startedAt := time.Now()

	matched := 0
	var out strings.Builder

	for _, dispatch := range dispatches {
		if workerName != "" && !strings.EqualFold(dispatch.Name, workerName) {
			continue
		}
		matched++

		// Attach the same context the manifest path attaches, then compose the
		// same brief the manifest carries — the inspector must show exactly what
		// a wrapper-spawned worker receives, steering sections included.
		single := []codexBuildDispatch{dispatch}
		attachBuildDispatchContext(root, phase, single, startedAt)
		brief := single[0].Brief

		out.WriteString(strings.Repeat("━", 72))
		out.WriteString(fmt.Sprintf("\n%s  %s  (%s)\n", casteEmoji(dispatch.Caste), dispatch.Name, dispatch.Caste))
		out.WriteString(strings.Repeat("━", 72))
		out.WriteString("\n\n")
		out.WriteString(brief)
		out.WriteString("\n")
		out.WriteString(renderBriefComposition(brief))
		out.WriteString("\n")
	}

	if matched == 0 {
		names := make([]string, 0, len(dispatches))
		for _, d := range dispatches {
			names = append(names, d.Name)
		}
		return fmt.Errorf("no worker named %q in phase %d (available: %s)", workerName, phaseNum, strings.Join(names, ", "))
	}

	fmt.Fprint(stdout, out.String())
	return nil
}

// briefSection is one "## Heading" block of a rendered worker brief.
type briefSection struct {
	Name  string
	Chars int
}

// renderBriefComposition breaks a brief into its markdown sections and reports
// the size of each as a share of the whole, largest first. The point is to make
// it obvious when framework scaffolding outweighs the worker's actual task.
func renderBriefComposition(brief string) string {
	sections := splitBriefSections(brief)
	total := len(brief)
	if total == 0 {
		return ""
	}

	sort.SliceStable(sections, func(i, j int) bool {
		return sections[i].Chars > sections[j].Chars
	})

	var b strings.Builder
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n  COMPOSITION\n")
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n")

	for _, section := range sections {
		pct := float64(section.Chars) / float64(total) * 100
		barWidth := int(pct / 2.5)
		if barWidth < 1 && section.Chars > 0 {
			barWidth = 1
		}
		b.WriteString(fmt.Sprintf("  %-34s %6d  %5.1f%%  %s\n",
			truncateSectionName(section.Name, 34),
			section.Chars,
			pct,
			strings.Repeat("█", barWidth),
		))
	}

	b.WriteString(strings.Repeat("─", 72))
	b.WriteString(fmt.Sprintf("\n  %-34s %6d  100.0%%\n", "TOTAL", total))
	return b.String()
}

// briefOwnedSections are the headings renderCodexBuildWorkerBrief itself emits.
// Anything else at "## " level came from injected content — notably the
// playbooks, whose markdown contains its own second-level headings that collide
// with the brief's structure. Without this set, injected playbook text is
// misattributed as brief sections, which understates how much of the prompt is
// scaffolding. It also means a worker cannot reliably tell where its own
// instructions end and the orchestrator playbook begins.
var briefOwnedSections = map[string]bool{
	"Assignment":               true,
	"Read Cache Discipline":    true,
	"Phase Objective":          true,
	"Dependencies":             true,
	"Task Constraints":         true,
	"Constraints":              true,
	"Hints":                    true,
	"Task Success Criteria":    true,
	"Phase Success Criteria":   true,
	"Heartbeat Protocol":       true,
	"Relevant Playbooks":       true,
	"Pheromone Signals":        true,
	"Territory Survey":         true,
	"Codegraph Context":        true,
	"Previous Worker Handoffs": true,
	"Expected Output":          true,
}

func splitBriefSections(brief string) []briefSection {
	lines := strings.Split(brief, "\n")
	sections := make([]briefSection, 0, 12)

	current := "(preamble)"
	size := 0

	flush := func() {
		if size > 0 {
			sections = append(sections, briefSection{Name: current, Chars: size})
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			// A heading the brief does not own belongs to injected content and
			// stays attributed to the section that injected it.
			if !briefOwnedSections[heading] {
				size += len(line) + 1
				continue
			}
			flush()
			current = heading
			size = len(line) + 1
			continue
		}
		size += len(line) + 1
	}
	flush()

	return sections
}

func truncateSectionName(name string, max int) string {
	if len(name) <= max {
		return name
	}
	if max <= 1 {
		return name[:max]
	}
	return name[:max-1] + "…"
}

// buildPrintBriefOptions mirrors the depth flags the real build path honours, so
// a printed brief matches what would actually be dispatched.
func buildPrintBriefOptions(workerTimeout time.Duration, force, light, heavy bool, verificationDepth string) codexBuildOptions {
	return codexBuildOptions{
		WorkerTimeout:     workerTimeout,
		Force:             force,
		LightFlag:         light,
		HeavyFlag:         heavy,
		VerificationDepth: verificationDepth,
	}
}

var _ = colony.PhaseModeProduction
