package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/events"
)

// TestWatchReplaysTheMostRecentEpisode drives three real episodes through
// the actual emission boundary (emitColonyLive) and proves
// buildReplayWatchResult describes the one whose own start event is the
// chronologically latest -- with a genuine timestamp gap separating the
// oldest episode from the other two, and a deliberately engineered
// same-second sequence tie between the remaining two, so the higher
// sequence number is what breaks it.
func TestWatchReplaysTheMostRecentEpisode(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	// Episode C (oldest): closes cleanly, then the clock genuinely moves
	// forward a full second before anything else is recorded, so its start
	// timestamp is provably earlier than A's and B's -- not merely earlier
	// in file order.
	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "episode-c", EpisodeKind: "build", Status: "starting"})
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "episode-c", EpisodeKind: "build", Status: "completed"})
	time.Sleep(1100 * time.Millisecond)

	// Episode A: its own first-ever event IS its episode-level start
	// boundary, so it carries sequence 1.
	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "episode-a", EpisodeKind: "continue", Status: "starting"})
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "episode-a", EpisodeKind: "continue", Status: "completed"})

	// Episode B: a non-boundary event is emitted first (consuming sequence
	// 1), so its own wave-level start boundary lands on sequence 2 -- same
	// second as A's start event, higher sequence.
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID: "episode-b", EpisodeKind: "swarm", WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout",
	})
	emitColonyLive(events.LiveTopicWaveStarted, events.ColonyLivePayload{EpisodeID: "episode-b", EpisodeKind: "swarm", Wave: 1})
	emitColonyLive(events.LiveTopicWorkerFinished, events.ColonyLivePayload{
		EpisodeID: "episode-b", EpisodeKind: "swarm", WorkerID: "Scout-1", WorkerName: "Scout-1", Caste: "scout", Status: "completed",
	})
	emitColonyLive(events.LiveTopicWaveEnded, events.ColonyLivePayload{EpisodeID: "episode-b", EpisodeKind: "swarm", Wave: 1, Status: "completed"})

	ctx := context.Background()
	result := buildReplayWatchResult(ctx, root, s, time.Now().UTC())
	if got := stringValue(result["episode_id"]); got != "episode-b" {
		t.Fatalf("buildReplayWatchResult picked episode %q, want %q (latest start timestamp, tie broken by the higher sequence number)", got, "episode-b")
	}
	if got := stringValue(result["episode_kind"]); got != "swarm" {
		t.Fatalf("episode_kind = %q, want %q", got, "swarm")
	}
	if got := intValue(result["worker_count"]); got != 1 {
		t.Fatalf("worker_count = %d, want 1", got)
	}
	if boolValue(result["interrupted"]) {
		t.Fatalf("episode-b was recorded closed (wave.ended emitted) but result reports interrupted")
	}
	if got := stringValue(result["outcome"]); got != "completed" {
		t.Fatalf("outcome = %q, want %q", got, "completed")
	}

	rendered := renderReplayWatchVisual(result)
	for _, want := range []string{"episode-b", "swarm", "nothing is running now"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered replay visual missing %q:\n%s", want, rendered)
		}
	}
	if strings.Contains(rendered, "episode-c") || strings.Contains(rendered, "episode-a") {
		t.Errorf("rendered replay visual named an episode other than the most recently started one:\n%s", rendered)
	}
}

// TestReplaySummaryCostComesFromTheLedger proves the rendered cost block
// equals renderSpendCostLineFromLedgers for the exact ledgers
// loadSpendLedgersForPhase resolves -- never a figure computed by this
// summary itself.
func TestReplaySummaryCostComesFromTheLedger(t *testing.T) {
	saveGlobals(t)
	_, root := seedRunFixture(t, 1)

	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "cost-episode", EpisodeKind: "build", Status: "starting"})
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "cost-episode", EpisodeKind: "build", Status: "completed"})

	ledger := spendLedger{
		Phase: 1, Workflow: spendWorkflowBuild,
		Rows: []spendRow{{AgentName: "Mason-67", Caste: "builder", Status: "completed", Usage: codex.WorkerUsage{InputTokens: 1000, Source: codex.UsageSourceProvider}}},
	}
	if err := saveSpendLedger(ledger); err != nil {
		t.Fatalf("save spend ledger: %v", err)
	}

	ctx := context.Background()
	result := buildReplayWatchResult(ctx, root, store, time.Now().UTC())
	rendered := renderReplayWatchVisual(result)

	ledgers, ok := loadSpendLedgersForPhase(1)
	if !ok {
		t.Fatalf("loadSpendLedgersForPhase(1) ok=false")
	}
	costBlock := renderSpendCostLineFromLedgers(ledgers)
	if !strings.HasSuffix(rendered, costBlock) {
		t.Fatalf("rendered replay visual does not end with the exact cost block:\n rendered: %q\n costBlock: %q", rendered, costBlock)
	}
}

