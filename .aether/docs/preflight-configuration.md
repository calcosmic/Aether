# Preflight Configuration

Updated: 2026-08-01

This is the single documented surface for every env var that controls the
worker-provider preflight probe: what it does, its default, what happens when
you get the value wrong, and where the cache it writes lives on disk.

## For dummies

Before Aether spawns an expensive worker (Builder, Watcher, Scout, and so on),
it sends the provider one tiny real message — "Return exactly OK." — and
checks that the answer comes back clean. That single round-trip proves two
things at once: you're actually logged in, and the model you configured
actually works. It used to run before *every single* build/plan/continue
dispatch, which meant paying for that round-trip over and over even when
nothing about your login had changed. Now the colony remembers a good answer
for an hour (the "trust window") and skips the paid check entirely until that
window closes — or until a real dispatch fails for an auth reason, which
forgets the good answer immediately so the very next command re-checks before
trusting the provider again.

## What the probe is, and why it costs money

The preflight probe is a real model round-trip: the runtime invokes the
selected provider CLI (Claude Code, Codex, or OpenCode) with the prompt
`Return exactly OK.` and inspects the response. This is deliberately NOT a
cheap non-AI substitute — nothing else actually proves auth and model
configuration work together. Because it is a real request to the provider, it
has a real cost and a real latency, which is exactly why this phase adds
caching, a skip switch, and a documented timeout instead of running it on
every dispatch.

## The three knobs

| Env var | Syntax | Default | What it does | On a malformed value |
|---|---|---|---|---|
| `AETHER_PREFLIGHT_TIMEOUT` | Go duration using `ms`/`s`/`m`/`h` units, compounds included (e.g. `90s`, `1500ms`, `2m`, `1m30s`, `1.5h`) | `45s` | How long the probe is allowed to run before it's treated as a timeout. Applies to both the Go runtime and the TypeScript host — one knob, one value, both hosts. | Falls back to the 45s default. A broken env var never bricks dispatch. The TS host also prints one loud stderr warning naming the bad value, and the exotic Go units (`ns`, `us`, `µs`) count as malformed there. |
| `AETHER_PREFLIGHT_CACHE_TTL` | Go duration (e.g. `90s`, `1500ms`, `2m`) | `1h` | How long a successful probe is trusted before the next dispatch has to pay for a fresh one (the "trust window"). | Falls back to the 1h default. Same never-brick guarantee. |
| `AETHER_SKIP_PREFLIGHT` | one of `1`, `true`, `yes`, `on` (case-insensitive) | unset (probe runs) | Skips the probe entirely for trusted setups or CI. Prints one loud warning line every time it's used — this switch is never silent, because a skipped auth check is a real risk the person running it needs to see. | Any other value (including `0`, `false`, or a typo) is treated as "not set" and the probe runs normally. |

## The cache

- **File:** `.aether/data/preflight-cache.json` (a runtime-owned state file,
  following the same convention as `COLONY_STATE.json` and the
  `handoffs/` directory).
- **Keyed per platform.** Switching `AETHER_WORKER_PLATFORM` between Claude,
  Codex, and OpenCode never destroys another platform's trust window, and a
  lookup for one platform never accidentally reuses another's cached success.
- **Only successes are ever stored.** A failed probe is never cached — a
  lapsed login keeps failing on every dispatch until it's actually fixed,
  instead of being remembered as "still broken" and skipped forever.
- **TTL-bounded.** Once the trust window (`AETHER_PREFLIGHT_CACHE_TTL`)
  closes, the next dispatch re-probes automatically.
- **Cleared automatically on a provider/auth-classified dispatch failure.**
  If a worker dispatch fails for a reason that means the *provider*, not the
  task, is broken (a lapsed login, missing credentials, invalid provider
  config), the cached entry for that platform is deleted immediately, so the
  very next command re-checks instead of trusting a stale success for up to
  an hour. Ordinary task failures (a timeout, a blocked task, a failing test
  suite) never touch the cache — a flaky task must not cost the next
  dispatch a paid probe.

## What you see

A cache hit prints one quiet line to stderr so you know why dispatch started
fast:

```
preflight: cached OK for claude, 43m left in trust window
```

A skipped probe prints one loud line — never silent:

```
Warning: preflight skipped via AETHER_SKIP_PREFLIGHT — provider auth and model config were NOT verified before dispatch
```

## Both hosts agree

The Go runtime (`cmd/preflight_cache.go`, `pkg/codex/platform_dispatch.go`)
and the TypeScript host (`.aether/ts-host/src/preflight-config.ts`,
`.aether/ts-host/src/host.ts`) read the same env var names, share the same
45-second default timeout, and print byte-identical skip-notice wording.
`cmd/preflight_docs_test.go` fails the build the moment either host drifts
from the other, or the moment a new `AETHER_PREFLIGHT_*`/
`AETHER_SKIP_PREFLIGHT` knob ships without an entry in this table.
