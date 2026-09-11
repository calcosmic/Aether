package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// watchMode is one of the three shapes `aether watch` can render.
type watchMode string

const (
	// watchModeLive means an episode is currently open (a started boundary
	// with no matching ended boundary yet, AND the run that owns it has not
	// itself already reached a terminal status) -- rendered by
	// renderLiveWatchVisual from a replayed colonyLiveSnapshot.
	watchModeLive watchMode = "live"
	// watchModeReplay means no episode is genuinely open (either every
	// recorded episode closed cleanly, or the most recent one was left open
	// by a run that has since terminated) but at least one episode was
	// recorded -- rendered by buildReplayWatchResult / renderReplayWatchVisual
	// (plan 202-09).
	watchModeReplay watchMode = "replay"
	// watchModeIdle means no live-colony evidence exists at all -- rendered
	// by the existing buildIdleWatchResult / renderIdleWatchVisual pair.
	watchModeIdle watchMode = "idle"
)

// resolveWatchMode is the one place a platform wrapper's `watch` command
// defers to for which of the three watch modes applies. It is resolved in
// Go, from persisted evidence, never guessed by a wrapper: no flag,
// environment variable, or wrapper argument can select a branch the
// recorded evidence does not support.
func resolveWatchMode(ctx context.Context, s *storage.Store, now time.Time) (watchMode, colonyLiveSnapshot) {
	if s == nil {
		return watchModeIdle, colonyLiveSnapshot{}
	}

	episodeID, ok := latestLiveEpisodeID(ctx, s)
	if !ok {
		return watchModeIdle, colonyLiveSnapshot{}
	}

	snapshot, err := replayColonyLiveSnapshot(ctx, s, episodeID, time.Time{})
	if err != nil || snapshot.EpisodeID == "" {
		return watchModeIdle, colonyLiveSnapshot{}
	}
	snapshot = applyUnfinishedWorkerInterruption(s, snapshot)

	// A start boundary with no matching end normally means "still running"
	// -- but only while the run that dispatched it is itself still alive.
	// Once that durable run record reaches a terminal status (the exact
	// rule applyUnfinishedWorkerInterruption already applies to an
	// individual unfinished worker row), the episode is not genuinely open
	// anymore; it was left open by a process that has since ended, so it
	// belongs to the replay branch (as an interrupted episode), never the
	// live one. An absent run record is NOT treated as terminated -- a
	// start boundary with nothing to contradict it is still genuinely open.
	if snapshot.Open && !colonyLiveEpisodeAbandoned(s, snapshot) {
		return watchModeLive, snapshot
	}
	return watchModeReplay, snapshot
}

// colonyLiveEpisodeAbandoned reports whether the process that owns an open
// episode has gone away -- the same question colonyLiveEpisodeRunHasTerminated
// already answers for every dispatch-tree lane (build, continue, plan,
// Swarm), asked here per episode kind instead of unconditionally.
//
// Oracle never registers a spawn run of its own -- its owning process is its
// own controller PID, tracked in its own durable state file, not in
// spawn-runs.json -- so for an Oracle episode this delegates to
// oracleLiveEpisodeAbandoned, which reads that state instead. Every other
// episode kind keeps colonyLiveEpisodeRunHasTerminated's rule byte-for-byte,
// so build/continue/plan/Swarm behaviour (TestWatchResolvesThreeBranchesFromEvidenceAlone)
// is completely unaffected by this dispatch (202-17, CR-01's second half).
func colonyLiveEpisodeAbandoned(s *storage.Store, snapshot colonyLiveSnapshot) bool {
	if snapshot.EpisodeKind == events.EpisodeKindOracle {
		return oracleLiveEpisodeAbandoned(snapshot.EpisodeID)
	}
	return colonyLiveEpisodeRunHasTerminated(s)
}

