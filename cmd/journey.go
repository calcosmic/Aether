package cmd

// 207-01-PLAN.md (UED-07/UED-08): the transcript and report library both
// the live journey test (cmd/journey_live_test.go, //go:build journey) and
// the offline unit tests (cmd/journey_test.go, default build) use. Plain
// functions and types only -- this file registers no cobra command and
// adds no new subcommand to the binary (milestone rule: no new features,
// no new strict rules).
//
// Every assertion this library exposes resolves to a transcript tool_use
// entry, a <command-name> tag, or on-disk state -- never the chat's own
// prose (CLAUDE.md: "Assertions are on files produced and commands the
// chat ran, never on wording").

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// journeyReportSchemaVersion is the schema_version constant the journey
// report carries -- mirrors evalGateManifestSchemaVersion's convention.
const journeyReportSchemaVersion = "journey-report/v1"

// journeyDefaultReportRelativePath is where the journey report is written
// inside the practice project by default. Already inside
// sanctionedDataWritePrefixes (cmd/hook_cmds.go's worker-debug/ entry), so
// no new write-allowlist entry is needed.
const journeyDefaultReportRelativePath = ".aether/data/worker-debug/journey-report.json"

// --- Declared step vocabulary ---
//
// Same completeness convention as evalGateName (cmd/eval_gates.go): const
// block, names slice, names() helper, declared-membership predicate.

// journeyStep is the declared, closed vocabulary of lifecycle steps the
// journey drives.
type journeyStep string

const (
	journeyStepStart         journeyStep = "start"
	journeyStepSurvey        journeyStep = "survey"
	journeyStepDiscuss       journeyStep = "discuss"
	journeyStepSpecification journeyStep = "specification"
	journeyStepPlanFirst     journeyStep = "plan-first"
	journeyStepPlanSecond    journeyStep = "plan-second"
	journeyStepBuild         journeyStep = "build"
	journeyStepCheck         journeyStep = "check"
	journeyStepStatus        journeyStep = "status"
	journeyStepPause         journeyStep = "pause"
	journeyStepResume        journeyStep = "resume"
	journeyStepFinish        journeyStep = "finish"
	journeyStepArchive       journeyStep = "archive"
	journeyStepStartAgain    journeyStep = "start-again"
)

// journeyStepVocabulary is the declared, ordered lifecycle step list. Plan
// 01 declares all fourteen steps now; only "status" is driven through a
// real chat in this plan -- later plans in this phase expand coverage to
// every step without changing this declared vocabulary.
var journeyStepVocabulary = []journeyStep{
	journeyStepStart,
	journeyStepSurvey,
	journeyStepDiscuss,
	journeyStepSpecification,
	journeyStepPlanFirst,
	journeyStepPlanSecond,
	journeyStepBuild,
	journeyStepCheck,
	journeyStepStatus,
	journeyStepPause,
	journeyStepResume,
	journeyStepFinish,
	journeyStepArchive,
	journeyStepStartAgain,
}

// journeyStepNames returns the declared step names, in declared order.
func journeyStepNames() []string {
	names := make([]string, 0, len(journeyStepVocabulary))
	for _, s := range journeyStepVocabulary {
		names = append(names, string(s))
	}
	return names
}

// journeyStepDeclared reports whether s is in the declared vocabulary.
func journeyStepDeclared(s journeyStep) bool {
	for _, v := range journeyStepVocabulary {
		if v == s {
			return true
		}
	}
	return false
}

// journeyStepMenuCommandMap maps each declared step to the menu command
// that drives it. "plan-first" and "plan-second" both map to /ant-plan --
// the second /ant-plan pass is what exercises the Scout new_evidence
// fingerprint fix (blocker #3, per 207-RESEARCH.md Q3/Q4); it is a
// structural "run /ant-plan twice" requirement, not a second command name.
var journeyStepMenuCommandMap = map[journeyStep]string{
	journeyStepStart:         "/ant-init",
	journeyStepSurvey:        "/ant-colonize",
	journeyStepDiscuss:       "/ant-discuss",
	journeyStepSpecification: "/ant-spec",
	journeyStepPlanFirst:     "/ant-plan",
	journeyStepPlanSecond:    "/ant-plan",
	journeyStepBuild:         "/ant-build",
	journeyStepCheck:         "/ant-continue",
	journeyStepStatus:        "/ant-status",
	journeyStepPause:         "/ant-pause",
	journeyStepResume:        "/ant-resume",
	journeyStepFinish:        "/ant-seal",
	journeyStepArchive:       "/ant-entomb",
	journeyStepStartAgain:    "/ant-init",
}

