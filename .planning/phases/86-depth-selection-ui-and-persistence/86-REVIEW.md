---
phase: 86-depth-selection-ui-and-persistence
reviewed: 2026-05-01T00:00:00Z
depth: standard
files_reviewed: 9
files_reviewed_list:
  - cmd/codex_build.go
  - cmd/codex_build_finalize.go
  - cmd/codex_plan.go
  - cmd/codex_visuals.go
  - cmd/codex_visuals_test.go
  - cmd/codex_workflow_cmds.go
  - cmd/review_depth.go
  - cmd/review_depth_test.go
  - pkg/colony/colony.go
findings:
  critical: 1
  warning: 3
  info: 4
  total: 8
status: issues_found
---

# Phase 86: Code Review Report

**Reviewed:** 2026-05-01T00:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Reviewed 9 files implementing depth selection UI and persistence for the Aether colony system. The core depth resolution logic (`review_depth.go`) is well-structured with thorough test coverage (`review_depth_test.go`). The visual rendering (`codex_visuals.go`) correctly surfaces depth information to users. The build pipeline (`codex_build.go`, `codex_build_finalize.go`) properly threads depth through the manifest and dispatch pipeline.

One critical bug was found where a keyword-matched phase with `--light` still receives heavy review via `resolveVerificationDepth`, contradicting the documented priority order. Three warnings cover keyword false-positive risks in `phaseRiskLevel`, unbounded recursion risk in `findRepoRelativePath`, and a potential double-write pattern in `codex_build_finalize.go`.

## Critical Issues

### CR-01: `resolveVerificationDepth` ignores `lightFlag` when keyword is matched, contradicting priority contract

**File:** `cmd/review_depth.go:62-87`
**Issue:** The documented priority in the comment on line 63 says: `Priority: final phase -> heavyFlag -> heavy keyword match -> lightFlag -> explicit --verification-depth string -> default standard`. However, the actual code checks keyword match at line 76 **before** `lightFlag` at line 78. This means if a phase name contains a keyword like "auth" and the user explicitly passes `--light`, they still get heavy review -- the light flag is silently ignored.

This is inconsistent with `resolveReviewDepth` (lines 27-42) which checks keyword match before the default but has no explicit light-flag path (and uses the older 2-level depth model). The 3-level `resolveVerificationDepth` introduced a `lightFlag` priority slot that is documented but unreachable when a keyword match triggers first.

**Fix:**
```go
func resolveVerificationDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool, verificationDepthStr string) colony.VerificationDepth {
	// Final phase is always heavy regardless of flags.
	if phase.ID == totalPhases {
		return colony.VerificationDepthHeavy
	}
	// Explicit heavy flag overrides everything else.
	if heavyFlag {
		return colony.VerificationDepthHeavy
	}
	// Keyword auto-detection triggers heavy review, BUT explicit light flag
	// should override keyword match (user intent takes priority).
	if phaseHasHeavyKeywords(phase.Name) && !lightFlag {
		return colony.VerificationDepthHeavy
	}
	// Explicit light flag.
	if lightFlag {
		return colony.VerificationDepthLight
	}
	// Explicit --verification-depth string (normalized).
	if verificationDepthStr != "" {
		return colony.NormalizeVerificationDepth(verificationDepthStr)
	}
	// Smart default based on phase position + code change risk.
	return resolveSmartVerificationDepth(phase, totalPhases)
}
```

## Warnings

### WR-01: Keyword false-positive risk in `phaseRiskLevel` due to substring matching on phase text

**File:** `cmd/review_depth.go:164-173`
**Issue:** `phaseRiskLevel` calls `collectPhaseText` which concatenates the phase name, description, success criteria, and all task goals/constraints/hints into a single lowercased string, then checks if any keyword is a **substring**. The `securityRiskKeywords` list includes short, common words like `"session"` (line 109) and `"token"` (line 108). A phase with description text like "restore session state" or "verify token handling" would be classified as "high" risk even if the phase has nothing to do with security. Similarly, `"password"` in `securityRiskKeywords` would match "account password policy documentation" in a phase about writing docs.

This cascades into `resolveSmartVerificationDepth` and `resolveSmartPlanningDepth`, causing false heavy/deep depth assignments.

**Fix:** Consider either: (a) requiring whole-word matching using word boundaries, or (b) prefixing keywords with spaces and checking for `" "+kw+" "` patterns, or (c) narrowing keyword matching to the phase name only (not description/task text). Option (c) is the safest short-term fix:

```go
func phaseRiskLevel(phase colony.Phase) string {
	// Check keywords against phase name only, not full phase text,
	// to avoid false positives from task descriptions and hints.
	nameLower := strings.ToLower(phase.Name)
	if matchesAnyKeyword(nameLower, securityRiskKeywords) {
		return "high"
	}
	if matchesAnyKeyword(nameLower, blastRadiusKeywords) {
		return "medium"
	}
	return "low"
}
```

### WR-02: `findRepoRelativePath` fallback uses unbounded recursive walk on the entire repo

