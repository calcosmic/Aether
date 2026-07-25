package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// Hive types for wisdom management.

// hiveEvidence records where one piece of cross-colony wisdom came from so a
// later colony can judge it instead of trusting prose. Reference identifies
// the producing artifact (for example "seal:<instinct-id>" or "hive-store").
type hiveEvidence struct {
	RepoID    string `json:"repo_id"`
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	At        string `json:"at"`
}

type hiveWisdomEntry struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Domain      string   `json:"domain"`
	SourceRepo  string   `json:"source_repo"`
	SourceRepos []string `json:"source_repos,omitempty"`
	// SourceRepoIDs holds stable repository identities (see
	// hive_repo_identity.go). Confidence boosts count these, never display
	// names, so the same repository cannot inflate its own count by changing
	// directories or names.
	SourceRepoIDs []string       `json:"source_repo_ids,omitempty"`
	Evidence      []hiveEvidence `json:"evidence,omitempty"`
	Confidence    float64        `json:"confidence"`
	CreatedAt     string         `json:"created_at"`
	AccessedAt    string         `json:"accessed_at"`
	AccessCount   int            `json:"access_count"`
	// LastConfirmedAt drives lazy decay: confidence is recomputed at read
	// time from this timestamp and never silently rewritten in the file.
	LastConfirmedAt string `json:"last_confirmed_at,omitempty"`
	// Revoked entries are never retrieved and are evicted first, but remain
	// in the file as an audit trail.
	Revoked       bool   `json:"revoked,omitempty"`
	RevokedAt     string `json:"revoked_at,omitempty"`
	RevokedReason string `json:"revoked_reason,omitempty"`
	// Quarantined entries contradicted an existing entry on arrival. Neither
	// side wins silently: the new entry waits for human resolution.
	Quarantined      bool     `json:"quarantined,omitempty"`
	QuarantineReason string   `json:"quarantine_reason,omitempty"`
	Contradicts      []string `json:"contradicts,omitempty"`
	ContradictedBy   []string `json:"contradicted_by,omitempty"`
	// EffectiveConfidence is the decay-adjusted confidence computed at read
	// time. It is report-only and never persisted as truth.
	EffectiveConfidence float64 `json:"effective_confidence,omitempty"`
}

type hiveWisdomData struct {
	Version int               `json:"version,omitempty"`
	Entries []hiveWisdomEntry `json:"entries"`
}

const hiveWisdomPath = "hive/wisdom.json"
const maxHiveEntries = 200

// hiveWisdomSchemaVersion is written on every save. Readers tolerate older
// files with missing fields and backfill them on the next write.
const hiveWisdomSchemaVersion = 2

// hiveDecayHalfLifeDays is the half-life applied lazily at read time. Wisdom
// that no colony re-confirms fades; reinforcement re-confirms it.
const hiveDecayHalfLifeDays = 180.0

// hiveRetrievalMinEffectiveConfidence is the floor below which decayed wisdom
// is treated as dormant and not injected into worker context.
const hiveRetrievalMinEffectiveConfidence = 0.3

// --- locked storage ---

// hiveStorageStore returns the locked, atomic store rooted at the hub's hive
// directory. All wisdom reads and writes go through it: two colonies must
// never lose each other's updates.
func hiveStorageStore(hub string) (*storage.Store, error) {
	return storage.NewStore(filepath.Join(hub, "hive"))
}

// loadWisdomLocked reads wisdom.json through the store. A missing file is an
// empty v2 dataset; a corrupted file is an error so writes never silently
// destroy existing knowledge.
func loadWisdomLocked(hub string) (hiveWisdomData, error) {
	s, err := hiveStorageStore(hub)
	if err != nil {
		return hiveWisdomData{}, err
	}
	var wf hiveWisdomData
	if err := s.LoadJSON("wisdom.json", &wf); err != nil {
		if os.IsNotExist(err) {
			return hiveWisdomData{Version: hiveWisdomSchemaVersion, Entries: []hiveWisdomEntry{}}, nil
		}
		return hiveWisdomData{}, fmt.Errorf("corrupted wisdom.json: %w", err)
	}
	migrateHiveWisdom(&wf)
	return wf, nil
}