// oracleLiveEpisodeAbandoned is Oracle's own instance of the identical
// question colonyLiveEpisodeRunHasTerminated answers for every other lane:
// has the process that owns this episode gone away? It stays read-only --
// loadOracleStateFile is a plain os.ReadFile that takes no lock, and
// oracleProcessExists only inspects the process table, exactly what
// `oracle status --follow` already does on every poll -- so calling it from
// `aether watch` adds no new mutation risk.
//
// The repository root is resolved the same way every other read-only
// Oracle-adjacent helper in this package does (resolveAetherRootPath,
// store-derived), so this works against a test store with no environment
// variable set. An unreadable state file (never started, or genuinely
// absent) returns false: absent evidence is not evidence of termination,
// exactly as the existing spawn-run rule already documents for every other
// lane.
func oracleLiveEpisodeAbandoned(episodeID string) bool {
	root := resolveAetherRootPath()
	state, err := loadOracleStateFile(oracleWorkspacePaths(root).StatePath)
	if err != nil {
		return false
	}
	if oracleLiveEpisodeID(state) != episodeID {
		// The durable state has moved on to a different run than the one
		// this episode names -- this episode's own owning run is gone.
		return true
	}
	status := strings.TrimSpace(state.Status)
	if !strings.EqualFold(status, "active") && !strings.EqualFold(status, "planned") {
		return true
	}
	return oracleStateHasStaleController(state)
}

// colonyLiveEpisodeRunHasTerminated reports whether the durable spawn-run
// record (read lock-free via latestSpawnRunRaw, mirroring
// applyUnfinishedWorkerInterruption's own read) has already reached a
// terminal status (agent.IsTerminalSpawnStatus). This is the episode-level
// application of the identical rule the dashboard already uses to
// reclassify an unfinished worker as interrupted, so the live/replay
// boundary and the dashboard's own worker-level boundary can never
// disagree about what "still running" means.
func colonyLiveEpisodeRunHasTerminated(s *storage.Store) bool {
	run, ok := latestSpawnRunRaw(s)
	return ok && agent.IsTerminalSpawnStatus(run.Status)
}

// colonyLiveSpawnRunFile is the persisted JSON filename the spawn tree's
// run tracker writes, relative to the store's base path. Matches
// agent.SpawnTree's own internal defaultSpawnRunFile constant
// (unexported in pkg/agent), duplicated here deliberately: reading it
// through agent.NewSpawnTree(...).CurrentRun() goes through
// storage.Store.ReadFile, which takes an RLock and creates a lock file on
// every `aether watch` invocation -- exactly the side effect
// readColonyLiveEventsRaw's own doc comment already refuses for the same
// reason (TestWatchIdle199ReadOnly / this plan's own read-only guarantee).
const colonyLiveSpawnRunFile = "spawn-runs.json"

// colonyLiveSpawnRunState mirrors the persisted shape of spawn-runs.json --
// agent.SpawnTree's own current-run-ID plus its bounded run history --
// closely enough to read run.Status without importing the unexported type.
type colonyLiveSpawnRunState struct {
	CurrentRunID string           `json:"current_run_id,omitempty"`
	Runs         []agent.SpawnRun `json:"runs,omitempty"`
}

// latestSpawnRunRaw reads spawn-runs.json directly via os.ReadFile
// (lock-free, mirroring readColonyLiveEventsRaw) and returns the run named
// by CurrentRunID, or -- when that ID does not resolve -- the most recently
// recorded run, matching agent.SpawnTree.CurrentRun()'s own fallback. An
// absent file, an unparseable file, or an empty run history all resolve to
// (agent.SpawnRun{}, false) -- never an error, since this runs on every
// `aether watch` invocation.
func latestSpawnRunRaw(s *storage.Store) (agent.SpawnRun, bool) {
	if s == nil {
		return agent.SpawnRun{}, false
	}
	path := filepath.Join(s.BasePath(), colonyLiveSpawnRunFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return agent.SpawnRun{}, false
	}
	var state colonyLiveSpawnRunState
	if err := json.Unmarshal(data, &state); err != nil {
		return agent.SpawnRun{}, false
	}
	for _, run := range state.Runs {
		if run.ID != "" && run.ID == state.CurrentRunID {
			return run, true
		}
	}
	if len(state.Runs) == 0 {
		return agent.SpawnRun{}, false
	}
	return state.Runs[len(state.Runs)-1], true
}

