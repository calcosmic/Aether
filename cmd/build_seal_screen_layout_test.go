package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The build result screen and the seal screen get the layout the owner chose
// on 2026-09-27 for the check screen (cmd/check_screen_layout_test.go): a
// verdict box first, then short sections under heavy headers, long lists
// capped with "… N more", and clutter kept in the JSON result instead of on
// the default screen. These tests drive the real renderers the owner reaches
// -- the wrapper's closeout (`aether ceremony closeout`), the direct build
// lane, and `aether seal` -- over fixtures shaped the way the runtime shapes
// its results (completion packets read through closeoutCompletionDetails).

// buildScreenVariant names one build outcome a fixture renders.
type buildScreenVariant string

const (
	buildScreenOneHelper buildScreenVariant = "one helper, every task done"
	buildScreenRealistic buildScreenVariant = "four helpers, every task done"
	buildScreenPartial   buildScreenVariant = "three of four tasks done"
	buildScreenFailed    buildScreenVariant = "one helper failed"
)

// buildScreenStateFixture saves the project the build ran in: phase 2 of 3,
// four tasks, all done except where the variant leaves one open, and five
// pending steering suggestions (the "generic suggestion" clutter the default
// screen no longer lists in full).
func buildScreenStateFixture(t *testing.T, variant buildScreenVariant) colony.ColonyState {
	t.Helper()
	goal := "Add a French deck to the flashcard app"
	ids := []string{"2.1", "2.2", "2.3", "2.4"}
	goals := []string{"Create the French Basics deck", "Limit it to 5 new cards a day", "Show the deck in the menu", "Save progress between sessions"}
	tasks := make([]colony.Task, 0, len(ids))
	for i := range ids {
		id := ids[i]
		status := colony.TaskCompleted
		if variant == buildScreenPartial && i == 3 {
			status = colony.TaskPending
		}
		if variant == buildScreenFailed && i >= 2 {
			status = colony.TaskPending
		}
		tasks = append(tasks, colony.Task{ID: &id, Goal: goals[i], Status: status})
	}
	pending := []colony.PendingSuggestion{}
	for i := 1; i <= 5; i++ {
		pending = append(pending, colony.PendingSuggestion{
			ID:          fmt.Sprintf("sig_%d", i),
			Type:        "FEEDBACK",
			Content:     fmt.Sprintf("generic suggestion number %d", i),
			Reason:      "pattern noticed during the build",
			ContentHash: fmt.Sprintf("sha256:%d", i),
		})
	}
	state := colony.ColonyState{
		Version:            "3.0",
		Goal:               &goal,
		State:              colony.StateBUILT,
		CurrentPhase:       2,
		Milestone:          "Open Chambers",
		PendingSuggestions: &pending,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Import old cards", Status: colony.PhaseCompleted},
			{ID: 2, Name: "French Basics deck", Status: colony.PhaseInProgress, Tasks: tasks},
			{ID: 3, Name: "Daily reminders", Status: colony.PhaseReady},
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}
	return state
}

