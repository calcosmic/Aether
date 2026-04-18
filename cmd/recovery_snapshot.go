package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

type sessionSyncOptions struct {
	CommandName    string
	SuggestedNext  string
	Summary        string
	ContextCleared *bool
}

type colonyArtifactOptions struct {
	CommandName    string
	SuggestedNext  string
	Summary        string
	SafeToClear    string
	HandoffTitle   string
	WriteHandoff   bool
	ContextCleared *bool
}

func contextDocumentPath() string {
	return filepath.Join(resolveAetherRootPath(), ".aether", "CONTEXT.md")
}

func handoffDocumentPath() string {
	return filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md")
}

func readContextDocument() ([]byte, error) {
	return os.ReadFile(contextDocumentPath())
}

func writeContextDocument(content string) error {
	path := contextDocumentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create context directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func readHandoffDocument() ([]byte, error) {
	return os.ReadFile(handoffDocumentPath())
}

func writeHandoffDocument(content string) error {
	path := handoffDocumentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create handoff directory: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func removeHandoffDocument() error {
	err := os.Remove(handoffDocumentPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func syncSessionFromState(state colony.ColonyState, opts sessionSyncOptions) (colony.SessionFile, error) {
	if store == nil {
		return colony.SessionFile{}, fmt.Errorf("no store initialized")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		sessionID := fmt.Sprintf("%d_%s", time.Now().Unix(), randomHex(4))
		if state.SessionID != nil && strings.TrimSpace(*state.SessionID) != "" {
			sessionID = strings.TrimSpace(*state.SessionID)
		}
		startedAt := now
		if state.InitializedAt != nil {
			startedAt = state.InitializedAt.UTC().Format(time.RFC3339)
		}
		goal := ""
		if state.Goal != nil {
			goal = strings.TrimSpace(*state.Goal)
		}
		session = colony.SessionFile{
			SessionID:        sessionID,
			StartedAt:        startedAt,
			ColonyGoal:       goal,
			CurrentPhase:     state.CurrentPhase,
			CurrentMilestone: state.Milestone,
			SuggestedNext:    "aether status",
			ContextCleared:   false,
			BaselineCommit:   getGitHEAD(),
			ResumedAt:        nil,
			ActiveTodos:      []string{},
			Summary:          "Session initialized",
		}
	}

	if session.SessionID == "" {
		session.SessionID = fmt.Sprintf("%d_%s", time.Now().Unix(), randomHex(4))
	}
	if session.StartedAt == "" {
		session.StartedAt = now
	}

	goal := ""
	if state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	session.ColonyGoal = goal
	session.CurrentPhase = state.CurrentPhase
	session.CurrentMilestone = state.Milestone
	session.ActiveTodos = sessionActiveTodosFromState(state)
	if opts.CommandName != "" {
		session.LastCommand = opts.CommandName
		session.LastCommandAt = now
	}
	if opts.SuggestedNext != "" {
		session.SuggestedNext = opts.SuggestedNext
	} else if strings.TrimSpace(session.SuggestedNext) == "" {
		session.SuggestedNext = nextCommandFromState(state)
	}
	if opts.Summary != "" {
		session.Summary = opts.Summary
	}
	if opts.ContextCleared != nil {
		session.ContextCleared = *opts.ContextCleared
	}
	if head := strings.TrimSpace(getGitHEAD()); head != "" {
		session.BaselineCommit = head
	}

	if err := store.SaveJSON("session.json", session); err != nil {
		return colony.SessionFile{}, fmt.Errorf("save session: %w", err)
	}
	mirrorToLegacy(store)
	return session, nil
}

func syncColonyArtifacts(state colony.ColonyState, opts colonyArtifactOptions) (colony.SessionFile, error) {
	session, err := syncSessionFromState(state, sessionSyncOptions{
		CommandName:    opts.CommandName,
		SuggestedNext:  opts.SuggestedNext,
		Summary:        opts.Summary,
		ContextCleared: opts.ContextCleared,
	})
	if err != nil {
		return colony.SessionFile{}, err
	}

	next := session.SuggestedNext
	if next == "" {
		next = nextCommandFromState(state)
	}
	safeToClear := strings.TrimSpace(opts.SafeToClear)
	if safeToClear == "" {
		safeToClear = defaultSafeToClear(state)
	}

	if err := writeContextDocument(renderContextSnapshot(state, session, next, opts.Summary, safeToClear)); err != nil {
		return colony.SessionFile{}, fmt.Errorf("write context snapshot: %w", err)
	}

	if opts.WriteHandoff {
		title := strings.TrimSpace(opts.HandoffTitle)
		if title == "" {
			title = "Session Snapshot"
		}
		if err := writeHandoffDocument(renderHandoffSnapshot(state, session, title, next, opts.Summary)); err != nil {
			return colony.SessionFile{}, fmt.Errorf("write handoff snapshot: %w", err)
		}
	}

	return session, nil
}

func nextCommandFromState(state colony.ColonyState) string {
	state = normalizeLegacyColonyState(state)
	if colonyNeedsEntomb(state) {
		return "aether entomb"
	}
	if state.Paused {
		return "aether resume"
	}
	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.State == colony.StateEXECUTING && state.BuildStartedAt == nil && state.CurrentPhase > 0 {
			return fmt.Sprintf("aether build %d", state.CurrentPhase)
		}
		return "aether continue"
	case colony.StateCOMPLETED:
		return "aether entomb"
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			return "aether plan"
		}
		if phase := recoveryPhase(&state); phase != nil && phase.Status != colony.PhaseCompleted {
			return fmt.Sprintf("aether build %d", phase.ID)
		}
		return "aether seal"
	default:
		if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
			return "aether init \"goal\""
		}
		return "aether status"
	}
}

func recoveryPhase(state *colony.ColonyState) *colony.Phase {
	if state == nil || len(state.Plan.Phases) == 0 {
		return nil
	}
	if state.State == colony.StateREADY {
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].Status != colony.PhaseCompleted {
				return &state.Plan.Phases[i]
			}
		}
	}
	if state.CurrentPhase > 0 && state.CurrentPhase <= len(state.Plan.Phases) {
		return &state.Plan.Phases[state.CurrentPhase-1]
	}
	return &state.Plan.Phases[0]
}

