package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

// spawnReapDefaultThresholdMinutes is what the reaper uses when colony state
// cannot be read, or carries no configured threshold, or carries one below 1
// minute. It is deliberately generous (D-16/D-17): this system has no
// liveness signal for a spawned helper, so a shorter default would reap
// slow-but-working helpers indistinguishably from abandoned ones.
const spawnReapDefaultThresholdMinutes = 120

// spawnReapThresholdMinutes reads the configured reap threshold from colony
// state. This is the one guard in this phase that fails toward NOT acting:
// a reaper that cannot read its own threshold must not start reaping on a
// guess, so every unreadable-state path here returns the generous default
// rather than a small or zero value that would make the reaper aggressive by
// accident.
func spawnReapThresholdMinutes() int {
	if store == nil {
		return spawnReapDefaultThresholdMinutes
	}
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return spawnReapDefaultThresholdMinutes
	}
	if state.SpawnReapThresholdMinutes == nil || *state.SpawnReapThresholdMinutes < 1 {
		return spawnReapDefaultThresholdMinutes
	}
	return *state.SpawnReapThresholdMinutes
}

// spawnReapEntryLastActivity returns the LATER of an entry's Timestamp and
// ActivityTimestamp, parsed as UTC. D-16 requires comparing against the
// later of the two, never the earlier: a refreshed activity stamp is the
// only evidence of life this system has, and reaping past it would kill
// live work on the strength of a stale spawn timestamp alone. A field that
// fails to parse is treated as absent rather than as "now" -- an entry
// cannot be judged stale against a timestamp that cannot be read.
func spawnReapEntryLastActivity(entry agent.SpawnEntry) time.Time {
	ts, tsErr := time.Parse(time.RFC3339, strings.TrimSpace(entry.Timestamp))
	act, actErr := time.Parse(time.RFC3339, strings.TrimSpace(entry.ActivityTimestamp))

	switch {
	case tsErr != nil && actErr != nil:
		return time.Time{}
	case tsErr != nil:
		return act.UTC()
	case actErr != nil:
		return ts.UTC()
	case act.After(ts):
		return act.UTC()
	default:
		return ts.UTC()
	}
}

// spawnReapCandidates parses the spawn tree once and returns every entry
// that is both currently live (agent.IsLiveSpawnStatus) and whose last
// activity is more than thresholdMinutes before now. This function only
// reads -- it never mutates the tree, which is what lets spawn-orphans'
// listing view (no --clear) call it directly with the D-14/CLAUDE.md
// dry-run guarantee that inspecting orphans never touches them.
func spawnReapCandidates(st *agent.SpawnTree, now time.Time, thresholdMinutes int) ([]agent.SpawnEntry, error) {
	entries, err := st.Parse()
	if err != nil {
		return nil, err
	}

	threshold := time.Duration(thresholdMinutes) * time.Minute
	var candidates []agent.SpawnEntry
	for _, entry := range entries {
		if !agent.IsLiveSpawnStatus(entry.Status) {
			continue
		}
		last := spawnReapEntryLastActivity(entry)
		if last.IsZero() {
			// An entry whose timestamps cannot be parsed cannot be judged
			// stale -- D-16's conservative bias applies here too: when the
			// evidence is unreadable, do not reap.
			continue
		}
		if now.Sub(last) > threshold {
			candidates = append(candidates, entry)
		}
	}
	return candidates, nil
}

