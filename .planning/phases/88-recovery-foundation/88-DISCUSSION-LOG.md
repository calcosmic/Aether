# Phase 88: Recovery Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-01
**Phase:** 88-Recovery Foundation
**Areas discussed:** Provenance depth, Gate failure UX, Privacy gate scope, Unblock command design

---

## Provenance depth

| Option | Description | Selected |
|--------|-------------|----------|
| Metadata-only | Validate worker result JSON has status=success AND files_modified > 0. Fast, no disk I/O, safe against worktree false negatives. | ✓ |
| Metadata + file existence | Also check that at least one claimed file exists on disk. Catches phantom claims but risks false negatives in worktree mode. | |
| Full git diff verification | Full git diff verification — check actual file changes match claimed modifications. Most thorough but expensive and fragile in worktree mode. | |

**User's choice:** Metadata-only
**Notes:** Worktree isolation makes filesystem-level checks unreliable. Metadata is the safest v1 approach per research recommendation.

| Option | Description | Selected |
|--------|-------------|----------|
| Build gate only | Build-complete rejects phantom builds. Continue trusts the build was clean when it ran. | ✓ |
| Both build and continue | Build rejects AND continue re-validates provenance. Double-check catches mid-session corruption. | |

**User's choice:** Build gate only
**Notes:** SAFE-03/04 are locked requirements so continue must trace provenance — the question became how deep that tracing goes.

| Option | Description | Selected |
|--------|-------------|----------|
| Manifest lookup | Continue checks stored build manifest for valid worker run (status + files_modified). No filesystem re-check. | ✓ |
| Disk re-validation | Continue re-reads worker results from disk, compares timestamps. More thorough but adds I/O. | |

**User's choice:** Manifest lookup

| Option | Description | Selected |
|--------|-------------|----------|
| Reject and halt | Reject the claim and stop with clear message pointing to /ant-continue or /ant-unblock. | ✓ |
| Warn but allow | Log warning but allow advancement. | |

**User's choice:** Reject and halt

---

## Gate failure UX

| Option | Description | Selected |
|--------|-------------|----------|
| JSON + wrapper rendering | Go runtime outputs structured JSON. Wrapper markdown renders formatted messages. Codex gets JSON. Consistent with OutputWorkflow pattern. | ✓ |
| Formatted text only | Go runtime outputs formatted text directly. Simpler but breaks JSON machine-readability pattern. | |

**User's choice:** JSON + wrapper rendering

| Option | Description | Selected |
|--------|-------------|----------|
| Aggregated summary | All gate failures in one block with per-gate status, then single recovery choice. | ✓ |
| Sequential per-gate | Show one gate at a time, resolve each before next. | |

**User's choice:** Aggregated summary

| Option | Description | Selected |
|--------|-------------|----------|
| Structured recovery messages | What failed, why, how to fix, two recovery options. Matches GATE-02 exactly. | ✓ |
| Warning with recovery hints | Softer warning with some urgency language. Middle ground. | |

**User's choice:** Structured recovery messages

| Option | Description | Selected |
|--------|-------------|----------|
| Extend gateCheck struct | Add optional fix_hint and recovery_options fields. Backward compatible with omitempty. | ✓ |
| New result type | New gate-failure-result type. Cleaner separation but more types. | |

**User's choice:** Extend gateCheck struct

---

## Privacy gate scope

| Option | Description | Selected |
|--------|-------------|----------|
| Standard secret patterns | API keys, private keys, passwords, env file patterns. Built on existing sanitize.go. | ✓ |
| Standard + path blocking | Plus path-based blocking (~/.ssh, ~/.aws, ~/.config/gcloud). More thorough but more false positives. | |
| Standard + custom patterns | Plus user-configurable regex. Most flexible but more complex. | |

**User's choice:** Standard secret patterns
**Notes:** PRIV-02 requires "local user paths" — clarified as redaction rather than blocking.

| Option | Description | Selected |
|--------|-------------|----------|
| Redact paths, don't block | Scrub absolute home directory paths from content, allow write to proceed. | ✓ |
| Block writes with paths | Reject any write containing home directory paths. | |

**User's choice:** Redact paths, don't block

| Option | Description | Selected |
|--------|-------------|----------|
| Block + log | Reject entire write, log matched pattern. Matches PRIV-01 "blocks writes" language. | ✓ |
| Redact + allow write | Strip secret and write the rest. More useful but risks leaking context. | |

**User's choice:** Block + log

| Option | Description | Selected |
|--------|-------------|----------|
| Extend existing security | Add privacy-scan to security_cmds.go, patterns alongside sanitize.go. | ✓ |
| New privacy package | New pkg/privacy/ with scanner, patterns, redaction. | |

**User's choice:** Extend existing security

---

## Unblock command design

| Option | Description | Selected |
|--------|-------------|----------|
| Info + manual path | Show gate failures, offer /ant-continue or view fix hints. No Fixer dispatch (Phase 89). | ✓ |
| Info + selective retry | Show failures and offer to retry individual gates. | |

**User's choice:** Info + manual path

| Option | Description | Selected |
|--------|-------------|----------|
| New Go command file | New cmd/unblock_cmd.go with cobra command. Follows existing pattern. | ✓ |
| Extend gate.go | Add unblock subcommand to gate.go. | |

**User's choice:** New Go command file

| Option | Description | Selected |
|--------|-------------|----------|
| Per-phase JSON file | gate-results.json in .aether/data/ per phase. Contains gate results with name, status, detail, fix_hint, timestamp, retry_count. | ✓ |
| Inside COLONY_STATE.json | Embedded as new field. Single file but risks bloat/corruption. | |

**User's choice:** Per-phase JSON file

| Option | Description | Selected |
|--------|-------------|----------|
| Except Flags + Watcher Veto | Those two always re-run (live state). All others skip if previously passed/skipped. | ✓ |
| Always re-run all | Simpler but slower. | |

**User's choice:** Except Flags + Watcher Veto

---

## Claude's Discretion

- Exact regex patterns for secret detection
- Gate result JSON schema field naming and nesting
- Error message wording for build provenance rejection
- Gate-results.json per-phase naming convention

## Deferred Ideas

None — discussion stayed within phase scope.
