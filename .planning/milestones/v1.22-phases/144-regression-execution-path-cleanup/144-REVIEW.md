---
phase: 144-regression-execution-path-cleanup
reviewed: 2026-05-18T23:06:17Z
depth: standard
files_reviewed: 6
files_reviewed_list:
  - .aether/commands/build.yaml
  - .aether/commands/plan.yaml
  - .aether/docs/command-playbooks/plan-prep.md
  - cmd/cli_flag_audit_test.go
  - cmd/codex_colonize_test.go
  - cmd/command_guide_test.go
findings:
  critical: 0
  warning: 2
  info: 3
  total: 5
status: issues_found
---

# Phase 144: Code Review Report

**Reviewed:** 2026-05-18T23:06:17Z
**Depth:** standard
**Files Reviewed:** 6
**Status:** issues_found

## Summary

Reviewed 6 files: 2 YAML command definitions, 1 markdown playbook, and 3 Go test files. The code is well-structured with thorough test coverage. The YAML command specs and playbooks consistently enforce the TypeScript host spine architecture and wrapper-runtime contract. The Go tests are comprehensive, covering colonize dispatch, finalization, path validation, manifest freshness, symlink rejection, and execution path auditing.

Two warnings found: (1) the CLI flag audit regex cannot verify flags on sub-subcommands like `aether host plan --depth`, producing a silent coverage gap, and (2) the `plan-prep.md` playbook instructs wrappers to save manifests to "a temporary manifest file outside `.aether/data/`" but never specifies cleanup of that file, risking stale temp artifacts. Three informational items cover duplication of a helper function and a comment referencing a wrong task.

No critical issues (security vulnerabilities, data loss, or crashes) were found.

## Critical Issues

None.

## Warnings

### WR-01: CLI flag audit regex silently misses flags on host sub-subcommands

**File:** `cmd/cli_flag_audit_test.go:32`
**Issue:** The regex `aether\s+([\w][\w-]*)\s+...` captures only the first word after `aether` as the subcommand. For calls like `aether host plan --depth <choice>`, it captures "host" as the subcommand and sees no flags (since "plan" starts with `p`, not `--`). This means `--depth`, `--planning-depth`, `--target`, and `--max-iterations` on `aether host plan` are never validated against the Go runtime. The host subcommands have `DisableFlagParsing: true` (by design, since flags are forwarded raw to the TypeScript host), so even if the regex captured "plan" as the subcommand, the flags wouldn't be registered. The audit therefore provides a false sense of coverage for the entire `host` subcommand tree.

This is an existing design limitation, not a regression. The host subcommand tree intentionally bypasses Cobra flag parsing. However, the test comment claims it "systematically compares markdown CLI calls against Go registrations," which overstates its coverage for the host path.

**Fix:** Consider either (a) updating the test comment to document the known coverage gap for `aether host <subcommand>` calls, or (b) adding a second regex pass that specifically handles the `aether host <subcommand> --flag` pattern by extracting the second token as the subcommand.

### WR-02: plan-prep.md instructs saving manifest to temp file without cleanup

**File:** `.aether/docs/command-playbooks/plan-prep.md:72-73`
**Issue:** Step 5 instructs: "Save the full JSON envelope to a temporary manifest file outside `.aether/data/`" but provides no guidance on when or how to clean up this file. If planning is interrupted (user cancels, network error), the temp manifest file remains. On subsequent runs, stale manifests could persist in temp directories. The `build.yaml` and `continue.yaml` wrappers reference similar patterns with explicit temp file contracts (`${TMPDIR:-/tmp}/aether-<workflow>-<run>/<workflow>-completion.json`), but `plan-prep.md` does not adopt this convention.

**Fix:** Align the plan-prep.md temp file guidance with the established contract pattern. For example:
```
Save the full JSON envelope to ${TMPDIR:-/tmp}/aether-plan-<run>/plan-completion.json.
Discard the temp directory after plan-finalize completes.
```

## Info

### IN-01: `stringSliceContains` helper is duplicated across test files

**File:** `cmd/command_guide_test.go:776-783` and also referenced in `cmd/codex_colonize_test.go:451,518`
**Issue:** The `stringSliceContains` function is defined in `command_guide_test.go` and also exists (or is imported) in `codex_colonize_test.go` via the `cmd` package. Since both files are in the same package, the function defined in `command_guide_test.go` is accessible from `codex_colonize_test.go`. However, there is also a `containsString` helper defined in `codex_visuals.go` (production code) that serves the same purpose. Having the same utility logic in multiple places increases maintenance burden.

**Fix:** Consolidate to a single `stringSliceContains` test helper or use the production `containsString` from `codex_visuals.go` consistently.

### IN-02: plan-prep.md lacks step numbering for manifest result field name ambiguity

**File:** `.aether/docs/command-playbooks/plan-prep.md:75`
**Issue:** Line 75 says "Parse `result.plan_manifest` or `result.planning_manifest`" -- the disjunction suggests uncertainty about which field name the runtime returns. If the runtime contract defines a canonical field name, the playbook should use only that name. If both are valid depending on version, the playbook should explain when each applies.

**Fix:** Confirm the canonical field name with the runtime implementation and use only that name, or add a note explaining the dual-field support.

### IN-03: Test comment references wrong task/plan number

**File:** `cmd/cli_flag_audit_test.go:156-157`
**Issue:** The comment says "this plan (71-02) is responsible for registering" but the test file is being reviewed as part of phase 144. The hardcoded reference to plan 71-02 is stale. This is a documentation accuracy issue in the test file -- the test itself is correct, but the comment is misleading for future maintainers.

**Fix:** Update the comment to remove the specific plan reference or update it to reflect the current phase:
```go
// TestCLIFlagAuditSubcommandsRegistered verifies specific subcommands
// that must be registered in the Go runtime for markdown wrapper calls.
```

---

_Reviewed: 2026-05-18T23:06:17Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
