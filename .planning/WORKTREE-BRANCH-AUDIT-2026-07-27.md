# Worktree Branch Audit — 2026-07-27

Ten orphaned `worktree-agent-*` branches accumulated from GSD executor runs between
2026-05-01 and 2026-05-14. All were audited and deleted. **This file exists so every
deletion is reversible:**

```bash
git branch <name> <sha>     # objects survive until `git gc` prunes unreachable commits
```

## Verdict: all ten redundant

| Branch | Tip SHA | Last | Work it held | Verified in main as |
|--------|---------|------|--------------|---------------------|
| `worktree-agent-a03484ed` | `34513f9951ad3dc75696f2f3bd18915fb8d4fa48` | 2026-05-01 | Phase 90-02 learning trigger + classification | learning classification present in `cmd/`/`pkg/` |
| `worktree-agent-a282aaf2` | `253df68331148f0248d84837310d5d89c547048e` | 2026-05-01 | Phase 88-04 `/ant-unblock` gate recovery | `cmd/unblock_cmd.go` |
| `worktree-agent-a2c284cc` | `dcf126d913edbb2f7e9ec4fa50e232d15f23f8cc` | 2026-05-01 | Phase 86-01/86-03 verification-depth flags | `verificationDepth` in `cmd/` |
| `worktree-agent-a4e2071d` | `5ec5366fccd5ce7054ec45cd404283564508dc31` | 2026-05-01 | Phase 88-02 structured gate failures + circuit breaker | circuit-breaker code in `cmd/`/`pkg/` |
| `worktree-agent-a500919c` | `710f355e174148d424eb0dfb5737b50d41f042d5` | 2026-05-14 | nothing — 0 commits ahead of main | n/a |
| `worktree-agent-aacd3622` | `4892424ce279706595a51a8169b845cd39587616` | 2026-05-04 | Phase 99-01/99-02 `--verbose` + queen audit consolidation | `cmd/codex_build.go`, queen audit code |
| `worktree-agent-ac571a71` | `92143f9b07325374fa5b527a4278e445775b7dcb` | 2026-05-01 | Phase 89-02/89-03 Oracle confidence + Gate Status dashboard | "Gate Status" section in `cmd/` |
| `worktree-agent-ae7b5b85` | `5f2a28936f1dbaa46e7bce5d242d744163fcc7f1` | 2026-05-14 | nothing — 0 commits ahead of main | n/a |
| `worktree-agent-aeb32f46` | `c1085e09fccec9886db422b56c219112b7783164` | 2026-05-04 | Phase 99-01/99-02 (duplicate lineage of `aacd3622`) | same as above |
| `worktree-agent-aed1d973` | `71f6a419c5f31dde21c21271bc9184f6ff749a4a` | 2026-05-03 | Phase 97-01 queen decision layer | queen decision-layer code in `cmd/`/`pkg/` |

All correspond to milestones **v1.12–v1.14 (phases 80–99), shipped 2026-05-01 to 2026-05-04**.

## How redundancy was established

Three independent checks, not one:

1. **Headline feature present.** Each branch's distinctive feature was grepped for in main.
   All eight substantive branches: present.
2. **No branch-only files.** For each branch, every file it added since its merge-base was
   checked for existence in main. The only files absent from main were:
   - `.aether/agents-claude/*.md`, `.aether/agents-codex/*.toml`,
     `.aether/commands/{claude,opencode}/*.md`, `.codex/skills/aether/**` — repo-local
     **packaging mirrors**, deliberately removed. CLAUDE.md states the canonical sources
     are `.claude/agents/ant/`, `.opencode/agents/`, `.codex/agents/`, `.aether/commands/*.yaml`
     and that "there are no repo-local packaging mirrors." All canonical locations verified present.
   - `cmd/codex_queen_policy_stubs.go` (on `aeb32f46`) — the only genuine Go file. Every one
     of its functions was located in main, relocated into `cmd/references.go` and
     `cmd/codex_dispatch_contract.go`. It was a scaffold that got split into proper homes.
3. **Commit-count caveat.** Several branches report 200–586 commits "ahead" of main. This is
   diverged lineage (main was re-synced several times — see the `backup/*-sync-*` branches),
   not hundreds of unique changes. Judging by commit count alone would have badly
   overstated what was at risk.

## Also removed

| Path | What it was |
|------|-------------|
| `cmd/.claude/worktrees/agent-a6a7076d/.claude/settings.json` | A tracked `settings.json` accidentally committed from inside a dead agent worktree, nested under `cmd/`. Polluted repo-wide greps. |
| `cmd/.aether/worktrees/` | Untracked, gitignored, contained only a `.DS_Store`. |

## Retained deliberately

| Branch | Why |
|--------|-----|
| `worktree-agent-a7549759443bf5bc6` | Phase 160 plan 04 — **unmerged**, holds the live security-gate wiring. See `.planning/phases/160-fail-loudly/.continue-here.md`. |
| `worktree-agent-aefb65ae5ce753cd9` | Phase 160 plan 05 — **unmerged**, holds debug-artifact work. |

Their worktrees under `.claude/worktrees/` (571 MB) stay until those two plans are finished
and merged.

## Root cause

GSD's `execute-phase` merges each wave's worktree branches back and deletes them at the end
of the wave. When a run is interrupted — context exhaustion, a crash, or (as on 2026-07-27)
an account spend limit — that cleanup never runs, and the branches survive with no owner.
Nothing in the workflow reaps them on a later run. Expect this to recur after any
interrupted phase; re-run this audit rather than assuming abandoned branches are worthless.
