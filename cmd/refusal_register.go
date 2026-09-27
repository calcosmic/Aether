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
		ID:           "build-dispatch-manifest-wrong-source",
		What:         "The build completion packet's dispatch manifest did not come from a real `aether build --plan-only` run.",
		Why:          "Aether cross-checks the dispatch manifest's own origin before trusting it to finalize a build; a manifest built any other way could dispatch or credit work that was never really planned.",
		NextCommand:  "aether build --plan-only",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would record a build as dispatched and later completed from a manifest that was not genuinely produced by Aether's own planning step -- ground (b), a false record.",
	},
	{
		ID:           "charter-field-too-long",
		What:         "One field of the project goal (charter) you gave `aether init` is too long.",
		Why:          "Aether stores this field as-is; a field this long usually means something else was pasted into it by mistake.",
		NextCommand:  `aether init "your goal"`,
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Nothing has been written yet when this fires; there is no safe way to guess how to shorten the owner's own words, so this stays a stop rather than a silent truncation.",
	},
	{
		ID:           "colonize-existing-survey-found",
		What:         "A territory survey already exists for this project.",
		Why:          "Aether will not silently replace an existing survey of the codebase with a new one -- the existing survey may still be the one later steps expect.",
		NextCommand:  "aether colonize --force-resurvey",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would overwrite an existing survey the owner (or an earlier step) already produced and may still be relying on -- ground (a).",
	},
	{
		ID:           "colonize-finalize-existing-survey-found",
		What:         "A territory survey already exists for this project.",
		Why:          "Aether will not silently replace an existing survey of the codebase with a new one -- the existing survey may still be the one later steps expect.",
		NextCommand:  "aether colonize --plan-only --force-resurvey",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would overwrite an existing survey the owner (or an earlier step) already produced and may still be relying on -- ground (a).",
	},
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
		ID:           "continue-on-paused-project",
		What:         "This project is paused, so it cannot be checked or signed off yet.",
		Why:          "A pause saves a snapshot to resume from. Checking a paused project used to write new results on top of that snapshot, which made the pause impossible to resume.",
		NextCommand:  "aether resume",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Refusing before any check runs keeps the saved pause valid; `aether resume` restores the project, then the check can run.",
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
		ID:   "criterion-binding-unsatisfiable",
		What: "One criterion in this phase is bound to evidence no build could ever satisfy.",
		Why:  "The bound artifact exists on disk as a folder, and Aether checks evidence file by file, never a whole folder.",
		// aether decision-answer has no /ant-... menu wrapper of its own: it
		// is only ever run with the exact, pre-filled --question/--phase
		// arguments ownerConfirmationCommand (cmd/criterion_owner_confirmation.go)
		// composes at the moment the criterion is evaluated -- no static
		// menu template could carry those. `aether continue` is the real
		// owner-typable route: it is what runs that evaluation and surfaces
		// the exact decision-answer invocation, matching the ExtraSteps
		// below (TestScreenGuidanceNamesCommandsTheOwnerCanRun).
		NextCommand:  "aether continue",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "This path routes the phase to owner-confirmation rather than silently recording it advanced (evaluatePhaseCriterionEvidence); it is a scoped stop that still requires an explicit owner answer, not a default that would record a completion that is not true.",
		ExtraSteps: []string{
			"Run `aether continue` on this phase to see the exact question and the exact command for this criterion.",
			"Answer it in your own words once you have confirmed the criterion is genuinely true.",
		},
	},
	{
		ID:           "entomb-manifest-digest-mismatch",
		What:         "The published archive manifest does not match the archive Aether actually wrote.",
		Why:          "Aether recomputes the manifest's own digest from the published bytes and compares it before trusting the archive is intact -- a mismatch here means the archive is not what its own manifest claims.",
		NextCommand:  "aether report",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would record this archived project as verified when its own manifest digest does not check out -- ground (b); this is an internal integrity check, not something the owner did wrong, so the next step is to report it.",
	},
	{
		ID:           "entomb-manifest-receipt-mismatch",
		What:         "The published archive manifest does not cross-reference the transaction that actually sealed it.",
		Why:          "Aether checks that a published archive's manifest points back at the exact durable transaction receipt that produced it -- a mismatch means the archive and the record of how it was made disagree.",
		NextCommand:  "aether report",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would record this archive as verified while its own manifest and the transaction that sealed it disagree -- ground (b); an internal integrity check, so the next step is to report it.",
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
		ID:           "invalid-timeout-value",
		What:         "A timeout you gave Aether was zero or negative.",
		Why:          "A timeout of zero or less has no sensible meaning here, so Aether falls back to its own built-in default instead of guessing what you meant.",
		NextCommand:  "aether status",
		ProtectsWork: false,
		Disposition:  "warn",
		Reason:       "Aether's own built-in timeout is a safe, defined fallback; carrying on with it risks nothing, so this warns and continues rather than stopping the command entirely.",
	},
	{
		ID:           "missing-required-flag",
		Pattern:      "flag --",
		What:         "This command needs more information to run.",
		Why:          "This command needs more information to run. Check the required flags and try again.",
		NextCommand:  "aether status",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "A command missing a required flag has not done anything yet; refusing before running loses nothing.",
	},
	{
		ID:           "no-active-phase-to-continue",
		What:         "There is no phase currently being built to check or continue.",
		Why:          "`aether continue` checks the work done on the phase `aether build` most recently started; there is no in-progress phase to check right now.",
		NextCommand:  "aether status",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Nothing has been checked or advanced when this fires; refusing loses nothing, and `aether status` tells the owner which phase (if any) to build next.",
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
		ID:           "no-project-plan",
		What:         "This project has no plan yet.",
		Why:          "Aether needs a plan -- phases with tasks -- before it can build or check anything.",
		NextCommand:  "aether plan",
		ProtectsWork: false,
		Disposition:  "stop",
		Reason:       "Nothing has been built or advanced when this fires; refusing loses nothing, and planning is exactly the missing step.",
	},
	{
		ID:           "pause-colony-state-not-runnable",
		What:         "Aether's own project state file is not in a state it can safely pause from.",
		Why:          "Aether checks that its own project state is genuinely runnable before recording a pause; this state is not.",
		NextCommand:  "aether status",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would record a pause as successful over a state Aether itself cannot confirm is runnable -- ground (b).",
	},
	{
		ID:           "pause-colony-state-unconfirmed",
		What:         "Aether could not confirm its own project state is trustworthy enough to pause.",
		Why:          "Pausing records a snapshot of the project's current state; when Aether cannot even confirm that state is genuine, it will not guess.",
		NextCommand:  "aether status",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would pause from state Aether cannot itself confirm, risking a pause that is recorded as successful when the underlying state was never verified -- ground (b).",
	},
	{
		ID:           "pause-handoff-missing-from-state",
		What:         "Aether's own project state does not reference a pause handoff, so there is nothing recorded to resume from.",
		Why:          "Resuming reads the handoff the last `aether pause` recorded; without that reference, Aether has no honest basis to reconstruct what was paused.",
		NextCommand:  "aether status",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would resume from nothing and risk recording the project as picked back up when there is no real basis for that -- ground (b).",
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
		ID:           "plan-empty-phase-list",
		What:         "The plan a planning worker produced has no phases in it.",
		Why:          "A plan with zero phases cannot be built or checked -- Aether refuses to accept it as the project's plan.",
		NextCommand:  "aether plan --refresh",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would record an empty plan as accepted, letting later steps report a project as planned when there is nothing in it to build -- ground (b).",
	},
	{
		ID:           "plan-missing-success-criteria",
		What:         "A phase in this plan has no success criteria.",
		Why:          "Every phase needs explicit, checkable success criteria; without them, Aether has no honest way to say the phase is ever actually done.",
		NextCommand:  "aether plan --refresh",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would let a phase advance with nothing to verify it against, so a later check could only ever be recording a completion that was never actually proven -- ground (b).",
	},
	{
		ID:           "resume-handoff-reference-missing",
		What:         "Aether's own project state has no handoff reference to resume from.",
		Why:          "A normal `aether pause` always leaves a handoff reference behind for `aether resume` to read; without one, there is nothing recorded to resume.",
		NextCommand:  "aether status",
		ProtectsWork: true,
		Disposition:  "stop",
		Reason:       "Carrying on would resume from nothing and risk recording the project as picked back up when there is no real basis for that -- ground (b).",
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
// given. It reports, by row id: a missing next command, a next command that
// is not a real, runnable command (still carrying a template placeholder
// like `<command>` -- CR-01, 208-REVIEW.md: a placeholder rendered verbatim
// is a dead end, not a way out), a duplicate id, an id out of ascending
// order, and -- if the whole slice is empty -- that the enumeration itself
// is broken rather than vacuously valid.
func refusalRegistryProblems(rows []refusalRow) []string {
	var problems []string
	if len(rows) == 0 {
		return []string{"refusal register enumerated zero rows -- the enumeration is broken, not vacuously valid"}
	}
	seen := map[string]bool{}
	for i, row := range rows {
		if strings.TrimSpace(row.NextCommand) == "" {
			problems = append(problems, fmt.Sprintf("refusal row %q has no next_command", row.ID))
		} else if strings.ContainsAny(row.NextCommand, "<>") {
			problems = append(problems, fmt.Sprintf("refusal row %q's next_command %q still carries an unsubstituted template placeholder", row.ID, row.NextCommand))
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
