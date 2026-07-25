# Session handoff — 25/26 July 2026

Written for a fresh context. Assumes no knowledge of the previous session.

---

## One-paragraph summary

Aether's worker prompts were measured at **76.6% orchestrator boilerplate** — every worker was being told *"YOU (the Queen) will spawn workers directly"* at five times the mass of its own assignment. That is fixed in both code paths, published to the hub, and verified working in a real project. Along the way the in-flight hive trust redesign was finished and the memory generator that produced 67 unusable wisdom entries was closed off. **Nothing has been measured yet** — the fix is proven mechanically, not yet proven useful.

---

## Current state

| | |
|---|---|
| Branch | `main`, working tree clean |
| Tests | `go test ./...` green (18 packages) · `-race` green · `go vet` clean · 516/516 TS host |
| Hub | Published and verified. `aether version --check` passes, binary and hub both `1.0.42` |
| Binary | `~/.local/bin/aether`, built from committed code |
| Pushed | **No.** 315 commits unpushed to origin — that backlog predates this session |

### Commits from this session

```
339dd88c docs: v1.25 milestone planning and phase archive
101d03d9 fix: finish the hive trust redesign and gate what enters memory
5b01f361 fix: stop the TS host re-injecting playbooks into worker briefs
3c65a9a3 fix: revert an in-flight hive call that was committed by mistake
efd5e663 merge: worker prompt fix — stop telling workers they are the Queen
01551c53 fix: stop telling workers they are the Queen
```

---

## The next thing to do

**Run `/ant-build` on a real phase in `/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System` and judge it as a user.** That colony is healthy — 4 phases planned, 8 signals active, runtime and hub agree, no migration needed.

What to watch for:

- **Working** — workers start editing sooner, less "let me first understand the structure", fewer runs where a worker does nothing useful.
- **Not working** — wrong-file edits or ignored conventions, meaning workers now lack context they genuinely needed. The fix would be adding *specific* context back, never the playbook.
- **No difference** — real information too. It means prompt bloat was not the bottleneck, and the competing hypothesis deserves attention: the codebase went from 91 Go files to 673, so the tasks themselves got harder.

Inspect any worker prompt without side effects:

```bash
aether build <n> --print-brief
aether build <n> --print-brief --worker Mason-67
```

---

## Open decisions

**1. The 67 hive entries are still in `~/.aether/hive/wisdom.json`.** Deliberately not cleared — that is the owner's call. They cannot be repaired (see below), only replaced by entries a corrected generator produces. Clearing them is safe: hive retrieval is off by default, so nothing reads them today.

**2. Nothing is pushed.** All work is local.

**3. The v1.25 milestone is written but unapproved** — 12 phases, 88 requirements, in `.planning/ROADMAP.md` and `.planning/REQUIREMENTS.md`. The judgement at the end of the session was that most of it dissolves if the prompt fix works. The parts worth doing regardless of the measurement:
   - `check-antipattern` is called with a positional argument the CLI rejects — **the Gatekeeper security gate has never run**
   - `consolidation-phase-end --dry-run` and `consolidation-seal --dry-run` both **mutate `instincts.json`** despite the flag reading "Report without modifying"
   - re-init after seal wipes phases, instincts and decisions into an unadvertised `.bak` with no restore command
   - `swarm-cleanup` reports `cleaned: true` while leaving files on disk

**4. The memory redesign spec** is in `.planning/REQUIREMENTS.md` under Phase A′ if it gets picked up: build durable memory on handoff records (`.aether/data/handoffs/worker-handoffs.json`), which are already file-anchored, dated and falsifiable. Promote at three independent confirmations across three phases. Decay on usage, not elapsed time.

---

## What changed, and why

### Worker prompts

Removed from every worker brief: playbook injection (5,733 chars of orchestrator guidance), the heartbeat protocol (asked a model to write a file "every 30 seconds" during its own turn — a model has no timer and never could), and read-cache discipline (both harnesses already tell the model when a file is unchanged).

Measured on the same phase: **7,485 → 688 characters**, every remaining section task-relevant.

The Go fix alone did nothing for `/ant-build`, because that wrapper calls `aether host build --dry-run`, which runs the TypeScript host, which re-glued the playbooks back on at three sites. Both paths are now fixed.

### Memory admissibility

`pkg/memory/admissible.go` is new. Content must name a **file, command or error** before it can become durable memory. Prose naming none of those cannot be checked against a repository later, so it can never be invalidated — and memory that cannot be invalidated accumulates forever. Every one of the 67 hub entries fails this test.

Also closed: the tautological `"When %s, apply observed pattern"` template (trigger and action were the same sentence), 100-char truncation that cut mid-word, the `max(0.2, …)` trust floor that meant nothing could ever expire, path-prefix stripping that pointed entries at files which do not exist, and the missing sanitiser on hive text — the one cross-colony channel without one.

### Hive trust model

The other session's design was sound and is preserved: revocation, contradiction quarantine, lazy read-time decay, stable repository identity. It had **zero tests**; `cmd/hive_trust_test.go` now covers all of it.

Retrieval stays **off by default** behind two gates — machine policy (`AETHER_HIVE_POLICY`) and per-colony consent (`aether hive-opt-in`). That is the documented intent, not an evasion.

---

## Gotchas for whoever picks this up

- **`aether publish` builds from the working tree, not committed code.** There is no clean-tree guard. Always `git status --porcelain` first. This bit twice in one session.
- **`~/.aether/` is not version-controlled and has no rollback.** Back it up before publishing: `cp -R ~/.aether ~/.aether-backup-$(date +%Y%m%d-%H%M)`. Existing backup: `~/.aether-backup-20260725-2328`.
- **`aether host build --dry-run` is NOT read-only** despite the name — it runs gates and opens a build attempt. Use `--print-brief` for inspection.
- **`aether update` unconditionally creates `.aether/ts-host/` and runs `npm ci`** in whatever repo it runs in, regardless of `--force`. Not needed for this change.
- **`TestMain` runs `git branch -D` on branches matching `phase-*` and two-segment `feature/*`.** Do not name a working branch that way.
- **Three tests are order-dependent** and pass in isolation but can fail in a full run: `TestCLIProviderBackedPlanRevisionJourney`, `TestCLICompiledInstallToSealJourney`, `TestCLICompiledInstallUpdateMigrationContract`. Pre-existing, unrelated to this work.
- **`.claude/agents/` and `.claude/commands/` are global**, not per-repo. Other projects have no local copies; they read `~/.claude/`. `aether publish` writes both.

## Rollback

| If | Then |
|---|---|
| The hub goes bad | `rm -rf ~/.aether && cp -R ~/.aether-backup-20260725-2328 ~/.aether` |
| The binary misbehaves | `cp ~/.local/bin/aether.pre-prompt-fix ~/.local/bin/aether` |
| The hive work goes wrong | `git revert 101d03d9` — the prompt fix is in separate earlier commits and unaffected |

## Verify everything still works

```bash
aether version --check                                                # ok:true, both 1.0.42
grep -c renderPlaybookContext ~/.aether/system/ts-host/dist/host.js   # 0
aether build --help | grep -c -- --print-brief                        # 2
cd "/Users/callumcowie/Documents/Max 9/M4L-AnalogWave-System" && aether status
```

---

## Background

The full investigation — eleven review agents, what was refuted, and the evidence behind each decision — is at
**https://claude.ai/code/artifact/90bc7fe7-d6e2-4e91-8936-8b3a26014e27**

Worth reading before re-planning anything, because it records several confident conclusions that turned out to be wrong, and why.
