package cmd

import (
	"fmt"
	"hash/fnv"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const visualDivider = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

var casteEmojiMap = map[string]string{
	"queen":         "👑",
	"builder":       "🔨",
	"watcher":       "👁️",
	"scout":         "🔍",
	"colonizer":     "🗺️",
	"surveyor":      "📊",
	"architect":     "🏛️",
	"chaos":         "🎲",
	"archaeologist": "🏺",
	"oracle":        "🔮",
	"route_setter":  "📋",
	"ambassador":    "🔌",
	"auditor":       "👥",
	"chronicler":    "📝",
	"gatekeeper":    "📦",
	"guardian":      "🛡️",
	"includer":      "♿",
	"keeper":        "📚",
	"measurer":      "⚡",
	"probe":         "🧪",
	"tracker":       "🐛",
	"weaver":        "🔄",
	"dreamer":       "💭",
	"medic":         "🩹",
}

var casteColorMap = map[string]string{
	"queen":         "35",
	"builder":       "33",
	"watcher":       "36",
	"scout":         "32",
	"colonizer":     "34",
	"surveyor":      "34",
	"architect":     "95",
	"chaos":         "31",
	"archaeologist": "93",
	"oracle":        "35",
	"route_setter":  "94",
	"ambassador":    "96",
	"auditor":       "37",
	"chronicler":    "92",
	"gatekeeper":    "91",
	"guardian":      "96",
	"includer":      "96",
	"keeper":        "92",
	"measurer":      "93",
	"probe":         "36",
	"tracker":       "31",
	"weaver":        "95",
	"dreamer":       "90",
	"medic":         "96",
}

var casteLabelMap = map[string]string{
	"queen":         "Queen",
	"builder":       "Builder",
	"watcher":       "Watcher",
	"scout":         "Scout",
	"colonizer":     "Colonizer",
	"surveyor":      "Surveyor",
	"architect":     "Architect",
	"chaos":         "Chaos",
	"archaeologist": "Archaeologist",
	"oracle":        "Oracle",
	"route_setter":  "Route-Setter",
	"ambassador":    "Ambassador",
	"auditor":       "Auditor",
	"chronicler":    "Chronicler",
	"gatekeeper":    "Gatekeeper",
	"guardian":      "Guardian",
	"includer":      "Includer",
	"keeper":        "Keeper",
	"measurer":      "Measurer",
	"probe":         "Probe",
	"tracker":       "Tracker",
	"weaver":        "Weaver",
	"dreamer":       "Dreamer",
	"medic":         "Medic",
}

var commandEmojiMap = map[string]string{
	"init":              "🥚",
	"colonize":          "🗺️",
	"plan":              "📋",
	"build":             "🔨",
	"continue":          "👁️",
	"continue-blocked":  "⛔",
	"seal":              "🏺",
	"install":           "📦",
	"lay-eggs":          "🥚",
	"update":            "🔄",
	"pause":             "💾",
	"resume":            "💾",
	"patrol":            "📊",
	"phase":             "🧱",
	"history":           "📜",
	"spawn-plan":        "🐜",
	"swarm":             "🔥",
	"run":               "⚡",
	"oracle":            "🔮",
	"status":            "📊",
	"artifacts":         "🗂️",
	"next-up":           "🐜",
	"colonize-dispatch": "🗺️",
	"plan-dispatch":     "📋",
	"build-dispatch":    "🔨",
	"entomb":            "⚰️",
	"tunnels":           "🕳️",
	"watch":             "👁️",
	"pheromones":        "🎯",
	"flags":             "🚩",
	"focus":             "🔦",
	"redirect":          "🚫",
	"feedback":          "💬",
	"dream":             "💭",
	"chaos":             "🎲",
	"archaeology":       "🏺",
	"organize":          "🧹",
	"council":           "📜",
	"interpret":         "🔍",
	"maturity":          "👑",
	"memory-details":    "📜",
	"verify-castes":     "✓",
	"flag":              "🚩",
	"help":              "🐜",
	"preferences":       "🧠",
	"profile":           "🧠",
	"assumptions":       "📐",
	"discuss":           "💬",
	"migrate-state":     "🚚",
	"bump-version":      "🚀",
	"insert-phase":      "➕",
	"quick":             "⚡",
	"skill-create":      "🧪",
	"data-clean":        "🧹",
	"export-signals":    "📤",
	"import-signals":    "📥",
	"swarm-display":     "🔥",
}

func commandEmoji(command string) string {
	if emoji, ok := commandEmojiMap[command]; ok {
		return emoji
	}
	return "🐜"
}

func shouldRenderVisualOutput(w io.Writer) bool {
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_OUTPUT_MODE")))
	switch mode {
	case "json":
		return false
	case "visual", "human", "pretty":
		return true
	}

	if os.Getenv("AETHER_FORCE_VISUAL") == "1" {
		return true
	}

	return isTerminalWriter(w)
}

func isTerminalWriter(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}

	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

func shouldUseLiveWatchRefresh(w io.Writer, once bool) bool {
	if once || !shouldRenderVisualOutput(w) {
		return false
	}
	return isTerminalWriter(w)
}

func outputWorkflow(result interface{}, visual string) {
	if shouldRenderVisualOutput(stdout) {
		if !strings.HasSuffix(visual, "\n") {
			visual += "\n"
		}
		fmt.Fprint(stdout, visual)
		return
	}
	outputOK(result)
}

func emitVisualProgress(visual string) {
	if !shouldRenderVisualOutput(stdout) {
		return
	}
	visual = strings.TrimSpace(visual)
	if visual == "" {
		return
	}
	fmt.Fprint(stdout, visual+"\n\n")
}

func spacedTitle(title string) string {
	words := strings.Fields(strings.ToUpper(strings.TrimSpace(title)))
	if len(words) == 0 {
		return ""
	}

	rendered := make([]string, 0, len(words))
	for _, word := range words {
		letters := strings.Split(word, "")
		rendered = append(rendered, strings.Join(letters, " "))
	}
	return strings.Join(rendered, "   ")
}

func renderBanner(emoji, title string) string {
	return fmt.Sprintf("━━ %s %s ━━\n", emoji, spacedTitle(title))
}

func renderStageMarker(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return "── " + title + " ──\n"
}

func renderArtifactsSection(paths ...string) string {
	filtered := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		filtered = append(filtered, path)
	}
	if len(filtered) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("artifacts"), "Artifacts"))
	b.WriteString(visualDivider)
	for _, path := range filtered {
		b.WriteString(path)
		b.WriteString("\n")
	}
	return b.String()
}

func renderNextUp(primary string, alternatives ...string) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderBanner(commandEmoji("next-up"), "Next Up"))
	if strings.TrimSpace(primary) != "" {
		b.WriteString(primary)
		b.WriteString("\n")
	}
	for _, alt := range alternatives {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		b.WriteString("Alternative: ")
		b.WriteString(alt)
		b.WriteString("\n")
	}
	return b.String()
}

func renderContextClearGuidance() string {
	return "It's safe to clear your context now. Run `/ant:resume` to restore.\n"
}