// buildScreenCompletionFixture writes the completion packet a wrapper hands
// `aether ceremony closeout --workflow build`: the manifest plus one terminal
// result per helper, with the files each one touched (including the
// runtime's own .aether/data bookkeeping files a real worker result lists).
func buildScreenCompletionFixture(t *testing.T, variant buildScreenVariant) string {
	t.Helper()
	manifest := map[string]interface{}{
		"phase":      2,
		"phase_name": "French Basics deck",
		"dispatches": []map[string]interface{}{
			{"name": "Forge-11", "caste": "builder", "task_id": "2.1", "task": "Create the deck", "execution_wave": 1},
		},
	}
	var workers []map[string]interface{}
	switch variant {
	case buildScreenOneHelper:
		workers = []map[string]interface{}{{
			"name": "Forge-11", "caste": "builder", "status": "completed", "task_id": "2.1",
			"summary":        "Created the French Basics deck",
			"files_modified": []string{"app/decks/french.json", "app/menu.ts"},
			"outputs":        []string{".aether/data/build/phase-2/worker-Forge-11.json"},
			"tool_count":     18,
		}}
	default:
		workers = []map[string]interface{}{
			{
				"name": "Forge-11", "caste": "builder", "status": "completed", "task_id": "2.1",
				"summary":        "Created the French Basics deck",
				"files_modified": []string{"app/decks/french.json", "app/decks/index.ts", "app/decks/loader.ts"},
				"outputs":        []string{".aether/data/build/phase-2/worker-Forge-11.json"},
				"tool_count":     18,
			},
			{
				"name": "Mason-41", "caste": "builder", "status": "completed", "task_id": "2.2",
				"summary":        "Capped new cards at five a day",
				"files_modified": []string{"app/schedule/limits.ts", "app/schedule/daily.ts", "app/schedule/daily.test.ts"},
				"outputs":        []string{".aether/data/last-build-claims.json"},
				"tool_count":     11,
			},
			{
				"name": "Brick-79", "caste": "builder", "status": "completed", "task_id": "2.3",
				"summary":        "Added the deck to the menu",
				"files_created":  []string{"app/menu/french-entry.tsx", "app/menu/french-entry.test.tsx"},
				"files_modified": []string{"app/menu/index.tsx"},
				"tool_count":     9,
			},
			{
				"name": "Keen-13", "caste": "watcher", "status": "completed", "task_id": "2.4",
				"summary":       "Checked progress is saved",
				"files_created": []string{"app/progress/save.ts", "app/progress/save.test.ts", "app/progress/restore.ts"},
				"outputs":       []string{".aether/data/spawn-tree.txt"},
				"tool_count":    7,
			},
		}
	}
	if variant == buildScreenFailed {
		workers[2]["status"] = "failed"
		workers[2]["summary"] = "Could not find the menu component"
		workers[2]["blockers"] = []string{"the menu component moved and the import no longer resolves"}
		workers[3]["status"] = "blocked"
		workers[3]["summary"] = "Waiting on the menu entry"
	}
	return writeCeremonyTestJSON(t, map[string]interface{}{
		"dispatch_manifest": manifest,
		"dispatches":        workers,
	})
}

func renderBuildScreenFixture(t *testing.T, variant buildScreenVariant) string {
	t.Helper()
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	buildScreenStateFixture(t, variant)
	_, visual := renderCeremonyCloseout("build", buildScreenCompletionFixture(t, variant))
	return stripANSI(visual)
}

var wholeWordFinishedRe = regexp.MustCompile(`\bFINISHED\b`)

