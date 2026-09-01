package cmd

import (
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
)

func autopilotLessonFixture(boundary time.Time) learn.Entry {
	return learn.Entry{
		ID:             "lesson-valid",
		Content:        "Keep provider readiness retries inside the provider layer.",
		Classification: learn.ClassRepoLocal,
		CreatedAt:      boundary.Add(time.Hour).Format(time.RFC3339Nano),
		Phase:          2,
		Status:         learn.StatusHypothesis,
		Evidence: learn.Evidence{
			RunID: "run-2",
			Phase: 2,
			Workers: []learn.WorkerEvidence{
				{Name: "Builder", Caste: "builder", Status: "completed"},
				{Name: "Watcher", Caste: "watcher", Status: "completed"},
			},
			GatesPassed: 3,
			GatesTotal:  3,
			Timestamp:   boundary.Add(time.Hour).Format(time.RFC3339Nano),
		},
	}
}

func autopilotLessonPlan(boundary time.Time, revisionID string) colony.Plan {
	return colony.Plan{
		GeneratedAt:      &boundary,
		ActiveRevisionID: revisionID,
		Revisions: []colony.PlanRevision{{
			SchemaVersion: 1,
			Number:        1,
			ID:            revisionID,
			CreatedAt:     boundary.Format(time.RFC3339Nano),
		}},
		Phases: []colony.Phase{{ID: 1}},
	}
}

func TestConfirmedAutopilotLessonsEligibility(t *testing.T) {
	boundary := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	plan := autopilotLessonPlan(boundary, "revision-1")
	valid := autopilotLessonFixture(boundary)

	tests := []struct {
		name   string
		mutate func(*learn.Entry)
	}{
		{name: "phase_non_positive", mutate: func(entry *learn.Entry) { entry.Phase = 0 }},
		{name: "evidence_phase_non_positive", mutate: func(entry *learn.Entry) { entry.Evidence.Phase = 0 }},
		{name: "empty_content", mutate: func(entry *learn.Entry) { entry.Content = "  " }},
		{name: "blocked_classification", mutate: func(entry *learn.Entry) { entry.Classification = learn.ClassBlocked }},
		{name: "disproven_status", mutate: func(entry *learn.Entry) { entry.Status = learn.StatusDisproven }},
		{name: "pre_revision", mutate: func(entry *learn.Entry) { entry.Evidence.Timestamp = boundary.Format(time.RFC3339Nano) }},
		{name: "missing_evidence_timestamp", mutate: func(entry *learn.Entry) { entry.Evidence.Timestamp = "" }},
		{name: "raw_observation_without_workers", mutate: func(entry *learn.Entry) { entry.Evidence.Workers = nil }},
		{name: "incomplete_worker", mutate: func(entry *learn.Entry) { entry.Evidence.Workers[1].Status = "running" }},
		{name: "missing_gates", mutate: func(entry *learn.Entry) { entry.Evidence.GatesPassed, entry.Evidence.GatesTotal = 0, 0 }},
		{name: "failing_gate", mutate: func(entry *learn.Entry) { entry.Evidence.GatesPassed = 2 }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := valid
			entry.Evidence.Workers = append([]learn.WorkerEvidence{}, valid.Evidence.Workers...)
			tc.mutate(&entry)
			lessons, err := confirmedAutopilotLessonsSincePlan(plan, []learn.Entry{entry})
			if err != nil {
				t.Fatalf("select confirmed lessons: %v", err)
			}
			if len(lessons) != 0 {
				t.Fatalf("excluded entry produced %d lesson(s): %+v", len(lessons), lessons)
			}
		})
	}

	lessons, err := confirmedAutopilotLessonsSincePlan(plan, []learn.Entry{valid})
	if err != nil {
		t.Fatalf("select valid confirmed lesson: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("valid entry produced %d lessons, want 1: %+v", len(lessons), lessons)
	}
	if lessons[0].EntryID != valid.ID || lessons[0].PlanRevisionID != "revision-1" {
		t.Fatalf("lesson lost evidence identity: %+v", lessons[0])
	}
}

