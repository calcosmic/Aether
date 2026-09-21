# Aether Colony System

> This repo uses the Aether colony system for multi-agent development.
> These rules are auto-distributed by `aether update` — do not edit directly.

## Session Recovery

The program does this for you. Open a chat, resume one, or carry one on after
clearing it, and Aether prints a short card saying what the project is, how far
along it is, the one command to run next, and a couple of alternatives. A folder
with no project set up in it is not greeted at all.

Nothing is restored automatically — the card tells you what to run and you
decide. `/ant-resume` is the one return and recovery command. The Go runtime
validates a saved handoff or reconstructs the safest honest recovery point from
durable evidence, keeps confirmed and reconstructed provenance distinct, and
stops without changing runnable state when evidence conflicts.

## Available Commands

### Setup & Getting Started
| Command | Purpose |
|---------|---------|
| `/ant-lay-eggs` | Set up Aether in this repo (one-time, creates .aether/) |
| `/ant-init "<goal>"` | Start a colony with a goal |
| `/ant-colonize` | Analyze existing codebase |
| `/ant-plan` | Generate project phases |
| `/ant-build <phase>` | Execute a phase with parallel workers |
| `/ant-continue` | Verify work, extract learnings, advance |

### Pheromone Signals
| Command | Priority | Purpose |
|---------|----------|---------|
| `/ant-focus "<area>"` | normal | Guide colony attention |
| `/ant-redirect "<pattern>"` | high | Hard constraint — avoid this |
| `/ant-feedback "<note>"` | low | Gentle adjustment |
| `/ant-pheromones` | — | View all active signals |
| `/ant-export-signals` | — | Export signals to XML |
| `/ant-import-signals` | — | Import signals from XML |

### Status & Monitoring
| Command | Purpose |
|---------|---------|
| `/ant-status` | Colony dashboard |
| `/ant-phase [N]` | View phase details |
| `/ant-flags` | List active flags |
| `/ant-flag "<title>"` | Create a flag |
| `/ant-history` | Browse colony events |
| `/ant-watch` | Colony watch dashboard / compatibility view |
| `/ant-memory-details` | Drill-down memory view |
| `/ant-patrol` | System health check |
| `/ant-help` | List available commands |

### Session Management
| Command | Purpose |
|---------|---------|
| `/ant-pause` | Stop at a safe boundary and save one validated handoff and receipt |
| `/ant-resume` | Validate or reconstruct the recovery point and report its provenance |

Both are thin runtime commands. Wrappers never write colony, session, or
handoff state themselves. Pause and resume use the handoff ID as the
idempotency key, so a verified replay returns the existing receipt rather than
creating a second recovery effect.

### Lifecycle
| Command | Purpose |
|---------|---------|
| `/ant-seal` | Seal colony and retain active state for review |
| `/ant-entomb` | Optional explicit owner-invoked archive-and-clear alternative |
| `/ant-maturity` | View colony maturity journey |
| `/ant-update` | Update system files from hub |
| `/ant-migrate-state` | Migrate colony state between versions |

### Advanced
| Command | Purpose |
|---------|---------|
| `/ant-run` | Autopilot — build, verify, advance automatically |
| `/ant-improve` | Check how the colony's own suggestions have been doing, or try one by hand |
| `/ant-quick` | Quick one-shot task |
| `/ant-swarm "<bug>"` | Parallel bug investigation |
| `/ant-oracle` | Deep research (RALF loop) |
| `/ant-dream` | Philosophical observation |
| `/ant-interpret` | Review dreams, discuss actions |
| `/ant-chaos` | Resilience testing |
| `/ant-archaeology` | Git history analysis |
| `/ant-organize` | Codebase hygiene report |
| `/ant-council` | Intent clarification |
| `/ant-preferences` | Set user preferences |
| `/ant-skill-create` | Create a custom skill |
| `/ant-insert-phase` | Insert phase into plan |
| `/ant-tunnels` | View colony communication tunnels |
| `/ant-data-clean` | Clean test artifacts from data files |
| `/ant-verify-castes` | Verify worker caste assignments |

## Typical Workflow

