package learn

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AutoSkillMode controls how auto-skills are created after difficult tasks (AUTO-01).
// REQUIREMENTS.md specifies default as "propose" (not "auto").
const (
	AutoSkillModeOff     = "off"     // No auto-skill creation
	AutoSkillModePropose = "propose" // Skill proposed but not created (requires approval)
	AutoSkillModeAuto    = "auto"    // Skill created immediately
)

// AutoSkillModeDefault is the default mode per REQUIREMENTS.md AUTO-01.
const AutoSkillModeDefault = AutoSkillModePropose

// DifficultyScoreThreshold is the minimum difficulty score to trigger auto-skill creation.
// Tasks scoring below this are "easy" and do not warrant a permanent skill.
const DifficultyScoreThreshold = 0.3

// DifficultyAssessment represents the result of difficulty analysis (D-05).
type DifficultyAssessment struct {
	IsDifficult bool     `json:"is_difficult"`
	Reasons     []string `json:"reasons,omitempty"`
	Score       float64  `json:"score"` // 0.0-1.0, higher = more difficult
}

// AutoSkillConfig holds the configuration for auto-skill creation.
type AutoSkillConfig struct {
	Mode string `json:"mode"` // off, propose, auto
}

// SkillProposalSink lets AutoCreateSkillIfDifficult hand off a
// difficulty-triggered skill candidate without pkg/learn importing package
// cmd (pkg/learn must never import cmd). The real implementation
// (cmd/suggest_approve.go's colonySkillProposalSink) enqueues the candidate
// into the SAME owner tick-to-approve queue (colony.PendingSuggestion)
// every other pending decision already uses -- LEARN-07,
// 204-CLASSIC-SYNTHESIS.md ruling (d): "propose" mode must actually
// propose a candidate the owner can act on, not silently do nothing while
// claiming otherwise.
type SkillProposalSink interface {
	ProposeSkill(proposal SkillProposal) error
}

// SkillProposal is everything a difficulty-triggered skill candidate
// carries into the owner's approval queue: the derived name, the generated
// content, how confident the runtime is, which run produced it, and which
// learning entry it came from -- so its provenance is nameable without
// opening a file.
type SkillProposal struct {
	Name            string
	Content         string
	Confidence      float64
	SourceRunID     string
	LearningEntryID string
}

// LoadAutoSkillMode reads the auto_skill_mode from config file or returns default.
// Config file location: .aether/data/auto_skill_mode (plain text, one word).
// If file does not exist or is unreadable, returns AutoSkillModeDefault ("propose").
func LoadAutoSkillMode(dataDir string) string {
	data, err := os.ReadFile(filepath.Join(dataDir, "auto_skill_mode"))
	if err != nil {
		return AutoSkillModeDefault
	}
	mode := strings.TrimSpace(string(data))
	switch mode {
	case AutoSkillModeOff, AutoSkillModePropose, AutoSkillModeAuto:
		return mode
	default:
		return AutoSkillModeDefault
	}
}

// AssessDifficulty analyzes evidence to determine if a task was "difficult" (D-05).
// A task is difficult if: worker retries, gate failures before pass, or time overruns.
func AssessDifficulty(evidence Evidence) DifficultyAssessment {
	var reasons []string
	var score float64

	// Check 1: Worker failures before success (retries)
	failures := 0
	for _, w := range evidence.Workers {
		if w.Status != "completed" && w.Status != "done" {
			failures++
		}
	}
	if failures > 0 && evidence.GatesPassed > 0 {
		reasons = append(reasons, fmt.Sprintf("%d worker(s) failed before success", failures))
		weight := float64(failures) / float64(len(evidence.Workers))
		if weight > 1.0 {
			weight = 1.0
		}
		score += 0.3 * weight
	}

	// Check 2: Gates failed before passing (replan/retry)
	if evidence.GatesTotal > 0 && evidence.GatesPassed > 0 {
		failedGates := evidence.GatesTotal - evidence.GatesPassed
		if failedGates > 0 {
			reasons = append(reasons, fmt.Sprintf("%d gate(s) failed before passing", failedGates))
			score += 0.2 * float64(failedGates) / float64(evidence.GatesTotal)
		}
	}

	// Check 3: Multiple workers indicates complex task
	if len(evidence.Workers) >= 3 {
		reasons = append(reasons, fmt.Sprintf("%d workers involved", len(evidence.Workers)))
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}

	return DifficultyAssessment{
		IsDifficult: score >= DifficultyScoreThreshold || len(reasons) > 0,
		Reasons:     reasons,
		Score:       score,
	}
}

// IsAutoSkillRejected checks hard rejection rules (D-07, AUTO-02).
// Returns true if the entry should NOT produce an auto-skill.
// Hard rejection rules:
//   - Classification is blocked (secrets found)
//   - Entry is redacted (secrets removed)
//   - Zero files touched (no modifications = no pattern to capture)
//   - Content is empty (nothing to learn)
func IsAutoSkillRejected(entry Entry) (bool, string) {
	if entry.Classification == ClassBlocked {
		return true, "entry classification is blocked (secrets detected)"
	}
	if entry.Redacted {
		return true, "entry contains redacted secrets"
	}
	if len(entry.Evidence.FilesTouched) == 0 {
		return true, "zero files touched (no modification pattern to capture)"
	}
	if strings.TrimSpace(entry.Content) == "" {
		return true, "empty content (nothing to learn)"
	}
	return false, ""
}

