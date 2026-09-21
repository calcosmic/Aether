package cmd

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

const visualDividerFallback = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"

func visualDividerStr() string {
	if loaded := loadVisualsConfig(); loaded != nil && loaded.VisualDivider != "" {
		return loaded.VisualDivider
	}
	return visualDividerFallback
}

var visualOutputMu sync.Mutex

const aetherWordmark = `
      █████╗ ███████╗████████╗██╗  ██╗███████╗██████╗
     ██╔══██╗██╔════╝╚══██╔══╝██║  ██║██╔════╝██╔══██╗
     ███████║█████╗     ██║   ███████║█████╗  ██████╔╝
     ██╔══██║██╔══╝     ██║   ██╔══██║██╔══╝  ██╔══██╗
     ██║  ██║███████╗   ██║   ██║  ██║███████╗██║  ██║
     ╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝
`

var casteEmojiMap = map[string]string{
	"queen":                "👑",
	"builder":              "🔨",
	"watcher":              "👁️",
	"scout":                "🔍",
	"colonizer":            "🗺️",
	"surveyor":             "📊",
	"surveyor_nest":        "📊",
	"surveyor_provisions":  "📊",
	"surveyor_disciplines": "📊",
	"surveyor_pathogens":   "📊",
	"architect":            "🏛️",
	"chaos":                "🎲",
	"archaeologist":        "🏺",
	"oracle":               "🔮",
	"route_setter":         "📋",
	"ambassador":           "🔌",
	"auditor":              "👥",
	"chronicler":           "📝",
	"gatekeeper":           "⚔️",
	"porter":               "📦",
	"guardian":             "🛡️",
	"includer":             "♿",
	"keeper":               "📚",
	"measurer":             "⚡",
	"probe":                "🧪",
	"tracker":              "🐛",
	"weaver":               "🔄",
	"dreamer":              "💭",
	"medic":                "🩹",
	"fixer":                "\U0001F527",
	// Curation ants (D-06): the 8 pkg/agent/curation ants
	// (sentinel, nurse, critic, herald, janitor, archivist, librarian,
	// scribe) plus "curator" for the aggregate orchestrator line. librarian
	// gets the 🧠 emoji D-06 calls for -- it is the identity used by the
	// phase-end learning beat.
	"sentinel":  "🚨",
	"nurse":     "🩺",
	"critic":    "🧐",
	"herald":    "📯",
	"janitor":   "🧹",
	"archivist": "🗄️",
	"librarian": "🧠",
	"scribe":    "🖋️",
	"curator":   "🖼️",
}

var casteColorMap = map[string]string{
	"queen":                "35",
	"builder":              "33",
	"watcher":              "36",
	"scout":                "32",
	"colonizer":            "34",
	"surveyor":             "34",
	"surveyor_nest":        "34",
	"surveyor_provisions":  "34",
	"surveyor_disciplines": "34",
	"surveyor_pathogens":   "34",
	"architect":            "95",
	"chaos":                "31",
	"archaeologist":        "93",
	"oracle":               "35",
	"route_setter":         "94",
	"ambassador":           "96",
	"auditor":              "37",
	"chronicler":           "92",
	"gatekeeper":           "91",
	"guardian":             "96",
	"includer":             "96",
	"keeper":               "92",
	"measurer":             "93",
	"probe":                "36",
	"tracker":              "31",
	"weaver":               "95",
	"dreamer":              "90",
	"medic":                "96",
	"fixer":                "33",
	"porter":               "96",
	// Curation ants (D-06)
	"sentinel":  "91",
	"nurse":     "92",
	"critic":    "33",
	"herald":    "94",
	"janitor":   "90",
	"archivist": "36",
	"librarian": "35",
	"scribe":    "37",
	"curator":   "93",
}

var casteLabelMap = map[string]string{
	"queen":     "Queen",
	"builder":   "Builder",
	"watcher":   "Watcher",
	"scout":     "Scout",
	"colonizer": "Colonizer",
	"surveyor":  "Surveyor",
	// The four surveyors all collapsed to the same glyph and the same word, so
	// a colonize rendered four identical "Surveyor" lines and the only thing
	// telling them apart was a generated name. They share the family glyph and
	// each says what it actually surveys.
	"surveyor_nest":        "Surveyor: architecture",
	"surveyor_provisions":  "Surveyor: dependencies",
	"surveyor_disciplines": "Surveyor: conventions",
	"surveyor_pathogens":   "Surveyor: risks",
	"architect":            "Architect",
	"chaos":                "Chaos",
	"archaeologist":        "Archaeologist",
	"oracle":               "Oracle",
	"route_setter":         "Route-Setter",
	"ambassador":           "Ambassador",
	"auditor":              "Auditor",
	"chronicler":           "Chronicler",
	"gatekeeper":           "Gatekeeper",
	"guardian":             "Guardian",
	"includer":             "Includer",
	"keeper":               "Keeper",
	"measurer":             "Measurer",
	"probe":                "Probe",
	"tracker":              "Tracker",
	"weaver":               "Weaver",
	"dreamer":              "Dreamer",
	"medic":                "Medic",
	"fixer":                "Fixer",
	"porter":               "Porter",
	// Curation ants (D-06)
	"sentinel":  "Sentinel",
	"nurse":     "Nurse",
	"critic":    "Critic",
	"herald":    "Herald",
	"janitor":   "Janitor",
	"archivist": "Archivist",
	"librarian": "Librarian",
	"scribe":    "Scribe",
	"curator":   "Curator",
}

// voiceGlyphMap is the one semantic line-type glyph table (Phase 202.1,
// CEC-09/SYN-VOICE-01): every non-caste glyph a rendered screen needs, keyed
// by the kind of fact the line states, restoring the Classic house style
// (github.com/calcosmic/Aether commit 3a5b81c2 "Open Chambers") where every
// content line opened with a glyph naming its category. Every value below is
// either lifted verbatim from a Classic display block quoted in
// .planning/phases/202.1-classic-visual-voice/202.1-RESEARCH.md section 2, or
// chosen by semantic analogy to the nearest Classic category, recorded here
// (RESEARCH.md's Open Question 2 / Assumption A2 resolution):
//
//   - task:        Classic build-full used 🐜 for both worker-tree entries and
//     completed-task lines ("🐜 {task_id}: done") -- the plain
//     ant is the generic "one unit of colony work" glyph.
//   - alternative: no Classic ancestor (Classic listed alternates under one
//     "Next Steps" glyph with each command owning its own
//     emoji). 🔀 (shuffle) reads as "a different path" without
//     colliding with `next`'s ➡️.
//   - colony:      Classic's own house symbol for the colony/its workers as a
//     whole, used throughout ("🐜 Colony Work Tree:").
//   - elapsed:     no Classic ancestor (no timing display existed in
//     February). ⏱️ is the ordinary stopwatch glyph for time.
//   - cost:        no Classic ancestor (spend/cost reporting is v1.27+). 💰
//     is the ordinary glyph for money/cost.
//   - evidence:    no Classic ancestor. 🔎 matches the existing
//     `source-check` command's magnifying glass -- "look closer".
//   - question:    no Classic ancestor. ❓ is the plain question mark.
//   - artifact:    no Classic ancestor. 🗂️ matches the existing `artifacts`
//     command glyph already used for saved output paths.
//   - requirement: no Classic ancestor. Reuses 📋, the route_setter/plan
//     planning-artifact glyph, per RESEARCH.md's own suggested
//     analogy for a planning-shaped concept with no ancestor.
//   - decision:    no Classic ancestor. 🧭 (compass) reads as "which way to
//     go" for an owner decision point.
//   - memory:      distinct from `learning` (🧠, Classic's instincts glyph):
//     📖 (open book) for "what the colony remembers about you"
//     (preferences/habits/relay notes), not learned patterns.
//   - history:     Classic reused 📜 for both `history` and `council`
//     commands (commandEmojiMap); reused here for the same
//     "past events" concept.
//   - family:      no Classic ancestor (recruitment is Phase 203). 🧬 reads
//     as "lineage" for a governed-subtree/family-tree row --
//     distinct from `task`'s plain ant, which is one unit of
//     colony work rather than a recruiting relationship.
//   - refusal:     no Classic ancestor. 🙅 (a person gesturing "no") reads
//     as "this was declined" for a recruitment refusal line --
//     distinct from `blocked`'s ⛔, which names a stalled state
//     rather than an ordinary, expected refusal (D-03: a refusal
//     is never a failure).
var voiceGlyphMap = map[string]string{
	"goal":        "👑",
	"phase":       "📍",
	"task":        "🐜",
	"focus":       "🎯",
	"avoid":       "🚫",
	"feedback":    "💬",
	"learning":    "🧠",
	"flag":        "🚩",
	"milestone":   "🏆",
	"dream":       "💭",
	"status":      "📊",
	"files":       "📁",
	"checkpoint":  "💾",
	"done":        "✅",
	"failed":      "❌",
	"next":        "➡️",
	"alternative": "🔀",
	"colony":      "🐜",
	"elapsed":     "⏱️",
	"cost":        "💰",
	"evidence":    "🔎",
	"question":    "❓",
	"warning":     "⚠️",
	"blocked":     "⛔",
	"artifact":    "🗂️",
	"requirement": "📋",
	"decision":    "🧭",
	"archive":     "⚰️",
	"memory":      "📖",
	"history":     "📜",
	"family":      "🧬",
	"refusal":     "🙅",
}

// voiceGlyph resolves a semantic line-type to its glyph, following
// commandEmoji's exact override shape: an operator override in
// loadVisualsConfig() first, then voiceGlyphMap, then the generic ant --
// never an empty string, so a missing key never produces a leading space.
func voiceGlyph(kind string) string {
	if loaded := loadVisualsConfig(); loaded != nil {
		if glyph, ok := loaded.VoiceGlyphMap[kind]; ok {
			return glyph
		}
	}
	if glyph, ok := voiceGlyphMap[kind]; ok {
		return glyph
	}
	return "🐜"
}

// voiceLine is the single funnel composing a glyph and a line of text. No
// renderer may compose a glyph and a line by string concatenation of its
// own -- every glyph-led content line goes through this function.
func voiceLine(kind, text string) string {
	return voiceGlyph(kind) + " " + text
}

// signalTypeGlyph resolves a pheromone signal type (FOCUS/REDIRECT/FEEDBACK)
// to its glyph through voiceGlyphMap, absorbing the three local literal
// `map[string]string{"FOCUS": ..., "REDIRECT": ..., "FEEDBACK": ...}`
// tables that used to be hand-rolled separately in renderSuggestedSteering,
// renderSteeringSignals and renderSignalVisual. An unknown or empty signal
// type falls back to the generic ant, exactly as the three local literals
// did.
func signalTypeGlyph(signalType string) string {
	switch strings.ToUpper(strings.TrimSpace(signalType)) {
	case "FOCUS":
		return voiceGlyph("focus")
	case "REDIRECT":
		return voiceGlyph("avoid")
	case "FEEDBACK":
		return voiceGlyph("feedback")
	default:
		return "🐜"
	}
}

var commandEmojiMap = map[string]string{
	"init":                   "🥚",
	"colonize":               "🗺️",
	"plan":                   "📋",
	"spec":                   "📜",
	"build":                  "🔨",
	"continue":               "👁️",
	"continue-blocked":       "⛔",
	"seal":                   "🏺",
	"install":                "📦",
	"lay-eggs":               "🥚",
	"update":                 "🔄",
	"pause":                  "💾",
	"resume":                 "💾",
	"patrol":                 "📊",
	"phase":                  "🧱",
	"skip-phase":             "⏭️",
	"history":                "📜",
	"spawn-plan":             "🐜",
	"swarm":                  "🔥",
	"run":                    "⚡",
	"oracle":                 "🔮",
	"status":                 "📊",
	"closeout":               "🏁",
	"ceremony":               "🐜",
	"artifacts":              "🗂️",
	"next-up":                "🐜",
	"print-next-up":          "🐜",
	"colonize-dispatch":      "🗺️",
	"plan-dispatch":          "📋",
	"build-dispatch":         "🔨",
	"entomb":                 "⚰️",
	"tunnels":                "🕳️",
	"watch":                  "👁️",
	"abandon":                "🗑️",
	"pheromones":             "🎯",
	"flags":                  "🚩",
	"focus":                  "🔦",
	"redirect":               "🚫",
	"feedback":               "💬",
	"dream":                  "💭",
	"chaos":                  "🎲",
	"archaeology":            "🏺",
	"organize":               "🧹",
	"council":                "📜",
	"interpret":              "🔍",
	"maturity":               "👑",
	"memory-details":         "📜",
	"memory-metrics":         "📈",
	"verify-castes":          "✓",
	"versions":               "🏷️",
	"flag":                   "🚩",
	"help":                   "🐜",
	"preferences":            "🧠",
	"profile":                "🧠",
	"assumptions":            "📐",
	"discuss":                "💬",
	"migrate-state":          "🚚",
	"bump-version":           "🚀",
	"insert-phase":           "➕",
	"quick":                  "⚡",
	"ask":                    "💭",
	"skill-create":           "🧪",
	"data-clean":             "🧹",
	"export-signals":         "📤",
	"import-signals":         "📥",
	"reference-index":        "📇",
	"reference-list":         "📋",
	"reference-match":        "🔍",
	"shelf":                  "📚",
	"shelf-list":             "📚",
	"shelf-add":              "📝",
	"shelf-promote":          "⬆️",
	"shelf-dismiss":          "🗑️",
	"queen-init":             "👑",
	"queen-read":             "👑",
	"queen-promote":          "👑",
	"queen-thresholds":       "📊",
	"queen-compose":          "📝",
	"queen-migrate":          "🚚",
	"queen-seed-from-hive":   "🍯",
	"queen-promote-instinct": "🧠",
	"queen-write-learnings":  "🧠",
	"charter-write":          "📜",
	"porter":                 "📦",
	"source-check":           "🔎",
	"proof":                  "🧾",
	"recipes":                "🍳",
	"swarm-display":          "🔥",
	"medic":                  "🩹",
}

type commandCeremonyLevel string

const (
	commandCeremonyLevelWorkerTheatre commandCeremonyLevel = "worker_theatre"
	commandCeremonyLevelGuidedRitual  commandCeremonyLevel = "guided_ritual"
	commandCeremonyLevelDashboard     commandCeremonyLevel = "dashboard"
	commandCeremonyLevelProgress      commandCeremonyLevel = "progress"
	commandCeremonyLevelQuiet         commandCeremonyLevel = "quiet"
)

func classifyCommandCeremonyLevel(command string) commandCeremonyLevel {
	command = strings.TrimSpace(strings.ToLower(command))
	command = strings.TrimPrefix(command, "aether ")
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return commandCeremonyLevelQuiet
	}
	command = fields[0]
	if strings.HasSuffix(command, "-finalize") {
		return commandCeremonyLevelQuiet
	}
	switch command {
	case "plan", "build", "colonize", "seal":
		return commandCeremonyLevelWorkerTheatre
	case "swarm":
		for _, field := range fields[1:] {
			switch field {
			case "--watch", "watch":
				return commandCeremonyLevelDashboard
			}
		}
		return commandCeremonyLevelWorkerTheatre
	case "init", "discuss", "oracle":
		return commandCeremonyLevelGuidedRitual
	case "status", "watch", "history", "phase", "resume":
		return commandCeremonyLevelDashboard
	case "run", "update", "publish", "continue", "install", "lay-eggs", "porter", "source-check", "bump-version":
		return commandCeremonyLevelProgress
	case "command-guide", "spawn-log", "spawn-complete", "ceremony", "completion", "version", "generate-progress-bar", "version-check-cached":
		return commandCeremonyLevelQuiet
	default:
		return commandCeremonyLevelQuiet
	}
}

func commandEmoji(command string) string {
	if loaded := loadVisualsConfig(); loaded != nil {
		if emoji, ok := loaded.CommandEmojiMap[command]; ok {
			return emoji
		}
	}
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
		writeVisualOutput(stdout, visual)
		return
	}
	outputOK(projectPlanningWorkflowResult(result))
}

// currentStreamingCommand is the top-level command of this invocation, set by
// the root PersistentPreRunE. The streaming emitters consult its ceremony
// class: classifyCommandCeremonyLevel was written as the streaming taxonomy
// and had zero production callers until this gate landed — the exact
// built-but-never-called failure this repo documents.
var currentStreamingCommand string

// streamingAllowedForCurrentCommand: quiet-classified commands (finalizers,
// plumbing subcommands) never stream progress; every other ceremony level may.
// The classifier returns quiet for unknown commands too, so the gate silences
// only the surfaces the taxonomy explicitly names — an unlisted subcommand
// that legitimately prints (e.g. a monitor's stale-worker warning) keeps its
// voice.
func streamingAllowedForCurrentCommand() bool {
	name := strings.TrimSpace(currentStreamingCommand)
	if name == "" {
		return true
	}
	if classifyCommandCeremonyLevel(name) != commandCeremonyLevelQuiet {
		return true
	}
	return !isExplicitlyQuietCommand(name)
}

func isExplicitlyQuietCommand(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if strings.HasSuffix(name, "-finalize") {
		return true
	}
	switch name {
	case "command-guide", "spawn-log", "spawn-complete", "ceremony", "completion", "version", "generate-progress-bar", "version-check-cached":
		return true
	}
	return false
}

// emitVisualLine writes a single progress line with one trailing newline, so
// repeated calls stack as readable scrollback. emitVisualProgress adds a blank
// line after each block, which is right for banners and wrong for a run that
// emits fifty rounds.
func emitVisualLine(line string) {
	if !shouldRenderVisualOutput(stdout) || !streamingAllowedForCurrentCommand() {
		return
	}
	line = strings.TrimRight(line, "\n")
	if strings.TrimSpace(line) == "" {
		return
	}
	writeVisualOutput(stdout, line+"\n")
}

// --- Inline recruitment lines (203-14, D-04/D-06/D-08) ---
//
// A recruit joining, a recruitment refused, and a note that changed a
// decision mid-run each print exactly one line through emitVisualLine --
// the SAME inline funnel worker start/finish lines already print through
// (cmd/codex_build_progress.go's emitCodexDispatchWorkerStarted/Finished),
// never a separate watch process -- the owner's own standing instruction is
// that colony liveness is never rendered anywhere but the one terminal they
// are already looking at (SEE-03). Every line
// reuses casteIdentity/casteLabel so a recruit looks like every other
// worker in the colony's own voice, and the cost figure always comes from
// the spend ledger authority (cmd/recruitment_subtree.go's
// recruitmentInlineCostFigure), never the event payload.

// emitInlineRecruitLine prints one line naming the caste, the deterministic
// worker name, the stated reason and the run cost so far, at the moment a
// recruitment is admitted.
func emitInlineRecruitLine(caste, workerName, reason, costFigure string) {
	emitVisualLine(renderInlineRecruitLine(caste, workerName, reason, costFigure))
}

func renderInlineRecruitLine(caste, workerName, reason, costFigure string) string {
	var b strings.Builder
	b.WriteString("+ ")
	b.WriteString(casteIdentity(caste))
	b.WriteString(" ")
	b.WriteString(workerName)
	b.WriteString(" joined")
	if reason = strings.TrimSpace(reason); reason != "" {
		b.WriteString(" -- ")
		b.WriteString(reason)
	}
	b.WriteString(fmt.Sprintf("  (cost so far: %s)", costFigure))
	return b.String()
}

// emitInlineRefusalLine prints one line naming the caste, the reason class
// and a plain sentence that the worker is carrying on alone, at the moment
// a recruitment is refused (D-03/D-06: a refusal is never a failure).
func emitInlineRefusalLine(caste, reasonClass, detail string) {
	emitVisualLine(renderInlineRefusalLine(caste, reasonClass, detail))
}

func renderInlineRefusalLine(caste, reasonClass, detail string) string {
	var b strings.Builder
	b.WriteString("x ")
	b.WriteString(casteLabel(caste))
	b.WriteString(" recruitment refused")
	if reasonClass = strings.TrimSpace(reasonClass); reasonClass != "" {
		b.WriteString(fmt.Sprintf(" (%s)", reasonClass))
	}
	if detail = strings.TrimSpace(detail); detail != "" {
		b.WriteString(": ")
		b.WriteString(detail)
	}
	b.WriteString(" -- the worker is carrying on alone")
	return b.String()
}

// emitInlineDecisionChangedLine prints one line naming the note and the
// decision it changed, at the moment the credit record recording that
// change is written (D-08's first half; the closing list is
// cmd/recruitment_subtree.go's renderNotesThatChangedDecisions).
func emitInlineDecisionChangedLine(contributionID, changedDecisionID, outcome string) {
	emitVisualLine(renderInlineDecisionChangedLine(contributionID, changedDecisionID, outcome))
}

func renderInlineDecisionChangedLine(contributionID, changedDecisionID, outcome string) string {
	return fmt.Sprintf("A note changed a decision: %s changed %s (%s)", contributionID, changedDecisionID, outcome)
}

func emitVisualProgress(visual string) {
	if !shouldRenderVisualOutput(stdout) || !streamingAllowedForCurrentCommand() {
		return
	}
	visual = strings.TrimSpace(visual)
	if visual == "" {
		return
	}
	writeVisualOutput(stdout, visual+"\n\n")
}

// writeVisualOutput is the single exit for every byte of human-facing visual
// output — banners, workflow renders, progress, and the visual error branch of
// outputError. Command naming is translated here rather than at the call sites
// because there is no reliable way to keep ~500 prose strings scattered across
// cmd/ individually correct: renderNextUp translated its own hints for months
// while the error path two functions away, the welcome banner, the recovery
// snapshot and every `Run \`aether plan\` first` message did not, so a Claude
// Code user was still told to type commands that only exist inside the wrapper.
//
// Translating at the exit makes the raw form structurally unable to reach a
// terminal on a slash-command platform, whatever new prose gets added later.
// TestVisualOutputNeverLeaksRawWrapperCommands locks that.
func writeVisualOutput(w io.Writer, text string) {
	// Gated on visual mode, not applied unconditionally. Some callers (publish
	// warnings, seal guidance) emit in both modes, and JSON is the machine
	// surface: a wrapper reading `next` out of an envelope has to receive a
	// command it can exec. Translation is a presentation concern and belongs
	// only on the presentation path.
	if shouldRenderVisualOutput(w) {
		text = translateHintCommandsForPlatform(text, detectPlatform())
	}
	visualOutputMu.Lock()
	defer visualOutputMu.Unlock()
	fmt.Fprint(w, text)
}

// visualFprint, visualFprintf and visualFprintln are drop-in replacements for
// the fmt equivalents at human-facing call sites. They exist so a renderer that
// builds its output inline — rather than assembling one string and handing it
// to writeVisualOutput — still gets platform command naming.
//
// Use these for anything a person reads. Machine surfaces (JSON envelopes,
// NDJSON streams, XML exports, worker briefs, raw worker output under
// --verbose) must keep using fmt directly, and are listed with their reasons in
// visualWriterExemptions.
func visualFprint(w io.Writer, a ...interface{}) {
	writeVisualOutput(w, fmt.Sprint(a...))
}

func visualFprintf(w io.Writer, format string, a ...interface{}) {
	writeVisualOutput(w, fmt.Sprintf(format, a...))
}

func visualFprintln(w io.Writer, a ...interface{}) {
	writeVisualOutput(w, fmt.Sprintln(a...))
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

// isAetherBannerLine reports whether a single line of rendered output is one
// of renderBanner's own banner lines -- "━━ <emoji> <S P A C E D   T I T L E>
// ━━" -- as opposed to the plain divider line (all ━ characters, no leading
// "━━ " marker) that renderBanner's callers usually print immediately below
// it. It is the ONE shared predicate for what a banner line looks like:
// TestBannerPredicateMatchesTheRenderer asserts every renderBanner output's
// first line satisfies it, and the Stop-hook screen check (cmd/hook_cmds.go)
// reuses it to decide which lines of a captured screen the owner is owed --
// so the renderer and the checkpoint cannot silently drift apart.
func isAetherBannerLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "━━ ") || !strings.HasSuffix(trimmed, " ━━") {
		return false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(trimmed, "━━ "), " ━━")
	return strings.TrimSpace(inner) != ""
}

func renderAetherWordmark() string {
	wordmark := strings.Trim(aetherWordmark, "\n")
	if loaded := loadVisualsConfig(); loaded != nil && loaded.AetherWordmark != "" {
		wordmark = strings.Trim(loaded.AetherWordmark, "\n")
	}
	if wordmark == "" {
		return ""
	}
	if shouldUseANSIColors() {
		return "\x1b[96m" + wordmark + "\x1b[0m\n\n"
	}
	return wordmark + "\n\n"
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
	b.WriteString(visualDividerStr())
	for _, path := range filtered {
		b.WriteString(path)
		b.WriteString("\n")
	}
	return b.String()
}

func renderNextUp(primary string, alternatives ...string) string {
	platform := detectPlatform()
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderBanner(commandEmoji("next-up"), "Next Up"))
	if strings.TrimSpace(primary) != "" {
		b.WriteString(voiceLine("next", translateHintCommandsForPlatform(primary, platform)))
		b.WriteString("\n")
	}
	for _, alt := range alternatives {
		alt = strings.TrimSpace(alt)
		if alt == "" {
			continue
		}
		b.WriteString(voiceLine("alternative", translateHintCommandsForPlatform(alt, platform)))
		b.WriteString("\n")
	}
	return b.String()
}