```
First time in a repo:
0. /ant-lay-eggs                           (set up Aether in this repo)

Starting a colony:
1. /ant-init "Build feature X"             (start colony with a goal)
2. /ant-colonize                           (if existing code)
3. /ant-plan                               (generates phases)
4. /ant-focus "security"                   (optional guidance)
5. /ant-build 1                            (workers execute phase 1)
6. /ant-continue                           (verify, learn, advance)
7. /ant-build 2                            (repeat until complete)
   /ant-run                                (or use autopilot for all phases)

Before a planned session break:
8. /ant-pause                              (runtime stops at a safe boundary and saves a receipt)

After /clear or session break:
9. /ant-resume                             (validate or safely reconstruct with provenance)
10. /ant-status                            (see where you left off)

After completing a colony:
11. /ant-seal                              (seal and retain active state for review)
12. /ant-status                            (review the retained sealed state first)
13. /ant-entomb                            (optional explicit owner-invoked archive-and-clear alternative)
14. /ant-init "next project goal"          (only after a successful archive-and-clear receipt)
```

### Sealed colony: review before archive

After `/ant-seal`, whether it is verified or a forced-incomplete closure, the
active colony remains retained for review; a forced-incomplete seal is never
verified completion.

1. First run `/ant-status` to review the retained sealed state.
2. `/ant-entomb` is an optional, explicit owner-invoked archive-and-clear
   alternative; it is never automatic or required after sealing.
3. A forced-incomplete marker remains visible in `/ant-status` and optional
   `/ant-entomb`.
4. Only after a successful archive-and-clear receipt has verified the archive
   and cleared active state may you run `/ant-init` for a new goal.

## Worker Castes

Workers are assigned to castes based on task type:

| Caste | Role |
|-------|------|
| builder | Implementation work |
| watcher | Monitoring, quality checks |
| scout | Research, discovery |
| chaos | Edge case testing |
| oracle | Deep research (RALF loop) |
| architect | Planning, design |
| colonizer | Codebase exploration |
| route_setter | Phase planning |
| archaeologist | Git history analysis |

## Protected Paths

**Never modify these programmatically:**

| Path | Reason |
|------|--------|
| `.aether/data/` (except sanctioned scratch subpaths below) | Colony state (COLONY_STATE.json, session files) |
| `.aether/dreams/` | Dream journal entries |
| `.aether/checkpoints/` | Session checkpoints |
| `.aether/locks/` | File locks |

**Sanctioned scratch subpaths** — a worker following its own task brief may write here; `protectedHookWriteReason` in `cmd/hook_cmds.go` allows exactly these five directory segments and nothing else under `.aether/data/`:

| Path | A worker writes here when |
|------|---------------------------|
| `.aether/data/planning/` | Persisting planning artifacts during the plan workflow |
| `.aether/data/phase-research/` | A scout writes phase domain research (`renderPhaseResearchBrief`) |
| `.aether/data/survey/` | A surveyor writes territory survey artifacts |
| `.aether/data/worker-debug/` | Persisting worker debug artifacts for diagnostics |
| `.aether/data/territory-candidates/` | A surveyor stages a territory refresh; `colonize-finalize` promotes it into `survey/` |

## Colony State

State is stored in `.aether/data/COLONY_STATE.json` and includes:
- Colony goal and current phase
- Task breakdown and completion status
- Instincts (learned patterns with confidence scores)
- Pheromone signals (FOCUS/REDIRECT/FEEDBACK)
- Event history

## Pheromone System

Signals guide colony behavior without hard-coding instructions:
- **FOCUS** — attracts attention to an area (expires at phase end)
- **REDIRECT** — repels workers from a pattern (high priority, hard constraint)
- **FEEDBACK** — calibrates behavior based on observation (low priority)

Use FOCUS + REDIRECT before builds to steer. Use FEEDBACK after builds to adjust.

## Biological Runtime (v1.28, Phase 203)

A helper stuck on its task can ask the program for backup with the real
`aether recruit` command, and the program — never the assistant — decides
whether the request is granted, through the same one gate every ordinary
helper assignment already goes through. The check looks at how deep the
chain of asks has gone (capped at two hops), how many helpers the whole run
has already used, whether the new helper is allowed near what it wants to
touch, and whether someone is already doing that exact job.