func TestBuildScreenLeadsWithTheVerdict(t *testing.T) {
	cases := []struct {
		variant  buildScreenVariant
		verdict  []string
		sentence string
		glyph    string
	}{
		{buildScreenOneHelper, []string{"PHASE 2 BUILT — READY TO CHECK"}, "1 helper finished its work.", voiceGlyph("done")},
		{buildScreenPartial, []string{"PHASE 2 PARTLY BUILT"}, "3 of 4 tasks done; 1 still to do.", voiceGlyph("warning")},
		{buildScreenFailed, []string{"PHASE 2 BUILD STOPPED", "helper failed"}, "", voiceGlyph("blocked")},
	}
	for _, tc := range cases {
		t.Run(string(tc.variant), func(t *testing.T) {
			rendered := renderBuildScreenFixture(t, tc.variant)
			box := firstNonBlankLines(rendered, 4)
			if len(box) != 4 {
				t.Fatalf("expected a four-line verdict box at the top, got %q\n%s", box, rendered)
			}
			if !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
				t.Errorf("verdict box is not framed by full heavy lines: %q\n%s", box, rendered)
			}
			if !strings.Contains(box[1], tc.glyph) {
				t.Errorf("verdict line %q does not lead with %s\n%s", box[1], tc.glyph, rendered)
			}
			for _, want := range tc.verdict {
				if !strings.Contains(box[1], want) {
					t.Errorf("verdict line = %q, want it to say %q\n%s", box[1], want, rendered)
				}
			}
			if tc.sentence != "" && !strings.HasPrefix(strings.TrimSpace(box[2]), tc.sentence) {
				t.Errorf("verdict sentence = %q, want it to start %q\n%s", strings.TrimSpace(box[2]), tc.sentence, rendered)
			}
			top := strings.Index(rendered, box[0])
			if strings.TrimSpace(rendered[:top]) != "" {
				t.Errorf("something is drawn above the verdict box:\n%s", rendered[:top])
			}
			// A build that is not complete is never dressed up as ready.
			if tc.variant != buildScreenOneHelper && strings.Contains(box[1], "READY TO CHECK") {
				t.Errorf("an unfinished build claims it is ready to check: %q", box[1])
			}
		})
	}

	t.Run("one finished helper folds into the verdict", func(t *testing.T) {
		rendered := renderBuildScreenFixture(t, buildScreenOneHelper)
		if strings.Contains(checkScreenBody(rendered), spacedTitle("Helpers")) {
			t.Errorf("a single helper that finished still gets its own section:\n%s", rendered)
		}
		box := firstNonBlankLines(rendered, 4)
		if !strings.Contains(box[2], "continue") {
			t.Errorf("the ready verdict does not say how to check the work: %q", box[2])
		}
	})

	t.Run("a failed build says what to fix", func(t *testing.T) {
		rendered := renderBuildScreenFixture(t, buildScreenFailed)
		body := checkScreenBody(rendered)
		fix := strings.Index(body, spacedTitle("What To Fix"))
		if fix < 0 || !strings.Contains(body[fix:], "the menu component moved") {
			t.Errorf("a stopped build does not name what stopped it under WHAT TO FIX:\n%s", body)
		}
		if !strings.Contains(body, "Brick-79") || !strings.Contains(body, spacedTitle("Helpers")) {
			t.Errorf("a failed helper is not named on the screen:\n%s", body)
		}
	})

	t.Run("a partial build lists what is still to do", func(t *testing.T) {
		body := checkScreenBody(renderBuildScreenFixture(t, buildScreenPartial))
		todo := strings.Index(body, spacedTitle("Still To Do"))
		if todo < 0 || !strings.Contains(body[todo:], "Save progress between sessions") {
			t.Errorf("a partial build does not list the unfinished task under STILL TO DO:\n%s", body)
		}
	})
}

func TestBuildScreenSectionsAreSeparated(t *testing.T) {
	for variant, headers := range map[buildScreenVariant][]string{
		buildScreenRealistic: {"What Changed", "Helpers", "Checks", "Behind The Scenes"},
		buildScreenPartial:   {"What Changed", "Helpers", "Checks", "Still To Do", "Behind The Scenes"},
		buildScreenFailed:    {"What To Fix", "What Changed", "Helpers", "Checks", "Behind The Scenes"},
	} {
		t.Run(string(variant), func(t *testing.T) {
			assertSectionsSeparated(t, checkScreenBody(renderBuildScreenFixture(t, variant)), headers)
		})
	}

	t.Run("direct lane", func(t *testing.T) {
		setupBuildFlowTest(t)
		state, phase, dispatches := buildScreenFixture(2, 4)
		body := checkScreenBody(stripANSI(renderBuildVisualWithDispatches(state, phase, dispatches, colony.VerificationDepthStandard)))
		assertSectionsSeparated(t, body, []string{"Helpers", "Checks", "Still To Do", "Behind The Scenes"})
	})
}

