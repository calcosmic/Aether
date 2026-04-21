package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- TestLoadAgentInstructions ---

func TestLoadAgentInstructions_ValidTOML(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "valid-agent.toml")
	content := `name = "aether-builder"
description = "A builder agent"
nickname_candidates = ["builder", "hammer"]

developer_instructions = '''
You are a Builder Ant in the Aether Colony.
You implement code with TDD discipline.
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	instructions, err := LoadAgentInstructions(tomlPath)
	if err != nil {
		t.Fatalf("LoadAgentInstructions returned error: %v", err)
	}

	expected := "You are a Builder Ant in the Aether Colony.\nYou implement code with TDD discipline.\n"
	if instructions != expected {
		t.Errorf("instructions = %q, want %q", instructions, expected)
	}
}

func TestLoadAgentInstructions_MissingFile(t *testing.T) {
	_, err := LoadAgentInstructions("/nonexistent/path/agent.toml")
	if err == nil {
		t.Fatal("LoadAgentInstructions should return error for missing file")
	}
	if !strings.Contains(err.Error(), "read") {
		t.Errorf("error should mention 'read', got: %v", err)
	}
}

func TestLoadAgentInstructions_MissingDeveloperInstructions(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "no-instructions.toml")
	content := `name = "test-agent"
description = "An agent without developer instructions"
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	_, err := LoadAgentInstructions(tomlPath)
	if err == nil {
		t.Fatal("LoadAgentInstructions should return error when developer_instructions is missing")
	}
	if !strings.Contains(err.Error(), "developer_instructions") {
		t.Errorf("error should mention 'developer_instructions', got: %v", err)
	}
}

// --- TestAssemblePrompt ---

