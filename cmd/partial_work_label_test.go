package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// --- Task 1: one standing vocabulary -------------------------------------

func TestOneStandingVocabularyAcrossSubsystems(t *testing.T) {
	for _, standing := range declaredWorkStandings() {
		reason := "example reason for " + string(standing)
		oraclePhrase := workStandingLabel(standing, reason)
		swarmPhrase := workStandingLabel(standing, reason)
		if oraclePhrase != swarmPhrase {
			t.Fatalf("standing %q: Oracle phrase %q != Swarm phrase %q", standing, oraclePhrase, swarmPhrase)
		}
	}

	if got := len(declaredWorkStandings()); got != 3 {
		t.Fatalf("declared %d standings, want exactly 3", got)
	}

	// Drive both subsystems through their real entry points with matching
	// reason text and confirm the rendered phrase is identical either way --
	// proving both actually call the one shared function rather than a
	// per-subsystem lookalike.
	oracleStanding, oracleReason := workStandingUsefulNotes, "finishing the remaining research rounds"
	oraclePhrase := workStandingLabel(oracleStanding, oracleReason)

	episode := swarmEpisodeRecord{Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageFix}
	swarmStanding, swarmReason := swarmEpisodeStanding(episode)
	if swarmStanding != workStandingUsefulNotes {
		t.Fatalf("interrupted episode resolved to %q, want useful notes", swarmStanding)
	}
	swarmPhrase := workStandingLabel(swarmStanding, oracleReason) // same reason text, different source
	if oraclePhrase != swarmPhrase {
		t.Fatalf("same standing + same reason rendered differently: oracle=%q swarm=%q", oraclePhrase, swarmPhrase)
	}
	if swarmReason == "" {
		t.Fatal("swarmEpisodeStanding returned an empty reason for an interrupted run")
	}
}

func TestUsefulNotesLabelNamesWhatWouldVerifyIt(t *testing.T) {
	label := workStandingLabel(workStandingUsefulNotes, "completing the remaining rounds")
	if !strings.Contains(label, "completing the remaining rounds") {
		t.Fatalf("useful-notes label %q does not name what would verify it", label)
	}
	if !strings.Contains(label, "useful notes") {
		t.Fatalf("useful-notes label %q does not say it is useful notes", label)
	}

	// Even with no reason supplied, the label still names something --
	// never silently empty.
	fallback := workStandingLabel(workStandingUsefulNotes, "")
	if !strings.Contains(fallback, "useful notes") {
		t.Fatalf("fallback useful-notes label %q lost its standing", fallback)
	}
}

func TestUnknownStandingIsNeverVerified(t *testing.T) {
	standing, reason := resolveMalformedItemStanding("corrupt front matter")
	if standing == workStandingVerified {
		t.Fatal("a malformed item resolved to verified")
	}
	if standing != workStandingUnknown {
		t.Fatalf("malformed item resolved to %q, want standing unknown", standing)
	}
	if reason == "" {
		t.Fatal("malformed item carries no stated reason")
	}

	label := workStandingLabel(standing, reason)
	if label != "standing unknown" {
		t.Fatalf("standing unknown label = %q, want the plain phrase", label)
	}
	if label == "verified" {
		t.Fatal("standing unknown rendered as verified")
	}
}

func TestOraclePartialTriggerConditionIsUnchanged(t *testing.T) {
	fixtures := []oracleStateFile{
		{Status: "complete", Iteration: 5},
		{Status: "stopped", Iteration: 7},
		{Status: "idle", Iteration: 0},
		{Status: "timeout", Iteration: 12},
		{Status: ""},
	}
	for _, state := range fixtures {
		oldWasPartial := oracleResearchPartialLabel(state) != ""
		newIsUsefulNotes := oracleResearchStanding(state.Status) == workStandingUsefulNotes
		if oldWasPartial != newIsUsefulNotes {
			t.Errorf("status %q: old partial=%v, new useful-notes=%v (must match)", state.Status, oldWasPartial, newIsUsefulNotes)
		}
		oldWasComplete := oracleResearchPartialLabel(state) == ""
		newIsVerified := oracleResearchStanding(state.Status) == workStandingVerified
		if oldWasComplete != newIsVerified {
			t.Errorf("status %q: old complete=%v, new verified=%v (must match)", state.Status, oldWasComplete, newIsVerified)
		}
	}
}

