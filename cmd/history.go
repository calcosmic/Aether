package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var (
	historyLimit  int
	historyFilter string
	historyKind   string
	historyJSON   bool
)

const (
	lifecycleHistoryCategoryEvent           = "event"
	lifecycleHistoryCategoryActiveWork      = "active_work"
	lifecycleHistoryCategoryRecentOutcome   = "recent_outcome"
	lifecycleHistoryCategoryUnknownEvidence = "unknown_evidence"
	// lifecycleHistoryCategoryEpisode and lifecycleHistoryCategoryUnverified
	// are 202-14's two additional row families: one durable, finished-or-not
	// episode (Swarm run, Oracle research, build/check attempt -- the shared
	// lineage, cmd/episode_index.go) and one piece of unproven work (Oracle
	// partial research, a Swarm repair idea, an interrupted episode, plan
	// research, a reflection -- cmd/partial_work_label.go's
	// collectUnverifiedWork). Both are appended to the existing event/actor/
	// receipt rows, never merged into an existing category -- the two
	// systems can list the SAME underlying Swarm episode or Oracle document
	// without this listing trying to reconcile them into one row.
	lifecycleHistoryCategoryEpisode    = "episode"
	lifecycleHistoryCategoryUnverified = "unverified_work"
)

// LifecycleHistoryRow is the human meaning of one recorded history fact. Raw
// holds machine detail for JSON callers; the visual renderer deliberately
// reads only Event, Actor, Result, and Timestamp.
type LifecycleHistoryRow struct {
	Timestamp      string                   `json:"timestamp"`
	Event          string                   `json:"event"`
	Actor          string                   `json:"actor"`
	Result         string                   `json:"result"`
	Category       string                   `json:"category"`
	EvidenceKind   string                   `json:"evidence_kind"`
	EvidenceSource string                   `json:"evidence_source,omitempty"`
	Known          bool                     `json:"known"`
	Raw            string                   `json:"raw,omitempty"`
	ReceiptID      string                   `json:"receipt_id,omitempty"`
	OutcomeKind    colony.OutcomeKind       `json:"outcome_kind,omitempty"`
	RawReceipt     *colony.LifecycleReceipt `json:"raw_receipt,omitempty"`

	// Kind, Standing, Cost and Path are populated ONLY by the 202-14 episode
	// and unverified-work rows below -- every existing lifecycle event,
	// actor and receipt row leaves all four empty, so writeLifecycleHistoryRow
	// renders those rows byte-identically to before this plan.
	Kind     string `json:"kind,omitempty"`
	Standing string `json:"standing,omitempty"`
	Cost     string `json:"cost,omitempty"`
	Path     string `json:"path,omitempty"`
}

// LifecycleHistoryResult is a focused view over the shared lifecycle
// projection. It copies shared answers; it does not resolve identity,
// blockers, or Next Up independently.
type LifecycleHistoryResult struct {
	SchemaVersion      string             `json:"schema_version"`
	Command            string             `json:"command"`
	OutcomeKind        colony.OutcomeKind `json:"outcome_kind"`
	ProjectionRevision string             `json:"projection_revision"`
	Platform           string             `json:"platform"`

	Identity LifecycleFact[LifecycleIdentityFacts]   `json:"identity"`
	Goal     LifecycleFact[string]                   `json:"goal"`
	Standing LifecycleFact[string]                   `json:"standing"`
	Phase    LifecycleFact[LifecyclePhaseProjection] `json:"phase"`

	Events         []LifecycleHistoryRow `json:"events"`
	ActiveWork     []LifecycleHistoryRow `json:"active_work"`
	RecentOutcomes []LifecycleHistoryRow `json:"recent_outcomes"`
	Count          int                   `json:"count"`
	Filter         string                `json:"filter,omitempty"`
	// Kind narrows the listing to one row kind (e.g. "swarm",
	// "oracle-research", "build-attempt", or one of collectUnverifiedWork's
	// kinds) without altering any surviving row's own content (202-14).
	Kind          string              `json:"kind,omitempty"`
	Limit         int                 `json:"limit"`
	HistorySource LifecycleFactSource `json:"history_source"`

	Blockers       []colony.LifecycleIssue     `json:"blockers"`
	OwnerDecisions []colony.LifecycleDecision  `json:"owner_decisions"`
	NextAction     LifecycleProjectedAction    `json:"next_action"`
	Alternatives   []LifecycleActionChoice     `json:"alternatives,omitempty"`
	StateEffect    colony.LifecycleStateEffect `json:"state_effect"`
}

