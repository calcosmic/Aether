package cmd

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

// pheromoneWriteError carries the exact CLI exit code pheromoneWriteCmd's RunE
// used to report a given failure, so the thin RunE wrapper around
// writePheromoneSignal can reproduce byte-identical outputError(code, ...)
// calls without writePheromoneSignal itself depending on CLI rendering
// (writePheromoneSignal is also called directly by cmd/phase_end_signals.go's
// non-CLI emitters, which must never call outputError).
type pheromoneWriteError struct {
	code int
	msg  string
}

func (e *pheromoneWriteError) Error() string { return e.msg }

// writePheromoneSignal is the extracted body of pheromoneWriteCmd's RunE
// (198.1-04, FEED-04): id generation, content hashing before sanitization,
// colony.SanitizeSignalContent, the {"text": "..."} content envelope,
// default priorities per type, SourcePhase population, and the type-based
// expiry defaults are all unchanged from the original CLI command -- this
// extraction moves nothing else and changes no defaulting rule. It returns
// the stored signal, whether it reinforced an existing one, and any error;
// it never calls outputError/outputOK itself so non-CLI callers (the
// phase-completion, decision-answer, and midden-threshold emitters) can
// write a signal without producing CLI output.
func writePheromoneSignal(sigType, content, priority, source, reason, ttl string, strength float64, tags []string) (colony.PheromoneSignal, bool, error) {
	if store == nil {
		return colony.PheromoneSignal{}, false, fmt.Errorf("no store initialized")
	}

	if sigType == "" || content == "" {
		return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 1, msg: "flags --type and --content are required"}
	}

	sigType = strings.ToUpper(sigType)
	switch sigType {
	case "FOCUS", "REDIRECT", "FEEDBACK":
	default:
		return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 1, msg: fmt.Sprintf("invalid type %q: must be FOCUS, REDIRECT, or FEEDBACK", sigType)}
	}

	if priority == "" {
		switch sigType {
		case "FOCUS":
			priority = "normal"
		case "REDIRECT":
			priority = "high"
		case "FEEDBACK":
			priority = "low"
		}
	}

	if strength == 0 {
		strength = 1.0
	}

	// Parse --ttl flag if provided
	var ttlDuration time.Duration
	if ttl != "" {
		d, err := parseTTL(ttl)
		if err != nil {
			return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 1, msg: fmt.Sprintf("invalid --ttl format %q: %s", ttl, err.Error())}
		}
		ttlDuration = d
	}

	// Generate ID: sig_<timestamp>_<random>
	rnd := make([]byte, 4)
	rand.Read(rnd)
	id := fmt.Sprintf("sig_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))

	now := time.Now().UTC().Format(time.RFC3339)

	// Compute content hash: SHA-256 of raw content (before sanitization)
	// so deduplication compares against raw input.
	h := sha256Sum(content)
	contentHash := "sha256:" + h

	// Sanitize content after hashing but before storage.
	sanitized, err := colony.SanitizeSignalContent(content)
	if err != nil {
		return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 1, msg: fmt.Sprintf("invalid signal content: %v", err)}
	}

	// Build content as JSON object matching shell format: {"text": "..."}
	contentJSON, _ := json.Marshal(map[string]string{"text": sanitized})

	signal := colony.PheromoneSignal{
		ID:          id,
		Type:        sigType,
		Content:     json.RawMessage(contentJSON),
		Priority:    priority,
		Source:      source,
		CreatedAt:   now,
		Active:      true,
		Strength:    &strength,
		ContentHash: &contentHash,
		Tags:        make([]colony.PheromoneTag, 0, len(tags)),
	}

	// Populate source_phase from current colony state
	var cs colony.ColonyState
	if loadErr := store.LoadJSON("COLONY_STATE.json", &cs); loadErr == nil && cs.CurrentPhase > 0 {
		signal.SourcePhase = &cs.CurrentPhase
	}

	if reason != "" {
		signal.Reason = &reason
	}

	if len(tags) > 0 {
		for _, t := range tags {
			signal.Tags = append(signal.Tags, colony.PheromoneTag{
				Value:    t,
				Category: "custom",
			})
		}
	}

	// Compute expiry: --ttl overrides type-based defaults
	if ttl != "" {
		expires := time.Now().UTC().Add(ttlDuration).Format(time.RFC3339)
		signal.ExpiresAt = &expires
	} else {
		switch sigType {
		case "REDIRECT":
			expires := time.Now().UTC().Add(30 * 24 * time.Hour).Format(time.RFC3339)
			signal.ExpiresAt = &expires
		case "FEEDBACK":
			expires := time.Now().UTC().Add(7 * 24 * time.Hour).Format(time.RFC3339)
			signal.ExpiresAt = &expires
			// FOCUS: no ExpiresAt (expires at phase end)
		}
	}

	// Load existing pheromones file
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		pf = colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}
	}
	if pf.Signals == nil {
		pf.Signals = []colony.PheromoneSignal{}
	}

	// Dedup: check for existing active signal with same type + content_hash
	replaced := false
	for i := range pf.Signals {
		sig := &pf.Signals[i]
		if !sig.Active {
			continue
		}
		if sig.Type == sigType && sig.ContentHash != nil && *sig.ContentHash == contentHash {
			// Reinforce existing signal instead of appending
			sig.CreatedAt = now
			if sig.ReinforcementCount == nil {
				rc := 0
				sig.ReinforcementCount = &rc
			}
			*sig.ReinforcementCount++
			maxStr := 1.0
			sig.Strength = &maxStr
			// Update source_phase on reinforcement
			var rcs colony.ColonyState
			if loadErr := store.LoadJSON("COLONY_STATE.json", &rcs); loadErr == nil && rcs.CurrentPhase > 0 {
				sig.SourcePhase = &rcs.CurrentPhase
			}
			signal = *sig
			replaced = true
			break
		}
	}

	if !replaced {
		pf.Signals = append(pf.Signals, signal)
	}

	if err := store.SaveJSON("pheromones.json", pf); err != nil {
		return colony.PheromoneSignal{}, false, &pheromoneWriteError{code: 2, msg: fmt.Sprintf("failed to save pheromones: %v", err)}
	}

	// Trace pheromone write if colony state has a run_id
	if tracer != nil {
		var state colony.ColonyState
		if loadErr := store.LoadJSON("COLONY_STATE.json", &state); loadErr == nil && state.RunID != nil {
			_ = tracer.LogPheromone(*state.RunID, sigType, "pheromone-write")
		}
	}
	status := "created"
	if replaced {
		status = "reinforced"
	}
	emitLifecycleCeremony(events.CeremonyTopicPheromoneEmit, events.CeremonyPayload{
		PheromoneType: sigType,
		Strength:      strength,
		Status:        status,
		Message:       extractText(signal.Content),
	}, "aether-pheromone")

	return signal, replaced, nil
}