// hintCommandRe matches an `aether <verb>` mention, capturing any preceding
// VAR=value assignment so literal shell invocations can be left alone.
// Quoted argument spans are matched separately and retained byte for byte.
var hintCommandRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*=(?:'[^']*'|"(?:\\.|[^"\\])*"|\S*)\s+)?\baether ([a-z][a-z0-9-]*)|(?:^|[\s=(])(?:'[^']*'|"(?:\\.|[^"\\])*")`)

// translateHintCommandsForPlatform rewrites next-step hints so they name the
// command the user actually types. In Claude Code and OpenCode the lifecycle
// commands are slash wrappers, so "Run `aether continue`" is not a command the
// user can run — it is the runtime describing itself to itself.
//
// Only available public skills or wrapperCommandNames are rewritten; `aether publish`,
// `aether host plan`, `aether flag-resolve` and friends have no wrapper and
// must survive verbatim. Invocations carrying an env prefix
// (AETHER_OUTPUT_MODE=visual aether ...) are literal shell commands wrappers
// execute, never something the user types, so they are left alone too.
func translateHintCommandsForPlatform(s, platform string) string {
	return hintCommandRe.ReplaceAllStringFunc(s, func(match string) string {
		groups := hintCommandRe.FindStringSubmatch(match)
		if len(groups) != 3 || groups[2] == "" {
			return match
		}
		if strings.TrimSpace(groups[1]) != "" {
			return match
		}
		return platformCommandName(groups[2], platform)
	})
}

// platformCommandName returns the way a user on this platform types a runtime
// verb: the available Codex skill or slash wrapper, the raw CLI form otherwise. Use it
// when building a command name for layout (padding, tables) — plain prose can
// just say `aether <verb>` and let writeVisualOutput translate it.
func platformCommandName(verb, platform string) string {
	if platform == "codex" {
		for _, command := range codexPublicSkillCommands() {
			if command == verb {
				return "$ant-" + command
			}
		}
	} else if wrapperCommandNames[verb] {
		return "/ant-" + verb
	}
	return "aether " + verb
}

func renderContextClearGuidance() string {
	return renderContextClearGuidanceForPlatform(detectPlatform())
}

func detectPlatform() string {
	if platform := strings.TrimSpace(os.Getenv("AETHER_PLATFORM")); platform != "" {
		return platform
	}
	// The runtime's own dispatch-layer detector knows OpenCode and Codex from a
	// wider set of signals; prefer it over the narrow env checks below so a
	// Codex or OpenCode session is not mistaken for Claude and shown /ant-*
	// commands it does not have.
	switch codex.DetectActivePlatform() {
	case codex.PlatformCodex:
		return "codex"
	case codex.PlatformOpenCode:
		return "opencode"
	}
	if os.Getenv("CODEX_CLI") != "" || os.Getenv("CODEX_API_KEY") != "" || os.Getenv("CODEX_HOME") != "" {
		return "codex"
	}
	return "claude"
}

func renderContextClearGuidanceForPlatform(platform string) string {
	// SEE criterion: never advise clearing context without confirming the
	// handoff is actually on disk. "Safe to clear" is a CLAIM — it may only
	// appear when the handoff file verifiably exists; when it does not, the
	// honest line is "don't clear yet", not a softer version of safe.
	if handoffPath := filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md"); !fileExists(handoffPath) {
		switch platform {
		case "codex":
			return "Handoff not confirmed on disk — don't clear your context yet. Run `aether status` first.\n"
		default:
			return "Handoff not confirmed on disk — don't clear your context yet. Run `/ant-status` first.\n"
		}
	}
	confirmation := "Handoff saved (.aether/HANDOFF.md) — safe to clear your context now."
	switch platform {
	case "codex":
		return confirmation + " Run `aether resume` to restore.\n"
	default:
		return confirmation + " Run `/ant-resume` to restore.\n"
	}
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

// ---------------------------------------------------------------------------
// The one answer, reshaped for the callers that already existed
// ---------------------------------------------------------------------------
//
// Phase 197 plan 02. Four functions used to decide the next step from the same
// saved state and disagreed with one another; every closing message inherited
// whichever one it happened to call. They are adapters now: they gather the
// input, ask resolveNextAction, and reshape the answer into the return type
// their callers already expect. TestEveryDeciderAgreesOnTheNextCommand drives
// all four over one state and fails if they separate again.

// nextActionInputForState gathers the facts the one decision needs about a
// state the caller already holds.
//
// It is deliberately NOT loadNextActionInput: these callers pass an explicit
// state -- sometimes an in-memory one that has not been saved yet -- so the
// state must come from the argument while everything else is read from disk.
// Most facts the decision does not itself consult (e.g. outstanding task
// goals) are still left out here as display-only extras that belong to a
// different card. Signals is the one exception (198.1-04): every caller of
// this function is the CLOSING card a finished command prints, and a signal
// written moments earlier in that same run (a phase-completion note, an
// auto-REDIRECT) must show up in it -- TestWorkLoopCardsComeFromTheResolver
// fails the moment this card and a freshly-resolved one disagree on what is
// active.
func nextActionInputForState(state colony.ColonyState, lastCommand string) nextActionInput {
	state = normalizeLegacyColonyState(state)
	in := nextActionInput{
		State:       state,
		LastCommand: strings.TrimSpace(lastCommand),
		NoColony:    colonyStateIsUnstarted(state),
	}
	if in.NoColony {
		return in
	}
	if blocker, ok := activePlanFinalizeFailureFlag(store); ok {
		found := blocker
		in.PlanBlocker = &found
	}
	in.Recovery = loadActiveRecoveryGuidance(state)
	in.HandoffExists = fileExists(handoffDocumentPath())
	in.BuildLooksAbandoned = buildLooksAbandoned(state)
	// Signals WAS display-only and intentionally omitted here (198.1-04 found
	// this was stale: an active pheromone signal -- e.g. the phase-completion
	// note runPhaseEndConsolidation now emits -- never reached the shared
	// closing card's own "Your standing instructions" section for ANY caller
	// of this function, only a freshly-resolved re-read of disk would show
	// it. TestWorkLoopCardsComeFromTheResolver exists precisely to catch a
	// printed card disagreeing with what the resolver would say right now;
	// one JSON read is not the "wasted reads" this file's doc comment
	// originally guarded against, since every caller here runs once per
	// finished command, not in a hot loop.
	in.Signals = extractSignalTexts(8)
	return in
}

// colonyStateIsUnstarted reports a state that records neither a goal nor any
// planning context. That is a leftover file, not a project, and the honest
// advice for it is "start one" -- the branch nextCommandFromState used to hold.
// A state that HAS phases or a current phase is a real project even when the
// goal line is missing, so it keeps its lifecycle advice.
func colonyStateIsUnstarted(state colony.ColonyState) bool {
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		return false
	}
	return len(state.Plan.Phases) == 0 && state.CurrentPhase < 1
}

// nextActionSuggestionLine renders one runtime command with the plain-English
// reason for it. The command stays in its platform-neutral runtime form;
// translateHintCommandsForPlatform rewrites it on the way to the terminal.
// nextActionSuggestionBody composes a next-step line WITHOUT its glyph, so a
// caller that puts its own label in front (Choice:, Alternative:) can lead the
// whole line with the glyph instead of wedging it between the label and the
// command. Keeping the command text adjacent to its label is what the existing
// next-action guardrails assert (golden_workflow_test.go).
func nextActionSuggestionBody(command, explanation string) string {
	command = strings.TrimSpace(command)
	explanation = strings.TrimSpace(explanation)
	switch {
	case command == "" && explanation == "":
		return ""
	case command == "":
		return explanation
	case explanation == "":
		return "Run `" + command + "`"
	default:
		return "Run `" + command + "` — " + explanation
	}
}

func nextActionSuggestionLine(command, explanation string) string {
	body := nextActionSuggestionBody(command, explanation)
	if body == "" {
		return ""
	}
	return voiceLine("next", body)
}

// nextActionPrimarySuggestion is the recommendation as one sentence, WITHOUT a
// glyph. It feeds machine-readable result["next"] fields (closeout_cmd.go,
// codex_plan.go) as well as the visual renderers, and JSON stays raw -- the
// glyph is applied by renderNextUp, the one visual funnel, never here.
func nextActionPrimarySuggestion(answer nextAction) string {
	return nextActionSuggestionBody(answer.Command, answer.Recommendation)
}

// nextActionAlternativeSuggestions is the same shape for the other ways
// forward, and is unglyphed for the same reason.
func nextActionAlternativeSuggestions(answer nextAction) []string {
	suggestions := make([]string, 0, len(answer.Alternatives))
	for _, alternative := range answer.Alternatives {
		if line := nextActionSuggestionBody(alternative.Command, alternative.Explanation); line != "" {
			suggestions = append(suggestions, line)
		}
	}
	return suggestions
}

func workflowSuggestionsForState(state colony.ColonyState) (string, []string) {
	answer := resolveNextAction(nextActionInputForState(state, ""))
	return nextActionPrimarySuggestion(answer), nextActionAlternativeSuggestions(answer)
}

// ---------------------------------------------------------------------------
// The one closing, for the lifecycle commands (Phase 197 plan 04)
// ---------------------------------------------------------------------------
//
// Seven commands used to end with their own hand-written block of advice, so a
// wording fix had to land seven times and usually landed in one. These three
// helpers are the whole migration: a command resolves ONE answer, renders it as
// the card the owner reads, and folds the same answer into the machine-readable
// result a wrapper reads. Screen and envelope cannot disagree because there is
// only one answer to disagree about.

// lifecycleNextAction resolves the closing answer from the project as it stands
// on disk, for a command that has just finished. Read-only: it never writes to
// saved state (that is asserted of the loader it calls, not promised here).
func lifecycleNextAction(lastCommand string) nextAction {
	return resolveNextAction(loadNextActionInputForCommand(lastCommand))
}

// lifecycleNextActionForState is the same for a caller that already holds the
// state -- sometimes one it has only just written, sometimes one still in
// memory. override, when non-empty, is the command that caller knows is right
// for this exact run; it is fed to the decision rather than applied after it,
// which is what stops the screen and the envelope naming different commands.
func lifecycleNextActionForState(state colony.ColonyState, lastCommand, override, why string) nextAction {
	in := nextActionInputForState(state, lastCommand)
	if command := strings.TrimSpace(override); command != "" {
		in.Override = &nextActionOverride{Command: command, Recommendation: strings.TrimSpace(why)}
	}
	return resolveNextAction(in)
}

// applyLifecycleNextAction resolves the closing answer for a finished command
// and folds it into the result map the wrapper and the TS host read. It returns
// the answer so the caller can render the very same one on screen.
//
// The existing `next` key is left exactly as it was: something downstream is
// already reading it, and migrating a reader is not the same job as breaking
// one.
func applyLifecycleNextAction(result map[string]interface{}, state colony.ColonyState, lastCommand, override, why string) nextAction {
	answer := lifecycleNextActionForState(state, lastCommand, override, why)
	applyNextActionToResult(result, answer)
	return answer
}

// closeLifecycleCommand is applyLifecycleNextAction for a command that has
// already saved its work: the project on disk IS the state, so the answer is
// resolved from there. Call it immediately before handing the result to the
// output path, so the card on screen and the fields in the envelope come from
// one resolve rather than two.
func closeLifecycleCommand(result map[string]interface{}, lastCommand, override, why string) nextAction {
	in := loadNextActionInputForCommand(lastCommand)
	if command := strings.TrimSpace(override); command != "" {
		in.Override = &nextActionOverride{Command: command, Recommendation: strings.TrimSpace(why)}
	}
	answer := resolveNextAction(in)
	applyNextActionToResult(result, answer)
	return answer
}

// The plain-English reasons behind a run's own more-specific command. They are
// constants because the SAME sentence has to reach the card and the
// machine-readable answer; two copies of it are two things that can drift.
const (
	nextActionUnfinishedWorkWhy = "Some of this phase was finished and some was never started. This starts only the " +
		"work still listed as to do, and does not redo anything already proven."
	nextActionBlockedCheckWhy = "The check stopped on the problems listed above. This is the exact command it named " +
		"for clearing them, which is more targeted than a general re-check."
	nextActionOpenQuestionWhy = "There are questions waiting on you that this run could not answer for you. " +
		"Answering them is what unblocks it."
)

// lifecycleOverrideFromResult is the command a finished run knows is right for
// this exact situation, when it has one. Three cases qualify, and only three:
// the redispatch that picks up the half of a phase that was never started, the
// exact command a blocked check named for clearing itself, and the question
// that has to be answered before this run can usefully be repeated. None of
// them can be worked out from the saved project alone.
//
// The ordinary "check it next" is NOT special knowledge and is deliberately not
// returned here: leaving it to the one decision is what stops a paused or
// failed project being told to carry on regardless, which is the drift this
// phase exists to close.
func lifecycleOverrideFromResult(result map[string]interface{}) (string, string) {
	if result == nil {
		return "", ""
	}
	if command := strings.TrimSpace(stringValue(result["recovery_command"])); command != "" {
		return command, nextActionUnfinishedWorkWhy
	}
	// The resume dashboard's own two special cases: a durable worker result
	// waiting to be finalized, or a build process it has verified is still
	// running. Neither can be worked out from the saved project alone.
	if command := strings.TrimSpace(stringValue(result["resume_override_command"])); command != "" {
		why := strings.TrimSpace(stringValue(result["resume_override_why"]))
		if why == "" {
			why = nextActionUnfinishedWorkWhy
		}
		return command, why
	}
	if guidance, ok := result["orchestrator_boundary_guidance"].(orchestratorBoundaryGuidance); ok && guidance.Active {
		if command := strings.TrimSpace(guidance.Next); command != "" {
			return command, nextActionOpenQuestionWhy
		}
	}
	if blocked, _ := result["blocked"].(bool); blocked {
		if command := strings.TrimSpace(stringValue(result["next"])); strings.HasPrefix(command, "aether ") {
			return command, nextActionBlockedCheckWhy
		}
	}
	return "", ""
}

// lifecycleCommandInProse pulls a runnable command out of a sentence a finished
// step wrote for itself -- "Run `aether build 1 --force` after fixing the
// blocked worker output" and friends.
//
// A command carrying a fill-in-the-blank (`<file>`) is deliberately NOT
// returned: it is not something the owner can type, and recommending one is the
// defect plan 197-02 found and fixed in the closeout's own placeholder command.
// Those sentences are still shown, as a report of what the step said, above the
// card rather than in place of it.
func lifecycleCommandInProse(text string) string {
	match := lifecycleProseCommandRe.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	command := strings.TrimSpace(match[1])
	if strings.ContainsAny(command, "<>") {
		return ""
	}
	return command
}

var lifecycleProseCommandRe = regexp.MustCompile("`(aether [^`]+)`")

// closeLifecycleRun folds the one closing answer into a finished run's result
// map, feeding that run's own more-specific command (when it has one) to the
// decision rather than writing it over the answer afterwards.
func closeLifecycleRun(result map[string]interface{}, state colony.ColonyState, lastCommand string) nextAction {
	override, why := lifecycleOverrideFromResult(result)
	return applyLifecycleNextAction(result, state, lastCommand, override, why)
}

// renderLifecycleClosing is the closing block for a command whose result map
// already carries the answer it resolved. A caller that has not folded one in
// -- an older path, or a test driving the renderer directly -- gets the answer
// resolved from the project on disk instead, so the card is never silently
// absent.
func renderLifecycleClosing(result map[string]interface{}, lastCommand string) string {
	if answer, ok := nextActionFromResult(result); ok {
		return renderNextActionCard(answer)
	}
	return renderNextActionCard(lifecycleNextAction(lastCommand))
}

// renderLifecycleClosingForState is renderLifecycleClosing for a renderer that
// was also handed the project state. When the result map carries no answer --
// an older path, or a test driving the renderer directly -- the state in hand is
// a better source than the project on disk, which may not have been written yet.
func renderLifecycleClosingForState(result map[string]interface{}, state colony.ColonyState, lastCommand string) string {
	if answer, ok := nextActionFromResult(result); ok {
		return renderNextActionCard(answer)
	}
	return renderNextActionCard(lifecycleNextActionForState(state, lastCommand, "", ""))
}

func renderInitVisual(goal, scope, sessionID, dataDir string, charter *colony.Charter, hiveSeeded int, proposals []initProposal, researchDocs ...string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("init"), "Colony Init"))
	b.WriteString(visualDividerStr())
	if charter != nil {
		// The classic birth ceremony: the approved charter is shown at the
		// moment the colony is created, not stored silently.
		b.WriteString(renderStageMarker("Charter"))
		b.WriteString(renderCharterFields(*charter))
	}
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
	if len(researchDocs) > 0 {
		b.WriteString("Research: ")
		b.WriteString(strings.Join(researchDocs, ", "))
		b.WriteString("\n")
	}
	// The classic colony-born close — the moment the colony exists.
	b.WriteString("\n👑 Queen has set the colony's intention\n\n")
	b.WriteString(fmt.Sprintf("   %q\n\n", goal))
	b.WriteString("   🟢 Colony Status: READY\n")
	if hiveSeeded > 0 {
		b.WriteString(fmt.Sprintf("   🧠 Hive wisdom: %d cross-colony pattern(s) seeded into QUEEN.md\n", hiveSeeded))
	}
	// Ranked, repo-aware next moves replace the old static trio that was
	// identical for every repo on earth. Fallback to the generic three only
	// when no proposals were computed (a proposal failure never fails init).
	if len(proposals) > 0 {
		b.WriteString(renderInitProposals(proposals))
	}
	// The closing block is the shared card (Phase 197 plan 04). It replaces
	// three fixed alternatives that were identical for every project on earth,
	// and it carries the "is it safe to close this chat" verdict too, so the
	// old sentence that used to follow is gone rather than said twice.
	b.WriteString(renderNextActionCard(lifecycleNextAction("init")))
	return b.String()
}

// renderCharterFields renders the seven charter fields as an aligned block —
// shared by the standalone charter display and the init birth ceremony.
func renderCharterFields(ch colony.Charter) string {
	var b strings.Builder
	// Each field line carries its own glyph via voiceLine -- the label
	// text and its alignment are unchanged (still "Intent:      value"
	// etc.), so every existing Contains-based assertion on a field's
	// label+value still matches; only the leading two-space indent is now
	// a glyph instead.
	b.WriteString(voiceLine("requirement", "Intent:      "+emptyFallback(ch.Intent, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("decision", "Vision:      "+emptyFallback(ch.Vision, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("requirement", "Governance:  "+emptyFallback(ch.Governance, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("goal", "Goals:       "+emptyFallback(ch.Goals, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("files", "Tech Stack:  "+emptyFallback(ch.TechStack, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("warning", "Key Risks:   "+emptyFallback(ch.KeyRisks, "(none)")))
	b.WriteString("\n")
	b.WriteString(voiceLine("avoid", "Constraints: "+emptyFallback(ch.Constraints, "(none)")))
	b.WriteString("\n")
	return b.String()
}

// renderCharterDisplay produces a visual rendering of the 7-section colony charter.
func renderCharterDisplay(ch colony.Charter) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("init"), "Colony Charter"))
	b.WriteString(visualDividerStr())
	b.WriteString(renderStageMarker("Charter"))
	b.WriteString(renderCharterFields(ch))
	b.WriteString(visualDividerStr())
	return b.String()
}

// renderResearchDisplay produces a visual rendering of the 4 research data sections
// extracted from the init-research JSON envelope. Returns empty string if all fields
// are nil/empty.
func renderResearchDisplay(data ceremonyResearchData) string {
	hasData := len(data.TechStackDetail) > 0 ||
		data.DirClassification.Type != "" ||
		len(data.GovernanceDetails) > 0 ||
		data.ColonyContextSummary.DetectedType != ""
	if !hasData {
		return ""
	}

	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("scout"), "Research Data"))
	b.WriteString(visualDividerStr())

	// Tech Stack Detail
	if len(data.TechStackDetail) > 0 {
		b.WriteString(renderStageMarker("Tech Stack Detail"))
		for _, ts := range data.TechStackDetail {
			depCount := len(ts.Deps) + len(ts.DevDeps)
			b.WriteString("  ")
			b.WriteString(emptyFallback(ts.Language, "(unknown)"))
			if ts.SourceFile != "" {
				b.WriteString("  (")
				b.WriteString(ts.SourceFile)
				b.WriteString(")")
			}
			b.WriteString(fmt.Sprintf("  %d dependencies", depCount))
			b.WriteString("\n")
		}
	}

	// Directory Classification
	if data.DirClassification.Type != "" {
		b.WriteString(renderStageMarker("Directory Classification"))
		b.WriteString("  Type:    ")
		b.WriteString(data.DirClassification.Type)
		b.WriteString("\n")
		if len(data.DirClassification.Signals) > 0 {
			b.WriteString("  Signals: ")
			b.WriteString(strings.Join(data.DirClassification.Signals, ", "))
			b.WriteString("\n")
		}
	}

	// Governance Details
	if len(data.GovernanceDetails) > 0 {
		b.WriteString(renderStageMarker("Governance Details"))
		for _, gd := range data.GovernanceDetails {
			b.WriteString("  ")
			b.WriteString(emptyFallback(gd.Tool, "(unknown)"))
			if gd.File != "" {
				b.WriteString("  [")
				b.WriteString(gd.File)
				b.WriteString("]")
			}
			if gd.Category != "" {
				b.WriteString("  (")
				b.WriteString(gd.Category)
				b.WriteString(")")
			}
			b.WriteString("\n")
		}
	}

	// Colony Context
	if data.ColonyContextSummary.DetectedType != "" {
		b.WriteString(renderStageMarker("Colony Context"))
		cs := data.ColonyContextSummary
		b.WriteString("  Detected Type:     ")
		b.WriteString(emptyFallback(cs.DetectedType, "(none)"))
		b.WriteString("\n")
		b.WriteString("  Dir Type:          ")
		b.WriteString(emptyFallback(cs.DirType, "(none)"))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Tech Stack Files:  %d\n", cs.TechStackCount))
		b.WriteString(fmt.Sprintf("  Governance Tools:  %d\n", cs.GovernanceToolCount))
		b.WriteString(fmt.Sprintf("  Pheromone Count:   %d\n", cs.PheromoneCount))
		b.WriteString(fmt.Sprintf("  File Count:        %d\n", cs.FileCount))
		b.WriteString(fmt.Sprintf("  Is Git Repo:       %t\n", cs.IsGitRepo))
	}

	b.WriteString(visualDividerStr())
	return b.String()
}

func renderColonizeVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("colonize"), "Colonize"))
	b.WriteString(visualDividerStr())
	dispatchMode := strings.TrimSpace(stringValue(result["dispatch_mode"]))
	requiresFinalizer, _ := result["requires_finalizer"].(bool)
	if requiresFinalizer || dispatchMode == "agent-delegate" || dispatchMode == "plan-only" {
		b.WriteString(voiceLine("status", "Territory survey manifest ready.") + "\n")
	} else {
		b.WriteString(voiceLine("status", "Territory survey complete.") + "\n")
	}
	b.WriteString(voiceLine("files", "Root: "+stringValue(result["root"])))
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
		b.WriteString(voiceLine("files", fmt.Sprintf("Files: %d across %d directories", intValue(stats["files"]), intValue(stats["directories"]))))
		b.WriteString("\n")
	}
	if surveyDir := strings.TrimSpace(stringValue(result["survey_dir"])); surveyDir != "" {
		b.WriteString(voiceLine("files", "Survey: "+surveyDir))
		b.WriteString("\n")
	}
	if warning := strings.TrimSpace(stringValue(result["survey_warning"])); warning != "" {
		b.WriteString(voiceLine("warning", "Survey Warning") + "\n")
		b.WriteString("  - ")
		b.WriteString(warning)
		b.WriteString("\n")
	}
	if surveyors, ok := result["surveyors"].([]interface{}); ok && len(surveyors) > 0 {
		dispatches := parseSurveyorMaps(surveyors)
		hasRealData := hasRealExecutionData(dispatches)
		if dispatchMode == "" {
			if hasRealData {
				dispatchMode = "real"
			} else {
				dispatchMode = "synthetic"
			}
		}
		b.WriteString(voiceLine("status", "Dispatch: "+humanizeDispatchMode(dispatchMode)))
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
				b.WriteString(dispatchTaskLine(d.Task))
				b.WriteString("\n")
			}
		}
	}
	if contract := renderDispatchContract(result["dispatch_contract"]); contract != "" {
		b.WriteString("\n")
		b.WriteString(contract)
	}
	if files := stringSliceValue(result["survey_files"]); len(files) > 0 {
		b.WriteString("\n" + voiceLine("files", "Reports") + "\n")
		b.WriteString(renderIndentedList(files))
	}
	b.WriteString("\n" + voiceLine("artifact", "Coordination: "+displayDataPath("spawn-tree.txt")))
	b.WriteString("\n")
	if requiresFinalizer || dispatchMode == "agent-delegate" || dispatchMode == "plan-only" {
		// This run only prepared the work; the platform running it does the
		// dispatching. That is what happened, so it is reported here, above the
		// card, rather than standing in for what the owner does next.
		finalizer := strings.TrimSpace(stringValue(result["finalizer_command"]))
		if finalizer == "" {
			finalizer = "aether colonize-finalize --completion-file <file>"
		}
		b.WriteString("\n")
		b.WriteString(renderStageMarker("How this run is being driven"))
		b.WriteString("The helpers listed above have not been sent yet. Whatever is running this — the chat\n")
		b.WriteString("app or the automation — sends them, then records the result with `" + finalizer + "`.\n")
		b.WriteString("Nothing under `.aether/data/` should be edited by hand; that step writes it.\n")
	}
	b.WriteString(renderLifecycleClosing(result, "colonize"))
	return b.String()
}