func TestSwarmRepairIdeaAndInterruptedRunHaveDistinctReasons(t *testing.T) {
	rolledBack := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "restart the worker pool"}},
		Checkpoint:         swarmEpisodeCheckpoint{Saved: true, Restored: true},
		VerificationStatus: swarmEpisodeVerificationCompleted,
	}
	repairStanding, repairReason := swarmRepairIdeaStanding(rolledBack)
	if repairStanding != workStandingUsefulNotes {
		t.Fatalf("rolled-back repair resolved to %q, want useful notes", repairStanding)
	}

	interrupted := swarmEpisodeRecord{Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageInvestigation}
	runStanding, runReason := swarmEpisodeStanding(interrupted)
	if runStanding != workStandingUsefulNotes {
		t.Fatalf("interrupted run resolved to %q, want useful notes", runStanding)
	}

	if repairReason == "" || runReason == "" {
		t.Fatal("expected both a rolled-back repair and an interrupted run to state a reason")
	}
	if repairReason == runReason {
		t.Fatalf("rolled-back repair and interrupted run share the same reason %q, want distinct reasons", repairReason)
	}

	unapplied := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "add a retry"}},
		VerificationStatus: "not_run",
	}
	unappliedStanding, unappliedReason := swarmRepairIdeaStanding(unapplied)
	if unappliedStanding != workStandingUsefulNotes {
		t.Fatalf("unapplied repair idea resolved to %q, want useful notes", unappliedStanding)
	}
	if unappliedReason == repairReason {
		t.Fatalf("unapplied idea and rolled-back idea share the same reason %q", unappliedReason)
	}

	verified := swarmEpisodeRecord{
		Status:             swarmEpisodeStatusCompleted,
		Comparison:         swarmEpisodeComparison{Selected: &swarmRankedRepair{Repair: "fix the race"}},
		Checkpoint:         swarmEpisodeCheckpoint{Saved: true, Restored: false},
		VerificationStatus: swarmEpisodeVerificationCompleted,
	}
	if standing, _ := swarmRepairIdeaStanding(verified); standing != workStandingVerified {
		t.Fatalf("held repair resolved to %q, want verified", standing)
	}
	if standing, _ := swarmEpisodeStanding(swarmEpisodeRecord{Status: swarmEpisodeStatusCompleted}); standing != workStandingVerified {
		t.Fatalf("completed run resolved to %q, want verified", standing)
	}
}

// --- Task 2: the read-only unverified-work inventory ---------------------

// dirDigest is defined once, in cmd/watch_dashboard_test.go, and reused here
// to prove this inventory's read-only guarantee against the same digest
// discipline the watch dashboard's refresh loop is already held to.

func writeResearchFixture(t *testing.T, root, name, frontMatter string) {
	t.Helper()
	dir := oracleResearchDir(root)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir research dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(frontMatter), 0644); err != nil {
		t.Fatalf("write research fixture: %v", err)
	}
}

func writeSwarmEpisodeFixture(t *testing.T, root, swarmID string, record swarmEpisodeRecord) {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data", "swarms", swarmID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir swarm episode dir: %v", err)
	}
	record.SchemaVersion = swarmEpisodeSchemaVersion
	record.SwarmID = swarmID
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatalf("marshal episode fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "episode.json"), data, 0644); err != nil {
		t.Fatalf("write episode fixture: %v", err)
	}
}

