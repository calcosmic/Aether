---
phase: 206-screens-reach-the-owner
reviewed: 2026-09-22T00:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - .claude/settings.json
  - CLAUDE.md
  - cmd/claudemd_direct_screen_test.go
  - cmd/hook_cmds.go
  - cmd/hook_direct_screen_test.go
  - cmd/hook_direct_screen.go
  - cmd/hook_session_start_test.go
  - cmd/status_line_test.go
  - cmd/status_line.go
  - cmd/stop_hook_screen_test.go
  - cmd/subcommand_reachability_ratchet_test.go
  - cmd/testdata/post-tool-use/status-screen-payload.json
  - scripts/proof-screens-reach-the-owner.sh
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 206: Code Review Report

**Reviewed:** 2026-09-22
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Reviewed the direct screen-relay route (`aether hook-post-tool-use`), the new
status line (`aether status-line`), the Stop-hook backstop's new
"already-delivered" exemption, and the settings/documentation/proof-script
changes that ship them. The core mechanism (`directScreenDelivery`,
`directScreenMessageWithinCap`, the one-decision sharing between the direct
route and the backstop) is sound and the targeted test suite for this phase
(`go test ./cmd -run 'TestDirectRoute|TestStatusLine|TestBackstop|...'`)
passes in full — 34/34 subtests green, no regressions found.

Four issues surfaced under adversarial reading, none of them data-loss or
security-grade, but two are real correctness gaps that the shipped tests do
not cover, and two are documentation-accuracy gaps that matter specifically
because this repository's own stated culture treats an unverified or
inconsistent documentation claim as a defect in its own right.

## Warnings

### WR-01: `directScreenRouteRegistered` never checks the hook's matcher, so it can wrongly report "already delivered"

**File:** `cmd/hook_direct_screen.go:120-170`
**Issue:** `directScreenRouteRegistered` (and the struct it parses settings
with, `directScreenHookSettingsFile`) decides whether the direct route is
"installed" purely by scanning every `PostToolUse` entry's inner `hooks[].command`
for the literal string `aether hook-post-tool-use`. It never reads or checks
the entry's own `matcher` field:

```go
type directScreenHookSettingsFile struct {
	Hooks map[string][]struct {
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}
...
for _, entry := range parsed.Hooks["PostToolUse"] {
    for _, h := range entry.Hooks {
        if strings.HasPrefix(strings.TrimSpace(h.Command), "aether hook-post-tool-use") {
            return true
        }
    }
}
```