// applyUnfinishedWorkerInterruption reclassifies every still-open worker
// row (Finished == false) as interrupted, naming the owning run's own
// terminal status as the reason, once that durable run record itself
// reaches a terminal status (agent.IsTerminalSpawnStatus) -- the identical
// vocabulary buildIdleWatchResult already applies to a spawn-tree actor's
// own status. A worker is NEVER shown as completed by this function; that
// can only ever come from its own worker.finished live event (Finished ==
// true), which this function leaves untouched.
//
// When no run record exists, or the current run is still active, snapshot
// is returned unchanged -- an open worker with an active run is simply
// still running.
func applyUnfinishedWorkerInterruption(s *storage.Store, snapshot colonyLiveSnapshot) colonyLiveSnapshot {
	if len(snapshot.Workers) == 0 {
		return snapshot
	}
	run, ok := latestSpawnRunRaw(s)
	if !ok || !agent.IsTerminalSpawnStatus(run.Status) {
		return snapshot
	}
	for i := range snapshot.Workers {
		if snapshot.Workers[i].Finished {
			continue
		}
		snapshot.Workers[i].Status = "interrupted"
		snapshot.Workers[i].InterruptedReason = run.Status
	}
	return snapshot
}

// latestLiveEpisodeID resolves the episode `aether watch`'s live branch
// follows: the most recently started episode that is STILL OPEN (its own
// start/end boundary balance, per colonyLiveBoundaryDelta and
// openColonyLiveEpisodeIDs, is currently positive), falling back to the
// most recently started episode among ALL recorded episodes when none is
// open -- exactly the same "latest started" selection
// mostRecentlyStartedLiveEpisode (cmd/watch_replay.go) uses for the replay
// branch, via the shared latestStartedLiveEpisodeAmong. With nothing open,
// this function and mostRecentlyStartedLiveEpisode therefore always name
// the same episode, so the live and replay paths can never disagree about
// what "just ran" means (D-03).
//
// This is deliberately independent of which episode owns the single
// chronologically newest EVENT: a recovery decision (or any other detail)
// recorded against an older, still-open episode must not cause a newer,
// already-closed episode to be preferred just because it happens to own
// the most recent timestamp -- "live" means the newest STILL-OPEN episode,
// never the newest event.
//
// Returns false only when no episode ID resolves at all. Reads via
// readColonyLiveEventsRaw -- lock-free, since this runs on every `aether
// watch` invocation including the idle path.
func latestLiveEpisodeID(ctx context.Context, s *storage.Store) (string, bool) {
	_ = ctx
	raw := readColonyLiveEventsRaw(s, time.Time{})
	if len(raw) == 0 {
		return "", false
	}
	decoded := decodeColonyLiveEvents(raw)
	if len(decoded) == 0 {
		return "", false
	}
	sortColonyLiveEntries(decoded)

	seen := map[string]bool{}
	episodeIDs := make([]string, 0, len(decoded))
	for _, entry := range decoded {
		id := entry.payload.EpisodeID
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		episodeIDs = append(episodeIDs, id)
	}
	if len(episodeIDs) == 0 {
		return "", false
	}

	if openSet := openColonyLiveEpisodeIDs(decoded); len(openSet) > 0 {
		openIDs := make([]string, 0, len(openSet))
		for _, id := range episodeIDs {
			if openSet[id] {
				openIDs = append(openIDs, id)
			}
		}
		if best := latestStartedLiveEpisodeAmong(decoded, openIDs); best != "" {
			return best, true
		}
	}

	best := latestStartedLiveEpisodeAmong(decoded, episodeIDs)
	return best, best != ""
}

