package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

// promptNumberedChoice presents a numbered list of options and reads user input.
// Returns 0 for invalid input.
func promptNumberedChoice(question string, options []string) int {
	fmt.Fprintf(os.Stderr, "\n%s\n", question)
	for i, opt := range options {
		fmt.Fprintf(os.Stderr, "  %d. %s\n", i+1, opt)
	}
	fmt.Fprintf(os.Stderr, "\n  Choice [1-%d]: ", len(options))

	reader := getStdinReader()
	response, _ := reader.ReadString('\n')
	trimmed := strings.TrimSpace(response)
	n, _ := strconv.Atoi(trimmed)
	if n < 1 || n > len(options) {
		return 0
	}
	return n
}

// promptString reads a single line of text from stdin.
func promptString(question string) string {
	fmt.Fprintf(os.Stderr, "\n%s: ", question)
	reader := getStdinReader()
	response, _ := reader.ReadString('\n')
	return strings.TrimSpace(response)
}

// stdinReader abstracts stdin reading for testability.
// It returns a *bufio.Reader. If stdinReader is set, it's called once
// and the result is cached for the lifetime of the ceremony to avoid
// multiple bufio.Reader instances competing for the same stream.
var stdinReader func() *bufio.Reader

var cachedStdinReader *bufio.Reader

func getStdinReader() *bufio.Reader {
	if cachedStdinReader != nil {
		return cachedStdinReader
	}
	if stdinReader != nil {
		cachedStdinReader = stdinReader()
		return cachedStdinReader
	}
	cachedStdinReader = bufio.NewReader(os.Stdin)
	return cachedStdinReader
}

// resetCachedStdinReader clears the cached reader (for tests).
func resetCachedStdinReader() {
	cachedStdinReader = nil
}

// runInitCeremony executes the full init ceremony flow:
// 1. Run init-research to scan and generate charter
// 2. Display charter using renderCharterDisplay
// 3. Auto-approve pheromone suggestions
// 4. Prompt user: Proceed / Revise / Cancel
// 5. Act on choice
func runInitCeremony(cmd *cobra.Command, args []string) error {
	if store == nil {
		outputErrorMessage("no store initialized")
		return nil
	}

	goal := strings.TrimSpace(args[0])
	if goal == "" {
		outputError(1, "goal must not be empty", nil)
		return nil
	}

	target, _ := cmd.Flags().GetString("target")
	if target == "" {
		target = "."
	}

	scopeRaw, _ := cmd.Flags().GetString("scope")
	scope, err := colony.ParseColonyScope(scopeRaw)
	if err != nil {
		outputError(1, fmt.Sprintf("invalid scope %q", scopeRaw), nil)
		return nil
	}

	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
	// In test mode (stdinReader is set), skip TTY check
	isTestMode := stdinReader != nil
	if !nonInteractive && !isTestMode && !isTerm(os.Stdin) {
		outputError(1, "init-ceremony requires an interactive terminal. Use --non-interactive with --charter-json for non-TTY environments.", nil)
		return nil
	}

	// Run the ceremony loop (supports Revise cycling)
	for {
		charter, pheromoneSuggestions, err := runCeremonyResearch(goal, target)
		if err != nil {
			outputError(1, fmt.Sprintf("research failed: %v", err), nil)
			return nil
		}

		// Emit scanned event
		emitLifecycleCeremony("colony:init:scanned", events.CeremonyPayload{
			Message: fmt.Sprintf("Research scan complete for goal: %s", goal),
		}, "init-ceremony")

		// Display charter
		fmt.Fprint(os.Stderr, renderCharterDisplay(*charter))

		// Auto-approve pheromone suggestions
		if len(pheromoneSuggestions) > 0 {
			fmt.Fprintf(os.Stderr, "\n  Auto-approved %d pheromone suggestion(s):\n", len(pheromoneSuggestions))
			for _, sug := range pheromoneSuggestions {
				fmt.Fprintf(os.Stderr, "    [%s] %s\n", sug.Type, sug.Content)
			}
		}

		// Final approval prompt
		choice := promptNumberedChoice("What would you like to do?", []string{
			"Proceed -- accept charter and create colony",
			"Revise -- provide a new goal and re-scan",
			"Cancel -- stop without creating anything",
		})

		switch choice {
		case 1: // Proceed
			emitLifecycleCeremony("colony:init:charter-approved", events.CeremonyPayload{
				Task:    fmt.Sprintf("Charter approved for goal: %s", goal),
				Message: fmt.Sprintf("Charter: %s", charter.Intent),
			}, "init-ceremony")

			if err := createCeremonyColony(goal, scope, *charter); err != nil {
				outputError(1, fmt.Sprintf("failed to create colony: %v", err), nil)
				return nil
			}

			emitLifecycleCeremony("colony:init:completed", events.CeremonyPayload{
				Task:    fmt.Sprintf("Colony created for goal: %s", goal),
				Message: fmt.Sprintf("Session: %s_%d", strings.ToLower(strings.Fields(goal)[0]), time.Now().Unix()),
			}, "init-ceremony")

			return nil

		case 2: // Revise
			newGoal := promptString("Enter new goal")
			if newGoal == "" {
				fmt.Fprintln(os.Stderr, "  Goal cannot be empty, keeping current goal.")
				continue
			}
			goal = newGoal
			// Clean restart -- re-run from research with new goal
			continue

		case 3: // Cancel
			fmt.Fprintln(os.Stderr, "  Colony creation cancelled. No artifacts created.")
			return nil

		default:
			fmt.Fprintln(os.Stderr, "  Invalid choice. Please enter 1, 2, or 3.")
			continue
		}
	}
}