func renderProgressSummary(current, total int) string {
	if total <= 0 {
		return "[Phase 0/0] " + generateProgressBar(0, 0, 16) + " 0%"
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	pct := current * 100 / total
	return fmt.Sprintf("[Phase %d/%d] %s %d%%", current, total, generateProgressBar(current, total, 16), pct)
}

func renderIndentedList(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.WriteString("  - ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func workflowSuggestionsForState(state colony.ColonyState) (string, []string) {
	state = normalizeLegacyColonyState(state)
	if colonyNeedsEntomb(state) {
		return `Run ` + "`aether entomb`" + ` to archive this sealed colony into chambers.`,
			[]string{`Run ` + "`aether init \"next goal\"`" + ` if you want to skip archiving and start fresh immediately.`}
	}
	if state.Paused {
		return `Run ` + "`aether resume`" + ` to restore the paused colony into a runnable state.`,
			[]string{`Run ` + "`aether status`" + ` if you only want to inspect the saved colony first.`}
	}

	if len(state.Plan.Phases) == 0 {
		return `Run ` + "`aether discuss`" + ` to capture intent clarifications before planning.`,
			[]string{
				`Run ` + "`aether plan`" + ` if you already know the tradeoffs and want phases now.`,
				`Run ` + "`aether colonize`" + ` first if you want a quick repo scan.`,
			}
	}

	switch state.State {
	case colony.StateEXECUTING, colony.StateBUILT:
		if state.State == colony.StateEXECUTING && state.BuildStartedAt == nil && state.CurrentPhase > 0 {
			return fmt.Sprintf("Run `aether build %d` to restart the interrupted phase.", state.CurrentPhase),
				[]string{`Run ` + "`aether status`" + ` if you want to inspect the saved colony first.`}
		}
		if guidance := loadActiveRecoveryGuidance(state); guidance != nil && guidance.HasTargetedRoute {
			alternatives := []string{`Run ` + "`aether status`" + ` to inspect the colony dashboard first.`}
			if guidance.Recovery.ReconcileCommand != "" && guidance.Recovery.ReconcileCommand != guidance.Next {
				alternatives = append(alternatives, `Run `+"`"+guidance.Recovery.ReconcileCommand+"`"+` if the code already landed and only needs reconciliation.`)
			}
			return fmt.Sprintf("Run `%s` to recover the blocked phase.", guidance.Next), alternatives
		}
		return `Run ` + "`aether continue`" + ` to verify the phase and advance.`,
			[]string{`Run ` + "`aether status`" + ` to inspect the colony dashboard first.`}
	case colony.StateCOMPLETED:
		return `Run ` + "`aether entomb`" + ` to archive this sealed colony into chambers.`, nil
	default:
		nextPhase := state.CurrentPhase
		if nextPhase < 1 {
			nextPhase = 1
		}
		return fmt.Sprintf("Run `aether build %d` to dispatch the next phase.", nextPhase),
			[]string{`Run ` + "`aether focus \"...\"`" + ` or ` + "`aether redirect \"...\"`" + ` if you want to steer the colony first.`}
	}
}

func renderInitVisual(goal, scope, sessionID, dataDir string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("init"), "Colony Init"))
	b.WriteString(visualDivider)
	b.WriteString(renderStageMarker("Colony"))
	b.WriteString("Queen charter accepted.\n")
	b.WriteString("Goal: ")
	b.WriteString(goal)
	b.WriteString("\n")
	b.WriteString("Scope: ")
	b.WriteString(emptyFallback(strings.TrimSpace(scope), string(colony.ScopeProject)))
	b.WriteString("\n")
	b.WriteString("Session: ")
	b.WriteString(sessionID)
	b.WriteString("\n")
	b.WriteString("Nest: ")
	b.WriteString(dataDir)
	b.WriteString("\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether discuss`"+` to lock down key clarifications before planning.`,
		`Run `+"`aether plan`"+` if you already know the tradeoffs and want the first phase map now.`,
		`Run `+"`aether colonize`"+` first if you want a quick codebase scan before planning.`,
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderColonizeVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("colonize"), "Colonize"))
	b.WriteString(visualDivider)
	b.WriteString("Territory survey complete.\n")
	b.WriteString("Root: ")
	b.WriteString(stringValue(result["root"]))
	b.WriteString("\n")
	b.WriteString("Primary type: ")
	b.WriteString(emptyFallback(stringValue(result["detected_type"]), "unknown"))
	b.WriteString("\n")
	b.WriteString("Languages: ")
	b.WriteString(renderCSV(stringSliceValue(result["languages"]), "not detected"))
	b.WriteString("\n")
	b.WriteString("Frameworks: ")
	b.WriteString(renderCSV(stringSliceValue(result["frameworks"]), "none detected"))
	b.WriteString("\n")
	b.WriteString("Domains: ")
	b.WriteString(renderCSV(stringSliceValue(result["domains"]), "none detected"))
	b.WriteString("\n")
	if stats, ok := result["stats"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Files: %d across %d directories\n", intValue(stats["files"]), intValue(stats["directories"])))
	}
	if surveyDir := strings.TrimSpace(stringValue(result["survey_dir"])); surveyDir != "" {
		b.WriteString("Survey: ")
		b.WriteString(surveyDir)
		b.WriteString("\n")
	}
	if surveyors, ok := result["surveyors"].([]interface{}); ok && len(surveyors) > 0 {
		dispatches := parseSurveyorMaps(surveyors)
		hasRealData := hasRealExecutionData(dispatches)
		dispatchMode := strings.TrimSpace(stringValue(result["dispatch_mode"]))
		if dispatchMode == "" {
			if hasRealData {
				dispatchMode = "real"
			} else {
				dispatchMode = "synthetic"
			}
		}
		b.WriteString("Dispatch: ")
		b.WriteString(humanizeDispatchMode(dispatchMode))
		b.WriteString("\n")
		if hasRealData {
			b.WriteString("\nSurveyors\n")
			b.WriteString(renderSurveyorResults(dispatches))
		} else {
			b.WriteString("\nSurveyors\n")
			for _, d := range dispatches {
				b.WriteString("  ")
				b.WriteString(casteIdentity(d.Caste))
				b.WriteString(" ")
				b.WriteString(d.Name)
				b.WriteString("  ")
				b.WriteString(d.Task)
				b.WriteString("\n")
			}
		}
	}
	if files := stringSliceValue(result["survey_files"]); len(files) > 0 {
		b.WriteString("\nReports\n")
		b.WriteString(renderIndentedList(files))
	}
	b.WriteString("\nCoordination: ")
	b.WriteString(displayDataPath("spawn-tree.txt"))
	b.WriteString("\n")
	b.WriteString(renderNextUp(`Run ` + "`aether plan`" + ` to turn this scan into a phase plan.`))
	return b.String()
}

func renderColonizeDispatchPreview(root string, dispatches []codexSurveyorDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("colonize-dispatch"), "Colonize Dispatch"))
	b.WriteString(visualDivider)
	b.WriteString("Surveyor wave dispatching.\n")
	b.WriteString("Root: ")
	b.WriteString(root)
	b.WriteString("\n\nSurveyors\n")
	for _, dispatch := range dispatches {
		b.WriteString("  ")
		b.WriteString(casteIdentity(dispatch.Caste))
		b.WriteString(" ")
		b.WriteString(dispatch.Name)
		b.WriteString("  ")
		b.WriteString(dispatch.Task)
		b.WriteString("\n")
	}
	b.WriteString("\nCoordination: ")
	b.WriteString(displayDataPath("spawn-tree.txt"))
	b.WriteString("\n")
	return b.String()
}

// parseSurveyorMaps converts a slice of surveyor result maps to codexSurveyorDispatch structs.
func parseSurveyorMaps(surveyors []interface{}) []codexSurveyorDispatch {
	dispatches := make([]codexSurveyorDispatch, 0, len(surveyors))
	for _, raw := range surveyors {
		entry, _ := raw.(map[string]interface{})
		if entry == nil {
			continue
		}
		d := codexSurveyorDispatch{
			Caste:    stringValue(entry["caste"]),
			Name:     stringValue(entry["name"]),
			Task:     stringValue(entry["task"]),
			Status:   stringValue(entry["status"]),
			Summary:  stringValue(entry["summary"]),
			Duration: floatValue(entry["duration"]),
		}
		if outputs, ok := entry["outputs"].([]interface{}); ok {
			for _, o := range outputs {
				d.Outputs = append(d.Outputs, stringValue(o))
			}
		}
		dispatches = append(dispatches, d)
	}
	return dispatches
}

// hasRealExecutionData returns true if any surveyor has a non-"spawned" status,
// indicating real worker execution data is available.
func hasRealExecutionData(dispatches []codexSurveyorDispatch) bool {
	for _, d := range dispatches {
		if d.Status != "spawned" {
			return true
		}
	}
	return false
}

// renderSurveyorResults formats surveyor execution data as a table with
// truthful worker identity, status, and worker summary context.
func renderSurveyorResults(surveyors []codexSurveyorDispatch) string {
	if len(surveyors) == 0 {
		return ""
	}
	var b strings.Builder
	completed := 0
	totalDuration := 0.0
	for _, s := range surveyors {
		status := normalizeRuntimeDispatchStatus(s.Status)
		if status == "completed" {
			completed++
		}
		fmt.Fprintf(&b, "  %s %s %s  %s", dispatchStatusIcon(status), casteIdentity(s.Caste), s.Name, status)
		if s.Duration > 0 {
			fmt.Fprintf(&b, "  %.1fs", s.Duration)
			totalDuration += s.Duration
		}
		b.WriteString("\n")
		if task := strings.TrimSpace(s.Task); task != "" {
			fmt.Fprintf(&b, "      Task: %s\n", task)
		}
		if summary := strings.TrimSpace(s.Summary); summary != "" && summary != strings.TrimSpace(s.Task) {
			fmt.Fprintf(&b, "      %s\n", summary)
		}
	}
	b.WriteString(fmt.Sprintf("\n%d/%d surveyors completed", completed, len(surveyors)))
	if totalDuration > 0 {
		b.WriteString(fmt.Sprintf(" in %.1fs", totalDuration))
	}
	b.WriteString("\n")
	return b.String()
}

// hasRealPlanningExecutionData returns true if any planning worker has a non-"spawned" status,
// indicating real worker execution data is available.
func hasRealPlanningExecutionData(dispatches []codexPlanningDispatch) bool {
	for _, d := range dispatches {
		if d.Status != "spawned" {
			return true
		}
	}
	return false
}

// parsePlanningDispatchMaps converts a slice of planning worker result maps to codexPlanningDispatch structs.
func parsePlanningDispatchMaps(dispatches []interface{}) []codexPlanningDispatch {
	parsed := make([]codexPlanningDispatch, 0, len(dispatches))
	for _, raw := range dispatches {
		entry, _ := raw.(map[string]interface{})
		if entry == nil {
			continue
		}
		d := codexPlanningDispatch{
			Caste:    stringValue(entry["caste"]),
			Name:     stringValue(entry["name"]),
			Task:     stringValue(entry["task"]),
			Status:   stringValue(entry["status"]),
			Summary:  stringValue(entry["summary"]),
			Duration: floatValue(entry["duration"]),
		}
		if outputs, ok := entry["outputs"].([]interface{}); ok {
			for _, o := range outputs {
				d.Outputs = append(d.Outputs, stringValue(o))
			}
		}
		parsed = append(parsed, d)
	}
	return parsed
}