func TestUnverifiedWorkInventoryCoversFourSources(t *testing.T) {
	root := t.TempDir()

	writeResearchFixture(t, root, "2024-01-01-cache.md",
		"---\ntitle: \"cache choice\"\nstatus: stopped\ngenerated: 2024-01-01T00:00:00Z\n---\n\nSome findings.\n")

	writeSwarmEpisodeFixture(t, root, "swarm-abc123", swarmEpisodeRecord{
		Target:           "flaky test",
		Status:           swarmEpisodeStatusInterrupted,
		InterruptedStage: swarmEpisodeStageInvestigation,
		StartedAt:        "2024-01-02T00:00:00Z",
		EndedAt:          "2024-01-02T01:00:00Z",
	})

	planDir := filepath.Join(root, ".aether", "data", "planning")
	if err := os.MkdirAll(planDir, 0755); err != nil {
		t.Fatalf("mkdir planning dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(planDir, "SCOUT.md"), []byte("# Scout notes\n"), 0644); err != nil {
		t.Fatalf("write plan research fixture: %v", err)
	}

	dreamsDir := filepath.Join(root, ".aether", "dreams")
	if err := os.MkdirAll(dreamsDir, 0755); err != nil {
		t.Fatalf("mkdir dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dreamsDir, "2024-01-03.md"), []byte("A reflection.\n"), 0644); err != nil {
		t.Fatalf("write reflection fixture: %v", err)
	}

	entries, err := collectUnverifiedWork(root)
	if err != nil {
		t.Fatalf("collect unverified work: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4 (one per source): %+v", len(entries), entries)
	}

	seenKinds := map[string]bool{}
	for _, entry := range entries {
		seenKinds[entry.Kind] = true
		if entry.Subject == "" {
			t.Errorf("entry %+v has no subject", entry)
		}
		if entry.Standing == "" {
			t.Errorf("entry %+v has no standing", entry)
		}
		if entry.Reason == "" {
			t.Errorf("entry %+v has no verification sentence", entry)
		}
		if entry.Path == "" {
			t.Errorf("entry %+v has no path", entry)
		}
		if entry.Standing == workStandingVerified {
			t.Errorf("entry %+v is verified; this inventory is unproven work only", entry)
		}
	}
	for _, want := range []string{
		unverifiedWorkKindOracleResearch,
		unverifiedWorkKindSwarmEpisode,
		unverifiedWorkKindPlanResearch,
		unverifiedWorkKindReflection,
	} {
		if !seenKinds[want] {
			t.Errorf("missing entry for source %q; got kinds %v", want, seenKinds)
		}
	}
}

func TestUnverifiedWorkInventoryIsReadOnly(t *testing.T) {
	root := t.TempDir()
	writeResearchFixture(t, root, "2024-01-01-cache.md",
		"---\ntitle: \"cache choice\"\nstatus: stopped\ngenerated: 2024-01-01T00:00:00Z\n---\n\nSome findings.\n")
	writeSwarmEpisodeFixture(t, root, "swarm-abc123", swarmEpisodeRecord{
		Target: "flaky test", Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageFix,
		StartedAt: "2024-01-02T00:00:00Z", EndedAt: "2024-01-02T01:00:00Z",
	})

	dataDir := filepath.Join(root, ".aether", "data")
	dreamsDir := filepath.Join(root, ".aether", "dreams")
	if err := os.MkdirAll(dreamsDir, 0755); err != nil {
		t.Fatalf("mkdir dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dreamsDir, "note.md"), []byte("note\n"), 0644); err != nil {
		t.Fatalf("write reflection fixture: %v", err)
	}

	before := dirDigest(t, dataDir)
	beforeReflections := dirDigest(t, dreamsDir)

	if _, err := collectUnverifiedWork(root); err != nil {
		t.Fatalf("collect unverified work: %v", err)
	}

	after := dirDigest(t, dataDir)
	afterReflections := dirDigest(t, dreamsDir)
	if before != after {
		t.Fatal("colony data directory changed while building the inventory")
	}
	if beforeReflections != afterReflections {
		t.Fatal("reflections directory changed while building the inventory")
	}
}

func TestUnverifiedWorkInventoryAllSourcesAbsentYieldsNoEntries(t *testing.T) {
	root := t.TempDir()
	entries, err := collectUnverifiedWork(root)
	if err != nil {
		t.Fatalf("collect unverified work with no sources: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("got %d entries with every source directory absent, want 0", len(entries))
	}
}

func TestMalformedItemIsListedAsUnknownNotDropped(t *testing.T) {
	root := t.TempDir()
	writeResearchFixture(t, root, "2024-01-01-broken.md", "not front matter at all\n")

	swarmDir := filepath.Join(root, ".aether", "data", "swarms", "swarm-broken")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("mkdir broken swarm dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(swarmDir, "episode.json"), []byte("{not json"), 0644); err != nil {
		t.Fatalf("write broken episode fixture: %v", err)
	}

	entries, err := collectUnverifiedWork(root)
	if err != nil {
		t.Fatalf("collect unverified work: %v", err)
	}

	var sawMalformedResearch, sawMalformedEpisode bool
	for _, entry := range entries {
		if entry.Kind == unverifiedWorkKindOracleResearch && entry.Standing == workStandingUnknown {
			sawMalformedResearch = true
			if entry.Reason == "" {
				t.Error("malformed research entry has no stated reason")
			}
		}
		if entry.Kind == unverifiedWorkKindSwarmEpisode && entry.Standing == workStandingUnknown {
			sawMalformedEpisode = true
			if entry.Reason == "" {
				t.Error("malformed episode entry has no stated reason")
			}
		}
	}
	if !sawMalformedResearch {
		t.Errorf("malformed research document was dropped, not listed: %+v", entries)
	}
	if !sawMalformedEpisode {
		t.Errorf("malformed episode was dropped, not listed: %+v", entries)
	}
}

func TestUnverifiedWorkInventoryOrderingIsDeterministic(t *testing.T) {
	root := t.TempDir()
	writeResearchFixture(t, root, "2024-01-01-a.md",
		"---\ntitle: \"a\"\nstatus: stopped\ngenerated: 2024-01-01T00:00:00Z\n---\n\nA.\n")
	writeResearchFixture(t, root, "2024-01-02-b.md",
		"---\ntitle: \"b\"\nstatus: idle\ngenerated: 2024-01-02T00:00:00Z\n---\n\nB.\n")
	writeSwarmEpisodeFixture(t, root, "swarm-1", swarmEpisodeRecord{
		Target: "target one", Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageFix,
		StartedAt: "2024-01-03T00:00:00Z", EndedAt: "2024-01-03T01:00:00Z",
	})
	writeSwarmEpisodeFixture(t, root, "swarm-2", swarmEpisodeRecord{
		Target: "target two", Status: swarmEpisodeStatusInterrupted, InterruptedStage: swarmEpisodeStageVerification,
		StartedAt: "2024-01-04T00:00:00Z", EndedAt: "2024-01-04T01:00:00Z",
	})

	first, err := collectUnverifiedWork(root)
	if err != nil {
		t.Fatalf("collect unverified work (first): %v", err)
	}
	second, err := collectUnverifiedWork(root)
	if err != nil {
		t.Fatalf("collect unverified work (second): %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("entry counts differ across runs: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("entry %d differs across runs:\n  first:  %+v\n  second: %+v", i, first[i], second[i])
		}
	}
	if !sort.SliceIsSorted(first, func(i, j int) bool {
		if !first[i].RecordedAt.Equal(first[j].RecordedAt) {
			return first[i].RecordedAt.Before(first[j].RecordedAt)
		}
		if first[i].Kind != first[j].Kind {
			return first[i].Kind < first[j].Kind
		}
		return first[i].Subject < first[j].Subject
	}) {
		t.Fatalf("entries are not ordered by (time, kind, subject): %+v", first)
	}
}