func renderColonizeDispatchPreview(root string, dispatches []codexSurveyorDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("colonize-dispatch"), "Colonize Dispatch"))
	b.WriteString(visualDividerStr())
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
		b.WriteString(dispatchTaskLine(dispatch.Task))
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
			Caste:     stringValue(entry["caste"]),
			Name:      stringValue(entry["name"]),
			Task:      stringValue(entry["task"]),
			TaskID:    stringValue(entry["task_id"]),
			AgentName: stringValue(entry["agent_name"]),
			Status:    stringValue(entry["status"]),
			Summary:   stringValue(entry["summary"]),
			Duration:  floatValue(entry["duration"]),
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
		if isSuccessfulExternalBuildStatus(status) {
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

// hasRealPlanningExecutionData returns true when a planning worker reports evidence
// beyond a manifest placeholder. Planned and spawned entries are not completion evidence.
func hasRealPlanningExecutionData(dispatches []codexPlanningDispatch) bool {
	for _, d := range dispatches {
		switch strings.TrimSpace(d.Status) {
		case "", "planned", "spawned":
			continue
		default:
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
		if isSuccessfulExternalBuildStatus(status) {
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
	case "completed", "completed_no_change", "passed", "success", "manually-reconciled":
		return "\u2713"
	case "blocked", "interrupted":
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
	if visual, ok := renderCanonicalPlanningResult(result, planningVisualOptions{Width: lifecycleStatusOutputWidth()}); ok {
		return visual
	}
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("plan"), "Plan"))
	b.WriteString(visualDividerStr())
	if _, ok := result["repair_source"]; ok {
		if stringValue(result["repair_scope"]) == "accepted_revision" {
			b.WriteString(voiceLine("done", "The accepted plan's task dependencies are valid. Its contents and approval are unchanged.") + "\n")
		} else if repaired, _ := result["repaired"].(bool); repaired {
			b.WriteString(voiceLine("done", "Repaired phase-plan dependency references.") + "\n")
		} else {
			b.WriteString(voiceLine("done", "Phase-plan dependency references are already valid.") + "\n")
		}
		if phasePlan := strings.TrimSpace(stringValue(result["phase_plan"])); phasePlan != "" {
			b.WriteString(voiceLine("artifact", "Artifact: "+phasePlan) + "\n")
		}
		b.WriteString(voiceLine("phase", fmt.Sprintf("Plan size: %d phases, %d tasks", intValue(result["phase_count"]), intValue(result["task_count"]))) + "\n\n")
		if repairs := stringSliceValue(result["repairs"]); len(repairs) > 0 {
			b.WriteString(voiceLine("checkpoint", "Repairs") + "\n")
			b.WriteString(renderIndentedList(voicedLines("checkpoint", repairs)))
			b.WriteString("\n")
		}
		b.WriteString(renderLifecycleClosing(result, "plan"))
		return b.String()
	}
	existing, _ := result["existing_plan"].(bool)
	planOnly, _ := result["plan_only"].(bool)
	requiresFinalizer, _ := result["requires_finalizer"].(bool)
	requiresNextIteration, _ := result["requires_next_iteration"].(bool)
	if existing {
		b.WriteString(voiceLine("decision", "Existing colony plan loaded — this project already has one.") + "\n")
	} else if requiresNextIteration {
		b.WriteString(voiceLine("decision", "Planning iteration recorded; another Scout and Route-Setter pass is required before the plan is written.") + "\n")
	} else if planOnly && requiresFinalizer {
		b.WriteString(voiceLine("decision", "Planning manifest prepared for host-dispatched Scout and Route-Setter workers (helpers).") + "\n")
	} else {
		b.WriteString(voiceLine("decision", "Scout and Route-Setter mapped the goal into executable phases.") + "\n")
	}
	b.WriteString(voiceLine("goal", "Goal: "+stringValue(result["goal"])) + "\n")
	if granularity := strings.TrimSpace(stringValue(result["granularity"])); granularity != "" {
		granularityLine := "Granularity: " + granularity
		if min, max := intValue(result["granularity_min"]), intValue(result["granularity_max"]); min > 0 && max > 0 {
			granularityLine += fmt.Sprintf(" (%d-%d phases)", min, max)
		}
		b.WriteString(voiceLine("decision", granularityLine) + "\n")
	}
	// Depth Selection Banner (per D-01, D-02)
	if planningDepth := strings.TrimSpace(stringValue(result["planning_depth"])); planningDepth != "" {
		verificationDepth := strings.TrimSpace(stringValue(result["verification_depth"]))
		planningSmartDefault, _ := result["planning_smart_default"].(bool)
		verificationSmartDefault, _ := result["verification_smart_default"].(bool)
		totalPhases := intValue(result["count"])
		if totalPhases == 0 {
			if phases := phaseSliceValue(result["phases"]); len(phases) > 0 {
				totalPhases = len(phases)
			}
		}
		b.WriteString(renderStageMarker("Depth Selection"))

		// Extract the planning Phase object from the result map for reason rendering
		planningPhase, _ := result["planning_phase"].(colony.Phase)

		// Planning depth line with full reason
		if planningSmartDefault && planningPhase.ID > 0 {
			reason := renderSmartDepthReason(planningPhase, totalPhases)
			b.WriteString(voiceLine("decision", fmt.Sprintf("Planning depth: %s (%s)", planningDepth, reason)) + "\n")
		} else {
			b.WriteString(voiceLine("decision", fmt.Sprintf("Planning depth: %s", planningDepth)) + "\n")
		}

		// Verification depth line with full reason
		if verificationDepth != "" {
			if verificationSmartDefault && planningPhase.ID > 0 {
				b.WriteString(voiceLine("decision", strings.TrimSuffix(renderReviewDepthLineWithReason(
					colony.NormalizeVerificationDepth(verificationDepth),
					planningPhase.ID, totalPhases, planningPhase, true,
				), "\n")) + "\n")
			} else {
				b.WriteString(voiceLine("decision", fmt.Sprintf("Verification depth: %s", verificationDepth)) + "\n")
			}
		}

		// Override hint when either was smart-defaulted
		if planningSmartDefault || verificationSmartDefault {
			b.WriteString(voiceLine("decision", "Override: --planning-depth <light|standard|deep> --verification-depth <light|standard|heavy>") + "\n")
		}
	}
	// Dual-type: both runCodexPlanWithOptions and runCodexPlanFinalize always
	// store confidence as a codexPlanConfidence struct, never as a map -- the
	// map-only branch below never matched on either the direct or chat path
	// (WINDOWS.md entry 5, found by 198-04). Mirrors planning_loop's existing
	// struct/map switch immediately below.
	if confidence, ok := result["confidence"].(codexPlanConfidence); ok {
		b.WriteString(voiceLine("evidence", fmt.Sprintf("Confidence: %d%% overall", int(confidence.Overall))) + "\n")
	} else if confidence, ok := result["confidence"].(map[string]interface{}); ok {
		b.WriteString(voiceLine("evidence", fmt.Sprintf("Confidence: %d%% overall", intValue(confidence["overall"]))) + "\n")
	}
	if planningLoop, ok := result["planning_loop"].(codexPlanningLoop); ok && planningLoop.TargetConfidence > 0 {
		b.WriteString(voiceLine("evidence", fmt.Sprintf("Planning loop: target %d%%, %d/%d iteration(s), stop=%s",
			planningLoop.TargetConfidence,
			planningLoop.Iterations,
			planningLoop.MaxIterations,
			planningLoop.StopReason,
		)) + "\n")
	} else if planningLoop, ok := result["planning_loop"].(map[string]interface{}); ok && intValue(planningLoop["target_confidence"]) > 0 {
		b.WriteString(voiceLine("evidence", fmt.Sprintf("Planning loop: target %d%%, %d/%d iteration(s), stop=%s",
			intValue(planningLoop["target_confidence"]),
			intValue(planningLoop["iterations"]),
			intValue(planningLoop["max_iterations"]),
			stringValue(planningLoop["stop_reason"]),
		)) + "\n")
	}
	phases := phaseSliceValue(result["phases"])
	b.WriteString(voiceLine("phase", fmt.Sprintf("Plan size: %d phases", len(phases))) + "\n\n")
	if revision, ok := result["plan_revision"].(colony.PlanRevision); ok && strings.TrimSpace(revision.ID) != "" {
		b.WriteString(voiceLine("history", fmt.Sprintf("Plan revision: r%d (%s) - %s", revision.Number, revision.ReasonType, revision.Reason)) + "\n\n")
	} else if revision, ok := result["plan_revision"].(map[string]interface{}); ok && strings.TrimSpace(stringValue(revision["id"])) != "" {
		b.WriteString(voiceLine("history", fmt.Sprintf("Plan revision: %s (%s)", stringValue(revision["id"]), stringValue(revision["reason_type"]))) + "\n\n")
	}
	if warning := strings.TrimSpace(stringValue(result["clarification_warning"])); warning != "" {
		b.WriteString(voiceLine("question", "Clarifications") + "\n")
		b.WriteString(renderIndentedList(voicedLines("question", []string{
			fmt.Sprintf("%d unresolved clarification(s)", intValue(result["unresolved_clarifications"])),
			warning,
		})))
		b.WriteString("\n")
	}
	if warning := strings.TrimSpace(stringValue(result["planning_warning"])); warning != "" {
		b.WriteString(voiceLine("warning", "Planning Warning") + "\n")
		b.WriteString(renderIndentedList(voicedLines("warning", []string{warning})))
		b.WriteString("\n")
	}
	// WINDOWS.md entry 6 (198-REVIEW.md WR scope): renderResearchFailedWarning's
	// own doc comment says this warning exists "so the omission is durable
	// and visible" when a phase was planned without its research, but it was
	// computed and carried, then silently dropped. research_failed_phases is
	// the phase-ID list that fed this warning's own text -- named again here,
	// structurally, rather than only as prose inside the warning string.
	if warning := strings.TrimSpace(stringValue(result["research_warning"])); warning != "" {
		b.WriteString(voiceLine("warning", "Research Warning") + "\n")
		researchWarningLines := []string{warning}
		if failedPhases := intSliceValue(result["research_failed_phases"]); len(failedPhases) > 0 {
			researchWarningLines = append(researchWarningLines, "Affected phase(s): "+joinInts(failedPhases))
		}
		b.WriteString(renderIndentedList(voicedLines("warning", researchWarningLines)))
		b.WriteString("\n")
	}
	if dispatches, ok := result["dispatches"].([]interface{}); ok && len(dispatches) > 0 {
		parsed := parsePlanningDispatchMaps(dispatches)
		hasRealData := hasRealPlanningExecutionData(parsed)
		dispatchMode := strings.TrimSpace(stringValue(result["dispatch_mode"]))
		if dispatchMode == "" && hasRealData {
			dispatchMode = "real"
		}
		if dispatchMode != "" {
			b.WriteString(voiceLine("status", "Dispatch: "+humanizeDispatchMode(dispatchMode)) + "\n")
		}
		b.WriteString("\n")
		b.WriteString(voiceLine("colony", "Workers") + "\n")
		if hasRealData {
			b.WriteString(renderPlanningWorkerResults(parsed))
		} else {
			for _, d := range parsed {
				b.WriteString("  ")
				b.WriteString(casteIdentity(d.Caste))
				b.WriteString(" ")
				b.WriteString(d.Name)
				b.WriteString("  ")
				b.WriteString(dispatchTaskLine(d.Task))
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}
	if contract := renderDispatchContract(result["dispatch_contract"]); contract != "" && (!existing || requiresFinalizer) {
		b.WriteString(contract)
		b.WriteString("\n")
	}
	if planOnly {
		if !existing {
			if agentDelegate, _ := result["agent_delegate"].(bool); agentDelegate || strings.TrimSpace(stringValue(result["dispatch_mode"])) == "agent-delegate" {
				if reason := strings.TrimSpace(stringValue(result["agent_delegate_reason"])); reason != "" {
					b.WriteString(voiceLine("decision", "Agent-Delegate") + "\n")
					b.WriteString(renderIndentedList(voicedLines("decision", []string{reason})))
					b.WriteString("\n")
				}
				b.WriteString(renderPlanManifestOnlyNotice())
				b.WriteString(renderLifecycleClosing(result, "plan"))
				return b.String()
			}
			b.WriteString(renderPlanManifestOnlyNotice())
			b.WriteString(renderLifecycleClosing(result, "plan"))
			return b.String()
		}
	}
	if files := stringSliceValue(result["planning_files"]); len(files) > 0 {
		b.WriteString(voiceLine("artifact", "Planning Artifacts") + "\n")
		b.WriteString(renderIndentedList(voicedLines("artifact", files)))
		b.WriteString("\n")
	}
	if files := stringSliceValue(result["phase_research_files"]); len(files) > 0 {
		b.WriteString(voiceLine("artifact", "Phase Research") + "\n")
		researchFileLines := voicedLines("artifact", limitStrings(files, 5))
		if len(files) > 5 {
			researchFileLines = append(researchFileLines, voiceLine("artifact", fmt.Sprintf("... and %d more phase research files", len(files)-5)))
		}
		b.WriteString(renderIndentedList(researchFileLines))
		b.WriteString("\n")
	}
	if requiresNextIteration {
		if gaps := stringSliceValue(result["selected_gaps"]); len(gaps) > 0 {
			b.WriteString(voiceLine("blocked", "Next Iteration Gaps") + "\n")
			b.WriteString(renderIndentedList(voicedLines("blocked", gaps)))
			b.WriteString("\n")
		}
		b.WriteString(voiceLine("artifact", "Coordination: "+displayDataPath("spawn-tree.txt")) + "\n\n")
		b.WriteString(voiceLine("blocked", "The plan is not finished: another research-and-planning pass is needed before there is anything to build.") + "\n")
		b.WriteString(renderLifecycleClosing(result, "plan"))
		return b.String()
	}

	for _, phase := range phases {
		b.WriteString(voiceLine("phase", fmt.Sprintf("Phase %d — %s", phase.ID, phase.Name)) + "\n")
		if strings.TrimSpace(phase.Description) != "" {
			b.WriteString(voiceLine("phase", "  "+strings.TrimSpace(phase.Description)) + "\n")
		}
		for _, task := range phase.Tasks {
			taskLabel := strings.TrimSpace(ptrStr(task.ID))
			if taskLabel == "" {
				taskLabel = task.Goal
			}
			b.WriteString(voiceLine("task", fmt.Sprintf("  Task %s", taskLabel)) + "\n")
			b.WriteString(voiceLine("goal", "    Goal: "+strings.TrimSpace(task.Goal)) + "\n")
			if len(task.DependsOn) > 0 {
				b.WriteString(voiceLine("history", "    Depends on: "+strings.Join(task.DependsOn, ", ")) + "\n")
			}
			if len(task.Constraints) > 0 {
				b.WriteString(voiceLine("blocked", "    Constraints:") + "\n")
				for _, constraint := range task.Constraints {
					b.WriteString(voiceLine("blocked", "      "+constraint) + "\n")
				}
			}
			if len(task.Hints) > 0 {
				b.WriteString(voiceLine("evidence", "    Hints:") + "\n")
				for _, hint := range task.Hints {
					b.WriteString(voiceLine("evidence", "      "+hint) + "\n")
				}
			}
			if len(task.SuccessCriteria) > 0 {
				b.WriteString(voiceLine("checkpoint", "    Success Criteria:") + "\n")
				for _, criterion := range task.SuccessCriteria {
					b.WriteString(voiceLine("checkpoint", "      "+criterion) + "\n")
				}
			}
		}
		if len(phase.Tasks) == 0 {
			b.WriteString(voiceLine("status", "  No explicit tasks captured for this phase.") + "\n")
		}
		if len(phase.SuccessCriteria) > 0 {
			b.WriteString(voiceLine("checkpoint", "  Phase Success Criteria:") + "\n")
			for _, criterion := range phase.SuccessCriteria {
				b.WriteString(voiceLine("checkpoint", "    "+criterion) + "\n")
			}
		}
		b.WriteString("\n")
	}
	// WINDOWS.md entry 6 (198-REVIEW.md WR scope): a completed plan's own
	// unresolved gaps were carried but never surfaced here -- distinct from
	// the mid-loop path above, where the narrower selected_gaps subset chosen
	// for the next iteration IS rendered ("Next Iteration Gaps").
	if gaps := stringSliceValue(result["gaps"]); len(gaps) > 0 {
		b.WriteString(voiceLine("blocked", "Unresolved Gaps") + "\n")
		b.WriteString(renderIndentedList(voicedLines("blocked", gaps)))
		b.WriteString("\n")
	}
	b.WriteString(voiceLine("artifact", "Coordination: "+displayDataPath("spawn-tree.txt")) + "\n\n")

	b.WriteString(renderLifecycleClosing(result, "plan"))
	return b.String()
}

// renderPlanManifestOnlyNotice reports what a plan-only run actually did. It is
// the run explaining itself, which is not the same thing as telling the owner
// what to type next -- that is the card's job, below it.
func renderPlanManifestOnlyNotice() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderStageMarker("How this run is being driven"))
	b.WriteString("No plan was written and nothing was changed. This run only prepared the work for\n")
	b.WriteString("the research and planning helpers; whatever is running it sends them, then records\n")
	b.WriteString("what they produced with `aether plan-finalize --completion-file <file>`.\n")
	return b.String()
}

func renderPlanDispatchPreview(goal string, dispatches []codexPlanningDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("plan-dispatch"), "Plan Dispatch"))
	b.WriteString(visualDividerStr())
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
		b.WriteString(dispatchTaskLine(dispatch.Task))
		b.WriteString("\n")
	}
	b.WriteString("\nCoordination: ")
	b.WriteString(displayDataPath("spawn-tree.txt"))
	b.WriteString("\n")
	return b.String()
}

func reviewDepthFromResult(result map[string]interface{}) colony.VerificationDepth {
	if rd, ok := result["review_depth"].(string); ok {
		return colony.NormalizeVerificationDepth(rd)
	}
	return colony.VerificationDepthLight
}

func renderReviewDepthLine(depth colony.VerificationDepth, phaseNum, totalPhases int) string {
	switch depth {
	case colony.VerificationDepthHeavy:
		if phaseNum == totalPhases {
			return "Review depth: heavy (final phase)"
		}
		return fmt.Sprintf("Review depth: heavy (Phase %d of %d)", phaseNum, totalPhases)
	case colony.VerificationDepthStandard:
		return fmt.Sprintf("Review depth: standard (Phase %d of %d)", phaseNum, totalPhases)
	default:
		return fmt.Sprintf("Review depth: light (Phase %d of %d)", phaseNum, totalPhases)
	}
}

// renderSmartDepthReason produces a human-readable reason for why a depth was
// auto-detected. Used by renderReviewDepthLineWithReason to annotate smart defaults.
func renderSmartDepthReason(phase colony.Phase, totalPhases int) string {
	risk := phaseRiskLevel(phase)
	position := phasePositionLevel(phase.ID, totalPhases)

	if risk == "high" {
		return getSmartDefaultReason("high_risk")
	}
	if position == "final" {
		return getSmartDefaultReason("final_phase")
	}
	if risk == "medium" {
		return getSmartDefaultReason("medium_risk")
	}
	if position == "early" {
		return getSmartDefaultReason("early_phase")
	}
	if position == "late" {
		return getSmartDefaultReason("late_phase")
	}
	return getSmartDefaultReason("standard")
}

// renderReviewDepthLineWithReason wraps renderReviewDepthLine with smart-default
// annotation. When smartDefault is true, the parenthetical reason is replaced with
// the auto-detection reason. Phase 86 will switch callers to use this function
// when it adds the UI layer that knows whether depth was auto-detected.
func renderReviewDepthLineWithReason(depth colony.VerificationDepth, phaseNum, totalPhases int, phase colony.Phase, smartDefault bool) string {
	base := renderReviewDepthLine(depth, phaseNum, totalPhases)
	if !smartDefault {
		return base
	}
	reason := renderSmartDepthReason(phase, totalPhases)
	switch depth {
	case colony.VerificationDepthHeavy:
		if phaseNum == totalPhases {
			return "Review depth: heavy (final phase)"
		}
		return fmt.Sprintf("Review depth: heavy (%s)", reason)
	case colony.VerificationDepthStandard:
		return fmt.Sprintf("Review depth: standard (%s)", reason)
	default:
		return fmt.Sprintf("Review depth: light (%s)", reason)
	}
}

func renderBuildVisual(state colony.ColonyState, phase colony.Phase) string {
	reviewDepth := resolveVerificationDepth(phase, len(state.Plan.Phases), false, false, "")
	dispatches, err := plannedBuildDispatches(phase, state.ColonyDepth)
	if err != nil {
		return renderUnplannablePhase(phase, err)
	}
	return renderBuildVisualWithDispatches(state, phase, dispatches, reviewDepth)
}

// renderUnplannablePhase is what every planned-team surface shows when the
// planner refuses a phase outright (WR-06, 195-REVIEW.md). Before this, those
// surfaces rendered an empty team, which reads as "this phase needs no work"
// -- the opposite of "this phase cannot be started until its plan is repaired".
func renderUnplannablePhase(phase colony.Phase, err error) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚠", fmt.Sprintf("Phase %d Cannot Be Started", phase.ID)))
	b.WriteString(visualDividerStr())
	b.WriteString("Phase: ")
	b.WriteString(strings.TrimSpace(phase.Name))
	b.WriteString("\n\n")
	b.WriteString("No workers can be planned for this phase, and this is NOT because there is nothing to do.\n")
	b.WriteString("The plan itself has to be repaired first:\n\n  ")
	b.WriteString(strings.TrimSpace(err.Error()))
	b.WriteString("\n")
	b.WriteString(renderNextUp("Fix the step order in the plan, then run the build for this phase again."))
	return b.String()
}

// renderSuggestedSteering renders the colony's unreviewed pheromone
// suggestions as numbered proposals for a multiple-choice ask. The analysis
// engine has stored these on colony state since v1.x and nothing ever showed
// them to the operator — recommendations piled up invisibly while the approve
// command sat on the orphan allowlist.
func renderSuggestedSteering(state colony.ColonyState) string {
	if state.PendingSuggestions == nil {
		return ""
	}
	active := filterActiveSuggestions(state.PendingSuggestions)
	if len(active) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderStageMarker("Suggested Steering"))
	b.WriteString("The colony noticed patterns worth steering on — proposals only, nothing is written until you approve it:\n")
	for i, suggestion := range active {
		emoji := signalTypeGlyph(suggestion.Type)
		b.WriteString(fmt.Sprintf("  %d. %s [%s] %s\n", i+1, emoji, strings.ToUpper(strings.TrimSpace(suggestion.Type)), strings.TrimSpace(suggestion.Content)))
		if reason := strings.TrimSpace(suggestion.Reason); reason != "" {
			b.WriteString("     └── " + reason + "\n")
		}
		b.WriteString("     └── adopt: `aether suggest-approve --approve " + suggestion.ID + "`\n")
	}
	b.WriteString("Dismiss one with `aether suggest-approve --dismiss <id>`, or everything with `--dismiss-all`.\n")
	return b.String()
}

