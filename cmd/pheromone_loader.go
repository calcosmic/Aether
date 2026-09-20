package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/calcosmic/Aether/pkg/cache"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// loadPheromonesOnce loads pheromones.json using the session cache if available,
// falling back to a direct store.LoadJSON read. On a cache miss (or nil cache),
// the result is stored in the cache for subsequent calls within the same session.
func loadPheromonesOnce(s *storage.Store, c *cache.SessionCache) (colony.PheromoneFile, error) {
	var pf colony.PheromoneFile

	if c != nil {
		fullPath := filepath.Join(s.BasePath(), "pheromones.json")
		if cached, ok := c.Get(fullPath); ok {
			// Re-marshal the interface{} from cache and unmarshal into typed struct.
			raw, err := json.Marshal(cached)
			if err == nil {
				if err := json.Unmarshal(raw, &pf); err == nil {
					return pf, nil
				}
			}
			// Cache data corrupted -- fall through to disk load
		}
	}

	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		return colony.PheromoneFile{}, err
	}

	if c != nil {
		fullPath := filepath.Join(s.BasePath(), "pheromones.json")
		_ = c.Set(fullPath, pf) // non-fatal: cache write failure doesn't affect result
	}

	return pf, nil
}

// loadPheromones loads pheromones.json using the global store, with no session cache.
// Returns nil if the file is missing or unreadable.
func loadPheromones() *colony.PheromoneFile {
	if store == nil {
		return nil
	}
	var pf colony.PheromoneFile
	if err := store.LoadJSON("pheromones.json", &pf); err != nil {
		return nil
	}
	return &pf
}

// signalActiveForPrompt is a thin adapter over resolveEffectivePheromones for
// callers that need a single signal's in-effect decision. It no longer
// recomputes its own active/expiry/strength predicate -- see
// TestOneEffectivePheromonePredicate.
func signalActiveForPrompt(sig colony.PheromoneSignal, now time.Time) bool {
	resolved := resolveEffectivePheromones(&colony.PheromoneFile{Signals: []colony.PheromoneSignal{sig}}, now)
	return len(resolved) > 0 && resolved[0].InEffect
}

// filterSignalsForPrompt is a thin adapter over resolveEffectivePheromones,
// keeping its existing signature so its many callers need no change. It
// previously duplicated signalActiveForPrompt's filter inline; both now
// derive from the one resolver.
func filterSignalsForPrompt(signals []colony.PheromoneSignal, now time.Time) []colony.PheromoneSignal {
	resolved := resolveEffectivePheromones(&colony.PheromoneFile{Signals: signals}, now)
	filtered := make([]colony.PheromoneSignal, 0, len(resolved))
	for _, r := range resolved {
		if r.InEffect {
			filtered = append(filtered, r.Signal)
		}
	}
	return filtered
}

// extractSignalTextsFrom derives formatted, top-N signal texts from a
// pre-loaded PheromoneFile via the one resolver. This avoids a redundant
// disk read when pheromones have already been loaded by the caller, and
// fixes the missing-expiry gap the old inline filter/sort here shared with
// cmd/context.go's extractSignalTexts (NOW-09/NOW-10 disagreement).
func extractSignalTextsFrom(pf *colony.PheromoneFile, maxSignals int) []string {
	if pf == nil || len(pf.Signals) == 0 {
		return nil
	}

	now := time.Now()
	resolved := resolveEffectivePheromones(pf, now)

	var result []string
	for _, r := range resolved {
		if !r.InEffect {
			continue
		}
		text := extractSignalText(r.Signal.Content)
		if text == "" {
			continue
		}
		result = append(result, fmt.Sprintf("%s: %s", r.Signal.Type, text))
		if len(result) >= maxSignals {
			break
		}
	}
	return result
}
