---
phase: 20-visual-ux-restoration-emoji-consistency
reviewed: 2026-04-28T12:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - cmd/codex_visuals.go
  - cmd/codex_visuals_test.go
  - cmd/status.go
  - cmd/swarm_cmd.go
  - cmd/compatibility_cmds.go
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 20: Code Review Report

**Reviewed:** 2026-04-28T12:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Phase 20 introduced `commandEmojiMap` (a centralized banner emoji map) and `commandEmoji()` helper, then refactored most `renderBanner()` calls to use the map instead of hardcoded emoji strings. The five reviewed files show consistent use of the new map for their own render calls. However, the map is incomplete -- several commands that still hardcode emoji in `renderBanner()` calls in *other* files (outside the review scope) are missing from `commandEmojiMap`. Additionally, a pre-existing bug in `renderStalePublishBanner` produces an empty emoji for the `staleCritical` classification, which results in a malformed banner.

## Critical Issues

### CR-01: renderStalePublishBanner produces empty emoji for staleCritical, resulting in malformed banner

**File:** `cmd/codex_visuals.go:2631-2632`
**Issue:** When `stale.Classification == staleCritical`, the emoji is set to an empty string `""`. The `renderBanner` function formats as `"━━ %s %s ━━\n"`, so the critical stale publish banner renders as `"━━  S T A L E   P U B L I S H   D E T E C T E D ━━"` with a visible gap where the emoji should be. This is the highest-severity stale publish case and it gets the *worst* visual treatment -- no emoji at all.

**Fix:**
```go
case staleCritical:
    emoji = "🚨"
    title = "STALE PUBLISH DETECTED"
```

## Warnings

### WR-01: commandEmojiMap is incomplete -- multiple commands still hardcoded outside reviewed files

**File:** `cmd/codex_visuals.go:112-172`
**Issue:** Several `renderBanner()` calls across the codebase still use hardcoded emoji strings instead of `commandEmoji()`, and the corresponding keys are missing from `commandEmojiMap`. The incomplete keys are:

| Hardcoded Call | File | Emoji | Status |
|---|---|---|---|
| `renderBanner("⚰️", "Entomb")` | `entomb_cmd.go:676` | ⚰️ | Key exists in map, file doesn't use it |
| `renderBanner("🧭", "Discuss")` | `discuss.go:394` | 🧭 | Key exists in map, file doesn't use it |
| `renderBanner("🔎", "Proof")` | `proof_cmd.go:297` | 🔎 | Key missing from map |
| `renderBanner("🧠", "Profile")` | `profile.go:571,590,611` | 🧠 | Key missing from map |
| `renderBanner("🧠", "Assumptions")` | `assumptions.go:435,458,482` | 🧠 | Key exists in map, file doesn't use it |
| `renderBanner("❌", "Error")` | `helpers.go:172` | ❌ | Key missing from map |
| `renderBanner("🔮🐜", "Oracle Loop")` | `oracle_loop.go:2015` | 🔮🐜 | Compound emoji, not in map |
| `renderBanner("🔁", "Oracle Retry")` | `oracle_loop.go:2035` | 🔁 | Key missing from map |

The keys `"entomb"`, `"discuss"`, and `"assumptions"` exist in the map but the respective files bypass it. The keys `"proof"`, `"profile"`, `"error"`, and `"oracle-loop"`/`"oracle-retry"` are not in the map at all.

**Fix:** Add missing keys to `commandEmojiMap` and refactor remaining hardcoded calls to use `commandEmoji()`:
```go
// Add to commandEmojiMap:
"proof":         "🔎",
"profile":       "🧠",
"error":         "❌",
"oracle-loop":   "🔮",
"oracle-retry":  "🔁",
```
Then update the call sites in `entomb_cmd.go`, `discuss.go`, `proof_cmd.go`, `profile.go`, `assumptions.go`, `helpers.go`, and `oracle_loop.go` to use `commandEmoji()`.

### WR-02: Test TestWrapperDescriptionEmojiConsistency only checks for stale 🐜, does not verify correct emoji per command

**File:** `cmd/codex_visuals_test.go:1948-1992`
**Issue:** The test reads `.claude/commands/ant/` and `.opencode/commands/ant/` to verify no `🐜` appears in description lines. This is a negative check only -- it verifies that the old generic emoji was removed but does not verify that the correct command-specific emoji from `commandEmojiMap` is present in each wrapper's description. A command file could use a completely wrong emoji (e.g., `🔨` instead of `📋` for the plan command) and this test would not catch it.

**Fix:** Add a positive verification test that cross-references command file names with `commandEmojiMap` entries to confirm each wrapper description uses the correct emoji:
```go
func TestWrapperDescriptionUsesCorrectCommandEmoji(t *testing.T) {
    repoRoot, err := repoRootForCommandSourceTest()
    if err != nil {
        t.Fatalf("repo root: %v", err)
    }
    dirs := []string{
        filepath.Join(repoRoot, ".claude", "commands", "ant"),
        filepath.Join(repoRoot, ".opencode", "commands", "ant"),
    }
    for _, dir := range dirs {
        entries, err := os.ReadDir(dir)
        if err != nil {
            continue
        }
        for _, entry := range entries {
            if !strings.HasSuffix(entry.Name(), ".md") {
                continue
            }
            cmdKey := strings.TrimSuffix(entry.Name(), ".md")
            expectedEmoji, ok := commandEmojiMap[cmdKey]
            if !ok {
                continue
            }
            content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
            if err != nil {
                continue
            }
            for _, line := range strings.Split(string(content), "\n") {
                if strings.HasPrefix(line, "description:") && !strings.Contains(line, expectedEmoji) {
                    t.Errorf("%s: description missing expected emoji %q: %s",
                        filepath.Join(dir, entry.Name()), expectedEmoji, line)
                }
            }
        }
    }
}
```

## Info

### IN-01: commandEmoji("medic") and casteEmoji("medic") return the same emoji

**File:** `cmd/codex_visuals.go:171` (casteEmojiMap) and `cmd/codex_visuals.go:171` (commandEmojiMap)
**Issue:** Both `commandEmojiMap["medic"]` and `casteEmojiMap["medic"]` return `"🩹"`. This means the medic command banner and medic caste identity are visually identical. If the medic command and medic caste serve different roles in the UI, they may benefit from distinct emoji. This is a design observation, not a bug.

---

_Reviewed: 2026-04-28T12:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