// renderSteeringSignals shows the operator's active pheromone signals at the
// moment they take effect — the build's Context stage. The signals were
// always injected into every worker prompt; until this render, nothing told
// the operator their steering was live, which made the steering loop feel
// disconnected ("did my note do anything?").
func renderSteeringSignals() string {
	pf := loadPheromones()
	var active []colony.PheromoneSignal
	if pf != nil {
		for _, sig := range pf.Signals {
			if sig.Active {
				active = append(active, sig)
			}
		}
	}
	if len(active) == 0 {
		return "Steering signals: none — run `aether focus \"<area>\"` or `aether redirect \"<avoid>\"` to steer this build.\n"
	}

	sort.SliceStable(active, func(i, j int) bool {
		return signalPriority(active[i].Type) < signalPriority(active[j].Type)
	})

	now := time.Now().UTC()
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Steering signals: %d active — injected into every worker prompt\n", len(active)))
	const shown = 5
	for i, sig := range active {
		if i >= shown {
			b.WriteString(fmt.Sprintf("  … and %d more — `aether pheromone-display` for the full view\n", len(active)-shown))
			break
		}
		emoji := signalTypeGlyph(sig.Type)
		text := strings.TrimSpace(extractText(sig.Content))
		if text == "" {
			text = "(no content)"
		}
		if len(text) > 70 {
			text = text[:67] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s [%d%%] %q\n", emoji, int(math.Round(computeEffectiveStrength(sig, now)*100)), text))
	}
	return b.String()
}

func renderBuildVisualWithDispatches(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch, reviewDepth colony.VerificationDepth, policyOpt ...codexQueenExecutionPolicy) string {
	var policy codexQueenExecutionPolicy
	if len(policyOpt) > 0 {
		policy = policyOpt[0]
	}
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build"), fmt.Sprintf("Build Phase %d", phase.ID)))
	b.WriteString(visualDividerStr())
	b.WriteString(voiceLine("phase", renderProgressSummary(phase.ID, len(state.Plan.Phases))))
	b.WriteString("\n")
	b.WriteString(voiceLine("phase", "Phase: "+phase.Name))
	b.WriteString("\n")
	b.WriteString(voiceLine("status", renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases))))
	b.WriteString("\n")
	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString(voiceLine("goal", "Objective: "+strings.TrimSpace(phase.Description)))
		b.WriteString("\n")
	}
	b.WriteString(renderStageMarker("Context"))
	b.WriteString(renderSteeringSignals())
	b.WriteString(renderStageMarker("Tasks"))
	for _, task := range phase.Tasks {
		kind := "task"
		if task.Status == colony.TaskCompleted {
			kind = "done"
		}
		b.WriteString("  ")
		b.WriteString(voiceLine(kind, strings.TrimSpace(task.Goal)))
		b.WriteString("\n")
	}
	if len(phase.Tasks) == 0 {
		b.WriteString("  ")
		b.WriteString(voiceLine("task", "No explicit tasks captured for this phase."))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Dispatch"))
	if teamChoice := renderQueenTeamChoice(policy, dispatches); teamChoice != "" {
		b.WriteString(teamChoice)
		b.WriteString("\n")
	}
	b.WriteString(renderSpawnPlanForDispatches(dispatches, effectiveParallelMode(state)))
	b.WriteString(renderArtifactsSection(
		displayDataPath(fmt.Sprintf("build/phase-%d/manifest.json", phase.ID)),
		displayDataPath("last-build-claims.json"),
		displayDataPath("spawn-tree.txt"),
	))
	b.WriteString(renderStageMarker(fmt.Sprintf("Verification [%s]", string(reviewDepth))))
	b.WriteString(voiceLine("evidence", "Verification happens during `aether continue`."))
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Housekeeping"))
	b.WriteString(voiceLine("focus", "Signal housekeeping (tidying up the notes that steer this project) runs during `aether continue`."))
	b.WriteString("\n")
	if len(state.Plan.Phases) == phase.ID {
		b.WriteString(renderStageMarker("Colony Complete"))
		b.WriteString(voiceLine("milestone", "This is the last phase in the plan. Once its work is checked, the project can be"))
		b.WriteString("\n")
		b.WriteString("signed off as finished.\n")
	} else {
		b.WriteString(renderStageMarker("Next Phase"))
		b.WriteString(voiceLine("next", fmt.Sprintf("Phase %d follows after continue.", phase.ID+1)))
		b.WriteString("\n")
	}
	b.WriteString(renderNextActionCard(lifecycleNextActionForState(state, "build", "", "")))
	return b.String()
}

// renderBuildPartialCreditVisual is the screen a partially credited build
// shows instead of the ordinary finished-build one (WR-05, 195-REVIEW.md).
//
// The ordinary screen unconditionally states that verification happens next,
// names the following phase, and tells the owner to run the continue command --
// all three untrue of a phase where some tasks are still unstarted. It also
// never showed the recovery command at all, so a half-built phase read as a
// finished one on the surface the owner actually looks at.
//
// Everything here is written for someone who has never opened a file: no task
// IDs without the task's own words beside them, no repo vocabulary, and one
// command to run next.
func renderBuildPartialCreditVisual(state colony.ColonyState, phase colony.Phase, unfinishedTaskIDs []string, recoveryCommand string) string {
	unfinished := make(map[string]struct{}, len(unfinishedTaskIDs))
	for _, id := range unfinishedTaskIDs {
		if id = strings.TrimSpace(id); id != "" {
			unfinished[id] = struct{}{}
		}
	}

	var done, remaining []string
	for idx := range phase.Tasks {
		id := buildTaskID(phase.Tasks[idx], idx)
		label := strings.TrimSpace(phase.Tasks[idx].Goal)
		if label == "" {
			label = id
		} else {
			label = fmt.Sprintf("%s (%s)", label, id)
		}
		if _, left := unfinished[id]; left {
			remaining = append(remaining, label)
			continue
		}
		if phase.Tasks[idx].Status == colony.TaskCompleted {
			done = append(done, label)
		}
	}

	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build"), fmt.Sprintf("Build Phase %d — Partly Done", phase.ID)))
	b.WriteString(visualDividerStr())
	b.WriteString(voiceLine("phase", renderProgressSummary(phase.ID, len(state.Plan.Phases))))
	b.WriteString("\n")
	b.WriteString(voiceLine("phase", "Phase: "+phase.Name))
	b.WriteString("\n\n")
	b.WriteString(voiceLine("warning", "Some of this phase is finished and saved. The rest was never started."))
	b.WriteString("\n")
	b.WriteString(voiceLine("warning", "Nothing that was finished has been undone, and no finished work will be done twice."))
	b.WriteString("\n")

	b.WriteString(renderStageMarker("Finished and kept"))
	if len(done) == 0 {
		b.WriteString("  (nothing)\n")
	}
	for _, label := range done {
		b.WriteString("  ")
		b.WriteString(voiceLine("done", label))
		b.WriteString("\n")
	}

	b.WriteString(renderStageMarker("Still to do"))
	if len(remaining) == 0 {
		b.WriteString("  (nothing)\n")
	}
	for _, label := range remaining {
		b.WriteString("  ")
		b.WriteString(voiceLine("task", label))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(voiceLine("warning", "This phase is NOT ready to be checked yet. Finish the remaining work first."))
	b.WriteString("\n")
	// The command that picks up ONLY the work listed above is knowledge this run
	// has and the saved project does not, so it is handed to the one decision as
	// an input rather than written over its answer afterwards.
	b.WriteString(renderNextActionCard(lifecycleNextActionForState(state, "build",
		strings.TrimSpace(recoveryCommand), nextActionUnfinishedWorkWhy)))
	return b.String()
}

// renderBuildPartialCreditResultVisual renders only from the typed, durable
// recovery child carried by the build result. A missing projection is an
// explicit error screen: falling back to the ordinary build-complete screen
// would falsely tell the owner that unfinished work was done.
func renderBuildPartialCreditResultVisual(state colony.ColonyState, phase colony.Phase, result map[string]interface{}) string {
	recovery, ok := partialBuildRecoveryFromResult(result)
	if !ok {
		return renderVisualError("Partial build recovery evidence is incomplete", map[string]interface{}{
			"recovery": "inspect the build attempt journal before continuing",
		})
	}
	return renderBuildPartialCreditVisual(state, phase, recovery.UnfinishedTaskIDs, recovery.RedispatchCommand)
}

// renderBuildAdvisoryResult consumes the same typed advisory projection JSON
// callers receive. It never re-checks mutable colony state or infers blocker
// truth from already-rendered text.
func renderBuildAdvisoryResult(result map[string]interface{}) string {
	advisory, ok := buildAdvisoryFromResult(result)
	if !ok {
		return ""
	}
	return renderBuildBlockerAdvisory(advisory)
}

func renderBuildPlanOnlyVisual(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch, reviewDepth colony.VerificationDepth, policyOpt ...codexQueenExecutionPolicy) string {
	var policy codexQueenExecutionPolicy
	if len(policyOpt) > 0 {
		policy = policyOpt[0]
	}
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build-dispatch"), fmt.Sprintf("Build Plan %d", phase.ID)))
	b.WriteString(visualDividerStr())
	b.WriteString("Dispatch manifest only. No state was changed and no workers were spawned.\n")
	b.WriteString(renderProgressSummary(phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase.Name)
	b.WriteString("\n")
	b.WriteString(renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	if strings.TrimSpace(phase.Description) != "" {
		b.WriteString("Objective: ")
		b.WriteString(strings.TrimSpace(phase.Description))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if teamChoice := renderQueenTeamChoice(policy, dispatches); teamChoice != "" {
		b.WriteString(teamChoice)
		b.WriteString("\n")
	}
	b.WriteString(renderSpawnPlanForDispatches(dispatches, effectiveParallelMode(state)))
	b.WriteString("\n")
	b.WriteString(renderStageMarker("How this run is being driven"))
	b.WriteString("The helpers listed above have not been sent yet, and nothing was changed. Whatever\n")
	b.WriteString("is running this sends them from the plan it was just handed.\n")
	b.WriteString(renderNextActionCard(lifecycleNextActionForState(state, "build", "", "")))
	return b.String()
}

func renderBuildFinalizeVisual(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build"), fmt.Sprintf("Build Finalize %d", phase.ID)))
	b.WriteString(visualDividerStr())
	b.WriteString("External Task worker results recorded.\n")
	b.WriteString(renderProgressSummary(phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase.Name)
	b.WriteString("\n\n")
	b.WriteString(renderSpawnPlanForDispatches(dispatches, effectiveParallelMode(state)))
	b.WriteString(renderArtifactsSection(
		displayDataPath(fmt.Sprintf("build/phase-%d/manifest.json", phase.ID)),
		displayDataPath("last-build-claims.json"),
		displayDataPath("spawn-tree.txt"),
	))
	b.WriteString(renderNextActionCard(lifecycleNextActionForState(state, "build", "", "")))
	return b.String()
}

func renderBuildDispatchPreview(state colony.ColonyState, phase colony.Phase, dispatches []codexBuildDispatch) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("build-dispatch"), fmt.Sprintf("Build Dispatch %d", phase.ID)))
	b.WriteString(visualDividerStr())
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
	b.WriteString(renderSpawnPlanForDispatches(dispatches, effectiveParallelMode(state)))
	return b.String()
}

func renderContinueVisual(state colony.ColonyState, phase colony.Phase, housekeeping *signalHousekeepingResult, final bool, nextPhase *colony.Phase, result map[string]interface{}, reviewDepth colony.VerificationDepth) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("continue"), "Continue"))
	b.WriteString(visualDividerStr())
	b.WriteString(voiceLine("status", renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases))))
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Verification"))
	if partial, _ := result["partial_success"].(bool); partial {
		b.WriteString(voiceLine("warning", "Verification passed with partial operational success."))
	} else {
		b.WriteString(voiceLine("done", "Verification pass complete."))
	}
	b.WriteString("\n")
	b.WriteString(voiceLine("done", fmt.Sprintf("Phase %d verified and completed: %s", phase.ID, phase.Name)))
	b.WriteString("\n")
	renderContinueVerificationSummaryMap(&b, continueTypedResultMapValue(result["verification"]))
	renderContinueVerificationDetail(&b, result["verification"])
	if issues := stringSliceValue(result["operational_issues"]); len(issues) > 0 {
		b.WriteString(voiceLine("evidence", "Operational evidence"))
		b.WriteString("\n")
		for _, issue := range issues {
			if issue = strings.TrimSpace(issue); issue == "" {
				continue
			}
			b.WriteString("  ")
			b.WriteString(voiceLine("evidence", issue))
			b.WriteString("\n")
		}
	}
	renderCriterionEvidenceLines(&b, result["verification"])
	b.WriteString(voiceLine("colony", "Workers (the helpers this phase used)"))
	b.WriteString("\n")
	if closed := stringSliceValue(result["closed_workers"]); len(closed) > 0 {
		b.WriteString(renderIndentedList(closed))
	} else {
		b.WriteString("  ")
		b.WriteString(voiceLine("done", "No workers (helpers) required closing"))
		b.WriteString("\n")
	}
	renderContinueWorkerFlowValue(&b, result["worker_flow"])
	renderSpecialistFindingBlocks(&b, result["worker_flow"])
	b.WriteString(voiceLine("evidence", "Verification passed during continue"))
	b.WriteString("\n")
	artifacts := []string{
		displayDataPath(fmt.Sprintf("build/phase-%d/verification.json", phase.ID)),
		displayDataPath(fmt.Sprintf("build/phase-%d/gates.json", phase.ID)),
	}
	if reviewReport := strings.TrimSpace(stringValue(result["review_report"])); reviewReport != "" {
		artifacts = append(artifacts, reviewReport)
	}
	artifacts = append(artifacts,
		displayDataPath(fmt.Sprintf("build/phase-%d/continue.json", phase.ID)),
		displayDataPath("spawn-tree.txt"),
	)
	b.WriteString(renderArtifactsSection(artifacts...))
	renderContinueGateSummaryMap(&b, continueTypedResultMapValue(result["gates"]))
	renderContinueGateDetail(&b, result["gates"])
	if closed := stringSliceValue(result["closed_workers"]); len(closed) > 0 {
		b.WriteString(voiceLine("colony", fmt.Sprintf("Workers (helpers) closed: %d", len(closed))))
		b.WriteString("\n")
	}
	b.WriteString(renderStageMarker("Housekeeping"))
	if housekeeping != nil {
		b.WriteString(voiceLine("focus", fmt.Sprintf("Signals: %d active -> %d active after housekeeping", housekeeping.ActiveBefore, housekeeping.ActiveAfter)))
		b.WriteString("\n")
		if housekeeping.Updated > 0 {
			b.WriteString(voiceLine("history", fmt.Sprintf("Expired: %d time-based, %d low-strength, %d stale continue signals",
				housekeeping.ExpiredByTime, housekeeping.DeactivatedByStrength, housekeeping.ExpiredWorkerContinue)))
			b.WriteString("\n")
		}
	}

	b.WriteString(renderLearningBeat(result["consolidation"]))
	b.WriteString(renderImprovementPassBeat(result["improvement_pass"]))
	b.WriteString(renderSuggestedSteering(state))

	if final {
		b.WriteString(renderStageMarker("Project Complete (Colony)"))
		b.WriteString(renderProjectComplete(state, len(state.Plan.Phases)))
		b.WriteString("\n\n")
		b.WriteString(voiceLine("milestone", "Every phase in the plan is finished. The project is ready to be signed off as"))
		b.WriteString("\n")
		b.WriteString("complete -- the stage this project calls Crowned Anthill.\n")
		b.WriteString(renderLifecycleClosingForState(result, state, "continue"))
		return b.String()
	}

	if nextPhase != nil {
		b.WriteString(renderStageMarker("Next Phase"))
		b.WriteString(voiceLine("next", fmt.Sprintf("Next phase ready: %d — %s", nextPhase.ID, nextPhase.Name)))
		b.WriteString("\n")
	}
	// The classic end-of-phase footer: flags, steering signals with content
	// and strength, and progress — the project's whole picture at the moment
	// you decide what to do next.
	b.WriteString(renderStageMarker("Project State (Colony)"))
	b.WriteString(renderPhaseEndFooter(state, phase.ID))
	b.WriteString(renderLifecycleClosingForState(result, state, "continue"))
	return b.String()
}

// renderLearningBeat renders phase-end consolidation's result as a single
// caste-styled "Learning" stage beat (D-06). raw is result["consolidation"]
// (attachConsolidationSummary's map[string]interface{}), which may be nil
// when no consolidation result was recorded at all -- e.g. an older report,
// or a code path that forgot to attach it. The beat is pure (no store, no
// I/O) and always renders something in one of four states: populated, zero,
// failed, or absent. Silence is not a reachable output (D-07).
func renderLearningBeat(raw interface{}) string {
	var b strings.Builder
	b.WriteString(renderStageMarker("Learning"))
	prefix := casteIdentity("librarian") + "  "

	consolidation, ok := raw.(map[string]interface{})
	if !ok || consolidation == nil {
		b.WriteString(prefix)
		b.WriteString("no consolidation result was recorded for this phase\n")
		return b.String()
	}

	summary := phaseEndConsolidationSummary{
		Ran:                 boolValue(consolidation["ran"]),
		Reason:              stringValue(consolidation["reason"]),
		PromotionCandidates: intValue(consolidation["promotion_candidates"]),
		QueenEligible:       intValue(consolidation["queen_eligible"]),
	}

	b.WriteString(prefix)
	b.WriteString(summary.LearningBeatLine())
	b.WriteString("\n")
	return b.String()
}

// improvementPassEventView is what renderImprovementPassBeat reduces both
// dual-typed shapes of result["improvement_pass"] to before rendering --
// deliberately carrying no candidate identifier at all, so this renderer is
// structurally unable to print one (204-12, D-03).
type improvementPassEventView struct {
	Kind   string
	Detail string
}

// improvementPassEventViewsFromRaw is the dual-type reduction
// renderImprovementPassBeat uses, following renderContinueGateDetail's own
// precedent: raw is result["improvement_pass"], either the in-process
// improvementPassSummary struct or the JSON-round-tripped
// map[string]interface{} attachConsolidationSummary's own snake_case-keyed
// shape produces.
func improvementPassEventViewsFromRaw(raw interface{}) []improvementPassEventView {
	switch v := raw.(type) {
	case improvementPassSummary:
		views := make([]improvementPassEventView, 0, len(v.Events))
		for _, e := range v.Events {
			views = append(views, improvementPassEventView{Kind: string(e.Kind), Detail: e.Detail})
		}
		return views
	case map[string]interface{}:
		// events may be []interface{} (after a JSON round-trip) or
		// []map[string]interface{} (attachConsolidationSummary's own
		// in-process shape, before any serialization) -- both are read
		// here so the map case is dual-safe on its own, not just this
		// function's outer struct/map switch.
		switch rawEvents := v["events"].(type) {
		case []interface{}:
			views := make([]improvementPassEventView, 0, len(rawEvents))
			for _, re := range rawEvents {
				entry, ok := re.(map[string]interface{})
				if !ok {
					continue
				}
				views = append(views, improvementPassEventView{
					Kind:   stringValue(entry["kind"]),
					Detail: stringValue(entry["detail"]),
				})
			}
			return views
		case []map[string]interface{}:
			views := make([]improvementPassEventView, 0, len(rawEvents))
			for _, entry := range rawEvents {
				views = append(views, improvementPassEventView{
					Kind:   stringValue(entry["kind"]),
					Detail: stringValue(entry["detail"]),
				})
			}
			return views
		default:
			return nil
		}
	default:
		return nil
	}
}

// improvementPassEventSentence renders one event view as one plain-English
// sentence, translating this repo's own invented words inline and never
// printing a raw verdict token, scope constant, status constant, or
// key=value pair. "compared" is deliberately absent from this switch: it is
// never its own closing-card line (see improvementPassEventKind's own doc
// comment) -- an empty result here is silently skipped by the caller.
func improvementPassEventSentence(e improvementPassEventView) string {
	switch improvementPassEventKind(e.Kind) {
	case improvementPassEventRefused:
		return "Tried a proposed change beside the current behaviour: it was not adopted -- it either did not do well enough, or touches something only you or an independent reviewer may change."
	case improvementPassEventStarted:
		return "A proposed change did well enough to be tried live, on a small, watched, reversible trial basis -- your project's current state was saved first, so it can be put back exactly if this does not work."
	case improvementPassEventCompleted:
		return "A trial run of a proposed change finished and kept passing -- the change stays."
	case improvementPassEventRolledBack:
		return "A trial run of a proposed change did not hold up -- your project has been put back exactly to the state it was saved in."
	case improvementPassEventProposalWritten:
		return fmt.Sprintf("The same kind of thing (%s) kept happening, so a plain-English case for a source change was written on its own isolated branch -- nothing has been merged, and a person must review it by hand before anything changes.", e.Detail)
	default:
		return ""
	}
}

// improvementPassEventVoiceGlyph picks the voiceLine glyph kind for e --
// mirroring the same status/done/failed distinctions this project's other
// closing-card lines already use.
func improvementPassEventVoiceGlyph(e improvementPassEventView) string {
	switch improvementPassEventKind(e.Kind) {
	case improvementPassEventCompleted:
		return "done"
	case improvementPassEventRolledBack:
		return "failed"
	default:
		return "status"
	}
}

// renderImprovementPassBeat renders the automatic improvement pass's own
// result (204-12, D-01, D-03) as zero or more plain-English lines -- one
// per card-worthy candidate outcome (a proposal refused, a trial started, a
// trial kept, or a trial undone) -- and renders NOTHING AT ALL when the
// pass did nothing (no declared candidate, no running canary). Unlike
// renderLearningBeat, an absent or empty result is not itself announced:
// silence about a pass that had nothing to do is the honest output, not a
// missing one.
func renderImprovementPassBeat(raw interface{}) string {
	events := improvementPassEventViewsFromRaw(raw)
	if len(events) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(renderStageMarker("Trying a Proposed Change"))
	for _, e := range events {
		line := improvementPassEventSentence(e)
		if line == "" {
			continue
		}
		b.WriteString(voiceLine(improvementPassEventVoiceGlyph(e), line))
		b.WriteString("\n")
	}
	return b.String()
}

// renderSealConsolidationBeats renders a seal consolidation attempt as a
// caste-styled "Consolidation" stage beat (D-06, LEARN-02). It is pure (no
// store, no I/O) and always renders something: eight distinct per-ant lines
// on success, or the loud "colony sealed WITHOUT consolidation" line on
// failure. Silence about consolidation is not a reachable output, mirroring
// renderLearningBeat's precedent.
func renderSealConsolidationBeats(s sealConsolidationSummary) string {
	var b strings.Builder
	b.WriteString(renderStageMarker("Consolidation"))

	if !s.Ran {
		b.WriteString("colony sealed WITHOUT consolidation — ")
		b.WriteString(strings.TrimSpace(s.Reason))
		b.WriteString("\n")
		return b.String()
	}

	for _, ant := range s.Ants {
		icon := "✗"
		if ant.Success {
			icon = "✓"
		}
		b.WriteString("  ")
		b.WriteString(icon)
		b.WriteString(" ")
		b.WriteString(casteIdentity(ant.Name))
		b.WriteString("  ")
		b.WriteString(ant.Detail)
		b.WriteString("\n")
	}

	b.WriteString(fmt.Sprintf("Instincts decayed: %d, archived: %d; observations decayed: %d\n", s.InstinctsDecayed, s.InstinctsArchived, s.ObservationsDecayed))

	reportLine := s.ReportPath
	if reportLine == "" {
		reportLine = "(not written)"
	}
	b.WriteString("Curation report: ")
	b.WriteString(reportLine)
	b.WriteString("\n")

	return b.String()
}

