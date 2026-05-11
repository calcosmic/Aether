package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// CasteRelevanceProfile defines when a caste is relevant to a phase.
type CasteRelevanceProfile struct {
	Caste      string
	Keywords   []string
	Conditions []string // e.g., "mode==discovery", "risk==high"
	BaseScore  int
}

// CasteDispatch represents a caste the Queen has chosen to spawn.
type CasteDispatch struct {
	Caste     string
	Score     int
	Rationale string
	FlowType  string
}

// casteRelevanceRegistry holds profiles for dispatchable worker castes.
// Base scores are deliberately low (10-20) so castes only spawn when keywords match
// or a flow policy marks them as required.
var casteRelevanceRegistry = []CasteRelevanceProfile{
	{Caste: "builder", Keywords: []string{"implement", "build", "create", "add", "write", "fix", "code", "deploy"}, BaseScore: 20},
	{Caste: "watcher", Keywords: []string{"verify", "test", "validate", "check", "review", "quality"}, BaseScore: 20},
	{Caste: "scout", Keywords: []string{"research", "investigate", "survey", "analyze", "document", "readme", "spec", "explore"}, BaseScore: 20},
	{Caste: "route_setter", Keywords: []string{"plan", "route", "decompose", "structure", "organize"}, BaseScore: 15},
	{Caste: "architect", Keywords: []string{"design", "schema", "architecture", "interface", "boundary", "structure", "evaluate"}, BaseScore: 20},
	{Caste: "oracle", Keywords: []string{"research", "spike", "investigate", "evaluate", "unknown", "deep dive", "survey"}, Conditions: []string{"mode==discovery"}, BaseScore: 20},
	{Caste: "chaos", Keywords: []string{"resilience", "failure", "robustness", "crash", "error handling", "stress test"}, BaseScore: 15},
	{Caste: "archaeologist", Keywords: []string{"legacy", "migration", "modernize", "rewrite", "history", "refactor old"}, BaseScore: 20},
	{Caste: "gatekeeper", Keywords: []string{"auth", "crypto", "security", "token", "secrets", "permissions", "compliance", "audit"}, BaseScore: 20},
	{Caste: "auditor", Keywords: []string{"compliance", "audit", "production", "release", "quality gate", "standards"}, Conditions: []string{"mode==production"}, BaseScore: 20},
	{Caste: "probe", Keywords: []string{"test coverage", "edge case", "validation", "verify", "missing tests", "coverage gap"}, BaseScore: 15},
	{Caste: "measurer", Keywords: []string{"performance", "optimize", "latency", "scale", "benchmark", "memory", "cpu"}, BaseScore: 20},
	{Caste: "ambassador", Keywords: []string{"api", "sdk", "oauth", "external service", "external integration", "integration", "webhook", "third-party", "stripe", "sendgrid", "twilio", "openai", "aws", "azure", "gcp"}, BaseScore: 20},
	{Caste: "tracker", Keywords: []string{"bug", "fix", "regression", "investigate failure", "root cause", "issue"}, BaseScore: 20},
	{Caste: "weaver", Keywords: []string{"refactor", "cleanup", "modernize", "extract", "simplify", "restructure"}, BaseScore: 20},
	{Caste: "keeper", Keywords: []string{"knowledge", "pattern", "convention", "standard", "document", "preserve", "wisdom"}, BaseScore: 10},
	{Caste: "chronicler", Keywords: []string{"documentation", "docs", "guide", "readme", "changelog", "manual"}, BaseScore: 15},
	{Caste: "includer", Keywords: []string{"accessibility", "a11y", "wcag", "screen reader", "aria", "inclusive"}, BaseScore: 15},
	{Caste: "surveyor-provisions", Keywords: []string{"dependency", "dependencies", "provisions", "external", "integration", "stack", "package"}, BaseScore: 10},
	{Caste: "surveyor-nest", Keywords: []string{"architecture", "structure", "layout", "map", "chamber", "directory"}, BaseScore: 10},
	{Caste: "surveyor-disciplines", Keywords: []string{"convention", "discipline", "testing", "pattern", "practice", "standard"}, BaseScore: 10},
	{Caste: "surveyor-pathogens", Keywords: []string{"pathogen", "debt", "fragile", "risk", "bug", "health", "failure"}, BaseScore: 10},
	{Caste: "medic", Keywords: []string{"health", "diagnose", "repair", "heal", "fix state"}, BaseScore: 10},
	{Caste: "fixer", Keywords: []string{"auto-fix", "repair", "patch", "remediate", "self-heal"}, BaseScore: 10},
	{Caste: "porter", Keywords: []string{"deploy", "deliver", "ship", "publish", "release", "package"}, BaseScore: 10},
	{Caste: "sage", Keywords: []string{"wisdom", "synthesize", "learn", "pattern", "retrospective"}, BaseScore: 10},
}

