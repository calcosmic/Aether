---
phase: 84-verification-depth-extension
plan: 02
status: complete
---

# Plan 84-02: Update YAML sources and platform wrappers for verification depth

## Summary

Updated YAML source definitions and both platform wrapper markdowns (Claude Code and OpenCode) to expose the new 3-level verification depth system. Users now see `--verification-depth <light|standard|heavy>` instead of the old `--light`/`--heavy` flags as the primary interface.

## What Changed

### Task 1: YAML Source Definitions
- `.aether/commands/continue.yaml`: Changed default runtime command from `--light` to `--verification-depth standard`, updated heavy path to `--verification-depth heavy`, added `verification_depth_ceremony` section explaining all three levels
- `.aether/commands/build.yaml`: Added note about `--verification-depth` flag availability (build command doesn't directly use depth flags in its base command)

### Task 2: Platform Wrapper Markdowns
- `.claude/commands/ant/continue.md`: Updated default command to `--verification-depth standard`, added `## Verification Depth` section with 3-level documentation, updated heavy external review path
- `.claude/commands/ant/build.md`: Added `## Verification Depth` section documenting the `--verification-depth` flag
- `.opencode/commands/ant/continue.md`: Same changes as Claude continue.md (structural parity confirmed)
- `.opencode/commands/ant/build.md`: Same changes as Claude build.md (structural parity confirmed)

## Key Decisions
- Kept `--light` and `--heavy` mentioned as backward-compatible aliases in documentation
- Default changed from `light` to `standard` for continue (probe-only review by default)
- Structural parity maintained between Claude and OpenCode wrappers

## Self-Check: PASSED
- [x] `verification-depth` present in all 6 target files
- [x] `## Verification Depth` section in both continue.md wrappers
- [x] No `--light`/`--heavy` in runtime command invocations (only in alias docs)
- [x] Claude and OpenCode wrappers have structural parity (identical content)

## Files Modified
- `.aether/commands/continue.yaml`
- `.aether/commands/build.yaml`
- `.claude/commands/ant/continue.md`
- `.claude/commands/ant/build.md`
- `.opencode/commands/ant/continue.md`
- `.opencode/commands/ant/build.md`
