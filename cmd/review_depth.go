package cmd

import (
	"strings"
	"sync"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ReviewDepth represents whether a phase should receive light or heavy review.
type ReviewDepth string

const (
	ReviewDepthLight ReviewDepth = "light"
	ReviewDepthHeavy ReviewDepth = "heavy"
)

// reviewDepthPathOverride allows tests to point at a temporary policy file.
// When empty, the default colony/policies/review-depth.yaml is used.
var reviewDepthPathOverride string

// reviewDepthPolicy mirrors the YAML structure for the review depth policy.
type reviewDepthPolicy struct {
	ReviewDepthVersion   string            `yaml:"review_depth_version"`
	HeavyKeywords        []string          `yaml:"heavy_keywords"`
	SecurityRiskKeywords []string          `yaml:"security_risk_keywords"`
	BlastRadiusKeywords  []string          `yaml:"blast_radius_keywords"`
	SmartDefaultReasons  map[string]string `yaml:"smart_default_reasons"`
}

var (
	loadedReviewDepthPolicy     *reviewDepthPolicy
	loadedReviewDepthPolicyOnce sync.Once
)

// loadReviewDepthPolicy loads the review depth policy from YAML.
// It caches the result and falls back to nil on any error so callers can use
// hardcoded defaults.
func loadReviewDepthPolicy() *reviewDepthPolicy {
	loadedReviewDepthPolicyOnce.Do(func() {
		path := reviewDepthPathOverride
		if path == "" {
			path = policyPath("review-depth")
		}
		var policy reviewDepthPolicy
		if err := loadYAMLPolicy(path, &policy); err == nil {
			loadedReviewDepthPolicy = &policy
		}
	})
	return loadedReviewDepthPolicy
}

// Fallback keyword slices (original hardcoded values).
var (
	heavyKeywordsFallback = []string{
		"security", "auth", "crypto", "secrets",
		"permissions", "compliance", "audit",
		"release", "deploy", "production", "ship", "launch",
	}
	securityRiskKeywordsFallback = []string{
		"security", "auth", "crypto", "secrets", "permissions",
		"compliance", "audit", "token", "session", "password",
	}
	blastRadiusKeywordsFallback = []string{
		"core runtime", "state mutation", "colony state", "state machine",
		"phase transition", "dispatch", "build command", "continue command",
		"verification depth", "planning depth",
	}
)

// getHeavyKeywords returns the heavy keywords from policy if available,
// otherwise the hardcoded fallback.
func getHeavyKeywords() []string {
	if p := loadReviewDepthPolicy(); p != nil && len(p.HeavyKeywords) > 0 {
		return p.HeavyKeywords
	}
	return heavyKeywordsFallback
}

// getSecurityRiskKeywords returns the security risk keywords from policy if available,
// otherwise the hardcoded fallback.
func getSecurityRiskKeywords() []string {
	if p := loadReviewDepthPolicy(); p != nil && len(p.SecurityRiskKeywords) > 0 {
		return p.SecurityRiskKeywords
	}
	return securityRiskKeywordsFallback
}

// getBlastRadiusKeywords returns the blast radius keywords from policy if available,
// otherwise the hardcoded fallback.
func getBlastRadiusKeywords() []string {
	if p := loadReviewDepthPolicy(); p != nil && len(p.BlastRadiusKeywords) > 0 {
		return p.BlastRadiusKeywords
	}
	return blastRadiusKeywordsFallback
}

// Fallback smart default reason strings.
const (
	fallbackReasonHighRisk   = "auto: security risk"
	fallbackReasonFinalPhase = "auto: final phase"
	fallbackReasonMediumRisk = "auto: high blast radius"
	fallbackReasonEarlyPhase = "auto: early phase"
	fallbackReasonLatePhase  = "auto: late phase"
	fallbackReasonStandard   = "auto: standard"
)

// getSmartDefaultReason returns the reason string for the given key from policy
// if available, otherwise the hardcoded fallback.
func getSmartDefaultReason(key string) string {
	if p := loadReviewDepthPolicy(); p != nil {
		if v, ok := p.SmartDefaultReasons[key]; ok && v != "" {
			return v
		}
	}
	switch key {
	case "high_risk":
		return fallbackReasonHighRisk
	case "final_phase":
		return fallbackReasonFinalPhase
	case "medium_risk":
		return fallbackReasonMediumRisk
	case "early_phase":
		return fallbackReasonEarlyPhase
	case "late_phase":
		return fallbackReasonLatePhase
	default:
		return fallbackReasonStandard
	}
}

// heavyKeywords is kept as a package-level var for backward compatibility with
// existing callers. It is initialized lazily via getHeavyKeywords().
var heavyKeywords = heavyKeywordsFallback

// resolveReviewDepth determines whether a phase gets light or heavy review.
// Priority: explicit heavy flag > explicit light flag > final phase > keyword match > default.
func resolveReviewDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool) ReviewDepth {
	// Explicit heavy flag overrides everything else.
	if heavyFlag {
		return ReviewDepthHeavy
	}
	// Explicit light flag honors the user's request, including on final phases.
	if lightFlag {
		return ReviewDepthLight
	}
	// Final phase defaults to heavy when the user did not ask for a lighter pass.
	if phase.ID == totalPhases {
		return ReviewDepthHeavy
	}
	// Keyword auto-detection triggers heavy review, BUT explicit light flag
	// overrides keyword match (user intent takes priority).
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
	for _, kw := range getHeavyKeywords() {
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

// resolveVerificationDepth determines the 3-level verification depth for a phase.
// Priority: explicit heavy flag -> explicit light flag -> explicit --verification-depth string -> keyword match -> smart default.
func resolveVerificationDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool, verificationDepthStr string) colony.VerificationDepth {
	// Explicit heavy flag overrides everything else.
	if heavyFlag {
		return colony.VerificationDepthHeavy
	}
	// Explicit light honors the user's request, including on final phases.
	if lightFlag {
		return colony.VerificationDepthLight
	}
	// Explicit --verification-depth string overrides smart defaults.
	if verificationDepthStr != "" {
		return colony.NormalizeVerificationDepth(verificationDepthStr)
	}
	// Keyword auto-detection triggers heavy review only when depth is automatic.
	if phaseHasHeavyKeywords(phase.Name) {
		return colony.VerificationDepthHeavy
	}
	// Mode-aware smart default based on phase position + code change risk.
	return resolveSmartVerificationDepth(phase, totalPhases)
}

// resolveVerificationDepthFlag returns the effective depth string for flag resolution.
// Boolean flags take priority: --heavy returns "heavy", --light returns "light".
// When both are set, heavy wins (heavier is safer).
// Otherwise returns the --verification-depth string value (may be empty for auto-detect).
func resolveVerificationDepthFlag(lightFlag, heavyFlag bool, verificationDepthStr string) string {
	if heavyFlag {
		return "heavy"
	}
	if lightFlag {
		return "light"
	}
	return verificationDepthStr
}

// resolveEffectiveContinueDepth resolves the verification depth for continue paths.
// It combines CLI flags (boolean and string) with persisted state depth, then
// passes the original boolean flags through to resolveVerificationDepth so that
// keyword-guard behavior (lightFlag blocking keyword-based heavy override) is
// preserved -- matching the aether build behavior.
func resolveEffectiveContinueDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool, verificationDepthStr string, stateDepth string) colony.VerificationDepth {
	effectiveDepthStr := resolveVerificationDepthFlag(lightFlag, heavyFlag, verificationDepthStr)
	if effectiveDepthStr == "" {
		effectiveDepthStr = strings.TrimSpace(stateDepth)
	}
	return resolveVerificationDepth(phase, totalPhases, lightFlag, heavyFlag, effectiveDepthStr)
}

