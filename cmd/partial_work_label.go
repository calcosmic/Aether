package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// partial_work_label.go is the one standing vocabulary every subsystem's
// half-finished work resolves through (LIVE-05, LIVE-07, D-12, CAP-072).
//
// Oracle already solved "useful but unproven" honestly for its own research
// (oracleResearchPartialLabel/oracleResearchStandingLabel,
// cmd/oracle_research_doc.go, Phase 198.2). The failure mode this file exists
// to prevent is solving the same problem three more times -- once per
// subsystem, with three different words for the same idea -- which is
// exactly the drift 202-CLASSIC-SYNTHESIS.md's SYN-202-12 row names as this
// repository's own recurring cost. Swarm's unproven repair ideas
// (cmd/swarm_episode.go), Oracle's partial research, plan research artifacts,
// and local reflections all resolve their standing through workStandingLabel
// below -- nothing renders a standing phrase locally.

// workStanding is the one standing vocabulary. Exactly three values exist on
// purpose: a fourth would be a second word for a standing this vocabulary
// already names.
type workStanding string

const (
	// workStandingVerified is a finished, checked result.
	workStandingVerified workStanding = "verified"
	// workStandingUsefulNotes is content with no verification behind it yet
	// -- kept, never discarded, and never presented as a checked result.
	workStandingUsefulNotes workStanding = "useful notes"
	// workStandingUnknown is an item whose standing could not be determined
	// -- e.g. a record that failed to parse. It is NEVER promoted to
	// verified.
	workStandingUnknown workStanding = "standing unknown"
)

// declaredWorkStandings is the full declared set, read by tests that assert
// the count rather than typing the number in.
func declaredWorkStandings() []workStanding {
	return []workStanding{workStandingVerified, workStandingUsefulNotes, workStandingUnknown}
}

// workStandingLabel renders a standing into an owner-facing phrase in plain
// words. whatWouldVerify names, in ordinary words, what would raise a
// useful-notes item to verified; every useful-notes label includes it. Both
// Swarm (cmd/swarm_episode.go) and Oracle (cmd/oracle_research_doc.go)
// render every standing phrase through this one function -- neither renders
// a standing phrase locally, so the same input standing always produces the
// same label regardless of which subsystem it came from.
func workStandingLabel(standing workStanding, whatWouldVerify string) string {
	switch standing {
	case workStandingVerified:
		return "verified"
	case workStandingUsefulNotes:
		reason := strings.TrimSpace(whatWouldVerify)
		if reason == "" {
			reason = "being checked and confirmed"
		}
		return fmt.Sprintf("useful notes, not verified — would be verified by %s", reason)
	default:
		return "standing unknown"
	}
}

// resolveMalformedItemStanding is the fallback every source below uses when
// an item cannot be parsed cleanly: it always resolves to workStandingUnknown
// paired with the reason it could not be determined. It never resolves to
// verified -- a record this repository could not read is never presented as
// a checked result.
func resolveMalformedItemStanding(reason string) (workStanding, string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "the item could not be parsed"
	}
	return workStandingUnknown, reason
}

// --- Task 2: the read-only inventory of everything unproven -------------

// The four kinds collectUnverifiedWork gathers, one per source (CAP-072,
// LIVE-05, LIVE-07, D-12).
const (
	unverifiedWorkKindOracleResearch  = "oracle-research"
	unverifiedWorkKindSwarmRepairIdea = "swarm-repair-idea"
	unverifiedWorkKindSwarmEpisode    = "swarm-episode"
	unverifiedWorkKindPlanResearch    = "plan-research"
	unverifiedWorkKindReflection      = "reflection"
)

// unverifiedWorkEntry is one item in the inventory: its kind, its subject,
// its standing, the plain-words sentence naming what would verify it (or,
// for standing unknown, why it could not be determined), the rendered label,
// its recorded time, and a path to read it in full.
type unverifiedWorkEntry struct {
	Kind       string       `json:"kind"`
	Subject    string       `json:"subject"`
	Standing   workStanding `json:"standing"`
	Reason     string       `json:"reason"`
	Label      string       `json:"label"`
	RecordedAt time.Time    `json:"recorded_at"`
	Path       string       `json:"path"`
}

