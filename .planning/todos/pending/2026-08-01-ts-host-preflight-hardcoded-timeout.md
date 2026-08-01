---
created: 2026-08-01T10:05:00Z
title: ts-host preflight timeout hardcoded, not configurable, runs in repo cwd
area: ts-host
source: M4L repo residual-issues check against v1.0.46 (original diagnosis session, item P1)
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