// journeyStepMenuCommand returns the menu command that drives step, and
// false if step is not in the declared vocabulary.
func journeyStepMenuCommand(step journeyStep) (string, bool) {
	cmd, ok := journeyStepMenuCommandMap[step]
	return cmd, ok
}

// journeyContainsString reports whether want is present in list. Shared by
// the offline tests (cmd/journey_test.go) and the live journey harness
// (cmd/journey_live_test.go, //go:build journey).
func journeyContainsString(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

// --- Transcript reading ---

// journeyTranscriptEntry is the minimal shape of one JSONL line this
// package reads from a Claude Code transcript -- only the fields the
// journey harness's own assertions need, never the full envelope. Content
// is left as raw JSON because it is polymorphic: sometimes a bare string
// (a typed slash-command prompt), sometimes a block array (assistant
// tool_use, user tool_result, or an isMeta wrapper's text block) -- see
// cmd/testdata/stop-hook/menu-command-transcript.jsonl, a real captured
// session, for both shapes.
type journeyTranscriptEntry struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type journeyContentBlock struct {
	Type string `json:"type"`
	Name string `json:"name"`
	Text string `json:"text"`
	// Content carries a tool_result block's own payload (cmd/hook_cmds.go's
	// transcriptContentBlock models the identical shape) -- a real
	// transcript has shown this as either a plain string or an array of
	// blocks carrying their own "text" field, never the top-level "text"
	// field above. Read it with toolResultText.
	Content json.RawMessage `json:"content"`
}

// journeyScanTranscriptLines calls fn once per non-empty JSONL line at
// transcriptPath that decodes as a journeyTranscriptEntry. A line that
// does not decode (non-JSON noise, or an envelope shape this struct does
// not model) is silently skipped -- this library only needs the fields
// above, never the full envelope.
func journeyScanTranscriptLines(transcriptPath string, fn func(journeyTranscriptEntry)) error {
	f, err := os.Open(transcriptPath)
	if err != nil {
		return fmt.Errorf("open transcript %s: %w", transcriptPath, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry journeyTranscriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		fn(entry)
	}
	return scanner.Err()
}

// journeyBashToolCallCount counts JSONL entries at transcriptPath whose
// message content carries a tool_use block named Bash -- the same
// selection scripts/proof-screens-reach-the-owner.sh already makes with
// jq (`select(.type=="tool_use" and .name=="Bash")`), expressed in Go.
func journeyBashToolCallCount(transcriptPath string) (int, error) {
	count := 0
	err := journeyScanTranscriptLines(transcriptPath, func(entry journeyTranscriptEntry) {
		if len(entry.Message.Content) == 0 {
			return
		}
		var blocks []journeyContentBlock
		if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
			return // content is a bare string on this line, not a block array
		}
		for _, b := range blocks {
			if b.Type == "tool_use" && b.Name == "Bash" {
				count++
			}
		}
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// journeyCommandNamePattern matches a <command-name>...</command-name> tag
// in a transcript's user-message content -- the exact shape a typed menu
// command expands into (verified against
// cmd/testdata/stop-hook/menu-command-transcript.jsonl, a real captured
// /ant-status session).
var journeyCommandNamePattern = regexp.MustCompile(`<command-name>([^<]+)</command-name>`)

// journeyMenuCommandNames returns every <command-name> value found in user
// messages at transcriptPath, in file order -- the proof that a menu
// command, not a raw CLI call, drove a step.
func journeyMenuCommandNames(transcriptPath string) ([]string, error) {
	var names []string
	err := journeyScanTranscriptLines(transcriptPath, func(entry journeyTranscriptEntry) {
		if entry.Type != "user" || len(entry.Message.Content) == 0 {
			return
		}

		var asString string
		if err := json.Unmarshal(entry.Message.Content, &asString); err == nil {
			for _, m := range journeyCommandNamePattern.FindAllStringSubmatch(asString, -1) {
				names = append(names, m[1])
			}
			return
		}

		// Content is a block array (e.g. an isMeta wrapper message), not a
		// bare string -- scan its text blocks too.
		var blocks []journeyContentBlock
		if err := json.Unmarshal(entry.Message.Content, &blocks); err == nil {
			for _, b := range blocks {
				for _, m := range journeyCommandNamePattern.FindAllStringSubmatch(b.Text, -1) {
					names = append(names, m[1])
				}
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return names, nil
}

// --- Printed refusals ---
//
// 208-07-PLAN.md (UED-10): a refusal a real chat met has its way out proven
// only when the command it printed actually runs. journeyPrintedRefusals
// finds every refusal a step's own transcript actually printed;
// printedCommandToRuntimeCommand turns what it printed into a command the
// live harness can execute.

// journeyPrintedRefusal is one refusal found in a step's own transcript: the
// exact next command from its "Next:" line, and the raw matched text for a
// failure message that needs to show the reader what was actually printed.
type journeyPrintedRefusal struct {
	NextCommand string
	Raw         string
}

// journeyPrintedRefusalNextLineRe matches the exact "Next: `<command>`" line
// renderRefusal (cmd/refusal.go) writes for every refusal block, on its own
// line with no other prefix. This is the stable marker: an ordinary sentence
// that happens to carry a backticked command is never mistaken for a
// refusal, because it is never preceded by this exact "Next:" label.
var journeyPrintedRefusalNextLineRe = regexp.MustCompile("(?m)^Next: `([^`\n]+)`")

// journeyPrintedRefusals reads the transcript at transcriptPath and returns
// every printed refusal found, in the order they appear. It looks only at
// the two places a real renderRefusal block can appear in a transcript: an
// assistant's own text block (the chat relaying what it saw), and a
// user-role tool_result block (a Bash call's own captured stdout/stderr) --
// the same two shapes journeyMenuCommandNames and owedScreenFrom already
// read. A transcript line that does not decode, or an empty transcript,
// yields no refusals and no error.
func journeyPrintedRefusals(transcriptPath string) ([]journeyPrintedRefusal, error) {
	var found []journeyPrintedRefusal
	err := journeyScanTranscriptLines(transcriptPath, func(entry journeyTranscriptEntry) {
		if len(entry.Message.Content) == 0 {
			return
		}
		var blocks []journeyContentBlock
		if err := json.Unmarshal(entry.Message.Content, &blocks); err != nil {
			return // content is a bare string on this line, not a block array
		}
		for _, b := range blocks {
			var text string
			switch {
			case b.Type == "text" && entry.Message.Role == "assistant":
				text = b.Text
			case b.Type == "tool_result" && entry.Message.Role == "user":
				text = toolResultText(b.Content)
			default:
				continue
			}
			for _, m := range journeyPrintedRefusalNextLineRe.FindAllStringSubmatch(text, -1) {
				found = append(found, journeyPrintedRefusal{
					NextCommand: strings.TrimSpace(m[1]),
					Raw:         strings.TrimSpace(m[0]),
				})
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// printedCommandToRuntimeCommand reverses platformCommandName's forward
// mapping (cmd/codex_visuals.go): a printed command already starting with
// `aether ` is returned unchanged; a printed `/ant-<verb> ...` has its verb
// looked up directly in wrapperCommandNames -- the same table
// platformCommandName reads when it produces the menu form, so this reverse
// mapping can never drift from the forward one -- and is rewritten to
// `aether <verb> ...` with every argument after the verb preserved byte for
// byte. A verb in neither form (not `aether `-prefixed, and either not
// `/ant-`-prefixed or naming a verb wrapperCommandNames does not carry)
// returns not-ok rather than a guess.
func printedCommandToRuntimeCommand(printed string) (string, bool) {
	printed = strings.TrimSpace(printed)
	if printed == "aether" || strings.HasPrefix(printed, "aether ") {
		return printed, true
	}
	if !strings.HasPrefix(printed, "/ant-") {
		return "", false
	}
	rest := strings.TrimPrefix(printed, "/ant-")
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", false
	}
	verb := fields[0]
	if !wrapperCommandNames[verb] {
		return "", false
	}
	remainder := strings.TrimPrefix(rest, verb)
	return "aether " + verb + remainder, true
}

// journeyPrintedRefusalEmptyNextCommandRe matches a "Next: `...`" line whose
// own backticked span may be empty -- unlike journeyPrintedRefusalNextLineRe
// above, which requires at least one character so the extractor never
// reports a refusal with no next command at all. This looser form exists
// only to recognise journeyNextCommandFailureReason's fourth condition: a
// printed next command's OWN subprocess output being itself a refusal block
// with nothing to run next.
var journeyPrintedRefusalEmptyNextCommandRe = regexp.MustCompile("(?m)^Next: `([^`\n]*)`")

// journeyNextCommandTimeoutFraction sizes a printed next command's own
// subprocess timeout as a fraction of the step's own wall-clock cap
// (journeyCaps.WallClockSecs) -- well inside it, per 208-07-PLAN.md Task 2,
// never a new fixed constant of its own.
const journeyNextCommandTimeoutFraction = 4

// journeyNextCommandFailureReason classifies the outcome of running one
// printed next command as a real subprocess, and is the one place all four
// named failure conditions (208-07-PLAN.md Task 2) are decided -- proven
// directly, with captured subprocess output, by a unit-level test in
// cmd/journey_test.go, never only through a live chat run. Returns "" when
// the outcome is not a failure: a non-zero exit that matches none of the
// four conditions may legitimately mean a recovery command found more work
// still outstanding.
//
// The four conditions, in the order checked: (1) the printed command could
// not be mapped to a runtime command at all; (2) the subprocess timed out;
// (3) the subprocess reported an unknown command or an unknown flag; (4) the
// subprocess's own output is itself a refusal block whose next command is
// empty.
func journeyNextCommandFailureReason(mapped, timedOut bool, output string) string {
	if !mapped {
		return "could not be mapped to a runtime command"
	}
	if timedOut {
		return "timed out"
	}
	lower := strings.ToLower(output)
	if strings.Contains(lower, "unknown command") {
		return "reported an unknown command"
	}
	if strings.Contains(lower, "unknown flag") || strings.Contains(lower, "unknown shorthand flag") {
		return "reported an unknown flag"
	}
	if m := journeyPrintedRefusalEmptyNextCommandRe.FindStringSubmatch(output); m != nil && strings.TrimSpace(m[1]) == "" {
		return "its own output is itself a refusal with nothing to run next"
	}
	return ""
}

// --- Failure classification ---

// journeyFailureKind is the declared, closed vocabulary a journey failure
// classifies into.
type journeyFailureKind string

const (
	journeyFailureTransient journeyFailureKind = "transient"
	journeyFailureReal      journeyFailureKind = "real"
)

// journeyTransientFailurePattern is the same small regex
// scripts/proof-screens-reach-the-owner.sh already uses (rate limit,
// overloaded, timed out) -- deliberately not widened to cline's full
// 5-category system, per the milestone's "one review round" framing
// (207-RESEARCH.md Q6).
var journeyTransientFailurePattern = regexp.MustCompile(`(?i)rate.?limit|overloaded|timed?.?out`)

// classifyJourneyFailure classifies failure text as transient (worth
// retrying once) or real (everything else, including the empty string).
func classifyJourneyFailure(text string) journeyFailureKind {
	if journeyTransientFailurePattern.MatchString(text) {
		return journeyFailureTransient
	}
	return journeyFailureReal
}

// --- Report ---

// journeyCaps records the caps in force for the claude -p invocation(s)
// driving one trial -- every chat call this phase makes carries these, and
// they are recorded here rather than only logged to stdout.
type journeyCaps struct {
	MaxTurns      int     `json:"max_turns"`
	MaxBudgetUSD  float64 `json:"max_budget_usd,omitempty"`
	BudgetCapped  bool    `json:"budget_capped"`
	WallClockSecs int     `json:"wall_clock_seconds"`
}

// journeyStepResult is the outcome of one lifecycle step within one trial.
type journeyStepResult struct {
	Name          string   `json:"name"`
	MenuCommand   string   `json:"menu_command"`
	Status        string   `json:"status"` // "pass" | "fail" | "not-reached"
	BashToolCalls int      `json:"bash_tool_calls"`
	FailureKind   string   `json:"failure_kind,omitempty"`
	Detail        string   `json:"detail,omitempty"`
	Retried       bool     `json:"retried"`
	TrapIDs       []string `json:"trap_ids,omitempty"`
	// RefusalsPrinted is how many printed refusals (journeyPrintedRefusals)
	// this step's own transcript carried -- 208-07-PLAN.md (UED-10). Zero is
	// a legitimate, honestly-reported value: a clean step meets none.
	RefusalsPrinted int `json:"refusals_printed"`
	// NextCommandsRun is how many of those printed refusals' own next
	// commands were actually run for real, as a subprocess, in the same
	// practice project (journeyRunPrintedNextCommands, cmd/journey_live_test.go).
	NextCommandsRun int `json:"next_commands_run"`
}

// --- Trial outcome vocabulary ---
//
// Same completeness convention as evalGateName and journeyStep: const block,
// names slice, names() helper, declared-membership predicate.

// journeyTrialOutcome is the declared, closed vocabulary a trial's overall
// result classifies into.
type journeyTrialOutcome string

const (
	// journeyTrialPassed: every step in the trial passed.
	journeyTrialPassed journeyTrialOutcome = "passed"
	// journeyTrialFlakyFailure: the trial failed, but every failure
	// classified transient (worth retrying, not a product defect on its
	// own).
	journeyTrialFlakyFailure journeyTrialOutcome = "flaky-failure"
	// journeyTrialRealFailure: at least one failure in the trial classified
	// real.
	journeyTrialRealFailure journeyTrialOutcome = "real-failure"
	// journeyTrialIncomplete: the trial was cut short before every declared
	// step ran -- never counted as passed, regardless of what ran before
	// the cutoff looked clean.
	journeyTrialIncomplete journeyTrialOutcome = "incomplete"
)

var journeyTrialOutcomeVocabulary = []journeyTrialOutcome{
	journeyTrialPassed,
	journeyTrialFlakyFailure,
	journeyTrialRealFailure,
	journeyTrialIncomplete,
}

// journeyTrialOutcomeNames returns the declared trial-outcome names.
func journeyTrialOutcomeNames() []string {
	names := make([]string, 0, len(journeyTrialOutcomeVocabulary))
	for _, o := range journeyTrialOutcomeVocabulary {
		names = append(names, string(o))
	}
	return names
}

// journeyTrialOutcomeDeclared reports whether o is in the declared
// vocabulary.
func journeyTrialOutcomeDeclared(o journeyTrialOutcome) bool {
	for _, v := range journeyTrialOutcomeVocabulary {
		if v == o {
			return true
		}
	}
	return false
}

// journeyTrial is one full attempt at the declared journey.
type journeyTrial struct {
	Index         int                 `json:"index"`
	SessionID     string              `json:"session_id"`
	StartedAt     string              `json:"started_at,omitempty"`
	EndedAt       string              `json:"ended_at,omitempty"`
	Caps          journeyCaps         `json:"caps"`
	Steps         []journeyStepResult `json:"steps"`
	StepsDeclared int                 `json:"steps_declared"`
	StepsExecuted int                 `json:"steps_executed"`
	// IncompleteAtStep names the step the trial stopped at when Outcome is
	// journeyTrialIncomplete -- a trial that stopped part way must show
	// where, never omit it silently.
	IncompleteAtStep string `json:"incomplete_at_step,omitempty"`
	Outcome          string `json:"outcome"` // one of journeyTrialOutcomeVocabulary
}

// journeyExpectedRedResult is the status step's genuine, real-run result for
// one case in the committed expected-red register (cmd/journey_expected_red.go),
// plus the unregistered-gap case: a failing check the register never named.
type journeyExpectedRedResult struct {
	ID       string `json:"id,omitempty"`
	ClosedBy string `json:"closed_by,omitempty"`
	// Result is one of "still-red" (the registered case is honestly red
	// today, as recorded), "now-green" (the register is stale -- the case
	// must be removed, never relied on to keep passing), or
	// "unregistered-gap" (a real failing check this run found that has no
	// matching case in the register at all).
	Result string `json:"result"`
	Detail string `json:"detail,omitempty"`
}

// journeyReport is the versioned, on-disk record of one journey run --
// written by writeJourneyReport, read back and judged by
// journeyGateVerdict.
type journeyReport struct {
	SchemaVersion string                     `json:"schema_version"`
	Scope         string                     `json:"scope"` // "whole-chain" | "one-step"
	Mode          string                     `json:"mode"`  // "live"
	StepsDeclared int                        `json:"steps_declared"`
	StepsExecuted int                        `json:"steps_executed"`
	Trials        []journeyTrial             `json:"trials"`
	ExpectedRed   []journeyExpectedRedResult `json:"expected_red,omitempty"`
	Verdict       string                     `json:"verdict"` // "pass" | "fail" | "pending"
	VerdictReason string                     `json:"verdict_reason,omitempty"`
}

// writeJourneyReport writes r as JSON to path, creating parent directories
// and stamping the schema version.
func writeJourneyReport(path string, r journeyReport) error {
	r.SchemaVersion = journeyReportSchemaVersion
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create journey report directory for %s: %w", path, err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal journey report: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write journey report %s: %w", path, err)
	}
	return nil
}

// readJourneyReport reads back a journey report written by
// writeJourneyReport.
func readJourneyReport(path string) (journeyReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return journeyReport{}, fmt.Errorf("read journey report %s: %w", path, err)
	}
	var r journeyReport
	if err := json.Unmarshal(data, &r); err != nil {
		return journeyReport{}, fmt.Errorf("unmarshal journey report %s: %w", path, err)
	}
	return r, nil
}

// journeyMinimumTrials is the fewest trials a whole-chain live report may
// carry and still be judged -- "three trials" is a milestone-level
// constraint (207-CONTEXT.md, must_haves), not a tunable.
const journeyMinimumTrials = 3

// journeyGateVerdict is the ONE place that decides whether a journey
// report passes the release gate. Plan 01 implemented the checks the
// tracer slice could honestly make (scope, mode, and a report-level
// declared/executed step count). Plan 04 extends this SAME function with
// the trial-count and flaky/real rules -- there must never be a second
// verdict function anywhere in this package.
//
// Refusals are enforced in this fixed order, each naming what was wrong:
//  1. scope must be "whole-chain"
//  2. mode must be "live"
//  3. at least journeyMinimumTrials trials
//  4. every trial's declared step count equals its executed step count
//  5. no trial may be a real failure, and not every trial may have failed
//     (three flaky failures in a row is itself a real failure)
//  6. every expected-red register case must be reported "still-red"
//  7. any check this run found failing with no matching register case
//     fails outright (an unregistered, silently-discovered gap)
func journeyGateVerdict(r journeyReport) error {
	if r.Scope != "whole-chain" {
		return fmt.Errorf("journey report scope is %q, want %q -- a partial run can never satisfy the release gate", r.Scope, "whole-chain")
	}
	if r.Mode != "live" {
		return fmt.Errorf("journey report mode is %q, want %q", r.Mode, "live")
	}
	if len(r.Trials) < journeyMinimumTrials {
		return fmt.Errorf("journey report carries %d trial(s), want at least %d -- a reduced trial count can never satisfy the release gate", len(r.Trials), journeyMinimumTrials)
	}

	for i, trial := range r.Trials {
		if trial.StepsDeclared != trial.StepsExecuted {
			return fmt.Errorf("trial %d declares %d steps but only executed %d (stopped at %q) -- a trial that stopped part way must not read as clean", i, trial.StepsDeclared, trial.StepsExecuted, trial.IncompleteAtStep)
		}
	}

	allFailed := true
	for i, trial := range r.Trials {
		outcome := journeyTrialOutcome(trial.Outcome)
		if !journeyTrialOutcomeDeclared(outcome) {
			return fmt.Errorf("trial %d carries undeclared outcome %q, want one of %v", i, trial.Outcome, journeyTrialOutcomeNames())
		}
		if outcome == journeyTrialRealFailure {
			failingStep := journeyFirstFailingStepName(trial)
			return fmt.Errorf("trial %d is a real failure at step %q -- a real failure in any trial refuses the whole run", i, failingStep)
		}
		if outcome != journeyTrialFlakyFailure {
			allFailed = false
		}
	}
	if allFailed {
		return fmt.Errorf("all %d trials failed (flaky-failure or worse) -- three consecutive flaky failures is itself a real failure, not noise", len(r.Trials))
	}

	for _, c := range r.ExpectedRed {
		if c.Result == "now-green" {
			return fmt.Errorf("expected-red register is stale: case %q now reports now-green -- remove it from cmd/testdata/journey/expected-red.json, never loosen this gate to keep passing it", c.ID)
		}
	}
	for _, c := range r.ExpectedRed {
		if c.Result == "unregistered-gap" {
			return fmt.Errorf("a check this run found failing has no matching case in the expected-red register: %s -- either fix it or add a named, closed-by case to cmd/testdata/journey/expected-red.json", c.Detail)
		}
	}

	return nil
}

// journeyFirstFailingStepName returns the name of the first step in trial
// whose status is not "pass" (a real failure, since flaky and passed
// outcomes never reach this call with a failing step at all) -- used only
// to name the offending step in journeyGateVerdict's refusal message.
func journeyFirstFailingStepName(trial journeyTrial) string {
	for _, step := range trial.Steps {
		if step.Status != "pass" && step.Status != "not-reached" {
			return step.Name
		}
	}
	return "unknown"
}

// journeyDeriveTrialOutcome classifies a completed or cut-short trial from
// its own step results -- the one place this decision is made, so the live
// harness and the offline tests derive it identically.
//
//   - Every step "pass" (and executed == declared) -> passed.
//   - executed < declared (a step failed and stopped the trial, or the
//     trial was otherwise cut short) -> incomplete, naming the step it
//     stopped at.
//   - Every step ran (executed == declared) but at least one failed:
//     every failing step's failure_kind is "transient" -> flaky-failure;
//     otherwise -> real-failure.
func journeyDeriveTrialOutcome(steps []journeyStepResult, declared, executed int) (outcome journeyTrialOutcome, incompleteAtStep string) {
	if executed < declared {
		for _, s := range steps {
			if s.Status != "pass" {
				return journeyTrialIncomplete, s.Name
			}
		}
		return journeyTrialIncomplete, ""
	}

	sawFailure := false
	allTransient := true
	for _, s := range steps {
		if s.Status != "pass" {
			sawFailure = true
			if s.FailureKind != string(journeyFailureTransient) {
				allTransient = false
			}
		}
	}
	if !sawFailure {
		return journeyTrialPassed, ""
	}
	if allTransient {
		return journeyTrialFlakyFailure, ""
	}
	return journeyTrialRealFailure, ""
}

// journeyExpectedRedCaseForVerb returns the registered case (if any) whose
// own detail prose names verb as an `aether <verb>` command, and whether one
// was found. Reads the register's own detail field via a simple substring
// check rather than duplicating statusAdvisedCommandRe's regex extraction
// (cmd/journey_expected_red.go) -- both derive the same fact, one by regex
// extraction (going forward, prose to verb), the other by substring
// containment (going backward, verb to prose); a case's detail always
// carries its own advised command as a backtick-quoted `aether <verb>` span
// by construction (207-03-PLAN.md Task 1), so containment is sufficient and
// avoids a second regex.
func journeyExpectedRedCaseForVerb(cases []journeyExpectedRedCase, verb string) (journeyExpectedRedCase, bool) {
	needle := "`aether " + verb
	for _, c := range cases {
		if strings.Contains(c.Detail, needle) {
			return c, true
		}
	}
	return journeyExpectedRedCase{}, false
}

// journeyEvaluateExpectedRed turns the real, live result of the status
// step's own guidance check (missingWrapperVerbs, straight from
// statusGuidanceCommandsWithoutMenuWrapper against the real practice
// project) into the report's expected_red section: every registered case
// still reflected in missingWrapperVerbs is "still-red"; a registered case
// NOT reflected is "now-green" (the register has gone stale); a missing
// verb with no registered case at all is a new "unregistered-gap".
func journeyEvaluateExpectedRed(registered []journeyExpectedRedCase, missingWrapperVerbs []string) []journeyExpectedRedResult {
	var results []journeyExpectedRedResult

	stillRed := make(map[string]bool, len(registered))
	for _, verb := range missingWrapperVerbs {
		if c, ok := journeyExpectedRedCaseForVerb(registered, verb); ok {
			stillRed[c.ID] = true
		} else {
			results = append(results, journeyExpectedRedResult{
				Result: "unregistered-gap",
				Detail: fmt.Sprintf("the status card advises `aether %s`, which has no menu wrapper, and no case in the register names it", verb),
			})
		}
	}
	for _, c := range registered {
		result := "now-green"
		if stillRed[c.ID] {
			result = "still-red"
		}
		results = append(results, journeyExpectedRedResult{
			ID: c.ID, ClosedBy: c.ClosedBy, Result: result, Detail: c.Detail,
		})
	}
	return results
}

// journeyReportSummary renders the short, plain-English, end-of-run summary
// Task 2's own action requires: how many trials passed, which steps failed
// and whether each failure was noise or real, the caps in force, the
// observed wall clock and cost, and one line per expected-red case. Every
// repo-invented word here ("colony", "chamber", ...) would need explaining
// in the same sentence it appears in -- this summary deliberately uses none,
// since it only ever names journey steps, menu commands, and plain counts.
func journeyReportSummary(r journeyReport) string {
	var b strings.Builder
	passed := 0
	for _, t := range r.Trials {
		if t.Outcome == string(journeyTrialPassed) {
			passed++
		}
	}
	fmt.Fprintf(&b, "%d of %d trial(s) passed cleanly.\n", passed, len(r.Trials))

	// 208-07-PLAN.md (UED-10): name the printed-refusal / next-command
	// totals for every trial, even when both are zero -- a run that met no
	// refusals must say exactly that, never read as a silent pass.
	for _, t := range r.Trials {
		printed, ran := 0, 0
		for _, s := range t.Steps {
			printed += s.RefusalsPrinted
			ran += s.NextCommandsRun
		}
		fmt.Fprintf(&b, "trial %d: %d printed refusal(s) found, %d next command(s) run.\n", t.Index, printed, ran)
	}

	for _, t := range r.Trials {
		if t.Outcome == string(journeyTrialPassed) {
			continue
		}
		for _, s := range t.Steps {
			if s.Status == "pass" || s.Status == "not-reached" {
				continue
			}
			noise := "a real problem"
			if s.FailureKind == string(journeyFailureTransient) {
				noise = "noise (a transient failure, not a product defect)"
			}
			fmt.Fprintf(&b, "trial %d: step %q failed -- %s\n", t.Index, s.Name, noise)
		}
		if t.IncompleteAtStep != "" {
			fmt.Fprintf(&b, "trial %d: stopped early at step %q\n", t.Index, t.IncompleteAtStep)
		}
	}

	for i, t := range r.Trials {
		if i == 0 || t.Caps != r.Trials[0].Caps {
			fmt.Fprintf(&b, "trial %d caps: max-turns=%d wall-clock=%ds max-budget-usd=%.2f (capped=%v)\n",
				t.Index, t.Caps.MaxTurns, t.Caps.WallClockSecs, t.Caps.MaxBudgetUSD, t.Caps.BudgetCapped)
		}
	}

	for _, c := range r.ExpectedRed {
		switch c.Result {
		case "still-red":
			fmt.Fprintf(&b, "expected-red case %q is still red, as recorded -- closed by %s.\n", c.ID, c.ClosedBy)
		case "now-green":
			fmt.Fprintf(&b, "expected-red case %q now passes -- the register entry is stale and must be removed.\n", c.ID)
		case "unregistered-gap":
			fmt.Fprintf(&b, "a new, unregistered gap was found: %s\n", c.Detail)
		}
	}

	return b.String()
}