// renderPlanningWorkerResults formats planning worker execution data as a table with
// truthful worker identity, status, and worker summary context.
func renderPlanningWorkerResults(workers []codexPlanningDispatch) string {
	if len(workers) == 0 {
		return ""
	}
	var b strings.Builder
	completed := 0
	totalDuration := 0.0
	for _, w := range workers {
		status := normalizeRuntimeDispatchStatus(w.Status)
		if status == "completed" {
			completed++
		}
		fmt.Fprintf(&b, "  %s %s %s  %s", dispatchStatusIcon(status), casteIdentity(w.Caste), w.Name, status)
		if w.Duration > 0 {
			fmt.Fprintf(&b, "  %.1fs", w.Duration)
			totalDuration += w.Duration
		}
		b.WriteString("\n")
		if task := strings.TrimSpace(w.Task); task != "" {
			fmt.Fprintf(&b, "      Task: %s\n", task)
		}
		if summary := strings.TrimSpace(w.Summary); summary != "" && summary != strings.TrimSpace(w.Task) {
			fmt.Fprintf(&b, "      %s\n", summary)
		}
	}
	b.WriteString(fmt.Sprintf("\n%d/%d workers completed", completed, len(workers)))
	if totalDuration > 0 {
		b.WriteString(fmt.Sprintf(" in %.1fs", totalDuration))
	}
	b.WriteString("\n")
	return b.String()
}

func dispatchStatusIcon(status string) string {
	switch normalizeRuntimeDispatchStatus(status) {
	case "completed", "manually-reconciled":
		return "\u2713"
	case "blocked":
		return "!"
	case "failed", "timeout", "superseded":
		return "\u2717"
	case "starting", "active", "running", "spawned":
		return "\u2026"
	default:
		return "\u2022"
	}
}

func renderPlanVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("plan"), "Plan"))
	b.WriteString(visualDivider)
	if existing, _ := result["existing_plan"].(bool); existing {
		b.WriteString("Existing colony plan loaded.\n")
	} else {
		b.WriteString("Scout and Route-Setter mapped the colony goal into executable phases.\n")
	}
	b.WriteString("Goal: ")
	b.WriteString(stringValue(result["goal"]))
	b.WriteString("\n")
	if granularity := strings.TrimSpace(stringValue(result["granularity"])); granularity != "" {
		b.WriteString("Granularity: ")
		b.WriteString(granularity)
		if min, max := intValue(result["granularity_min"]), intValue(result["granularity_max"]); min > 0 && max > 0 {
			b.WriteString(fmt.Sprintf(" (%d-%d phases)", min, max))
		}
		b.WriteString("\n")
	}
	if confidence, ok := result["confidence"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Confidence: %d%% overall\n", intValue(confidence["overall"])))
	}
	phases := phaseSliceValue(result["phases"])
	b.WriteString("Plan size: ")
	b.WriteString(fmt.Sprintf("%d phases\n\n", len(phases)))
	if warning := strings.TrimSpace(stringValue(result["clarification_warning"])); warning != "" {
		b.WriteString("Clarifications\n")
		b.WriteString(fmt.Sprintf("  - %d unresolved clarification(s)\n", intValue(result["unresolved_clarifications"])))
		b.WriteString("  - ")
		b.WriteString(warning)
		b.WriteString("\n\n")
	}
	if dispatches, ok := result["dispatches"].([]interface{}); ok && len(dispatches) > 0 {
		parsed := parsePlanningDispatchMaps(dispatches)
		hasRealData := hasRealPlanningExecutionData(parsed)
		dispatchMode := strings.TrimSpace(stringValue(result["dispatch_mode"]))
		if dispatchMode == "" && hasRealData {
			dispatchMode = "real"
		}
		if dispatchMode != "" {
			b.WriteString("Dispatch: ")
			b.WriteString(humanizeDispatchMode(dispatchMode))
			b.WriteString("\n")
		}
		b.WriteString("\nWorkers\n")
		if hasRealData {
			b.WriteString(renderPlanningWorkerResults(parsed))
		} else {
			for _, d := range parsed {
				b.WriteString("  ")
				b.WriteString(casteIdentity(d.Caste))
				b.WriteString(" ")
				b.WriteString(d.Name)
				b.WriteString("  ")
				b.WriteString(d.Task)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	if files := stringSliceValue(result["planning_files"]); len(files) > 0 {
		b.WriteString("Planning Artifacts\n")
		b.WriteString(renderIndentedList(files))
		b.WriteString("\n")
	}
	if files := stringSliceValue(result["phase_research_files"]); len(files) > 0 {
		b.WriteString("Phase Research\n")
		b.WriteString(renderIndentedList(limitStrings(files, 5)))
		if len(files) > 5 {
			b.WriteString(fmt.Sprintf("  - ... and %d more phase research files\n", len(files)-5))
		}
		b.WriteString("\n")
	}

	for _, phase := range phases {
		b.WriteString(fmt.Sprintf("Phase %d — %s\n", phase.ID, phase.Name))
		if strings.TrimSpace(phase.Description) != "" {
			b.WriteString("  ")
			b.WriteString(strings.TrimSpace(phase.Description))
			b.WriteString("\n")
		}
		for _, task := range phase.Tasks {
			taskLabel := strings.TrimSpace(ptrStr(task.ID))
			if taskLabel == "" {
				taskLabel = task.Goal
			}
			b.WriteString(fmt.Sprintf("  Task %s\n", taskLabel))
			b.WriteString("    Goal: ")
			b.WriteString(strings.TrimSpace(task.Goal))
			b.WriteString("\n")
			if len(task.DependsOn) > 0 {
				b.WriteString("    Depends on: ")
				b.WriteString(strings.Join(task.DependsOn, ", "))
				b.WriteString("\n")
			}
			if len(task.Constraints) > 0 {
				b.WriteString("    Constraints:\n")
				for _, constraint := range task.Constraints {
					b.WriteString("      - ")
					b.WriteString(constraint)
					b.WriteString("\n")
				}
			}
			if len(task.Hints) > 0 {
				b.WriteString("    Hints:\n")
				for _, hint := range task.Hints {
					b.WriteString("      - ")
					b.WriteString(hint)
					b.WriteString("\n")
				}
			}
			if len(task.SuccessCriteria) > 0 {
				b.WriteString("    Success Criteria:\n")
				for _, criterion := range task.SuccessCriteria {
					b.WriteString("      - ")
					b.WriteString(criterion)
					b.WriteString("\n")
				}
			}
		}
		if len(phase.Tasks) == 0 {
			b.WriteString("  - No explicit tasks captured for this phase.\n")
		}
		if len(phase.SuccessCriteria) > 0 {
			b.WriteString("  Phase Success Criteria:\n")
			for _, criterion := range phase.SuccessCriteria {
				b.WriteString("    - ")
				b.WriteString(criterion)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("Coordination: ")
	b.WriteString(displayDataPath("spawn-tree.txt"))
	b.WriteString("\n\n")

	nextBuild := "aether build 1"
	if nextPhase := firstBuildablePhase(phases); nextPhase > 0 {
		nextBuild = fmt.Sprintf("aether build %d", nextPhase)
	}
	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `%s` to start the next planned phase.", nextBuild),
		`Run `+"`aether focus \"...\"`"+` or `+"`aether redirect \"...\"`"+` if you want to adjust the colony before the first wave.`,
	))
	return b.String()
}

func renderPlanDispatchPreview(goal string, dispatches []codexPlanningDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("plan-dispatch"), "Plan Dispatch"))
	b.WriteString(visualDivider)
	b.WriteString("Planning worker wave dispatching.\n")
	b.WriteString("Goal: ")
	b.WriteString(goal)
	b.WriteString("\n\nWorkers\n")
	for _, dispatch := range dispatches {
		b.WriteString("  ")
		b.WriteString(casteIdentity(dispatch.Caste))
		b.WriteString(" ")
		b.WriteString(dispatch.Name)
		b.WriteString("  ")
		b.WriteString(dispatch.Task)
		b.WriteString("\n")
	}
	b.WriteString("\nCoordination: ")
	b.WriteString(displayDataPath("spawn-tree.txt"))
	b.WriteString("\n")
	return b.String()
}

func renderBuildVisual(state colony.ColonyState, phase colony.Phase) string {
	return renderBuildVisualWithDispatches(state, phase, plannedBuildDispatches(phase, state.ColonyDepth))
}

func renderBuildVisualWithDispatches(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build"), fmt.Sprintf("Build Phase %d", phase.ID)))
	b.WriteString(visualDivider)
	b.WriteString(renderProgressSummary(phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase.Name)
	b.WriteString("\n")
	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString("Objective: ")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n")
	}
	b.WriteString(renderStageMarker("Context"))
	b.WriteString(renderStageMarker("Tasks"))
	for _, task := range phase.Tasks {
		b.WriteString("  [ ] ")
		b.WriteString(strings.TrimSpace(task.Goal))
		b.WriteString("\n")
	}
	if len(phase.Tasks) == 0 {
		b.WriteString("  [ ] No explicit tasks captured for this phase.\n")
	}
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Dispatch"))
	b.WriteString(renderSpawnPlanForDispatches(dispatches))
	b.WriteString(renderArtifactsSection(
		displayDataPath(fmt.Sprintf("build/phase-%d/manifest.json", phase.ID)),
		displayDataPath("last-build-claims.json"),
		displayDataPath("spawn-tree.txt"),
	))
	b.WriteString(renderStageMarker("Verification"))
	b.WriteString("Verification happens during `aether continue`.\n")
	b.WriteString(renderStageMarker("Housekeeping"))
	b.WriteString("Signal housekeeping runs during `aether continue`.\n")
	if len(state.Plan.Phases) == phase.ID {
		b.WriteString(renderStageMarker("Colony Complete"))
		b.WriteString("This is the final phase. Seal the colony after continue.\n")
	} else {
		b.WriteString(renderStageMarker("Next Phase"))
		b.WriteString(fmt.Sprintf("Phase %d follows after continue.\n", phase.ID+1))
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether continue`"+` after the work is implemented and independently verified.`,
		`Run `+"`aether status`"+` if you want to inspect progress before advancing.`,
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderBuildDispatchPreview(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build-dispatch"), fmt.Sprintf("Build Dispatch %d", phase.ID)))
	b.WriteString(visualDivider)
	b.WriteString("Worker wave dispatching.\n")
	b.WriteString(renderProgressSummary(phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase.Name)
	b.WriteString("\n")
	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString("Objective: ")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderSpawnPlanForDispatches(dispatches))
	return b.String()
}

func renderContinueVisual(state colony.ColonyState, phase colony.Phase, housekeeping *signalHousekeepingResult, final bool, nextPhase *colony.Phase, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("continue"), "Continue"))
	b.WriteString(visualDivider)
	b.WriteString(renderStageMarker("Verification"))
	if partial, _ := result["partial_success"].(bool); partial {
		b.WriteString("Verification passed with partial operational success.\n")
	} else {
		b.WriteString("Verification pass complete.\n")
	}
	b.WriteString(fmt.Sprintf("Phase %d verified and completed: %s\n", phase.ID, phase.Name))
	renderContinueVerificationSummaryMap(&b, mapValue(result["verification"]))
	if issues := stringSliceValue(result["operational_issues"]); len(issues) > 0 {
		b.WriteString("Operational evidence\n")
		b.WriteString(renderIndentedList(issues))
	}
	b.WriteString("Workers\n")
	if closed := stringSliceValue(result["closed_workers"]); len(closed) > 0 {
		b.WriteString(renderIndentedList(closed))
	} else {
		b.WriteString("  - No workers required closing\n")
	}
	b.WriteString("Verification passed during continue\n")
	b.WriteString(renderArtifactsSection(
		displayDataPath(fmt.Sprintf("build/phase-%d/verification.json", phase.ID)),
		displayDataPath(fmt.Sprintf("build/phase-%d/gates.json", phase.ID)),
		displayDataPath(fmt.Sprintf("build/phase-%d/continue.json", phase.ID)),
		displayDataPath("spawn-tree.txt"),
	))
	renderContinueGateSummaryMap(&b, mapValue(result["gates"]))
	if closed := stringSliceValue(result["closed_workers"]); len(closed) > 0 {
		b.WriteString(fmt.Sprintf("Workers closed: %d\n", len(closed)))
	}
	b.WriteString(renderStageMarker("Housekeeping"))
	if housekeeping != nil {
		b.WriteString(fmt.Sprintf("Signals: %d active -> %d active after housekeeping\n", housekeeping.ActiveBefore, housekeeping.ActiveAfter))
		if housekeeping.Updated > 0 {
			b.WriteString(fmt.Sprintf("Expired: %d time-based, %d low-strength, %d stale continue signals\n",
				housekeeping.ExpiredByTime, housekeeping.DeactivatedByStrength, housekeeping.ExpiredWorkerContinue))
		}
	}

	if final {
		b.WriteString(renderStageMarker("Colony Complete"))
		b.WriteString("All planned phases are complete. The colony is ready for Crowned Anthill.\n")
		b.WriteString(renderNextUp(
			`Run `+"`aether seal`"+` to finalize the colony.`,
			`Run `+"`aether status`"+` if you want one last dashboard pass before sealing.`,
		))
		b.WriteString(renderContextClearGuidance())
		return b.String()
	}

	if nextPhase != nil {
		b.WriteString(renderStageMarker("Next Phase"))
		b.WriteString(fmt.Sprintf("Next phase ready: %d — %s\n", nextPhase.ID, nextPhase.Name))
	}
	nextBuild := phase.ID + 1
	if nextPhase != nil && nextPhase.ID > 0 {
		nextBuild = nextPhase.ID
	}
	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `aether build %d` to dispatch the next worker wave.", nextBuild),
		`Run `+"`aether focus \"...\"`"+` or `+"`aether feedback \"...\"`"+` if you want to steer the next phase before it starts.`,
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderContinueBlockedVisual(state colony.ColonyState, phase colony.Phase, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("continue-blocked"), "Continue Blocked"))
	b.WriteString(visualDivider)
	b.WriteString(fmt.Sprintf("Phase %d remains active: %s\n", phase.ID, phase.Name))
	renderContinueVerificationSummaryMap(&b, mapValue(result["verification"]))
	renderContinueGateSummaryMap(&b, mapValue(result["gates"]))
	if issues := stringSliceValue(result["operational_issues"]); len(issues) > 0 {
		b.WriteString("Operational issues\n")
		b.WriteString(renderIndentedList(issues))
	}
	if blockers := stringSliceValue(result["blocking_issues"]); len(blockers) > 0 {
		b.WriteString("Blocking issues\n")
		b.WriteString(renderIndentedList(blockers))
	}
	primary := `Fix the blocking issues, then run ` + "`aether continue`" + ` again.`
	if next := strings.TrimSpace(stringValue(result["next"])); next != "" {
		primary = `Run ` + "`" + next + "`" + ` to recover the blocked work.`
	}
	recovery := mapValue(result["recovery"])
	secondary := `Run ` + "`aether status`" + ` to inspect the active colony before retrying.`
	if reconcile := strings.TrimSpace(stringValue(recovery["reconcile_command"])); reconcile != "" {
		secondary = `Run ` + "`" + reconcile + "`" + ` if the code landed manually and only needs reconciliation.`
	}
	b.WriteString(renderNextUp(primary, secondary))
	return b.String()
}

func renderContinueVerificationSummaryMap(b *strings.Builder, verification map[string]interface{}) {
	if len(verification) == 0 {
		return
	}
	steps, _ := verification["steps"].([]interface{})
	passed := 0
	skipped := 0
	for _, raw := range steps {
		entry, _ := raw.(map[string]interface{})
		if skip, _ := entry["skipped"].(bool); skip {
			skipped++
			continue
		}
		if ok, _ := entry["passed"].(bool); ok {
			passed++
		}
	}
	b.WriteString(fmt.Sprintf("Verification: %d passed, %d skipped\n", passed, skipped))
	if claims, ok := verification["claims"].(map[string]interface{}); ok {
		summary := strings.TrimSpace(stringValue(claims["summary"]))
		if summary != "" {
			b.WriteString("Claims: ")
			b.WriteString(summary)
			b.WriteString("\n")
		}
	}
}

func renderContinueGateSummaryMap(b *strings.Builder, gates map[string]interface{}) {
	if len(gates) == 0 {
		return
	}
	checks, _ := gates["checks"].([]interface{})
	if len(checks) == 0 {
		return
	}
	passed := 0
	for _, raw := range checks {
		entry, _ := raw.(map[string]interface{})
		if ok, _ := entry["passed"].(bool); ok {
			passed++
		}
	}
	b.WriteString(fmt.Sprintf("Gates: %d/%d passed\n", passed, len(checks)))
}

func mapValue(raw interface{}) map[string]interface{} {
	value, _ := raw.(map[string]interface{})
	return value
}

func renderSealVisual(state colony.ColonyState, summaryPath string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("seal"), "Seal"))
	b.WriteString(visualDivider)
	b.WriteString(renderStageMarker("Summary"))
	b.WriteString("Colony sealed at Crowned Anthill.\n")
	if state.Goal != nil {
		b.WriteString("Goal: ")
		b.WriteString(*state.Goal)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("Completed phases: %d\n", len(state.Plan.Phases)))
	b.WriteString("Summary: ")
	b.WriteString(summaryPath)
	b.WriteString("\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether entomb`"+` to archive this completed colony into chambers.`,
		`Run `+"`aether init \"next goal\"`"+` if you want to start the next colony immediately.`,
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderSignalVisual(sigType, content, priority string, replaced bool) string {
	emoji := map[string]string{
		"FOCUS":    "🎯",
		"REDIRECT": "🚫",
		"FEEDBACK": "💬",
	}[sigType]
	if emoji == "" {
		emoji = "🐜"
	}

	status := "New signal laid."
	if replaced {
		status = "Existing signal reinforced."
	}

	var b strings.Builder
	b.WriteString(renderBanner(emoji, sigType+" Signal"))
	b.WriteString(visualDivider)
	b.WriteString(status)
	b.WriteString("\n")
	b.WriteString("Priority: ")
	b.WriteString(emptyFallback(priority, "normal"))
	b.WriteString("\n")
	b.WriteString("Content: ")
	b.WriteString(strings.TrimSpace(content))
	b.WriteString("\n")
	b.WriteString(renderNextUp(
		`Run `+"`aether pheromones`"+` to inspect all active colony guidance.`,
		`Run `+"`aether status`"+` to see where the signal will apply next.`,
	))
	return b.String()
}

func renderNextUpVisual(suggestions []string) string {
	primary := `Run ` + "`aether status`" + ` to inspect the colony.`
	var alts []string
	if len(suggestions) > 0 {
		primary = suggestions[0]
	}
	if len(suggestions) > 1 {
		alts = suggestions[1:]
	}
	return renderNextUp(primary, alts...)
}

func renderInstallVisual(homeDir string, results []map[string]interface{}, totalCopied, totalSkipped int, binaryMode string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("install"), "Install"))
	b.WriteString(visualDivider)
	b.WriteString("Aether hub refreshed.\n")
	b.WriteString("Home: ")
	b.WriteString(homeDir)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Assets: %d copied, %d unchanged\n\n", totalCopied, totalSkipped))
	b.WriteString(renderSyncSummary(results))
	b.WriteString("\nRuntime\n")
	for _, line := range installRuntimeLines(binaryMode) {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether lay-eggs`"+` inside a repo to set up a local nest.`,
		`Run `+"`aether update`"+` in existing repos to pull the refreshed companion files.`,
	))
	return b.String()
}

func renderSetupVisual(repoDir string, results []map[string]interface{}, totalCopied, totalSkipped int, restartTargets []string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("lay-eggs"), "Lay Eggs"))
	b.WriteString(visualDivider)
	b.WriteString("Nest prepared in this repository.\n")
	b.WriteString("Repo: ")
	b.WriteString(repoDir)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Assets: %d copied, %d unchanged\n\n", totalCopied, totalSkipped))
	b.WriteString(renderSyncSummary(results))
	if restartNote := codexRestartMessage(restartTargets); restartNote != "" {
		b.WriteString("\nCodex Refresh\n")
		b.WriteString("  ")
		b.WriteString(restartNote)
		b.WriteString("\n")
	}
	primaryNext := `Run ` + "`aether init \"your goal\"`" + ` to start a colony.`
	secondaryNext := `Run ` + "`aether colonize`" + ` after init if you want a quick territory scan before planning.`
	if len(restartTargets) > 0 {
		primaryNext = `Close this Codex chat, start a new Codex session in this repo, then run ` + "`aether init \"your goal\"`" + `.`
		secondaryNext = `After reopening Codex, run ` + "`aether colonize`" + ` if you want a quick territory scan before planning.`
	}
	b.WriteString(renderNextUp(
		primaryNext,
		secondaryNext,
	))
	return b.String()
}