var historyCmd = &cobra.Command{
	Use:         "history",
	Short:       "Show focused colony activity history",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		root := resolveAetherRoot()
		now := time.Now().UTC()
		facts, err := loadLifecycleFacts(root, store, now)
		if err != nil {
			facts = unavailableLifecycleFacts(root, now, err.Error())
		}
		projection := projectLifecycle(facts, LifecycleViewFocused, detectPlatform())
		projection.Command = "history"
		result := buildLifecycleHistoryProjection(facts, projection, historyFilter, historyLimit, historyKind, root, store)

		if historyJSON {
			outputOK(result)
			return nil
		}
		outputWorkflow(result, renderLifecycleHistory(result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().IntVar(&historyLimit, "limit", 20, "Maximum number of history rows to show")
	historyCmd.Flags().StringVar(&historyFilter, "filter", "", "Filter history by event, actor, result, or evidence source")
	historyCmd.Flags().StringVar(&historyKind, "kind", "", "Narrow the listing to one row kind (e.g. swarm, oracle-research, build-attempt, check-attempt, or an unverified-work kind)")
	historyCmd.Flags().BoolVar(&historyJSON, "json", false, "Output as JSON, including raw evidence detail")
}

// parseEvent splits the durable pipe-delimited event shape without assigning
// semantic meaning to its source field. A source such as "build" is a command,
// not evidence that an actor named Build performed the work.
func parseEvent(event string) (timestamp, eventType, source, message string) {
	parts := strings.SplitN(event, "|", 4)
	switch len(parts) {
	case 4:
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), strings.TrimSpace(parts[3])
	case 3:
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), strings.TrimSpace(parts[2]), ""
	case 2:
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), "", ""
	default:
		return strings.TrimSpace(event), "", "", ""
	}
}

