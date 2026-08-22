---
created: 2026-08-01T10:05:00Z
title: ts-host preflight timeout hardcoded, not configurable, runs in repo cwd
area: ts-host
source: M4L repo residual-issues check against v1.0.46 (original diagnosis session, item P1)
audit_acknowledged:
  milestone: v1.26
  at: 2026-08-22
---

## Problem

`.aether/ts-host/src/platform-dispatcher.ts:155` hardcodes the worker-platform
preflight timeout at `20_000` ms:

```ts
const result = await runPreflight(binary, args, cwd, "", 20_000);
```

- No `AETHER_*PREFLIGHT*` env var or config key exists to tune it
- Preflight runs with the repo as working directory
- Downstream cost report from the M4L colony: ~$0.59 per dispatch attributable
  to preflight, plus timeout fragility on slow provider auth

## Verified still open in v1.0.46

Phase 163.1 (wrapper-runtime completion contract) did not touch ts-host; last
changes to this file are PRs #75/#69. Confirmed by grep on 2026-08-01.

## Suggested shape of fix

- Read timeout from env (e.g. `AETHER_PREFLIGHT_TIMEOUT_MS`) or `.aether` config,
  defaulting to 20s

- Consider a cheaper/cached preflight (per-session, not per-dispatch)
- Remember: ts-host changes need a dist rebuild and publish to reach downstream
  repos (see M4L dispatch postmortem — fixes need both Go AND ts-host paths)

---

## Matching gap in the same file: the cheap auth probe (added 2026-08-21)

`runProbe` (`.aether/ts-host/src/platform-dispatcher.ts:594`) hardcodes
`spawn(binary, args, { timeout: 5000 })` for `claude auth status --json` and
`codex login status`. No env override, no retry, and Node's `spawn` timeout signals only
the direct child — so a wrapper CLI that leaves a grandchild holding the pipes is not
actually bounded by it.

This is the same defect the Go side had and that was fixed on 2026-08-21: budget raised to
10s, one retry for timeouts only, `AETHER_PROBE_TIMEOUT` override, and
`configureWorkerCommand` for process-group teardown (see
`pkg/codex/platform_dispatch.go`, `TestAvailabilityProbeBudgetBoundsWallClock` — without
the teardown a 200ms budget took **60 seconds**).

**Not fixed here because** the field failure that prompted the Go fix (Formica build,
2026-08-21, two of four workers lost to `claude auth status failed: timed out`) came
through the Go path — its error wording is Go's `formatAvailabilityProbeError`. The TS
twin is the same shape but has not been observed failing, so it is logged rather than
changed on speculation. Fix it alongside the preflight timeout above; the Go side is the
worked example.

Node has no direct process-group option, so the teardown half needs `detached: true` plus
`process.kill(-pid)` rather than a straight port.
