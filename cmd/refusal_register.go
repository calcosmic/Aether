package cmd

import (
	"fmt"
	"strings"
)

// refusalRow is one entry in the checked-in refusal table. ID is stable,
// kebab-case, and never reused once shipped -- other code (the refusal log,
// a report bundle, a future dashboard) may reference it by id. Pattern is an
// optional legacy substring match, used only by the pattern-matching path
// friendlyErrorForPattern folds in on top of this same table (Task 2) --
// most rows never need one.
type refusalRow struct {
	ID           string
	Pattern      string
	What         string
	Why          string
	NextCommand  string
	ProtectsWork bool
	ExtraSteps   []string
}

// refusalRegistry is the ONE refusal table in the program. It is checked in,
// id-sorted, by hand -- not sorted at runtime -- so the committed order
// itself is what TestRefusalRegisterIsSortedAndUnique proves, not a sort
// call that would make the test pass no matter what order someone commits.
//
// TestFriendlyErrorsReadTheOneRefusalTable (Task 2) fails naming any file
// that plants a second hinted-error table literal in cmd/*.go -- this is the
// only one that is ever allowed to exist.
var refusalRegistry = []refusalRow{
	{
		ID:           "colonize-finalize-invalid-timestamp",
		What:         "The survey result's generation time is not in a format Aether understands.",
		Why:          "Aether stamps this field itself in a standard timestamp format; a different value usually means it was edited by hand.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: false,
	},
	{
		ID:           "colonize-finalize-missing-timestamp",
		What:         "The survey result you sent is missing the time it was generated, so Aether cannot tell whether it is still fresh.",
		Why:          "Aether checks that a survey result came from a recent `aether colonize --plan-only` run before publishing it, and cannot do that without a generation time.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: false,
	},
	{
		ID:           "colonize-finalize-timestamp-in-future",
		What:         "The survey result claims to have been generated in the future.",
		Why:          "Aether stamps this field itself at the moment the survey plan is written; a future time usually means it was edited by hand or copied from a different run.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: false,
	},
}

// refusalForID looks up a registered row by id. Two calls with the same id
// return the same row.
func refusalForID(id string) (refusalRow, bool) {
	for _, row := range refusalRegistry {
		if row.ID == id {
			return row, true
		}
	}
	return refusalRow{}, false
}

// refusalRegistryProblems is the one check both TestEveryRefusalRowNamesANextCommand
// and TestRefusalRegisterIsSortedAndUnique run against the real refusalRegistry,
// and TestRefusalCheckCanFail runs against an isolated, fabricated slice --
// proof the check itself can fail rather than passing no matter what it is
// given. It reports, by row id: a missing next command, a duplicate id, an
// id out of ascending order, and -- if the whole slice is empty -- that the
// enumeration itself is broken rather than vacuously valid.
func refusalRegistryProblems(rows []refusalRow) []string {
	var problems []string
	if len(rows) == 0 {
		return []string{"refusal register enumerated zero rows -- the enumeration is broken, not vacuously valid"}
	}
	seen := map[string]bool{}
	for i, row := range rows {
		if strings.TrimSpace(row.NextCommand) == "" {
			problems = append(problems, fmt.Sprintf("refusal row %q has no next_command", row.ID))
		}
		if seen[row.ID] {
			problems = append(problems, fmt.Sprintf("duplicate refusal id %q", row.ID))
		}
		seen[row.ID] = true
		if i > 0 && rows[i-1].ID >= row.ID {
			problems = append(problems, fmt.Sprintf("refusal row %q is out of ascending id order (follows %q)", row.ID, rows[i-1].ID))
		}
	}
	return problems
}