Every dispatched helper is told the ability exists, on every lane and on all
three assistant platforms, from one shared source so the lanes cannot drift
apart. Two deliberate exceptions: the security reviewer and the quality
reviewer hold no shell at all, so they are never told to run any command.

A refusal never stops the work — the helper carries on and finishes the task
alone, and the command still reports success, not a failure. The owner never
approves a routine backup request; the program's own limits are the leash.
What the owner does see, live, in the one window they are already using: one
line when a helper joins, one line when a request is refused, and — once the
run ends — the whole family tree of who asked for backup, what each branch
cost, and every refusal along the way.

The program's own steering notes now get more trusted the more they actually
help and less trusted — or set aside — the more they don't, based on what a
note genuinely did afterward, never merely on whether a helper saw it. A note
the owner pinned in place is never moved by this automatic tuning.

None of this costs anything on an ordinary run that never asks for backup —
that has been measured, not just promised.

**One honest limit, left open rather than hidden:** the depth check trusts a
short, fixed list of coordinator names on its own word alone, with nothing
yet proving that a caller claiming one of those names really is the
coordinator. This gap predates this phase and is tracked, not silently
fixed, in `.planning/WINDOWS.md`. See CLAUDE.md's "Biological Runtime"
section for the full account and the tests that lock every claim above.

## Learning Governor (v1.28, Phase 204)

The evidence-gated outcome ledger the previous phase built now has a real
production writer, reached from both places the program checks its own
work, earning credit only when a real decision changed and a real effect
was measured afterward.

A lesson the program recorded but never checked is no longer shown to a
helper under a heading that calls it proven — one shared rule now decides,
everywhere, what counts as verified. A lesson recorded only as a guess is
meant to be promoted to genuinely verified automatically, at the end of
every check, on both check lanes, once the program's own records show it
truly helped, checked independently rather than taken on a helper's own
word — that gating rule is real, but the promotion itself cannot happen in
the running program yet: nothing today connects a recorded guess to the
proof that it helped, so this pass finds nothing to promote on any real
check, a confirmed, openly recorded gap (WINDOWS.md entry 44, reopened
2026-09-15), not a silent one.

Every remembered record now carries its own version and says where it came
from; an old record is read as the older shape it actually is, and every
field is either filled by something real or sits on a reason-carrying list
that may only shrink.

Every run the program does now leaves a permanent record of what it cost,
what it decided and how it ended — one that outlives the live activity
screen's own thirty-day memory.

Guidance now moves through nine tracked states, a helper's claim that it
used something is independently checked against what the program can
actually see for itself, and a lesson must have genuinely helped once
before it reaches the shared instruction file.

This project's own confirmed failures are now a versioned bank of
regression fixtures, each traceable to a real incident, either guarded by a
named check or on a counted list that may only shrink.

The test suite now runs as seven named, budgeted gates, each proving it ran
everything it found — closing the cause of three separate entries in the
project's own defect register.

A proposed change to settings or routing can be tried beside current
behaviour, graded by a judge it structurally cannot reach or edit, and a
change that only looks better on the work it could see is named as exactly
that. The judge is now a real grader, not a placeholder that always agreed,
and the whole sequence — compare, admit, and start a small, watched,
reversible trial — now runs by itself at the end of every check, on both
check lanes, with no command required. A new hand-run command, `aether
improve`, lets you check on its own how those suggestions have been doing,
without waiting for the next check; by itself it only reads and reports,
changing nothing on disk.

Two kinds of thing may be changed this way — remembered project facts, and
task routing — and nine may never be: preferences, skills, workflows,
source code, security settings, deletion, permissions, verification steps,
and external actions. The nine have no code path into the automatic route
at all.

Finally, how often the program's own suggestions genuinely helped and how
often a person had to step in are two separate figures that can never be
blended into one. And when the same reason for stepping in keeps recurring
— the owner intervening for the same reason on three or more separate runs
— the program now genuinely writes up that case itself, on its own isolated
branch, for a person to read; a proposed source-code change, automatic or
hand-typed, can only ever become an ordinary, reviewable change — never
something the program approves, merges, publishes, or deploys itself.

See CLAUDE.md's "Learning Governor" section for the full account and the
tests that lock every claim above.
