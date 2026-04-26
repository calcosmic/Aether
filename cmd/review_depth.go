package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ReviewDepth represents whether a phase should receive light or heavy review.
type ReviewDepth string

const (
	ReviewDepthLight ReviewDepth = "light"
	ReviewDepthHeavy ReviewDepth = "heavy"
)

// heavyKeywords lists phase-name substrings that always trigger heavy review.
var heavyKeywords = []string{
	"security", "auth", "crypto", "secrets",
	"permissions", "compliance", "audit",
	"release", "deploy", "production", "ship", "launch",
}

// resolveReviewDepth determines whether a phase gets light or heavy review.
// Priority: final phase > heavy flag > keyword match > light/default.
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
	// Final phase is always heavy regardless of flags.
	if phase.ID == totalPhases {
		return ReviewDepthHeavy
	}
	// Explicit heavy flag overrides everything else.
	if heavyFlag {
		return ReviewDepthHeavy
	}
	// Keyword auto-detection triggers heavy review.
	if phaseHasHeavyKeywords(phase.Name) {
		return ReviewDepthHeavy
	}
	// Default to light for intermediate phases.
	return ReviewDepthLight
}

// phaseHasHeavyKeywords checks if a phase name contains any heavy keyword.
// Matching is case-insensitive and uses substring matching.
func phaseHasHeavyKeywords(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range heavyKeywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

// chaosShouldRunInLightMode deterministically returns true for ~30% of phases.
// Phase IDs where phaseID % 10 < 3 (i.e. ending in 0, 1, 2) get chaos runs.
func chaosShouldRunInLightMode(phaseID int) bool {
	return phaseID%10 < 3
}