func sessionActiveTodosFromState(state colony.ColonyState) []string {
	phase := recoveryPhase(&state)
	if phase == nil {
		return []string{}
	}
	todos := make([]string, 0, len(phase.Tasks))
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			continue
		}
		goal := strings.TrimSpace(task.Goal)
		if goal == "" {
			continue
		}
		todos = append(todos, goal)
	}
	return todos
}

func defaultSafeToClear(state colony.ColonyState) string {
	switch state.State {
	case colony.StateEXECUTING:
		return "NO — Build in progress"
	case colony.StateBUILT:
		return "YES — Build complete, ready to continue"
	case colony.StateCOMPLETED:
		return "YES — Colony complete"
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			return "YES — Colony initialized, ready to plan"
		}
		return "YES — Plan persisted, ready for the next command"
	default:
		return "YES — State persisted"
	}
}

func defaultProgressSummary(state colony.ColonyState, nextAction string) string {
	phase := recoveryPhase(&state)
	switch state.State {
	case colony.StateEXECUTING:
		if phase != nil {
			return fmt.Sprintf("Phase %d build is in progress: %s", phase.ID, phase.Name)
		}
		return "Build is in progress."
	case colony.StateBUILT:
		if phase != nil {
			return fmt.Sprintf("Phase %d build completed and is waiting for verification: %s", phase.ID, phase.Name)
		}
		return "Build completed and is waiting for verification."
	case colony.StateREADY:
		if len(state.Plan.Phases) == 0 {
			return "Colony initialized. Generate a plan to begin coordinated work."
		}
		if phase != nil {
			return fmt.Sprintf("Next ready phase: %d — %s", phase.ID, phase.Name)
		}
		return "Plan is ready."
	case colony.StateCOMPLETED:
		return "All planned phases are complete. Seal the colony when ready."
	default:
		if nextAction != "" {
			return fmt.Sprintf("State persisted. Next suggested command: %s", nextAction)
		}
		return "State persisted."
	}
}