func buildLifecycleHistoryProjection(facts LifecycleFacts, projection LifecycleProjection, filter string, limit int, kind, root string, s *storage.Store) LifecycleHistoryResult {
	rows := make([]LifecycleHistoryRow, 0, len(facts.History.Value)+len(facts.Actors.Value)+1)
	for _, raw := range facts.History.Value {
		rows = append(rows, lifecycleHistoryEventRow(raw))
	}
	for _, actorFact := range facts.Actors.Value {
		rows = append(rows, lifecycleHistoryActorRow(actorFact))
	}
	if receipt := facts.Evidence.Value.Receipt; receipt != nil {
		rows = append(rows, lifecycleHistoryReceiptRow(receipt))
	}
	// 202-14: every episode in the shared lineage, and every piece of
	// unverified work, in the SAME listing -- never a second screen the
	// owner has to remember to check. Neither call ever writes anything
	// (loadColonyEpisodeIndex, collectUnverifiedWork are both pure reads);
	// an unresolvable source simply contributes no rows.
	rows = append(rows, lifecycleHistoryEpisodeRows(root, s)...)
	rows = append(rows, lifecycleHistoryUnverifiedRows(root)...)

	filter = strings.TrimSpace(filter)
	if filter != "" {
		filtered := rows[:0]
		needle := strings.ToLower(filter)
		for _, row := range rows {
			haystack := strings.ToLower(strings.Join([]string{
				row.Event, row.Actor, row.Result, row.Category,
				row.EvidenceKind, row.EvidenceSource, row.Raw,
			}, "\n"))
			if strings.Contains(haystack, needle) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	kind = strings.TrimSpace(kind)
	if kind != "" {
		filtered := rows[:0]
		for _, row := range rows {
			if row.Kind == kind {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left, leftOK := lifecycleHistoryTime(rows[i].Timestamp)
		right, rightOK := lifecycleHistoryTime(rows[j].Timestamp)
		if leftOK != rightOK {
			return leftOK
		}
		if leftOK && !left.Equal(right) {
			return left.After(right)
		}
		return lifecycleHistoryTieKey(rows[i]) < lifecycleHistoryTieKey(rows[j])
	})
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}

	active := make([]LifecycleHistoryRow, 0)
	recent := make([]LifecycleHistoryRow, 0)
	for _, row := range rows {
		switch row.Category {
		case lifecycleHistoryCategoryActiveWork:
			active = append(active, row)
		case lifecycleHistoryCategoryRecentOutcome:
			recent = append(recent, row)
		}
	}

	return LifecycleHistoryResult{
		SchemaVersion:      projection.SchemaVersion,
		Command:            "history",
		OutcomeKind:        projection.OutcomeKind,
		ProjectionRevision: projection.ProjectionRevision,
		Platform:           projection.Platform,
		Identity:           projection.Identity,
		Goal:               projection.Goal,
		Standing:           projection.Standing,
		Phase:              projection.Phase,
		Events:             append([]LifecycleHistoryRow{}, rows...),
		ActiveWork:         active,
		RecentOutcomes:     recent,
		Count:              len(rows),
		Filter:             filter,
		Kind:               kind,
		Limit:              limit,
		HistorySource:      facts.History.Source,
		Blockers:           append([]colony.LifecycleIssue(nil), projection.Blockers...),
		OwnerDecisions:     append([]colony.LifecycleDecision(nil), projection.OwnerDecisions...),
		NextAction:         projection.NextAction,
		Alternatives:       append([]LifecycleActionChoice(nil), projection.Alternatives...),
		StateEffect:        colony.LifecycleStateEffectNone,
	}
}

func lifecycleHistoryEventRow(raw string) LifecycleHistoryRow {
	timestamp, eventType, source, message := parseEvent(raw)
	event, known := lifecycleHistoryEventName(eventType)
	_, timestampKnown := lifecycleHistoryTime(timestamp)
	if !timestampKnown {
		known = false
	}
	category := lifecycleHistoryCategoryEvent
	if !known {
		category = lifecycleHistoryCategoryUnknownEvidence
		label := strings.TrimSpace(eventType)
		if label == "" {
			label = "untyped"
		}
		event = "unknown event evidence: " + label
	}
	if message == "" {
		message = "No result was recorded."
	}
	return LifecycleHistoryRow{
		Timestamp:      timestamp,
		Event:          event,
		Actor:          "Unknown",
		Result:         message,
		Category:       category,
		EvidenceKind:   "event",
		EvidenceSource: source,
		Known:          known,
		Raw:            raw,
	}
}

func lifecycleHistoryActorRow(actorFact LifecycleActorFact) LifecycleHistoryRow {
	name := strings.TrimSpace(actorFact.Name)
	if name == "" {
		name = "Unknown"
	}
	result := strings.TrimSpace(actorFact.Summary)
	if result == "" {
		result = strings.TrimSpace(actorFact.Task)
	}
	if result == "" {
		result = "No result was recorded."
	}
	row := LifecycleHistoryRow{
		Timestamp:      strings.TrimSpace(actorFact.Timestamp),
		Actor:          name,
		Result:         result,
		EvidenceKind:   "actor",
		EvidenceSource: "spawn tree",
	}
	switch {
	case agent.IsLiveSpawnStatus(actorFact.Status):
		row.Event = "work in progress"
		row.Category = lifecycleHistoryCategoryActiveWork
		row.Known = true
	case agent.IsTerminalSpawnStatus(actorFact.Status):
		row.Event = "work completed"
		row.Category = lifecycleHistoryCategoryRecentOutcome
		row.Known = true
	default:
		row.Event = "unknown worker evidence: " + emptyFallback(strings.TrimSpace(actorFact.Status), "untyped")
		row.Category = lifecycleHistoryCategoryUnknownEvidence
	}
	if _, ok := lifecycleHistoryTime(row.Timestamp); !ok {
		row.Event = "unknown worker evidence: invalid timestamp"
		row.Category = lifecycleHistoryCategoryUnknownEvidence
		row.Known = false
	}
	return row
}

func lifecycleHistoryReceiptRow(receipt *colony.LifecycleReceipt) LifecycleHistoryRow {
	if receipt == nil {
		return LifecycleHistoryRow{}
	}
	command := strings.TrimSpace(receipt.Command)
	event := strings.ReplaceAll(command, "-", " ")
	if event == "" {
		event = "untyped"
	}
	known := receipt.OutcomeKind.Valid() && command != ""
	category := lifecycleHistoryCategoryRecentOutcome
	if receipt.OutcomeKind == colony.OutcomeKindInProgress {
		category = lifecycleHistoryCategoryActiveWork
	}
	if !known {
		category = lifecycleHistoryCategoryUnknownEvidence
		event = "unknown receipt evidence: " + event
	} else {
		event += " receipt"
	}
	result := strings.ReplaceAll(string(receipt.OutcomeKind), "_", " ")
	if result == "" {
		result = "No result was recorded."
	}
	return LifecycleHistoryRow{
		Event:          event,
		Actor:          "Unknown",
		Result:         result,
		Category:       category,
		EvidenceKind:   "receipt",
		EvidenceSource: "lifecycle receipt",
		Known:          known,
		ReceiptID:      strings.TrimSpace(receipt.ReceiptID),
		OutcomeKind:    receipt.OutcomeKind,
		RawReceipt:     receipt,
	}
}

func lifecycleHistoryEventName(eventType string) (string, bool) {
	names := map[string]string{
		"initialized":                "colony initialized",
		"init":                       "colony initialized",
		"state_repaired":             "colony state repaired",
		"state_recovered":            "colony state recovered",
		"plan":                       "plan accepted",
		"phase_started":              "phase started",
		"build":                      "build started",
		"build_dispatched":           "build dispatched",
		"build_completed":            "build completed",
		"build_partial_credit":       "build partial result recorded",
		"build_dispatch_failed":      "build dispatch failed",
		"phase_tasks_repaired":       "phase tasks repaired",
		"continue_review":            "continue review recorded",
		"deterministic_verification": "deterministic verification recorded",
		"watcher_verification":       "independent verification recorded",
		"signal_housekeeping":        "signal housekeeping completed",
		"continue_blocked":           "continue blocked",
		"phase_advanced":             "phase advanced",
		"phase_completed":            "phase completed",
		"complete":                   "phase complete",
		"phase_skipped":              "phase skipped",
		"sealed":                     "colony sealed",
		"sealed_forced":              "colony force-sealed",
		"territory_surveyed":         "territory surveyed",
	}
	name, ok := names[strings.ToLower(strings.TrimSpace(eventType))]
	return name, ok
}

func lifecycleHistoryTime(raw string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
	return parsed, err == nil
}

func lifecycleHistoryTieKey(row LifecycleHistoryRow) string {
	return strings.Join([]string{
		row.Event,
		row.Actor,
		row.Result,
		row.EvidenceKind,
		row.EvidenceSource,
		row.ReceiptID,
		row.Raw,
	}, "\x00")
}

func renderLifecycleHistory(result LifecycleHistoryResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("history"), "History"))
	b.WriteString(visualDividerStr())
	b.WriteString("Colony\n")
	b.WriteString("Name: " + emptyFallback(strings.TrimSpace(result.Identity.Value.Name), "Unnamed colony") + "\n")
	b.WriteString("Goal: " + emptyFallback(strings.TrimSpace(result.Goal.Value), "Not recorded") + "\n")
	b.WriteString("Standing: " + emptyFallback(strings.TrimSpace(result.Standing.Value), "UNKNOWN") + "\n")
	phase := result.Phase.Value
	if phase.Current != nil {
		b.WriteString(fmt.Sprintf("Current phase: %d/%d — %s\n", phase.CurrentNumber, phase.TotalPhases, emptyFallback(strings.TrimSpace(phase.Current.Name), "Unnamed phase")))
	}

	b.WriteString("\nActivity\n")
	if len(result.Events) == 0 {
		b.WriteString("No events recorded.\n")
	} else {
		for _, row := range result.Events {
			writeLifecycleHistoryRow(&b, row)
		}
	}

	b.WriteString("\nActive work\n")
	if len(result.ActiveWork) == 0 {
		b.WriteString("None recorded.\n")
	} else {
		for _, row := range result.ActiveWork {
			writeLifecycleHistoryRow(&b, row)
		}
	}

	b.WriteString("\nRecent completed outcomes\n")
	if len(result.RecentOutcomes) == 0 {
		b.WriteString("None recorded.\n")
	} else {
		for _, row := range result.RecentOutcomes {
			writeLifecycleHistoryRow(&b, row)
		}
	}

	if len(result.Blockers) > 0 {
		b.WriteString("\nBlockers\n")
		for _, blocker := range result.Blockers {
			b.WriteString("• " + emptyFallback(strings.TrimSpace(blocker.Summary), blocker.ID) + "\n")
		}
	}

	b.WriteString(renderLifecycleProjectionNextUp(LifecycleProjection{
		NextAction:   result.NextAction,
		Alternatives: result.Alternatives,
	}, result.Platform))
	return b.String()
}