`screenRelayBlockReason` (`cmd/hook_cmds.go:359-383`) uses this function's
answer to decide whether the Stop-hook backstop can stay silent, on the
strength of CLAUDE.md's own claim: *"the whole screen provably already
arrived through the direct route, and this project's own settings have the
direct route installed."* But a `PostToolUse` entry that names this exact
command under a matcher that excludes `Bash` (e.g. `"matcher": "Write|Edit"`)
would still satisfy this check, even though the hook would never actually
fire for the Bash call that drew the screen — so the screen was never
delivered at all, and the backstop would incorrectly stay quiet, silently
losing the very guarantee Phase 205/206 exist to provide. `aether update`'s
own merge always ships the command under a `Bash` matcher, so this cannot
happen via the normal install path, but the function is general-purpose (it
also reads a project's own hand-edited `.claude/settings.local.json`) and
nothing enforces the matcher anywhere in this call path. No test exercises a
non-Bash-matcher registration of this exact command (only
`TestDirectRouteHookIsRegistered`, a different function, checks the matcher —
against the *shipped* file only).
**Fix:** Add a `Matcher string` field to the inner entry struct and require
`Bash` to appear in it (splitting on `|`, matching
`TestDirectRouteHookIsRegistered`'s own logic) before returning true:
```go
type directScreenHookSettingsFile struct {
	Hooks map[string][]struct {
		Matcher string `json:"matcher"`
		Hooks []struct {
			Command string `json:"command"`
		} `json:"hooks"`
	} `json:"hooks"`
}
...
for _, entry := range parsed.Hooks["PostToolUse"] {
    matchesBash := false
    for _, m := range strings.Split(entry.Matcher, "|") {
        if strings.TrimSpace(m) == "Bash" || strings.TrimSpace(entry.Matcher) == "" {
            matchesBash = true
        }
    }
    if !matchesBash {
        continue
    }
    for _, h := range entry.Hooks {
        if strings.HasPrefix(strings.TrimSpace(h.Command), "aether hook-post-tool-use") {
            return true
        }
    }
}
```

### WR-02: `shortenStatusLineGoal` truncates mid-word, contradicting its own doc comment

**File:** `cmd/status_line.go:64-78`
**Issue:** The function's own comment says it "cuts back to the last whole
word that fits and appends a single-character ellipsis -- never mid-word."
The implementation only handles that case when a space exists inside the
first `statusLineMaxTaskRunes` (40) runes:

```go
window := string(runes[:statusLineMaxTaskRunes])
if idx := strings.LastIndex(window, " "); idx > 0 {
	window = window[:idx]
}
return strings.TrimRight(window, " ") + "…"
```

When the goal's first 40+ characters contain no space at all (a single long
identifier, URL, or filename used as a task goal — plausible for this
project, which routinely has task goals naming file paths), `idx` is `-1`,
the guard is skipped, and the raw 40-rune window is used as-is — cutting the
word in half. Verified directly:
```
goal := "ImplementTheVeryLongSingleWordIdentifierWithNoSpacesAtAllInItWhichExceedsTheCharacterLimit"
shortenStatusLineGoal(goal) // => "ImplementTheVeryLongSingleWordIdentifier…" (cut mid-word)
```
No test in `cmd/status_line_test.go` exercises a goal shaped this way — every
fixture goal used contains multiple short words.
**Fix:** Fall back to a hard rune cut only when no space exists, and treat
that as the documented exception, or change the doc comment to match reality.
Minimal correctness fix (still respects the "never mid-word" contract when a
word boundary exists, and is honest about the one case it can't avoid):
```go
if idx := strings.LastIndex(window, " "); idx > 0 {
    window = window[:idx]
}
// idx == -1 (no space in the window at all): there is no whole-word boundary
// to cut back to, so the single long word itself is truncated -- document
// this explicitly rather than leaving the "never mid-word" claim false.
```
At minimum, update the doc comment to no longer claim an unconditional
guarantee the code does not provide.

### WR-03: Status line feature is entirely undocumented in CLAUDE.md

**File:** `CLAUDE.md` (no matching section); `cmd/status_line.go`
**Issue:** This phase shipped two features: the direct screen route (Phase
206 plan 01) and the permanent status line (Phase 206 plan 02,
`cmd/status_line.go`, 150 new lines, its own 425-line test file, a new
top-level `statusLine` key in `.claude/settings.json`, and a dedicated
`subcommand_reachability_ratchet_test.go` change to credit it). CLAUDE.md
documents the direct-route feature exhaustively (a full new `###` section,
"for dummies" explanation, and a dedicated ratchet test —
`TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest` — that fails if any
cited test symbol for that section goes missing) but contains **zero**
mentions of the status line: no section, no reference to `aether status-line`
or the `statusLine` settings key anywhere in the file (confirmed by
case-insensitive grep). CLAUDE.md's own header states this repository's
policy is that "a documentation claim about runtime behaviour must be
testable or removed" — the inverse gap (a shipped, tested, user-visible
runtime behavior with no documentation claim at all, and no ratchet
protecting it from silent regression) is exactly the kind of drift this
project's `claudemd_*_test.go` guards exist to prevent for its sibling
feature in the same phase.
**Fix:** Add a `### The permanent status line (v1.29, Phase 206)` section to
CLAUDE.md (mirroring the direct-route section's structure: what it does, its
"decides nothing / writes nothing" guarantees, and citations to
`TestStatusLineIsRegistered`, `TestStatusLineComesFromTheSharedDecision`,
`TestStatusLineIsSilentWithoutAProject`, `TestStatusLineChangesNothingAndRepeatsItself`,
`TestStatusLineIsSafeUnderConcurrentReads`, `TestStatusLineSpeaksPlainEnglish`,
`TestStatusLineInstallOnlyWhenTheProjectHasNone`), and extend
`claudeMDDirectScreenSections` (or add a sibling ratchet) to cover it.

## Info

### IN-01: Proof script comment misattributes the platform's own message prefix to Aether's code

**File:** `scripts/proof-screens-reach-the-owner.sh:170-176`
**Issue:** The comment above the "display refinement" step claims:
> "every hook-relayed line carries a stable `"<hook name> says: "` prefix
> (Aether's own `directScreenDelivery` composes it, and it is what the owner
> actually sees on screen)"

This is incorrect. Neither `directScreenDelivery` nor `emitDirectScreen`
(`cmd/hook_direct_screen.go`) construct any `"<hook> says: "` prefix — the
message they emit is the screen text verbatim (or its truncated tail behind
`directScreenTruncationNotice`). The `"PostToolUse:Bash says: <line>"` shape
the script's own gate-4 comment describes a few lines earlier (line 134-135)
is Claude Code's own platform-rendered wrapper around the delivered
`systemMessage`, not something Aether's Go code produces. A future
maintainer debugging why "the prefix disappeared" could waste time looking
in the wrong codebase.
**Fix:** Reword to attribute the prefix correctly, e.g.: "every hook-relayed
line Claude Code itself shows carries a stable platform-added `"<hook name>
says: "` prefix (observed empirically, not something Aether's code
constructs)..."

### IN-02: Newly committed real-capture fixture embeds the exact raw state token CLAUDE.md says must never reach the owner

**File:** `cmd/testdata/post-tool-use/status-screen-payload.json:18`
**Issue:** This phase committed a new, real-capture fixture of a genuine
`aether status` screen. Its `tool_response.stdout` field contains the line:
```
- 📜 handoff=handoff-060d355f95c7b74a744bd164 between_commands_boundary
```
CLAUDE.md's "Classic Visual Voice" section names `between_commands_boundary`
verbatim as the canonical example of a raw internal state token that "is
never printed to the owner" and is locked by
`TestLifecycleEventSentenceTypeCannotBeBypassed`. This fixture is real,
byte-real capture evidence (per its own provenance comment in
`cmd/hook_direct_screen_test.go`) that, as of the capture date, `aether
status`'s WHAT NEXT card still leaks that exact raw token in production
output. This is not a defect in any file changed by this phase (the leak
lives in the "what changed" line renderer, outside this diff's scope), and
none of the direct-route or status-line tests depend on that specific
substring, so it does not affect this phase's correctness. Flagged only
because the evidence is now permanently committed as part of this phase's
own test data and directly contradicts a claim this repo already asserts is
locked elsewhere.
**Fix:** Out of scope for this phase; worth a follow-up ticket against the
"what changed" card renderer (likely `renderNextActionCard` /
`syncColonyArtifacts` handoff-summary formatting) and its own voice-corpus
test coverage.

---

_Reviewed: 2026-09-22_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