func renderContextSnapshot(state colony.ColonyState, session colony.SessionFile, nextAction, summary, safeToClear string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	if nextAction == "" {
		nextAction = nextCommandFromState(state)
	}
	if summary == "" {
		summary = defaultProgressSummary(state, nextAction)
	}

	phase := recoveryPhase(&state)
	phaseNumber := 0
	phaseName := "initialization"
	phaseStatus := string(state.State)
	if phase != nil {
		phaseNumber = phase.ID
		if strings.TrimSpace(phase.Name) != "" {
			phaseName = phase.Name
		}
		if strings.TrimSpace(phase.Status) != "" {
			phaseStatus = phase.Status
		}
	}
	milestone := strings.TrimSpace(state.Milestone)
	if milestone == "" {
		milestone = "First Mound"
	}
	goal := strings.TrimSpace(session.ColonyGoal)
	if goal == "" && state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	if goal == "" {
		goal = "No goal set"
	}

	signals := extractSignalTexts(8)
	redirects := []string{}
	others := []string{}
	for _, sig := range signals {
		switch {
		case strings.HasPrefix(sig, "REDIRECT:"):
			redirects = append(redirects, strings.TrimSpace(strings.TrimPrefix(sig, "REDIRECT:")))
		default:
			others = append(others, strings.TrimSpace(sig))
		}
	}
	blockers := extractBlockerTexts()
	recentEvents := lastEventTexts(state.Events, 5)
	activeTasks := sessionActiveTodosFromState(state)

	var b strings.Builder
	b.WriteString("# Aether Colony — Current Context\n\n")
	b.WriteString("> **This document is the colony's memory. If context collapses, read this file first.**\n\n")
	b.WriteString("---\n\n")
	b.WriteString("## System Status\n\n")
	b.WriteString("| Field | Value |\n")
	b.WriteString("|-------|-------|\n")
	b.WriteString(fmt.Sprintf("| **Last Updated** | %s |\n", now))
	b.WriteString(fmt.Sprintf("| **Current Phase** | %d |\n", phaseNumber))
	b.WriteString(fmt.Sprintf("| **Phase Name** | %s |\n", phaseName))
	b.WriteString(fmt.Sprintf("| **Phase Status** | %s |\n", phaseStatus))
	b.WriteString(fmt.Sprintf("| **Milestone** | %s |\n", milestone))
	b.WriteString(fmt.Sprintf("| **Colony Status** | %s |\n", state.State))
	b.WriteString(fmt.Sprintf("| **Safe to Clear?** | %s |\n", safeToClear))
	b.WriteString("\n---\n\n")
	b.WriteString("## Current Goal\n\n")
	b.WriteString(goal)
	b.WriteString("\n\n---\n\n")
	b.WriteString("## What's In Progress\n\n")
	b.WriteString(summary)
	b.WriteString("\n\n---\n\n")
	b.WriteString("## Active Constraints (REDIRECT Signals)\n\n")
	if len(redirects) == 0 {
		b.WriteString("*None active*\n")
	} else {
		b.WriteString("| Constraint | Source | Date Set |\n")
		b.WriteString("|------------|--------|----------|\n")
		for _, item := range redirects {
			b.WriteString(fmt.Sprintf("| %s | pheromone | active |\n", item))
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Active Pheromones\n\n")
	if len(others) == 0 {
		b.WriteString("*None active*\n")
	} else {
		for _, item := range others {
			b.WriteString("- ")
			b.WriteString(item)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Open Blockers\n\n")
	if len(blockers) == 0 {
		b.WriteString("*None active*\n")
	} else {
		for _, blocker := range blockers {
			b.WriteString("- ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n---\n\n")
	if phase != nil {
		b.WriteString(fmt.Sprintf("## Tasks For Phase %d — %s\n\n", phase.ID, phase.Name))
		if len(phase.Tasks) == 0 {
			b.WriteString("*No tasks defined*\n")
		} else {
			for _, task := range phase.Tasks {
				box := "[ ]"
				switch task.Status {
				case colony.TaskCompleted:
					box = "[x]"
				case colony.TaskInProgress:
					box = "[>]"
				}
				goal := strings.TrimSpace(task.Goal)
				if goal == "" {
					goal = "(unnamed task)"
				}
				b.WriteString(fmt.Sprintf("- %s %s\n", box, goal))
			}
		}
		b.WriteString("\n---\n\n")
	}
	b.WriteString("## Recent Decisions\n\n")
	b.WriteString("| Date | Decision | Rationale | Made By |\n")
	b.WriteString("|------|----------|-----------|---------|\n")
	if len(state.Memory.Decisions) == 0 {
		b.WriteString("| — | No recorded decisions | — | — |\n")
	} else {
		start := 0
		if len(state.Memory.Decisions) > 5 {
			start = len(state.Memory.Decisions) - 5
		}
		for _, decision := range state.Memory.Decisions[start:] {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | Queen |\n",
				strings.TrimSpace(decision.Timestamp),
				emptyFallback(strings.TrimSpace(decision.Claim), "decision"),
				emptyFallback(strings.TrimSpace(decision.Rationale), "—"),
			))
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Recent Activity (Last 5 Events)\n\n")
	if len(recentEvents) == 0 {
		b.WriteString("*No recent events*\n")
	} else {
		for _, event := range recentEvents {
			b.WriteString("- ")
			b.WriteString(event)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n---\n\n")
	b.WriteString("## Next Steps\n\n")
	b.WriteString(fmt.Sprintf("1. Run `%s`\n", nextAction))
	if phase != nil {
		b.WriteString(fmt.Sprintf("2. Run `aether phase --number %d` to inspect the tracked phase details\n", phase.ID))
	} else {
		b.WriteString("2. Run `aether status` for the colony dashboard\n")
	}
	b.WriteString("3. Run `aether resume-colony` after a context clear if you want the full recovery view\n")
	b.WriteString("\n---\n\n")
	b.WriteString("## If Context Collapses\n\n")
	b.WriteString("1. Run `aether resume` for the quick dashboard restore\n")
	b.WriteString("2. Run `aether resume-colony` for the full handoff and task view\n")
	b.WriteString("3. Read `.aether/HANDOFF.md` if a richer session summary was persisted\n")
	if len(activeTasks) > 0 {
		b.WriteString("\n### Active Todos\n")
		for _, task := range activeTasks {
			b.WriteString("- ")
			b.WriteString(task)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderHandoffSnapshot(state colony.ColonyState, session colony.SessionFile, title, nextAction, summary string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	if nextAction == "" {
		nextAction = nextCommandFromState(state)
	}
	if summary == "" {
		summary = defaultProgressSummary(state, nextAction)
	}

	goal := strings.TrimSpace(session.ColonyGoal)
	if goal == "" && state.Goal != nil {
		goal = strings.TrimSpace(*state.Goal)
	}
	if goal == "" {
		goal = "No goal set"
	}

	phase := recoveryPhase(&state)
	phaseLine := "0/0 — initialization"
	if phase != nil {
		phaseLine = fmt.Sprintf("%d/%d — %s", phase.ID, len(state.Plan.Phases), phase.Name)
	}

	signals := extractSignalTexts(8)
	blockers := extractBlockerTexts()
	tasks := sessionActiveTodosFromState(state)

	var b strings.Builder
	b.WriteString("# Colony Session — ")
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString("Updated: ")
	b.WriteString(now)
	b.WriteString("\n\n")
	b.WriteString("## Goal\n\n")
	b.WriteString("- ")
	b.WriteString(goal)
	b.WriteString("\n\n")
	b.WriteString("## Phase\n\n")
	b.WriteString("- Current: ")
	b.WriteString(phaseLine)
	b.WriteString("\n")
	b.WriteString("- State: ")
	b.WriteString(string(state.State))
	b.WriteString("\n")
	if strings.TrimSpace(state.Milestone) != "" {
		b.WriteString("- Milestone: ")
		b.WriteString(state.Milestone)
		b.WriteString("\n")
	}
	b.WriteString("\n## Signals\n\n")
	if len(signals) == 0 {
		b.WriteString("- None\n")
	} else {
		for _, signal := range signals {
			b.WriteString("- ")
			b.WriteString(signal)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Blockers\n\n")
	if len(blockers) == 0 {
		b.WriteString("- None\n")
	} else {
		for _, blocker := range blockers {
			b.WriteString("- ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Next Step\n\n")
	b.WriteString("- Run `")
	b.WriteString(nextAction)
	b.WriteString("`\n")
	b.WriteString("- Quick restore: `aether resume`\n")
	b.WriteString("- Full restore: `aether resume-colony`\n")
	b.WriteString("\n## Tasks\n\n")
	if len(tasks) == 0 {
		b.WriteString("- None\n")
	} else {
		for _, task := range tasks {
			b.WriteString("- ")
			b.WriteString(task)
			b.WriteString("\n")
		}
	}
	b.WriteString("\n## Session Summary\n\n")
	b.WriteString(summary)
	b.WriteString("\n")
	return b.String()
}

func lastEventTexts(events []string, n int) []string {
	if len(events) == 0 {
		return nil
	}
	if n > len(events) {
		n = len(events)
	}
	result := make([]string, 0, n)
	for i := len(events) - n; i < len(events); i++ {
		result = append(result, strings.TrimSpace(events[i]))
	}
	return result
}