// collectUnverifiedWork gathers every piece of unproven work the runtime
// already knows about, from four sources: the durable Oracle research
// directory, the Swarm episode store's unapplied/rolled-back repair ideas
// and interrupted episodes, the plan research directory the planning flow
// writes under the sanctioned scratch path, and the local reflections
// (Dreams) directory.
//
// This is strictly read-only: it creates no directory, opens no file for
// writing, and initializes no store. A missing source directory contributes
// no entries and no error. A malformed item is listed with standing unknown
// and a stated reason rather than dropped -- a silently dropped item is
// exactly the thing an owner would never learn about.
//
// Ordering is deterministic: by recorded time, then kind, then subject, so
// two runs over the same set produce identical output.
func collectUnverifiedWork(root string) ([]unverifiedWorkEntry, error) {
	var out []unverifiedWorkEntry

	research, err := collectUnverifiedOracleResearch(root)
	if err != nil {
		return nil, err
	}
	out = append(out, research...)

	swarm, err := collectUnverifiedSwarmWork(root)
	if err != nil {
		return nil, err
	}
	out = append(out, swarm...)

	plan, err := collectUnverifiedPlanResearch(root)
	if err != nil {
		return nil, err
	}
	out = append(out, plan...)

	reflections, err := collectUnverifiedReflections(root)
	if err != nil {
		return nil, err
	}
	out = append(out, reflections...)

	sort.Slice(out, func(i, j int) bool {
		if !out[i].RecordedAt.Equal(out[j].RecordedAt) {
			return out[i].RecordedAt.Before(out[j].RecordedAt)
		}
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Subject < out[j].Subject
	})
	return out, nil
}

// collectUnverifiedOracleResearch reads every saved research document
// (.aether/research/*.md) and resolves its standing through
// oracleResearchStanding -- the same function that decides
// oracleResearchPartialLabel's output -- so this listing and Oracle's own
// document body can never disagree about which runs are unfinished. Only
// non-verified documents are listed; a clean completion is not "unproven
// work."
func collectUnverifiedOracleResearch(root string) ([]unverifiedWorkEntry, error) {
	dir := oracleResearchDir(root)
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("collect oracle research: %w", err)
	}
	sort.Strings(matches)

	var out []unverifiedWorkEntry
	for _, match := range matches {
		relPath, relErr := filepath.Rel(root, match)
		if relErr != nil {
			relPath = match
		}
		relPath = filepath.ToSlash(relPath)

		data, readErr := os.ReadFile(match)
		if readErr != nil {
			standing, reason := resolveMalformedItemStanding("the document could not be read: " + readErr.Error())
			out = append(out, unverifiedWorkEntry{
				Kind:     unverifiedWorkKindOracleResearch,
				Subject:  filepath.Base(match),
				Standing: standing,
				Reason:   reason,
				Label:    workStandingLabel(standing, reason),
				Path:     relPath,
			})
			continue
		}

		text := string(data)
		if !strings.HasPrefix(text, "---\n") {
			standing, reason := resolveMalformedItemStanding("the document has no readable front matter")
			out = append(out, unverifiedWorkEntry{
				Kind:     unverifiedWorkKindOracleResearch,
				Subject:  filepath.Base(match),
				Standing: standing,
				Reason:   reason,
				Label:    workStandingLabel(standing, reason),
				Path:     relPath,
			})
			continue
		}

		entry := parseOracleResearchFrontMatter(text)
		standing := oracleResearchStanding(entry.Status)
		if standing == workStandingVerified {
			continue
		}
		reason := "finishing the remaining research rounds and reaching a clean completion"
		var recordedAt time.Time
		if t, perr := time.Parse(time.RFC3339, entry.Generated); perr == nil {
			recordedAt = t
		}
		subject := emptyFallback(entry.CoreQuestion, entry.Title)
		out = append(out, unverifiedWorkEntry{
			Kind:       unverifiedWorkKindOracleResearch,
			Subject:    subject,
			Standing:   standing,
			Reason:     reason,
			Label:      workStandingLabel(standing, reason),
			RecordedAt: recordedAt,
			Path:       relPath,
		})
	}
	return out, nil
}