// spawnReapStaleEntries is D-15/D-18's mutation half: it reads the
// configured threshold, finds every stale live entry, and marks each one
// SpawnStatusAbandoned via UpdateStatusPreserveActivity -- never the
// non-preserving UpdateStatus, because a reap must not look like fresh
// activity, which would move the entry back inside the run window and
// corrupt the whole-run budget count (D-18's entire point). Individual
// mutation failures are collected and continue to the next candidate rather
// than aborting the batch, so one unreadable entry never stops every other
// ghost from being cleared; every error encountered is joined into the
// returned error.
func spawnReapStaleEntries(st *agent.SpawnTree, now time.Time) ([]string, error) {
	thresholdMinutes := spawnReapThresholdMinutes()
	candidates, err := spawnReapCandidates(st, now, thresholdMinutes)
	if err != nil {
		return nil, err
	}

	var reaped []string
	var errs []error
	for _, entry := range candidates {
		elapsed := now.Sub(spawnReapEntryLastActivity(entry)).Round(time.Minute)
		summary := fmt.Sprintf(
			"marked abandoned: %s has passed since this helper started, past the %dm threshold, with no completion reported",
			elapsed, thresholdMinutes,
		)
		if err := st.UpdateStatusPreserveActivity(entry.AgentName, agent.SpawnStatusAbandoned, summary); err != nil {
			errs = append(errs, fmt.Errorf("reap %q: %w", entry.AgentName, err))
			continue
		}
		reaped = append(reaped, entry.AgentName)
	}

	return reaped, errors.Join(errs...)
}

// spawnOrphanEntryJSON is the machine-readable shape for one candidate
// orphan, used by both the listing and the --clear views.
type spawnOrphanEntryJSON struct {
	Name           string `json:"name"`
	Caste          string `json:"caste"`
	Parent         string `json:"parent"`
	Task           string `json:"task"`
	ElapsedMinutes int    `json:"elapsed_minutes"`
	LastActivity   string `json:"last_activity"`
}

func spawnOrphanEntriesJSON(entries []agent.SpawnEntry, now time.Time) []spawnOrphanEntryJSON {
	out := make([]spawnOrphanEntryJSON, 0, len(entries))
	for _, entry := range entries {
		last := spawnReapEntryLastActivity(entry)
		out = append(out, spawnOrphanEntryJSON{
			Name:           entry.AgentName,
			Caste:          entry.Caste,
			Parent:         entry.ParentName,
			Task:           entry.Task,
			ElapsedMinutes: int(now.Sub(last).Minutes()),
			LastActivity:   last.Format(time.RFC3339),
		})
	}
	return out
}

// renderSpawnOrphansListingText is D-14's plain-English primary view: caste
// names via casteIdentity, the worker's own name, how long it has gone
// without activity, and who called it -- never raw JSON or a bare status
// token as the thing the operator reads first.
func renderSpawnOrphansListingText(entries []agent.SpawnEntry, now time.Time, thresholdMinutes int) string {
	var b strings.Builder
	if len(entries) == 0 {
		b.WriteString(fmt.Sprintf("No helpers have gone past the %d-minute reap threshold with no completion reported.\n", thresholdMinutes))
		return b.String()
	}

	b.WriteString(fmt.Sprintf(
		"%d helper(s) have passed %d minutes with no completion reported and no reported activity since:\n\n",
		len(entries), thresholdMinutes,
	))
	for _, entry := range entries {
		elapsed := now.Sub(spawnReapEntryLastActivity(entry))
		b.WriteString(fmt.Sprintf(
			"  %s %s -- %s since it started, spawned by %s\n",
			casteIdentity(entry.Caste), entry.AgentName, formatDuration(elapsed), entry.ParentName,
		))
	}
	b.WriteString("\nThis is a listing only; nothing has been changed. Run `aether spawn-orphans --clear` to mark these abandoned and free their budget slots.\n")
	return b.String()
}

// renderSpawnOrphansClearText is the --clear view: what was reaped, in
// plain English, plus the budget count before and after so D-18's release
// is directly visible to the operator, not something they have to trust.
func renderSpawnOrphansClearText(reapedEntries []agent.SpawnEntry, now time.Time, thresholdMinutes int, before, after spawnTreeBudget) string {
	var b strings.Builder
	if len(reapedEntries) == 0 {
		b.WriteString(fmt.Sprintf(
			"No helpers had passed the %d-minute reap threshold; nothing was reaped. Budget remains %d of %d.\n",
			thresholdMinutes, before.Consumed, before.Max,
		))
		return b.String()
	}

	b.WriteString(fmt.Sprintf("Reaped %d helper(s) that passed %d minutes with no completion reported:\n\n", len(reapedEntries), thresholdMinutes))
	for _, entry := range reapedEntries {
		elapsed := now.Sub(spawnReapEntryLastActivity(entry))
		b.WriteString(fmt.Sprintf(
			"  %s %s -- was %s old, spawned by %s\n",
			casteIdentity(entry.Caste), entry.AgentName, formatDuration(elapsed), entry.ParentName,
		))
	}
	b.WriteString(fmt.Sprintf("\nBudget: %d of %d consumed before reaping, %d of %d after.\n", before.Consumed, before.Max, after.Consumed, after.Max))
	return b.String()
}

