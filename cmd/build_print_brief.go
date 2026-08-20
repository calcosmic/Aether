package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
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

	// The context capsule is manifest-level, not per-dispatch (CONTEXT-07): it
	// is resolved once here for display only. Nothing below may write it into
	// single[0].Brief or any dispatch — that would reintroduce the per-dispatch
	// duplication plan 01 removed.
	capsule := resolveCodexWorkerContext()

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

		// D-05/D-06: assert zero duplicated sections on the SAME assembled text
		// (capsule + brief + skill section) renderBriefComposition/
		// renderBriefChecklist already treat as the total assembled worker
		// context — a repeated owned heading or a repeated handoff-schema
		// sentence means the worker would receive the same steering content
		// twice. This must fail --print-brief regardless of which mode
		// (checklist or --full) is active, so it runs once here, ahead of
		// either rendering branch.
		assembled := capsule + "\n" + brief + "\n" + single[0].SkillSection
		if duplicated := duplicatedBriefSections(assembled); len(duplicated) > 0 {
			return fmt.Errorf("dispatch %s delivers duplicated context: %s (each owned section and the handoff schema must appear exactly once in the assembled worker context)", dispatch.Name, strings.Join(duplicated, ", "))
		}

		out.WriteString(strings.Repeat("━", 72))
		out.WriteString(fmt.Sprintf("\n%s  %s  (%s)\n", casteEmoji(dispatch.Caste), dispatch.Name, dispatch.Caste))
		out.WriteString(strings.Repeat("━", 72))
		out.WriteString("\n\n")

		if options.Full {
			// Print every piece a wrapper-spawned worker actually receives
			// (WR-04): the manifest-level capsule (clearly marked as such,
			// since it is resolved once above and shared across dispatches,
			// not per-worker), then the brief, then the skill section. The
			// wrapper contract (.claude/commands/ant/build.md:97) prompts
			// workers with context_capsule + brief + skill_section, so
			// --full printing brief alone was not "exactly what the worker
			// receives" despite the function's own doc comment promising it.
			out.WriteString("── Context Capsule (manifest-level) ──\n\n")
			out.WriteString(capsule)
			out.WriteString("\n\n")
			out.WriteString(brief)
			if skill := single[0].SkillSection; skill != "" {
				out.WriteString("\n\n── Skill Section ──\n\n")
				out.WriteString(skill)
			}
			out.WriteString("\n")
			out.WriteString(renderBriefComposition(capsule, brief, single[0].SkillSection))
			out.WriteString("\n")
		} else {
			out.WriteString(renderBriefChecklist(single[0], brief, capsule))
			out.WriteString("\n")
		}
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