// updateWisdomLocked applies a mutation under the store's cross-process lock
// and persists atomically.
func updateWisdomLocked(hub string, mutate func(*hiveWisdomData) error) error {
	s, err := hiveStorageStore(hub)
	if err != nil {
		return err
	}
	var wf hiveWisdomData
	err = s.UpdateJSONAtomically("wisdom.json", &wf, func() error {
		migrateHiveWisdom(&wf)
		return mutate(&wf)
	})
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("hive is not initialized; run aether hive-init first")
	}
	return err
}

// migrateHiveWisdom backfills v2 fields on pre-versioning entries.
func migrateHiveWisdom(wf *hiveWisdomData) {
	if wf.Version >= hiveWisdomSchemaVersion {
		return
	}
	wf.Version = hiveWisdomSchemaVersion
	for i := range wf.Entries {
		if strings.TrimSpace(wf.Entries[i].LastConfirmedAt) == "" {
			wf.Entries[i].LastConfirmedAt = firstNonEmpty(
				strings.TrimSpace(wf.Entries[i].CreatedAt),
				strings.TrimSpace(wf.Entries[i].AccessedAt),
			)
		}
	}
}

// --- activity, decay, and contradiction ---

func hiveEntryActive(entry hiveWisdomEntry) bool {
	return !entry.Revoked && !entry.Quarantined
}

// effectiveHiveConfidence applies lazy decay from the last confirmation
// timestamp. The stored confidence is never rewritten by reads.
func effectiveHiveConfidence(entry hiveWisdomEntry, now time.Time) float64 {
	base := entry.Confidence
	anchor := strings.TrimSpace(entry.LastConfirmedAt)
	if anchor == "" {
		anchor = strings.TrimSpace(entry.CreatedAt)
	}
	if anchor == "" || base <= 0 {
		return base
	}
	parsed, err := time.Parse(time.RFC3339, anchor)
	if err != nil {
		return base
	}
	days := now.Sub(parsed).Hours() / 24
	if days <= 0 {
		return base
	}
	return base * math.Pow(0.5, days/hiveDecayHalfLifeDays)
}

// hiveNegationMarkers are the asymmetry signals used by the contradiction
// heuristic. Detection is deliberately conservative: high token overlap plus
// opposite negation parity, nothing else.
var hiveNegationMarkers = []string{"not", "never", "no ", "don't", "dont", "avoid", "stop"}

func hiveTokens(text string) map[string]bool {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	tokens := map[string]bool{}
	for _, field := range fields {
		if field != "" {
			tokens[field] = true
		}
	}
	return tokens
}

