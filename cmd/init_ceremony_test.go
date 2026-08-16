package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestSynthesizeLaunchBrief verifies that synthesizeLaunchBrief produces
// markdown with all 6 required sections: Goal, Scope, Risks, Tech Stack, Dependencies, Success Criteria.
func TestSynthesizeLaunchBrief(t *testing.T) {
	goal := "Build a REST API"
	charter := &colony.Charter{
		Intent:      "Build a REST API",
		Vision:      "A clean API server",
		Governance:  "golangci-lint",
		Goals:       "Ship v1.0 with auth endpoints",
		TechStack:   "Go, PostgreSQL",
		KeyRisks:    "No tests yet",
		Constraints: "Follow lint rules",
	}
	researchData := ceremonyResearchData{
		TechStackDetail: []techStackDetail{
			{Language: "go", SourceFile: "go.mod", Deps: []depEntry{{Name: "cobra"}}},
		},
		DirClassification: dirClassification{Type: "standard"},
		ColonyContextSummary: colonyContextSummary{
			DetectedType: "go-project",
			Languages:    []string{"go"},
		},
	}

	brief := synthesizeLaunchBrief(goal, charter, researchData)

	requiredSections := []string{
		"# Colony Launch Brief",
		"## Goal",
		"## Scope",
		"## Risks",
		"## Tech Stack",
		"## Dependencies",
		"## Success Criteria",
	}
	for _, section := range requiredSections {
		if !strings.Contains(brief, section) {
			t.Errorf("synthesizeLaunchBrief missing section %q", section)
		}
	}

	// Goal section should contain the colony goal
	if !strings.Contains(brief, "Build a REST API") {
		t.Error("synthesizeLaunchBrief missing goal content")
	}

	// Tech Stack section should include detected tech stack
	if !strings.Contains(brief, "Go") {
		t.Error("synthesizeLaunchBrief missing detected language Go")
	}
	if !strings.Contains(brief, "cobra") {
		t.Error("synthesizeLaunchBrief missing detected dependency cobra")
	}
}

// TestSynthesizeLaunchBriefIncludesTechStackFromCharter verifies tech stack
// detected from research data appears in the brief.
func TestSynthesizeLaunchBriefIncludesTechStackFromCharter(t *testing.T) {
	charter := &colony.Charter{
		Intent:    "Build something",
		TechStack: "Go, Cobra, PostgreSQL",
	}
	researchData := ceremonyResearchData{
		TechStackDetail: []techStackDetail{
			{Language: "go", SourceFile: "go.mod", Deps: []depEntry{{Name: "github.com/lib/pq"}}},
		},
	}

	brief := synthesizeLaunchBrief("Build something", charter, researchData)

	if !strings.Contains(brief, "PostgreSQL") {
		t.Error("synthesizeLaunchBrief should include PostgreSQL from charter tech stack")
	}
	if !strings.Contains(brief, "github.com/lib/pq") {
		t.Error("synthesizeLaunchBrief should include dependency from research data")
	}
}

// TestSynthesizeLaunchBriefEmptyData verifies that sections with no data
// show "To be determined" rather than being empty.
func TestSynthesizeLaunchBriefEmptyData(t *testing.T) {
	charter := &colony.Charter{
		Intent: "Some goal",
	}
	researchData := ceremonyResearchData{}

	brief := synthesizeLaunchBrief("Some goal", charter, researchData)

	// Empty sections should show TBD
	if !strings.Contains(brief, "To be determined") {
		t.Error("synthesizeLaunchBrief should show 'To be determined' for empty sections")
	}
}

// TestSynthesizeLaunchBriefWithRisks verifies that KeyRisks from the charter
// appear in the Risks section.
func TestSynthesizeLaunchBriefWithRisks(t *testing.T) {
	charter := &colony.Charter{
		Intent:      "Risky project",
		KeyRisks:    "No test coverage, tight deadline",
		Constraints: "Must ship in 2 weeks",
	}
	researchData := ceremonyResearchData{}

	brief := synthesizeLaunchBrief("Risky project", charter, researchData)

	if !strings.Contains(brief, "No test coverage") {
		t.Error("synthesizeLaunchBrief missing KeyRisks content in Risks section")
	}
	if !strings.Contains(brief, "tight deadline") {
		t.Error("synthesizeLaunchBrief missing KeyRisks content in Risks section")
	}
}

// TestRenderCharterDisplay verifies that renderCharterDisplay produces output
// containing all 7 charter section names.
func TestRenderCharterDisplay(t *testing.T) {
	ch := colony.Charter{
		Intent:      "Build great software",
		Vision:      "A world-class go project",
		Governance:  "Linting: golangci-lint. CI: GitHub Actions",
		Goals:       "Goal: Build great software. Focus on quality.",
		TechStack:   "Languages: go. Frameworks/Tools: cobra",
		KeyRisks:    "No test framework detected -- regression risk",
		Constraints: "Follow golangci-lint rules",
	}

	output := renderCharterDisplay(ch)

	labels := []string{"Intent:", "Vision:", "Governance:", "Goals:", "Tech Stack:", "Key Risks:", "Constraints:"}
	for _, label := range labels {
		if !strings.Contains(output, label) {
			t.Errorf("renderCharterDisplay output missing label %q", label)
		}
	}

	// Verify content appears
	if !strings.Contains(output, "Build great software") {
		t.Error("renderCharterDisplay output missing Intent content")
	}
	if !strings.Contains(output, "A world-class go project") {
		t.Error("renderCharterDisplay output missing Vision content")
	}
}
