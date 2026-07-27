package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// RETIRE-01/02/03/04. Phase 160 deleted `control-ts/` — the one component
// confirmed genuinely dead — while explicitly KEEPING `.aether/ts-host/` and
// `.aether/ts/`, which are embedded via the repo-root embedded_assets.go and
// whose removal breaks `go build`, `aether publish`, and `aether integrity`.
//
// The asymmetry is the whole point: "delete the dead TypeScript" is exactly the
// kind of instruction that over-applies. These tests make the boundary
// executable so a future cleanup cannot quietly take the wrong one.

func retiredPackagesRepoRoot(t *testing.T) string {
	t.Helper()
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func TestRetiredPackagesStayRetired(t *testing.T) {
	root := retiredPackagesRepoRoot(t)

	t.Run("ControlTSIsGone", func(t *testing.T) {
		p := filepath.Join(root, "control-ts")
		if _, err := os.Stat(p); err == nil {
			t.Errorf("control-ts/ exists at %s — it was retired in Phase 160 (RETIRE-01); if it is being reintroduced, that needs a written decision, not a directory", p)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat control-ts: %v", err)
		}
	})

	// RETIRE-02 / RETIRE-03. These are NOT dead. Deleting either breaks the
	// build via //go:embed in embedded_assets.go at the repo root (note: repo
	// root, not cmd/ — an earlier canonical reference had this path wrong).
	t.Run("TSHostAndTSAreKept", func(t *testing.T) {
		for _, rel := range []string{
			filepath.Join(".aether", "ts-host"),
			filepath.Join(".aether", "ts"),
		} {
			p := filepath.Join(root, rel)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("%s is missing (%v) — it is embedded via //go:embed in embedded_assets.go; removing it breaks `go build`, `aether publish`, and `aether integrity` (RETIRE-02/03)", rel, err)
			}
		}
	})

	// The embed directives are the reason ts-host/ts must stay. Assert they
	// still reference them, so "nothing references it" can never be concluded
	// from a grep that missed the embed line.
	t.Run("EmbedDirectivesStillReferenceKeptTrees", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join(root, "embedded_assets.go"))
		if err != nil {
			t.Fatalf("read embedded_assets.go at repo root: %v", err)
		}
		src := string(raw)
		for _, needle := range []string{".aether/ts-host/dist", ".aether/ts/dist/narrator.js"} {
			if !strings.Contains(src, needle) {
				t.Errorf("embedded_assets.go no longer embeds %q — if this was intentional, RETIRE-02/03's justification for keeping these trees no longer holds and needs revisiting", needle)
			}
		}
	})
}

// RETIRE-04: no test leaves this repository without a recorded disposition.
// This is what makes the ledger a contract rather than a document — a
// `recovered-by:` pointing at a test that does not exist is a silent coverage
// hole wearing the label of a recovery.
func TestRetiredTestsLedgerDispositionsAreHonest(t *testing.T) {
	root := retiredPackagesRepoRoot(t)
	ledgerPath := filepath.Join(root, ".aether", "docs", "retired-tests-ledger.md")

	raw, err := os.ReadFile(ledgerPath)
	if err != nil {
		t.Fatalf("read retired-tests ledger: %v — RETIRE-04 requires it to exist", err)
	}
	ledger := string(raw)

	entries := regexp.MustCompile(`(?m)^### ` + "`" + `([^` + "`" + `]+)` + "`").FindAllStringSubmatch(ledger, -1)
	if len(entries) == 0 {
		t.Fatal("retired-tests ledger contains no entries; Phase 160 retired at least control-ts/tests/schemas/policy.schema.test.ts")
	}

	dispositions := regexp.MustCompile(`\*\*Disposition:\*\*\s*` + "`?" + `(dead-with-no-replacement|recovered-by:([^` + "`" + `\s,]+))`).FindAllStringSubmatch(ledger, -1)
	if len(dispositions) != len(entries) {
		t.Fatalf("ledger has %d entries but %d valid dispositions — every retired test needs exactly one disposition of `dead-with-no-replacement` or `recovered-by:<path>` (RETIRE-04)", len(entries), len(dispositions))
	}

	for _, d := range dispositions {
		if d[2] == "" {
			continue // dead-with-no-replacement: knowingly accepted, nothing to resolve
		}
		replacement := filepath.Join(root, filepath.FromSlash(d[2]))
		if _, err := os.Stat(replacement); err != nil {
			t.Errorf("ledger claims coverage was recovered by %q, but that file does not exist (%v) — a recovered-by pointing nowhere is a coverage hole labelled as a recovery", d[2], err)
		}
	}

	// The retired TS schema test is the specific constraint CONTEXT.md called
	// out: its replacement had to exist before or alongside the deletion.
	if !strings.Contains(ledger, "control-ts/tests/schemas/policy.schema.test.ts") {
		t.Error("ledger does not record control-ts/tests/schemas/policy.schema.test.ts — it was the only validation of colony/policies/*.yaml before Phase 160 deleted it")
	}
}