func hiveNegationParity(text string) bool {
	lower := strings.ToLower(text)
	for _, marker := range hiveNegationMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

// detectHiveContradiction reports whether candidate contradicts existing.
// It fires only when the two texts share most of their tokens but disagree
// on negation, which is the strongest cheap signal available without model
// judgment. Same-domain comparison only.
func detectHiveContradiction(candidate, existing string) bool {
	candidateTokens := hiveTokens(candidate)
	existingTokens := hiveTokens(existing)
	if len(candidateTokens) == 0 || len(existingTokens) == 0 {
		return false
	}
	overlap := 0
	for token := range candidateTokens {
		if existingTokens[token] {
			overlap++
		}
	}
	smaller := len(candidateTokens)
	if len(existingTokens) < smaller {
		smaller = len(existingTokens)
	}
	if float64(overlap)/float64(smaller) < 0.6 {
		return false
	}
	return hiveNegationParity(candidate) != hiveNegationParity(existing)
}

// evictHiveEntryForCapacity removes the least valuable entry when the file is
// at capacity: revoked entries first, then quarantined, then least recently
// accessed.
func evictHiveEntryForCapacity(wf *hiveWisdomData) {
	if len(wf.Entries) < maxHiveEntries {
		return
	}
	oldestIdx := -1
	oldestQuarantined := -1
	oldestRevoked := -1
	for i, e := range wf.Entries {
		if e.Revoked {
			if oldestRevoked < 0 || e.AccessedAt < wf.Entries[oldestRevoked].AccessedAt {
				oldestRevoked = i
			}
			continue
		}
		if e.Quarantined {
			if oldestQuarantined < 0 || e.AccessedAt < wf.Entries[oldestQuarantined].AccessedAt {
				oldestQuarantined = i
			}
			continue
		}
		if oldestIdx < 0 || e.AccessedAt < wf.Entries[oldestIdx].AccessedAt {
			oldestIdx = i
		}
	}
	evict := oldestRevoked
	if evict < 0 {
		evict = oldestQuarantined
	}
	if evict < 0 {
		evict = oldestIdx
	}
	if evict >= 0 {
		wf.Entries = append(wf.Entries[:evict], wf.Entries[evict+1:]...)
	}
}

// --- colony retrieval consent ---

const hiveRetrievalConsentFile = "hive_retrieval.json"

type hiveRetrievalConsent struct {
	OptIn     bool   `json:"opt_in"`
	UpdatedAt string `json:"updated_at"`
}

// hiveRetrievalConsentPath resolves the current colony's consent record. The
// colony data dir takes precedence; the working directory is the fallback so
// data-only subcommands still resolve it.
func hiveRetrievalConsentPath() string {
	if store != nil && strings.TrimSpace(store.BasePath()) != "" {
		return filepath.Join(store.BasePath(), hiveRetrievalConsentFile)
	}
	if envDir := strings.TrimSpace(os.Getenv("COLONY_DATA_DIR")); envDir != "" {
		return filepath.Join(envDir, hiveRetrievalConsentFile)
	}
	if root := strings.TrimSpace(os.Getenv("AETHER_ROOT")); root != "" {
		return filepath.Join(root, ".aether", "data", hiveRetrievalConsentFile)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return filepath.Join(cwd, ".aether", "data", hiveRetrievalConsentFile)
}

// hiveRetrievalOptedIn reports whether this colony has explicitly consented
// to receiving cross-project wisdom. Consent is per colony, always: the
// global policy only decides whether the feature exists, never whether this
// repository receives it.
func hiveRetrievalOptedIn() bool {
	path := hiveRetrievalConsentPath()
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var consent hiveRetrievalConsent
	if err := json.Unmarshal(raw, &consent); err != nil {
		return false
	}
	return consent.OptIn
}

func writeHiveRetrievalConsent(optIn bool) error {
	path := hiveRetrievalConsentPath()
	if path == "" {
		return fmt.Errorf("no colony data directory; run inside an initialized colony")
	}
	consent := hiveRetrievalConsent{OptIn: optIn, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	encoded, err := json.MarshalIndent(consent, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0644)
}

// --- hive-init ---

var hiveInitCmd = &cobra.Command{
	Use:   "hive-init",
	Short: "Initialize hive directory and empty wisdom file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		hiveDir := filepath.Join(hub, "hive")

		if err := os.MkdirAll(hiveDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create hive dir: %v", err), nil)
			return nil
		}

		wisdomPath := filepath.Join(hiveDir, "wisdom.json")
		if _, err := os.Stat(wisdomPath); err == nil {
			outputOK(map[string]interface{}{"initialized": true, "note": "already exists"})
			return nil
		}

		data := hiveWisdomData{Entries: []hiveWisdomEntry{}}
		encoded, _ := json.MarshalIndent(data, "", "  ")
		if err := os.WriteFile(wisdomPath, append(encoded, '\n'), 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write wisdom.json: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"initialized": true, "path": wisdomPath})
		return nil
	},
}

// --- hive-store ---

var hiveStoreCmd = &cobra.Command{
	Use:   "hive-store [text] [domain] [source-repo]",
	Short: "Store a wisdom entry with deduplication and LRU cap",
	Args:  cobra.MaximumNArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := mustGetStringCompat(cmd, args, "text", 0)
		if text == "" {
			return nil
		}
		domain := firstNonEmpty(mustGetStringCompatOptional(cmd, "domain"), optionalArg(args, 1))
		if domain == "" {
			domain = "general"
		}
		sourceRepo := mustGetStringCompat(cmd, args, "source-repo", 2)
		if sourceRepo == "" {
			return nil
		}

		hub := resolveHubPath()
		repoID := currentRepoIdentity()
		evidence := hiveEvidence{
			RepoID:    repoID,
			Kind:      "manual",
			Reference: "hive-store",
			At:        time.Now().UTC().Format(time.RFC3339),
		}

		var stored hiveWisdomEntry
		var outcome string
		storeErr := updateWisdomLocked(hub, func(wf *hiveWisdomData) error {
			entry, result, err := storeHiveWisdomEntry(wf, text, domain, sourceRepo, repoID, 0.5, evidence)
			if err != nil {
				return err
			}
			stored = entry
			outcome = result
			return nil
		})
		if storeErr != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", storeErr), nil)
			return nil
		}

		emitLifecycleCeremony(events.CeremonyTopicHiveStore, events.CeremonyPayload{
			TaskID:  stored.ID,
			Task:    domain,
			Status:  outcome,
			Message: text,
		}, "aether-hive")
		result := map[string]interface{}{
			"stored":     outcome != "quarantined",
			"reinforced": outcome == "reinforced",
			"id":         stored.ID,
			"status":     outcome,
		}
		if stored.Quarantined {
			result["quarantined"] = true
			result["quarantine_reason"] = stored.QuarantineReason
		}
		outputOK(result)
		return nil
	},
}

