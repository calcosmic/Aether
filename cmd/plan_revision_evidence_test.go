package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --revision-evidence used to store only the path string; the Route-Setter
// never saw the findings unless a human pasted them. The appendix now carries
// bounded file contents.
func TestRevisionAppendixCarriesEvidenceContents(t *testing.T) {
	root := t.TempDir()
	sentinel := "ORACLE-FINDING: the exporter must use the streaming API, not batch"
	if err := os.WriteFile(filepath.Join(root, "oracle-report.md"), []byte("# Findings\n\n"+sentinel+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	appendix := renderPlanRevisionWorkerAppendix(root, &codexPlanRevisionContext{
		ReasonType: colony.PlanRevisionResearch,
		Reason:     "oracle research completed",
		Evidence:   []string{"oracle-report.md"},
	})
	if !strings.Contains(appendix, sentinel) {
		t.Fatalf("appendix missing evidence contents:\n%s", appendix)
	}
	if !strings.Contains(appendix, "### Evidence: oracle-report.md") {
		t.Fatalf("appendix missing evidence heading:\n%s", appendix)
	}
}

// Evidence excerpts are bounded — a huge report must not flood the brief.
func TestRevisionEvidenceExcerptsAreBounded(t *testing.T) {
	root := t.TempDir()
	huge := strings.Repeat("Finding paragraph with detail.\n\n", 1000)
	for _, name := range []string{"a.md", "b.md", "c.md", "d.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(huge), 0644); err != nil {
			t.Fatal(err)
		}
	}
	excerpts := renderRevisionEvidenceExcerpts(root, []string{"a.md", "b.md", "c.md", "d.md"})
	if len(excerpts) > revisionEvidenceExcerptBudget+1000 {
		t.Fatalf("excerpts = %d chars, budget %d", len(excerpts), revisionEvidenceExcerptBudget)
	}
	if !strings.Contains(excerpts, "truncated — full evidence") && !strings.Contains(excerpts, "excerpt budget exhausted") {
		t.Error("bounded excerpts should say what was cut")
	}
}