func TestBuildScreenStaysShort(t *testing.T) {
	rendered := renderBuildScreenFixture(t, buildScreenRealistic)
	body := strings.TrimRight(checkScreenBody(rendered), "\n")
	lines := strings.Split(body, "\n")
	if len(lines) > 35 {
		t.Errorf("build screen body is %d lines, want at most 35:\n%s", len(lines), body)
	}
	for _, want := range []string{"PHASE 2 BUILT — READY TO CHECK", "4 helpers finished their work.", "app/decks/french.json", "… 6 more", "Forge-11", "Keen-13", "5 steering suggestions", "Full detail:"} {
		if !strings.Contains(body, want) {
			t.Errorf("build screen is missing %q:\n%s", want, body)
		}
	}
	for _, banned := range []string{".aether/data", "Credited files", "Reason: pattern noticed", "generic suggestion number 4", "Tools:", "── Worker Results", "Changed / Produced", "── Colony State", "── Handoff"} {
		if strings.Contains(rendered, banned) {
			t.Errorf("build screen still shows %q:\n%s", banned, rendered)
		}
	}
}

// sealScreenCeremonyFixture saves a sealed (or force-closed) project and the
// seal completion packet its three final reviewers returned.
func sealScreenCeremonyFixture(t *testing.T, forced bool) string {
	t.Helper()
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "Add a French deck to the flashcard app so learners can practise five new words every single day"
	outcome := colony.SealOutcome{
		OutcomeKind:     colony.OutcomeKindVerifiedCompletion,
		Disposition:     colony.SealDispositionVerified,
		CompletedPhases: []int{1, 2, 3},
		StateEffect:     colony.LifecycleStateEffectCommitted,
		Transaction:     colony.LifecycleTransactionReference{Stage: colony.TransactionStageVerified},
		Provenance:      colony.RecoveryProvenanceConfirmed,
	}
	phases := []colony.Phase{
		{ID: 1, Name: "Import old cards", Status: colony.PhaseCompleted},
		{ID: 2, Name: "French Basics deck", Status: colony.PhaseCompleted},
		{ID: 3, Name: "Daily reminders", Status: colony.PhaseCompleted},
	}
	milestone := "Crowned Anthill"
	if forced {
		outcome.OutcomeKind = colony.OutcomeKindForcedIncompleteClosure
		outcome.Disposition = colony.SealDispositionForcedIncomplete
		outcome.CompletedPhases = []int{1}
		outcome.IncompletePhases = []int{2, 3}
		outcome.OwnerReason = "the deadline moved"
		phases[1].Status = colony.PhaseInProgress
		phases[2].Status = colony.PhaseReady
		milestone = "Sealed Chambers"
	}
	state := colony.ColonyState{
		Version:     "3.0",
		Goal:        &goal,
		State:       colony.StateCOMPLETED,
		Milestone:   milestone,
		SealOutcome: &outcome,
		Plan:        colony.Plan{Phases: phases},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save colony state: %v", err)
	}

	finding := func(severity, description string) map[string]interface{} {
		return map[string]interface{}{"domain": "quality", "severity": severity, "description": description, "blocking": false}
	}
	return writeCeremonyTestJSON(t, map[string]interface{}{
		"seal_manifest": map[string]interface{}{"workflow": "seal", "phase": 3, "phase_name": "Daily reminders"},
		"dispatches": []map[string]interface{}{
			{
				"stage": "seal-review", "wave": 1, "caste": "gatekeeper", "name": "Gate-12", "task_id": "seal-review-gatekeeper",
				"status": "completed", "summary": "No release blockers found.",
				"findings": []map[string]interface{}{
					finding("MEDIUM", "Release notes skip the new daily limit"),
					finding("MEDIUM", "The menu entry has no keyboard shortcut"),
					finding("LOW", "One log line still says TODO"),
					finding("LOW", "A test name has a typo"),
					finding("LOW", "The deck icon is slightly blurry"),
				},
			},
			{
				"stage": "seal-review", "wave": 1, "caste": "auditor", "name": "Ledger-9", "task_id": "seal-review-auditor",
				"status": "completed", "summary": "Quality is acceptable.",
				"findings": []map[string]interface{}{
					finding("MEDIUM", "Two helpers duplicate the same date maths"),
					finding("MEDIUM", "The loader swallows one error"),
					finding("LOW", "A comment is out of date"),
					finding("LOW", "One file is longer than the others"),
				},
			},
			{
				"stage": "seal-review", "wave": 1, "caste": "probe", "name": "Probe-3", "task_id": "seal-review-probe",
				"status": "completed", "summary": "Coverage is fine.",
			},
		},
	})
}

