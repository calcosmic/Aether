package cmd

import (
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// synthesizeLaunchBrief produces a structured markdown launch brief from
// the colony goal, charter, and research data. Each section shows content
// from the charter and research data where available; sections with no data
// show "To be determined" rather than being empty.
func synthesizeLaunchBrief(goal string, charter *colony.Charter, researchData ceremonyResearchData) string {
	tbd := "To be determined"

	// Extract tech stack lines from research data
	var techLines []string
	for _, ts := range researchData.TechStackDetail {
		if ts.Language != "" {
			techLines = append(techLines, "- Language: "+ts.Language)
		}
		for _, dep := range ts.Deps {
			techLines = append(techLines, "- "+dep.Name)
		}
	}
	// Include charter tech stack if not already covered
	if charter.TechStack != "" {
		techLines = append([]string{emptyFallback(charter.TechStack, tbd)}, techLines...)
	}

	// Extract dependencies from research data
	var depLines []string
	for _, ts := range researchData.TechStackDetail {
		for _, dep := range ts.Deps {
			depLines = append(depLines, "- "+dep.Name+" ("+ts.Language+")")
		}
		for _, dep := range ts.DevDeps {
			depLines = append(depLines, "- "+dep.Name+" (dev)")
		}
	}

	// Scope from charter
	scope := emptyFallback(charter.Goals, tbd)
	// Add vision context if available
	if charter.Vision != "" {
		scope = emptyFallback(charter.Vision, tbd) + "\n\n" + scope
	}

	// Risks from charter
	risks := emptyFallback(charter.KeyRisks, tbd)
	if charter.Constraints != "" {
		risks += "\n- " + charter.Constraints
	}

	// Success criteria from charter goals
	successCriteria := emptyFallback(charter.Goals, tbd)

	// Build sections
	var b strings.Builder
	b.WriteString("# Colony Launch Brief\n\n")

	b.WriteString("## Goal\n")
	b.WriteString(emptyFallback(goal, tbd))
	b.WriteString("\n\n")

	b.WriteString("## Scope\n")
	b.WriteString(scope)
	b.WriteString("\n\n")

	b.WriteString("## Risks\n")
	b.WriteString(risks)
	b.WriteString("\n\n")

	b.WriteString("## Tech Stack\n")
	if len(techLines) > 0 {
		for _, line := range techLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(tbd)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Dependencies\n")
	if len(depLines) > 0 {
		for _, line := range depLines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	} else {
		b.WriteString(tbd)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString("## Success Criteria\n")
	b.WriteString(successCriteria)
	b.WriteString("\n")

	return b.String()
}

// ceremonyResearchData holds the four research data fields extracted from the
// init-research JSON envelope for display in the init ceremony.
type ceremonyResearchData struct {
	TechStackDetail      []techStackDetail
	DirClassification    dirClassification
	GovernanceDetails    []governanceDetail
	ColonyContextSummary colonyContextSummary
}