func renderUpdateVisual(repoDir, hubVersion, localVersion string, force, dryRun bool, details []map[string]interface{}, totalCopied, totalSkipped int, restartTargets []string, binaryMode string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("update"), "Update"))
	b.WriteString(visualDivider)
	if dryRun {
		b.WriteString("Dry run complete. No files were changed.\n")
	} else {
		b.WriteString("Companion files refreshed from the hub.\n")
	}
	b.WriteString("Repo: ")
	b.WriteString(repoDir)
	b.WriteString("\n")
	if hubVersion != "" {
		b.WriteString("Hub version: ")
		b.WriteString(hubVersion)
		b.WriteString("\n")
	}
	if localVersion != "" {
		b.WriteString("Binary version: ")
		b.WriteString(localVersion)
		b.WriteString("\n")
	}
	mode := "safe"
	if force {
		mode = "force"
	}
	b.WriteString("Mode: ")
	b.WriteString(mode)
	b.WriteString("\n")
	if !dryRun {
		b.WriteString(fmt.Sprintf("Assets: %d copied, %d unchanged\n", totalCopied, totalSkipped))
	}
	b.WriteString("\n")
	b.WriteString(renderSyncSummary(details))
	b.WriteString("\nRuntime\n")
	for _, line := range updateRuntimeLines(binaryMode) {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	if dryRun {
		next := `Run ` + "`aether update`" + ` to apply the previewed changes.`
		alt := `Run ` + "`aether update --force`" + ` only if you want tracked companion files overwritten and stale files removed.`
		runtimeAlt := `Run ` + "`aether update --download-binary`" + ` only if you need a published runtime update as well.`
		if force {
			next = `Run ` + "`aether update --force`" + ` to apply the forced sync.`
			alt = `Run ` + "`aether update`" + ` instead for the safer non-force path.`
			runtimeAlt = `Run ` + "`aether install --package-dir <Aether checkout>`" + ` in the Aether repo first if you need an unreleased local runtime fix on this machine.`
		}
		b.WriteString(renderNextUp(next, alt, runtimeAlt))
		return b.String()
	}
	if restartNote := codexRestartMessage(restartTargets); restartNote != "" {
		b.WriteString("\nCodex Refresh\n")
		b.WriteString("  ")
		b.WriteString(restartNote)
		b.WriteString("\n")
	}
	if totalCopied == 0 && len(restartTargets) == 0 {
		b.WriteString(renderNextUp(
			`No follow-up is required. Run `+"`aether status`"+` only if you want to inspect the colony.`,
			`Run `+"`aether init \"next goal\"`"+` only if this repo does not have an active colony yet.`,
			`Run `+"`aether update --download-binary`"+` only if you explicitly need a published runtime refresh as well.`,
		))
		return b.String()
	}
	primaryNext := `Run ` + "`aether status`" + ` to inspect the colony after the refresh.`
	secondaryNext := `Run ` + "`aether init \"next goal\"`" + ` if this repo does not have an active colony yet.`
	runtimeNext := `Run ` + "`aether update --download-binary`" + ` if you also need a published runtime update.`
	if len(restartTargets) > 0 {
		primaryNext = `Close this Codex chat, start a new Codex session in this repo, then run ` + "`aether status`" + `.`
		secondaryNext = `After reopening Codex, run ` + "`aether init \"next goal\"`" + ` if this repo does not have an active colony yet.`
	}
	b.WriteString(renderNextUp(
		primaryNext,
		secondaryNext,
		runtimeNext,
	))
	return b.String()
}

func installRuntimeLines(binaryMode string) []string {
	lines := []string{
		"Other repos on this machine use the shared `aether` binary plus their own synced companion files.",
	}
	switch strings.TrimSpace(binaryMode) {
	case "release-download":
		lines = append([]string{
			"Shared binary: a published release download was requested and will run next.",
		}, lines...)
	case "local-build":
		lines = append([]string{
			"Shared binary: this install came from a source checkout, so the local `aether` binary will be rebuilt next unless `--skip-build-binary` was used.",
		}, lines...)
	default:
		lines = append([]string{
			"Shared binary: unchanged by the file-sync step unless a release download or local rebuild is requested.",
		}, lines...)
	}
	lines = append(lines,
		"`aether update` in another repo syncs companion files only; it does not publish an unreleased local runtime change.",
	)
	return lines
}

func updateRuntimeLines(binaryMode string) []string {
	switch strings.TrimSpace(binaryMode) {
	case "release-download-preview":
		return []string{
			"Companion files would be synced first, then a published release binary would be downloaded.",
			"`aether update --download-binary` only installs released runtime builds, not unreleased local source changes.",
			"For an unreleased local runtime fix on this machine, run `aether install --package-dir <Aether checkout>` in the Aether repo first.",
		}
	case "release-download":
		return []string{
			"Companion files were synced first; a published release binary will be downloaded next.",
			"`aether update --download-binary` only installs released runtime builds, not unreleased local source changes.",
		}
	default:
		return []string{
			"Binary: unchanged by `aether update`.",
			"This command syncs repo companion files from the hub; it does not publish or rebuild the shared runtime.",
			"For a published runtime update, run `aether update --download-binary`.",
			"For an unreleased local runtime fix on this machine, run `aether install --package-dir <Aether checkout>` in the Aether repo first.",
		}
	}
}

func renderBinaryActionVisual(title, message, version, path string) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚡", title))
	b.WriteString(visualDivider)
	b.WriteString(strings.TrimSpace(message))
	b.WriteString("\n")
	if strings.TrimSpace(version) != "" {
		b.WriteString("Version: ")
		b.WriteString(version)
		b.WriteString("\n")
	}
	if strings.TrimSpace(path) != "" {
		b.WriteString("Path: ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(
		`Run ` + "`aether lay-eggs`" + ` in a repo to use the refreshed binary and companion files.`,
	))
	return b.String()
}

func renderPauseVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("pause"), "Pause Colony"))
	b.WriteString(visualDivider)
	b.WriteString(renderStageMarker("Handoff"))
	b.WriteString("Colony handoff saved for later resumption.\n")
	if goal := strings.TrimSpace(stringValue(result["goal"])); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	phase := intValue(result["current_phase"])
	if phase > 0 {
		b.WriteString("Phase: ")
		b.WriteString(fmt.Sprintf("%d", phase))
		if phaseName := strings.TrimSpace(stringValue(result["phase_name"])); phaseName != "" && phaseName != "(unnamed)" {
			b.WriteString(" — ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}
	if handoffPath := strings.TrimSpace(stringValue(result["handoff_path"])); handoffPath != "" {
		b.WriteString("Handoff: ")
		b.WriteString(handoffPath)
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether resume`"+` when you want to restore the paused colony.`,
		`Run `+"`aether resume`"+` for the compact dashboard view instead.`,
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderResumeVisual(result map[string]interface{}, handoffText string, full bool) string {
	var b strings.Builder
	title := "Resume"
	if full {
		title = "Resume Colony"
	}
	b.WriteString(renderBanner(commandEmoji("resume"), title))
	b.WriteString(visualDivider)
	b.WriteString(renderStageMarker("Restored"))

	// Freshness warning for stale sessions
	if freshData, ok := result["freshness"].(map[string]interface{}); ok {
		fresh, _ := freshData["fresh"].(bool)
		if !fresh {
			ageHours := stringValue(freshData["age_hours"])
			b.WriteString(fmt.Sprintf("⚠️ Session is %s hours old — spawn state cleared\n", ageHours))
		}
	}

	// Worktree cleanup summary
	if wtGC, ok := result["worktree_gc"].(map[string]interface{}); ok {
		cleaned := intValue(wtGC["cleaned"])
		orphaned := intValue(wtGC["orphaned"])
		if cleaned > 0 {
			b.WriteString(fmt.Sprintf("🧹 %d stale worktree(s) cleaned up\n", cleaned))
		}
		if orphaned > 0 {
			b.WriteString(fmt.Sprintf("⚠️ %d worktree(s) could not be cleaned — run `aether worktree-cleanup`\n", orphaned))
		}
	}

	current, _ := result["current"].(map[string]interface{})
	goal := strings.TrimSpace(stringValue(current["goal"]))
	state := strings.TrimSpace(stringValue(current["state"]))
	phase := intValue(current["phase"])
	totalPhases := intValue(current["total_phases"])
	phaseName := strings.TrimSpace(stringValue(current["phase_name"]))

	if goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	if state != "" {
		b.WriteString("State: ")
		b.WriteString(state)
		b.WriteString("\n")
	}
	if phase > 0 || totalPhases > 0 {
		b.WriteString("Phase: ")
		if totalPhases > 0 {
			b.WriteString(fmt.Sprintf("%d/%d", phase, totalPhases))
		} else {
			b.WriteString(fmt.Sprintf("%d", phase))
		}
		if phaseName != "" && phaseName != "(unnamed)" {
			b.WriteString(" — ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}
	if parallelMode := strings.TrimSpace(stringValue(current["parallel_mode"])); parallelMode != "" {
		b.WriteString("Parallel: ")
		b.WriteString(parallelMode)
		b.WriteString("\n")
	}
	if nextPhase, ok := result["next_phase"].(map[string]interface{}); ok {
		id := intValue(nextPhase["id"])
		name := strings.TrimSpace(stringValue(nextPhase["name"]))
		if id > 0 && id != phase {
			b.WriteString("Next phase: ")
			if totalPhases > 0 {
				b.WriteString(fmt.Sprintf("%d/%d", id, totalPhases))
			} else {
				b.WriteString(fmt.Sprintf("%d", id))
			}
			if name != "" && name != "(unnamed)" {
				b.WriteString(" — ")
				b.WriteString(name)
			}
			b.WriteString("\n")
		}
	}

	var suggestedNext string
	if session, ok := result["session"].(map[string]interface{}); ok {
		if summary := strings.TrimSpace(stringValue(session["summary"])); summary != "" {
			b.WriteString("\nSession Summary\n")
			b.WriteString("  ")
			b.WriteString(summary)
			b.WriteString("\n")
		}
		if todos := stringSliceValue(session["active_todos"]); len(todos) > 0 {
			b.WriteString("\nActive Todos\n")
			for _, todo := range limitStrings(todos, 5) {
				b.WriteString("  - ")
				b.WriteString(todo)
				b.WriteString("\n")
			}
		}
		suggestedNext = strings.TrimSpace(stringValue(session["suggested_next"]))
	}

	if signals, ok := result["signals"].(map[string]interface{}); ok {
		items := stringSliceValue(signals["items"])
		b.WriteString("\nActive Signals\n")
		if len(items) == 0 {
			b.WriteString("  None\n")
		} else {
			for _, item := range limitStrings(items, 6) {
				b.WriteString("  - ")
				b.WriteString(item)
				b.WriteString("\n")
			}
		}
	}

	blockers := stringSliceValue(result["blockers"])
	b.WriteString("\nBlockers\n")
	if len(blockers) == 0 {
		b.WriteString("  None\n")
	} else {
		for _, blocker := range limitStrings(blockers, 5) {
			b.WriteString("  - ")
			b.WriteString(blocker)
			b.WriteString("\n")
		}
	}

	if survey, ok := result["survey"].(map[string]interface{}); ok {
		b.WriteString("\nSurvey Context\n")
		if surveyedAt := strings.TrimSpace(stringValue(survey["territory_surveyed"])); surveyedAt != "" {
			b.WriteString("  Surveyed: ")
			b.WriteString(surveyedAt)
			b.WriteString("\n")
		}
		files := stringSliceValue(survey["files"])
		if len(files) == 0 {
			b.WriteString("  Files: none\n")
		} else {
			b.WriteString("  Files: ")
			b.WriteString(strings.Join(limitStrings(files, 6), ", "))
			b.WriteString("\n")
		}
		if surveyPath := strings.TrimSpace(stringValue(survey["path"])); surveyPath != "" {
			b.WriteString("  Path: ")
			b.WriteString(surveyPath)
			b.WriteString("\n")
		}
	}

	if mh, ok := result["memory_health"].(map[string]interface{}); ok {
		b.WriteString("\nMemory Health\n")
		b.WriteString(fmt.Sprintf("  Wisdom: %d\n", intValue(mh["wisdom_count"])))
		b.WriteString(fmt.Sprintf("  Pending promotions: %d\n", intValue(mh["pending_promotions"])))
		b.WriteString(fmt.Sprintf("  Recent failures: %d\n", intValue(mh["recent_failures"])))
	}

	if strings.TrimSpace(handoffText) != "" {
		b.WriteString("\nHandoff\n")
		for _, line := range truncateLines(handoffText, 6) {
			if strings.TrimSpace(line) == "" {
				continue
			}
			b.WriteString("  ")
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if recovery, ok := result["recovery"].(map[string]interface{}); ok {
		source := strings.TrimSpace(stringValue(recovery["source"]))
		contextPath := strings.TrimSpace(stringValue(recovery["context_path"]))
		handoffPath := strings.TrimSpace(stringValue(recovery["handoff_path"]))
		recoverySummary := strings.TrimSpace(stringValue(recovery["summary"]))
		recoveryNext := strings.TrimSpace(stringValue(recovery["next"]))
		continueReport := strings.TrimSpace(stringValue(recovery["continue_report"]))
		if source != "" || contextPath != "" || handoffPath != "" || recoverySummary != "" || recoveryNext != "" || continueReport != "" {
			b.WriteString("\nRecovery\n")
			if source != "" {
				b.WriteString("  Source: ")
				b.WriteString(source)
				b.WriteString("\n")
			}
			if contextPath != "" {
				b.WriteString("  Context: ")
				b.WriteString(contextPath)
				b.WriteString("\n")
			}
			if handoffPath != "" {
				b.WriteString("  Handoff: ")
				b.WriteString(handoffPath)
				b.WriteString("\n")
			}
			if recoverySummary != "" {
				b.WriteString("  Summary: ")
				b.WriteString(recoverySummary)
				b.WriteString("\n")
			}
			if recoveryNext != "" {
				b.WriteString("  Next: ")
				b.WriteString(recoveryNext)
				b.WriteString("\n")
			}
			if continueReport != "" {
				b.WriteString("  Continue Report: ")
				b.WriteString(continueReport)
				b.WriteString("\n")
			}
		}
	}

	nextCommand := suggestedNext
	if nextCommand == "" && state != "" {
		nextCommand = computeNextAction(state, phase, totalPhases)
	}
	if nextCommand == "" {
		nextCommand = "aether status"
	}
	alt := "`aether memory-details`"
	if full {
		alt = "`aether resume`"
	}
	b.WriteString(renderNextUp(
		fmt.Sprintf("Run `%s` to continue from the restored colony state.", nextCommand),
		fmt.Sprintf("Run %s for additional inspection.", alt),
	))
	b.WriteString(renderContextClearGuidance())
	return b.String()
}

func renderPatrolVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("patrol"), "Patrol"))
	b.WriteString(visualDivider)
	label := strings.TrimSpace(stringValue(result["health_label"]))
	score := intValue(result["overall_health"])
	if label != "" {
		b.WriteString("Health: ")
		b.WriteString(label)
		if score > 0 {
			b.WriteString(fmt.Sprintf(" (%d)", score))
		}
		b.WriteString("\n")
	}
	if signalHealth, ok := result["signal_health"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Signals: %d active (%s)\n", intValue(signalHealth["active_count"]), stringValue(signalHealth["status"])))
	}
	if mem, ok := result["memory_pressure"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Instincts: %d (%s)\n", intValue(mem["instinct_count"]), stringValue(mem["status"])))
	}
	if errs, ok := result["error_rate"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Errors/day: %d (%s)\n", intValue(errs["errors_per_day"]), stringValue(errs["status"])))
	}
	if velocity, ok := result["build_velocity"].(map[string]interface{}); ok {
		b.WriteString(fmt.Sprintf("Build velocity: %d phases/day (%s)\n", intValue(velocity["phases_per_day"]), stringValue(velocity["trend"])))
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether status`"+` for the full colony dashboard.`,
		`Run `+"`aether pheromones`"+` or `+"`aether memory-details`"+` if you want to inspect the health inputs directly.`,
	))
	return b.String()
}

func renderPhaseVisual(result map[string]interface{}) string {
	var b strings.Builder
	number := intValue(result["number"])
	total := intValue(result["total_phases"])

	b.WriteString(renderBanner(commandEmoji("phase"), fmt.Sprintf("Phase %d", number)))
	b.WriteString(visualDivider)
	if total > 0 {
		b.WriteString(renderProgressSummary(number, total))
		b.WriteString("\n")
	}
	b.WriteString("Phase: ")
	b.WriteString(emptyFallback(stringValue(result["name"]), "(unnamed)"))
	b.WriteString("\n")
	b.WriteString("Status: ")
	b.WriteString(emptyFallback(stringValue(result["status"]), "unknown"))
	b.WriteString("\n")
	if desc := strings.TrimSpace(stringValue(result["description"])); desc != "" {
		b.WriteString("Objective: ")
		b.WriteString(desc)
		b.WriteString("\n")
	}

	taskCount := intValue(result["task_count"])
	if taskCount > 0 {
		b.WriteString(fmt.Sprintf("\nTasks (%d/%d complete, %d%%)\n",
			intValue(result["completed"]), taskCount, intValue(result["progress_pct"])))
		switch tasks := result["tasks"].(type) {
		case []map[string]interface{}:
			for _, task := range tasks {
				writePhaseTaskLine(&b, task)
			}
		case []interface{}:
			for _, raw := range tasks {
				task, _ := raw.(map[string]interface{})
				writePhaseTaskLine(&b, task)
			}
		}
	} else {
		b.WriteString("\nTasks\n")
		b.WriteString("  [ ] No tasks defined for this phase.\n")
	}

	next := fmt.Sprintf("Run `aether build %d` to dispatch this phase.", number)
	alternatives := []string{`Run ` + "`aether status`" + ` to inspect the broader colony state.`}
	status := strings.ToLower(strings.TrimSpace(stringValue(result["status"])))
	if status == "in_progress" || status == "executing" || status == "built" {
		next = `Run ` + "`aether continue`" + ` to verify and advance this phase.`
		alternatives = []string{`Run ` + "`aether history --limit 10`" + ` to inspect the recent event trail for this phase.`}
	}
	b.WriteString(renderNextUp(next, alternatives...))
	return b.String()
}

func writePhaseTaskLine(b *strings.Builder, task map[string]interface{}) {
	goal := strings.TrimSpace(stringValue(task["goal"]))
	if goal == "" {
		goal = "(unnamed task)"
	}
	status := strings.ToLower(strings.TrimSpace(stringValue(task["status"])))
	box := "[ ]"
	switch status {
	case "completed", "done":
		box = "[x]"
	case "in_progress", "executing", "running":
		box = "[>]"
	}
	b.WriteString("  ")
	b.WriteString(box)
	b.WriteString(" ")
	b.WriteString(goal)
	if id := strings.TrimSpace(stringValue(task["id"])); id != "" {
		b.WriteString("  {")
		b.WriteString(id)
		b.WriteString("}")
	}
	if status != "" {
		b.WriteString("  ")
		b.WriteString(strings.ToUpper(status))
	}
	b.WriteString("\n")
}

func renderHistoryVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("history"), "History"))
	b.WriteString(visualDivider)

	switch events := result["events"].(type) {
	case []interface{}:
		if len(events) == 0 {
			b.WriteString(emptyFallback(stringValue(result["empty_message"]), "No events recorded."))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("Recent events: %d\n", len(events)))
			if filter := strings.TrimSpace(stringValue(result["filter"])); filter != "" {
				b.WriteString("Filter: ")
				b.WriteString(filter)
				b.WriteString("\n")
			}
			b.WriteString("\n")
			for _, raw := range events {
				entry, _ := raw.(map[string]interface{})
				writeHistoryEntry(&b, entry)
			}
		}
	case []map[string]interface{}:
		if len(events) == 0 {
			b.WriteString(emptyFallback(stringValue(result["empty_message"]), "No events recorded."))
			b.WriteString("\n")
		} else {
			b.WriteString(fmt.Sprintf("Recent events: %d\n\n", len(events)))
			for _, entry := range events {
				writeHistoryEntry(&b, entry)
			}
		}
	default:
		b.WriteString(emptyFallback(stringValue(result["empty_message"]), "No events recorded."))
		b.WriteString("\n")
	}

	b.WriteString(renderNextUp(
		`Run `+"`aether status`"+` for the live colony dashboard.`,
		`Run `+"`aether phase`"+` to inspect the current phase in detail.`,
	))
	return b.String()
}