func renderSealScreenFixture(t *testing.T, forced bool) string {
	t.Helper()
	completion := sealScreenCeremonyFixture(t, forced)
	_, visual := renderCeremonyCloseout("seal", completion)
	return stripANSI(visual)
}

func TestSealScreenLeadsWithTheVerdict(t *testing.T) {
	assertBox := func(t *testing.T, rendered string) []string {
		t.Helper()
		box := firstNonBlankLines(rendered, 4)
		if len(box) != 4 || !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
			t.Fatalf("seal screen does not lead with a verdict box: %q\n%s", box, rendered)
		}
		top := strings.Index(rendered, box[0])
		if strings.TrimSpace(rendered[:top]) != "" {
			t.Errorf("something is drawn above the verdict box:\n%s", rendered[:top])
		}
		return box
	}

	t.Run("finished, wrapper closeout", func(t *testing.T) {
		rendered := renderSealScreenFixture(t, false)
		box := assertBox(t, rendered)
		if !strings.Contains(box[1], "🏺") || !strings.Contains(box[1], "PROJECT SEALED — FINISHED") {
			t.Errorf("verdict = %q, want 🏺 PROJECT SEALED — FINISHED\n%s", box[1], rendered)
		}
		sentence := strings.TrimSpace(box[2])
		if n := len(strings.Fields(sentence)); n > 16 || !strings.Contains(sentence, "French deck") {
			t.Errorf("verdict sentence %q should carry the goal, shortened (got %d words)", sentence, n)
		}
	})

	t.Run("forced, wrapper closeout", func(t *testing.T) {
		rendered := renderSealScreenFixture(t, true)
		box := assertBox(t, rendered)
		if !strings.Contains(box[1], "PROJECT CLOSED EARLY — 2 phases unfinished") || !strings.Contains(box[1], voiceGlyph("warning")) {
			t.Errorf("forced verdict = %q, want ⚠️ PROJECT CLOSED EARLY — 2 phases unfinished\n%s", box[1], rendered)
		}
		if wholeWordFinishedRe.MatchString(rendered) || strings.Contains(rendered, "🏺") {
			t.Errorf("an incomplete seal is presented as finished:\n%s", rendered)
		}
		if !strings.Contains(rendered, "the deadline moved") {
			t.Errorf("the forced seal hides the owner's reason:\n%s", rendered)
		}
	})

	t.Run("finished, aether seal", func(t *testing.T) {
		result := lifecycleCloseout199SealResult(colony.SealDispositionVerified)
		result.Outcome.CompletedPhases = []int{1, 2, 3}
		rendered := stripANSI(RenderSealOutcome(result))
		box := assertBox(t, rendered)
		if !strings.Contains(box[1], "PROJECT SEALED — FINISHED") {
			t.Errorf("verdict = %q, want PROJECT SEALED — FINISHED\n%s", box[1], rendered)
		}
		if strings.TrimSpace(box[2]) != "All 3 phases are finished and checked." {
			t.Errorf("verdict sentence = %q", strings.TrimSpace(box[2]))
		}
	})

	t.Run("forced, aether seal", func(t *testing.T) {
		result := lifecycleCloseout199SealResult(colony.SealDispositionForcedIncomplete)
		result.Outcome.IncompletePhases = []int{2}
		rendered := stripANSI(RenderSealOutcome(result))
		box := assertBox(t, rendered)
		if !strings.Contains(box[1], "PROJECT CLOSED EARLY — 1 phase unfinished") {
			t.Errorf("forced verdict = %q, want PROJECT CLOSED EARLY — 1 phase unfinished\n%s", box[1], rendered)
		}
		if wholeWordFinishedRe.MatchString(rendered) || strings.Contains(rendered, "SEALED — ") {
			t.Errorf("an incomplete seal is presented as finished:\n%s", rendered)
		}
	})
}