// storeHiveWisdomEntry applies one store or promote mutation inside the
// locked update. Exact duplicates of active entries reinforce them; exact
// duplicates of revoked entries are refused so revocation stays meaningful;
// contradictions are stored quarantined and cross-linked, never silently
// accepted. Returns the stored entry and an outcome label.
func storeHiveWisdomEntry(wf *hiveWisdomData, text, domain, sourceRepo, repoID string, confidence float64, evidence hiveEvidence) (hiveWisdomEntry, string, error) {
	// Hive text is injected into worker prompts in other repositories, so it is
	// an untrusted cross-colony channel and must be sanitized on the same terms
	// as pheromone signals. Pheromones have gone through SanitizeSignalContent
	// since v2.0; the hive never did, which left a path for one colony's stored
	// text to carry instruction-override content into another colony's prompts.
	sanitized, err := colony.SanitizeSignalContent(text)
	if err != nil {
		return hiveWisdomEntry{}, "", fmt.Errorf("hive text rejected by sanitizer: %w", err)
	}
	text = sanitized

	now := time.Now().UTC().Format(time.RFC3339)
	if confidence <= 0 {
		confidence = 0.5
	}

	for i := range wf.Entries {
		existing := &wf.Entries[i]
		if existing.Text != text || existing.Domain != domain {
			continue
		}
		if existing.Revoked {
			return hiveWisdomEntry{}, "", fmt.Errorf("entry %s is revoked (%s); run `aether hive-revoke --id %s --unrevoke` to restore it before storing the same text", existing.ID, firstNonEmpty(existing.RevokedReason, "no reason recorded"), existing.ID)
		}
		if existing.Quarantined {
			return hiveWisdomEntry{}, "", fmt.Errorf("entry %s is quarantined (%s); resolve the contradiction before reinforcing it", existing.ID, existing.QuarantineReason)
		}
		reinforceHiveWisdomEntry(existing, sourceRepo, repoID, confidence, evidence)
		existing.AccessCount++
		existing.AccessedAt = now
		return *existing, "reinforced", nil
	}

	evictHiveEntryForCapacity(wf)

	textHash := fmt.Sprintf("%x", sha256.Sum256([]byte(text)))
	entry := hiveWisdomEntry{
		ID:              fmt.Sprintf("%s_%s", domain, textHash[:12]),
		Text:            text,
		Domain:          domain,
		SourceRepo:      sourceRepo,
		SourceRepos:     uniqueSortedStrings([]string{sourceRepo}),
		Confidence:      confidence,
		CreatedAt:       now,
		AccessedAt:      now,
		AccessCount:     0,
		LastConfirmedAt: now,
	}
	if repoID != "" {
		entry.SourceRepoIDs = []string{repoID}
		entry.Evidence = []hiveEvidence{evidence}
	}

	contradictedIDs := make([]string, 0)
	for _, existing := range wf.Entries {
		if !hiveEntryActive(existing) || existing.Domain != domain {
			continue
		}
		if detectHiveContradiction(text, existing.Text) {
			contradictedIDs = append(contradictedIDs, existing.ID)
		}
	}
	outcome := "stored"
	if len(contradictedIDs) > 0 {
		entry.Quarantined = true
		entry.QuarantineReason = fmt.Sprintf("contradicts %s", strings.Join(contradictedIDs, ", "))
		entry.Contradicts = contradictedIDs
		for i := range wf.Entries {
			for _, id := range contradictedIDs {
				if wf.Entries[i].ID == id {
					wf.Entries[i].ContradictedBy = uniqueSortedStrings(append(wf.Entries[i].ContradictedBy, entry.ID))
				}
			}
		}
		outcome = "quarantined"
	}

	wf.Entries = append(wf.Entries, entry)
	return entry, outcome, nil
}

