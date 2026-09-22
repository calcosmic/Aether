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
//
// The classification rule (208-06-PLAN.md Task 1), written down here once
// and never re-derived per row: ProtectsWork is true when carrying on
// (instead of refusing) would (a) overwrite or delete something the owner
// made and cannot get back, (b) record a completion, verification or
// advancement that is not true, or (c) spend money or contact something
// outside the program. Everything else is false. A row whose ProtectsWork
// is true must always carry Disposition "stop" -- TestEveryRowCarriesAClassificationReason
// fails any row that pairs ProtectsWork=true with Disposition="warn".
type refusalRow struct {
	ID          string
	Pattern     string
	What        string
	Why         string
	NextCommand string
	// ProtectsWork is the verdict the doc comment above argues: would
	// carrying on past this refusal risk losing the owner's work, record a
	// false completion, or spend money / reach outside the program.
	ProtectsWork bool
	// Disposition is "stop" (the site returns the refusal as an error and
	// nothing after it runs) or "warn" (warnAndCarryOn, cmd/refusal.go,
	// renders the same block headed as something Aether noticed rather than
	// something that stopped, and the code after it keeps running). Every
	// row not yet converted by 208-06 Task 2 stays "stop", matching what the
	// site's real code still does.
	Disposition string
	// Reason is one line saying which of ProtectsWork's three grounds --
	// (a), (b) or (c) from the doc comment above -- applies, or plainly
	// that none does.
	Reason     string
	ExtraSteps []string
}

// refusalDeclaredFiles is the declared scope for the two ratchets in
// cmd/refusal_enumerate_test.go: the ten lifecycle files behind the
// fourteen steps an owner actually walks (cmd/journey.go's own
// journeyStepMenuCommandMap names the steps; these are the files those
// steps' commands run through). The rest of the program is out of scope for
// this phase -- cmd/testdata/refusals/untyped-floor.json is the honest,
// shrink-only record of what inside this declared scope is not yet a typed
// refuse(...) call.
var refusalDeclaredFiles = []string{
	"codex_colonize_finalize.go",
	"codex_plan_finalize.go",
	"codex_build_finalize.go",
	"codex_continue_finalize.go",
	"codex_continue.go",
	"criterion_evidence.go",
	"entomb_cmd.go",
	"init_cmd.go",
	"session_flow_cmds.go",
	"codex_workflow_cmds.go",
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
		Disposition:  "stop",
		Reason:       "Refusing only blocks publishing a survey result; nothing the owner made is deleted or overwritten, and re-surveying records nothing false.",
	},
	{
		ID:           "colonize-finalize-missing-timestamp",
		What:         "The survey result you sent is missing the time it was generated, so Aether cannot tell whether it is still fresh.",
		Why:          "Aether checks that a survey result came from a recent `aether colonize --plan-only` run before publishing it, and cannot do that without a generation time.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing only blocks publishing a survey result; Aether already tries recovering the timestamp from its own receipt first (codex_colonize_finalize.go), so this only fires when it genuinely cannot tell freshness, and nothing is deleted or falsely recorded either way.",
	},
	{
		ID:           "colonize-finalize-timestamp-in-future",
		What:         "The survey result claims to have been generated in the future.",
		Why:          "Aether stamps this field itself at the moment the survey plan is written; a future time usually means it was edited by hand or copied from a different run.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing only blocks publishing a survey result whose own freshness claim cannot be trusted; nothing the owner made is deleted or overwritten.",
	},
	{
		ID:           "corrupted-colony-data",
		Pattern:      "json:",
		What:         "Aether's own project data file looks corrupted or was edited outside of Aether.",
		Why:          "Aether's data file is corrupted or was modified outside of Aether.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing here happens before any write to the data file; nothing already on disk is changed or deleted, and nothing is recorded as verified using a read that could not be trusted.",
		ExtraSteps:   []string{"Check `.aether/data/COLONY_STATE.json` for syntax errors."},
	},
	{
		ID:           "criterion-artifact-is-a-directory",
		What:         "A phase criterion is bound to a folder instead of a file.",
		Why:          "Aether checks evidence one file at a time; it cannot check a whole folder, so a folder bound this way could never be satisfied by any build.",
		NextCommand:  "aether plan --refresh",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing here happens before any worker is dispatched; nothing has been built or changed yet, and letting the phase proceed would only let it advance toward a criterion no build could ever satisfy.",
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
		Disposition:  "stop",
		Reason:       "This path routes the phase to owner-confirmation rather than silently recording it advanced (evaluatePhaseCriterionEvidence); it is a scoped stop that still requires an explicit owner answer, not a default that would record a completion that is not true.",
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
		Disposition:  "stop",
		Reason:       "Aether cannot set up its own storage here; refusing happens before any data is written, so nothing on disk is changed, deleted or falsely recorded.",
		ExtraSteps:   []string{"Check that `.aether/data/` exists and is writable."},
	},
	{
		ID:           "failed-to-load-colony-state",
		Pattern:      "failed to load colony state",
		What:         "Aether could not read its own project data file.",
		Why:          "Aether could not read the colony data file. This may be corrupted or was modified outside of Aether.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing here happens before any write to the data file; nothing already on disk is changed or deleted, and nothing is recorded as verified using a read that could not be trusted.",
		ExtraSteps:   []string{"Check `.aether/data/COLONY_STATE.json` for syntax errors."},
	},
	{
		ID:           "invalid-charter-json",
		Pattern:      "invalid charter JSON",
		What:         "The charter text handed to `aether init` was not valid JSON.",
		Why:          "The charter passed to Aether is not valid JSON. The colony state file was not changed.",
		NextCommand:  `aether init "your goal"`,
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "The colony state file is left unchanged when this fires (the Why line says so); refusing an unparsable charter loses nothing the owner made.",
		ExtraSteps:   []string{"If an assistant generated the command, ask it to compact the charter or escape quotes/newlines correctly."},
	},
	{
		ID:           "missing-required-flag",
		Pattern:      "flag --",
		What:         "This command needs more information to run.",
		Why:          "This command needs more information to run. Check the required flags and try again.",
		NextCommand:  "aether <command> --help",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "A command missing a required flag has not done anything yet; refusing before running loses nothing.",
	},
	{
		ID:           "no-colony-initialized",
		Pattern:      "no colony initialized",
		What:         "Aether has no project set up in this folder yet.",
		Why:          "Aether needs a colony to work with. A colony is a workspace for building toward a specific goal.",
		NextCommand:  `aether init "your goal"`,
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "There is no project to lose here -- this fires precisely when no colony exists yet.",
		ExtraSteps:   []string{"Run `aether lay-eggs` first if this repo is brand new."},
	},
	{
		ID:           "permission-denied",
		Pattern:      "permission denied",
		What:         "Aether does not have permission to access a file or directory it needs.",
		Why:          "Aether does not have permission to access a file or directory.",
		NextCommand:  "aether patrol",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Refusing here happens before Aether touches the inaccessible file or directory; nothing is deleted, overwritten or falsely recorded.",
		ExtraSteps:   []string{"Check file permissions. On macOS/Linux: `ls -la <path>` to inspect."},
	},
	{
		ID:           "verification-command-not-understood",
		What:         "A verification command line Aether found could not be understood.",
		Why:          "Aether recognises only a fixed set of build/type/lint/test command shapes; a line whose kind is labelled (for example \"- tests: ...\") but whose command doesn't match one of them is never silently dropped or reported as absent.",
		NextCommand:  "aether patrol",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Continuing past an unreadable verification line would record a phase as checked when one of its own declared checks was never actually run -- ground (b), a false verification.",
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