func writeLifecycleHistoryRow(b *strings.Builder, row LifecycleHistoryRow) {
	label := formatTimestamp(row.Timestamp)
	if label == "" {
		label = "unknown time"
	}
	b.WriteString("[" + label + "] " + emptyFallback(strings.TrimSpace(row.Event), "unknown evidence") + "\n")
	b.WriteString("  Actor: " + emptyFallback(strings.TrimSpace(row.Actor), "Unknown") + "\n")
	b.WriteString("  Result: " + emptyFallback(strings.TrimSpace(row.Result), "No result was recorded.") + "\n")
	// The three lines below are ONLY written when the row actually carries
	// this 202-14 content -- every pre-existing event/actor/receipt row
	// leaves Standing/Cost/Path empty, so this function's output for those
	// rows is byte-identical to before this plan.
	if standing := strings.TrimSpace(row.Standing); standing != "" {
		b.WriteString("  Standing: " + standing + "\n")
	}
	if cost := strings.TrimSpace(row.Cost); cost != "" {
		b.WriteString("  Cost: " + cost + "\n")
	}
	if path := strings.TrimSpace(row.Path); path != "" {
		b.WriteString("  Read the full write-up: " + path + "\n")
	}
}

// --- 202-14: episode and unverified-work rows -----------------------------