// --- hive-read ---

var hiveReadCmd = &cobra.Command{
	Use:   "hive-read",
	Short: "Read wisdom entries with optional domain and confidence filtering",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, _ := cmd.Flags().GetString("domain")
		minConfidence, _ := cmd.Flags().GetFloat64("min-confidence")
		forWorker, _ := cmd.Flags().GetBool("for-worker")
		includeAll, _ := cmd.Flags().GetBool("all")
		if forWorker && !automaticHiveReadEnabled() {
			outputOK(map[string]interface{}{
				"entries": []hiveWisdomEntry{},
				"total":   0,
				"policy":  string(currentHiveRuntimePolicy()),
				"enabled": false,
				"reason":  "automatic cross-project wisdom injection is disabled; set AETHER_HIVE_POLICY=read to opt in",
			})
			return nil
		}
		if forWorker && !hiveRetrievalOptedIn() {
			outputOK(map[string]interface{}{
				"entries": []hiveWisdomEntry{},
				"total":   0,
				"policy":  string(currentHiveRuntimePolicy()),
				"enabled": false,
				"reason":  "this colony has not consented to cross-project wisdom; run `aether hive-opt-in` to enable it here",
			})
			return nil
		}

		hub := resolveHubPath()
		now := time.Now().UTC()

		var results []hiveWisdomEntry
		readErr := updateWisdomLocked(hub, func(wf *hiveWisdomData) error {
			for i := range wf.Entries {
				e := &wf.Entries[i]
				if !includeAll && !hiveEntryActive(*e) {
					continue
				}
				if domain != "" && e.Domain != domain {
					continue
				}
				effective := effectiveHiveConfidence(*e, now)
				if minConfidence > 0 && effective < minConfidence {
					continue
				}
				e.AccessCount++
				e.AccessedAt = now.Format(time.RFC3339)
				result := *e
				result.EffectiveConfidence = effective
				results = append(results, result)
			}
			return nil
		})
		if readErr != nil {
			if strings.Contains(readErr.Error(), "not initialized") {
				outputOK(map[string]interface{}{"entries": []hiveWisdomEntry{}, "total": 0})
				return nil
			}
			outputError(2, fmt.Sprintf("failed to read hive wisdom: %v", readErr), nil)
			return nil
		}

		outputOK(map[string]interface{}{"entries": results, "total": len(results), "policy": string(currentHiveRuntimePolicy()), "enabled": true})
		return nil
	},
}

// --- hive-abstract ---

