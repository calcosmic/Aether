package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// synthesizeLaunchBrief produces a structured markdown launch brief from
// the colony goal, charter, and research data. Each section shows content
// from the charter and research data where available; sections with no data
// show "To be determined" rather than being empty.
func synthesizeLaunchBrief(goal string, charter *colony.Charter, researchData ceremonyResearchData) string {
	tbd := "To be determined"

	// Extract tech stack lines from research data
	var techLines []string
	for _, ts := range researchData.TechStackDetail {
		if ts.Language != "" {
			techLines = append(techLines, "- Language: "+ts.Language)
		}
		for _, dep := range ts.Deps {
			techLines = append(techLines, "- "+dep.Name)
		}
	}
	// Include charter tech stack if not already covered
	if charter.TechStack != "" {
		techLines = append([]string{emptyFallback(charter.TechStack, tbd)}, techLines...)
	}

	// Extract dependencies from research data
	var depLines []string
	for _, ts := range researchData.TechStackDetail {
		for _, dep := range ts.Deps {
			depLines = append(depLines, "- "+dep.Name+" ("+ts.Language+")")
		}
		for _, dep := range ts.DevDeps {
			depLines = append(depLines, "- "+dep.Name+" (dev)")
		}
	}

	// Scope from charter
	scope := emptyFallback(charter.Goals, tbd)
	// Add vision context if available
	if charter.Vision != "" {
		scope = emptyFallback(charter.Vision, tbd) + "\n\n" + scope
	}

	// Risks from charter
	risks := emptyFallback(charter.KeyRisks, tbd)
	if charter.Constraints != "" {
		risks += "\n- " + charter.Constraints
	}

	// Success criteria from charter goals
	successCriteria := emptyFallback(charter.Goals, tbd)

	// Build sections
	var b strings.Builder
	b.WriteString("# Colony Launch Brief\n\n")

	b.WriteString("## Goal\n")
	b.WriteString(emptyFallback(goal, tbd))
	b.WriteString("\n\n")

	b.WriteString("## Scope\n")
	b.WriteString(scope)
	b.WriteString("\n\n")

	b.WriteString("## Risks\n")
	b.WriteString(risks)
	b.WriteString("\n\n")

	b.WriteString("## Tech Stack\n")
	if len(techLines) > 0 {
		for _, line := range techLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(tbd)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Dependencies\n")
	if len(depLines) > 0 {
		for _, line := range depLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(tbd)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Success Criteria\n")
	b.WriteString(successCriteria)
	b.WriteString("\n")

	return b.String()
}

// runInitCeremony executes the full init ceremony flow:
// 1. Run init-research to scan and generate charter
// 2. Display charter using renderCharterDisplay
// 3. Auto-approve pheromone suggestions
// 4. Synthesize launch brief from charter + research data
// 5. Prompt user: Approve / Edit / Reject
// 6. Act on choice
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

	// In test mode (stdinReader is set), skip TTY check
	isTestMode := stdinReader != nil
	if !isTestMode && !isTerm(os.Stdin) {
		outputError(1, "init-ceremony requires an interactive terminal", nil)
		return nil
	}

	// Run the ceremony loop (supports reject-restart cycling)
	for {
		charter, pheromoneSuggestions, researchData, err := runCeremonyResearch(goal, target)
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

		// Display research data if available
		if researchOutput := renderResearchDisplay(researchData); researchOutput != "" {
			fmt.Fprint(os.Stderr, researchOutput)
		}

		// Auto-approve pheromone suggestions
		if len(pheromoneSuggestions) > 0 {
			fmt.Fprintf(os.Stderr, "\n  Auto-approved %d pheromone suggestion(s):\n", len(pheromoneSuggestions))
			for _, sug := range pheromoneSuggestions {
				fmt.Fprintf(os.Stderr, "    [%s] %s\n", sug.Type, sug.Content)
			}
		}

		// Synthesize and display launch brief (CONF-04)
		brief := synthesizeLaunchBrief(goal, charter, researchData)
		fmt.Fprint(os.Stderr, brief)

		// Brief approval flow (CONF-05) -- replaces old Proceed/Revise/Cancel
		for {
			choice := promptNumberedChoice("What would you like to do with this launch brief?", []string{
				"Approve -- accept brief and create colony",
				"Edit -- modify the brief in your editor",
				"Reject -- return to goal prompt",
			})

			switch choice {
			case 1: // Approve
				emitLifecycleCeremony("colony:init:charter-approved", events.CeremonyPayload{
					Task:    fmt.Sprintf("Launch brief approved for goal: %s", goal),
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

			case 2: // Edit
				tmpFile := filepath.Join(os.TempDir(), fmt.Sprintf("aether-launch-brief-%d.md", time.Now().UnixNano()))
				if err := os.WriteFile(tmpFile, []byte(brief), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not create temp file for editing: %v\n", err)
					continue
				}
				editor := os.Getenv("EDITOR")
				if editor == "" {
					editor = "vi"
				}
				editCmd := exec.Command(editor, tmpFile)
				editCmd.Stdin = os.Stdin
				editCmd.Stdout = os.Stdout
				editCmd.Stderr = os.Stderr
				if err := editCmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: editor exited with error: %v\n", err)
					os.Remove(tmpFile)
					continue
				}
				edited, err := os.ReadFile(tmpFile)
				os.Remove(tmpFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "  Warning: could not read edited brief: %v\n", err)
					continue
				}
				brief = string(edited)
				fmt.Fprint(os.Stderr, "\n--- Edited Launch Brief ---\n\n")
				fmt.Fprint(os.Stderr, brief)
				continue

			case 3: // Reject
				fmt.Fprintln(os.Stderr, "  Brief rejected. Returning to goal prompt.")
				goal = ""
				break

			default:
				fmt.Fprintln(os.Stderr, "  Invalid choice. Please enter 1, 2, or 3.")
				continue
			}
			break
		}

		if goal == "" {
			continue
		}
	}
}

// ceremonyResearchData holds the four research data fields extracted from the
// init-research JSON envelope for display in the init ceremony.
type ceremonyResearchData struct {
	TechStackDetail      []techStackDetail
	DirClassification    dirClassification
	GovernanceDetails    []governanceDetail
	ColonyContextSummary colonyContextSummary
}

// runCeremonyResearch runs init-research internally and extracts charter + pheromone suggestions + research data.
func runCeremonyResearch(goal, target string) (*colony.Charter, []pheromoneSuggestion, ceremonyResearchData, error) {
	// Capture init-research output by running it directly
	var researchResult map[string]interface{}

	// We need to call the initResearchCmd's RunE function, but we need to
	// capture its output. We'll do this by temporarily redirecting stdout.
	origStdout := stdout
	buf := bytes.NewBuffer(nil)
	stdout = buf

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
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("init-research failed: %w", err)
	}

	// Parse the JSON output
	output := strings.TrimSpace(buf.String())
	if output == "" {
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("init-research produced no output")
	}

	// Parse the envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("failed to parse research output: %w", err)
	}

	if ok, _ := envelope["ok"].(bool); !ok {
		errMsg, _ := envelope["error"].(string)
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("init-research error: %s", errMsg)
	}

	researchResult, _ = envelope["result"].(map[string]interface{})
	if researchResult == nil {
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("no result in init-research output")
	}

	// Extract charter
	charterMap, ok := researchResult["charter"].(map[string]interface{})
	if !ok {
		return nil, nil, ceremonyResearchData{}, fmt.Errorf("no charter in init-research output")
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

	return &charter, suggestions, extractCeremonyResearchData(researchResult), nil
}