// renderBriefComposition breaks the full assembled worker context -- the
// manifest-level capsule, the composed brief, and the per-dispatch skill
// section -- into named parts and reports the size of each as a share of the
// whole, largest first. The point is to make it obvious when framework
// scaffolding outweighs the worker's actual task.
//
// The denominator is capsule+brief+skills, not brief alone (WR-04): the
// wrapper contract (.claude/commands/ant/build.md:97) prompts workers with
// context_capsule + brief + skill_section, and the checklist's own TOTAL
// line (renderBriefChecklist) already uses that same three-way sum, so
// --full's composition table must agree with the checklist about what "the
// total assembled context" means for the same prompt.
func renderBriefComposition(capsule, brief, skillSection string) string {
	sections := splitBriefSections(brief)
	if len(capsule) > 0 {
		sections = append(sections, briefSection{Name: "Context Capsule (manifest-level)", Chars: len(capsule)})
	}
	if len(skillSection) > 0 {
		sections = append(sections, briefSection{Name: "Skill Section", Chars: len(skillSection)})
	}
	total := len(capsule) + len(brief) + len(skillSection)
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
	"Task Resolution Notice":   true,
	"Phase Success Criteria":   true,
	"Heartbeat Protocol":       true,
	"Relevant Playbooks":       true,
	"Pheromone Signals":        true,
	"Territory Survey":         true,
	"Phase Research":           true,
	"Verification Command":     true,
	"Codebase Graph Context":   true,
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

// handoffSectionDuplicationAnchorChars is how many leading characters of
// codex.HandoffFieldsSummary anchor the handoff-sentence duplication check.
// The handoff schema sentence has no "## " heading of its own (189's D-04
// design -- it is plain text appended after composeBuildManifestBrief's base
// render), so splitBriefSections's heading walk cannot see it repeat; a
// substring of the constant's own content is used instead of a hand-copied
// piece of English, so the anchor can never drift from the schema
// ValidateWorkerHandoff actually enforces. 40 chars of this specific,
// generated sentence is long enough that it cannot plausibly false-match
// unrelated prose.
const handoffSectionDuplicationAnchorChars = 40

// handoffSectionDuplicationAnchor returns the stable substring
// duplicatedBriefSections counts occurrences of.
func handoffSectionDuplicationAnchor() string {
	if len(codex.HandoffFieldsSummary) <= handoffSectionDuplicationAnchorChars {
		return codex.HandoffFieldsSummary
	}
	return codex.HandoffFieldsSummary[:handoffSectionDuplicationAnchorChars]
}

// duplicatedBriefSectionHandoffLabel is the name duplicatedBriefSections
// reports when the handoff-schema sentence repeats. It has no heading of its
// own, so it is not a member of briefOwnedSections -- this label exists only
// to give a duplication finding a human-readable name distinct from any real
// heading.
const duplicatedBriefSectionHandoffLabel = "Handoff Schema"

// duplicatedBriefSections reports every owned heading (per briefOwnedSections)
// or the handoff-schema sentence that occurs more than once in assembled --
// the same capsule + composed brief + skill-section text printWorkerBriefs
// already has in hand for --full rendering, and the same denominator
// renderBriefComposition/renderBriefChecklist already use (WR-04: capsule +
// brief + skills, not brief alone).
//
// It performs two independent counting passes: heading occurrences via the
// same "## " walk splitBriefSections uses, but COUNTING every occurrence
// instead of splitBriefSections's first-wins/restart behavior (a second
// "## Pheromone Signals" today silently becomes a second, same-named
// briefSection entry that renderBriefComposition/renderBriefChecklist never
// flag as a repeat); and occurrences of a stable anchor drawn from
// codex.HandoffFieldsSummary's own text, since the handoff sentence carries
// no heading for the first pass to see.
//
// Returns the sorted, deduplicated names of anything found more than once;
// an empty slice when nothing repeats -- the healthy, common case.
func duplicatedBriefSections(assembled string) []string {
	counts := make(map[string]int)
	for _, line := range strings.Split(assembled, "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if !briefOwnedSections[heading] {
			// An unrecognized "## " heading belongs to injected content
			// (e.g. a playbook), not the brief's own structure -- mirrors
			// splitBriefSections's existing exclusion exactly.
			continue
		}
		counts[heading]++
	}

	if anchor := handoffSectionDuplicationAnchor(); anchor != "" {
		if n := strings.Count(assembled, anchor); n > 1 {
			counts[duplicatedBriefSectionHandoffLabel] = n
		}
	}

	duplicated := make([]string, 0, len(counts))
	for name, n := range counts {
		if n > 1 {
			duplicated = append(duplicated, name)
		}
	}
	sort.Strings(duplicated)
	return duplicated
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
// a printed brief matches what would actually be dispatched. full threads the
// --full flag through so printWorkerBriefs knows whether to render the raw
// prompt or the checklist.
func buildPrintBriefOptions(workerTimeout time.Duration, force, light, heavy, full bool, verificationDepth string) codexBuildOptions {
	return codexBuildOptions{
		WorkerTimeout:     workerTimeout,
		Force:             force,
		LightFlag:         light,
		HeavyFlag:         heavy,
		VerificationDepth: verificationDepth,
		Full:              full,
	}
}

// briefTaskContentAllowanceChars bounds everything an assembled worker
// context carries that has no named budget constant of its own: assignment,
// dependencies, constraints, hints, success criteria, pheromone signals,
// previous worker handoffs, and the territory survey pointer list. This is a
// judgement call, not a measurement — it exists so the budget ceiling
// (D-03's growth guard) has a concrete number to sum against, distinct from
// the grounding budgets (capsule, skills, research, codegraph) that already
// declare their own constants. If real usage shows this is consistently too
// tight or too loose, that is a finding to raise, not a number to creep.
const briefTaskContentAllowanceChars = 6000

// assembledContextBudgetCeilingChars is the derived ceiling for the total
// assembled worker context: the sum of every named budget constant plus the
// task-content allowance above. It is derived, never a literal — a literal
// would silently drift the moment any budget constant moves.
//
// Sums colonyPrimeCompactBudgetChars, not colonyPrimeBudgetChars (WR-03):
// every capsule this codebase actually delivers to a worker goes through
// resolveCodexWorkerContext(), which always calls
// buildColonyPrimeOutput(true) -- the compact budget. Summing the
// non-compact constant here made the ceiling ~4000 chars looser than the
// real delivery budget, so this guard couldn't trip until reality had
// drifted 2x past its actual cap.
func assembledContextBudgetCeilingChars() int {
	return colonyPrimeCompactBudgetChars +
		skillInjectNormalBudgetChars +
		phaseResearchBriefBudgetChars +
		codegraphWorkerContextBudgetChars +
		briefTaskContentAllowanceChars
}

// briefChecklistRow is one line of the D-06 inspector checklist: a named
// context section, whether it arrived, its size, and an optional annotation
// (used for the survey staleness notice).
type briefChecklistRow struct {
	Label   string
	Present bool
	Chars   int
	Note    string
}

// locateChecklistSection finds an exact heading line (e.g. "## Phase
// Research" or "### Territory Survey") inside text and returns whether it is
// present, plus the character span from that heading through the next "## "
// or "### " heading line, or the end of the text. Unlike splitBriefSections,
// this does not depend on the heading being registered in
// briefOwnedSections, so it works for headings the checklist needs to detect
// (the charter heading inside the capsule, the survey's own "### " heading)
// without needing that registry's exact spelling to match. Named distinctly
// from oracle_loop.go's extractBriefSection (a different, single-value
// helper) to avoid collision.
func locateChecklistSection(text, heading string) (present bool, chars int) {
	idx := strings.Index(text, heading)
	if idx < 0 {
		return false, 0
	}
	rest := text[idx:]
	pieces := strings.SplitAfter(rest, "\n")
	if len(pieces) == 0 {
		return true, len(rest)
	}
	size := len(pieces[0])
	for _, piece := range pieces[1:] {
		if strings.HasPrefix(piece, "## ") || strings.HasPrefix(piece, "### ") {
			break
		}
		size += len(piece)
	}
	return true, size
}

// checklistRowFor builds a checklist row by locating heading inside brief.
func checklistRowFor(brief, label, heading string) briefChecklistRow {
	present, chars := locateChecklistSection(brief, heading)
	return briefChecklistRow{Label: label, Present: present, Chars: chars}
}

// checklistRowForEither builds a checklist row by locating heading inside
// EITHER brief or capsule, whichever carries it (190-03, D-190-01-A). Since
// composeBuildManifestBrief now omits pheromone signals and prior worker
// handoffs from the brief for every caller that also carries a capsule (the
// checklist's own caller included), those two sections live in the capsule
// exclusively -- checking brief alone (checklistRowFor's behavior) would
// report them ABSENT even though the worker still receives them, which is
// exactly the misreport renderBriefChecklist's own doc comment warns against
// ("a section the runtime silently stopped delivering shows up as ABSENT
// here even if the state that would produce it still exists" -- the inverse
// error, reporting ABSENT for a section that IS delivered elsewhere, is just
// as wrong). Checking both sources keeps this row honest regardless of which
// side currently owns the content.
func checklistRowForEither(brief, capsule, label, heading string) briefChecklistRow {
	if present, chars := locateChecklistSection(brief, heading); present {
		return briefChecklistRow{Label: label, Present: true, Chars: chars}
	}
	if present, chars := locateChecklistSection(capsule, heading); present {
		return briefChecklistRow{Label: label, Present: true, Chars: chars}
	}
	return briefChecklistRow{Label: label, Present: false, Chars: 0}
}

// renderBriefChecklist is the default `--print-brief` output (D-06): a
// ten-second, sectioned answer to "which context arrived and which did not,"
// with sizes and a total against a real, derived budget. It reports what the
// worker would actually receive — brief sections plus the manifest-level
// capsule and skill section read for display only — never what the database
// contains, so a section the runtime silently stopped delivering shows up as
// ABSENT here even if the state that would produce it still exists.
func renderBriefChecklist(dispatch codexBuildDispatch, brief, capsule string) string {
	var rows []briefChecklistRow

	taskChars := 0
	taskAnyPresent := false
	for _, section := range splitBriefSections(brief) {
		switch section.Name {
		case "Assignment", "Phase Objective", "Phase Success Criteria",
			"Task Success Criteria", "Dependencies", "Task Constraints",
			"Constraints", "Hints":
			taskChars += section.Chars
			taskAnyPresent = true
		}
	}
	rows = append(rows, briefChecklistRow{Label: "Assignment & Task Content", Present: taskAnyPresent, Chars: taskChars})

	territoryRow := checklistRowFor(brief, "Territory Survey", "### Territory Survey")
	if territoryRow.Present {
		switch {
		case strings.Contains(brief, "STALE MAP WARNING"):
			territoryRow.Note = "STALE MAP WARNING — codebase map may not match the tree, run /ant-colonize"
		case strings.Contains(brief, "never been surveyed"):
			territoryRow.Note = "territory has never been surveyed"
		}
	}
	rows = append(rows, territoryRow)

	rows = append(rows, checklistRowFor(brief, "Phase Research", "## Phase Research"))
	rows = append(rows, checklistRowFor(brief, "Colony Research", "## Colony Research"))
	rows = append(rows, checklistRowFor(brief, "Codegraph Context", "## Codebase Graph Context"))
	rows = append(rows, checklistRowForEither(brief, capsule, "Pheromone Signals", "## Pheromone Signals"))
	rows = append(rows, checklistRowForEither(brief, capsule, "Previous Worker Handoffs", "## Previous Worker Handoffs"))
	rows = append(rows, checklistRowFor(brief, "Expected Output", "## Expected Output"))

	capsulePresent := strings.TrimSpace(capsule) != ""
	rows = append(rows, briefChecklistRow{Label: "Context Capsule (manifest-level)", Present: capsulePresent, Chars: len(capsule)})

	charterPresent, charterChars := locateChecklistSection(capsule, charterSectionHeading())
	rows = append(rows, briefChecklistRow{Label: "Charter (inside capsule)", Present: charterPresent, Chars: charterChars})

	var b strings.Builder
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n  CONTEXT CHECKLIST\n")
	b.WriteString(strings.Repeat("─", 72))
	b.WriteString("\n")

	for _, row := range rows {
		marker := "ABSENT"
		if row.Present {
			marker = "present"
		}
		b.WriteString(fmt.Sprintf("  %-34s %-9s %6d\n", truncateSectionName(row.Label, 34), marker, row.Chars))
		if row.Note != "" {
			b.WriteString(fmt.Sprintf("    ⚠ %s\n", row.Note))
		}
	}

	skillChars := len(dispatch.SkillSection)
	total := len(capsule) + len(brief) + skillChars
	ceiling := assembledContextBudgetCeilingChars()
	pct := 0.0
	if ceiling > 0 {
		pct = float64(total) / float64(ceiling) * 100
	}

	b.WriteString(strings.Repeat("─", 72))
	b.WriteString(fmt.Sprintf("\n  %-34s %6d / %-6d %5.1f%%\n", "TOTAL (capsule+brief+skills)", total, ceiling, pct))
	b.WriteString("\n  Run with --full to print the raw assembled prompt.\n")

	return b.String()
}

var _ = colony.PhaseModeProduction