// TestReplaySummaryOfInterruptedEpisodeSaysInterrupted proves an episode
// with a start event and no matching end renders as interrupted, naming
// the last recorded transition and its time -- never as finished.
func TestReplaySummaryOfInterruptedEpisodeSaysInterrupted(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "interrupted-episode", EpisodeKind: "build", Status: "starting"})
	emitColonyLive(events.LiveTopicWorkerStarted, events.ColonyLivePayload{
		EpisodeID: "interrupted-episode", EpisodeKind: "build", WorkerID: "Mason-1", WorkerName: "Mason-1", Caste: "builder",
	})
	// No episode.ended is ever emitted -- the episode was interrupted.

	ctx := context.Background()
	result := buildReplayWatchResult(ctx, root, s, time.Now().UTC())
	if !boolValue(result["interrupted"]) {
		t.Fatalf("episode with no end event reports interrupted=false")
	}
	if got := stringValue(result["outcome"]); got != "" {
		t.Fatalf("interrupted episode carries an outcome %q -- it must never claim a terminal status it never recorded", got)
	}
	if got := stringValue(result["last_transition"]); got == "" {
		t.Fatalf("interrupted episode names no last recorded transition")
	}
	if got := stringValue(result["last_transition_at"]); got == "" {
		t.Fatalf("interrupted episode names no timestamp for its last recorded transition")
	}

	rendered := renderReplayWatchVisual(result)
	if !strings.Contains(rendered, "interrupted") {
		t.Errorf("rendered replay visual does not say interrupted:\n%s", rendered)
	}
	if strings.Contains(rendered, "Outcome: completed") {
		t.Errorf("rendered replay visual describes the interrupted episode as completed:\n%s", rendered)
	}
}

// TestReplaySummaryNextCommandComesFromTheProjection proves the next
// command equals the lifecycle projection's own next action for the exact
// same fixture state -- never a command chosen locally by this summary.
func TestReplaySummaryNextCommandComesFromTheProjection(t *testing.T) {
	saveGlobals(t)
	_, root := seedRunFixture(t, 1)

	emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "next-command-episode", EpisodeKind: "build", Status: "starting"})
	emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "next-command-episode", EpisodeKind: "build", Status: "completed"})

	ctx := context.Background()
	now := time.Now().UTC()
	result := buildReplayWatchResult(ctx, root, store, now)

	facts, err := loadLifecycleFacts(root, store, now)
	if err != nil {
		facts = unavailableLifecycleFacts(root, now, err.Error())
	}
	projection := projectLifecycle(facts, LifecycleViewCompact, detectPlatform())
	want := lifecycleStatusActionCommand(projection.NextAction)

	if got := stringValue(result["next"]); got != want {
		t.Fatalf("next command = %q, want the lifecycle projection's own next action %q", got, want)
	}
	if want != "" && !strings.Contains(renderReplayWatchVisual(result), want) {
		t.Errorf("rendered replay visual does not carry the next command %q", want)
	}
}