func TestSealScreenSummarisesTheReview(t *testing.T) {
	rendered := renderSealScreenFixture(t, false)
	body := checkScreenBody(rendered)
	assertSectionsSeparated(t, body, []string{"Final Review"})
	review := body[strings.Index(body, spacedTitle("Final Review")):]
	if end := strings.Index(review[1:], "━━ "); end > 0 {
		review = review[:end+1]
	}
	if !strings.Contains(review, "4 medium and 5 small notes, none blocking") {
		t.Errorf("FINAL REVIEW does not count the notes in plain words:\n%s", review)
	}
	shown := 0
	for _, f := range []string{"Release notes skip", "no keyboard shortcut", "log line still says TODO", "test name has a typo", "icon is slightly blurry", "duplicate the same date maths", "swallows one error", "comment is out of date", "longer than the others"} {
		if strings.Contains(review, f) {
			shown++
		}
	}
	if shown == 0 || shown > 3 {
		t.Errorf("FINAL REVIEW shows %d findings, want between 1 and 3:\n%s", shown, review)
	}
	for _, banned := range []string{"── Worker Results", "Workers:", "Delivery Readiness"} {
		if strings.Contains(rendered, banned) {
			t.Errorf("seal screen still shows %q:\n%s", banned, rendered)
		}
	}

	t.Run("delivery keeps only what matters", func(t *testing.T) {
		agree := sealDeliveryLines("  Source version: 1.0.88 ✓\n  Binary version: 1.0.88 ✓\n  Hub version:   1.0.88 ✓\n  Git status:    82 uncommitted changes\n")
		joined := strings.Join(agree, "\n")
		if !strings.Contains(joined, "82 changes not saved to version history") {
			t.Errorf("delivery does not say the changes are unsaved: %q", joined)
		}
		if strings.Contains(joined, "1.0.88") {
			t.Errorf("delivery shows version lines that agree: %q", joined)
		}
		disagree := strings.Join(sealDeliveryLines("  Source version: 1.0.88 ✓\n  Binary version: 1.0.88 ✓\n  Hub version:   1.0.87\n  Git status:    clean\n"), "\n")
		if !strings.Contains(disagree, "1.0.87") {
			t.Errorf("delivery hides a version that disagrees: %q", disagree)
		}
		if len(sealDeliveryLines("  Source version: 1.0.88 ✓\n  Binary version: 1.0.88 ✓\n  Hub version:   1.0.88 ✓\n  Git status:    clean\n")) != 0 {
			t.Errorf("delivery shows a section when nothing needs attention")
		}
	})
}

