package cmd

// 207-03-PLAN.md (UED-09, owner ruling D-01, .planning/phases/207-messy-practice-project-gate/207-CONTEXT.md):
// research found five of the six 2026-09-21 blockers have a landed fix, proven
// by reverting each fix in turn and watching the journey fail at that step
// (207-05). The sixth -- the project's own status screen advising the owner
// to run `aether midden-review` when failures are waiting, a command with no
// menu wrapper on any platform -- has no landed fix; that is Phase 208's job
// (UED-13). This file builds the check for that sixth case now: it runs for
// real against the runtime's own status guidance code, it is honestly red
// today, and it is recorded as red in a committed register naming Phase 208
// as what closes it. It is never a silent pass, and this phase never claims
// six blockers are proven -- five are proven, the sixth is a standing,
// named, expected-red case until Phase 208 lands its fix.
//
// Plain functions only -- this file registers no cobra command and adds no
// new subcommand to the binary (milestone rule: no new features, no new
// strict rules).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// journeyExpectedRedPath is the repo-relative path of the committed register
// of standing expected-red journey cases.
const journeyExpectedRedPath = "cmd/testdata/journey/expected-red.json"

// journeyExpectedRedSchemaVersion is the schema_version constant the
// register file carries.
const journeyExpectedRedSchemaVersion = "journey-expected-red/v1"

// journeyExpectedRedRepoRootOverride lets tests redirect register reads to
// an isolated temporary directory instead of the real repository checkout,
// mirroring evalGateRepoRootOverride's precedent exactly.
var journeyExpectedRedRepoRootOverride string

// journeyExpectedRedRepoRoot resolves the directory journeyExpectedRedPath
// is relative to: journeyExpectedRedRepoRootOverride when a test has set it,
// otherwise the real Aether module root -- the same resolution
// evalGateRepoRoot and journeyTrapsRepoRoot use.
func journeyExpectedRedRepoRoot() (string, error) {
	if journeyExpectedRedRepoRootOverride != "" {
		return journeyExpectedRedRepoRootOverride, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	root := findAetherModuleRoot(cwd)
	if root == "" {
		return "", fmt.Errorf("locate Aether module root from %s", cwd)
	}
	return root, nil
}

// journeyExpectedRedCase is one standing, named, honestly-red case: a gap
// the journey's own checks prove exists today, recorded with exactly what
// closes it so it can never be mistaken for a silent pass.
type journeyExpectedRedCase struct {
	ID       string `json:"id"`
	Blocker  string `json:"blocker"`
	Detail   string `json:"detail"`
	ClosedBy string `json:"closed_by"`
	Proof    string `json:"proof"`
}

// journeyExpectedRedFile is the on-disk container at journeyExpectedRedPath.
type journeyExpectedRedFile struct {
	SchemaVersion string                   `json:"schema_version"`
	Cases         []journeyExpectedRedCase `json:"cases"`
}

// loadJourneyExpectedRed reads and parses the committed expected-red
// register.
func loadJourneyExpectedRed() (journeyExpectedRedFile, error) {
	root, err := journeyExpectedRedRepoRoot()
	if err != nil {
		return journeyExpectedRedFile{}, err
	}
	path := filepath.Join(root, journeyExpectedRedPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return journeyExpectedRedFile{}, fmt.Errorf("read journey expected-red register %s: %w", path, err)
	}
	var file journeyExpectedRedFile
	if err := json.Unmarshal(data, &file); err != nil {
		return journeyExpectedRedFile{}, fmt.Errorf("unmarshal journey expected-red register %s: %w", path, err)
	}
	return file, nil
}

// statusAdvisedCommandRe extracts a runtime CLI invocation from a
// backtick-quoted span inside a prose warning string, e.g.
// "Run `aether midden-review` to inspect." yields "aether midden-review".
// Status warnings are the only place a command name is ever embedded in
// free prose rather than carried on a struct field; this is the one place
// that prose is parsed back out, never re-typed.
var statusAdvisedCommandRe = regexp.MustCompile("`(aether [^`]+)`")

// statusAdvisedCommandVerb returns the runtime verb (the first token after
// `aether`) a status-advised command names, and whether the command is an
// `aether` command at all. A command this project's status screen names
// that is not an `aether` invocation (there are none today, but the check
// must not assume that stays true) is never treated as a gap.
func statusAdvisedCommandVerb(command string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) < 2 || fields[0] != "aether" {
		return "", false
	}
	return fields[1], true
}

// statusGuidanceAdvisedCommands returns every command the project's own
// status screen can put in front of the owner, derived by calling the real
// guidance code: the Command and AlternativeCommand of every guided action
// loadGuidedActions returns (in that order, per action), plus every
// `aether ...` command named inside the warnings the same store produces
// (in the order the warnings are appended). This never parses
// cmd/status.go's source and never sorts its result -- the order is exactly
// what an owner running `aether status` would actually be shown.
func statusGuidanceAdvisedCommands(s *storage.Store, root string) ([]string, error) {
	var commands []string

	for _, action := range loadGuidedActions(s, root) {
		if cmd := strings.TrimSpace(action.Command); cmd != "" {
			commands = append(commands, cmd)
		}
		if cmd := strings.TrimSpace(action.AlternativeCommand); cmd != "" {
			commands = append(commands, cmd)
		}
	}

	var state colony.ColonyState
	if s != nil {
		// A missing or unparseable COLONY_STATE.json leaves state at its
		// zero value -- the same shape computeWarnings already tolerates
		// for an un-onboarded or freshly seeded store.
		_ = s.LoadJSON("COLONY_STATE.json", &state)
	}
	for _, warning := range computeWarnings(state, s) {
		for _, match := range statusAdvisedCommandRe.FindAllStringSubmatch(warning, -1) {
			commands = append(commands, match[1])
		}
	}

	return commands, nil
}

// statusGuidanceCommandsWithoutMenuWrapper returns, in the order the status
// card produces them, the runtime verbs among the advised commands that have
// no menu wrapper file -- using the same path convention
// TestDeclaredAliasesHaveWrappersOnEveryPlatform already uses for the Claude
// surface (cmd/canonical_alias_test.go). repoRoot is resolved separately
// from root (the guided-action workspace root) so tests can point the
// wrapper-existence lookup at an isolated fixture directory without
// disturbing what store/state the guidance code itself reads.
func statusGuidanceCommandsWithoutMenuWrapper(s *storage.Store, root, repoRoot string) ([]string, error) {
	advised, err := statusGuidanceAdvisedCommands(s, root)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(advised))
	var missing []string
	for _, command := range advised {
		verb, ok := statusAdvisedCommandVerb(command)
		if !ok {
			// Not an `aether` command at all -- not a gap.
			continue
		}
		if seen[verb] {
			continue
		}
		seen[verb] = true

		wrapperPath := filepath.Join(repoRoot, ".claude", "commands", "ant", verb+".md")
		if _, statErr := os.Stat(wrapperPath); statErr != nil {
			missing = append(missing, verb)
		}
	}
	return missing, nil
}
