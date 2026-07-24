# Manual Provider Smoke Checklist

Updated: 2026-05-17

## Purpose

Use this checklist when validating real Claude Code, OpenCode, or Codex worker
providers outside deterministic fake-provider CI.

For dummies: fake-provider tests prove Aether sends the right commands without
touching real accounts. This checklist is for the optional human run that proves
your real provider login can actually accept worker jobs.

## Preconditions

- Run from a clean Aether source checkout.
- Confirm the local runtime and hub agree:

```bash
aether version --check
```

- Confirm the provider you intend to smoke is installed and authenticated.
- Use a temporary test repository or a throwaway branch.
- Do not paste provider stdout, stderr, tokens, or auth probe details into
  issues, docs, release notes, or generated context.

## Provider Selection

Run one provider at a time so failures are attributable:

```bash
export AETHER_WORKER_PLATFORM=claude
# or
export AETHER_WORKER_PLATFORM=opencode
# or
export AETHER_WORKER_PLATFORM=codex
```

If testing a wrapper binary, point only the matching provider path:

```bash
export AETHER_CLAUDE_PATH=/absolute/path/to/claude
export AETHER_OPENCODE_PATH=/absolute/path/to/opencode
export AETHER_CODEX_PATH=/absolute/path/to/codex
```

## Smoke Flow

1. Start a tiny throwaway colony:

```bash
aether init "Provider smoke: add one harmless text fixture"
aether plan
```

2. Build only the first phase:

```bash
AETHER_OUTPUT_MODE=visual aether build 1
```

3. Verify and advance:

```bash
AETHER_OUTPUT_MODE=visual aether continue
```

4. Inspect state and worker evidence:

```bash
aether status
aether watch --once
aether proof
```

## Passing Evidence

A provider smoke passes only when all of these are true:

- The build command renders visible worker ceremony.
- `spawn-log` and `spawn-complete` entries appear in the spawn tree.
- The worker result contains valid Aether claims JSON.
- Claimed files are repo-relative, real, and outside `.aether/data`.
- `aether continue` advances or blocks for a real code/test reason, not a
  provider parse or auth diagnostic.

## Failure Triage

- If preflight fails, report only the sanitized provider, cause, and next action
  from the Go-owned diagnostic.
- If the worker launches but returns provider/API/auth output instead of worker
  claims JSON, classify it as a post-launch provider/API/auth failure.
- If secrets appear in output, stop and file a release blocker without copying
  the secret value.

## Release Rule

Manual live-provider smoke is optional for ordinary CI and should not replace
the deterministic fake-provider test suite. It becomes release-blocking only
when release notes claim live Claude/OpenCode/Codex provider validation.