func writeHistoryEntry(b *strings.Builder, entry map[string]interface{}) {
	if entry == nil {
		return
	}
	ts := strings.TrimSpace(stringValue(entry["timestamp"]))
	msg := strings.TrimSpace(stringValue(entry["message"]))
	eventType := strings.TrimSpace(stringValue(entry["type"]))
	source := strings.TrimSpace(stringValue(entry["source"]))

	label := formatTimestamp(ts)
	if label == "" {
		label = "unknown time"
	}
	b.WriteString("• ")
	b.WriteString(label)
	if eventType != "" {
		b.WriteString("  [")
		b.WriteString(eventType)
		b.WriteString("]")
	}
	if source != "" {
		b.WriteString("  ")
		b.WriteString(source)
	}
	b.WriteString("\n")
	if msg != "" {
		b.WriteString("  ")
		b.WriteString(msg)
		b.WriteString("\n")
	}
}

func resultSignalHousekeeping(result map[string]interface{}) *signalHousekeepingResult {
	if result == nil {
		return nil
	}
	raw, ok := result["signal_housekeeping"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case signalHousekeepingResult:
		copy := v
		return &copy
	case *signalHousekeepingResult:
		return v
	case map[string]interface{}:
		h := &signalHousekeepingResult{}
		h.TotalSignals = intValue(v["total_signals"])
		h.ActiveBefore = intValue(v["active_before"])
		h.ActiveAfter = intValue(v["active_after"])
		h.ExpiredByTime = intValue(v["expired_by_time"])
		h.DeactivatedByStrength = intValue(v["deactivated_by_strength"])
		h.ExpiredWorkerContinue = intValue(v["expired_worker_continue"])
		h.Updated = intValue(v["updated"])
		if dryRun, ok := v["dry_run"].(bool); ok {
			h.DryRun = dryRun
		}
		return h
	default:
		return nil
	}
}

func renderSpawnPlan(phase colony.Phase, depth string) string {
	return renderSpawnPlanForDispatches(plannedBuildDispatches(phase, depth))
}

