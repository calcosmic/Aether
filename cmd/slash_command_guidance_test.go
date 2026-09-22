package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// LOUD-08. `cmd/unblock_cmd.go` told users to "Run /ant-unblock --dispatch" —
// a slash command that existed on no platform. Decision D-02 chose to BUILD the
// wrapper rather than reword the guidance, because the Go side (`aether unblock`
// with --phase/--dispatch/--fixer-mode) was already complete and working.
//
// This test generalises past that one string: any /ant-* slash command that Go
// code tells a user to run must actually exist as a wrapper on both maintained
// platforms. Fixing only the known-bad string would leave the next one free to
// appear the same way.

var slashCommandRe = regexp.MustCompile(`/ant-([a-z][a-z0-9-]*)`)

// Slash commands that are deliberately not backed by a wrapper file.
var slashCommandExemptions = map[string]string{}

func TestSlashCommandGuidancePointsAtRealCommands(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	cmdDir := filepath.Join(root, "cmd")
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}

	type reference struct{ file, command string }
	var refs []reference

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(cmdDir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range slashCommandRe.FindAllStringSubmatch(string(raw), -1) {
			refs = append(refs, reference{file: e.Name(), command: m[1]})
		}
	}

	if len(refs) == 0 {
		t.Fatal("found no /ant-* references in cmd/*.go — this test would pass vacuously; the extractor is broken")
	}

	seen := map[string]bool{}
	for _, r := range refs {
		if seen[r.command] || slashCommandExemptions[r.command] != "" {
			continue
		}
		seen[r.command] = true

		for _, platform := range []string{
			filepath.Join(".claude", "commands", "ant"),
			filepath.Join(".opencode", "commands", "ant"),
		} {
			wrapper := filepath.Join(root, platform, r.command+".md")
			if _, err := os.Stat(wrapper); err != nil {
				t.Errorf("%s tells the user to run /ant-%s, but %s does not exist — guidance that names a command available on no platform is a dead end for the user (LOUD-08)",
					r.file, r.command, filepath.Join(platform, r.command+".md"))
			}
		}
	}
}

// The wrapper this phase added must stay wired to the CLI that already works,
// and must not drift from its OpenCode mirror.
func TestUnblockWrapperIsWiredAndAtParity(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	yamlPath := filepath.Join(root, ".aether", "commands", "unblock.yaml")
	yaml, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read unblock.yaml: %v — the YAML source is the head of the generation chain (LOUD-08)", err)
	}
	if !strings.Contains(string(yaml), "aether unblock") {
		t.Error("unblock.yaml does not route to `aether unblock`; the wrapper must delegate to the existing CLI, not reimplement gate recovery")
	}

	claude, err := os.ReadFile(filepath.Join(root, ".claude", "commands", "ant", "unblock.md"))
	if err != nil {
		t.Fatalf("read Claude unblock wrapper: %v", err)
	}
	opencode, err := os.ReadFile(filepath.Join(root, ".opencode", "commands", "ant", "unblock.md"))
	if err != nil {
		t.Fatalf("read OpenCode unblock wrapper: %v", err)
	}
	if string(claude) != string(opencode) {
		t.Error("the Claude and OpenCode unblock wrappers differ; platform wrappers are maintained at parity")
	}

	// The Go command is the thing the wrapper promises. If a flag it documents
	// disappears, the wrapper starts lying.
	target, _, err := rootCmd.Find([]string{"unblock"})
	if err != nil || target == nil || target == rootCmd {
		t.Fatal("`aether unblock` is not a registered command — the wrapper would point at nothing")
	}
	for _, flag := range []string{"dispatch", "fixer-mode", "phase"} {
		if target.Flags().Lookup(flag) == nil {
			t.Errorf("`aether unblock` no longer has --%s, but the wrapper documents it", flag)
		}
	}
}

// A refusal block (renderRefusal, cmd/refusal.go) is an owner-facing screen
// like any other -- it always closes with "Run `aether report`" -- so it
// joins the classic voice corpus here, the one place a new screen's own
// registering init belongs. This is what makes source 2 of
// screenAdvisedProgramCommands (below) pick up `aether report` as advised,
// so TestScreenGuidanceNamesCommandsTheOwnerCanRun is sensitive to
// .claude/commands/ant/report.md disappearing.
func init() {
	registerVoiceScreen("refusal", func(t *testing.T) string {
		return renderRefusal(refuse("colonize-finalize-missing-timestamp"))
	})
}

// 208-03-PLAN.md Task 2 (UED-13/UED-14): the check above only ever caught a
// /ant-* slash command with no wrapper. That is a subset of the real
// failure mode -- a screen can just as easily advise an `aether <verb>`
// program command with no menu wrapper behind it, the exact shape of the
// sixth 2026-09-21 blocker this phase's Task 1 closed. This widens the
// check from slash commands to every program command an owner-facing
// screen can put in front of the owner.