// AutoCreateSkillIfDifficult assesses difficulty and, when warranted, hands
// a skill candidate to sink -- it never creates an active skill directly,
// in ANY mode (AUTO-01, LEARN-07, 204-CLASSIC-SYNTHESIS.md ruling (d)).
//
// mode controls whether a proposal is raised at all, never whether the
// owner is bypassed: "off" raises nothing; "propose" and "auto" both raise
// the IDENTICAL proposal through sink, so the difference between them is
// how eagerly a proposal is raised, not whether the owner approves it.
// Prior to this, "auto" mode created a skill with zero owner involvement,
// and "propose" mode's own doc comment claimed a candidate was "logged" --
// it never wrote or returned anything at all. Both defects are closed
// here: every mode that raises a candidate routes it into the same
// tick-to-approve queue every other pending decision already uses.
//
// Returns nil if no proposal was raised (easy task, rejected, or mode is
// off). Returns error only if the proposal itself could not be handed to
// sink (caller decides how to handle).
func AutoCreateSkillIfDifficult(entry Entry, mode string, sink SkillProposalSink) error {
	// Mode check: off = skip entirely
	if mode == AutoSkillModeOff {
		return nil
	}

	// Hard rejection check (D-07, AUTO-02)
	rejected, _ := IsAutoSkillRejected(entry)
	if rejected {
		return nil // Silent skip -- not an error
	}

	// Difficulty assessment (D-05)
	assessment := AssessDifficulty(entry.Evidence)
	if !assessment.IsDifficult {
		return nil // Easy task -- no skill needed
	}

	// Generate skill name from entry content
	skillName := deriveSkillName(entry)
	if skillName == "" {
		return nil // Cannot derive a meaningful name
	}

	if sink == nil {
		return fmt.Errorf("auto-skill mode %q raised a candidate with no proposal sink to hand it to", mode)
	}

	// Build skill content from entry (AUTO-03's evidence-bearing shape,
	// unchanged from the pre-existing content builder)
	content := buildSkillContent(entry, assessment)

	return sink.ProposeSkill(SkillProposal{
		Name:            skillName,
		Content:         content,
		Confidence:      entry.Confidence,
		SourceRunID:     entry.Evidence.RunID,
		LearningEntryID: entry.ID,
	})
}

// deriveSkillName generates a kebab-case skill name from entry content.
// Uses phase, content keywords, and a hash for uniqueness.
func deriveSkillName(entry Entry) string {
	if entry.Phase > 0 {
		// Use phase-based naming with content hash for uniqueness
		hash := sha256.Sum256([]byte(entry.Content))
		shortHash := fmt.Sprintf("%x", hash)[:8]
		// Extract key words from content (first 3 meaningful words)
		words := extractKeywords(entry.Content)
		if len(words) > 0 {
			return fmt.Sprintf("phase%d-%s-%s", entry.Phase, strings.Join(words, "-"), shortHash)
		}
		return fmt.Sprintf("phase%d-pattern-%s", entry.Phase, shortHash)
	}
	return ""
}

// extractKeywords pulls meaningful words from content for skill naming.
func extractKeywords(content string) []string {
	// Simple keyword extraction: lowercase, split on spaces/punctuation,
	// skip common words, take first 3
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "in": true, "for": true,
		"to": true, "of": true, "and": true, "is": true, "with": true,
		"was": true, "completed": true, "successfully": true, "phase": true,
	}
	var keywords []string
	lower := strings.ToLower(content)
	for _, word := range strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == ':' || r == ';' || r == '(' || r == ')'
	}) {
		if len(word) < 3 || stopWords[word] {
			continue
		}
		keywords = append(keywords, word)
		if len(keywords) >= 3 {
			break
		}
	}
	return keywords
}

// buildSkillContent generates the markdown body of an auto-created skill from entry data.
func buildSkillContent(entry Entry, assessment DifficultyAssessment) string {
	var b strings.Builder

	b.WriteString("# ")
	b.WriteString(filepath.Base(entry.Content))
	b.WriteString("\n\n")

	b.WriteString("## Difficulty Assessment\n\n")
	b.WriteString(fmt.Sprintf("- **Score:** %.2f / 1.0\n", assessment.Score))
	b.WriteString(fmt.Sprintf("- **Threshold:** %.2f\n", DifficultyScoreThreshold))
	if len(assessment.Reasons) > 0 {
		b.WriteString("- **Reasons:**\n")
		for _, r := range assessment.Reasons {
			b.WriteString(fmt.Sprintf("  - %s\n", r))
		}
	}

	b.WriteString("\n## Evidence\n\n")
	b.WriteString(fmt.Sprintf("- **Run ID:** %s\n", entry.Evidence.RunID))
	b.WriteString(fmt.Sprintf("- **Phase:** %d\n", entry.Evidence.Phase))
	b.WriteString(fmt.Sprintf("- **Gates:** %d / %d passed\n", entry.Evidence.GatesPassed, entry.Evidence.GatesTotal))
	b.WriteString(fmt.Sprintf("- **Confidence:** %.2f\n", entry.Confidence))
	b.WriteString(fmt.Sprintf("- **Privacy Scan:** passed\n"))

	if len(entry.Evidence.Workers) > 0 {
		b.WriteString("\n## Workers\n\n")
		for _, w := range entry.Evidence.Workers {
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", w.Name, w.Caste, w.Status))
		}
	}

	if len(entry.Evidence.FilesTouched) > 0 {
		b.WriteString("\n## Files Touched\n\n")
		for _, f := range entry.Evidence.FilesTouched {
			b.WriteString(fmt.Sprintf("- `%s`\n", f))
		}
	}

	b.WriteString("\n## Pattern\n\n")
	b.WriteString(entry.Content)
	b.WriteString("\n")

	return b.String()
}

// RepoFingerprint generates a unique identifier for the current repo context.
// Used in auto-created skills to track which repo they originated from (AUTO-03).
func RepoFingerprint(repoPath string) string {
	hash := sha256.Sum256([]byte(repoPath))
	return fmt.Sprintf("%x", hash)[:16]
}