// TestBuildAndSealScreensLeaveTheResultUntouched: the layout change is
// presentation only. Drawing the build and seal screens must leave the
// result -- the same value the JSON output is written from -- unchanged.
func TestBuildAndSealScreensLeaveTheResultUntouched(t *testing.T) {
	assertUnchanged := func(t *testing.T, name string, value interface{}, draw func()) {
		t.Helper()
		before, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal before: %v", err)
		}
		draw()
		after, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal after: %v", err)
		}
		if string(before) != string(after) {
			t.Errorf("drawing the %s changed its result:\nbefore: %s\nafter:  %s", name, before, after)
		}
	}

	for _, variant := range []buildScreenVariant{buildScreenRealistic, buildScreenPartial, buildScreenFailed} {
		t.Run(string(variant), func(t *testing.T) {
			saveGlobals(t)
			s, tmpDir := newTestStore(t)
			t.Cleanup(func() { os.RemoveAll(tmpDir) })
			store = s
			buildScreenStateFixture(t, variant)
			result, _ := renderCeremonyCloseout("build", buildScreenCompletionFixture(t, variant))
			assertUnchanged(t, "build screen", result, func() { renderCeremonyCloseoutVisualBody(result) })
		})
	}

	t.Run("seal", func(t *testing.T) {
		result, _ := renderCeremonyCloseout("seal", sealScreenCeremonyFixture(t, false))
		assertUnchanged(t, "seal screen", result, func() { renderCeremonyCloseoutVisualBody(result) })
		for _, disposition := range []colony.SealDisposition{colony.SealDispositionVerified, colony.SealDispositionForcedIncomplete} {
			outcome := lifecycleCloseout199SealResult(disposition)
			outcome.Outcome.Transaction.Stage = colony.TransactionStageVerified
			outcome.Outcome.Provenance = colony.RecoveryProvenanceConfirmed
			outcome.Receipt.Provenance = colony.RecoveryProvenanceConfirmed
			outcome.Receipt.OutcomeKind = outcome.Outcome.OutcomeKind
			outcome.Evidence.OutcomeKind = outcome.Outcome.OutcomeKind
			outcome.Evidence.Disposition = outcome.Outcome.Disposition
			outcome.Receipt.Transaction.Stage = colony.TransactionStageVerified
			assertUnchanged(t, "aether seal screen", outcome, func() { RenderSealOutcome(outcome) })
		}
	})
}

func TestBuildScreenFailedChecksOverrideCompletedTasks(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	state := buildScreenStateFixture(t, buildScreenOneHelper)
	path := "build/phase-2/attempt-screen.json"
	attempt := buildAttemptRecord{ID: "screen-attempt", Phase: 2, Status: buildAttemptBuilt,
		FreeChecks: &buildFreeCheckReport{ChecksRun: []string{"build", "tests"}, Failed: []string{"tests"}, Passed: false}}
	if err := s.SaveJSON(path, attempt); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(latestBuildAttemptPointerPath(2), latestBuildAttemptPointer{AttemptID: attempt.ID, Path: path}); err != nil {
		t.Fatal(err)
	}
	body := renderBuildScreen(state, state.Plan.Phases[1], []map[string]interface{}{{"name": "Builder", "status": "completed"}}, nil)
	if !strings.Contains(body, "BUILD STOPPED — checks failed") || strings.Contains(body, "READY TO CHECK") {
		t.Fatalf("failed check hidden by completed tasks:\n%s", body)
	}
}

func TestSealScreenWithoutOutcomeCannotClaimFinished(t *testing.T) {
	body := sealScreenVerdict(nil, "A saved goal")
	if strings.Contains(body, "SEALED — FINISHED") || !strings.Contains(body, "NOT VERIFIED") {
		t.Fatalf("missing outcome became success:\n%s", body)
	}
}

func TestBuildScreenUsesCommittedBuildCreditBeforeContinue(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	store = s
	id := "1.1"
	phase := colony.Phase{ID: 1, Name: "Write the deck", Tasks: []colony.Task{{ID: &id, Goal: "Create the deck", Status: colony.TaskInProgress}}}
	state := colony.ColonyState{State: colony.StateBUILT, CurrentPhase: 1, Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	attempt := buildAttemptRecord{ID: "screen-credit", Phase: 1, Status: buildAttemptBuilt, Dispatches: []codexBuildDispatch{{Name: "Builder", TaskID: id, Status: "completed"}}}
	path := "build/phase-1/screen-credit.json"
	if err := s.SaveJSON(path, attempt); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON(latestBuildAttemptPointerPath(1), latestBuildAttemptPointer{AttemptID: attempt.ID, Path: path}); err != nil {
		t.Fatal(err)
	}
	body := renderBuildScreen(state, phase, buildScreenWorkerMaps(attempt.Dispatches), nil)
	if !strings.Contains(body, "BUILT — READY TO CHECK") || strings.Contains(body, spacedTitle("Still To Do")) {
		t.Fatalf("recorded build credit was lost:\n%s", body)
	}
	if phase.Tasks[0].Status != colony.TaskInProgress {
		t.Fatal("rendering advanced verification state")
	}
}