func renderContinuePlanOnlyVisual(state colony.ColonyState, phase colony.Phase, dispatches []codexContinueExternalDispatch, reviewDepth colony.VerificationDepth) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("continue"), "Continue Plan"))
	b.WriteString(visualDividerStr())
	b.WriteString("Verification snapshot and review manifest only. No state was changed and no review workers were spawned.\n")
	b.WriteString(renderProgressSummary(phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString(renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString("Phase: ")
	b.WriteString(phase.Name)
	b.WriteString("\n\n")
	if len(dispatches) > 0 {
		b.WriteString("Planned Continue Workers\n")
		lastWave := 0
		for _, dispatch := range dispatches {
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
			b.WriteString(dispatchTaskLine(dispatch.Task))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(renderStageMarker("How this run is being driven"))
	b.WriteString("The checking helpers listed above have not been sent yet, and nothing was changed.\n")
	b.WriteString("Whatever is running this sends them, then records what they found with\n")
	b.WriteString("`aether continue-finalize --completion-file <file>`.\n")
	b.WriteString(renderNextActionCard(lifecycleNextActionForState(state, "continue", "", "")))
	return b.String()
}

func renderContinueBlockedVisual(state colony.ColonyState, phase colony.Phase, result map[string]interface{}, reviewDepth colony.VerificationDepth) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("continue-blocked"), "Continue Blocked"))
	b.WriteString(visualDividerStr())
	b.WriteString(renderReviewDepthLine(reviewDepth, phase.ID, len(state.Plan.Phases)))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Phase %d remains active: %s\n", phase.ID, phase.Name))
	renderContinueVerificationSummaryMap(&b, continueTypedResultMapValue(result["verification"]))
	renderContinueVerificationDetail(&b, result["verification"])
	renderContinueGateSummaryMap(&b, continueTypedResultMapValue(result["gates"]))
	renderContinueGateDetail(&b, result["gates"])
	renderContinueWorkerFlowValue(&b, result["worker_flow"])
	renderSpecialistFindingBlocks(&b, result["worker_flow"])
	artifacts := []string{}
	if verificationReport := strings.TrimSpace(stringValue(result["verification_report"])); verificationReport != "" {
		artifacts = append(artifacts, verificationReport)
	}
	if gateReport := strings.TrimSpace(stringValue(result["gate_report"])); gateReport != "" {
		artifacts = append(artifacts, gateReport)
	}
	if reviewReport := strings.TrimSpace(stringValue(result["review_report"])); reviewReport != "" {
		artifacts = append(artifacts, reviewReport)
	}
	if continueReport := strings.TrimSpace(stringValue(result["continue_report"])); continueReport != "" {
		artifacts = append(artifacts, continueReport)
	}
	if len(artifacts) > 0 {
		artifacts = append(artifacts, displayDataPath("spawn-tree.txt"))
		b.WriteString(renderArtifactsSection(artifacts...))
	}
	if issues := stringSliceValue(result["operational_issues"]); len(issues) > 0 {
		b.WriteString("Operational issues\n")
		b.WriteString(renderIndentedList(issues))
	}
	renderCriterionEvidenceLines(&b, result["verification"])
	if blockers := stringSliceValue(result["blocking_issues"]); len(blockers) > 0 {
		b.WriteString("Blocking issues\n")
		b.WriteString(renderIndentedList(blockers))
		b.WriteString(renderBlockedWayForward(continueTypedResultMapValue(result["gates"])))
	}
	// D-11: a still-failing automatic repair's four-part handback -- what is
	// failing, what was tried and why it did not take, where the project
	// stands now, and the one thing to do next -- read before the closing
	// next-step line so the owner sees what happened before what to do.
	// Additive only: repairHandbackFromVerificationValue returns nil for
	// every outcome except a still-failing check-fix repair, so a blocked
	// screen with nothing to hand back is unchanged.
	if handback := repairHandbackFromVerificationValue(result["verification"]); handback != nil {
		b.WriteString(renderFailedRepairHandback(*handback))
	}
	b.WriteString(renderLifecycleClosingForState(result, state, "continue"))
	return b.String()
}

// renderBlockedWayForward turns the failed gates' fix hints and recovery
// options into the block's way forward — a critic that stops the line brings
// its fix in the same breath, and the Fixer (aether unblock) is always on the
// list so "it doesn't work" is never a dead end.
func renderBlockedWayForward(gates map[string]interface{}) string {
	lines := []string{}
	seen := map[string]bool{}
	appendLine := func(text string) {
		text = strings.TrimSpace(text)
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		lines = append(lines, text)
	}
	checks, _ := gates["checks"].([]interface{})
	for _, raw := range checks {
		check, _ := raw.(map[string]interface{})
		if check == nil {
			continue
		}
		if passed, _ := check["passed"].(bool); passed {
			continue
		}
		appendLine(stringValue(check["fix_hint"]))
		for _, option := range stringSliceValue(check["recovery_options"]) {
			appendLine(option)
		}
	}
	appendLine("Run aether unblock --dispatch to dispatch the Fixer against the blocking issues")

	var b strings.Builder
	b.WriteString("🧭 Way forward\n")
	shown := lines
	const maxWayForwardLines = 6
	if len(shown) > maxWayForwardLines {
		shown = shown[:maxWayForwardLines]
	}
	for _, line := range shown {
		b.WriteString("   └── ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	if extra := len(lines) - len(shown); extra > 0 {
		b.WriteString(fmt.Sprintf("   └── (+%d more in the gate report)\n", extra))
	}
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
	kind := "done"
	if failed := len(steps) - passed - skipped; failed > 0 {
		kind = "failed"
	}
	b.WriteString(voiceLine(kind, fmt.Sprintf("Verification: %d passed, %d skipped", passed, skipped)))
	b.WriteString("\n")
	if claims, ok := verification["claims"].(map[string]interface{}); ok {
		summary := strings.TrimSpace(stringValue(claims["summary"]))
		if summary != "" {
			b.WriteString(voiceLine("evidence", "Claims: "+summary))
			b.WriteString("\n")
		}
	}
}

// workerMeasurementFigures is the single formatter turning a worker's
// measured duration and tool-call count into display text -- used
// identically by the live finishing line (emitCodexDispatchWorkerFinished,
// cmd/codex_build_progress.go) and the continue worker-flow summary render
// below, so the two surfaces can never disagree about the same worker's
// figures (D-03). An unmeasured figure renders with the same "not reported"
// wording the cost line already uses (spendMarkNotReported,
// cmd/spend_cmd.go) rather than a fabricated zero (Phase 196 D-01) -- a
// worker that genuinely made zero tool calls still renders "0 tool calls".
func workerMeasurementFigures(durationSeconds float64, durationReported bool, toolCount int, toolCountReported bool) string {
	var durationPart, toolPart string
	if durationReported {
		durationPart = formatWorkerMeasuredDuration(durationSeconds)
	}
	if toolCountReported {
		toolPart = fmt.Sprintf("%d tool call", toolCount)
		if toolCount != 1 {
			toolPart += "s"
		}
	}

	switch {
	case durationReported && toolCountReported:
		return fmt.Sprintf("%s, %s", durationPart, toolPart)
	case durationReported:
		return fmt.Sprintf("%s, tool calls %s", durationPart, spendMarkNotReported)
	case toolCountReported:
		return fmt.Sprintf("duration %s, %s", spendMarkNotReported, toolPart)
	default:
		return spendMarkNotReported
	}
}

// formatWorkerMeasuredDuration renders a measured duration (in seconds) as
// plain English -- "3m 10s" once it crosses a minute, "%.1fs" below that --
// used only by workerMeasurementFigures above.
func formatWorkerMeasuredDuration(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	if seconds < 60 {
		return fmt.Sprintf("%.1fs", seconds)
	}
	total := int(seconds + 0.5)
	minutes := total / 60
	secs := total % 60
	return fmt.Sprintf("%dm %ds", minutes, secs)
}

func renderContinueWorkerFlowValue(b *strings.Builder, raw interface{}) {
	switch flow := raw.(type) {
	case []codexContinueWorkerFlowStep:
		if len(flow) == 0 {
			return
		}
		b.WriteString("Continue Worker Flow (the helpers who worked this phase)\n")
		for _, step := range flow {
			renderContinueWorkerFlowLine(b, step.Name, step.Caste, step.Status, step.Summary)
			findings := make([]string, 0, len(step.Findings))
			for _, finding := range step.Findings {
				label := strings.TrimSpace(finding.Title)
				if label == "" {
					label = strings.TrimSpace(finding.Description)
				}
				if severity := strings.TrimSpace(finding.Severity); severity != "" && label != "" {
					label = severity + ": " + label
				}
				if label != "" {
					findings = append(findings, label)
				}
			}
			measured := ""
			if step.Stage == "review" && step.Caste != "system" {
				measured = workerMeasurementFigures(step.Duration, step.DurationReported, step.ToolCount, step.ToolCountReported)
			}
			renderContinueWorkerFlowDetail(b, findings, step.Recommendations, step.WeakSpots, step.EdgeCases, step.Blockers, measured)
		}
	case []interface{}:
		renderContinueWorkerFlowMap(b, flow)
	}
}

func renderContinueWorkerFlowMap(b *strings.Builder, flow []interface{}) {
	if len(flow) == 0 {
		return
	}
	b.WriteString("Continue Worker Flow (the helpers who worked this phase)\n")
	for _, raw := range flow {
		step, _ := raw.(map[string]interface{})
		name := strings.TrimSpace(stringValue(step["name"]))
		if name == "" {
			continue
		}
		renderContinueWorkerFlowLine(b, name, stringValue(step["caste"]), stringValue(step["status"]), stringValue(step["summary"]))
		findings := []string{}
		if rawFindings, ok := step["findings"].([]interface{}); ok {
			for _, rawFinding := range rawFindings {
				finding, _ := rawFinding.(map[string]interface{})
				label := strings.TrimSpace(stringValue(finding["title"]))
				if label == "" {
					label = strings.TrimSpace(stringValue(finding["description"]))
				}
				if severity := strings.TrimSpace(stringValue(finding["severity"])); severity != "" && label != "" {
					label = severity + ": " + label
				}
				if label != "" {
					findings = append(findings, label)
				}
			}
		}
		measured := ""
		if stringValue(step["stage"]) == "review" && stringValue(step["caste"]) != "system" {
			measured = workerMeasurementFigures(
				floatValue(step["duration"]), boolValue(step["duration_reported"]),
				intValue(step["tool_count"]), boolValue(step["tool_count_reported"]),
			)
		}
		renderContinueWorkerFlowDetail(b, findings,
			stringSliceValue(step["recommendations"]),
			stringSliceValue(step["weak_spots"]),
			stringSliceValue(step["edge_cases_discovered"]),
			stringSliceValue(step["blockers"]),
			measured)
	}
}

func renderContinueWorkerFlowLine(b *strings.Builder, name, caste, status, summary string) {
	line := "  - "
	if caste = strings.TrimSpace(caste); caste != "" {
		line += casteIdentity(caste) + " "
	}
	line += strings.TrimSpace(name)
	if status = strings.TrimSpace(status); status != "" {
		line += " " + status
	}
	if summary = strings.TrimSpace(summary); summary != "" {
		line += " — " + summary
	}
	b.WriteString(line)
	b.WriteString("\n")
}

// renderContinueWorkerFlowDetail is the progressive-disclosure layer beneath
// each worker line: what the worker actually found, capped per category with
// an honest overflow count — the data was always carried, never shown.
// measured is the already-formatted workerMeasurementFigures output for a
// review-stage worker (empty string for non-review flow steps, which never
// went through a dispatch this plan measures) -- D-03: the same figures the
// live finishing line showed.
func renderContinueWorkerFlowDetail(b *strings.Builder, findings, recommendations, weakSpots, edgeCases, blockers []string, measured string) {
	const perCategoryCap = 2
	writeCategory := func(label string, items []string) {
		for i, item := range items {
			if i >= perCategoryCap {
				b.WriteString(fmt.Sprintf("      └── %s: (+%d more)\n", label, len(items)-perCategoryCap))
				return
			}
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			b.WriteString(fmt.Sprintf("      └── %s: %s\n", label, item))
		}
	}
	if measured = strings.TrimSpace(measured); measured != "" {
		b.WriteString(fmt.Sprintf("      └── measured: %s\n", measured))
	}
	writeCategory("found", findings)
	writeCategory("recommends", recommendations)
	writeCategory("weak spot", weakSpots)
	writeCategory("edge case", edgeCases)
	writeCategory("blocker", blockers)
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
	kind := "done"
	if passed < len(checks) {
		kind = "failed"
	}
	b.WriteString(voiceLine(kind, fmt.Sprintf("Gates: %d/%d passed", passed, len(checks))))
	b.WriteString("\n")
}

// continueDetailCap is the per-category nested-detail cap shared by
// renderContinueVerificationDetail and renderContinueGateDetail (D-10):
// beneath this many named lines, an honest "(+N more)" line reports the
// real arithmetic remainder rather than silently truncating. Set above the
// largest count either category produces today (4 verification checks, 6
// continue gates) so ordinary output is never capped; it exists to keep a
// pathological input from producing an unbounded screen.
const continueDetailCap = 8

// verificationStepDetailView is the shape both the in-process typed
// codexVerificationStep and the JSON-round-tripped map entry are reduced to
// before rendering, so renderVerificationStepDetailLines has exactly one
// formatting implementation for both (the dual-type rendering pattern
// renderContinueWorkerFlowValue already established, 198-PATTERNS.md).
type verificationStepDetailView struct {
	Name     string
	Skipped  bool
	Passed   bool
	Summary  string
	Duration float64
	Command  string
}

// formatVerificationStepResultLine is the single formatter for a
// verification check's outcome text -- shared by the live progress line
// (emitVerificationStepFinish) and the closing verification detail
// (renderVerificationStepDetailLines) so the two surfaces can never disagree
// about the same check's result (D-01/D-03's "one source" rule applied to
// verification checks). A skipped check names its reason and claims no
// elapsed time; a passed check shows its measured duration when one was
// recorded; a failed check carries its own Summary inline.
func formatVerificationStepResultLine(label string, skipped, passed bool, summary string, duration float64) string {
	if skipped {
		reason := strings.TrimSpace(summary)
		if reason == "" {
			reason = "skipped"
		}
		return fmt.Sprintf("%s — skipped: %s", label, reason)
	}
	durationText := ""
	if duration > 0 {
		durationText = fmt.Sprintf(" (%.1fs)", duration)
	}
	if passed {
		return fmt.Sprintf("%s ✓%s", label, durationText)
	}
	reason := strings.TrimSpace(summary)
	if reason == "" {
		reason = "failed"
	}
	return fmt.Sprintf("%s ✗%s — %s", label, durationText, reason)
}

// renderContinueVerificationDetail names every check the verification tally
// line summarised (SHOW-02): the plain-English check name, its pass/fail/
// skip mark, and its measured elapsed time -- never the internal key. A
// failed check's command goes on the nested "└──" detail line beneath it.
// Dual-type: raw is result["verification"], which is either the in-process
// typed codexContinueVerificationReport struct or the JSON-round-tripped
// map[string]interface{} a completion file produces.
func renderContinueVerificationDetail(b *strings.Builder, raw interface{}) {
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		renderVerificationStepDetailLines(b, verificationStepDetailViewsFromTyped(v.Steps))
	case map[string]interface{}:
		steps, _ := v["steps"].([]interface{})
		renderVerificationStepDetailLines(b, verificationStepDetailViewsFromMap(steps))
	}
}

// repairHandbackFromVerificationValue is D-11's dual-type reader for the
// verification report's RepairHandback field -- raw is result["verification"],
// either the in-process typed codexContinueVerificationReport or the
// JSON-round-tripped map[string]interface{} a completion file produces
// (renderContinueVerificationDetail's own precedent, just above). Returns
// nil whenever no handback is present, so a blocked screen with nothing to
// hand back renders no section at all.
func repairHandbackFromVerificationValue(raw interface{}) *repairHandback {
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		return v.RepairHandback
	case map[string]interface{}:
		hbRaw, ok := v["repair_handback"]
		if !ok || hbRaw == nil {
			return nil
		}
		hbMap, ok := hbRaw.(map[string]interface{})
		if !ok {
			return nil
		}
		actionMap, _ := hbMap["owner_action"].(map[string]interface{})
		handback := repairHandback{
			Diagnosis:        stringValue(hbMap["diagnosis"]),
			AttemptedAndWhy:  stringValue(hbMap["attempted_and_why"]),
			RestoredPosition: stringValue(hbMap["restored_position"]),
			OwnerAction: LifecycleCloseoutRecommendedAction{
				Command:      stringValue(actionMap["command"]),
				Reason:       stringValue(actionMap["reason"]),
				Alternatives: stringSliceValue(actionMap["alternatives"]),
			},
		}
		return &handback
	default:
		return nil
	}
}

func verificationStepDetailViewsFromTyped(steps []codexVerificationStep) []verificationStepDetailView {
	views := make([]verificationStepDetailView, 0, len(steps))
	for _, step := range steps {
		views = append(views, verificationStepDetailView{
			Name:     step.Name,
			Skipped:  step.Skipped,
			Passed:   step.Passed,
			Summary:  step.Summary,
			Duration: step.Duration,
			Command:  step.Command,
		})
	}
	return views
}

func verificationStepDetailViewsFromMap(steps []interface{}) []verificationStepDetailView {
	views := make([]verificationStepDetailView, 0, len(steps))
	for _, raw := range steps {
		entry, _ := raw.(map[string]interface{})
		if entry == nil {
			continue
		}
		views = append(views, verificationStepDetailView{
			Name:     stringValue(entry["name"]),
			Skipped:  boolValue(entry["skipped"]),
			Passed:   boolValue(entry["passed"]),
			Summary:  stringValue(entry["summary"]),
			Duration: floatValue(entry["duration_seconds"]),
			Command:  stringValue(entry["command"]),
		})
	}
	return views
}

func renderVerificationStepDetailLines(b *strings.Builder, views []verificationStepDetailView) {
	if len(views) == 0 {
		return
	}
	shown := views
	overflow := 0
	if len(shown) > continueDetailCap {
		overflow = len(shown) - continueDetailCap
		shown = shown[:continueDetailCap]
	}
	for _, v := range shown {
		label := verificationStepDisplayName(v.Name)
		kind := "done"
		switch {
		case v.Skipped:
			kind = "status"
		case !v.Passed:
			kind = "failed"
		}
		b.WriteString("  ")
		b.WriteString(voiceLine(kind, formatVerificationStepResultLine(label, v.Skipped, v.Passed, v.Summary, v.Duration)))
		b.WriteString("\n")
		if !v.Skipped && !v.Passed {
			if cmd := strings.TrimSpace(v.Command); cmd != "" {
				b.WriteString("      └── ")
				b.WriteString(cmd)
				b.WriteString("\n")
			}
		}
	}
	if overflow > 0 {
		b.WriteString(fmt.Sprintf("  └── (+%d more checks)\n", overflow))
	}
}

// gateCheckDisplayNames is the runtime's own translation table from a
// gate's internal snake_case key to the plain-English sentence the owner
// sees. gateCheckDisplayName's lookup reads from this table. The internal
// key never reaches the owner directly (CLAUDE.md "translate every repo
// word inline"). This covers every key in gateClassifications (cmd/gate.go
// -- the classified security/quality gates) plus the four structural keys
// below that come from the continue flow's own checks rather than
// gate.go -- see continueGateCheckNames, which is derived from those two
// real sources rather than from this map, so a gate name added to
// gateClassifications without a translation here fails
// TestRestoredDetailIsPlainEnglish instead of silently passing.
var gateCheckDisplayNames = map[string]string{
	// Structural keys (continue flow's own checks, not in gateClassifications)
	"manifest_present":           "the build's own plan file is on disk",
	"verification_steps_passed":  "the build/test checks passed",
	"implementation_evidence":    "there's evidence the work was actually done",
	"owner_confirmation_pending": "nothing is waiting on your confirmation",
	// hard_block gates (cmd/gate.go gateClassifications)
	"gatekeeper":                  "the security check found nothing concerning",
	"watcher_veto":                "the quality reviewer didn't block the work",
	"flags":                       "nothing was flagged that needs your attention",
	"tests_pass":                  "the tests passed",
	"no_critical_flags":           "no critical problems were flagged",
	"anti_pattern_executed":       "the risky-code scan actually ran",
	"charter_compliance_executed": "the project's own rules check actually ran",
	// soft_block gates (cmd/gate.go gateClassifications)
	"auditor":            "the quality reviewer's checks passed",
	"complexity":         "the code wasn't too complex to maintain",
	"tdd_evidence":       "there's evidence tests were written before the code",
	"anti_pattern":       "no risky code patterns were found",
	"charter_compliance": "the project's own rules were followed",
	"verification_loop":  "the verification steps completed",
	"spawn_gate":         "every helper that was needed was actually sent",
	// advisory gates (cmd/gate.go gateClassifications)
	"medic":   "the health check found nothing needing attention",
	"runtime": "nothing went wrong while it was running",
}

// continueStructuralGateNames lists the four gate keys used by the continue
// flow's own structural checks -- they are not part of gate.go's
// gateClassifications table (that table only covers the classified
// security/quality gates), but they still need a plain-English translation
// on this screen.
var continueStructuralGateNames = []string{
	"manifest_present",
	"verification_steps_passed",
	"implementation_evidence",
	"owner_confirmation_pending",
}

// continueGateCheckNames is derived from the runtime's real universe of gate
// keys -- gateClassifications (cmd/gate.go, every classified security/quality
// gate) unioned with continueStructuralGateNames (the continue flow's own
// structural checks) -- rather than from gateCheckDisplayNames itself. That
// way, a gate name that is added to the runtime but never given a
// plain-English translation actually fails TestRestoredDetailIsPlainEnglish,
// instead of the test tautologically passing because the untranslated name
// was never in its own input list.
var continueGateCheckNames = func() []string {
	names := make([]string, 0, len(gateClassifications)+len(continueStructuralGateNames))
	for name := range gateClassifications {
		names = append(names, name)
	}
	names = append(names, continueStructuralGateNames...)
	return names
}()

// gateCheckDisplayName translates a gate's internal key into the
// plain-English sentence the owner sees, via gateCheckDisplayNames above.
func gateCheckDisplayName(name string) string {
	key := strings.ToLower(strings.TrimSpace(name))
	if display, ok := gateCheckDisplayNames[key]; ok {
		return display
	}
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "check"
	}
	return strings.ReplaceAll(trimmed, "_", " ")
}

// gateCheckDetailView is the shape both the in-process typed gateCheck and
// the JSON-round-tripped map entry are reduced to before rendering (same
// dual-type pattern as verificationStepDetailView).
type gateCheckDetailView struct {
	Name    string
	Passed  bool
	FixHint string
}

// renderContinueGateDetail names every gate the "Gates: N/M passed" tally
// line summarised (SHOW-02): the gate translated into plain English and its
// outcome, with a failing gate's fix hint on the nested "└──" detail line.
// Dual-type: raw is result["gates"], either the in-process typed
// codexContinueGateReport struct or the JSON-round-tripped map a completion
// file produces.
func renderContinueGateDetail(b *strings.Builder, raw interface{}) {
	switch v := raw.(type) {
	case codexContinueGateReport:
		views := make([]gateCheckDetailView, 0, len(v.Checks))
		for _, c := range v.Checks {
			views = append(views, gateCheckDetailView{Name: c.Name, Passed: c.Passed, FixHint: c.FixHint})
		}
		renderGateCheckDetailLines(b, views)
	case map[string]interface{}:
		checks, _ := v["checks"].([]interface{})
		views := make([]gateCheckDetailView, 0, len(checks))
		for _, raw := range checks {
			entry, _ := raw.(map[string]interface{})
			if entry == nil {
				continue
			}
			views = append(views, gateCheckDetailView{
				Name:    stringValue(entry["name"]),
				Passed:  boolValue(entry["passed"]),
				FixHint: stringValue(entry["fix_hint"]),
			})
		}
		renderGateCheckDetailLines(b, views)
	}
}

func renderGateCheckDetailLines(b *strings.Builder, views []gateCheckDetailView) {
	if len(views) == 0 {
		return
	}
	shown := views
	overflow := 0
	if len(shown) > continueDetailCap {
		overflow = len(shown) - continueDetailCap
		shown = shown[:continueDetailCap]
	}
	for _, v := range shown {
		mark := "✓"
		kind := "done"
		if !v.Passed {
			mark = "✗"
			kind = "failed"
		}
		b.WriteString(voiceLine(kind, fmt.Sprintf("%s %s", mark, gateCheckDisplayName(v.Name))))
		b.WriteString("\n")
		if !v.Passed {
			if hint := strings.TrimSpace(v.FixHint); hint != "" {
				b.WriteString("      └── ")
				b.WriteString(hint)
				b.WriteString("\n")
			}
		}
	}
	if overflow > 0 {
		b.WriteString(fmt.Sprintf("  └── (+%d more gates)\n", overflow))
	}
}

// criterionEvidenceView is the shape both the in-process typed
// codexCriterionVerification and the JSON-round-tripped map entry are
// reduced to before rendering (D-11).
type criterionEvidenceView struct {
	Criterion      string
	Evidence       []string
	Summary        string
	State          string
	Passed         bool
	BlockingIssues []string
}

// criterionEvidenceDisplayState computes, from the criterion's own fields
// and nothing else, which of the four D-11 display states a requirement is
// in -- "satisfied", "awaiting_owner", "blocked", or "unproven". Both the
// mark (✓/⏳/✗/?) and the words used ("proved by"/"awaiting your
// confirmation"/etc.) are derived from this single function's return value,
// so a rendering change can never make an unproven requirement look
// satisfied (198-06-PLAN.md Task 2 behavior: "the mark and the state must
// come from the same field").
func criterionEvidenceDisplayState(v criterionEvidenceView) string {
	if strings.EqualFold(strings.TrimSpace(v.State), criterionStateNeedsOwnerConfirmation) {
		return "awaiting_owner"
	}
	if !v.Passed || len(v.BlockingIssues) > 0 {
		return "blocked"
	}
	if len(v.Evidence) == 0 {
		return "unproven"
	}
	return "satisfied"
}

// renderCriterionEvidenceLines renders D-11's requirement-plus-proof shape
// for every criterion the verification report carries: "✓ Login works —
// proved by: 3 tests passed, auth.go present". A requirement is never shown
// with a satisfied mark unless it is genuinely proven (see
// criterionEvidenceDisplayState); an unproven, blocked, or
// awaiting-owner-confirmation requirement names its own state in plain
// English instead. Dual-type: raw is result["verification"], either the
// in-process typed codexContinueVerificationReport struct or the JSON-
// round-tripped map a completion file produces.
func renderCriterionEvidenceLines(b *strings.Builder, raw interface{}) {
	switch v := raw.(type) {
	case codexContinueVerificationReport:
		renderCriterionEvidenceViewLines(b, criterionEvidenceViewsFromTyped(v.Criteria))
	case map[string]interface{}:
		criteria, _ := v["criteria"].([]interface{})
		renderCriterionEvidenceViewLines(b, criterionEvidenceViewsFromMap(criteria))
	}
}

func criterionEvidenceViewsFromTyped(criteria []codexCriterionVerification) []criterionEvidenceView {
	views := make([]criterionEvidenceView, 0, len(criteria))
	for _, c := range criteria {
		views = append(views, criterionEvidenceView{
			Criterion:      c.Criterion,
			Evidence:       c.Evidence,
			Summary:        c.Summary,
			State:          c.State,
			Passed:         c.Passed,
			BlockingIssues: c.BlockingIssues,
		})
	}
	return views
}

func criterionEvidenceViewsFromMap(criteria []interface{}) []criterionEvidenceView {
	views := make([]criterionEvidenceView, 0, len(criteria))
	for _, raw := range criteria {
		entry, _ := raw.(map[string]interface{})
		if entry == nil {
			continue
		}
		views = append(views, criterionEvidenceView{
			Criterion:      stringValue(entry["criterion"]),
			Evidence:       stringSliceValue(entry["evidence"]),
			Summary:        stringValue(entry["summary"]),
			State:          stringValue(entry["state"]),
			Passed:         boolValue(entry["passed"]),
			BlockingIssues: stringSliceValue(entry["blocking_issues"]),
		})
	}
	return views
}

func renderCriterionEvidenceViewLines(b *strings.Builder, views []criterionEvidenceView) {
	if len(views) == 0 {
		return
	}
	b.WriteString("📋 Requirement Evidence\n")
	shown := views
	overflow := 0
	if len(shown) > continueDetailCap {
		overflow = len(shown) - continueDetailCap
		shown = shown[:continueDetailCap]
	}
	for _, v := range shown {
		label := strings.TrimSpace(v.Criterion)
		if label == "" {
			label = "(unnamed requirement)"
		}
		switch criterionEvidenceDisplayState(v) {
		case "satisfied":
			b.WriteString("  ")
			b.WriteString(voiceLine("evidence", fmt.Sprintf("✓ %s — proved by: %s", label, strings.Join(v.Evidence, ", "))))
			b.WriteString("\n")
		case "awaiting_owner":
			b.WriteString("  ")
			b.WriteString(voiceLine("evidence", fmt.Sprintf("⏳ %s — awaiting your confirmation", label)))
			b.WriteString("\n")
		case "blocked":
			reason := strings.TrimSpace(v.Summary)
			if reason == "" && len(v.BlockingIssues) > 0 {
				reason = strings.Join(v.BlockingIssues, "; ")
			}
			if reason == "" {
				reason = "blocked"
			}
			b.WriteString("  ")
			b.WriteString(voiceLine("evidence", fmt.Sprintf("✗ %s — %s", label, reason)))
			b.WriteString("\n")
		default: // unproven
			b.WriteString("  ")
			b.WriteString(voiceLine("evidence", fmt.Sprintf("? %s — unproven: no evidence recorded", label)))
			b.WriteString("\n")
		}
	}
	if overflow > 0 {
		b.WriteString(fmt.Sprintf("  └── (+%d more requirements)\n", overflow))
	}
}

