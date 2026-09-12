# Deferred Items — Phase 202.1 Classic Visual Voice

## Pre-existing failure discovered during Plan 01 Task 1 verification

**Test:** `TestNextActionNeverHardcoded` (named in Plan 01 Task 1's `<verify>` block)

**Finding:** `cmd/swarm_cmd.go:runSwarmDestroy` (line ~552) hand-writes a
`"next": "aether status"` literal into a result map instead of routing it
through `resolveNextAction`/`applyNextActionToResult`. This is caught by the
hardcode ratchet as a new, unrecorded hand-typed command-advice site.

**Confirmed out of scope for this plan:**
- `cmd/swarm_cmd.go` carries zero diff against this plan's base commit
  (`2757534e2b8f35d4099ac1307786d3ed02598698`) — this executor made no
  change to that file.
- `cmd/swarm_cmd.go` is not in Plan 01's declared `files_modified` list, and
  the fix (routing swarm's repair-not-restored result through the shared
  next-action resolver) requires understanding `runSwarmDestroy`'s
  intervention/repair contract, which is outside this phase's research
  domain (Classic visual voice/glyph density).
- `cmd/testdata/next_action_hardcode_baseline.json` was last touched in
  Phase 200 plan 54, well before the swarm repair code that introduced this
  literal — confirming the drift predates Phase 202.1 entirely.

**Action:** Not auto-fixed (Scope Boundary — pre-existing failure in an
unrelated file). Every other test named in Plan 01 Task 1's `<verify>` block
passes. Route this through the normal continue/verify cycle for
`swarm_cmd.go`'s owning phase, or open a follow-up plan.