// runCeremonyResearch runs init-research internally and extracts charter + pheromone suggestions.
func runCeremonyResearch(goal, target string) (*colony.Charter, []pheromoneSuggestion, error) {
	// Build a temporary cobra command to run init-research's RunE
	tmpCmd := &cobra.Command{
		Use: "init-research",
	}
	tmpCmd.Flags().String("goal", "", "")
	tmpCmd.Flags().String("target", "", "")
	tmpCmd.SetArgs([]string{})

	// Capture init-research output by running it directly
	var researchResult map[string]interface{}

	// We need to call the initResearchCmd's RunE function, but we need to
	// capture its output. We'll do this by temporarily redirecting stdout.
	origStdout := stdout
	var buf *bytes.Buffer
	// If stdout is already a buffer, wrap it
	if origStdout != nil {
		buf = bytes.NewBuffer(nil)
		stdout = buf
	} else {
		buf = bytes.NewBuffer(nil)
		stdout = buf
	}

	// Reset after function
	defer func() { stdout = origStdout }()

	// Create a temporary command that mimics init-research
	researchCmd := &cobra.Command{
		Use:  "init-research",
		Args: cobra.NoArgs,
		RunE: initResearchCmd.RunE,
	}
	researchCmd.Flags().String("goal", "", "")
	researchCmd.Flags().String("target", "", "")
	_ = researchCmd.Flags().Set("goal", goal)
	_ = researchCmd.Flags().Set("target", target)

	if err := researchCmd.RunE(researchCmd, []string{}); err != nil {
		return nil, nil, fmt.Errorf("init-research failed: %w", err)
	}

	// Parse the JSON output
	output := strings.TrimSpace(buf.String())
	if output == "" {
		return nil, nil, fmt.Errorf("init-research produced no output")
	}

	// Parse the envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		return nil, nil, fmt.Errorf("failed to parse research output: %w", err)
	}

	if ok, _ := envelope["ok"].(bool); !ok {
		errMsg, _ := envelope["error"].(string)
		return nil, nil, fmt.Errorf("init-research error: %s", errMsg)
	}

	researchResult, _ = envelope["result"].(map[string]interface{})
	if researchResult == nil {
		return nil, nil, fmt.Errorf("no result in init-research output")
	}

	// Extract charter
	charterMap, ok := researchResult["charter"].(map[string]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("no charter in init-research output")
	}
	charter := extractCharterFromMap(charterMap)

	// Extract pheromone suggestions
	var suggestions []pheromoneSuggestion
	if sugList, ok := researchResult["pheromone_suggestions"].([]interface{}); ok {
		for _, s := range sugList {
			if sm, ok := s.(map[string]interface{}); ok {
				suggestions = append(suggestions, pheromoneSuggestion{
					Type:    stringOrEmpty(sm["type"]),
					Content: stringOrEmpty(sm["content"]),
					Reason:  stringOrEmpty(sm["reason"]),
				})
			}
		}
	}

	return &charter, suggestions, nil
}

// extractCharterFromMap converts a raw map to colony.Charter.
func extractCharterFromMap(m map[string]interface{}) colony.Charter {
	return colony.Charter{
		Intent:      stringOrEmpty(m["intent"]),
		Vision:      stringOrEmpty(m["vision"]),
		Governance:  stringOrEmpty(m["governance"]),
		Goals:       stringOrEmpty(m["goals"]),
		TechStack:   stringOrEmpty(m["tech_stack"]),
		KeyRisks:    stringOrEmpty(m["key_risks"]),
		Constraints: stringOrEmpty(m["constraints"]),
	}
}

