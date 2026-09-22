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
	Name          string `json:"name"`
	MenuCommand   string `json:"menu_command"`
	Status        string `json:"status"` // "pass" | "fail"
	BashToolCalls int    `json:"bash_tool_calls"`
	FailureKind   string `json:"failure_kind,omitempty"`
	Detail        string `json:"detail,omitempty"`
	Retried       bool   `json:"retried"`
}

// journeyTrial is one full attempt at the declared journey.
type journeyTrial struct {
	SessionID string              `json:"session_id"`
	Caps      journeyCaps         `json:"caps"`
	Steps     []journeyStepResult `json:"steps"`
	Outcome   string              `json:"outcome"` // "pass" | "fail"
}

// journeyReport is the versioned, on-disk record of one journey run --
// written by writeJourneyReport, read back and judged by
// journeyGateVerdict.
type journeyReport struct {
	SchemaVersion string         `json:"schema_version"`
	Scope         string         `json:"scope"` // "whole-chain" | "one-step"
	Mode          string         `json:"mode"`  // "live"
	StepsDeclared int            `json:"steps_declared"`
	StepsExecuted int            `json:"steps_executed"`
	Trials        []journeyTrial `json:"trials"`
	Verdict       string         `json:"verdict"` // "pass" | "fail" | "pending"
	VerdictReason string         `json:"verdict_reason,omitempty"`
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

// journeyGateVerdict is the ONE place that decides whether a journey
// report passes the release gate. Plan 01 implements the checks the
// tracer slice can honestly make: scope must be whole-chain, mode must be
// live, and the declared step count must equal the executed count. Plan 04
// adds the trial-count and flaky/real rules to this SAME function -- there
// must never be a second verdict function anywhere in this package.
func journeyGateVerdict(r journeyReport) error {
	if r.Scope != "whole-chain" {
		return fmt.Errorf("journey report scope is %q, want %q -- a partial run can never satisfy the release gate", r.Scope, "whole-chain")
	}
	if r.Mode != "live" {
		return fmt.Errorf("journey report mode is %q, want %q", r.Mode, "live")
	}
	if r.StepsDeclared != r.StepsExecuted {
		return fmt.Errorf("journey report declares %d steps but only executed %d -- a truncated run must not read as clean", r.StepsDeclared, r.StepsExecuted)
	}
	return nil
}