// lifecycleHistoryEpisodeRows renders the shared lineage (cmd/episode_index.go)
// as history rows: one per Swarm episode, saved Oracle research document, or
// build/check attempt, each carrying its outcome, its standing, its cost
// where a ledger reference resolves to one, and the path to read it in
// full. Never merges an episode into an existing event/actor/receipt row --
// this is purely additive.
func lifecycleHistoryEpisodeRows(root string, s *storage.Store) []LifecycleHistoryRow {
	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		return nil
	}
	rows := make([]LifecycleHistoryRow, 0, len(idx.Entries))
	for _, entry := range idx.Entries {
		rows = append(rows, lifecycleHistoryRowFromEpisode(entry))
	}
	return rows
}

func lifecycleHistoryRowFromEpisode(entry colonyEpisodeEntry) LifecycleHistoryRow {
	at := entry.EndedAt
	if at.IsZero() {
		at = entry.StartedAt
	}
	timestamp := ""
	if !at.IsZero() {
		timestamp = at.UTC().Format(time.RFC3339Nano)
	}

	outcome := entry.Outcome
	if !entry.OutcomeKnown {
		outcome = "unknown"
	}

	return LifecycleHistoryRow{
		Timestamp:      timestamp,
		Event:          colonyEpisodeKindLabel(entry.Kind) + ": " + entry.Subject,
		Actor:          "Unknown",
		Result:         "Outcome: " + outcome,
		Category:       lifecycleHistoryCategoryEpisode,
		EvidenceKind:   "episode",
		EvidenceSource: entry.Kind,
		Known:          entry.OutcomeKnown,
		Kind:           entry.Kind,
		Standing:       workStandingLabel(entry.Standing, entry.StandingReason),
		Cost:           colonyEpisodeCostBlock(entry.Cost),
		Path:           entry.Path,
	}
}

// colonyEpisodeCostBlock renders an episode entry's cost reference through
// the exact SAME renderer status's own section uses
// (renderSpendCostLineFromLedgers, cmd/spend_cost_line.go) -- never a
// second accounting path or a second rendering -- so status and history can
// never disagree about what one episode cost. Empty when the entry carries
// no ledger reference at all (Swarm and Oracle research today, whose Cost
// field is the zero value): "cost where applicable" (202-14) never prints a
// line for an entry that was never billed against a phase ledger to begin
// with.
func colonyEpisodeCostBlock(ref colonyEpisodeCostRef) string {
	if ref.Phase <= 0 {
		return ""
	}
	return renderSpendCostLineFromLedgers(colonyEpisodeLedgers(ref))
}

// lifecycleHistoryUnverifiedRows renders collectUnverifiedWork's inventory
// (cmd/partial_work_label.go) as history rows: Oracle partial research, an
// unapplied or rolled-back Swarm repair idea, an interrupted Swarm episode,
// plan research, and local reflections -- every piece of work this
// repository has recorded that is not yet a checked result, each carrying
// its standing label and the path to read it in full. This listing can
// name the same underlying Swarm episode or Oracle document
// lifecycleHistoryEpisodeRows already named -- the two are deliberately
// separate rows (one lineage entry, one standing-vocabulary entry), never
// merged or deduplicated against each other.
func lifecycleHistoryUnverifiedRows(root string) []LifecycleHistoryRow {
	items, err := collectUnverifiedWork(root)
	if err != nil {
		return nil
	}
	rows := make([]LifecycleHistoryRow, 0, len(items))
	for _, item := range items {
		rows = append(rows, lifecycleHistoryRowFromUnverified(item))
	}
	return rows
}

func lifecycleHistoryRowFromUnverified(item unverifiedWorkEntry) LifecycleHistoryRow {
	timestamp := ""
	if !item.RecordedAt.IsZero() {
		timestamp = item.RecordedAt.UTC().Format(time.RFC3339Nano)
	}
	return LifecycleHistoryRow{
		Timestamp:      timestamp,
		Event:          "Unverified work: " + item.Subject,
		Actor:          "Unknown",
		Result:         item.Reason,
		Category:       lifecycleHistoryCategoryUnverified,
		EvidenceKind:   "unverified-work",
		EvidenceSource: item.Kind,
		Known:          item.Standing != workStandingUnknown,
		Kind:           item.Kind,
		Standing:       item.Label,
		Path:           item.Path,
	}
}
