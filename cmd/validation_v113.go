package cmd

import (
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/learn"
)

// validClassifications lists the allowed learning entry classification values.
var validClassifications = []string{
	string(learn.ClassBlocked),
	string(learn.ClassRepoLocal),
	string(learn.ClassHiveShareable),
	string(learn.ClassNeedsApproval),
}

// ValidateLearningEntry validates a learning entry, returning an actionable
// error message when the format is invalid. Error messages include: format
// name ("learning entry"), field name, expected value, and actual value.
func ValidateLearningEntry(entry learn.Entry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("learning entry: missing required field 'id'")
	}
	if entry.Phase <= 0 {
		return fmt.Errorf("learning entry: phase must be > 0, got %d", entry.Phase)
	}
	if strings.TrimSpace(entry.Content) == "" {
		return fmt.Errorf("learning entry: missing required field 'content'")
	}
	validClassification := false
	for _, c := range validClassifications {
		if string(entry.Classification) == c {
			validClassification = true
			break
		}
	}
	if !validClassification {
		return fmt.Errorf(
			"learning entry: invalid classification %q, valid values: %s",
			entry.Classification, strings.Join(validClassifications, ", "),
		)
	}
	if entry.Confidence < 0.0 || entry.Confidence > 1.0 {
		return fmt.Errorf(
			"learning entry: confidence must be between 0.0 and 1.0, got %v",
			entry.Confidence,
		)
	}
	if strings.TrimSpace(entry.Evidence.Timestamp) == "" {
		return fmt.Errorf("learning entry: missing required field 'evidence.timestamp'")
	}
	return nil
}

// ValidateSkillFrontmatter validates SKILL.md content by parsing its
// frontmatter and checking required fields. Error messages include: format
// name ("skill frontmatter"), field name, expected value, and actual value.
func ValidateSkillFrontmatter(content string) error {
	fm := parseSkillFrontmatter(content)
	if fm == nil {
		return fmt.Errorf("skill frontmatter: no frontmatter found, expected YAML block delimited by '---'")
	}
	if strings.TrimSpace(fm.Name) == "" {
		return fmt.Errorf("skill frontmatter: missing required field 'name'")
	}
	cat := strings.TrimSpace(fm.Category)
	if cat != "colony" && cat != "domain" {
		return fmt.Errorf(
			"skill frontmatter: invalid category %q, expected 'colony' or 'domain'",
			cat,
		)
	}
	if len(fm.Roles) == 0 {
		return fmt.Errorf("skill frontmatter: missing required field 'roles', expected non-empty array (e.g., [builder, watcher])")
	}
	return nil
}

// ValidateTrackedProcessJSON validates a slice of tracked processes,
// returning an actionable error message when the format is invalid.
// Error messages include: format name ("worker-processes"), field name,
// expected value, and actual value.
func ValidateTrackedProcessJSON(processes []codex.TrackedProcess) error {
	for i, proc := range processes {
		if proc.PID <= 0 {
			return fmt.Errorf(
				"worker-processes: process at index %d has PID %d, PID must be a positive integer",
				i, proc.PID,
			)
		}
		if strings.TrimSpace(proc.WorkerName) == "" {
			return fmt.Errorf(
				"worker-processes: process at index %d (PID %d) missing required field 'worker_name'",
				i, proc.PID,
			)
		}
		if proc.SpawnedAt.IsZero() {
			return fmt.Errorf(
				"worker-processes: process at index %d (PID %d) missing required field 'spawned_at'",
				i, proc.PID,
			)
		}
	}
	return nil
}
