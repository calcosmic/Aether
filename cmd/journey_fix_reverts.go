package cmd

// 207-05-PLAN.md (UED-09, owner ruling D-01,
// .planning/phases/207-messy-practice-project-gate/207-CONTEXT.md): the
// journey (207-01..04) is only a test suite worth running once it is shown
// to catch something. Five of the six 2026-09-21 blockers have a landed
// fix; this file declares the committed table that names each of those
// five fixes -- the blocker in the owner's own words, the commit, the file,
// the symbol, and an exact, minimal, single-occurrence text swap that
// restores the pre-fix behaviour -- and the journey step that fix protects.
// scripts/prove-journey-catches-the-2026-09-21-fixes.sh applies each swap
// in a throwaway working copy, reruns the matching journey step, and
// asserts it fails. The sixth blocker (a status card advising a menu
// command that does not exist) has no landed fix; it stays the standing
// expected-red case cmd/journey_expected_red.go / expected-red.json
// declares. This file must never grow a sixth entry without a real,
// landed commit to point at -- cmd/journey_fix_reverts_test.go enforces
// exactly five.
//
// Plain functions only -- this file registers no cobra command and adds no
// new subcommand to the binary (milestone rule: no new features, no new
// strict rules).

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// journeyFixRevertsPath is the repo-relative path of the committed table of
// declared fix reverts.
const journeyFixRevertsPath = "cmd/testdata/journey/fix-reverts.json"

// journeyFixRevertsSchemaVersion is the schema_version constant the table
// file carries.
const journeyFixRevertsSchemaVersion = "journey-fix-reverts/v1"

// journeyFixRevertsRepoRootOverride lets tests redirect table reads to an
// isolated temporary directory instead of the real repository checkout,
// mirroring evalGateRepoRootOverride's and journeyExpectedRedRepoRootOverride's
// precedent exactly.
var journeyFixRevertsRepoRootOverride string

// journeyFixRevertsRepoRoot resolves the directory journeyFixRevertsPath is
// relative to: journeyFixRevertsRepoRootOverride when a test has set it,
// otherwise the real Aether module root -- the same resolution
// evalGateRepoRoot, journeyTrapsRepoRoot and journeyExpectedRedRepoRoot use.
func journeyFixRevertsRepoRoot() (string, error) {
	if journeyFixRevertsRepoRootOverride != "" {
		return journeyFixRevertsRepoRootOverride, nil
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

// journeyFixRevertMutation is one exact, minimal, single-occurrence literal
// text swap that restores a fix's pre-fix behaviour. It is never a whole-
// commit undo -- each of these commits also carries tests and documents
// whose removal would make the journey fail for the wrong reason.
type journeyFixRevertMutation struct {
	Find    string `json:"find"`
	Replace string `json:"replace"`
}

// journeyFixRevert is one declared, landed 2026-09-21 fix: the blocker it
// fixed (in the owner's own decision-record words), the commit that landed
// it, the file and symbol it touched, the journey step that exercises it,
// the mutation that reverts it, and what the journey should fail on once
// reverted.
type journeyFixRevert struct {
	ID              string                   `json:"id"`
	Blocker         string                   `json:"blocker"`
	Commit          string                   `json:"commit"`
	File            string                   `json:"file"`
	Symbol          string                   `json:"symbol"`
	JourneyStep     string                   `json:"journey_step"`
	Mutation        journeyFixRevertMutation `json:"mutation"`
	ExpectedFailure string                   `json:"expected_failure"`
	Note            string                   `json:"note,omitempty"`
}

// journeyFixRevertFile is the on-disk container at journeyFixRevertsPath.
type journeyFixRevertFile struct {
	SchemaVersion string             `json:"schema_version"`
	Note          string             `json:"note"`
	Reverts       []journeyFixRevert `json:"reverts"`
}

// loadJourneyFixReverts reads and parses the committed fix-revert table.
func loadJourneyFixReverts() (journeyFixRevertFile, error) {
	root, err := journeyFixRevertsRepoRoot()
	if err != nil {
		return journeyFixRevertFile{}, err
	}
	path := filepath.Join(root, journeyFixRevertsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return journeyFixRevertFile{}, fmt.Errorf("read journey fix-revert table %s: %w", path, err)
	}
	var file journeyFixRevertFile
	if err := json.Unmarshal(data, &file); err != nil {
		return journeyFixRevertFile{}, fmt.Errorf("unmarshal journey fix-revert table %s: %w", path, err)
	}
	return file, nil
}

// journeyFixRevertIDs returns the declared revert ids, in declared file
// order -- never sorted. Results (Plan 05 Task 2's script, and any Go
// caller) are reported one row per fix in this order, so two runs over the
// same table name the same first failure.
func journeyFixRevertIDs() []string {
	file, err := loadJourneyFixReverts()
	if err != nil {
		return nil
	}
	ids := make([]string, 0, len(file.Reverts))
	for _, r := range file.Reverts {
		ids = append(ids, r.ID)
	}
	return ids
}