var pheromoneWriteCmd = &cobra.Command{
	Use:   "pheromone-write",
	Short: "Create a new pheromone signal",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sigType, _ := cmd.Flags().GetString("type")
		content, _ := cmd.Flags().GetString("content")
		priority, _ := cmd.Flags().GetString("priority")
		strength, _ := cmd.Flags().GetFloat64("strength")
		sourceFlag, _ := cmd.Flags().GetString("source")
		reasonFlag, _ := cmd.Flags().GetString("reason")
		ttlFlag, _ := cmd.Flags().GetString("ttl")
		var tags []string

		if sigType == "" || content == "" {
			outputError(1, "flags --type and --content are required", nil)
			return nil
		}

		signal, replaced, err := writePheromoneSignal(sigType, content, priority, sourceFlag, reasonFlag, ttlFlag, strength, tags)
		if err != nil {
			if pwErr, ok := err.(*pheromoneWriteError); ok {
				outputError(pwErr.code, pwErr.msg, nil)
			} else {
				outputError(1, err.Error(), nil)
			}
			return nil
		}

		var pf colony.PheromoneFile
		total := 0
		if loadErr := store.LoadJSON("pheromones.json", &pf); loadErr == nil {
			total = len(pf.Signals)
		}

		outputOK(map[string]interface{}{
			"created":  true,
			"signal":   signal,
			"total":    total,
			"replaced": replaced,
		})
		return nil
	},
}

var pheromoneExpireCmd = &cobra.Command{
	Use:   "pheromone-expire",
	Short: "Expire a pheromone signal by ID",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		sigID := mustGetString(cmd, "id")
		if sigID == "" {
			return nil
		}

		var pf colony.PheromoneFile
		if err := store.LoadJSON("pheromones.json", &pf); err != nil {
			outputError(1, "pheromones.json not found", nil)
			return nil
		}

		found := false
		now := time.Now().UTC().Format(time.RFC3339)
		var expiredSignal colony.PheromoneSignal
		for i := range pf.Signals {
			if pf.Signals[i].ID == sigID {
				pf.Signals[i].Active = false
				pf.Signals[i].ExpiresAt = &now
				expiredSignal = pf.Signals[i]
				found = true
				break
			}
		}

		if !found {
			outputError(1, fmt.Sprintf("signal %q not found", sigID), nil)
			return nil
		}

		if err := store.SaveJSON("pheromones.json", pf); err != nil {
			outputError(2, fmt.Sprintf("failed to save: %v", err), nil)
			return nil
		}

		// A high-value signal (REDIRECT, or ever reinforced) is preserved in
		// long-term memory before it is gone for good (198.1-04, FEED-04).
		// Best-effort: a hub write failure never fails the expiry itself.
		promoted := promoteExpiredSignalToEternal(expiredSignal)

		outputOK(map[string]interface{}{
			"expired":  true,
			"id":       sigID,
			"promoted": promoted,
		})
		return nil
	},
}

