package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDocsDoNotClaimConsolidationRunsToday exists because the claim it guards
// against survived four milestones (v1.10, v1.11, v1.13, v1.23 all declared the
// learning pipeline "restored" while consolidation-phase-end and
// consolidation-seal had zero callers). CLAUDE.md's Definition of Done corollary
// states a documentation claim about runtime behaviour must be testable or
// removed — this test is that enforcement for the specific claim that
// consolidation runs automatically at phase-end and/or seal.
//
// It asserts three things:
//  1. None of CLAUDE.md, AGENTS.md, or .aether/docs/structural-learning-stack.md
//     contain the exact stale phrases that previously asserted consolidation runs
//     today.
//  2. None of those files contain a paraphrase of the same claim, detected via a
//     verb-at-phase regexp combined with a consolidation-identifier check on the
//     same line — this catches a reworded reintroduction that an exact-string
//     list would miss.
//  3. If consolidation-phase-end or consolidation-seal ever gains a real caller
//     in non-test Go code outside cmd/graph_consolidation_cmds.go, this test
//     fails and tells the developer to update the three documents together —
//     because at that point the "no caller" claim becomes false in the other
//     direction and the docs need to catch up (this is expected to happen
//     deliberately in Phase 162).
func TestDocsDoNotClaimConsolidationRunsToday(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeMDPath := filepath.Join(repoRoot, "CLAUDE.md")
	agentsMDPath := filepath.Join(repoRoot, "AGENTS.md")
	stackDocPath := filepath.Join(repoRoot, ".aether", "docs", "structural-learning-stack.md")

	type forbiddenClaim struct {
		file   string
		phrase string
		why    string
	}

	claims := []forbiddenClaim{
		{
			file:   claudeMDPath,
			phrase: "Lifecycle integration: phase-end at /ant-continue, full at /ant-seal",
			why:    "claims /ant-continue and /ant-seal already invoke consolidation; neither does",
		},
		{
			file:   agentsMDPath,
			phrase: "Lifecycle integration: phase-end at `aether continue`, full at `aether seal`",
			why:    "AGENTS.md's phrasing of the same false claim",
		},
		{
			file:   stackDocPath,
			phrase: "The stack runs automatically at phase-end and seal — workers never call it directly.",
			why:    "overview sentence asserting automatic invocation that never happens",
		},
		{
			file:   stackDocPath,
			phrase: "Runs at the end of every phase (`/ant-continue`)",
			why:    "phase-end mode description asserting it runs when nothing calls it",
		},
		{
			file:   stackDocPath,
			phrase: "Runs once during `/ant-seal`",
			why:    "seal mode description asserting it runs when nothing calls it",
		},
		{
			file:   stackDocPath,
			phrase: "| `/ant-continue` | `consolidation-phase-end` |",
			why:    "Integration Points table row wiring /ant-continue to consolidation-phase-end as if live",
		},
		{
			file:   stackDocPath,
			phrase: "| `/ant-seal` | `consolidation-seal` |",
			why:    "Integration Points table row wiring /ant-seal to consolidation-seal as if live",
		},
	}

	fileContents := map[string]string{}
	for _, path := range []string{claudeMDPath, agentsMDPath, stackDocPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		fileContents[path] = string(data)
	}

	// Assertion 1: exact stale phrases must be absent.
	for _, claim := range claims {
		content := fileContents[claim.file]
		if idx := strings.Index(content, claim.phrase); idx != -1 {
			line := 1 + strings.Count(content[:idx], "\n")
			t.Errorf(
				"%s:%d contains a stale consolidation-runs claim: %q (%s)",
				filepath.Base(claim.file), line, claim.phrase, claim.why,
			)
		}
	}

	// Guard: CLAUDE.md's honest lines (37, 44) must survive this test unmodified
	// and must never be treated as forbidden — they are the historical record
	// this correction answers to, not a claim to remove.
	honestPhrase := "`consolidation-phase-end` and `consolidation-seal` still have no caller"
	if !strings.Contains(fileContents[claudeMDPath], honestPhrase) {
		t.Errorf("CLAUDE.md no longer contains the honest 'no caller' sentence (expected substring: %q) — this line must not be touched by corrections to the stale claim", honestPhrase)
	}

	// Assertion 2: paraphrase invariant. A line that both (a) mentions
	// consolidation and (b) uses a runs/executes/invoked ... at ... phase
	// construction is a reworded version of the same false claim, regardless of
	// exact wording.
	verbAtPhaseRe := regexp.MustCompile(`(?i)\b(runs?|executes?|invoked)\b.*?\bat\b.*?\b(the end of )?(every )?phase\b`)
	consolidationRe := regexp.MustCompile(`(?i)consolidation`)

	for _, path := range []string{claudeMDPath, agentsMDPath, stackDocPath} {
		content := fileContents[path]
		for i, line := range strings.Split(content, "\n") {
			if !consolidationRe.MatchString(line) {
				continue
			}
			if verbAtPhaseRe.MatchString(line) {
				t.Errorf(
					"%s:%d reads as a paraphrase of the consolidation-runs claim (mentions consolidation and matches a runs/executes/invoked...at...phase pattern): %q",
					filepath.Base(path), i+1, strings.TrimSpace(line),
				)
			}
		}
	}

	// Assertion 3: trip-wire for the opposite direction. If a real caller for
	// either subcommand appears in non-test Go code outside
	// cmd/graph_consolidation_cmds.go, the "no caller" documentation claim
	// becomes false and needs a coordinated doc update (expected in Phase 162).
	subcommandNames := []string{"consolidation-phase-end", "consolidation-seal"}
	excludedFile := filepath.Join(repoRoot, "cmd", "graph_consolidation_cmds.go")

	var callerSites []string
	for _, dir := range []string{filepath.Join(repoRoot, "cmd"), filepath.Join(repoRoot, "pkg")} {
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			if path == excludedFile {
				return nil
			}

			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}

			for i, line := range strings.Split(string(data), "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "//") {
					continue // comment-only line, does not count as a caller
				}
				for _, name := range subcommandNames {
					if strings.Contains(line, name) {
						callerSites = append(callerSites, fmt.Sprintf("%s:%d: %s", path, i+1, trimmed))
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(callerSites) > 0 {
		t.Errorf(
			"consolidation-phase-end or consolidation-seal now has a real caller outside cmd/graph_consolidation_cmds.go — update CLAUDE.md, AGENTS.md, and .aether/docs/structural-learning-stack.md to reflect that consolidation is now wired (this is expected to happen deliberately in Phase 162). Sites:\n%s",
			strings.Join(callerSites, "\n"),
		)
	}
}