// renderLiveWatchVisual renders a colonyLiveSnapshot as the live watch
// screen: episode identity, current wave, and every worker's identity,
// caste, and status.
func renderLiveWatchVisual(snapshot colonyLiveSnapshot) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("watch"), "Watch"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Live colony activity -- episode %s (%s)\n", emptyFallback(snapshot.EpisodeID, "unknown"), emptyFallback(snapshot.EpisodeKind, "unknown"))
	if snapshot.Wave > 0 {
		fmt.Fprintf(&b, "Wave %d\n", snapshot.Wave)
	}
	if snapshot.StartedAt != "" {
		fmt.Fprintf(&b, "Started: %s | Elapsed: %.0fs\n", snapshot.StartedAt, snapshot.ElapsedSeconds)
	}
	b.WriteString("\n")

	if len(snapshot.Workers) == 0 {
		b.WriteString("No workers active in this episode yet.\n")
	} else {
		for _, w := range snapshot.Workers {
			fmt.Fprintf(&b, "%s (%s) -- wave %d -- %s\n", casteIdentity(w.Caste), w.WorkerName, w.Wave, emptyFallback(w.Status, "active"))
			if w.Question != "" {
				fmt.Fprintf(&b, "  question: %s\n", w.Question)
			}
			for _, finding := range w.Findings {
				fmt.Fprintf(&b, "  finding: %s\n", finding)
			}
		}
	}

	if snapshot.Confidence > 0 || snapshot.TargetConfidence > 0 {
		fmt.Fprintf(&b, "\nConfidence: %.2f / target %.2f\n", snapshot.Confidence, snapshot.TargetConfidence)
	}
	if len(snapshot.Contradictions) > 0 {
		b.WriteString("\nContradictions:\n")
		for _, c := range snapshot.Contradictions {
			fmt.Fprintf(&b, "  - %s\n", c)
		}
	}
	if snapshot.RecoveryState != "" {
		fmt.Fprintf(&b, "\nRecovery state: %s\n", snapshot.RecoveryState)
	}

	return b.String()
}

// liveWatchResult builds the JSON envelope for the live watch screen,
// mirroring buildIdleWatchResult's shape (schema_version/mode/command) so
// downstream consumers of `aether watch --json` see a consistent envelope
// regardless of which mode resolved.
func liveWatchResult(snapshot colonyLiveSnapshot, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"schema_version":  LifecycleResultSchemaVersion,
		"mode":            "live_watch",
		"command":         "watch",
		"live_capability": "supported",
		"active_count":    len(snapshot.Workers),
		"captured_at":     now.UTC().Format(time.RFC3339Nano),
		"snapshot":        snapshot,
	}
}

// colonyLiveRefreshClearSequence is the in-place redraw control sequence
// written before every frame after the first in the refresh loop: move the
// cursor home and clear the screen, so each new frame replaces the
// previous one on screen rather than appending below it.
const colonyLiveRefreshClearSequence = "\x1b[H\x1b[2J"

// colonyLiveRefreshTickerOverride lets a test drive the refresh loop off a
// channel and stop function it controls instead of a real wall-clock
// ticker, so a test can prove three redraws happen without a real sleep.
// nil (unset) in production.
var colonyLiveRefreshTickerOverride func(time.Duration) (<-chan time.Time, func())

// newColonyLiveRefreshTicker returns the tick source the refresh loop reads
// from -- a real time.Ticker in production, or a test's own override.
func newColonyLiveRefreshTicker(interval time.Duration) (<-chan time.Time, func()) {
	if colonyLiveRefreshTickerOverride != nil {
		return colonyLiveRefreshTickerOverride(interval)
	}
	t := time.NewTicker(interval)
	return t.C, t.Stop
}