func TestConfirmedAutopilotLessonsDeduplicatesNormalizedContent(t *testing.T) {
	boundary := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	first := autopilotLessonFixture(boundary)
	second := autopilotLessonFixture(boundary)
	second.ID = "lesson-duplicate"
	second.Content = "  KEEP   provider readiness retries inside the provider layer.  "
	second.Evidence.Timestamp = boundary.Add(2 * time.Hour).Format(time.RFC3339Nano)

	lessons, err := confirmedAutopilotLessonsSincePlan(autopilotLessonPlan(boundary, "revision-1"), []learn.Entry{second, first})
	if err != nil {
		t.Fatalf("select confirmed lessons: %v", err)
	}
	if len(lessons) != 1 {
		t.Fatalf("normalized duplicates produced %d lessons, want 1: %+v", len(lessons), lessons)
	}
	if lessons[0].EntryID != first.ID {
		t.Fatalf("dedup winner = %q, want earliest evidence %q", lessons[0].EntryID, first.ID)
	}
}

func TestConfirmedAutopilotLessonsUsesGeneratedAtForLegacyPlan(t *testing.T) {
	boundary := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)
	plan := colony.Plan{GeneratedAt: &boundary, Phases: []colony.Phase{{ID: 1}}}
	lessons, err := confirmedAutopilotLessonsSincePlan(plan, []learn.Entry{autopilotLessonFixture(boundary)})
	if err != nil {
		t.Fatalf("select legacy-plan lessons: %v", err)
	}
	if len(lessons) != 1 || lessons[0].PlanRevisionID == "" {
		t.Fatalf("legacy boundary did not produce a revision-scoped lesson: %+v", lessons)
	}
}

func TestLessonAwareReplanMatrix(t *testing.T) {
	lesson := confirmedAutopilotLesson{EntryID: "lesson-1"}
	tests := []struct {
		name      string
		completed int
		interval  int
		lessons   []confirmedAutopilotLesson
		bypass    bool
		wantDue   bool
	}{
		{name: "interval_only", completed: 2, interval: 2, wantDue: false},
		{name: "lesson_only", completed: 1, interval: 2, lessons: []confirmedAutopilotLesson{lesson}, wantDue: false},
		{name: "interval_and_lesson", completed: 2, interval: 2, lessons: []confirmedAutopilotLesson{lesson}, wantDue: true},
		{name: "continue_bypass", completed: 2, interval: 2, lessons: []confirmedAutopilotLesson{lesson}, bypass: true, wantDue: false},
		{name: "disabled_interval", completed: 2, interval: 0, lessons: []confirmedAutopilotLesson{lesson}, wantDue: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := lessonAwareReplanDue(tc.completed, tc.interval, tc.lessons, tc.bypass); got != tc.wantDue {
				t.Fatalf("lessonAwareReplanDue() = %t, want %t", got, tc.wantDue)
			}
		})
	}
}

func autopilotCadencePlan(revisionID string, snapshot, current []colony.Phase) colony.Plan {
	boundary := time.Date(2026, time.September, 1, 8, 0, 0, 0, time.UTC)
	return colony.Plan{
		GeneratedAt:      &boundary,
		ActiveRevisionID: revisionID,
		Revisions: []colony.PlanRevision{{
			SchemaVersion: 1,
			Number:        1,
			ID:            revisionID,
			CreatedAt:     boundary.Format(time.RFC3339Nano),
			Phases:        snapshot,
		}},
		Phases: current,
	}
}

