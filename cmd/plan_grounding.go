package cmd

import (
	"regexp"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// filePathPattern matches strings containing file paths with at least one path
// separator followed by a filename with an optional extension. Anchored to word
// boundaries (space, quote, backtick, paren, or start/end of string) to avoid
// false positives on English text like "step 1/2" or "next/version".
var filePathPattern = regexp.MustCompile(`(?:^|[\s"'` + "`" + `(])(?:[a-zA-Z0-9._-]+/){1,}[a-zA-Z0-9._-]+(?:\.[a-zA-Z0-9]+)?(?:$|[\s"'` + "`" + `)])`)

// researchPhaseKeywords identifies phases that legitimately lack file targets.
// These phases are exempt from grounding validation.
var researchPhaseKeywords = []string{"research", "survey", "architecture", "design", "planning", "discovery"}

// planGroundingWarning is returned when source anchors exist but no tasks
// in a phase contain concrete file/path references.
type planGroundingWarning struct {
	PhaseID         int    `json:"phase_id"`
	PhaseName       string `json:"phase_name"`
	AnchorCount     int    `json:"anchor_count"`
	UngroundedTasks int    `json:"ungrounded_tasks"`
}

// checkPlanGrounding scans plan phases for concrete file references.
// Returns warnings for non-research phases where source anchors exist but
// zero tasks contain file-path-like strings in Goal, Constraints, or Hints.
// Research/architecture phases are exempted (GROUND-07).
// The grounding gate is always soft -- it never returns an error.
func checkPlanGrounding(phases []colony.Phase, sourceAnchors []string) []planGroundingWarning {
	if len(sourceAnchors) == 0 {
		return nil
	}
	var warnings []planGroundingWarning
	for _, phase := range phases {
		// Exemption from grounding is decided by the phase's TYPED mode, never
		// by name keywords. The keyword path meant any phase with "design" or
		// "research" in its title escaped grounding validation entirely —
		// prose steering a gate. Discovery-mode phases legitimately lack file
		// targets; everything else must ground. The keyword check survives
		// only as a fallback for phases that predate typed modes.
		if phase.Mode.Valid() {
			if phase.Mode == colony.PhaseModeDiscovery {
				continue
			}
		} else if isResearchPhase(phase.Name) {
			continue
		}
		groundedCount := 0
		for _, task := range phase.Tasks {
			if isGroundedTask(task) {
				groundedCount++
			}
		}
		if groundedCount == 0 && len(phase.Tasks) > 0 {
			warnings = append(warnings, planGroundingWarning{
				PhaseID:         phase.ID,
				PhaseName:       phase.Name,
				AnchorCount:     len(sourceAnchors),
				UngroundedTasks: len(phase.Tasks),
			})
		}
	}
	return warnings
}

// isGroundedTask returns true if any task field contains a file path reference.
// Checks Goal, joined Constraints, and joined Hints against the filePathPattern,
// then checks individual Hints for extension-only matches via looksLikeFile.
func isGroundedTask(task colony.Task) bool {
	for _, text := range []string{task.Goal, strings.Join(task.Constraints, " "), strings.Join(task.Hints, " ")} {
		if filePathPattern.MatchString(text) {
			return true
		}
	}
	for _, hint := range task.Hints {
		if looksLikeFile(hint) {
			return true
		}
	}
	return false
}

// isResearchPhase returns true for phases that legitimately lack file targets.
// Case-insensitive check against known research/architecture keywords.
func isResearchPhase(name string) bool {
	lower := strings.ToLower(name)
	for _, keyword := range researchPhaseKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}
	return false
}

// looksLikeFile checks if a string contains a common source file extension,
// indicating it references an actual file rather than an abstract concept.
func looksLikeFile(s string) bool {
	sourceExts := []string{".go", ".ts", ".js", ".py", ".rs", ".java", ".yaml", ".json", ".toml", ".md"}
	for _, ext := range sourceExts {
		if strings.Contains(s, ext) {
			return true
		}
	}
	return false
}