// specialistFindingView is the shape both the in-process typed
// codexContinueWorkerFlowStep and the JSON-round-tripped map entry are
// reduced to before rendering the specialist-findings block (Task 2, plan
// 198-06): the data was always carried on worker_flow, previously visible
// only by reading every worker's own nested detail line-by-line.
type specialistFindingView struct {
	Name            string
	Caste           string
	Findings        []string
	Recommendations []string
	WeakSpots       []string
	EdgeCases       []string
	Blockers        []string
}

func specialistFindingViewIsEmpty(v specialistFindingView) bool {
	return len(v.Findings) == 0 && len(v.Recommendations) == 0 && len(v.WeakSpots) == 0 && len(v.EdgeCases) == 0 && len(v.Blockers) == 0
}

// renderSpecialistFindingBlocks renders every reviewer's findings,
// recommendations, weak spots, edge cases, and blockers as their own headed
// block -- visible without reading every worker's per-line nested detail
// (198-06-PLAN.md Task 2, item 4). Dual-type: raw is result["worker_flow"],
// either the in-process typed []codexContinueWorkerFlowStep slice or the
// JSON-round-tripped []interface{} a completion file produces.
func renderSpecialistFindingBlocks(b *strings.Builder, raw interface{}) {
	var views []specialistFindingView
	switch flow := raw.(type) {
	case []codexContinueWorkerFlowStep:
		views = specialistFindingViewsFromTyped(flow)
	case []interface{}:
		views = specialistFindingViewsFromMap(flow)
	}
	if len(views) == 0 {
		return
	}
	b.WriteString("🔍 Specialist Findings\n")
	const perWorkerCategoryCap = 3
	for _, v := range views {
		line := "  - "
		if v.Caste != "" {
			line += casteIdentity(v.Caste) + " "
		}
		line += strings.TrimSpace(v.Name)
		b.WriteString(line)
		b.WriteString("\n")
		writeSpecialistFindingCategory(b, "found", v.Findings, perWorkerCategoryCap)
		writeSpecialistFindingCategory(b, "recommends", v.Recommendations, perWorkerCategoryCap)
		writeSpecialistFindingCategory(b, "weak spot", v.WeakSpots, perWorkerCategoryCap)
		writeSpecialistFindingCategory(b, "edge case", v.EdgeCases, perWorkerCategoryCap)
		writeSpecialistFindingCategory(b, "blocker", v.Blockers, perWorkerCategoryCap)
	}
}

func writeSpecialistFindingCategory(b *strings.Builder, label string, items []string, cap int) {
	for i, item := range items {
		if i >= cap {
			b.WriteString(fmt.Sprintf("      └── %s: (+%d more)\n", label, len(items)-cap))
			return
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("      └── %s: %s\n", label, item))
	}
}

func specialistFindingViewsFromTyped(flow []codexContinueWorkerFlowStep) []specialistFindingView {
	views := make([]specialistFindingView, 0, len(flow))
	for _, step := range flow {
		findings := make([]string, 0, len(step.Findings))
		for _, finding := range step.Findings {
			label := strings.TrimSpace(finding.Title)
			if label == "" {
				label = strings.TrimSpace(finding.Description)
			}
			if severity := strings.TrimSpace(finding.Severity); severity != "" && label != "" {
				label = severity + ": " + label
			}
			if label != "" {
				findings = append(findings, label)
			}
		}
		v := specialistFindingView{
			Name:            step.Name,
			Caste:           step.Caste,
			Findings:        findings,
			Recommendations: step.Recommendations,
			WeakSpots:       step.WeakSpots,
			EdgeCases:       step.EdgeCases,
			Blockers:        step.Blockers,
		}
		if specialistFindingViewIsEmpty(v) {
			continue
		}
		views = append(views, v)
	}
	return views
}

func specialistFindingViewsFromMap(flow []interface{}) []specialistFindingView {
	views := make([]specialistFindingView, 0, len(flow))
	for _, raw := range flow {
		step, _ := raw.(map[string]interface{})
		if step == nil {
			continue
		}
		findings := []string{}
		if rawFindings, ok := step["findings"].([]interface{}); ok {
			for _, rawFinding := range rawFindings {
				finding, _ := rawFinding.(map[string]interface{})
				label := strings.TrimSpace(stringValue(finding["title"]))
				if label == "" {
					label = strings.TrimSpace(stringValue(finding["description"]))
				}
				if severity := strings.TrimSpace(stringValue(finding["severity"])); severity != "" && label != "" {
					label = severity + ": " + label
				}
				if label != "" {
					findings = append(findings, label)
				}
			}
		}
		v := specialistFindingView{
			Name:            stringValue(step["name"]),
			Caste:           stringValue(step["caste"]),
			Findings:        findings,
			Recommendations: stringSliceValue(step["recommendations"]),
			WeakSpots:       stringSliceValue(step["weak_spots"]),
			EdgeCases:       stringSliceValue(step["edge_cases_discovered"]),
			Blockers:        stringSliceValue(step["blockers"]),
		}
		if specialistFindingViewIsEmpty(v) {
			continue
		}
		views = append(views, v)
	}
	return views
}

func mapValue(raw interface{}) map[string]interface{} {
	value, _ := raw.(map[string]interface{})
	return value
}

// continueTypedResultMapValue converts a continue result field's value into
// map[string]interface{} regardless of whether it arrived as the in-process
// typed struct (codexContinueVerificationReport, codexContinueGateReport)
// or as the JSON-round-tripped map a completion file produces. mapValue alone
// only handles the already-map case; a raw typed struct silently resolves to
// len==0 there and the whole section renders as if empty -- the dual-type
// rendering precedent renderContinueWorkerFlowValue already established
// (198-PATTERNS.md) applied here so the direct path and the closeout path
// show identical detail from the identical underlying value.
func continueTypedResultMapValue(raw interface{}) map[string]interface{} {
	if m, ok := raw.(map[string]interface{}); ok {
		return m
	}
	if raw == nil {
		return nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil
	}
	return decoded
}

// renderDecisionBlock (SEE-06) is the one visually distinct frame for moments
// that need the operator: a halted wave, a tripped breaker, a paused
// autopilot. One shape everywhere, so "the colony needs you" is recognizable
// at a glance instead of buried in prose.
func renderDecisionBlock(emoji, title string, lines ...string) string {
	var b strings.Builder
	b.WriteString("━━━ " + emoji + " " + spacedTitle(title) + " ━━━\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// renderProjectComplete is the classic v5.4.0 project-complete celebration.
// It fires once, when the final phase advances — from the autopilot loop and
// from a final `aether continue` — and the runtime owns it: wrappers must not
// hand-render this banner.
func renderProjectComplete(state colony.ColonyState, phasesCompleted int) string {
	goal := "(no goal recorded)"
	if state.Goal != nil && strings.TrimSpace(*state.Goal) != "" {
		goal = strings.TrimSpace(*state.Goal)
	}
	total := phasesCompleted
	if len(state.Plan.Phases) > total {
		total = len(state.Plan.Phases)
	}
	var b strings.Builder
	rule := strings.Repeat("━", 50)
	b.WriteString(rule + "\n")
	b.WriteString("   🎉 " + spacedTitle("Project Complete") + " 🎉\n")
	b.WriteString(rule + "\n\n")
	b.WriteString(fmt.Sprintf("👑 Goal Achieved: %s\n", goal))
	b.WriteString(fmt.Sprintf("📍 Phases Completed: %d\n\n", total))
	b.WriteString("🐜 The project (this colony) rests. Well done!")
	return b.String()
}

// crownedAnthillArt is the classic v5.4.0 seal ceremony drawing (seal.yaml
// Step 7), byte-faithful to the original.
const crownedAnthillArt = `        .     .
       /|\   /|\
      / | \ / | \
     /  |  X  |  \
    /   | / \ |   \
   /    |/   \|    \
  /     /     \     \
 /____ /  ___  \ ____\
      / /   \ \
     / /     \ \
    /_/       \_\
     |  CROWNED |
     | ANTHILL  |
     |__________|`

func renderSealVisual(result map[string]interface{}, state colony.ColonyState, summaryPath string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("seal"), "Seal"))
	b.WriteString(visualDividerStr())
	// The classic crowning ceremony: the anthill drawing, the letter-spaced
	// title with the colony's version, then the facts.
	b.WriteString(crownedAnthillArt)
	b.WriteString("\n\n")
	rule := strings.Repeat("━", 50)
	b.WriteString(rule + "\n")
	b.WriteString(fmt.Sprintf("   %s   v%d\n", spacedTitle("Crowned Anthill"), state.ColonyVersion))
	b.WriteString(rule + "\n\n")
	b.WriteString(renderStageMarker("Summary"))
	b.WriteString(voiceLine("milestone", "This project (the colony) is now finished — sealed at Crowned Anthill."))
	b.WriteString("\n")
	if state.Goal != nil {
		b.WriteString(voiceLine("goal", "Goal: "+*state.Goal))
		b.WriteString("\n")
	}
	b.WriteString(voiceLine("phase", fmt.Sprintf("Completed phases: %d", len(state.Plan.Phases))))
	b.WriteString("\n")
	b.WriteString(voiceLine("artifact", "Summary: "+summaryPath))
	b.WriteString("\n\n")
	b.WriteString(voiceLine("milestone", "The project (this colony) stands crowned and finished (sealed)."))
	b.WriteString("\n")
	b.WriteString(voiceLine("learning", "The coordinator (Queen) that decides your team keeps this wisdom in QUEEN.md for next time."))
	b.WriteString("\n")
	b.WriteString(voiceLine("archive", "The anthill has reached its final form."))
	b.WriteString("\n")
	// The card below is the one resolver's answer for a just-sealed project:
	// it explains, in plain words, what finishing means and what archiving it
	// would do (S-05) -- state was saved with the final milestone before this
	// renders, so the resolver's own colonyNeedsEntomb branch applies.
	b.WriteString(renderLifecycleClosing(result, "seal"))
	return b.String()
}

func renderSignalVisual(sigType, content, priority string, replaced bool) string {
	emoji := signalTypeGlyph(sigType)

	status := "New signal laid."
	if replaced {
		status = "Existing signal reinforced."
	}

	var b strings.Builder
	b.WriteString(renderBanner(emoji, sigType+" Signal"))
	b.WriteString(visualDividerStr())
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
	b.WriteString(visualDividerStr())
	b.WriteString(renderAetherWordmark())
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
	b.WriteString(renderInstallPaths(binaryMode))
	b.WriteString(renderNextUp(
		`Run `+"`aether lay-eggs`"+` inside a repo to set up a local nest.`,
		`Run `+"`aether update --force`"+` in existing repos to pull the refreshed companion files.`,
	))
	return b.String()
}

func renderSetupVisual(repoDir string, results []map[string]interface{}, totalCopied, totalSkipped int, restartTargets []string) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("lay-eggs"), "Lay Eggs"))
	b.WriteString(visualDividerStr())
	b.WriteString("Nest prepared in this repository.\n")
	b.WriteString("Repo: ")
	b.WriteString(repoDir)
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Assets: %d copied, %d unchanged\n\n", totalCopied, totalSkipped))
	b.WriteString(renderSyncSummary(results))
	if restartNote := platformRestartMessage(restartTargets); restartNote != "" {
		b.WriteString("\nSession Refresh\n")
		b.WriteString("  ")
		b.WriteString(restartNote)
		b.WriteString("\n")
	}
	primaryNext := `Run ` + "`aether init \"your goal\"`" + ` to start a colony.`
	secondaryNext := `Run ` + "`aether colonize`" + ` after init if you want a quick territory scan before planning.`
	if len(restartTargets) > 0 {
		primaryNext = `Restart your session in this repo (Codex or OpenCode), then run ` + "`aether init \"your goal\"`" + `.`
		secondaryNext = `After restarting, run ` + "`aether colonize`" + ` if you want a quick territory scan before planning.`
	}
	b.WriteString(renderNextUp(
		primaryNext,
		secondaryNext,
	))
	return b.String()
}

func renderUpdateVisual(repoDir, hubVersion, localVersion, repoTransition string, force, dryRun bool, details []map[string]interface{}, totalCopied, totalSkipped int, restartTargets []string, binaryMode string, versionsMatch bool, result map[string]interface{}) string {
	var b strings.Builder
	totalRemoved := syncDetailsRemoved(details)
	b.WriteString(renderBanner(commandEmoji("update"), "Update"))
	b.WriteString(visualDividerStr())
	b.WriteString(renderAetherWordmark())
	if dryRun {
		b.WriteString("Dry run complete. No files were changed.\n")
	} else {
		b.WriteString("Companion files refreshed from the hub.\n")
	}
	b.WriteString("Repo: ")
	b.WriteString(repoDir)
	b.WriteString("\n")
	// The repo's own before/after is the question `/ant-update` exists to
	// answer; hub and binary versions alone never told the user whether they
	// had actually been behind.
	if repoTransition != "" {
		b.WriteString(repoTransition)
	}
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
		b.WriteString(fmt.Sprintf("Assets: %d copied, %d unchanged", totalCopied, totalSkipped))
		if totalRemoved > 0 {
			b.WriteString(fmt.Sprintf(", %d removed", totalRemoved))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(renderSyncSummary(details))
	b.WriteString("\nRuntime\n")
	for _, line := range updateRuntimeLines(binaryMode, versionsMatch) {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString(renderUpdatePaths(force, binaryMode))
	if dryRun {
		next := `Run ` + "`aether update`" + ` to apply the previewed changes.`
		alt := `Run ` + "`aether update --force`" + ` only if you want tracked companion files overwritten and stale files removed.`
		runtimeAlt := `Run ` + "`aether update --download-binary`" + ` only if you need a published runtime update as well.`
		if force {
			next = `Run ` + "`aether update --force`" + ` to apply the forced sync.`
			alt = `Run ` + "`aether update`" + ` instead for the safer non-force path.`
			runtimeAlt = `Run ` + "`aether publish --package-dir <Aether checkout>`" + ` in the Aether repo first if you need an unreleased local runtime fix on this machine.`
		}
		b.WriteString(renderNextUp(next, alt, runtimeAlt))
		return b.String()
	}
	if restartNote := platformRestartMessage(restartTargets); restartNote != "" {
		b.WriteString("\nSession Refresh\n")
		b.WriteString("  ")
		b.WriteString(restartNote)
		b.WriteString("\n")
	}
	// Both branches below used to hand-write their own recommendation. The
	// one resolver's answer is about the PROJECT, not about update's own
	// flags, so it replaces them here; the restart note above (when present)
	// and the repair report folded into "changed" below cover what update
	// itself did.
	b.WriteString(renderLifecycleClosing(result, "update"))
	return b.String()
}

func syncDetailsRemoved(details []map[string]interface{}) int {
	total := 0
	for _, entry := range details {
		total += intValue(entry["removed"])
	}
	return total
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

func renderInstallPaths(binaryMode string) string {
	lines := []string{
		`Published release repos: use ` + "`aether update --force --download-binary`" + ` when you want companion files and the matching released binary together.`,
		`Local Aether development: use ` + "`aether publish --channel stable --binary-dest \"$HOME/.local/bin\"`" + ` in the Aether repo when testing unreleased changes on this machine.`,
	}
	if strings.TrimSpace(binaryMode) == "release-download" {
		lines[0] = "Published release repos: this install will refresh companion files first and download the published binary next."
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Paths"))
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func updateRuntimeLines(binaryMode string, versionsMatch bool) []string {
	switch strings.TrimSpace(binaryMode) {
	case "release-download-preview":
		return []string{
			"Companion files would be synced first, then a published release binary would be downloaded.",
			"`aether update --download-binary` only installs released runtime builds, not unreleased local source changes.",
			"For an unreleased local runtime fix on this machine, run `aether publish --package-dir <Aether checkout>` in the Aether repo first.",
		}
	case "release-download":
		return []string{
			"Companion files were synced first; a published release binary will be downloaded next.",
			"`aether update --download-binary` only installs released runtime builds, not unreleased local source changes.",
		}
	default:
		if versionsMatch {
			return []string{
				"Binary: already at the current hub version. No action needed.",
				"`aether update` syncs companion files only. The shared binary is updated via `aether publish` in the Aether repo.",
			}
		}
		return []string{
			"Binary: version differs from hub. The shared binary is not updated by `aether update`.",
			"For a published runtime update, run `aether update --download-binary`.",
			"For an unreleased local runtime fix on this machine, run `aether publish` in the Aether repo first.",
		}
	}
}

func renderUpdatePaths(force bool, binaryMode string) string {
	releaseCmd := "`aether update --download-binary`"
	if force {
		releaseCmd = "`aether update --force --download-binary`"
	}

	lines := []string{
		`Published release: run ` + releaseCmd + ` to keep companion files and the matching released binary in lockstep.`,
		`Local source checkout: run ` + "`aether publish --package-dir <Aether checkout>`" + ` in the Aether repo, then rerun ` + "`aether update --force`" + ` here.`,
	}

	switch strings.TrimSpace(binaryMode) {
	case "release-download-preview":
		lines[0] = "Published release: this preview includes a matching release binary download after the companion-file sync."
	case "release-download":
		lines[0] = "Published release: companion files and the matching released binary are being installed together now."
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Paths"))
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func renderBinaryActionVisual(title, message, version, path string) string {
	var b strings.Builder
	b.WriteString(renderBanner("⚡", title))
	b.WriteString(visualDividerStr())
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
	b.WriteString(renderBinaryActionNextUp(title))
	return b.String()
}

func renderBinaryActionNextUp(title string) string {
	switch strings.ToLower(strings.TrimSpace(title)) {
	case "publish complete":
		return renderNextUp(
			`Existing repos: run `+"`aether update --force`"+` to refresh companion files from the hub.`,
			`New repos: run `+"`aether lay-eggs`"+` to set up Aether.`,
			`Active Codex chats: restart after updating if Codex shims or agents changed.`,
		)
	case "binary build":
		return renderNextUp(
			`The shared binary is available on the next `+"`aether`"+` invocation.`,
			`Existing repos: run `+"`aether update --force`"+` after publish/install completes if companion files changed.`,
			`New repos: run `+"`aether lay-eggs`"+` to set up Aether.`,
		)
	default:
		return renderNextUp(
			`The shared binary is available on the next `+"`aether`"+` invocation.`,
			`Existing repos: run `+"`aether update --force`"+` if companion files also need refresh.`,
			`New repos: run `+"`aether lay-eggs`"+` to set up Aether.`,
		)
	}
}

func renderPauseVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("pause"), "Pause Colony"))
	b.WriteString(visualDividerStr())
	b.WriteString(renderStageMarker("Handoff"))
	b.WriteString("A save file for this project was written, so you can pick it back up later.\n")
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
	// The closeout comes from the one lifecycle resolver so it names the
	// canonical resume route once and may pair it only with a genuinely
	// different read-only quick view.
	b.WriteString(renderLifecycleClosing(result, "pause"))
	return b.String()
}

func renderResumeVisual(result map[string]interface{}, handoffText string, full bool) string {
	var b strings.Builder
	title := "Resume"
	if full {
		title = "Resume Colony"
	}
	b.WriteString(renderBanner(commandEmoji("resume"), title))
	b.WriteString(visualDividerStr())
	b.WriteString(renderStageMarker("Restored"))

	// Freshness warning for stale sessions
	if freshData, ok := result["freshness"].(map[string]interface{}); ok {
		fresh, _ := freshData["fresh"].(bool)
		if !fresh {
			ageHours := stringValue(freshData["age_hours"])
			b.WriteString(fmt.Sprintf("⚠️ Session is %s hours old — spawn state cleared\n", ageHours))
		}
	}

	// Worktree preservation summary — this is where "your work is still
	// here" has to appear on the exact screen a crash-recovery user sees.
	// Nothing here is destroyed automatically (D-01); this only reports
	// what was kept and what was forgotten because its path no longer
	// exists on disk.
	if errMsg := stringValue(result["worktree_gc_error"]); errMsg != "" {
		b.WriteString(fmt.Sprintf("⚠️ Could not check worker workspaces for leftover work: %s\n", errMsg))
	}
	if wtPreserved, ok := result["worktrees_preserved"].(map[string]interface{}); ok {
		cleaned := intValue(wtPreserved["cleaned"])
		preserved := intValue(wtPreserved["preserved"])
		if cleaned > 0 {
			b.WriteString(fmt.Sprintf("%d worker workspace(s) forgotten (their folder was already gone, nothing to keep)\n", cleaned))
		}
		if preserved > 0 {
			b.WriteString(fmt.Sprintf("Kept %d worker workspace(s) because they still hold work — nothing was deleted. Inspect them with `aether maintenance recovery-inspect` (State effect: none). To restore runnable lifecycle state, run `aether resume`.\n", preserved))
		}
	}

	// Stale FOCUS pheromone warning (Codex gets runtime-native warning only)
	if staleRaw, ok := result["stale_signals"]; ok {
		if staleList, ok := staleRaw.([]map[string]interface{}); ok && len(staleList) > 0 {
			b.WriteString(fmt.Sprintf("Warning: %d stale FOCUS signal(s) detected from previous phases\n", len(staleList)))
			for _, sig := range staleList {
				content, _ := sig["content"].(string)
				srcPhase, _ := sig["source_phase"].(int)
				b.WriteString(fmt.Sprintf("  - Phase %d: %s\n", srcPhase, content))
			}
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

	renderResumePhaseProgress(&b, result["phase_progress"])
	renderResumeDriftNote(&b, result["plan_revision"])
	if recent, ok := result["recent"].(map[string]interface{}); ok {
		renderResumeRecentDecisions(&b, recent["decisions"])
	}

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

	// The closing used to compute its own recommendation (suggestedNext,
	// falling back to computeNextAction) and a hand-typed "additional
	// inspection" alternative. Both are the one resolver's job now:
	// buildResumeDashboardResult already resolved and folded the answer for
	// this exact project into the result, including the two override facts
	// (a durable worker result waiting to finalize, or a build genuinely
	// still running) that only this dashboard knows.
	b.WriteString(renderLifecycleClosing(result, "resume-dashboard"))
	return b.String()
}

// resumePhaseProgressCap is the per-category nested-detail cap for the resume
// dashboard's phase-by-phase progress list (D-10): beneath this many named
// lines, an honest "(+N more)" line reports the real arithmetic remainder
// rather than silently truncating.
const resumePhaseProgressCap = 8

// resumePhaseStatusDisplay translates a phase's own recorded status value
// into the plain-English word the owner sees. A status this repo has not
// named falls back to the raw value rather than fabricated wording.
func resumePhaseStatusDisplay(status string) string {
	switch status {
	case colony.PhaseCompleted:
		return "finished"
	case colony.PhaseInProgress:
		return "in progress"
	case colony.PhasePending, "":
		return "not started"
	default:
		return status
	}
}

// renderResumePhaseProgress renders one line per phase naming it and its
// plain-English status (SHOW-02), beneath the resume dashboard's existing
// overall fraction. A colony with no plan renders nothing. Dual-type: raw is
// result["phase_progress"], either the in-process []resumePhaseProgressEntry
// or the JSON-round-tripped []interface{} a completion file would produce
// (renderContinueWorkerFlowValue precedent, 198-PATTERNS.md).
func renderResumePhaseProgress(b *strings.Builder, raw interface{}) {
	var entries []resumePhaseProgressEntry
	switch v := raw.(type) {
	case []resumePhaseProgressEntry:
		entries = v
	case []interface{}:
		for _, item := range v {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			entries = append(entries, resumePhaseProgressEntry{
				Phase:  intValue(entry["phase"]),
				Name:   stringValue(entry["name"]),
				Status: stringValue(entry["status"]),
			})
		}
	}
	if len(entries) == 0 {
		return
	}
	shown := entries
	overflow := 0
	if len(shown) > resumePhaseProgressCap {
		overflow = len(shown) - resumePhaseProgressCap
		shown = shown[:resumePhaseProgressCap]
	}
	b.WriteString("\nPhase Progress\n")
	for _, entry := range shown {
		line := fmt.Sprintf("  - Phase %d", entry.Phase)
		if name := strings.TrimSpace(entry.Name); name != "" {
			line += " — " + name
		}
		line += ": " + resumePhaseStatusDisplay(entry.Status)
		b.WriteString(line)
		b.WriteString("\n")
	}
	if overflow > 0 {
		b.WriteString(fmt.Sprintf("  (+%d more)\n", overflow))
	}
}

// renderResumeRecentDecisions renders up to five recent decisions (SHOW-02),
// most recent first, each naming what was decided with the reason on the
// nested "└──" detail line. extractRecentDecisions already caps at five and
// orders most-recent-first; this renderer neither re-slices nor re-orders.
// A colony with no recorded decisions renders nothing.
func renderResumeRecentDecisions(b *strings.Builder, raw interface{}) {
	items, ok := raw.([]interface{})
	if !ok || len(items) == 0 {
		return
	}
	b.WriteString("\nRecent Decisions\n")
	for _, item := range items {
		dec, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		claim := strings.TrimSpace(stringValue(dec["claim"]))
		if claim == "" {
			continue
		}
		b.WriteString("  - ")
		b.WriteString(claim)
		b.WriteString("\n")
		if rationale := strings.TrimSpace(stringValue(dec["rationale"])); rationale != "" {
			b.WriteString("      └── ")
			b.WriteString(rationale)
			b.WriteString("\n")
		}
	}
}

// intSliceLen reports the length of an int slice regardless of whether it
// arrives as the in-process []int or the JSON-round-tripped []interface{}.
func intSliceLen(value interface{}) int {
	switch v := value.(type) {
	case []int:
		return len(v)
	case []interface{}:
		return len(v)
	default:
		return 0
	}
}

// renderResumeDriftNote renders one plain-English sentence saying whether the
// plan has been revised since it was written (SHOW-02, Claude's Discretion).
// It reads only result["plan_revision"] -- planRevisionSummary's own output
// -- and computes no time-since or count-of-changes figure of its own
// (Phase 196 D-01 applied here: a signal the runtime cannot stand behind is
// never fabricated). With no recorded revision it says so plainly, and a
// colony with no plan at all never gets a fabricated one.
func renderResumeDriftNote(b *strings.Builder, raw interface{}) {
	revision, ok := raw.(map[string]interface{})
	b.WriteString("\nPlan Revision\n")
	if !ok || len(revision) == 0 {
		b.WriteString("  No plan revision has been recorded.\n")
		return
	}
	reasonType := stringValue(revision["reason_type"])
	if reasonType == string(colony.PlanRevisionLegacyImport) {
		b.WriteString("  The plan has not been revised since it was written.\n")
		return
	}
	line := "  The plan has been revised"
	if reason := strings.TrimSpace(stringValue(revision["reason"])); reason != "" {
		line += ": " + reason
	}
	if superseded := intSliceLen(revision["superseded_phase_ids"]); superseded > 0 {
		unit := "phase"
		if superseded != 1 {
			unit = "phases"
		}
		line += fmt.Sprintf(" (replaced %d %s)", superseded, unit)
	}
	line += ".\n"
	b.WriteString(line)
}

func renderPatrolVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("patrol"), "Patrol"))
	b.WriteString(visualDividerStr())
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
	b.WriteString(visualDividerStr())
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
	b.WriteString(visualDividerStr())

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
	// Classic activity-feed form: [time] icon [TYPE] source — every line
	// carries an action icon so the feed reads at a glance.
	b.WriteString("[")
	b.WriteString(label)
	b.WriteString("] ")
	b.WriteString(historyEventIcon(eventType, msg))
	if eventType != "" {
		b.WriteString(" [")
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

// historyEventIcon maps an event to the classic v5.4.0 activity-feed icon set
// (colorize-log.sh): ⚡ spawn, ✅ complete, ❌ error, ✨ created, 📝 modified,
// 🔬 research, ⚙️ executing.
func historyEventIcon(eventType, message string) string {
	probe := strings.ToUpper(eventType + " " + message)
	switch {
	case strings.Contains(probe, "SPAWN"):
		return "⚡"
	case strings.Contains(probe, "COMPLETE"), strings.Contains(probe, "SEALED"), strings.Contains(probe, "ADVANCE"):
		return "✅"
	case strings.Contains(probe, "ERROR"), strings.Contains(probe, "FAIL"), strings.Contains(probe, "BLOCK"):
		return "❌"
	case strings.Contains(probe, "CREATED"), strings.Contains(probe, "INIT"):
		return "✨"
	case strings.Contains(probe, "MODIFIED"), strings.Contains(probe, "REPAIR"), strings.Contains(probe, "UPDATE"):
		return "📝"
	case strings.Contains(probe, "RESEARCH"), strings.Contains(probe, "EXPLOR"), strings.Contains(probe, "SURVEY"):
		return "🔬"
	case strings.Contains(probe, "EXECUT"), strings.Contains(probe, "BUILD"):
		return "⚙️"
	case strings.Contains(probe, "PHASE"):
		return "🐜"
	default:
		return "•"
	}
}

func renderReferenceIndexVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("reference-index"), "Reference Index"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Library: %s\n", emptyFallback(stringValue(result["root"]), "not found")))
	b.WriteString(fmt.Sprintf("References indexed: %d\n", intValue(result["total"])))
	if categories := categoryCountsValue(result["categories"]); len(categories) > 0 {
		b.WriteString("\nCategories\n")
		for _, name := range sortedMapKeys(categories) {
			b.WriteString(fmt.Sprintf("  - %s: %d\n", name, categories[name]))
		}
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether reference-list`"+` to browse installed references.`,
		`Run `+"`aether reference-match --task \"...\"`"+` to match references to work.`,
	))
	return b.String()
}

func renderReferenceListVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("reference-list"), "Reference Library"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Library: %s\n", emptyFallback(stringValue(result["root"]), "not found")))
	refs := mapSliceValue(result["references"])
	if len(refs) == 0 {
		b.WriteString("No references matched the current filters.\n")
	} else {
		b.WriteString(fmt.Sprintf("References: %d\n\n", len(refs)))
		writeReferenceSummaryLines(&b, refs)
	}
	b.WriteString(renderNextUp(
		`Run ` + "`aether reference-match --task \"...\"`" + ` to select references for a worker task.`,
	))
	return b.String()
}

func renderReferenceMatchVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("reference-match"), "Reference Match"))
	b.WriteString(visualDividerStr())
	if task := strings.TrimSpace(stringValue(result["task"])); task != "" {
		b.WriteString("Task: ")
		b.WriteString(task)
		b.WriteString("\n")
	}
	if role := strings.TrimSpace(stringValue(result["role"])); role != "" {
		b.WriteString("Role: ")
		b.WriteString(role)
		b.WriteString("\n")
	}
	refs := mapSliceValue(result["references"])
	if len(refs) == 0 {
		b.WriteString("No matching references found.\n")
	} else {
		b.WriteString(fmt.Sprintf("\nMatches: %d\n", len(refs)))
		writeReferenceSummaryLines(&b, refs)
	}
	b.WriteString(renderNextUp(
		`Use the matched reference paths as worker context.`,
		`Run `+"`aether reference-list`"+` to browse the full library.`,
	))
	return b.String()
}