**File:** `cmd/codex_build_finalize.go:526-546`
**Issue:** When `git ls-files` fails or returns no results, `findRepoRelativePath` falls back to `filepath.WalkDir` which recursively walks the entire repo tree. For a large repository this can be very slow and could cause performance problems. More critically, the walk uses `filepath.WalkDir` with an `err` parameter but the callback returns `nil` for directory errors (line 529), meaning permission-denied directories are silently skipped but the walk continues, potentially exhausting file descriptors or walking massive node_modules/vendor directories.

**Fix:** Add a depth limit and skip known noisy directories:

```go
func findRepoRelativePath(root, claimed string) string {
	base := filepath.Base(claimed)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	out, err := exec.Command("git", "-C", root, "ls-files", "--", "*"+base).Output()
	if err == nil {
		candidates := parseGitNameOutput(out)
		if len(candidates) == 1 {
			return candidates[0]
		}
		if len(candidates) > 1 {
			if best := bestMatchForClaimedPath(claimed, candidates); best != "" {
				return best
			}
		}
	}

	// Skip the filesystem walk entirely -- git already covers the repo.
	// If git ls-files found nothing, the file likely doesn't exist in the repo.
	return ""
}
```

### WR-03: Redundant state save in `runCodexBuildFinalize`

**File:** `cmd/codex_build_finalize.go:196-228`
**Issue:** At line 196-198, the function saves a checkpoint of the pre-modified colony state. Then at line 200-207, it modifies `updatedState` (a local copy) and writes events. Then at line 227, it saves the full modified state via `store.SaveJSON("COLONY_STATE.json", updatedState)`. However, between lines 207 and 227, the function calls `claimsOrAggregate` (line 209), `writeCodexBuildOutcomeReports` (line 214), `buildCodexBuildManifest` (line 219, which writes manifest.json), and `recordExternalBuildSpawnTree` (line 224). If any of these fail, the function returns an error without having saved the state mutation to `updatedState` at all. But the checkpoint at line 196-198 was already written, creating a state where the checkpoint says one thing and the live state says another. The atomic save via `store.UpdateJSONAtomically` (used in `runCodexBuildWithOptions` at line 330) is not used here, creating a race window.

Additionally, the function builds `finalManifest` at line 219 but writes the `updatedState` separately at line 227 without atomicity -- if the manifest write succeeds but the state write fails, the colony state is inconsistent.

**Fix:** Wrap the state mutation and saves in a single atomic operation:

```go
if err := store.UpdateJSONAtomically("COLONY_STATE.json", &updatedState, func() error {
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs)
	reconcileCompletedBuildTasks(&updatedState, phaseNum, dispatches)
	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|build_completed|build-finalize|Phase %d external Task workers recorded", completedAt.Format(time.RFC3339), phaseNum),
	)
	return nil
}); err != nil {
	return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to save built colony state: %w", err)
}
```

## Info

### IN-01: `resolveReviewDepth` is unused dead code

**File:** `cmd/review_depth.go:27-42`
**Issue:** The `resolveReviewDepth` function (2-level: light/heavy) is called by zero production code paths. All call sites use the 3-level `resolveVerificationDepth` (VerificationDepthLight/Standard/Heavy). The `ReviewDepth` type and the `resolveReviewDepth` function exist only for test coverage.

**Fix:** Consider marking `resolveReviewDepth` as deprecated, or removing it if no backward compatibility contract requires it.

### IN-02: `resolveVerificationDepthFlag` returns a bare string, not a typed constant

**File:** `cmd/review_depth.go:93-101`
**Issue:** `resolveVerificationDepthFlag` returns a raw `string` (e.g., "heavy", "light", or empty). Callers must pass this through `colony.NormalizeVerificationDepth` to get a typed `VerificationDepth`. If a caller forgets normalization, they would compare the raw string against typed constants. Currently the function is only used in tests, but if it were used in production it would be error-prone.

**Fix:** Return `colony.VerificationDepth` directly, or rename to make it clear the output is unnormalized.

### IN-03: Duplicate keyword lists between `heavyKeywords` and `securityRiskKeywords`

**File:** `cmd/review_depth.go:19-23` and `cmd/review_depth.go:106-109`
**Issue:** `heavyKeywords` (used by `phaseHasHeavyKeywords`) and `securityRiskKeywords` (used by `phaseRiskLevel`) contain nearly identical lists. Both include "security", "auth", "secrets", "permissions", "compliance", "audit". `securityRiskKeywords` additionally includes "token", "session", "password" while `heavyKeywords` additionally includes "release", "deploy", "production", "ship", "launch". These parallel lists will drift over time as one is updated without the other.

**Fix:** Consolidate into a single authoritative keyword list with metadata tags (risk level, category), or derive `heavyKeywords` from `securityRiskKeywords` plus deployment keywords.

### IN-04: `normalizeLegacyColonyState` is called on every visual render but never defined in reviewed files

**File:** `cmd/codex_visuals.go:378`
**Issue:** `workflowSuggestionsForState` calls `normalizeLegacyColonyState(state)` at line 378 but this function is not defined in any of the reviewed files. While this is likely defined elsewhere in the `cmd/` package, the function silently transforms state before rendering, and if the normalization is lossy it could mask bugs in the state machine.

**Fix:** This is informational only -- the function exists elsewhere in the package. No action needed, but noting it for cross-reference completeness.

---

_Reviewed: 2026-05-01T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