// guidanceCommandRef is one `aether <verb> ...` command a screen advised,
// together with the source it came from -- used only to make a failure
// message name where the advice was seen, never to change what "missing a
// wrapper" means.
type guidanceCommandRef struct {
	Command string
	Source  string
}

// seedOpenFlagForGuidance writes a single unresolved issue flag so
// activeFlagGuidedAction fires alongside middenGuidedAction, exactly as
// TestStatusGuidanceGapsAreReportedSeparatelyAndInOrder
// (journey_expected_red_test.go) already seeds a second guided action.
func seedOpenFlagForGuidance(t *testing.T, s interface{ SaveJSON(string, interface{}) error }) {
	t.Helper()
	flags := colony.FlagsFile{
		Version: "1.0.0",
		Decisions: []colony.FlagEntry{
			{
				ID:          "flag-1",
				Type:        "issue",
				Description: "a genuine open issue",
				Source:      "test",
				CreatedAt:   "2026-09-21T00:00:00Z",
				Resolved:    false,
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatalf("seed pending-decisions.json: %v", err)
	}
}

// screenAdvisedProgramCommands collects every `aether <verb> ...` command an
// owner-facing screen can put in front of the owner, from exactly three
// sources and no others: the status card's own guided actions and warnings
// (statusGuidanceAdvisedCommands, seeded the same way
// seedUnacknowledgedMiddenFailure and seedOpenFlagForGuidance make a second
// guided action fire), every backticked `aether <verb>` span inside every
// screen renderedVoiceScreens(t) returns (extracted with the existing
// statusAdvisedCommandRe -- never a second regular expression), and every
// non-empty NextCommand in refusalRegistry. It never parses cmd/status.go's
// source.
func screenAdvisedProgramCommands(t *testing.T) []guidanceCommandRef {
	t.Helper()

	var refs []guidanceCommandRef

	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	seedUnacknowledgedMiddenFailure(t, s)
	seedOpenFlagForGuidance(t, s)

	advised, err := statusGuidanceAdvisedCommands(s, tmpDir)
	if err != nil {
		t.Fatalf("statusGuidanceAdvisedCommands: %v", err)
	}
	for _, command := range advised {
		refs = append(refs, guidanceCommandRef{Command: command, Source: "status card"})
	}

	for _, screen := range renderedVoiceScreens(t) {
		for _, match := range statusAdvisedCommandRe.FindAllStringSubmatch(screen.Rendered, -1) {
			refs = append(refs, guidanceCommandRef{Command: match[1], Source: "voice screen " + screen.Name})
		}
	}

	for _, row := range refusalRegistry {
		if command := strings.TrimSpace(row.NextCommand); command != "" {
			refs = append(refs, guidanceCommandRef{Command: command, Source: "refusal " + row.ID})
		}
	}

	return refs
}

// guidanceCommandAllowlistPath is the repo-relative path of the committed,
// shrink-only allowlist of verbs a screen may legitimately advise with no
// menu wrapper behind them.
const guidanceCommandAllowlistPath = "cmd/testdata/guidance_command_allowlist.json"

// guidanceCommandAllowlistRow is one allowlisted verb and the written
// reason an owner does not need a menu command for it.
type guidanceCommandAllowlistRow struct {
	Verb   string `json:"verb"`
	Reason string `json:"reason"`
}

// loadGuidanceCommandAllowlist reads and parses the committed allowlist.
func loadGuidanceCommandAllowlist(t *testing.T) []guidanceCommandAllowlistRow {
	t.Helper()
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(root, guidanceCommandAllowlistPath))
	if err != nil {
		t.Fatalf("read %s: %v", guidanceCommandAllowlistPath, err)
	}
	var rows []guidanceCommandAllowlistRow
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal %s: %v", guidanceCommandAllowlistPath, err)
	}
	return rows
}

// guidanceCommandAllowlistCeiling is the recorded maximum set of allowlisted
// verbs -- the same shrink-only discipline journeyExpectedRedCeiling
// (cmd/journey_expected_red_test.go) applies to the expected-red register.
// This ceiling may only shrink: adding a menu command for an allowlisted
// verb removes it here; adding a NEW verb to the allowlist requires the
// same written-reason discipline as raising journeyExpectedRedCeiling, and
// TestGuidanceCommandAllowlistOnlyShrinks is what enforces it.
var guidanceCommandAllowlistCeiling = []string{
	"<command>",
}