// extractCeremonyResearchData extracts the 4 research fields from the init-research
// JSON envelope using json.Marshal/Unmarshal round-trip for type conversion.
func extractCeremonyResearchData(result map[string]interface{}) ceremonyResearchData {
	var data ceremonyResearchData

	if raw, ok := result["tech_stack_detail"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &data.TechStackDetail)
		}
	}

	if raw, ok := result["dir_classification"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &data.DirClassification)
		}
	}

	if raw, ok := result["governance_details"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &data.GovernanceDetails)
		}
	}

	if raw, ok := result["colony_context_summary"]; ok {
		if b, err := json.Marshal(raw); err == nil {
			_ = json.Unmarshal(b, &data.ColonyContextSummary)
		}
	}

	return data
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
	dataDir := store.BasePath()
	statePath := filepath.Join(dataDir, "COLONY_STATE.json")

	// Check idempotency: if COLONY_STATE.json exists, inspect it
	if _, err := os.Stat(statePath); err == nil {
		var existing colony.ColonyState
		if loadErr := store.LoadJSON("COLONY_STATE.json", &existing); loadErr == nil {
			// An entombed/reset colony clears the goal. Treat that as no active colony.
			if existing.Goal == nil || strings.TrimSpace(ptrStr(existing.Goal)) == "" || existing.State == colony.StateIDLE {
				goto createFreshColony
			}
			// If colony is sealed, check for in-progress seal (uncommitted changes)
			if existing.Milestone == "Crowned Anthill" {
				if sealInProgress(dataDir) {
					return fmt.Errorf("a seal operation appears to be in progress")
				}
				// Sealed colony with committed state -- allow overwrite (fall through)
			} else {
				// Active (non-sealed) colony -- block
				return fmt.Errorf("colony already initialized (state=%s, phase=%d)", existing.State, existing.CurrentPhase)
			}
		}
	}

createFreshColony:
	// Backup existing state before overwriting (sealed colony fresh-init)
	if _, err := os.Stat(statePath); err == nil {
		backupDir := filepath.Join(dataDir, "backups")
		if err := os.MkdirAll(backupDir, 0755); err == nil {
			backupFile := filepath.Join(backupDir, fmt.Sprintf("COLONY_STATE.pre-init-ceremony.%s.bak", time.Now().Format("20060102-150405")))
			if err := copyFile(statePath, backupFile); err == nil {
				fmt.Fprintf(os.Stderr, "warning: backed up previous colony state to %s\n", backupFile)
			}
		}
	}

	// Validate charter field lengths before saving state
	if err := validateCharterFieldLength(charter); err != nil {
		return fmt.Errorf("charter validation failed: %w", err)
	}

	now := time.Now()
	nowStr := now.Format(time.RFC3339)

	sanitizedGoal := strings.ToLower(strings.Fields(goal)[0])
	sessionID := fmt.Sprintf("%s_%d", sanitizedGoal, now.Unix())
	runID := fmt.Sprintf("%s_%d_%s", sanitizedGoal, now.Unix(), randomHex(4))

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
	Short: "Run the full colony init ceremony (scan, charter, brief, approve)",
	Args:  cobra.ExactArgs(1),
	RunE:  runInitCeremony,
}

func init() {
	initCeremonyCmd.Flags().String("target", ".", "Directory to scan")
	initCeremonyCmd.Flags().String("scope", string(colony.ScopeProject), "Colony scope: project or meta")
	rootCmd.AddCommand(initCeremonyCmd)
}