// --- Smart depth default functions (Phase 85) ---

// securityRiskKeywords lists substrings that trigger "high" risk classification.
var securityRiskKeywords = securityRiskKeywordsFallback

// blastRadiusKeywords lists substrings that trigger "medium" risk classification.
var blastRadiusKeywords = blastRadiusKeywordsFallback

// phasePositionLevel classifies a phase by its position within the plan.
// Returns "final", "early", "late", or "intermediate".
func phasePositionLevel(phaseID, totalPhases int) string {
	if phaseID == totalPhases || totalPhases <= 1 {
		return "final"
	}
	threshold25 := float64(totalPhases) * 0.25
	threshold75 := float64(totalPhases) * 0.75
	if float64(phaseID) <= threshold25 {
		return "early"
	}
	if float64(phaseID) >= threshold75 {
		return "late"
	}
	return "intermediate"
}

// collectPhaseText concatenates all analyzable text from a phase into a
// single lowercased string for risk keyword matching.
func collectPhaseText(phase colony.Phase) string {
	var parts []string
	parts = append(parts, phase.Name)
	parts = append(parts, phase.Description)
	parts = append(parts, phase.SuccessCriteria...)
	for _, task := range phase.Tasks {
		parts = append(parts, task.Goal)
		parts = append(parts, task.Constraints...)
		parts = append(parts, task.Hints...)
		parts = append(parts, task.SuccessCriteria...)
	}
	return strings.ToLower(strings.Join(parts, " "))
}