func writeReferenceSummaryLines(b *strings.Builder, refs []map[string]interface{}) {
	limit := len(refs)
	if limit > 8 {
		limit = 8
	}
	for i := 0; i < limit; i++ {
		ref := refs[i]
		title := emptyFallback(stringValue(ref["title"]), stringValue(ref["id"]))
		path := strings.TrimSpace(stringValue(ref["path"]))
		category := strings.TrimSpace(stringValue(ref["category"]))
		score := intValue(ref["score"])
		b.WriteString("  - ")
		b.WriteString(title)
		if category != "" {
			b.WriteString(" [")
			b.WriteString(category)
			b.WriteString("]")
		}
		if score > 0 {
			b.WriteString(fmt.Sprintf(" score=%d", score))
		}
		if path != "" {
			b.WriteString("\n    ")
			b.WriteString(path)
		}
		b.WriteString("\n")
	}
	if len(refs) > limit {
		b.WriteString(fmt.Sprintf("  ... %d more\n", len(refs)-limit))
	}
}

func renderFlagsVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("flags"), "Flags"))
	b.WriteString(visualDividerStr())
	// The classic 🚩 renderer (written in the restoration round, finally
	// wired): one line per flag with nested detail, triage counts, and the
	// Iron Law reminder when blockers are open.
	b.WriteString(renderFlagsTable(flagEntriesValue(result["flags"])))
	b.WriteString(renderNextUp(
		`Run `+"`aether flag \"...\"`"+` to create a new flag.`,
		`Run `+"`aether flag-resolve --id <id>`"+` after a blocker or issue is handled.`,
	))
	return b.String()
}

func renderFlagActionVisual(command, title string, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji(command), title))
	b.WriteString(visualDividerStr())
	if flag, ok := result["flag"].(colony.FlagEntry); ok {
		b.WriteString(fmt.Sprintf("ID: %s\n", flag.ID))
		b.WriteString(fmt.Sprintf("Type: %s\n", flag.Type))
		b.WriteString(fmt.Sprintf("Description: %s\n", emptyFallback(flag.Description, "(none)")))
		if flag.Phase != nil && *flag.Phase > 0 {
			b.WriteString(fmt.Sprintf("Phase: %d\n", *flag.Phase))
		}
	} else if id := strings.TrimSpace(stringValue(result["id"])); id != "" {
		b.WriteString(fmt.Sprintf("ID: %s\n", id))
	}
	for _, key := range []string{"created", "resolved", "acknowledged", "max_days", "total"} {
		if _, ok := result[key]; ok {
			b.WriteString(fmt.Sprintf("%s: %s\n", humanizeKey(key), stringValue(result[key])))
		}
	}
	if message := strings.TrimSpace(stringValue(result["message"])); message != "" {
		b.WriteString("Message: ")
		b.WriteString(message)
		b.WriteString("\n")
	}
	b.WriteString(renderNextUp(`Run ` + "`aether flags`" + ` to inspect active flags.`))
	return b.String()
}

func renderShelfListVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("shelf-list"), "Shelf"))
	b.WriteString(visualDividerStr())
	if status := strings.TrimSpace(stringValue(result["status"])); status != "" {
		b.WriteString("Filter: ")
		b.WriteString(status)
		b.WriteString("\n")
	}
	entries := shelfEntriesValue(result["entries"])
	if len(entries) == 0 {
		b.WriteString("No shelf entries found.\n")
	} else {
		b.WriteString(fmt.Sprintf("Entries: %d\n\n", len(entries)))
		for _, entry := range entries {
			mark := "[ ]"
			switch entry.Status {
			case colony.ShelfPromoted:
				mark = "[+]"
			case colony.ShelfDismissed:
				mark = "[-]"
			}
			b.WriteString(fmt.Sprintf("  %s %s", mark, emptyFallback(entry.Text, "(empty entry)")))
			if entry.Category != "" {
				b.WriteString(fmt.Sprintf("  %s", entry.Category))
			}
			if entry.ID != "" {
				b.WriteString(fmt.Sprintf("  {%s}", entry.ID))
			}
			if entry.PromotedTo != "" {
				b.WriteString(fmt.Sprintf("\n    promoted to: %s", entry.PromotedTo))
			}
			b.WriteString("\n")
		}
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether shelf-add --text \"...\"`"+` to capture a deferred signal.`,
		`Run `+"`aether shelf-promote --id <id> --to <target>`"+` when an entry should become active work.`,
	))
	return b.String()
}

func renderShelfActionVisual(command, title string, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji(command), title))
	b.WriteString(visualDividerStr())
	if entry, ok := result["entry"].(colony.ShelfEntry); ok {
		b.WriteString(fmt.Sprintf("Entry: %s\n", emptyFallback(entry.Text, "(empty entry)")))
		b.WriteString(fmt.Sprintf("Category: %s\n", entry.Category))
		if entry.ID != "" {
			b.WriteString(fmt.Sprintf("ID: %s\n", entry.ID))
		}
	} else if id := strings.TrimSpace(stringValue(result["id"])); id != "" {
		b.WriteString(fmt.Sprintf("ID: %s\n", id))
	}
	if to := strings.TrimSpace(stringValue(result["to"])); to != "" {
		b.WriteString(fmt.Sprintf("Target: %s\n", to))
	}
	if total := intValue(result["total"]); total > 0 {
		b.WriteString(fmt.Sprintf("Shelf total: %d\n", total))
	}
	b.WriteString(renderNextUp(`Run ` + "`aether shelf-list`" + ` to inspect the shelf.`))
	return b.String()
}

func renderQueenActionVisual(command, title string, result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji(command), title))
	b.WriteString(visualDividerStr())
	if path := strings.TrimSpace(stringValue(result["path"])); path != "" {
		b.WriteString("Path: ")
		b.WriteString(path)
		b.WriteString("\n")
	}
	if target := strings.TrimSpace(stringValue(result["target"])); target != "" {
		b.WriteString("Target: ")
		b.WriteString(target)
		b.WriteString("\n")
	}
	if reason := strings.TrimSpace(stringValue(result["reason"])); reason != "" {
		b.WriteString("Reason: ")
		b.WriteString(reason)
		b.WriteString("\n")
	}
	for _, key := range []string{"created", "local_created", "written", "promoted", "migrated", "seeded", "skipped", "total", "size"} {
		if _, ok := result[key]; ok {
			b.WriteString(fmt.Sprintf("%s: %s\n", humanizeKey(key), stringValue(result[key])))
		}
	}
	if sections := stringSliceValue(result["sections"]); len(sections) > 0 {
		b.WriteString("Sections: ")
		b.WriteString(strings.Join(sections, ", "))
		b.WriteString("\n")
	}
	if boolValue(result["needs_input"]) {
		b.WriteString("\nInput needed before writing project memory.\n")
	}
	if _, ok := result["trust_promote_threshold"]; ok {
		b.WriteString("\nThresholds\n")
		for _, key := range []string{"trust_promote_threshold", "trust_hive_threshold", "trust_decay_half_life", "trust_floor", "max_instincts", "max_wisdom_entries"} {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", humanizeKey(key), stringValue(result[key])))
		}
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether queen-read`"+` to inspect Queen memory.`,
		`Run `+"`aether status`"+` to return to colony state.`,
	))
	return b.String()
}

func renderExportSignalsVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("export-signals"), "Signals Exported"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Signals: %d\n", intValue(result["count"])))
	if file := strings.TrimSpace(stringValue(result["file"])); file != "" {
		b.WriteString("File: ")
		b.WriteString(file)
		b.WriteString("\n")
	} else if result["xml"] == nil {
		b.WriteString("No pheromone file found; exported an empty signal set.\n")
	}
	b.WriteString(renderNextUp(
		`Share the XML with another colony, then run ` + "`aether import-signals --file <path>`" + ` there.`,
	))
	return b.String()
}

func renderImportSignalsVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("import-signals"), "Signals Imported"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Imported: %d\n", intValue(result["imported"])))
	b.WriteString(fmt.Sprintf("Total active file entries: %d\n", intValue(result["total"])))
	if source := strings.TrimSpace(stringValue(result["source"])); source != "" {
		b.WriteString("Source: ")
		b.WriteString(source)
		b.WriteString("\n")
	}
	if warnings := stringSliceValue(result["warnings"]); len(warnings) > 0 {
		b.WriteString("\nWarnings\n")
		b.WriteString(renderIndentedList(warnings))
	}
	b.WriteString(renderNextUp(
		`Run ` + "`aether pheromones`" + ` to inspect active steering signals.`,
	))
	return b.String()
}

func renderTunnelsVisual(result map[string]interface{}) string {
	switch strings.ToLower(strings.TrimSpace(stringValue(result["mode"]))) {
	case "list":
		return renderTunnelsListVisual(result)
	case "compare":
		return renderTunnelsCompareVisual(result)
	case "import":
		return renderTunnelsImportVisual(result)
	default:
		return renderTunnelsDetailVisual(result)
	}
}

func renderTunnelsListVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("tunnels"), "Colony Timeline"))
	b.WriteString(visualDividerStr())
	total := intValue(result["total"])
	b.WriteString(fmt.Sprintf("%d colonies archived\n", total))
	chambers := mapSliceValue(result["chambers"])
	if len(chambers) == 0 {
		b.WriteString("\nThe tunnel network is empty.\n")
		b.WriteString(renderNextUp(`Run ` + "`aether entomb`" + ` after sealing a colony to preserve it.`))
		return b.String()
	}
	b.WriteString("\n")
	for _, chamber := range chambers {
		name := emptyFallback(stringValue(chamber["name"]), "(unnamed chamber)")
		date := strings.TrimSpace(stringValue(chamber["entombed_at"]))
		if len(date) >= 10 {
			date = date[:10]
		}
		if date == "" {
			date = "unknown-date"
		}
		milestone := emptyFallback(stringValue(chamber["milestone"]), "unknown milestone")
		goal := truncateString(emptyFallback(stringValue(chamber["goal"]), "No goal recorded"), 72)
		fmt.Fprintf(&b, "[%s] %s %s\n", date, milestoneIcon(milestone), name)
		fmt.Fprintf(&b, "           %s\n", goal)
		fmt.Fprintf(&b, "           %d/%d phases | %s\n", intValue(chamber["phases_completed"]), intValue(chamber["total_phases"]), milestone)
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether tunnels <chamber>`"+` to view a sealed chamber.`,
		`Run `+"`aether tunnels <chamber_a> <chamber_b>`"+` to compare two colonies.`,
	))
	return b.String()
}

func renderTunnelsDetailVisual(result map[string]interface{}) string {
	var b strings.Builder
	chamber := emptyFallback(stringValue(result["chamber"]), "unknown")
	b.WriteString(renderBanner(commandEmoji("tunnels"), "Chamber Details"))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Chamber: %s\n", chamber))
	if manifest := mapValue(result["manifest"]); len(manifest) > 0 {
		if goal := strings.TrimSpace(stringValue(manifest["goal"])); goal != "" {
			b.WriteString("Goal: ")
			b.WriteString(goal)
			b.WriteString("\n")
		}
		if milestone := strings.TrimSpace(stringValue(manifest["milestone"])); milestone != "" {
			b.WriteString("Milestone: ")
			b.WriteString(milestone)
			b.WriteString("\n")
		}
	}
	files := stringSliceValue(result["files"])
	if len(files) > 0 {
		b.WriteString(fmt.Sprintf("\nFiles: %d\n", len(files)))
		b.WriteString(renderIndentedList(files))
	}
	if summary := strings.TrimSpace(stringValue(result["seal_summary"])); summary != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Seal Summary"))
		b.WriteString(summary)
		if !strings.HasSuffix(summary, "\n") {
			b.WriteString("\n")
		}
	}
	next := []string{`Run ` + "`aether tunnels`" + ` to list sealed chambers.`}
	if importCommand := strings.TrimSpace(stringValue(result["import_command"])); importCommand != "" {
		next = append([]string{`Run ` + "`" + importCommand + "`" + ` to import this chamber's pheromone signals.`}, next...)
	}
	next = append(next, `Run `+"`aether tunnels "+chamber+" <other_chamber>`"+` to compare chambers.`)
	b.WriteString(renderNextUp(next[0], next[1:]...))
	return b.String()
}