// stringOrEmpty extracts a string from an interface{} or returns "".
func stringOrEmpty(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// createCeremonyColony creates the colony state, session, and artifacts
// with the approved charter.
func createCeremonyColony(goal string, scope colony.ColonyScope, charter colony.Charter) error {
	now := time.Now()
	nowStr := now.Format(time.RFC3339)

	sanitizedGoal := strings.ToLower(strings.Fields(goal)[0])
	sessionID := fmt.Sprintf("%s_%d", sanitizedGoal, now.Unix())
	runID := fmt.Sprintf("%s_%d_%s", sanitizedGoal, now.Unix(), randomHex(4))

	dataDir := store.BasePath()
	aetherDir := filepath.Dir(dataDir)

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(aetherDir, "dreams"), 0755); err != nil {
		return fmt.Errorf("failed to create directory structure: %w", err)
	}

	// Clean up any leftover worktrees
	if cleaned, orphaned, err := gcOrphanedWorktrees(); err == nil && (cleaned > 0 || orphaned > 0) {
		fmt.Fprintf(os.Stderr, "warning: cleaned %d stale worktree(s), %d orphaned\n", cleaned, orphaned)
	}
	_ = os.RemoveAll(filepath.Join(aetherDir, "worktrees"))
	_ = os.RemoveAll(filepath.Join(dataDir, "reviews"))

	// Create COLONY_STATE.json v3.0
	state := colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		Scope:         scope,
		ColonyVersion: 0,
		State:         colony.StateREADY,
		CurrentPhase:  0,
		SessionID:     &sessionID,
		RunID:         &runID,
		InitializedAt: &now,
		Plan:          colony.Plan{Phases: []colony.Phase{}},
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{},
			Decisions:      []colony.Decision{},
			Instincts:      []colony.Instinct{},
		},
		Errors:        colony.Errors{Records: []colony.ErrorRecord{}, FlaggedPatterns: []colony.FlaggedPattern{}},
		Signals:       []colony.Signal{},
		Graveyards:    []colony.Graveyard{},
		Events:        []string{},
		ParallelMode:  colony.ModeInRepo,
		Charter:       &charter,
	}

	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		return fmt.Errorf("failed to create COLONY_STATE.json: %w", err)
	}

	// Create session.json
	session := colony.SessionFile{
		SessionID:        sessionID,
		StartedAt:        nowStr,
		ColonyGoal:       goal,
		CurrentPhase:     0,
		CurrentMilestone: "",
		SuggestedNext:    "aether plan",
		ActiveTodos:      []string{},
		Summary:          "Colony initialized via ceremony",
	}

	if err := store.SaveJSON("session.json", session); err != nil {
		return fmt.Errorf("failed to create session.json: %w", err)
	}

	if _, err := syncColonyArtifacts(state, colonyArtifactOptions{
		CommandName:   "init-ceremony",
		SuggestedNext: "aether plan",
		Summary:       "Colony initialized via ceremony",
		HandoffTitle:  "Initialized Colony",
		WriteHandoff:  true,
	}); err != nil {
		return fmt.Errorf("failed to create recovery artifacts: %w", err)
	}

	// Create activity.log entry
	activityEntry := map[string]interface{}{
		"timestamp": nowStr,
		"action":    "COLONY_INITIALIZED",
		"detail":    fmt.Sprintf("goal=%q session=%s ceremony=true", goal, sessionID),
	}

	if err := store.AppendJSONL("activity.log", activityEntry); err != nil {
		return fmt.Errorf("failed to create activity.log: %w", err)
	}

	return nil
}

// isTerm checks if the file is a terminal.
func isTerm(f *os.File) bool {
	fi, _ := f.Stat()
	return (fi.Mode() & os.ModeCharDevice) != 0
}

var initCeremonyCmd = &cobra.Command{
	Use:   "init-ceremony <goal>",
	Short: "Run the full colony init ceremony (scan, charter, approve)",
	Args:  cobra.ExactArgs(1),
	RunE:  runInitCeremony,
}

func init() {
	initCeremonyCmd.Flags().String("target", ".", "Directory to scan")
	initCeremonyCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope: project or meta")
	initCeremonyCmd.Flags().Bool("non-interactive", false, "Skip terminal interaction (requires --charter-json)")
	initCeremonyCmd.Flags().String("charter-json", "", "Charter data as JSON (for non-interactive mode)")
	rootCmd.AddCommand(initCeremonyCmd)
}