func TestAssemblePrompt_CombinesAllSections(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
You are a Builder Ant.
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	capsule := "--- CONTEXT CAPSULE ---\nGoal: Build feature X\n--- END CONTEXT CAPSULE ---"
	brief := "# Task 2.1\n\nImplement the thing."

	prompt, err := AssemblePrompt(tomlPath, capsule, "", "", brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	// Verify all three sections are present
	if !strings.Contains(prompt, "You are a Builder Ant.") {
		t.Error("prompt should contain developer_instructions")
	}
	if !strings.Contains(prompt, "--- CONTEXT CAPSULE ---") {
		t.Error("prompt should contain context capsule")
	}
	if !strings.Contains(prompt, "# Task 2.1") {
		t.Error("prompt should contain task brief")
	}

	// Verify ordering: instructions before capsule before brief
	instrIdx := strings.Index(prompt, "You are a Builder Ant.")
	capsuleIdx := strings.Index(prompt, "--- CONTEXT CAPSULE ---")
	briefIdx := strings.Index(prompt, "# Task 2.1")

	if instrIdx >= capsuleIdx {
		t.Error("developer_instructions should appear before context capsule")
	}
	if capsuleIdx >= briefIdx {
		t.Error("context capsule should appear before task brief")
	}
}

func TestAssemblePrompt_EmptyCapsule(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
You are a Builder Ant.
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	brief := "# Task 2.1\n\nImplement the thing."

	prompt, err := AssemblePrompt(tomlPath, "", "", "", brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	// Instructions and brief should still be present
	if !strings.Contains(prompt, "You are a Builder Ant.") {
		t.Error("prompt should contain developer_instructions even with empty capsule")
	}
	if !strings.Contains(prompt, "# Task 2.1") {
		t.Error("prompt should contain task brief even with empty capsule")
	}
}

func TestAssemblePrompt_RespectsOrder(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	// Use distinct markers to verify order
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	capsule := "[SECTION:CAPSULE]"
	skill := "[SECTION:SKILL]"
	pheromone := "[SECTION:PHEROMONE]"
	brief := "[SECTION:BRIEF]"

	prompt, err := AssemblePrompt(tomlPath, capsule, skill, pheromone, brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	// Check strict ordering: instructions < capsule < skill < pheromone < brief
	instrIdx := strings.Index(prompt, "[SECTION:INSTRUCTIONS]")
	capsuleIdx := strings.Index(prompt, "[SECTION:CAPSULE]")
	skillIdx := strings.Index(prompt, "[SECTION:SKILL]")
	pheromoneIdx := strings.Index(prompt, "[SECTION:PHEROMONE]")
	briefIdx := strings.Index(prompt, "[SECTION:BRIEF]")

	if instrIdx < 0 {
		t.Fatal("instructions marker not found in prompt")
	}
	if capsuleIdx < 0 {
		t.Fatal("capsule marker not found in prompt")
	}
	if skillIdx < 0 {
		t.Fatal("skill marker not found in prompt")
	}
	if pheromoneIdx < 0 {
		t.Fatal("pheromone marker not found in prompt")
	}
	if briefIdx < 0 {
		t.Fatal("brief marker not found in prompt")
	}
	if instrIdx >= capsuleIdx {
		t.Errorf("instructions (idx=%d) should come before capsule (idx=%d)", instrIdx, capsuleIdx)
	}
	if capsuleIdx >= skillIdx {
		t.Errorf("capsule (idx=%d) should come before skill (idx=%d)", capsuleIdx, skillIdx)
	}
	if skillIdx >= pheromoneIdx {
		t.Errorf("skill (idx=%d) should come before pheromone (idx=%d)", skillIdx, pheromoneIdx)
	}
	if pheromoneIdx >= briefIdx {
		t.Errorf("pheromone (idx=%d) should come before brief (idx=%d)", pheromoneIdx, briefIdx)
	}
}

func TestAssemblePrompt_EmptySkillAndPheromoneOmitted(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	capsule := "[SECTION:CAPSULE]"
	brief := "[SECTION:BRIEF]"

	prompt, err := AssemblePrompt(tomlPath, capsule, "", "", brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	// Should only have instructions, capsule, brief -- no skill or pheromone sections
	if !strings.Contains(prompt, "[SECTION:INSTRUCTIONS]") {
		t.Error("prompt should contain developer_instructions")
	}
	if !strings.Contains(prompt, "[SECTION:CAPSULE]") {
		t.Error("prompt should contain context capsule")
	}
	if !strings.Contains(prompt, "[SECTION:BRIEF]") {
		t.Error("prompt should contain task brief")
	}

	// Verify ordering: instructions < capsule < brief
	instrIdx := strings.Index(prompt, "[SECTION:INSTRUCTIONS]")
	capsuleIdx := strings.Index(prompt, "[SECTION:CAPSULE]")
	briefIdx := strings.Index(prompt, "[SECTION:BRIEF]")
	if instrIdx >= capsuleIdx {
		t.Error("instructions should come before capsule")
	}
	if capsuleIdx >= briefIdx {
		t.Error("capsule should come before brief")
	}
}

func TestAssemblePrompt_OnlySkillProvided(t *testing.T) {
	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	prompt, err := AssemblePrompt(tomlPath, "", "[SECTION:SKILL]", "", "[SECTION:BRIEF]")
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	if !strings.Contains(prompt, "[SECTION:SKILL]") {
		t.Error("prompt should contain skill section")
	}
	if strings.Contains(prompt, "[SECTION:PHEROMONE]") {
		t.Error("prompt should NOT contain pheromone section when empty")
	}
}

func TestAssemblePrompt_RespectsGlobalBudget(t *testing.T) {
	t.Setenv(promptBudgetEnvVar, "220")

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
` + strings.Repeat("I", 80) + `
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	context := "[SECTION:CONTEXT]\n" + strings.Repeat("C", 70)
	skill := "[SECTION:SKILL]\n" + strings.Repeat("S", 120)
	pheromone := "[SECTION:PHEROMONE]\n" + strings.Repeat("P", 90)
	brief := "[SECTION:BRIEF]\n" + strings.Repeat("B", 60)

	prompt, err := AssemblePrompt(tomlPath, context, skill, pheromone, brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	if len(prompt) > 220 {
		t.Fatalf("prompt length = %d, want <= 220\n%s", len(prompt), prompt)
	}
	if !strings.Contains(prompt, "[SECTION:INSTRUCTIONS]") {
		t.Fatal("prompt should keep developer instructions")
	}
	if !strings.Contains(prompt, "[SECTION:CONTEXT]") {
		t.Fatal("prompt should keep context section ahead of optional trims")
	}
	if !strings.Contains(prompt, "[SECTION:BRIEF]") {
		t.Fatal("prompt should keep task brief")
	}
	if strings.Contains(prompt, "[SECTION:SKILL]") {
		t.Fatalf("skill section should be trimmed before required sections\n%s", prompt)
	}
}

func TestAssemblePrompt_WeightedContextStillBeatsOptionalSections(t *testing.T) {
	t.Setenv(promptBudgetEnvVar, "220")

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
` + strings.Repeat("I", 70) + `
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	context := "[SECTION:CONTEXT]\n[WEIGHTED:0.91]\n" + strings.Repeat("C", 95)
	skill := "[SECTION:SKILL]\n" + strings.Repeat("S", 120)
	brief := "[SECTION:BRIEF]\n" + strings.Repeat("B", 60)

	prompt, err := AssemblePrompt(tomlPath, context, skill, "", brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	if !strings.Contains(prompt, "[SECTION:CONTEXT]") {
		t.Fatal("weighted runtime context should survive ahead of optional sections")
	}
	if strings.Contains(prompt, "[SECTION:SKILL]") {
		t.Fatal("optional skill section should still trim before weighted context")
	}
}

func TestAssemblePrompt_TruncatesRequiredSectionsAsLastResort(t *testing.T) {
	t.Setenv(promptBudgetEnvVar, "120")

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, "agent.toml")
	content := `name = "aether-builder"
description = "Builder agent"

developer_instructions = '''
[SECTION:INSTRUCTIONS]
` + strings.Repeat("I", 180) + `
'''
`
	if err := os.WriteFile(tomlPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test TOML: %v", err)
	}

	brief := "[SECTION:BRIEF]\n" + strings.Repeat("B", 180)
	prompt, err := AssemblePrompt(tomlPath, "", "", "", brief)
	if err != nil {
		t.Fatalf("AssemblePrompt returned error: %v", err)
	}

	if len(prompt) > 120 {
		t.Fatalf("prompt length = %d, want <= 120\n%s", len(prompt), prompt)
	}
	if !strings.Contains(prompt, "[SECTION:INSTRUCTIONS]") {
		t.Fatal("required instructions should survive last-resort trimming")
	}
	if !strings.Contains(prompt, "[SECTION:BRIEF]") {
		t.Fatal("required task brief should survive last-resort trimming")
	}
}

// --- TestRenderTaskBrief ---

func TestRenderTaskBrief_FormatsCorrectly(t *testing.T) {
	data := TaskBriefData{
		TaskID:          "2.1",
		Goal:            "Implement prompt assembly for Codex workers",
		Constraints:     []string{"Must use existing context-capsule logic", "Respect token budget"},
		Hints:           []string{"Check cmd/context.go for ContextCapsuleOutput struct"},
		SuccessCriteria: []string{"All tests pass", "No regressions in existing tests"},
	}

	result := RenderTaskBrief(data)

	// Verify key fields appear
	if !strings.Contains(result, "# Task 2.1") {
		t.Error("brief should contain task ID heading")
	}
	if !strings.Contains(result, "Implement prompt assembly for Codex workers") {
		t.Error("brief should contain goal")
	}
	if !strings.Contains(result, "Must use existing context-capsule logic") {
		t.Error("brief should contain first constraint")
	}
	if !strings.Contains(result, "Check cmd/context.go") {
		t.Error("brief should contain hint")
	}
	if !strings.Contains(result, "All tests pass") {
		t.Error("brief should contain success criterion")
	}
}

func TestRenderTaskBrief_EmptyFields(t *testing.T) {
	data := TaskBriefData{
		TaskID: "1.0",
		Goal:   "Do something",
	}

	result := RenderTaskBrief(data)

	if !strings.Contains(result, "# Task 1.0") {
		t.Error("brief should contain task ID heading")
	}
	if !strings.Contains(result, "Do something") {
		t.Error("brief should contain goal")
	}
	// Should not have section headers for empty slices
	if strings.Contains(result, "## Constraints") {
		t.Error("brief should not show Constraints section when empty")
	}
	if strings.Contains(result, "## Hints") {
		t.Error("brief should not show Hints section when empty")
	}
	if strings.Contains(result, "## Success Criteria") {
		t.Error("brief should not show Success Criteria section when empty")
	}
}