var pheromoneValidateXMLCmd = &cobra.Command{
	Use:   "pheromone-validate-xml",
	Short: "Validate pheromone XML structure (placeholder)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputOK(map[string]interface{}{
			"valid": true,
		})
		return nil
	},
}

// expireSignalsByType deactivates all active signals of the given type.
// Returns the count of signals expired and the count promoted into
// long-term (eternal) memory before they were deactivated (198.1-04,
// FEED-04: a REDIRECT, or a signal ever reinforced, is preserved rather than
// lost when it expires). Uses deactivateSignal for consistency. This is an
// internal helper, not a CLI command.
func expireSignalsByType(s *storage.Store, sigType string) (expired int, promoted int) {
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return 0, 0
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for i := range pf.Signals {
		sig := &pf.Signals[i]
		if sig.Active && sig.Type == sigType {
			toPromote := *sig
			deactivateSignal(sig, now)
			expired++
			if promoteExpiredSignalToEternal(toPromote) {
				promoted++
			}
		}
	}
	if expired > 0 {
		_ = s.SaveJSON("pheromones.json", pf)
	}
	return expired, promoted
}

func init() {
	pheromoneWriteCmd.Flags().String("type", "", "Signal type: FOCUS, REDIRECT, or FEEDBACK (required)")
	pheromoneWriteCmd.Flags().String("content", "", "Signal content (required)")
	pheromoneWriteCmd.Flags().String("priority", "", "Priority: low, normal, high (default based on type)")
	pheromoneWriteCmd.Flags().Float64("strength", 0, "Signal strength (default 1.0)")
	pheromoneWriteCmd.Flags().StringArray("tag", nil, "Signal tags (repeatable)")
	pheromoneWriteCmd.Flags().String("source", "cli", "Signal source (default \"cli\")")
	pheromoneWriteCmd.Flags().String("reason", "", "Reason for the signal (optional)")
	pheromoneWriteCmd.Flags().String("ttl", "", "Override expiry duration: Nd (days), Nh (hours), Nw (weeks) (optional)")

	pheromoneExpireCmd.Flags().String("id", "", "Signal ID to expire (required)")
	pheromoneExpireCmd.Flags().Bool("phase-end-only", false, "Only expire signals marked as phase-scoped")

	rootCmd.AddCommand(pheromoneWriteCmd)
	rootCmd.AddCommand(pheromoneExpireCmd)
	rootCmd.AddCommand(pheromoneValidateXMLCmd)
}

// parseTTL parses a duration string like "30d", "7d", "24h", "1w" into a time.Duration.
func parseTTL(s string) (time.Duration, error) {
	re := regexp.MustCompile(`^(\d+)([dhwm])$`)
	m := re.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("must match format: <number><unit> where unit is d (days), h (hours), w (weeks), or m (minutes)")
	}
	val, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number %q: %w", m[1], err)
	}
	switch m[2] {
	case "d":
		return time.Duration(val) * 24 * time.Hour, nil
	case "h":
		return time.Duration(val) * time.Hour, nil
	case "w":
		return time.Duration(val) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(val) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unsupported unit %q", m[2])
	}
}

// sha256Sum returns the hex-encoded SHA-256 hash of the input string.
func sha256Sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

// generateSignalID creates a pheromone signal ID.
func generateSignalID() string {
	rnd := make([]byte, 4)
	rand.Read(rnd)
	return fmt.Sprintf("sig_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))
}

// generateFlagID creates a flag ID.
func generateFlagID() string {
	rnd := make([]byte, 4)
	rand.Read(rnd)
	return fmt.Sprintf("flag_%d_%s", time.Now().Unix(), hex.EncodeToString(rnd))
}

// mustGetDuration retrieves a duration flag in days, returning a default.
func mustGetDuration(cmd *cobra.Command, flag string, defaultDays int) int {
	val, err := cmd.Flags().GetInt(flag)
	if err != nil || val == 0 {
		return defaultDays
	}
	return val
}