// casteRelevanceScore returns a 0-100 score for how relevant a caste is to a phase.
func casteRelevanceScore(phase colony.Phase, caste string) int {
	profile := findProfile(caste)
	if profile == nil {
		return 0
	}

	score := profile.BaseScore
	text := collectPhaseText(phase)

	// Keyword matching
	keywordMatches := 0
	for _, kw := range profile.Keywords {
		if strings.Contains(text, kw) {
			keywordMatches++
		}
	}
	score += keywordMatches * 10

	// Condition matching
	for _, cond := range profile.Conditions {
		score += conditionScore(phase, cond)
	}

	// Special rules
	score = applySpecialRules(phase, caste, score)

	// Clamp to 0-100
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// queenOrchestrate decides which castes to dispatch for a given flow.
func queenOrchestrate(phase colony.Phase, flowType string, state colony.ColonyState) []CasteDispatch {
	var dispatches []CasteDispatch
	seen := make(map[string]struct{}, len(casteRelevanceRegistry))
	flowType = normalizeQueenFlowType(flowType)
	threshold := spawnThreshold(flowType, state)

	for _, profile := range casteRelevanceRegistry {
		if !casteAllowedForFlow(profile.Caste, flowType) {
			continue
		}
		if isCasteSuppressed(profile.Caste, flowType, phase, state) {
			continue
		}

		score := casteRelevanceScore(phase, profile.Caste)
		always := isAlwaysRequired(profile.Caste, flowType, phase, state)

		if always || score >= threshold {
			rationale := fmt.Sprintf("Score %d >= threshold %d for %s flow", score, threshold, flowType)
			if always {
				score = 100
				rationale = fmt.Sprintf("%s is always required for %s flow", profile.Caste, flowType)
			}
			if _, ok := seen[profile.Caste]; ok {
				continue
			}
			seen[profile.Caste] = struct{}{}
			dispatches = append(dispatches, CasteDispatch{
				Caste:     profile.Caste,
				Score:     score,
				Rationale: rationale,
				FlowType:  flowType,
			})
		}
	}

	return dispatches
}

// findProfile looks up a caste's relevance profile.
func findProfile(caste string) *CasteRelevanceProfile {
	for i := range casteRelevanceRegistry {
		if casteRelevanceRegistry[i].Caste == caste {
			return &casteRelevanceRegistry[i]
		}
	}
	return nil
}

// conditionScore adds score for matched conditions.
func conditionScore(phase colony.Phase, condition string) int {
	switch condition {
	case "mode==discovery":
		if phase.Mode == colony.PhaseModeDiscovery {
			return 20
		}
	case "mode==production":
		if phase.Mode == colony.PhaseModeProduction {
			return 20
		}
	}
	return 0
}

// applySpecialRules handles hardcoded caste rules.
func applySpecialRules(phase colony.Phase, caste string, score int) int {
	text := collectPhaseText(phase)

	switch caste {
	case "builder":
		// Always include if any task has implementation keywords
		if hasImplementationTask(phase.Tasks) {
			return 100
		}
	case "architect":
		// Auto-include for high-risk phases (security work needs design boundaries)
		if phaseRiskLevel(phase) == "high" {
			return 100
		}
	case "watcher":
		// Flow policy decides whether watcher is mandatory.
		return score
	case "oracle":
		// Auto-include for discovery mode
		if phase.Mode == colony.PhaseModeDiscovery {
			return 100
		}
	case "chaos":
		// Exclude for discovery mode
		if phase.Mode == colony.PhaseModeDiscovery {
			return 0
		}
	case "gatekeeper":
		// Auto-include if risk is high
		if phaseRiskLevel(phase) == "high" {
			return 100
		}
	case "auditor":
		// Auto-include for production mode or final phase
		if phase.Mode == colony.PhaseModeProduction {
			return 100
		}
	case "scout":
		// Boost for research-heavy phases
		if strings.Contains(text, "research") || strings.Contains(text, "investigate") {
			score += 15
		}
	}

	return score
}

// spawnThreshold returns the minimum score to spawn for a flow type.
func spawnThreshold(flowType string, state colony.ColonyState) int {
	switch flowType {
	case "build", "continue":
		if flowType == "continue" && stateVerificationDepth(state) == colony.VerificationDepthHeavy {
			return 25
		}
		return 30 // Allow more castes through; Queen filters later
	case "plan":
		return 40
	case "colonize", "swarm":
		return 35
	case "seal":
		return 50
	default:
		return 35
	}
}

// isAlwaysRequired checks if a caste must always spawn for a flow.
func isAlwaysRequired(caste, flowType string, phase colony.Phase, state colony.ColonyState) bool {
	switch flowType {
	case "build":
		if phase.Mode == colony.PhaseModeDiscovery {
			return caste == "watcher" || caste == "probe" // No builder for discovery
		}
		return caste == "builder" || caste == "watcher" || caste == "probe"
	case "continue":
		switch stateVerificationDepth(state) {
		case colony.VerificationDepthLight:
			return caste == "watcher"
		case colony.VerificationDepthHeavy:
			return caste == "watcher" || caste == "gatekeeper" || caste == "auditor" || caste == "probe"
		default:
			return caste == "watcher" || caste == "probe"
		}
	case "plan":
		return caste == "scout" || caste == "route_setter"
	case "colonize":
		return caste == "surveyor-provisions" ||
			caste == "surveyor-nest" ||
			caste == "surveyor-disciplines" ||
			caste == "surveyor-pathogens"
	case "swarm":
		return caste == "tracker" ||
			caste == "scout" ||
			caste == "archaeologist" ||
			caste == "builder" ||
			caste == "watcher"
	case "seal":
		switch stateVerificationDepth(state) {
		case colony.VerificationDepthLight:
			return false
		case colony.VerificationDepthHeavy:
			return caste == "gatekeeper" || caste == "auditor" || caste == "probe"
		default:
			return caste == "auditor" || caste == "probe"
		}
	}
	return false
}

func normalizeQueenFlowType(flowType string) string {
	flowType = strings.ToLower(strings.TrimSpace(flowType))
	if flowType == "" {
		return "build"
	}
	return flowType
}

func stateVerificationDepth(state colony.ColonyState) colony.VerificationDepth {
	return colony.NormalizeVerificationDepth(state.VerificationDepth)
}

func casteAllowedForFlow(caste, flowType string) bool {
	switch flowType {
	case "build":
		return !strings.HasPrefix(caste, "surveyor-")
	case "continue":
		return oneOf(caste, "watcher", "gatekeeper", "auditor", "probe", "measurer", "chaos", "includer", "keeper", "sage", "medic", "fixer")
	case "plan":
		return oneOf(caste, "scout", "route_setter", "architect", "oracle", "keeper", "chronicler", "includer", "gatekeeper")
	case "colonize":
		return strings.HasPrefix(caste, "surveyor-")
	case "swarm":
		return oneOf(caste, "tracker", "scout", "archaeologist", "builder", "watcher", "gatekeeper", "probe", "weaver", "medic", "fixer")
	case "seal":
		return oneOf(caste, "gatekeeper", "auditor", "probe", "porter", "chronicler", "keeper", "sage", "measurer", "includer")
	default:
		return !strings.HasPrefix(caste, "surveyor-")
	}
}

func isCasteSuppressed(caste, flowType string, phase colony.Phase, state colony.ColonyState) bool {
	if flowType == "build" && phase.Mode == colony.PhaseModeDiscovery {
		return oneOf(caste, "builder", "weaver", "fixer", "porter")
	}
	if flowType == "seal" && stateVerificationDepth(state) == colony.VerificationDepthLight {
		return oneOf(caste, "gatekeeper", "auditor", "probe")
	}
	if flowType == "continue" || flowType == "seal" {
		return oneOf(caste, "builder", "weaver", "tracker", "archaeologist", "ambassador")
	}
	return false
}

func oneOf(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if value == candidate {
			return true
		}
	}
	return false
}

// hasImplementationTask checks if any task has implementation keywords.
func hasImplementationTask(tasks []colony.Task) bool {
	// Use word-boundary matching to avoid false positives like "research" matching "search"
	implKeywords := []string{"implement", "build", "create", "fix", "add", "write", "code", "deploy"}
	for _, task := range tasks {
		text := strings.ToLower(" " + task.Goal + " ")
		for _, kw := range implKeywords {
			if strings.Contains(text, " "+kw+" ") {
				return true
			}
		}
	}
	return false
}

// HasCaste checks if a caste is in the dispatch list.
func HasCaste(dispatches []CasteDispatch, caste string) bool {
	for _, d := range dispatches {
		if d.Caste == caste {
			return true
		}
	}
	return false
}

// FilterCastesByMinScore filters dispatches below a score threshold.
func FilterCastesByMinScore(dispatches []CasteDispatch, minScore int) []CasteDispatch {
	var filtered []CasteDispatch
	for _, d := range dispatches {
		if d.Score >= minScore {
			filtered = append(filtered, d)
		}
	}
	return filtered
}
