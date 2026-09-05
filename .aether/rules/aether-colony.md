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

**Sanctioned scratch subpaths** — a worker following its own task brief may write here; `protectedHookWriteReason` in `cmd/hook_cmds.go` allows exactly these four directory segments and nothing else under `.aether/data/`:

| Path | A worker writes here when |
|------|---------------------------|
| `.aether/data/planning/` | Persisting planning artifacts during the plan workflow |
| `.aether/data/phase-research/` | A scout writes phase domain research (`renderPhaseResearchBrief`) |
| `.aether/data/survey/` | A surveyor writes territory survey artifacts |
| `.aether/data/worker-debug/` | Persisting worker debug artifacts for diagnostics |

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