var hiveAbstractCmd = &cobra.Command{
	Use:   "hive-abstract [instinct]",
	Short: "Abstract repo-specific text into generalized wisdom",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instinct := mustGetStringCompat(cmd, args, "instinct", 0)
		if instinct == "" {
			return nil
		}
		sourceRepo, _ := cmd.Flags().GetString("source-repo")

		// Simple abstraction: remove repo-specific identifiers
		abstracted := instinct
		if sourceRepo != "" {
			abstracted = strings.ReplaceAll(abstracted, sourceRepo, "<repo>")
		}
		// Remove common repo path prefixes
		for _, prefix := range []string{"src/", "lib/", "pkg/", "cmd/", "internal/"} {
			abstracted = strings.ReplaceAll(abstracted, prefix, "")
		}

		outputOK(map[string]interface{}{
			"original":    instinct,
			"abstracted":  abstracted,
			"source_repo": sourceRepo,
		})
		return nil
	},
}

// --- hive-promote ---

// promoteToHive is a reusable function that abstracts text, stores it in the hive
// wisdom file, and emits a promotion event. It returns an error on failure so callers
// can decide whether to block or continue.
func promoteToHive(text, domain, sourceRepo string, confidence float64) error {
	return promoteToHiveWithReference(text, domain, sourceRepo, confidence, "hive-promote")
}

func promoteToHiveWithReference(text, domain, sourceRepo string, confidence float64, reference string) error {
	if text == "" {
		return nil
	}
	if domain == "" {
		domain = "general"
	}
	if confidence <= 0 {
		confidence = 0.75
	}

	// Replace the repository name so the claim reads sensibly elsewhere. Source
	// directory prefixes are deliberately preserved — stripping "src/", "pkg/"
	// and friends was labelled abstraction but was plain string replacement that
	// pointed entries at paths which do not exist, making them unverifiable.
	// Same reasoning as HiveStore.abstractContent in pkg/learn/hive_store.go.
	abstracted := text
	if sourceRepo != "" {
		abstracted = strings.ReplaceAll(abstracted, sourceRepo, "<repo>")
	}

	hub := resolveHubPath()
	repoID := currentRepoIdentity()
	evidence := hiveEvidence{
		RepoID:    repoID,
		Kind:      "promote",
		Reference: reference,
		At:        time.Now().UTC().Format(time.RFC3339),
	}

	var stored hiveWisdomEntry
	var outcome string
	if err := updateWisdomLocked(hub, func(wf *hiveWisdomData) error {
		entry, result, err := storeHiveWisdomEntry(wf, abstracted, domain, sourceRepo, repoID, confidence, evidence)
		if err != nil {
			return err
		}
		stored = entry
		outcome = result
		return nil
	}); err != nil {
		return fmt.Errorf("failed to save wisdom: %w", err)
	}

	emitLifecycleCeremony(events.CeremonyTopicHivePromote, events.CeremonyPayload{
		TaskID:  stored.ID,
		Task:    domain,
		Status:  outcome,
		Message: abstracted,
	}, "aether-hive")

	return nil
}

func reinforceHiveWisdomEntry(entry *hiveWisdomEntry, sourceRepo, repoID string, confidence float64, evidence hiveEvidence) {
	if entry == nil {
		return
	}
	repos := hiveSourceRepos(*entry)
	sourceRepo = strings.TrimSpace(sourceRepo)
	if sourceRepo != "" {
		repos = append(repos, sourceRepo)
	}
	repos = uniqueSortedStrings(repos)
	sort.Strings(repos)
	entry.SourceRepos = repos
	if strings.TrimSpace(entry.SourceRepo) == "" && len(repos) > 0 {
		entry.SourceRepo = repos[0]
	}
	if repoID != "" {
		entry.SourceRepoIDs = uniqueSortedStrings(append(entry.SourceRepoIDs, repoID))
	}
	if evidence.Kind != "" {
		entry.Evidence = append(entry.Evidence, evidence)
	}
	entry.LastConfirmedAt = time.Now().UTC().Format(time.RFC3339)
	boosted := confidence
	if tier := hiveConfidenceForRepoCount(hiveRepoConfirmationCount(*entry)); tier > boosted {
		boosted = tier
	}
	if boosted > entry.Confidence {
		entry.Confidence = boosted
	}
}

// hiveRepoConfirmationCount is the number of distinct repositories that
// confirmed an entry. Stable identities win; display names are only a legacy
// fallback for entries written before identities existed.
func hiveRepoConfirmationCount(entry hiveWisdomEntry) int {
	if len(entry.SourceRepoIDs) > 0 {
		return len(uniqueSortedStrings(entry.SourceRepoIDs))
	}
	return len(hiveSourceRepos(entry))
}

