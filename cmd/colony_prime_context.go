package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
)

type colonyPrimeOutput struct {
	Context       string   `json:"context"`
	PromptSection string   `json:"prompt_section"`
	SignalCount   int      `json:"signal_count"`
	InstinctCount int      `json:"instinct_count"`
	LogLine       string   `json:"log_line"`
	Budget        int      `json:"budget"`
	Used          int      `json:"used"`
	Sections      int      `json:"sections"`
	Trimmed       []string `json:"trimmed"`
}

type colonyPrimeSection struct {
	name     string
	content  string
	priority int // lower = trimmed first
}

func buildColonyPrimeOutput(compact bool) colonyPrimeOutput {
	budget := 8000
	if compact {
		budget = 4000
	}
	result := colonyPrimeOutput{
		Budget:  budget,
		Trimmed: []string{},
	}
	if store == nil {
		return result
	}

	sc := cache.NewSessionCache(store.BasePath())
	sc.ClearStale(24 * time.Hour)

	sections := make([]colonyPrimeSection, 0, 9)

	var state colony.ColonyState
	statePath := filepath.Join(store.BasePath(), "COLONY_STATE.json")
	if err := sc.Load(statePath, &state); err != nil {
		_ = store.LoadJSON("COLONY_STATE.json", &state)
	}

	var stateSection strings.Builder
	stateSection.WriteString("## Colony State\n\n")
	if state.Goal != nil {
		stateSection.WriteString(fmt.Sprintf("Goal: %s\n", *state.Goal))
	}
	stateSection.WriteString(fmt.Sprintf("State: %s\n", state.State))
	stateSection.WriteString(fmt.Sprintf("Phase: %d\n", state.CurrentPhase))
	if len(state.Plan.Phases) > 0 && state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		phase := state.Plan.Phases[state.CurrentPhase-1]
		stateSection.WriteString(fmt.Sprintf("Phase Name: %s\n", phase.Name))
		if len(phase.Tasks) > 0 {
			stateSection.WriteString("Tasks:\n")
			for _, t := range phase.Tasks {
				stateSection.WriteString(fmt.Sprintf("  - [%s] %s\n", t.Status, t.Goal))
			}
		}
	}
	mode := state.ParallelMode
	if mode == "" {
		mode = colony.ModeInRepo
	}
	stateSection.WriteString(fmt.Sprintf("Parallel Mode: %s\n", mode))
	sections = append(sections, colonyPrimeSection{name: "state", content: stateSection.String(), priority: 5})

	now := time.Now().UTC()
	pf, phErr := loadPheromonesOnce(store, sc)
	if phErr == nil && len(pf.Signals) > 0 {
		activeSignals := filterSignalsForPrompt(pf.Signals, now)
		result.SignalCount = len(activeSignals)
		if len(activeSignals) > 0 {
			var phSB strings.Builder
			phSB.WriteString("## Pheromone Signals\n\n")
			phSB.WriteString(colonyLifecycleSignalContext(state))
			phSB.WriteString("\n\n")
			for _, sig := range activeSignals {
				text := extractText(sig.Content)
				if text == "" {
					continue
				}
				phSB.WriteString(fmt.Sprintf("- [%s] %s\n", sig.Type, text))
			}
			if strings.TrimSpace(phSB.String()) != "" {
				sections = append(sections, colonyPrimeSection{name: "pheromones", content: phSB.String(), priority: 9})
			}
		}
	}

	var instincts []struct {
		trigger    string
		action     string
		confidence float64
	}
	var instFile colony.InstinctsFile
	instinctsPath := filepath.Join(store.BasePath(), "instincts.json")
	instinctsLoaded := false
	if err := sc.Load(instinctsPath, &instFile); err == nil {
		instinctsLoaded = true
	} else if err := store.LoadJSON("instincts.json", &instFile); err == nil {
		instinctsLoaded = true
	}
	if instinctsLoaded {
		for _, inst := range instFile.Instincts {
			if inst.Archived {
				continue
			}
			instincts = append(instincts, struct {
				trigger    string
				action     string
				confidence float64
			}{trigger: inst.Trigger, action: inst.Action, confidence: inst.Confidence})
		}
	} else if state.Memory.Instincts != nil {
		for _, inst := range state.Memory.Instincts {
			instincts = append(instincts, struct {
				trigger    string
				action     string
				confidence float64
			}{trigger: inst.Trigger, action: inst.Action, confidence: inst.Confidence})
		}
	}
	if len(instincts) > 0 {
		var instSB strings.Builder
		instSB.WriteString("## Active Instincts\n\n")
		for _, inst := range instincts {
			instSB.WriteString(fmt.Sprintf("- [%s] %s (confidence: %.2f)\n", inst.trigger, inst.action, inst.confidence))
		}
		sections = append(sections, colonyPrimeSection{name: "instincts", content: instSB.String(), priority: 6})
	}
	result.InstinctCount = len(instincts)

	if state.Memory.Decisions != nil && len(state.Memory.Decisions) > 0 {
		var decSB strings.Builder
		decSB.WriteString("## Key Decisions\n\n")
		for _, d := range state.Memory.Decisions {
			decSB.WriteString(fmt.Sprintf("- Phase %d: %s — %s\n", d.Phase, d.Claim, d.Rationale))
		}
		sections = append(sections, colonyPrimeSection{name: "decisions", content: decSB.String(), priority: 3})
	}

	if state.Memory.PhaseLearnings != nil && len(state.Memory.PhaseLearnings) > 0 {
		var learnSB strings.Builder
		learnSB.WriteString("## Phase Learnings\n\n")
		for _, pl := range state.Memory.PhaseLearnings {
			learnSB.WriteString(fmt.Sprintf("### Phase %d: %s\n", pl.Phase, pl.PhaseName))
			for _, l := range pl.Learnings {
				learnSB.WriteString(fmt.Sprintf("  - %s [%s]\n", l.Claim, l.Status))
			}
		}
		sections = append(sections, colonyPrimeSection{name: "learnings", content: learnSB.String(), priority: 2})
	}

	hubDir := resolveHubPath()
	var fallbacks []string
	hiveEntries := readHiveWisdom(hubDir, 5, &fallbacks)
	if len(hiveEntries) > 0 {
		var hiveSB strings.Builder
		hiveSB.WriteString("## HIVE WISDOM (Cross-Colony Patterns)\n\n")
		for _, entry := range hiveEntries {
			hiveSB.WriteString(fmt.Sprintf("- %s\n", entry))
		}
		sections = append(sections, colonyPrimeSection{name: "hive_wisdom", content: hiveSB.String(), priority: 4})
	}

	userPrefs := readUserPreferences(filepath.Join(hubDir, "QUEEN.md"))
	if len(userPrefs) > 0 {
		var prefsSB strings.Builder
		prefsSB.WriteString("## USER PREFERENCES\n\n")
		for _, pref := range userPrefs {
			prefsSB.WriteString(fmt.Sprintf("- %s\n", pref))
		}
		sections = append(sections, colonyPrimeSection{name: "user_preferences", content: prefsSB.String(), priority: 7})
	}

	if clarifications := clarifiedIntentPromptEntries(); len(clarifications) > 0 {
		var clarifySB strings.Builder
		clarifySB.WriteString("## CLARIFIED INTENT\n\n")
		for _, clarification := range clarifications {
			clarifySB.WriteString(clarification)
			clarifySB.WriteString("\n")
		}
		sections = append(sections, colonyPrimeSection{name: "clarified_intent", content: clarifySB.String(), priority: 8})
	}

	var blockerFile colony.FlagsFile
	if err := store.LoadJSON("pending-decisions.json", &blockerFile); err != nil {
		_ = store.LoadJSON("flags.json", &blockerFile)
	}
	if len(blockerFile.Decisions) > 0 {
		var blockerSB strings.Builder
		for _, blocker := range blockerFile.Decisions {
			if blocker.Resolved || blocker.Type != "blocker" {
				continue
			}
			if blockerSB.Len() == 0 {
				blockerSB.WriteString("## Active Blockers\n\n")
			}
			blockerSB.WriteString(fmt.Sprintf("- %s\n", blocker.Description))
		}
		if blockerSB.Len() > 0 {
			sections = append(sections, colonyPrimeSection{name: "blockers", content: blockerSB.String(), priority: 10})
		}
	}

	sort.Slice(sections, func(i, j int) bool {
		if sections[i].priority != sections[j].priority {
			return sections[i].priority > sections[j].priority
		}
		return sections[i].name < sections[j].name
	})

	result.Sections = len(sections)
	var assembled strings.Builder
	currentLen := 0
	for _, sec := range sections {
		if currentLen+len(sec.content) > budget {
			if sec.name == "blockers" {
				assembled.WriteString(sec.content)
				assembled.WriteString("\n")
				currentLen += len(sec.content)
				continue
			}
			result.Trimmed = append(result.Trimmed, sec.name)
			continue
		}
		assembled.WriteString(sec.content)
		assembled.WriteString("\n")
		currentLen += len(sec.content)
	}

	context := strings.TrimSpace(assembled.String())
	result.Context = context
	result.PromptSection = context
	result.Used = currentLen
	result.LogLine = fmt.Sprintf("colony-prime loaded %d signal(s), %d instinct(s), used %d/%d chars", result.SignalCount, result.InstinctCount, currentLen, budget)
	return result
}

func resolveCodexWorkerContext() string {
	context := strings.TrimSpace(buildColonyPrimeOutput(true).PromptSection)
	if context != "" {
		return context
	}
	return buildContextCapsuleOutput(true, 8, 3, 2, 220).PromptSection
}