// TestScreenGuidanceNamesCommandsTheOwnerCanRun fails naming any `aether
// <verb>` command an owner-facing screen advises that has no menu wrapper
// on both platforms, unless the verb is recorded on the shrink-only
// allowlist with a written reason.
func TestScreenGuidanceNamesCommandsTheOwnerCanRun(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	refs := screenAdvisedProgramCommands(t)
	if len(refs) == 0 {
		t.Fatal("screenAdvisedProgramCommands returned nothing -- the extractor is broken")
	}

	sourceKinds := map[string]bool{"status card": false, "voice screen": false, "refusal": false}
	for _, ref := range refs {
		for kind := range sourceKinds {
			if strings.HasPrefix(ref.Source, kind) {
				sourceKinds[kind] = true
			}
		}
	}
	for kind, found := range sourceKinds {
		if !found {
			t.Errorf("screenAdvisedProgramCommands contributed no command from source %q -- the extractor is broken", kind)
		}
	}

	allowlisted := map[string]bool{}
	for _, row := range loadGuidanceCommandAllowlist(t) {
		allowlisted[row.Verb] = true
	}

	type verbSeen struct {
		verb    string
		sources []string
	}
	seenOrder := []string{}
	seen := map[string]*verbSeen{}
	for _, ref := range refs {
		verb, ok := statusAdvisedCommandVerb(ref.Command)
		if !ok {
			continue
		}
		if allowlisted[verb] {
			continue
		}
		entry, ok := seen[verb]
		if !ok {
			entry = &verbSeen{verb: verb}
			seen[verb] = entry
			seenOrder = append(seenOrder, verb)
		}
		entry.sources = append(entry.sources, ref.Source)
	}

	for _, verb := range seenOrder {
		entry := seen[verb]
		for _, platform := range []string{
			filepath.Join(".claude", "commands", "ant"),
			filepath.Join(".opencode", "commands", "ant"),
		} {
			wrapper := filepath.Join(root, platform, entry.verb+".md")
			if _, statErr := os.Stat(wrapper); statErr != nil {
				t.Errorf("a screen advises `aether %s` (seen via: %s) but %s does not exist -- "+
					"either add the wrapper, or record the verb in %s with a written reason",
					entry.verb, strings.Join(entry.sources, "; "), filepath.Join(platform, entry.verb+".md"), guidanceCommandAllowlistPath)
			}
		}
	}
}

// TestGuidanceCommandAllowlistOnlyShrinks is the ratchet against the
// recorded ceiling: every verb in the committed allowlist must already be
// present in guidanceCommandAllowlistCeiling, naming any that is not -- the
// allowlist may never gain a verb silently. Mirrors
// TestExpectedRedRegisterOnlyShrinks (cmd/journey_expected_red_test.go)
// exactly, in the opposite domain.
func TestGuidanceCommandAllowlistOnlyShrinks(t *testing.T) {
	rows := loadGuidanceCommandAllowlist(t)
	allowed := make(map[string]bool, len(guidanceCommandAllowlistCeiling))
	for _, verb := range guidanceCommandAllowlistCeiling {
		allowed[verb] = true
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Reason) == "" {
			t.Errorf("allowlist row %q has no reason -- every row must say why an owner does not need a menu command for it", row.Verb)
		}
		if !allowed[row.Verb] {
			t.Errorf("verb %q is not in the recorded allowlist ceiling -- the allowlist may only "+
				"shrink toward zero as menu commands are added; adding a verb requires the same "+
				"written-reason discipline as raising guidanceCommandAllowlistCeiling", row.Verb)
		}
	}
}

// TestStatusWarningsSpeakPlainEnglish runs computeWarnings against a store
// seeded the same way screenAdvisedProgramCommands seeds it, and fails
// naming the warning and the word if any warning uses a word this
// repository invented without explaining it in the same sentence
// (untranslatedRepoWords, cmd/next_action_card_test.go).
func TestStatusWarningsSpeakPlainEnglish(t *testing.T) {
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	seedUnacknowledgedMiddenFailure(t, s)
	seedOpenFlagForGuidance(t, s)

	var state colony.ColonyState
	_ = s.LoadJSON("COLONY_STATE.json", &state)

	for _, warning := range computeWarnings(state, s) {
		if violations := untranslatedRepoWords(warning); len(violations) > 0 {
			t.Errorf("status warning uses a word this repository invented without explaining it:\n  %s\n\nwarning: %s",
				strings.Join(violations, "\n  "), warning)
		}
	}

	// Guard-on-the-guard: the helper genuinely reports a violation for a
	// planted warning naming an invented word with no explanation.
	planted := "5 unacknowledged failure(s). Run `aether midden-review` to inspect the midden."
	if violations := untranslatedRepoWords(planted); len(violations) == 0 {
		t.Fatal("untranslatedRepoWords reported no violation for a planted warning naming an unexplained invented word -- the guard itself is broken")
	}
}