// runWatchCommand is watchCmd's RunE body. It renders exactly one frame
// and, unless the single-snapshot flag is set or the output is not going
// to a human-readable surface, redraws in place on the configured interval
// until interrupted (SIGINT/SIGTERM). Every read the loop performs
// (resolveWatchMode, the idle floor, the spawn-run lookup, the cost
// ledger) is a plain read already proven read-only elsewhere in this
// package -- the loop itself creates no lock file, writes no state, and
// starts no store write path.
func runWatchCommand(cmd *cobra.Command, args []string) error {
	once, _ := cmd.Flags().GetBool("once")
	interval, _ := cmd.Flags().GetDuration("interval")
	if interval <= 0 {
		interval = 2 * time.Second
	}
	drill := parseColonyLiveDrillSelector(args)

	// The redraw loop is gated on isTerminalWriter, not
	// shouldRenderVisualOutput: the latter is also true for a forced/piped
	// "pretty" render (AETHER_FORCE_VISUAL, AETHER_OUTPUT_MODE=visual, a
	// test's own buffer) that wants exactly one frame back, not an
	// unbounded loop that never returns until a real terminal's owner
	// interrupts it. Only a genuine TTY gets the live redraw.
	if once || !isTerminalWriter(stdout) {
		writeColonyWatchFrame(context.Background(), drill, time.Now().UTC(), true)
		return nil
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	return runColonyLiveRefreshLoop(ctx, drill, interval)
}

// colonyLiveHideCursorSequence / colonyLiveShowCursorSequence bracket the
// refresh loop's whole run: the cursor is hidden once, before the first
// frame, and restored once, when the loop returns for any reason --
// including an interrupt -- so the terminal is left in a usable state
// rather than showing a cursor blinking mid-frame.
const (
	colonyLiveHideCursorSequence = "\x1b[?25l"
	colonyLiveShowCursorSequence = "\x1b[?25h"
)

// runColonyLiveRefreshLoop renders one frame immediately, then redraws in
// place on every tick from newColonyLiveRefreshTicker(interval) until ctx
// is done. An interrupt (ctx cancelled) is a clean stop -- it returns nil,
// never an error -- and always restores the cursor via its deferred write,
// however the loop exits. Split out of runWatchCommand so a test can drive
// it from its own cancellable context instead of sending a real OS signal.
//
// Every frame is written through the package's own `stdout` var (the same
// writer every other command in this package uses), never a writer passed
// in separately -- a test drives this by reassigning `stdout`, matching
// the existing convention (saveGlobals / TestResumeDashboard and friends),
// rather than by threading a second, potentially-divergent writer through
// the envelope path.
func runColonyLiveRefreshLoop(ctx context.Context, drill colonyLiveDrillSelector, interval time.Duration) error {
	fmt.Fprint(stdout, colonyLiveHideCursorSequence)
	defer fmt.Fprint(stdout, colonyLiveShowCursorSequence)

	writeColonyWatchFrame(ctx, drill, time.Now().UTC(), true)

	ticker, stopTicker := newColonyLiveRefreshTicker(interval)
	defer stopTicker()
	for {
		select {
		case <-ctx.Done():
			return nil
		case tick, ok := <-ticker:
			if !ok {
				return nil
			}
			fmt.Fprint(stdout, colonyLiveRefreshClearSequence)
			writeColonyWatchFrame(ctx, drill, tick.UTC(), false)
		}
	}
}

// writeColonyWatchFrame renders and writes exactly one watch frame to
// `stdout`: the live dashboard when an episode is open, the honest idle
// floor otherwise.
//
// useEnvelope selects between the full JSON+visual envelope
// (outputWorkflow, used for the first/only frame so `aether watch --once`
// and `aether watch --json` keep their existing shape) and a bare visual
// write (writeVisualOutput, used for every later redraw in the loop, where
// emitting a fresh JSON document on every tick would not be meaningful).
func writeColonyWatchFrame(ctx context.Context, drill colonyLiveDrillSelector, now time.Time, useEnvelope bool) {
	mode, snapshot := resolveWatchMode(ctx, store, now)
	switch mode {
	case watchModeLive:
		state, _ := readColonyStateWithoutWriting()
		ledgers, _ := loadSpendLedgersForPhase(state.CurrentPhase)
		visual := renderColonyLiveDashboard(snapshot, state, ledgers, drill)
		if useEnvelope {
			outputWorkflow(liveWatchResult(snapshot, now), visual)
			return
		}
		writeVisualOutput(stdout, visual)
	case watchModeReplay:
		// No episode is genuinely open (every recorded episode closed
		// cleanly, or the most recent one was left open by a run that has
		// since terminated) but at least one was recorded -- the
		// replay-backed summary of the most recently started one (plan
		// 202-09), never the honest idle floor below, which is reserved for
		// "nothing has ever run at all".
		result := buildReplayWatchResult(ctx, resolveAetherRoot(), store, now)
		visual := renderReplayWatchVisual(result)
		if useEnvelope {
			outputWorkflow(result, visual)
			return
		}
		writeVisualOutput(stdout, visual)
	default:
		// watchModeIdle: no live-colony evidence exists at all.
		result := buildIdleWatchResult(resolveAetherRoot(), store, now)
		visual := renderIdleWatchVisual(result)
		if useEnvelope {
			outputWorkflow(result, visual)
			return
		}
		writeVisualOutput(stdout, visual)
	}
}