func renderSpawnPlanForDispatches(dispatches []codexBuildDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("spawn-plan"), "Spawn Plan"))
	b.WriteString(visualDivider)

	lastWave := 0
	for _, dispatch := range dispatches {
		if dispatch.Stage != "wave" {
			continue
		}
		if dispatch.Wave != lastWave {
			if lastWave > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("Wave %d\n", dispatch.Wave))
			lastWave = dispatch.Wave
		}
		b.WriteString("  ")
		b.WriteString(casteIdentity(dispatch.Caste))
		b.WriteString(" ")
		b.WriteString(dispatch.Name)
		b.WriteString("  ")
		b.WriteString(strings.TrimSpace(dispatch.Task))
		writeDispatchExecutionStatus(&b, dispatch)
		b.WriteString("\n")
	}
	if lastWave > 0 {
		b.WriteString("\n")
	}

	strategy := filterBuildDispatches(dispatches, "strategy")
	if len(strategy) > 0 {
		b.WriteString("Strategy\n")
		for _, dispatch := range strategy {
			b.WriteString("  ")
			b.WriteString(casteIdentity(dispatch.Caste))
			b.WriteString(" ")
			b.WriteString(dispatch.Name)
			b.WriteString("  ")
			b.WriteString(strings.TrimSpace(dispatch.Task))
			writeDispatchExecutionStatus(&b, dispatch)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	verification := filterBuildDispatches(dispatches, "verification")
	resilience := filterBuildDispatches(dispatches, "resilience")
	if len(verification) > 0 || len(resilience) > 0 {
		b.WriteString("Verification\n")
		for _, dispatch := range verification {
			b.WriteString("  ")
			b.WriteString(casteIdentity(dispatch.Caste))
			b.WriteString(" ")
			b.WriteString(dispatch.Name)
			b.WriteString("  ")
			b.WriteString(strings.TrimSpace(dispatch.Task))
			writeDispatchExecutionStatus(&b, dispatch)
			b.WriteString("\n")
		}
		for _, dispatch := range resilience {
			b.WriteString("  ")
			b.WriteString(casteIdentity(dispatch.Caste))
			b.WriteString(" ")
			b.WriteString(dispatch.Name)
			b.WriteString("  ")
			b.WriteString(strings.TrimSpace(dispatch.Task))
			writeDispatchExecutionStatus(&b, dispatch)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total planned dispatches: %d\n", len(dispatches)))
	return b.String()
}

func writeDispatchExecutionStatus(b *strings.Builder, dispatch codexBuildDispatch) {
	status := strings.TrimSpace(dispatch.Status)
	if status == "" || status == "spawned" {
		return
	}
	icon := "\u2717"
	switch status {
	case "completed":
		icon = "\u2713"
	case "blocked":
		icon = "!"
	case "active", "starting", "running":
		icon = "…"
	}
	b.WriteString("  ")
	b.WriteString(icon)
	b.WriteString(" ")
	b.WriteString(status)
	if dispatch.Duration > 0 {
		b.WriteString(fmt.Sprintf(" %.1fs", dispatch.Duration))
	}
	if summary := strings.TrimSpace(dispatch.Summary); summary != "" {
		b.WriteString("  ")
		b.WriteString(summary)
	}
	if len(dispatch.Blockers) > 0 {
		b.WriteString("  blockers: ")
		b.WriteString(strings.Join(dispatch.Blockers, "; "))
	}
}

func filterBuildDispatches(dispatches []codexBuildDispatch, stage string) []codexBuildDispatch {
	filtered := make([]codexBuildDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		if dispatch.Stage == stage {
			filtered = append(filtered, dispatch)
		}
	}
	return filtered
}

func suggestedBuildCaste(task colony.Task) string {
	text := strings.ToLower(strings.TrimSpace(task.Goal + " " + strings.Join(task.Hints, " ") + " " + strings.Join(task.SuccessCriteria, " ")))
	for _, token := range []string{"research", "investigat", "survey", "analy", "document", "readme", "spec"} {
		if strings.Contains(text, token) {
			return "scout"
		}
	}
	return "builder"
}

func taskWaves(tasks []colony.Task) [][]int {
	if len(tasks) == 0 {
		return nil
	}

	taskIDs := make([]string, len(tasks))
	indexByID := make(map[string]int, len(tasks))
	for i, task := range tasks {
		id := fmt.Sprintf("task-%d", i+1)
		if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
			id = strings.TrimSpace(*task.ID)
		}
		taskIDs[i] = id
		indexByID[id] = i
	}

	satisfied := make(map[string]bool, len(tasks))
	remaining := make(map[int]bool, len(tasks))
	for i := range tasks {
		remaining[i] = true
	}

	var waves [][]int
	for len(remaining) > 0 {
		var wave []int
		for idx := range remaining {
			if dependenciesSatisfied(tasks[idx].DependsOn, satisfied, indexByID) {
				wave = append(wave, idx)
			}
		}

		if len(wave) == 0 {
			for idx := range remaining {
				wave = append(wave, idx)
			}
		}

		sort.Ints(wave)
		for _, idx := range wave {
			delete(remaining, idx)
			satisfied[taskIDs[idx]] = true
		}
		waves = append(waves, wave)
	}

	return waves
}

func dependenciesSatisfied(dependsOn []string, satisfied map[string]bool, indexByID map[string]int) bool {
	if len(dependsOn) == 0 {
		return true
	}
	for _, dep := range dependsOn {
		dep = strings.TrimSpace(dep)
		if dep == "" || dep == "none" {
			continue
		}
		if _, known := indexByID[dep]; !known {
			continue
		}
		if !satisfied[dep] {
			return false
		}
	}
	return true
}

func deterministicAntName(caste, seed string) string {
	prefixes, ok := castePrefixes[caste]
	if !ok || len(prefixes) == 0 {
		prefixes = defaultPrefixes
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(caste + "|" + seed))
	sum := h.Sum32()
	prefix := prefixes[int(sum)%len(prefixes)]
	number := int(sum%99) + 1
	return fmt.Sprintf("%s-%d", prefix, number)
}

func normalizeCasteKey(caste string) string {
	caste = strings.ToLower(strings.TrimSpace(strings.ReplaceAll(caste, "-", "_")))
	if caste == "" {
		return ""
	}
	if _, ok := casteEmojiMap[caste]; ok {
		return caste
	}
	if strings.HasPrefix(caste, "surveyor") {
		return "surveyor"
	}
	return caste
}

func casteEmoji(caste string) string {
	caste = normalizeCasteKey(caste)
	if emoji, ok := casteEmojiMap[caste]; ok {
		return emoji
	}
	return "🐜"
}

func casteLabel(caste string) string {
	caste = normalizeCasteKey(caste)
	if label, ok := casteLabelMap[caste]; ok {
		return label
	}
	return "Ant"
}

func casteANSIColor(caste string) string {
	caste = normalizeCasteKey(caste)
	return casteColorMap[caste]
}

func casteIdentity(caste string) string {
	return casteEmoji(caste) + " " + colorizeCaste(caste, casteLabel(caste))
}

func colorizeCaste(caste, text string) string {
	if !shouldUseANSIColors() {
		return text
	}
	color := casteANSIColor(caste)
	if color == "" {
		return text
	}
	return "\x1b[" + color + "m" + text + "\x1b[0m"
}

func shouldUseANSIColors() bool {
	if strings.TrimSpace(os.Getenv("NO_COLOR")) != "" {
		return false
	}
	if strings.TrimSpace(os.Getenv("AETHER_FORCE_COLOR")) == "1" || strings.TrimSpace(os.Getenv("CLICOLOR_FORCE")) != "" {
		return true
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_OUTPUT_MODE")))
	if mode == "json" {
		return false
	}
	return shouldRenderVisualOutput(stdout)
}

func humanizeDispatchMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return ""
	}
	return strings.ToUpper(mode[:1]) + mode[1:]
}

func signalPriorityValue(sigType string) string {
	switch strings.ToUpper(strings.TrimSpace(sigType)) {
	case "FOCUS":
		return "normal"
	case "REDIRECT":
		return "high"
	case "FEEDBACK":
		return "low"
	default:
		return "normal"
	}
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func stringSliceValue(value interface{}) []string {
	switch v := value.(type) {
	case []string:
		return append([]string{}, v...)
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s := strings.TrimSpace(stringValue(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func phaseSliceValue(value interface{}) []colony.Phase {
	switch v := value.(type) {
	case []colony.Phase:
		return append([]colony.Phase{}, v...)
	case []interface{}:
		phases := make([]colony.Phase, 0, len(v))
		for _, rawPhase := range v {
			phaseMap, ok := rawPhase.(map[string]interface{})
			if !ok {
				continue
			}
			phase := colony.Phase{
				ID:              intValue(phaseMap["id"]),
				Name:            stringValue(phaseMap["name"]),
				Description:     stringValue(phaseMap["description"]),
				Status:          stringValue(phaseMap["status"]),
				SuccessCriteria: stringSliceValue(phaseMap["success_criteria"]),
			}
			if rawTasks, ok := phaseMap["tasks"].([]interface{}); ok {
				for _, rawTask := range rawTasks {
					taskMap, ok := rawTask.(map[string]interface{})
					if !ok {
						continue
					}
					var idPtr *string
					if id := strings.TrimSpace(stringValue(taskMap["id"])); id != "" {
						idPtr = &id
					}
					phase.Tasks = append(phase.Tasks, colony.Task{
						ID:              idPtr,
						Goal:            stringValue(taskMap["goal"]),
						Status:          stringValue(taskMap["status"]),
						Constraints:     stringSliceValue(taskMap["constraints"]),
						Hints:           stringSliceValue(taskMap["hints"]),
						SuccessCriteria: stringSliceValue(taskMap["success_criteria"]),
						DependsOn:       stringSliceValue(taskMap["depends_on"]),
					})
				}
			}
			phases = append(phases, phase)
		}
		return phases
	default:
		return nil
	}
}

func renderCSV(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, ", ")
}

func intValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	default:
		return 0
	}
}

func floatValue(value interface{}) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func boolValue(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

func stringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func renderSyncSummary(details []map[string]interface{}) string {
	if len(details) == 0 {
		return "No sync details recorded.\n"
	}
	var b strings.Builder
	b.WriteString("Assets\n")
	for _, entry := range details {
		label := strings.TrimSpace(stringValue(entry["label"]))
		if label == "" {
			label = "Asset"
		}
		copied := intValue(entry["copied"])
		skipped := intValue(entry["skipped"])
		removed := intValue(entry["removed"])
		b.WriteString(fmt.Sprintf("  - %s — %d copied, %d unchanged", label, copied, skipped))
		if removed > 0 {
			b.WriteString(fmt.Sprintf(", %d removed", removed))
		}
		if errorsValue, ok := entry["errors"]; ok {
			errorCount := 0
			switch errs := errorsValue.(type) {
			case []string:
				errorCount = len(errs)
			case []interface{}:
				errorCount = len(errs)
			}
			if errorCount > 0 {
				b.WriteString(fmt.Sprintf(", %d errors", errorCount))
			}
		}
		if errText := strings.TrimSpace(stringValue(entry["error"])); errText != "" {
			b.WriteString(", error: ")
			b.WriteString(errText)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	return b.String()
}

func truncateLines(text string, maxLines int) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	return append(lines[:maxLines], "...")
}
