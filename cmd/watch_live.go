package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/storage"
)

// watchMode is one of the three shapes `aether watch` can render.
type watchMode string

const (
	// watchModeLive means an episode is currently open (a started boundary
	// with no matching ended boundary yet) -- rendered by
	// renderLiveWatchVisual from a replayed colonyLiveSnapshot.
	watchModeLive watchMode = "live"
	// watchModeReplay means a persisted live episode exists but is closed.
	// Plan 202-09 owns rendering this branch; until then it falls through to
	// the honest idle floor, unchanged.
	watchModeReplay watchMode = "replay"
	// watchModeIdle means no live-colony evidence exists at all -- rendered
	// by the existing buildIdleWatchResult / renderIdleWatchVisual pair.
	watchModeIdle watchMode = "idle"
)

// resolveWatchMode is the one place a platform wrapper's `watch` command
// defers to for which of the three watch modes applies. It is resolved in
// Go, from persisted evidence, never guessed by a wrapper.
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
	if snapshot.Open {
		return watchModeLive, snapshot
	}
	return watchModeReplay, snapshot
}

// latestLiveEpisodeID finds the episode ID the chronologically-latest
// persisted live.* event belongs to. Returns false when no live events have
// ever been recorded. Reads via readColonyLiveEventsRaw -- lock-free, since
// this runs on every `aether watch` invocation including the idle path.
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
	episodeID := decoded[len(decoded)-1].payload.EpisodeID
	return episodeID, episodeID != ""
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
		"schema_version": LifecycleResultSchemaVersion,
		"mode":           "live_watch",
		"command":        "watch",
		"live_capability": "supported",
		"active_count":    len(snapshot.Workers),
		"captured_at":     now.UTC().Format(time.RFC3339Nano),
		"snapshot":        snapshot,
	}
}