func writeSpawnOrphansEnvelope(result map[string]interface{}) {
	if shouldRenderVisualOutput(stdout) {
		if text, ok := result["_text"].(string); ok {
			delete(result, "_text")
			writeVisualOutput(stdout, text)
			return
		}
	}
	delete(result, "_text")
	data, err := json.Marshal(map[string]interface{}{"ok": true, "result": result})
	if err != nil {
		outputError(2, fmt.Sprintf("failed to marshal spawn-orphans result: %v", err), nil)
		return
	}
	fmt.Fprintln(stdout, string(data))
}

var spawnOrphansCmd = &cobra.Command{
	Use:   "spawn-orphans",
	Short: "List helpers that have exceeded the reap threshold, or clear them",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		clear, _ := cmd.Flags().GetBool("clear")
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		now := time.Now().UTC()
		thresholdMinutes := spawnReapThresholdMinutes()

		if !clear {
			// Listing path: read-only. This calls spawnReapCandidates
			// directly and nothing else -- no status update, no store
			// write. CLAUDE.md's --dry-run corollary applies: an
			// inspection command must not mutate the thing it inspects.
			candidates, err := spawnReapCandidates(st, now, thresholdMinutes)
			if err != nil {
				outputError(1, fmt.Sprintf("failed to scan spawn tree: %v", err), nil)
				return nil
			}
			result := map[string]interface{}{
				"threshold_minutes": thresholdMinutes,
				"orphan_count":      len(candidates),
				"orphans":           spawnOrphanEntriesJSON(candidates, now),
				"_text":             renderSpawnOrphansListingText(candidates, now, thresholdMinutes),
			}
			writeSpawnOrphansEnvelope(result)
			return nil
		}

		before, err := spawnTreeBudgetState()
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read budget before reaping: %v", err), nil)
			return nil
		}

		// Resolve the candidates before reaping so the report can name which
		// entries were reaped, not merely how many.
		candidates, err := spawnReapCandidates(st, now, thresholdMinutes)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to scan spawn tree: %v", err), nil)
			return nil
		}
		reaped, reapErr := spawnReapStaleEntries(st, now)
		if reapErr != nil {
			outputError(1, fmt.Sprintf("reap encountered an error: %v", reapErr), nil)
			return nil
		}

		reapedSet := make(map[string]bool, len(reaped))
		for _, name := range reaped {
			reapedSet[name] = true
		}
		var reapedEntries []agent.SpawnEntry
		for _, entry := range candidates {
			if reapedSet[entry.AgentName] {
				reapedEntries = append(reapedEntries, entry)
			}
		}

		after, err := spawnTreeBudgetState()
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read budget after reaping: %v", err), nil)
			return nil
		}

		if reaped == nil {
			reaped = []string{}
		}
		result := map[string]interface{}{
			"threshold_minutes": thresholdMinutes,
			"reaped":            reaped,
			"reaped_count":      len(reaped),
			"budget_before":     map[string]interface{}{"consumed": before.Consumed, "max": before.Max},
			"budget_after":      map[string]interface{}{"consumed": after.Consumed, "max": after.Max},
			"_text":             renderSpawnOrphansClearText(reapedEntries, now, thresholdMinutes, before, after),
		}
		writeSpawnOrphansEnvelope(result)
		return nil
	},
}

func init() {
	spawnOrphansCmd.Flags().Bool("clear", false, "Mark stale entries abandoned and release their budget slots (default: list only, no mutation)")
	rootCmd.AddCommand(spawnOrphansCmd)
}