// collectUnverifiedSwarmWork reads every persisted Swarm episode
// (.aether/data/swarms/*/episode.json) directly off disk -- deliberately not
// through storage.NewStore, which creates directories on construction and
// would violate this function's read-only guarantee. It resolves each
// episode's own run standing (swarmEpisodeStanding) and, when the episode
// proposed a repair, that idea's standing (swarmRepairIdeaStanding), both
// defined in cmd/swarm_episode.go against this file's shared vocabulary.
func collectUnverifiedSwarmWork(root string) ([]unverifiedWorkEntry, error) {
	swarmsDir := filepath.Join(root, ".aether", "data", "swarms")
	entries, err := os.ReadDir(swarmsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("collect swarm work: read swarms: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	var out []unverifiedWorkEntry
	for _, name := range names {
		episodePath := filepath.Join(swarmsDir, name, "episode.json")
		relPath := filepath.ToSlash(filepath.Join(".aether", "data", "swarms", name, "episode.json"))

		data, readErr := os.ReadFile(episodePath)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			standing, reason := resolveMalformedItemStanding("the episode record could not be read: " + readErr.Error())
			out = append(out, unverifiedWorkEntry{
				Kind:     unverifiedWorkKindSwarmEpisode,
				Subject:  name,
				Standing: standing,
				Reason:   reason,
				Label:    workStandingLabel(standing, reason),
				Path:     relPath,
			})
			continue
		}

		var record swarmEpisodeRecord
		if jsonErr := json.Unmarshal(data, &record); jsonErr != nil || record.SchemaVersion != swarmEpisodeSchemaVersion {
			standing, reason := resolveMalformedItemStanding("the episode record is malformed or from an incompatible schema version")
			out = append(out, unverifiedWorkEntry{
				Kind:     unverifiedWorkKindSwarmEpisode,
				Subject:  name,
				Standing: standing,
				Reason:   reason,
				Label:    workStandingLabel(standing, reason),
				Path:     relPath,
			})
			continue
		}

		var recordedAt time.Time
		if t, perr := time.Parse(time.RFC3339Nano, record.EndedAt); perr == nil {
			recordedAt = t
		}

		if standing, reason := swarmEpisodeStanding(record); standing != workStandingVerified {
			out = append(out, unverifiedWorkEntry{
				Kind:       unverifiedWorkKindSwarmEpisode,
				Subject:    emptyFallback(record.Target, name),
				Standing:   standing,
				Reason:     reason,
				Label:      workStandingLabel(standing, reason),
				RecordedAt: recordedAt,
				Path:       relPath,
			})
		}
		if standing, reason := swarmRepairIdeaStanding(record); standing != workStandingVerified {
			out = append(out, unverifiedWorkEntry{
				Kind:       unverifiedWorkKindSwarmRepairIdea,
				Subject:    record.Comparison.Selected.Repair,
				Standing:   standing,
				Reason:     reason,
				Label:      workStandingLabel(standing, reason),
				RecordedAt: recordedAt,
				Path:       relPath,
			})
		}
	}
	return out, nil
}

// collectUnverifiedPlanResearch lists every file the planning flow has
// written under its sanctioned scratch path (.aether/data/planning/ --
// SCOUT.md, ROUTE-SETTER.md, phase-plan.json and similar). Plan research is
// always useful notes: it is read as input to a plan, not a checked result
// on its own.
func collectUnverifiedPlanResearch(root string) ([]unverifiedWorkEntry, error) {
	dir := filepath.Join(root, ".aether", "data", "planning")
	return collectUnverifiedDirectory(root, dir, unverifiedWorkKindPlanResearch,
		"being carried into a phase plan the owner accepts and the colony executes")
}

// collectUnverifiedReflections lists every local reflection
// (.aether/dreams/*) the lifecycle facts reader already locates
// (readLifecycleResearch, cmd/lifecycle_facts.go). A reflection is always
// useful notes: it is a private observation, not a checked result.
func collectUnverifiedReflections(root string) ([]unverifiedWorkEntry, error) {
	dir := filepath.Join(root, ".aether", "dreams")
	return collectUnverifiedDirectory(root, dir, unverifiedWorkKindReflection,
		"being reviewed and acted on (see `aether interpret`)")
}

// collectUnverifiedDirectory is the shared read-only walk both plan research
// and reflections use: every regular file under dir becomes one useful-notes
// entry, ordered by path for a stable base ordering before
// collectUnverifiedWork's final sort. A missing directory yields no entries
// and no error.
func collectUnverifiedDirectory(root, dir, kind, whatWouldVerify string) ([]unverifiedWorkEntry, error) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("collect %s: %w", kind, err)
	}
	if !info.IsDir() {
		return nil, nil
	}

	var out []unverifiedWorkEntry
	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relPath, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relPath = path
		}
		var recordedAt time.Time
		if fi, statErr := entry.Info(); statErr == nil {
			recordedAt = fi.ModTime()
		}
		out = append(out, unverifiedWorkEntry{
			Kind:       kind,
			Subject:    filepath.Base(path),
			Standing:   workStandingUsefulNotes,
			Reason:     whatWouldVerify,
			Label:      workStandingLabel(workStandingUsefulNotes, whatWouldVerify),
			RecordedAt: recordedAt,
			Path:       filepath.ToSlash(relPath),
		})
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("collect %s: %w", kind, walkErr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}