func hiveSourceRepos(entry hiveWisdomEntry) []string {
	repos := make([]string, 0, len(entry.SourceRepos)+1)
	if strings.TrimSpace(entry.SourceRepo) != "" {
		repos = append(repos, entry.SourceRepo)
	}
	repos = append(repos, entry.SourceRepos...)
	return uniqueSortedStrings(repos)
}

func hiveConfidenceForRepoCount(count int) float64 {
	switch {
	case count >= 4:
		return 0.95
	case count == 3:
		return 0.85
	case count == 2:
		return 0.70
	default:
		return 0
	}
}

var hivePromoteCmd = &cobra.Command{
	Use:   "hive-promote [text] [domain] [source-repo] [confidence]",
	Short: "End-to-end abstract + store pipeline for wisdom promotion",
	Args:  cobra.MaximumNArgs(4),
	RunE: func(cmd *cobra.Command, args []string) error {
		automatic, _ := cmd.Flags().GetBool("automatic")
		if automatic && !automaticHivePromotionEnabled() {
			outputOK(map[string]interface{}{
				"promoted": false,
				"skipped":  true,
				"policy":   string(currentHiveRuntimePolicy()),
				"reason":   "automatic cross-project promotion is disabled; set AETHER_HIVE_POLICY=promote to opt in",
			})
			return nil
		}
		text := mustGetStringCompat(cmd, args, "text", 0)
		if text == "" {
			return nil
		}
		domain := firstNonEmpty(mustGetStringCompatOptional(cmd, "domain"), optionalArg(args, 1))
		if domain == "" {
			domain = "general"
		}
		sourceRepo := firstNonEmpty(mustGetStringCompatOptional(cmd, "source-repo"), optionalArg(args, 2))
		confidence, _ := cmd.Flags().GetFloat64("confidence")
		if confidence <= 0 {
			if argConfidence := optionalArg(args, 3); argConfidence != "" {
				if parsed, err := strconv.ParseFloat(argConfidence, 64); err == nil && parsed > 0 {
					confidence = parsed
				}
			}
			if confidence <= 0 {
				confidence = 0.75
			}
		}

		if err := promoteToHive(text, domain, sourceRepo, confidence); err != nil {
			outputError(2, fmt.Sprintf("hive promotion failed: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{"promoted": true})
		return nil
	},
}

// --- hive-revoke ---

var hiveRevokeCmd = &cobra.Command{
	Use:   "hive-revoke --id <entry-id> [--reason \"why\"] [--unrevoke]",
	Short: "Revoke a wisdom entry so it is never retrieved, or restore it",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		reason, _ := cmd.Flags().GetString("reason")
		unrevoke, _ := cmd.Flags().GetBool("unrevoke")
		if strings.TrimSpace(id) == "" {
			outputError(2, "hive-revoke requires --id", nil)
			return nil
		}

		hub := resolveHubPath()
		var updated hiveWisdomEntry
		found := false
		err := updateWisdomLocked(hub, func(wf *hiveWisdomData) error {
			for i := range wf.Entries {
				if wf.Entries[i].ID != id {
					continue
				}
				found = true
				if unrevoke {
					wf.Entries[i].Revoked = false
					wf.Entries[i].RevokedAt = ""
					wf.Entries[i].RevokedReason = ""
					wf.Entries[i].LastConfirmedAt = time.Now().UTC().Format(time.RFC3339)
				} else {
					if wf.Entries[i].Revoked {
						return fmt.Errorf("entry %s is already revoked", id)
					}
					wf.Entries[i].Revoked = true
					wf.Entries[i].RevokedAt = time.Now().UTC().Format(time.RFC3339)
					wf.Entries[i].RevokedReason = strings.TrimSpace(reason)
				}
				updated = wf.Entries[i]
				return nil
			}
			if !found {
				return fmt.Errorf("no hive entry with id %s", id)
			}
			return nil
		})
		if err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":      updated.ID,
			"revoked": updated.Revoked,
			"reason":  updated.RevokedReason,
		})
		return nil
	},
}