func renderTunnelsCompareVisual(result map[string]interface{}) string {
	var b strings.Builder
	leftName := emptyFallback(stringValue(result["chamber_a"]), "left")
	rightName := emptyFallback(stringValue(result["chamber_b"]), "right")
	left := mapValue(result["manifest_a"])
	right := mapValue(result["manifest_b"])
	leftSummary := mapValue(result["summary_a"])
	rightSummary := mapValue(result["summary_b"])
	growth := mapValue(result["growth"])

	b.WriteString(renderBanner(commandEmoji("tunnels"), "Chamber Comparison"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "%s  vs  %s\n\n", leftName, rightName)
	b.WriteString(fmt.Sprintf("%-18s | %s\n", "Goal", ""))
	b.WriteString(fmt.Sprintf("  %-16s | %s\n", leftName, truncateString(stringValue(left["goal"]), 64)))
	b.WriteString(fmt.Sprintf("  %-16s | %s\n", rightName, truncateString(stringValue(right["goal"]), 64)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "Milestone: %s -> %s\n", emptyFallback(stringValue(left["milestone"]), "unknown"), emptyFallback(stringValue(right["milestone"]), "unknown"))
	fmt.Fprintf(&b, "Phases: %d/%d -> %d/%d\n",
		intValue(left["phases_completed"]), intValue(left["total_phases"]),
		intValue(right["phases_completed"]), intValue(right["total_phases"]),
	)
	fmt.Fprintf(&b, "Decisions: %d -> %d\n", intValue(leftSummary["decisions"]), intValue(rightSummary["decisions"]))
	fmt.Fprintf(&b, "Learnings: %d -> %d\n", intValue(leftSummary["learnings"]), intValue(rightSummary["learnings"]))
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Growth"))
	fmt.Fprintf(&b, "Phases: %+d\n", intValue(growth["phases_diff"]))
	fmt.Fprintf(&b, "Decisions: %+d\n", intValue(growth["decisions_diff"]))
	fmt.Fprintf(&b, "Learnings: %+d\n", intValue(growth["learnings_diff"]))
	if days := intValue(growth["days_between"]); days > 0 {
		fmt.Fprintf(&b, "Time: %d days apart\n", days)
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether tunnels`"+` to see all chambers.`,
		`Run `+"`aether tunnels <chamber>`"+` to view a single seal document.`,
	))
	return b.String()
}

func renderTunnelsImportVisual(result map[string]interface{}) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("import-signals"), "Signals Imported"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Chamber: %s\n", emptyFallback(stringValue(result["chamber"]), "unknown"))
	fmt.Fprintf(&b, "Imported: %d pheromone signals\n", intValue(result["imported"]))
	if prefix := strings.TrimSpace(stringValue(result["id_prefix"])); prefix != "" {
		fmt.Fprintf(&b, "Tagged with: %s\n", prefix)
	}
	if archive := strings.TrimSpace(stringValue(result["archive"])); archive != "" {
		fmt.Fprintf(&b, "Archive: %s\n", archive)
	}
	if warnings := stringSliceValue(result["warnings"]); len(warnings) > 0 {
		b.WriteString("\nWarnings\n")
		b.WriteString(renderIndentedList(warnings))
	}
	b.WriteString(renderNextUp(
		`Run `+"`aether pheromones`"+` to inspect active steering signals.`,
		`Run `+"`aether tunnels`"+` to return to the chamber timeline.`,
	))
	return b.String()
}

func milestoneIcon(milestone string) string {
	switch strings.ToLower(strings.TrimSpace(milestone)) {
	case "crowned anthill":
		return "👑"
	case "sealed chambers":
		return "🔒"
	default:
		return "•"
	}
}

func renderCloseoutVisual(result map[string]interface{}) string {
	var b strings.Builder
	workflow := emptyFallback(stringValue(result["workflow"]), "workflow")
	b.WriteString(renderBanner(commandEmoji("closeout"), fmt.Sprintf("%s Closeout", strings.Title(workflow))))
	b.WriteString(visualDividerStr())
	if !boolValue(result["state_available"]) {
		b.WriteString(emptyFallback(stringValue(result["message"]), "No colony state available."))
		b.WriteString("\n")
		b.WriteString(renderLifecycleClosing(result, workflow))
		return b.String()
	}
	if goal := strings.TrimSpace(stringValue(result["goal"])); goal != "" {
		b.WriteString("Goal: ")
		b.WriteString(goal)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("State: %s\n", emptyFallback(stringValue(result["state"]), "unknown")))
	total := intValue(result["total_phases"])
	current := intValue(result["current_phase"])
	if total > 0 {
		b.WriteString(fmt.Sprintf("Progress: %d/%d phases complete\n", intValue(result["completed_phases"]), total))
	}
	if current > 0 {
		b.WriteString(fmt.Sprintf("Current phase: %d", current))
		if phaseName := strings.TrimSpace(stringValue(result["phase_name"])); phaseName != "" && phaseName != "(unnamed)" {
			b.WriteString(" - ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}
	if completionFile := strings.TrimSpace(stringValue(result["completion_file"])); completionFile != "" {
		b.WriteString("Completion packet: ")
		b.WriteString(completionFile)
		b.WriteString("\n")
	}
	renderCloseoutCompletionSection(&b, result)
	if readiness := strings.TrimSpace(stringValue(result["porter_readiness"])); readiness != "" {
		b.WriteString("\n")
		b.WriteString(renderStageMarker("Post-Seal: Delivery Readiness"))
		b.WriteString(readiness)
		if !strings.HasSuffix(readiness, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("\nRun `/ant-porter` or `aether porter check` to validate and deliver.\n")
	}
	b.WriteString(renderLifecycleClosing(result, stringValue(result["workflow"])))
	return b.String()
}

func renderCloseoutCompletionSection(b *strings.Builder, result map[string]interface{}) {
	if !boolValue(result["completion_loaded"]) && strings.TrimSpace(stringValue(result["completion_error"])) == "" {
		return
	}
	b.WriteString("\n")
	b.WriteString(renderStageMarker("Worker Results"))
	if errText := strings.TrimSpace(stringValue(result["completion_error"])); errText != "" {
		b.WriteString(errText)
		b.WriteString("\n")
		return
	}
	workerCount := intValue(result["completion_worker_count"])
	dispatchCount := intValue(result["completion_dispatch_count"])
	if dispatchCount > 0 {
		fmt.Fprintf(b, "Dispatches: %d\n", dispatchCount)
	}
	fmt.Fprintf(b, "Workers: %d completed, %d blocked, %d failed",
		intValue(result["completion_completed"]),
		intValue(result["completion_blocked"]),
		intValue(result["completion_failed"]),
	)
	if workerCount > 0 {
		fmt.Fprintf(b, " (%d total)", workerCount)
	}
	b.WriteString("\n")
	if phase := intValue(result["completion_phase"]); phase > 0 {
		fmt.Fprintf(b, "Phase: %d", phase)
		if phaseName := strings.TrimSpace(stringValue(result["completion_phase_name"])); phaseName != "" {
			b.WriteString(" - ")
			b.WriteString(phaseName)
		}
		b.WriteString("\n")
	}
	for _, worker := range mapSliceValue(result["completion_workers"]) {
		name := emptyFallback(stringValue(worker["name"]), stringValue(worker["agent_name"]))
		if name == "" {
			name = emptyFallback(stringValue(worker["ant_name"]), "worker")
		}
		status := emptyFallback(stringValue(worker["status"]), "unknown")
		caste := stringValue(worker["caste"])
		fmt.Fprintf(b, "  %s %s  %s", dispatchStatusIcon(normalizeRuntimeDispatchStatus(status)), casteIdentity(caste), name)
		if summary := strings.TrimSpace(stringValue(worker["summary"])); summary != "" {
			b.WriteString(" - ")
			b.WriteString(summary)
		}
		b.WriteString("\n")
	}
	if blockers := stringSliceValue(result["completion_blockers"]); len(blockers) > 0 {
		b.WriteString("\nBlockers\n")
		b.WriteString(renderIndentedList(blockers))
	}
	if artifacts := stringSliceValue(result["completion_artifacts"]); len(artifacts) > 0 {
		b.WriteString("\nArtifacts\n")
		limit := artifacts
		if len(limit) > 8 {
			limit = limit[:8]
		}
		b.WriteString(renderIndentedList(limit))
		if len(artifacts) > len(limit) {
			fmt.Fprintf(b, "  - ...and %d more\n", len(artifacts)-len(limit))
		}
	}
}

func categoryCountsValue(raw interface{}) map[string]int {
	out := map[string]int{}
	switch v := raw.(type) {
	case map[string]int:
		for key, value := range v {
			out[key] = value
		}
	case map[string]interface{}:
		for key, value := range v {
			out[key] = intValue(value)
		}
	}
	return out
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func mapSliceValue(raw interface{}) []map[string]interface{} {
	switch v := raw.(type) {
	case []map[string]interface{}:
		return v
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if mapped, ok := item.(map[string]interface{}); ok {
				out = append(out, mapped)
			}
		}
		return out
	default:
		return nil
	}
}

func shelfEntriesValue(raw interface{}) []colony.ShelfEntry {
	switch v := raw.(type) {
	case []colony.ShelfEntry:
		return v
	case []interface{}:
		out := make([]colony.ShelfEntry, 0, len(v))
		for _, item := range v {
			entryMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, colony.ShelfEntry{
				ID:         stringValue(entryMap["id"]),
				Text:       stringValue(entryMap["text"]),
				Category:   colony.ShelfCategory(stringValue(entryMap["category"])),
				Status:     colony.ShelfStatus(stringValue(entryMap["status"])),
				PromotedTo: stringValue(entryMap["promoted_to"]),
			})
		}
		return out
	default:
		return nil
	}
}

func flagEntriesValue(raw interface{}) []colony.FlagEntry {
	switch v := raw.(type) {
	case []colony.FlagEntry:
		return v
	case []interface{}:
		out := make([]colony.FlagEntry, 0, len(v))
		for _, item := range v {
			entryMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			var phase *int
			if phaseValue := intValue(entryMap["phase"]); phaseValue > 0 {
				phase = &phaseValue
			}
			out = append(out, colony.FlagEntry{
				ID:          stringValue(entryMap["id"]),
				Type:        stringValue(entryMap["type"]),
				Description: stringValue(entryMap["description"]),
				Source:      stringValue(entryMap["source"]),
				Phase:       phase,
				Resolved:    boolValue(entryMap["resolved"]),
			})
		}
		return out
	default:
		return nil
	}
}

func humanizeKey(key string) string {
	key = strings.ReplaceAll(key, "_", " ")
	if key == "" {
		return ""
	}
	return strings.ToUpper(key[:1]) + key[1:]
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
	dispatches, err := plannedBuildDispatches(phase, depth)
	if err != nil {
		return renderUnplannablePhase(phase, err)
	}
	return renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)
}

// renderQueenTeamChoice makes the Queen's team decision readable. The
// rationale strings are composed on every build and carried in the dispatch
// contract (SelectedReasons/PrunedReasons, cmd/codex_dispatch_contract.go) —
// and until this renderer existed they were never shown to a human, so "why
// didn't it use the security one?" required opening a JSON manifest.
//
// Everything printed here is read from the contract, never recomputed and
// never hardcoded: the render test blanks the contract fields and asserts the
// clauses disappear with them.
func renderQueenTeamChoice(policy codexQueenExecutionPolicy, dispatches []codexBuildDispatch) string {
	budget := policy.SpawnBudget
	if budget == nil || (len(budget.SelectedReasons) == 0 && len(budget.PrunedReasons) == 0 && len(budget.PreservedCastes) == 0) {
		return ""
	}

	var b strings.Builder
	b.WriteString("Queen's Team\n")

	// One clause per selected caste, in dispatch order so the list reads the
	// way the workers will actually spawn.
	seen := map[string]bool{}
	orderedCastes := make([]string, 0, len(budget.SelectedReasons))
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" || seen[caste] {
			continue
		}
		seen[caste] = true
		orderedCastes = append(orderedCastes, caste)
	}
	// Castes with a recorded reason but no dispatch row still get their clause.
	for _, caste := range sortedStringKeys(budget.SelectedReasons) {
		if !seen[caste] {
			seen[caste] = true
			orderedCastes = append(orderedCastes, caste)
		}
	}
	for _, caste := range orderedCastes {
		reason := strings.TrimSpace(budget.SelectedReasons[caste])
		if reason == "" {
			continue
		}
		b.WriteString("  ")
		b.WriteString(casteIdentity(caste))
		b.WriteString(" — ")
		b.WriteString(reason)
		b.WriteString("\n")
	}

	// Safety restorations and policy additions: when the runtime keeps a caste
	// the depth flag or budget would have dropped, it says which caste and why
	// instead of silently correcting.
	//
	// The preserved-castes clause renders ONLY when the budget actually cut
	// something. PreservedCastes is the intersection of selected and required,
	// which on an ordinary build always contains the Watcher — rendering it
	// unconditionally would announce a "safety intervention" on every build,
	// which is both noise and untrue. When nothing was pruned, nothing was
	// protected from anything.
	budgetCut := len(budget.PrunedReasons) > 0 || intDeref(budget.PrunedCastes) > 0 || intDeref(budget.PrunedWorkers) > 0
	if budgetCut {
		for _, caste := range budget.PreservedCastes {
			reason := strings.TrimSpace(budget.SelectedReasons[caste])
			if reason == "" {
				reason = "required by safety policy for this phase"
			}
			b.WriteString("  Kept by safety policy: ")
			b.WriteString(casteLabel(caste))
			b.WriteString(" — ")
			b.WriteString(reason)
			b.WriteString("\n")
		}
	}
	for _, caste := range budget.PolicyAddedCastes {
		b.WriteString("  Added by build policy: ")
		b.WriteString(casteLabel(caste))
		b.WriteString("\n")
	}

	// The castes considered and not called, as ONE short clause. 27 castes
	// exist; a per-build absentee table is exactly the ceremony the reshape
	// removed, so at most three are named and the rest are a count.
	if len(budget.PrunedReasons) > 0 {
		pruned := sortedStringKeys(budget.PrunedReasons)
		b.WriteString(fmt.Sprintf("  Not called (%d): ", len(pruned)))
		shown := pruned
		if len(shown) > 3 {
			shown = shown[:3]
		}
		clauses := make([]string, 0, len(shown))
		for _, caste := range shown {
			reason := strings.TrimSpace(budget.PrunedReasons[caste])
			if reason == "" {
				clauses = append(clauses, casteLabel(caste))
				continue
			}
			clauses = append(clauses, casteLabel(caste)+" — "+reason)
		}
		b.WriteString(strings.Join(clauses, "; "))
		if remaining := len(pruned) - len(shown); remaining > 0 {
			b.WriteString(fmt.Sprintf("; and %d more below the relevance threshold", remaining))
		}
		b.WriteString("\n")
	}
	if trimmed := intDeref(budget.PrunedWorkers); trimmed > 0 && len(budget.PrunedReasons) == 0 {
		b.WriteString(fmt.Sprintf("  Trimmed to the worker cap: %d caste(s)\n", trimmed))
	}

	return b.String()
}

// queenPolicyFromResult recovers the typed Queen policy from a build result
// map. The build paths store the struct value directly, so no JSON round-trip
// is involved.
func queenPolicyFromResult(result map[string]interface{}) codexQueenExecutionPolicy {
	if policy, ok := result["queen_execution_policy"].(codexQueenExecutionPolicy); ok {
		return policy
	}
	return codexQueenExecutionPolicy{}
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func intDeref(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func renderSpawnPlanForDispatches(dispatches []codexBuildDispatch, parallelMode colony.ParallelMode) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("spawn-plan"), "Spawn Plan"))
	b.WriteString(visualDividerStr())

	executionPlans := buildExecutionPlans(dispatches, parallelMode)
	dispatchesByExecutionWave := map[int][]codexBuildDispatch{}
	for _, dispatch := range dispatches {
		wave := normalizedDispatchWave(dispatch)
		dispatchesByExecutionWave[wave] = append(dispatchesByExecutionWave[wave], dispatch)
	}
	for idx, plan := range executionPlans {
		if idx > 0 {
			b.WriteString("\n")
		}
		b.WriteString(buildExecutionPlanLabel(plan))
		b.WriteString("\n")
		b.WriteString("  Execution: ")
		b.WriteString(plan.Strategy)
		if strings.TrimSpace(plan.Reason) != "" {
			b.WriteString(" — ")
			b.WriteString(plan.Reason)
		}
		if plan.Stage == "wave" && plan.Strategy == "serial" && plan.WorkerCount > 1 && parallelMode == colony.ModeInRepo {
			b.WriteString(" (set `aether parallel-mode set worktree` for isolated parallel builders)")
		}
		b.WriteString("\n")
		for _, dispatch := range dispatchesByExecutionWave[plan.ExecutionWave] {
			b.WriteString("  ")
			b.WriteString(casteIdentity(dispatch.Caste))
			b.WriteString(" ")
			b.WriteString(dispatch.Name)
			b.WriteString("  ")
			b.WriteString(dispatchTaskLine(dispatch.Task))
			b.WriteString(dispatchCoveredTasksNote(dispatch))
			writeDispatchExecutionStatus(&b, dispatch)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("Total planned dispatches: %d\n", len(dispatches)))
	b.WriteString(renderCoveredTaskSummary(dispatches))
	b.WriteString(renderSpawnTeamExplanation(dispatches))
	return b.String()
}

// renderSpawnTeamExplanation says, in one line a non-specialist can read, who
// the Queen picked and how to ask for a different size.
//
// The plan above lists castes and task strings, which answers "what will run"
// but never "why these, and how do I get fewer" — the question an operator
// actually has when a small change appears to summon a committee. Without an
// answer, the only discoverable lever is reading the source.
func renderSpawnTeamExplanation(dispatches []codexBuildDispatch) string {
	if len(dispatches) == 0 {
		return ""
	}

	seen := map[string]bool{}
	names := make([]string, 0, len(dispatches))
	for _, dispatch := range dispatches {
		caste := strings.TrimSpace(dispatch.Caste)
		if caste == "" || seen[caste] {
			continue
		}
		seen[caste] = true
		names = append(names, casteLabel(caste))
	}
	if len(names) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nThe Queen (the coordinator that decides which helpers a phase needs) chose this team for the phase: ")
	b.WriteString(strings.Join(names, ", "))
	b.WriteString(".\n")
	b.WriteString("Want a smaller team? Add `--light`. Want every check? Add `--heavy`.\n")
	b.WriteString("Safety-critical helpers (the castes a risky phase needs) are kept at every size.\n")
	return b.String()
}

func buildExecutionPlanLabel(plan codexBuildExecutionPlan) string {
	switch plan.Stage {
	case "prep":
		return "Pre-Wave: Archaeology"
	case "research":
		return "Pre-Wave: Oracle Research"
	case "design":
		return "Pre-Wave: Architect Design"
	case "integration":
		return "Pre-Wave: Ambassador Integration"
	case "wave":
		if plan.Wave > 0 {
			return fmt.Sprintf("Wave %d", plan.Wave)
		}
		return "Worker Wave"
	case "probe":
		return "Post-Wave: Probe"
	case "verification":
		return "Post-Wave: Watcher"
	case "measurement":
		return "Post-Wave: Measurer"
	case "resilience":
		return "Post-Wave: Chaos"
	default:
		stage := strings.TrimSpace(plan.Stage)
		if stage == "" {
			return fmt.Sprintf("Execution Step %d", plan.ExecutionWave)
		}
		return fmt.Sprintf("Execution Step %d: %s", plan.ExecutionWave, stage)
	}
}

func writeDispatchExecutionStatus(b *strings.Builder, dispatch codexBuildDispatch) {
	status := strings.TrimSpace(dispatch.Status)
	if status == "" || status == "spawned" || status == "planned" {
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
	// A worker that surfaced something and a worker that came back with
	// nothing to report are different outcomes, not two identical "completed"
	// lines. Blockers are the actionable-finding channel in the result schema,
	// so completion is qualified by it.
	if status == "completed" {
		if len(dispatch.Blockers) > 0 {
			b.WriteString(fmt.Sprintf(" — flagged %d issue(s)", len(dispatch.Blockers)))
		} else {
			b.WriteString(" — nothing to flag")
		}
	}
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
	words := buildCasteKeywordWords(text)
	// Check builder keywords first (higher priority)
	for _, token := range []string{"implement", "build", "create", "fix", "add", "write", "code", "refactor", "test", "deploy"} {
		if buildCasteWordMatches(words, token) {
			return "builder"
		}
	}
	// Then check scout keywords
	for _, token := range []string{"research", "investigat", "survey", "analy", "document", "readme", "spec"} {
		if buildCasteWordMatches(words, token) {
			return "scout"
		}
	}
	return "builder"
}

func buildCasteKeywordWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func buildCasteWordMatches(words []string, token string) bool {
	for _, word := range words {
		if word == token || word == token+"s" || word == token+"ed" || word == token+"ing" {
			return true
		}
		if token == "investigat" && strings.HasPrefix(word, token) {
			return true
		}
		if token == "analy" && strings.HasPrefix(word, token) {
			return true
		}
		if token == "spec" && (word == "specification" || word == "specifications") {
			return true
		}
	}
	return false
}

func taskWaves(tasks []colony.Task) [][]int {
	if len(tasks) == 0 {
		return nil
	}
	references, err := colony.NewTaskReferenceIndex([]colony.Phase{{Tasks: tasks}})
	if err != nil {
		return nil
	} // Build admission reports ambiguity before scheduling.

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
	dependencies := make([][]string, len(tasks))
	for i, task := range tasks {
		for _, dependency := range task.DependsOn {
			if target, ok := references.Resolve(dependency); ok {
				dependency = taskIDs[target.TaskIndex]
			}
			dependencies[i] = append(dependencies[i], dependency)
		}
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
			if dependenciesSatisfied(dependencies[idx], satisfied, indexByID) {
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
	var prefixes []string
	var ok bool

	// File first
	if filePrefixes, fileOk := fileCastePrefixes(caste); fileOk && len(filePrefixes) > 0 {
		prefixes = filePrefixes
		ok = true
	}
	// Fallback to hardcoded castePrefixes
	if !ok {
		prefixes, ok = castePrefixes[caste]
		if ok && len(prefixes) == 0 {
			ok = false
		}
	}
	// Fallback to default prefixes (file first, then hardcoded)
	if !ok {
		if fileDefaults, fileOk := fileDefaultPrefixes(); fileOk && len(fileDefaults) > 0 {
			prefixes = fileDefaults
		} else {
			prefixes = defaultPrefixes
		}
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
	if loaded := loadVisualsConfig(); loaded != nil {
		if emoji, ok := loaded.CasteEmojiMap[caste]; ok {
			return emoji
		}
	}
	if emoji, ok := casteEmojiMap[caste]; ok {
		return emoji
	}
	return "🐜"
}

func casteLabel(caste string) string {
	caste = normalizeCasteKey(caste)
	if loaded := loadVisualsConfig(); loaded != nil {
		if label, ok := loaded.CasteLabelMap[caste]; ok {
			return label
		}
	}
	if label, ok := casteLabelMap[caste]; ok {
		return label
	}
	return "Ant"
}

func casteANSIColor(caste string) string {
	caste = normalizeCasteKey(caste)
	if loaded := loadVisualsConfig(); loaded != nil {
		if color, ok := loaded.CasteColorMap[caste]; ok {
			return color
		}
	}
	return casteColorMap[caste]
}

func casteIdentity(caste string) string {
	// Classic house style (v5.4.0 caste-system.md): the caste glyph is always
	// followed by the ant — 🔨🐜 Builder Hammer-42 — except when the glyph IS
	// the generic ant, which stays single.
	emoji := casteEmoji(caste)
	if emoji != "🐜" {
		emoji += "🐜"
	}
	return emoji + " " + colorizeCaste(caste, casteLabel(caste))
}

// casteModelSlot maps each caste to the model slot its agent definition
// declares in `.claude/agents/ant/aether-<role>.md` frontmatter. DISPLAY
// ONLY: the platform routes agents natively from that frontmatter; nothing
// in the runtime reads this table to choose a model (automatic model routing
// was rejected 2026-07-28 and stays rejected). To change a role's model,
// edit the single `model:` line in its agent file — this table then fails
// TestCasteModelSlotMatchesAgentFrontmatter until it agrees, which is the
// point: a static table plus a parity test cannot go stale silently.
// Castes with no agent file (colonizer, dreamer, the curation ants…) are
// deliberately absent and get no model tag.
var casteModelSlot = map[string]string{
	"ambassador":    "sonnet",
	"archaeologist": "opus",
	"architect":     "opus",
	"auditor":       "opus",
	"builder":       "sonnet",
	"chaos":         "sonnet",
	"chronicler":    "sonnet",
	"fixer":         "sonnet",
	"gatekeeper":    "opus",
	"includer":      "sonnet",
	"keeper":        "sonnet",
	"measurer":      "opus",
	"medic":         "sonnet",
	"oracle":        "opus",
	"porter":        "sonnet",
	"probe":         "sonnet",
	"queen":         "opus",
	"route_setter":  "opus",
	"sage":          "opus",
	"scout":         "sonnet",
	"surveyor":      "sonnet",
	"tracker":       "opus",
	"watcher":       "sonnet",
	"weaver":        "sonnet",
}

// resolveCasteModel returns the display name of the model a caste's workers
// actually run on: the ANTHROPIC_DEFAULT_<SLOT>_MODEL environment variable's
// value when the user has redirected that slot (e.g. sonnet → glm-5-turbo),
// otherwise the slot name itself. Unknown castes get "" — no tag.
//
// The "inherit" branch below is a defensive fallback only: D-02 (Phase 196)
// pinned every role to a real slot, and TestRoutineBuilderIsSonnetNeverInherit
// fails by name if one is ever put back on the sentinel that means "whatever
// model happened to run last". The branch stays so such a role renders as
// "session" — visibly wrong — rather than silently as a real model.
func resolveCasteModel(caste string) string {
	slot, ok := casteModelSlot[normalizeCasteKey(caste)]
	if !ok {
		return ""
	}
	if slot == "inherit" {
		return "session"
	}
	if override := strings.TrimSpace(os.Getenv("ANTHROPIC_DEFAULT_" + strings.ToUpper(slot) + "_MODEL")); override != "" {
		return override
	}
	return slot
}

// casteIdentityWithModel is casteIdentity plus the resolved model tag —
// `🔨🐜 Builder [sonnet]` — used at SPAWN announcements only. Tagging every
// identity line (status lists, history rows, ~30 call sites) would be noise;
// the moment a worker is dispatched is where "which brain is this?" matters.
func casteIdentityWithModel(caste string) string {
	identity := casteIdentity(caste)
	if model := resolveCasteModel(caste); model != "" {
		identity += " [" + model + "]"
	}
	return identity
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

// shouldUseANSIColors decides whether to emit ANSI escape codes, separately
// from shouldRenderVisualOutput (which decides whether to draw the rich
// screen at all). A screen can be drawn in AETHER_OUTPUT_MODE=visual while
// piped into something that is not a real terminal -- a chat tool call, a
// captured log, a file -- and colour codes pasted into such a place are junk
// characters, not colour. Order: NO_COLOR always wins and turns colour off;
// an explicit force (AETHER_FORCE_COLOR=1 or CLICOLOR_FORCE) always wins and
// turns colour on; json output never carries colour; otherwise colour is on
// only when stdout is a real terminal.
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
	return isTerminalWriter(stdout)
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

// intSliceValue is stringSliceValue's sibling for a slice of ints -- the
// in-process typed value is []int, but a JSON-round-tripped completion file
// produces []interface{} of float64, exactly like every other numeric slice
// this codebase reduces before rendering.
func intSliceValue(value interface{}) []int {
	switch v := value.(type) {
	case []int:
		return append([]int{}, v...)
	case []interface{}:
		out := make([]int, 0, len(v))
		for _, item := range v {
			out = append(out, intValue(item))
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

func renderStalePublishBanner(stale stalePublishResult) string {
	var b strings.Builder
	b.WriteString("\n")

	emoji := "🐜"
	title := "PUBLISH STATUS"
	switch stale.Classification {
	case staleCritical:
		emoji = ""
		title = "STALE PUBLISH DETECTED"
	case staleWarning:
		emoji = "⚠️"
		title = "STALE PUBLISH DETECTED"
	case staleInfo:
		emoji = "ℹ️"
		title = "PUBLISH STATUS"
	}

	b.WriteString(renderBanner(emoji, title))
	b.WriteString(visualDividerStr())
	b.WriteString(fmt.Sprintf("Classification: %s\n", stale.Classification))
	b.WriteString(fmt.Sprintf("Binary version: %s\n", stale.BinaryVersion))
	b.WriteString(fmt.Sprintf("Hub version: %s\n", stale.HubVersion))
	b.WriteString(fmt.Sprintf("Channel: %s\n", stale.Channel))

	if len(stale.Components) > 0 {
		b.WriteString("\nComponents\n")
		for _, c := range stale.Components {
			b.WriteString(fmt.Sprintf("  - %s: %d found, expected %d\n", c.Name, c.Actual, c.Expected))
		}
	}

	b.WriteString(fmt.Sprintf("\n%s\n", stale.Message))
	b.WriteString("\nRecovery\n")
	b.WriteString(fmt.Sprintf("  %s\n", stale.RecoveryCommand))
	return b.String()
}

func truncateLines(text string, maxLines int) []string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}
	return append(lines[:maxLines], "...")
}

// dispatchTaskLine renders a dispatch's task for a one-line display.
//
// Phase 184 lets one worker own a chain of dependent steps, so a task can now
// be a numbered list. Written straight into the dispatch line that produced
// output like:
//
//	🔨 Builder Mason-67  1. Copy the daily-note templates
//	2. Copy the meeting templates
//
// with the continuation unindented and the layout broken. The worker still
// receives every step in full; only this display is summarised.
// dispatchCoveredTasksNote names the tasks a merged chain worker owns.
//
// Merged chains rendered as "first step (+2 more steps)", which says the
// worker does more but never which task IDs those are. An operator counting
// nine tasks against six workers has no way to see that three were folded in,
// and a real colony concluded three tasks had "no slot" and hand-reconciled
// work that already had a proper claimant. Naming the IDs is the difference
// between a plan you can reconcile against and one you have to guess at.
func dispatchCoveredTasksNote(dispatch codexBuildDispatch) string {
	if len(dispatch.CoveredTaskIDs) < 2 {
		return ""
	}
	return "  [covers tasks " + strings.Join(dispatch.CoveredTaskIDs, ", ") + "]"
}

// renderCoveredTaskSummary states the task-count arithmetic outright, so a
// worker count lower than the task count reads as deliberate folding rather
// than as tasks that were dropped.
func renderCoveredTaskSummary(dispatches []codexBuildDispatch) string {
	folded := 0
	covered := 0
	for _, dispatch := range dispatches {
		if len(dispatch.CoveredTaskIDs) < 2 {
			continue
		}
		folded++
		covered += len(dispatch.CoveredTaskIDs)
	}
	if folded == 0 {
		return ""
	}
	workerWord := "worker"
	if folded > 1 {
		workerWord = "workers"
	}
	return fmt.Sprintf("%d covered tasks were folded into %d %s — every task still has a claimant, listed above.\n",
		covered, folded, workerWord)
}

func dispatchTaskLine(task string) string {
	task = strings.TrimSpace(task)
	if !strings.Contains(task, "\n") {
		return task
	}
	lines := []string{}
	for _, line := range strings.Split(task, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	if len(lines) <= 1 {
		return strings.Join(lines, "")
	}
	// Only a merged chain is summarised, and it is recognised by the exact shape
	// the merge writes: every line numbered from 1 in order. Plenty of ordinary
	// task text runs to several lines -- a watcher's brief carries a follow-up
	// instruction on its own line -- and reporting that as "+1 more step" would
	// be a lie about how many things the worker was asked to do.
	for i, line := range lines {
		if !strings.HasPrefix(line, fmt.Sprintf("%d. ", i+1)) {
			// Not a merged chain: leave it exactly as it was rendered before.
			return task
		}
	}
	first := strings.TrimSpace(strings.TrimPrefix(lines[0], "1."))
	if len(lines) == 2 {
		return fmt.Sprintf("%s (+1 more step)", first)
	}
	return fmt.Sprintf("%s (+%d more steps)", first, len(lines)-1)
}
