package cmd

import (
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/learn"
)

// ValidateLearningEntry validates a learning entry.
// Stub: returns error to fail TDD RED phase.
func ValidateLearningEntry(entry learn.Entry) error {
	return nil
}

// ValidateSkillFrontmatter validates SKILL.md content.
// Stub: returns error to fail TDD RED phase.
func ValidateSkillFrontmatter(content string) error {
	return nil
}

// ValidateTrackedProcessJSON validates a slice of tracked processes.
// Stub: returns error to fail TDD RED phase.
func ValidateTrackedProcessJSON(processes []codex.TrackedProcess) error {
	return nil
}