func TestAutopilotReplanCadenceReconstructsAcrossRestart(t *testing.T) {
	snapshot := []colony.Phase{
		{ID: 1, Status: colony.PhaseReady},
		{ID: 2, Status: colony.PhasePending},
		{ID: 3, Status: colony.PhasePending},
	}
	current := append([]colony.Phase(nil), snapshot...)
	plan := autopilotCadencePlan("revision-cadence", snapshot, current)

	assertCadence := func(name string, decisions []PendingDecision, completed, lastHandled, nextBoundary, dueBoundary, checkpointPhase int) {
		t.Helper()
		got, err := projectAutopilotReplanCadence(plan, decisions, 2)
		if err != nil {
			t.Fatalf("%s: project cadence: %v", name, err)
		}
		if got.CompletedSinceRevision != completed || got.LastHandledBoundary != lastHandled || got.NextBoundary != nextBoundary || got.DueBoundary != dueBoundary || got.CheckpointPhaseID != checkpointPhase {
			t.Fatalf("%s: cadence = %+v, want completed=%d handled=%d next=%d due=%d phase=%d", name, got, completed, lastHandled, nextBoundary, dueBoundary, checkpointPhase)
		}
	}

	assertCadence("accepted revision", nil, 0, 0, 2, 0, 0)
	plan.Phases[0].Status = colony.PhaseCompleted
	assertCadence("one durable completion", nil, 1, 0, 2, 0, 0)
	plan.Phases[1].Status = colony.PhaseCompleted
	assertCadence("missed persistence remains due after restart", nil, 2, 0, 2, 2, 2)

	resolvedHandled := PendingDecision{
		Type:                  autopilotReplanDecisionType,
		PlanRevisionID:        "revision-cadence",
		LatestCheckpointPhase: 2,
		Resolved:              true,
	}
	otherRevision := PendingDecision{
		Type:                  autopilotReplanDecisionType,
		PlanRevisionID:        "revision-other",
		LatestCheckpointPhase: 3,
	}
	unresolvedHandled := resolvedHandled
	unresolvedHandled.Resolved = false
	assertCadence("unresolved same-revision note handles boundary", []PendingDecision{otherRevision, unresolvedHandled}, 2, 2, 4, 0, 0)
	assertCadence("resolved same-revision note handles boundary", []PendingDecision{otherRevision, resolvedHandled}, 2, 2, 4, 0, 0)
}

func TestAutopilotReplanCadenceResetsOnlyOnAcceptedRevision(t *testing.T) {
	snapshot := []colony.Phase{
		{ID: 1, Status: colony.PhaseReady},
		{ID: 2, Status: colony.PhasePending},
	}
	current := []colony.Phase{
		{ID: 1, Status: colony.PhaseCompleted},
		{ID: 2, Status: colony.PhaseCompleted},
	}
	plan := autopilotCadencePlan("revision-one", snapshot, current)
	resolved := []PendingDecision{{
		Type:                  autopilotReplanDecisionType,
		PlanRevisionID:        "revision-one",
		LatestCheckpointPhase: 2,
		Resolved:              true,
	}}

	beforeRestart, err := projectAutopilotReplanCadence(plan, resolved, 2)
	if err != nil {
		t.Fatalf("project before restart: %v", err)
	}
	afterRestart, err := projectAutopilotReplanCadence(plan, resolved, 2)
	if err != nil {
		t.Fatalf("project after restart: %v", err)
	}
	if beforeRestart != afterRestart || afterRestart.CompletedSinceRevision != 2 || afterRestart.LastHandledBoundary != 2 {
		t.Fatalf("restart or note resolution reset cadence: before=%+v after=%+v", beforeRestart, afterRestart)
	}

	acceptedAt := time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	plan.ActiveRevisionID = "revision-two"
	plan.Revisions = append(plan.Revisions, colony.PlanRevision{
		SchemaVersion: 1,
		Number:        2,
		ID:            "revision-two",
		ParentID:      "revision-one",
		CreatedAt:     acceptedAt.Format(time.RFC3339Nano),
		Phases:        append([]colony.Phase(nil), current...),
	})
	reset, err := projectAutopilotReplanCadence(plan, resolved, 2)
	if err != nil {
		t.Fatalf("project accepted revision: %v", err)
	}
	if reset.CompletedSinceRevision != 0 || reset.LastHandledBoundary != 0 || reset.NextBoundary != 2 || reset.DueBoundary != 0 || reset.CheckpointPhaseID != 0 {
		t.Fatalf("new accepted revision did not reset cadence: %+v", reset)
	}
}

func TestAutopilotReplanCadenceLegacyCompatibilityCountsCompletedPhases(t *testing.T) {
	plan := colony.Plan{Phases: []colony.Phase{
		{ID: 4, Status: colony.PhaseCompleted},
		{ID: 9, Status: colony.PhaseCompleted},
		{ID: 12, Status: colony.PhaseReady},
	}}
	got, err := projectAutopilotReplanCadence(plan, nil, 2)
	if err != nil {
		t.Fatalf("project legacy cadence: %v", err)
	}
	if got.CompletedSinceRevision != 2 || got.DueBoundary != 2 || got.CheckpointPhaseID != 9 {
		t.Fatalf("legacy cadence = %+v, want two durable completions due at phase 9", got)
	}
}