// --- hive-opt-in / hive-opt-out ---

var hiveOptInCmd = &cobra.Command{
	Use:   "hive-opt-in",
	Short: "Consent this colony to receiving cross-project wisdom in worker context",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := writeHiveRetrievalConsent(true); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}
		outputOK(map[string]interface{}{
			"opt_in": true,
			"note":   "cross-project wisdom will be injected into this colony's worker context when AETHER_HIVE_POLICY allows reads",
		})
		return nil
	},
}

var hiveOptOutCmd = &cobra.Command{
	Use:   "hive-opt-out",
	Short: "Withdraw this colony's consent to receiving cross-project wisdom",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := writeHiveRetrievalConsent(false); err != nil {
			outputError(2, err.Error(), nil)
			return nil
		}
		outputOK(map[string]interface{}{"opt_in": false})
		return nil
	},
}

// --- eternal-init ---

// eternalInitCmd initializes the eternal memory fallback storage directory and file.
var eternalInitCmd = &cobra.Command{
	Use:          "eternal-init",
	Short:        "Initialize eternal memory fallback storage",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		hub := resolveHubPath()
		eternalDir := filepath.Join(hub, "eternal")

		if err := os.MkdirAll(eternalDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create eternal dir: %v", err), nil)
			return nil
		}

		memoryPath := filepath.Join(eternalDir, "memory.json")
		if _, err := os.Stat(memoryPath); err == nil {
			outputOK(map[string]interface{}{
				"initialized": true,
				"path":        memoryPath,
				"note":        "already exists",
			})
			return nil
		}

		// Initialize with empty entries
		emptyData := []byte(`{"entries":[]}
`)
		if err := os.WriteFile(memoryPath, emptyData, 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write memory.json: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"initialized": true,
			"path":        memoryPath,
		})
		return nil
	},
}

func init() {
	hiveStoreCmd.Flags().String("text", "", "Wisdom text (required)")
	hiveStoreCmd.Flags().String("domain", "", "Domain tag (required)")
	hiveStoreCmd.Flags().String("source-repo", "", "Source repository (required)")

	hiveReadCmd.Flags().String("domain", "", "Filter by domain")
	hiveReadCmd.Flags().Float64("min-confidence", 0, "Minimum confidence threshold")
	hiveReadCmd.Flags().Bool("for-worker", false, "Apply the automatic worker-injection policy")
	hiveReadCmd.Flags().Bool("all", false, "Include revoked and quarantined entries")

	hiveAbstractCmd.Flags().String("instinct", "", "Instinct text to abstract (required)")
	hiveAbstractCmd.Flags().String("source-repo", "", "Source repository")

	hivePromoteCmd.Flags().String("text", "", "Wisdom text (required)")
	hivePromoteCmd.Flags().String("domain", "", "Domain tag (required)")
	hivePromoteCmd.Flags().String("source-repo", "", "Source repository")
	hivePromoteCmd.Flags().Float64("confidence", 0.75, "Confidence score")
	hivePromoteCmd.Flags().Bool("automatic", false, "Apply the automatic cross-project promotion policy")

	hiveRevokeCmd.Flags().String("id", "", "Wisdom entry ID (required)")
	hiveRevokeCmd.Flags().String("reason", "", "Why this entry is revoked")
	hiveRevokeCmd.Flags().Bool("unrevoke", false, "Restore a revoked entry")

	rootCmd.AddCommand(hiveInitCmd)
	rootCmd.AddCommand(hiveStoreCmd)
	rootCmd.AddCommand(hiveReadCmd)
	rootCmd.AddCommand(hiveAbstractCmd)
	rootCmd.AddCommand(hivePromoteCmd)
	rootCmd.AddCommand(hiveRevokeCmd)
	rootCmd.AddCommand(hiveOptInCmd)
	rootCmd.AddCommand(hiveOptOutCmd)
	rootCmd.AddCommand(eternalInitCmd)
}
