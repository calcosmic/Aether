# Phase 40: Stable Publish Hardening - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-23
**Phase:** 40-stable-publish-hardening
**Areas discussed:** Atomicity strategy, Publish entry point, Verification approach

---

## Atomicity Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Hub follows binary | Hub version.json updated to match binary version. Warns if hub was behind. | ✓ |
| Fail on mismatch | Both must already agree before install proceeds. | |
| Hub follows binary with tag gate | Hub follows binary, but only if source repo has matching git tag. | |

**User's choice:** Hub follows binary
**Notes:** Simplest approach — binary is the version authority. Hub version.json is a derived artifact.

---

## Publish Entry Point

| Option | Description | Selected |
|--------|-------------|----------|
| Fix existing install | Fix `aether install --package-dir` to guarantee version agreement. No new commands. | |
| New `aether publish` command | Dedicated command wrapping binary build + hub sync + version stamping. | ✓ |
| Fix now, publish later | Fix install now, add publish command later as convenience wrapper. | |

**User's choice:** New `aether publish` command
**Notes:** User specifically requested a new publish command and pointed to the operations guide for pipeline design. The operations guide currently shows `go run ./cmd/aether install --channel dev --package-dir "$PWD" --binary-dest "..."` — this should become `aether publish`.

---

## Verification Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-check + manual flag | Publish auto-checks version agreement. Also adds `aether version --check` for manual verification. | ✓ |
| Manual flag only | Just add a flag to `aether version` that checks agreement. No auto-check during publish. | |
| Dedicated integrity command | New `aether integrity` subcommand for full chain validation. | |

**User's choice:** Auto-check + manual flag
**Notes:** Phase 43 covers the full integrity chain (source → binary → hub → downstream). This phase only handles binary ↔ hub agreement at publish time.

---

## Claude's Discretion

- Exact `aether publish` command flag names and defaults
- Whether `aether publish` runs `go build` automatically or requires a pre-built binary
- Error message wording for version mismatch
- Test structure and organization

## Deferred Ideas

- Dev channel isolation (Phase 41)
- Downstream stale-publish detection (Phase 42)
- Full integrity chain check (Phase 43)
- Auto-version bumping from git tags
