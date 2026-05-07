---
phase: 101-platform-parity-verification
reviewed: 2026-05-07T21:17:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - cmd/parity_test.go
  - cmd/testdata/parity_snapshot.json
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 101: Code Review Report

**Reviewed:** 2026-05-07T21:17:00Z
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed `cmd/parity_test.go` (317 lines of Go test code) and its golden file `cmd/testdata/parity_snapshot.json`. The test suite validates parity across five surfaces (YAML catalog, Claude wrappers, OpenCode wrappers, command guide, Cobra runtime) using four test functions: golden snapshot comparison, phantom detection, wrapper/guide coverage, and alias resolution completeness.

The test logic is sound in its core intent but has a data quality issue in the golden file (duplicate runtime entries) and a coverage gap (no reverse-direction parity check). Three warnings and two informational findings.

## Warnings

### WR-01: Duplicate runtime command names baked into golden snapshot

**File:** `cmd/testdata/parity_snapshot.json:289-290` (and 8 other pairs)
**Issue:** The `runtime_catalog_names` array contains duplicate entries for commands that are registered both as root-level commands and as subcommands of parent commands. Verified duplicates include: `closeout` (lines 289-290), `council-advocate` (lines 309-310), `council-budget-check` (lines 311-312), `council-challenger` (lines 313-314), `council-deliberate` (lines 315-316), `council-history` (lines 317-318), `council-sage` (lines 319-320), `get` (lines 369-371, triple), `pheromones` (lines 452-453), `registry` (lines 484-485), `set` (lines 509-511, triple), `wisdom` (lines 617-618).

Root cause: `extractRuntimeNames()` in `parity_test.go:92-100` flattens all entries from `buildAuditCatalog()` using `entry.Name` only, ignoring `ParentCommand`. Commands registered under multiple parents (e.g., `closeout` is both a root command and a subcommand of `ceremony`) produce duplicate names.

This does not cause false test results currently because `TestNoPhantomCommands` uses `map[string]bool` lookups (duplicates are harmless in a set). However, the golden snapshot is misleading as a reference document -- it claims there are more unique runtime commands than actually exist. If anyone uses this file for documentation or downstream tooling, the duplicates will cause confusion.
**Fix:**
```go
// In extractRuntimeNames, deduplicate names before returning:
func extractRuntimeNames() []string {
	catalog := buildAuditCatalog(rootCmd)
	seen := make(map[string]bool, len(catalog))
	names := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		if !seen[entry.Name] {
			seen[entry.Name] = true
			names = append(names, entry.Name)
		}
	}
	sort.Strings(names)
	return names
}
```

### WR-02: No reverse parity check -- orphan runtime commands are invisible

**File:** `cmd/parity_test.go:175-241` (TestNoPhantomCommands)
**Issue:** `TestNoPhantomCommands` checks only one direction: that every YAML/wrapper name resolves to a runtime command. It does not check the reverse direction: that every user-facing runtime command has a corresponding YAML source and wrapper. If a runtime command is added to Cobra but no YAML file, wrapper, or guide entry is created, the existing tests will pass despite the command being unreachable through the documented user surfaces.

The same gap applies to `TestAllYamlHaveWrappersAndGuide` (lines 243-297), which checks YAML -> wrappers and wrappers -> YAML but never checks that the runtime has no commands without YAML sources.

This means a developer could add a Cobra subcommand, forget to create the YAML/wrapper/guide, and all four parity tests would still pass.
**Fix:** Add a `TestNoOrphanRuntimeCommands` test that iterates over runtime command names and verifies each has a corresponding YAML source (after reverse-resolving from Cobra name to YAML name). Exclude internal-only commands that are intentionally not surfaced through wrappers.

### WR-03: `promptOnlyCommands` and `cobraBuiltinCommands` are fragile manual lists with no self-validation

**File:** `cmd/parity_test.go:42-53`
**Issue:** Both `promptOnlyCommands` and `cobraBuiltinCommands` are manually maintained maps that determine which YAML names are excluded from phantom checks. If a new prompt-only command is added to the YAML catalog but someone forgets to add it to `promptOnlyCommands`, the `TestNoPhantomCommands` test will fail with a false positive -- flagging a legitimate prompt-only command as "not in runtime."

Conversely, if a command is removed from the YAML catalog but its entry remains in `promptOnlyCommands`, the stale entry silently goes undetected.

There is no test that validates these maps are themselves current.
**Fix:** Add a test that verifies every entry in `promptOnlyCommands` actually exists in the YAML catalog. This ensures stale entries are caught:
```go
func TestPromptOnlyCommandsAreCurrent(t *testing.T) {
    yamlNames := extractYAMLNames(t)
    yamlSet := make(map[string]bool, len(yamlNames))
    for _, name := range yamlNames {
        yamlSet[name] = true
    }
    for name := range promptOnlyCommands {
        if !yamlSet[name] {
            t.Errorf("promptOnlyCommands entry %q not found in YAML catalog (stale entry?)", name)
        }
    }
}
```

## Info

### IN-01: `yamlToRuntimeName` alias map is not validated for completeness

**File:** `cmd/parity_test.go:26-39`
**Issue:** `TestAliasResolutionCompleteness` (lines 299-317) verifies that every entry in `yamlToRuntimeName` resolves to an actual runtime command. This is good. However, it does not check the reverse: that every YAML command whose name differs from its Cobra `Use` field has an entry in the map. If a new YAML command is added with a name that differs from its Cobra command name and the mapping is forgotten, the phantom test would flag it as missing from the runtime. The test currently relies on the golden snapshot to catch this indirectly (the golden would change), but there is no targeted assertion.
**Fix:** Consider adding a check that every YAML name that does NOT match a runtime name directly either has an entry in `yamlToRuntimeName` or is in `promptOnlyCommands`/`cobraBuiltinCommands`. This is essentially what `TestNoPhantomCommands` does, so this is low priority.

### IN-02: Golden test provides byte-count diff but no content diff

**File:** `cmd/parity_test.go:168-172`
**Issue:** When the golden snapshot mismatches, the test logs only byte counts (`got N bytes`, `want N bytes`). For a file with 600+ lines of JSON, this makes diagnosing the root cause slow. The error message does mention `-update-golden` which is helpful, but a content-level diff would speed up debugging.
**Fix:** Consider using `github.com/google/go-cmp` or computing a line-level diff between got and want to show which sections changed:
```go
if got != want {
    t.Errorf("parity snapshot golden mismatch; run with -update-golden to refresh")
    // Show first few differing lines for quick diagnosis
    gotLines := strings.Split(got, "\n")
    wantLines := strings.Split(want, "\n")
    for i := 0; i < len(gotLines) && i < len(wantLines); i++ {
        if gotLines[i] != wantLines[i] {
            t.Logf("first diff at line %d:\n  got:  %s\n  want: %s", i+1, gotLines[i], wantLines[i])
            break
        }
    }
}
```

---

_Reviewed: 2026-05-07T21:17:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