func TestReplanDecisionUpsertAccumulatesOneNotePerRevision(t *testing.T) {
	s, tmpDir := newTestStore(t)
	store = s
	t.Cleanup(func() { store = nil })
	goal := "Run safely overnight"
	initialized := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	boundary := initialized.Add(time.Hour)
	state := colony.ColonyState{
		Goal:          &goal,
		InitializedAt: &initialized,
		Plan:          autopilotLessonPlan(boundary, "revision-1"),
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed colony state in %s: %v", tmpDir, err)
	}

	lessons := []confirmedAutopilotLesson{{
		EntryID:        "lesson-1",
		Content:        "Keep provider retries bounded.",
		ContentHash:    "hash-1",
		Phase:          1,
		PlanRevisionID: "revision-1",
	}}
	for _, checkpoint := range []int{2, 4, 6} {
		lessons = append(lessons, confirmedAutopilotLesson{
			EntryID:        "lesson-" + time.Unix(int64(checkpoint), 0).Format("150405"),
			Content:        "Evidence from checkpoint",
			ContentHash:    "hash-checkpoint",
			Phase:          checkpoint,
			PlanRevisionID: "revision-1",
		})
		if _, err := upsertAutopilotReplanDecision(state, checkpoint, lessons, initialized.Add(time.Duration(checkpoint)*time.Hour)); err != nil {
			t.Fatalf("upsert checkpoint %d: %v", checkpoint, err)
		}
	}

	file := loadPendingDecisionFile()
	unresolved := []PendingDecision{}
	for _, decision := range file.Decisions {
		if decision.Type == "replan" && !decision.Resolved {
			unresolved = append(unresolved, decision)
		}
	}
	if len(unresolved) != 1 {
		t.Fatalf("unresolved replan notes = %d, want 1: %+v", len(unresolved), file.Decisions)
	}
	note := unresolved[0]
	if note.PlanRevisionID != "revision-1" || note.FirstCheckpointPhase != 2 || note.LatestCheckpointPhase != 6 {
		t.Fatalf("replan note did not span checkpoints: %+v", note)
	}
	if note.LessonCount != 2 {
		t.Fatalf("lesson_count = %d, want 2 normalized references", note.LessonCount)
	}
}

func TestReplanDecisionUpsertSupersedesOldRevision(t *testing.T) {
	s, _ := newTestStore(t)
	store = s
	t.Cleanup(func() { store = nil })
	goal := "Run safely overnight"
	now := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	state := colony.ColonyState{Goal: &goal, InitializedAt: &now, Plan: autopilotLessonPlan(now, "revision-1")}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	first := []confirmedAutopilotLesson{{EntryID: "first", Content: "First", ContentHash: "hash-first", Phase: 1, PlanRevisionID: "revision-1"}}
	if _, err := upsertAutopilotReplanDecision(state, 2, first, now.Add(time.Hour)); err != nil {
		t.Fatalf("upsert revision 1: %v", err)
	}

	state.Plan = autopilotLessonPlan(now.Add(2*time.Hour), "revision-2")
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save revision 2: %v", err)
	}
	second := []confirmedAutopilotLesson{{EntryID: "second", Content: "Second", ContentHash: "hash-second", Phase: 3, PlanRevisionID: "revision-2"}}
	if _, err := upsertAutopilotReplanDecision(state, 4, second, now.Add(3*time.Hour)); err != nil {
		t.Fatalf("upsert revision 2: %v", err)
	}

	file := loadPendingDecisionFile()
	unresolved, resolved := 0, 0
	for _, decision := range file.Decisions {
		if decision.Type != "replan" {
			continue
		}
		if decision.Resolved {
			resolved++
			if decision.PlanRevisionID != "revision-1" {
				t.Fatalf("resolved wrong revision: %+v", decision)
			}
		} else {
			unresolved++
			if decision.PlanRevisionID != "revision-2" {
				t.Fatalf("active note has wrong revision: %+v", decision)
			}
		}
	}
	if unresolved != 1 || resolved != 1 {
		t.Fatalf("replan revision rows unresolved=%d resolved=%d, want 1/1: %+v", unresolved, resolved, file.Decisions)
	}
}