// matchesAnyKeyword returns true if any keyword appears as a substring in text.
// The caller is responsible for ensuring text is lowercased.
func matchesAnyKeyword(text string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			return true
		}
	}
	return false
}

// phaseRiskLevel classifies a phase's risk as "high", "medium", or "low"
// based on keyword matching against phase name only (not description or task text)
// to avoid false positives from common words like "session", "token", "password".
func phaseRiskLevel(phase colony.Phase) string {
	nameLower := strings.ToLower(phase.Name)
	if matchesAnyKeyword(nameLower, getSecurityRiskKeywords()) {
		return "high"
	}
	if matchesAnyKeyword(nameLower, getBlastRadiusKeywords()) {
		return "medium"
	}
	return "low"
}

// resolveSmartPlanningDepth combines position and risk signals to select
// planning depth. Uses the "safer principle": higher depth wins when
// signals disagree.
func resolveSmartPlanningDepth(phase colony.Phase, totalPhases int) colony.PlanningDepth {
	risk := phaseRiskLevel(phase)
	position := phasePositionLevel(phase.ID, totalPhases)

	if risk == "high" || position == "final" {
		return colony.PlanningDepthDeep
	}
	if risk == "medium" || position == "late" {
		return colony.PlanningDepthStandard
	}
	if position == "early" {
		return colony.PlanningDepthLight
	}
	return colony.PlanningDepthStandard
}

// resolveSmartVerificationDepth combines mode, position, and risk signals to select
// verification depth. Same logic as resolveSmartPlanningDepth but returns
// verification depth values (heavy instead of deep).
func resolveSmartVerificationDepth(phase colony.Phase, totalPhases int) colony.VerificationDepth {
	// Mode overrides position/risk for discovery and production.
	switch phase.Mode {
	case colony.PhaseModeDiscovery:
		return colony.VerificationDepthLight
	case colony.PhaseModeProduction:
		// Production always gets at least standard, heavy for final/high-risk.
		risk := phaseRiskLevel(phase)
		position := phasePositionLevel(phase.ID, totalPhases)
		if risk == "high" || position == "final" {
			return colony.VerificationDepthHeavy
		}
		return colony.VerificationDepthStandard
	}

	// Prototype and maintenance use position + risk.
	risk := phaseRiskLevel(phase)
	position := phasePositionLevel(phase.ID, totalPhases)

	if risk == "high" || position == "final" {
		return colony.VerificationDepthHeavy
	}
	if risk == "medium" || position == "late" {
		return colony.VerificationDepthStandard
	}
	if position == "early" {
		return colony.VerificationDepthLight
	}
	return colony.VerificationDepthStandard
}

// resolveVerificationDepthSmart wraps NormalizeVerificationDepth with smart defaults.
// When depth is empty (no explicit user flag), it uses resolveSmartVerificationDepth
// to auto-select based on phase position and risk signals.
func resolveVerificationDepthSmart(depth string, phase colony.Phase, totalPhases int) (string, error) {
	normalized := colony.NormalizeVerificationDepth(depth)
	if depth != "" {
		return string(normalized), nil
	}
	return string(resolveSmartVerificationDepth(phase, totalPhases)), nil
}
