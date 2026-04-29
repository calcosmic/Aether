package cmd

import (
	"testing"
)

func TestRenderResearchDisplayProducesOutputWithAllSections(t *testing.T) {
	data := ceremonyResearchData{
		TechStackDetail: []techStackDetail{
			{Language: "Go", SourceFile: "go.mod", Deps: []depEntry{{Name: "cobra"}}},
		},
		DirClassification: dirClassification{
			Type:    "standard_app",
			Signals: []string{"has src/", "has cmd/"},
		},
		GovernanceDetails: []governanceDetail{
			{Tool: "golangci-lint", File: ".golangci.yml", Category: "linter"},
		},
		ColonyContextSummary: colonyContextSummary{
			DetectedType:        "go",
			DirType:             "standard_app",
			TechStackCount:      1,
			GovernanceToolCount: 1,
			PheromoneCount:      0,
			IsGitRepo:           true,
			FileCount:           42,
		},
	}

	output := renderResearchDisplay(data)

	if output == "" {
		t.Fatal("renderResearchDisplay returned empty string for non-empty data")
	}

	// Check for section headers
	expected := []string{
		"Tech Stack Detail",
		"Go",
		"go.mod",
		"Directory Classification",
		"standard_app",
		"Governance Details",
		"golangci-lint",
		"Colony Context",
		"true",
		"42",
	}
	for _, want := range expected {
		if !testContains(output, want) {
			t.Errorf("renderResearchDisplay output missing %q", want)
		}
	}
}

func TestRenderResearchDisplayReturnsEmptyWhenAllFieldsNil(t *testing.T) {
	data := ceremonyResearchData{}

	output := renderResearchDisplay(data)

	if output != "" {
		t.Errorf("renderResearchDisplay should return empty string for nil data, got: %q", output)
	}
}

func TestRenderResearchDisplayReturnsEmptyWhenFieldsAreEmpty(t *testing.T) {
	data := ceremonyResearchData{
		TechStackDetail:      []techStackDetail{},
		DirClassification:    dirClassification{},
		GovernanceDetails:    []governanceDetail{},
		ColonyContextSummary: colonyContextSummary{},
	}

	output := renderResearchDisplay(data)

	if output != "" {
		t.Errorf("renderResearchDisplay should return empty string for empty slices, got: %q", output)
	}
}

func testContains(s, substr string) bool {
	return len(s) >= len(substr) && testSearchString(s, substr)
}

func testSearchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
