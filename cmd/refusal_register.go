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
	{
		ID:           "corrupted-colony-data",
		Pattern:      "json:",
		What:         "Aether's own project data file looks corrupted or was edited outside of Aether.",
		Why:          "Aether's data file is corrupted or was modified outside of Aether.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		ExtraSteps:   []string{"Check `.aether/data/COLONY_STATE.json` for syntax errors."},
	},
	{
		ID:           "criterion-artifact-is-a-directory",
		What:         "A phase criterion is bound to a folder instead of a file.",
		Why:          "Aether checks evidence one file at a time; it cannot check a whole folder, so a folder bound this way could never be satisfied by any build.",
		NextCommand:  "aether plan --refresh",
		ProtectsWork: false,
		ExtraSteps: []string{
			"Bind the specific files inside the folder instead of the folder itself.",
			"Or bind a verification check (for example a test or lint run) instead of an artifact path.",
		},
	},
	{
		ID:           "criterion-binding-unsatisfiable",
		What:         "One criterion in this phase is bound to evidence no build could ever satisfy.",
		Why:          "The bound artifact exists on disk as a folder, and Aether checks evidence file by file, never a whole folder.",
		NextCommand:  "aether decision-answer",
		ProtectsWork: false,
		ExtraSteps: []string{
			"Run `aether continue` on this phase to see the exact question and the exact command for this criterion.",
			"Answer it in your own words once you have confirmed the criterion is genuinely true.",
		},
	},
	{
		ID:           "failed-to-initialize-store",
		Pattern:      "failed to initialize store",
		What:         "Aether could not set up its own data storage.",
		Why:          "Aether could not set up its data storage. This usually means the data directory is inaccessible.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		ExtraSteps:   []string{"Check that `.aether/data/` exists and is writable."},
	},
	{
		ID:           "failed-to-load-colony-state",
		Pattern:      "failed to load colony state",
		What:         "Aether could not read its own project data file.",
		Why:          "Aether could not read the colony data file. This may be corrupted or was modified outside of Aether.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		ExtraSteps:   []string{"Check `.aether/data/COLONY_STATE.json` for syntax errors."},
	},
	{
		ID:           "invalid-charter-json",
		Pattern:      "invalid charter JSON",
		What:         "The charter text handed to `aether init` was not valid JSON.",
		Why:          "The charter passed to Aether is not valid JSON. The colony state file was not changed.",
		NextCommand:  `aether init "your goal"`,
		ProtectsWork: false,
		ExtraSteps:   []string{"If an assistant generated the command, ask it to compact the charter or escape quotes/newlines correctly."},
	},
	{
		ID:           "missing-required-flag",
		Pattern:      "flag --",
		What:         "This command needs more information to run.",
		Why:          "This command needs more information to run. Check the required flags and try again.",
		NextCommand:  "aether <command> --help",
		ProtectsWork: false,
	},
	{
		ID:           "no-colony-initialized",
		Pattern:      "no colony initialized",
		What:         "Aether has no project set up in this folder yet.",
		Why:          "Aether needs a colony to work with. A colony is a workspace for building toward a specific goal.",
		NextCommand:  `aether init "your goal"`,
		ProtectsWork: false,
		ExtraSteps:   []string{"Run `aether lay-eggs` first if this repo is brand new."},
	},
	{
		ID:           "permission-denied",
		Pattern:      "permission denied",
		What:         "Aether does not have permission to access a file or directory it needs.",
		Why:          "Aether does not have permission to access a file or directory.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		ExtraSteps:   []string{"Check file permissions. On macOS/Linux: `ls -la <path>` to inspect."},
	},
	{
		ID:           "verification-command-not-understood",
		What:         "A verification command line Aether found could not be understood.",
		Why:          "Aether recognises only a fixed set of build/type/lint/test command shapes; a line whose kind is labelled (for example \"- tests: ...\") but whose command doesn't match one of them is never silently dropped or reported as absent.",
		NextCommand:  "aether patrol",
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

// refusalRowForPattern is friendlyErrorForPattern's real lookup, kept here
// beside the table it reads. Rows are checked in ascending id order (the
// registry's own order, D-required by TestRefusalRegisterIsSortedAndUnique)
// -- but specificity, not table position, decides a tie: when more than one
// row's Pattern matches, the LONGEST pattern wins, so a general row like
// "json:" (corrupted-colony-data) can never shadow a more specific row like
// "invalid charter JSON" (invalid-charter-json) just because it happens to
// sort earlier alphabetically. Matching is case-insensitive.
func refusalRowForPattern(message string) (refusalRow, bool) {
	lower := strings.ToLower(message)
	best, found := refusalRow{}, false
	for _, row := range refusalRegistry {
		if row.Pattern == "" {
			continue
		}
		if !strings.Contains(lower, strings.ToLower(row.Pattern)) {
			continue
		}
		if !found || len(row.Pattern) > len(best.Pattern) {
			best, found = row, true
		}
	}
	return best, found
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