// TestWatchResolvesThreeBranchesFromEvidenceAlone proves the three-way
// watch-mode decision is made entirely from recorded evidence in Go: an
// open episode resolves live; an episode left open by a run that has since
// terminated resolves replay (never live); episodes that are all closed
// resolve replay; no episode at all resolves idle; and no environment
// variable changes which branch a fixture resolves to.
func TestWatchResolvesThreeBranchesFromEvidenceAlone(t *testing.T) {
	t.Run("open episode resolves live", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "open-ep", EpisodeKind: "build", Status: "starting"})

		mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeLive {
			t.Fatalf("mode = %q, want %q", mode, watchModeLive)
		}
		if !snapshot.Open {
			t.Fatalf("snapshot.Open = false for a genuinely open episode")
		}
	})

	t.Run("closed-only episode resolves replay", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "closed-ep", EpisodeKind: "build", Status: "starting"})
		emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "closed-ep", EpisodeKind: "build", Status: "completed"})

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeReplay {
			t.Fatalf("mode = %q, want %q", mode, watchModeReplay)
		}
	})

	t.Run("open episode whose owning run has terminated resolves replay, not live", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "abandoned-ep", EpisodeKind: "build", Status: "starting"})
		writeSpawnRunFixture(t, s.BasePath(), "run-1", "failed")

		mode, snapshot := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeReplay {
			t.Fatalf("mode = %q, want %q (start boundary with no end, but the owning run already terminated)", mode, watchModeReplay)
		}
		if !snapshot.Open {
			t.Fatalf("snapshot.Open = false; the replayed boundary balance itself must be unaffected by the mode decision")
		}
	})

	t.Run("no episode at all resolves idle", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s

		mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if mode != watchModeIdle {
			t.Fatalf("mode = %q, want %q", mode, watchModeIdle)
		}
	})

	t.Run("no environment variable changes the branch a fixture resolves to", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "env-proof-ep", EpisodeKind: "build", Status: "starting"})
		emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "env-proof-ep", EpisodeKind: "build", Status: "completed"})

		base, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
		if base != watchModeReplay {
			t.Fatalf("fixture setup broken: base mode = %q, want %q", base, watchModeReplay)
		}

		for _, env := range [][2]string{
			{"AETHER_OUTPUT_MODE", "visual"},
			{"AETHER_FORCE_VISUAL", "1"},
			{"AETHER_WATCH_MODE", "live"},
			{"AETHER_LIVE", "1"},
		} {
			t.Setenv(env[0], env[1])
			mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC())
			if mode != base {
				t.Fatalf("setting %s=%s changed the resolved watch mode from %q to %q", env[0], env[1], base, mode)
			}
		}
	})
}

// TestWatchIsReadOnlyInEveryBranch compares a digest of the colony data
// directory before and after each of the three branches renders, proving
// the read-only guarantee (watchCmd's aether.io/read-only annotation) holds
// structurally for every branch, not just the idle one.
func TestWatchIsReadOnlyInEveryBranch(t *testing.T) {
	runOnce := func(t *testing.T) {
		t.Helper()
		var buf bytes.Buffer
		stdout = &buf
		c := newTestWatchCmd()
		c.Flags().Set("once", "true")
		if err := runWatchCommand(c, nil); err != nil {
			t.Fatalf("runWatchCommand returned an error: %v", err)
		}
	}

	t.Run("live branch", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "ro-live", EpisodeKind: "build", Status: "starting"})
		if mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC()); mode != watchModeLive {
			t.Fatalf("fixture setup broken: mode = %q, want %q", mode, watchModeLive)
		}

		before := dirDigest(t, s.BasePath())
		runOnce(t)
		after := dirDigest(t, s.BasePath())
		if before != after {
			t.Fatalf("live branch mutated the colony data directory: %s -> %s", before, after)
		}
	})

	t.Run("replay branch", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		emitColonyLive(events.LiveTopicEpisodeStarted, events.ColonyLivePayload{EpisodeID: "ro-replay", EpisodeKind: "build", Status: "starting"})
		emitColonyLive(events.LiveTopicEpisodeEnded, events.ColonyLivePayload{EpisodeID: "ro-replay", EpisodeKind: "build", Status: "completed"})
		if mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC()); mode != watchModeReplay {
			t.Fatalf("fixture setup broken: mode = %q, want %q", mode, watchModeReplay)
		}

		before := dirDigest(t, s.BasePath())
		runOnce(t)
		after := dirDigest(t, s.BasePath())
		if before != after {
			t.Fatalf("replay branch mutated the colony data directory: %s -> %s", before, after)
		}
	})

	t.Run("idle branch", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		if mode, _ := resolveWatchMode(context.Background(), s, time.Now().UTC()); mode != watchModeIdle {
			t.Fatalf("fixture setup broken: mode = %q, want %q", mode, watchModeIdle)
		}

		before := dirDigest(t, s.BasePath())
		runOnce(t)
		after := dirDigest(t, s.BasePath())
		if before != after {
			t.Fatalf("idle branch mutated the colony data directory: %s -> %s", before, after)
		}
	})
}
