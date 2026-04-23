## Use Case: From Zero to Shipped

You have a blank directory and an idea: a REST API for a task management app. Users can create accounts, manage projects, and track tasks. Nothing fancy -- just solid, tested, shipped.

Here is what it looks like to build it with Aether, start to finish.

---

### Step 0 -- Install Aether

```bash
go install github.com/calcosmic/Aether@latest
aether install        # Populate the colony hub
aether setup          # Sync companion files to local repo
```

One-time setup. The colony hub lives at `~/.aether/` and persists across every project you ever work on. Companion files (agent definitions, commands, skills) land in `.aether/` inside your repo.

---

### Step 1 -- Lay Eggs

```bash
cd ~/projects/task-api
/ant-lay-eggs
```

```
Colony nest initialized.
  .aether/            Colony files (repo-local)
  .aether/commands/   45 slash commands
  .aether/agents-claude/   Agent definitions
  .aether/skills/     28 skills loaded
  .aether/data/       Colony state (local only)
```

This creates the nest -- the directory structure the colony needs to operate. You only run this once per repo. Think of it as setting up an ant farm before the colony moves in.

---

### Step 2 -- State the Goal

```bash
/ant-init "Build a REST API for task management with user auth, project CRUD, and task tracking"
```

```
Colony initialized.
  Goal: Build a REST API for task management with user auth, project CRUD, and task tracking
  Phase: 0/0
  Workers: standing by
```

The colony now has a purpose. Every worker that spawns from this point knows the goal. No repeating yourself in prompts, no copy-pasting context into new threads.

---

### Step 3 -- Colonize (Optional, but Smart)

```bash
/ant-colonize
```

```
Colonizing...
  Scanning codebase...
  Found: 0 files (new project)
  Language: Go (detected from go.mod)
  Framework: None detected
  Dependencies: None
  Structure: Empty project

Recommendation: New project. Route Setter should generate a full phased plan.
```

Since this is a new project, there is not much to map. But if you were adding Aether to an existing codebase, this is where the Colonizer ant would catalog your structure, identify patterns, and flag potential hazards before work begins.

---

### Step 4 -- Plan

```bash
/ant-plan
```

```
Generating phased roadmap...

Phase 1: Foundation
  - Project scaffolding (Go module, directory structure)
  - Database setup and connection layer
  - User model and migration

Phase 2: Authentication
  - JWT token generation and validation
  - Login and registration endpoints
  - Auth middleware

Phase 3: Projects
  - Project model and CRUD endpoints
  - User-project association
  - Input validation

Phase 4: Tasks
  - Task model and CRUD endpoints
  - Task-project association
  - Status workflow (todo, in-progress, done)

Phase 5: Integration and Polish
  - End-to-end testing
  - Error handling consistency
  - API documentation
```

The Route Setter ant analyzed the goal and broke it into phases. Each phase has clear deliverables. The colony will not jump ahead -- it builds phase 1 first, verifies it works, then moves on.

You can adjust the plan. Add phases, remove them, merge them. The colony follows your lead.

---

### Step 5 -- Steer with Pheromones

Before the first build, you set the guardrails:

```bash
/ant-focus "database migrations -- use versioned migrations, no schema drift"
/ant-redirect "No raw SQL in application code -- use parameterized queries only"
/ant-feedback "Prefer standard library where possible -- minimize dependencies"
```

```
Signal emitted: FOCUS "database migrations -- use versioned migrations, no schema drift"
Signal emitted: REDIRECT "No raw SQL in application code -- use parameterized queries only"
Signal emitted: FEEDBACK "Prefer standard library where possible -- minimize dependencies"

Active signals: 3
  FOCUS:    1
  REDIRECT: 1
  FEEDBACK: 1
```

These signals are visible to every worker in the next build wave. The FOCUS signal tells builders to be careful with migrations. The REDIRECT signal is a hard constraint -- no builder will write raw SQL. The FEEDBACK signal gently nudges toward minimal dependencies.

Signals expire at the end of the current phase. You can also set a wall-clock expiration with `--ttl 2h` if you want a signal to persist longer.

---

### Step 6 -- Build Phase 1

```bash
/ant-build 1
```

```
Deploying worker wave for Phase 1: Foundation...

Workers spawned:
  Chip-12 (Builder)     -> Project scaffolding
  Chip-34 (Builder)     -> Database setup
  Chip-56 (Probe)       -> Test scaffolding
  Dot-09 (Watcher)      -> Quality verification

Chip-12: Creating go.mod, directory structure...
Chip-34: Setting up database connection layer...
Chip-56: Writing initial test suite...
Dot-09: Monitoring build quality...

[Chip-12] go.mod created: module github.com/you/task-api
[Chip-34] db/connection.go created -- uses pgxpool
[Chip-34] db/migrations/001_create_users.up.sql created
[Chip-56] db/connection_test.go created
[Chip-56] models/user_test.go created

Build complete. 4 files created, 6 tests passing.
Phase 1 status: VERIFIED
```

Behind the scenes, multiple builder ants worked in parallel. A Probe ant wrote tests. A Watcher verified everything compiled and tests passed. The REDIRECT signal about raw SQL was active -- Chip-34 used `pgxpool` with parameterized queries instead.

---

### Step 7 -- Continue (Verify, Learn, Advance)

```bash
/ant-continue
```

```
Verifying Phase 1...
  Build: PASS
  Tests: 6/6 passing
  Coverage: 78%

Extracting learnings...
  [instinct] pgxpool connection pooling works well with this project structure (confidence: 0.85)
  [instinct] Versioned migrations prevent schema drift effectively (confidence: 0.90)

Advancing to Phase 2: Authentication
  Active signals: 2 (expired: 1 FOCUS -- end of phase)
```

The colony ran a six-point verification before advancing: build check, test pass, coverage gate, file existence, no regressions, and human review trigger. It also extracted instincts -- observations about what worked that will inform future builds.

The FOCUS signal about migrations expired because the phase ended. If you want it to persist into the next phase, emit it again.

---

### Step 8 -- Build Phase 2

You emit new signals for the auth phase:

```bash
/ant-focus "JWT token security -- use short expiry, secure refresh flow"
/ant-redirect "Never store passwords in plain text -- always bcrypt"
/ant-build 2
```

```
Deploying worker wave for Phase 2: Authentication...

Workers spawned:
  Chip-71 (Builder)     -> JWT token implementation
  Chip-88 (Builder)     -> Auth endpoints
  Chip-23 (Probe)       -> Auth test suite
  Dot-15 (Watcher)      -> Security verification

[Chip-71] internal/auth/jwt.go created -- RS256 signing, 15min access tokens
[Chip-88] handlers/auth.go created -- /register, /login endpoints
[Chip-88] middleware/auth.go created -- token validation middleware
[Chip-23] internal/auth/jwt_test.go created -- 12 tests
[Chip-23] handlers/auth_test.go created -- 8 tests
[Dot-15] Password hashing verified: bcrypt with cost 12
[Dot-15] No plaintext secrets detected

Build complete. 5 files created, 20 tests passing.
Phase 2 status: VERIFIED
```

The REDIRECT signal about bcrypt was active. The Watcher explicitly verified no plaintext secrets made it into the code. This is the colony self-enforcing your constraints without you having to review every line.

---

### Steps 9-12 -- Phases 3, 4, and 5

The pattern repeats. `/ant-build N`, then `/ant-continue`. Each phase builds on the verified output of the last. Instincts accumulate. The colony gets smarter about your project's patterns.

For phases 3 and 4, you decide to let autopilot handle it:

```bash
/ant-run --max-phases 2
```

```
Autopilot engaged. Max phases: 2.
Current phase: 3 (Projects)

[Phase 3: Projects]
  Spawning builders... Chip-42, Chip-67
  Spawning probe... Chip-91
  Spawning watcher... Dot-22
  Building... (4 files, 14 tests)
  Verifying... PASS
  Advancing to Phase 4.

[Phase 4: Tasks]
  Spawning builders... Chip-05, Chip-33
  Spawning probe... Chip-78
  Spawning watcher... Dot-31
  Building... (5 files, 18 tests)
  Verifying... PASS

Autopilot paused. Max phases reached (2/2).
Run /ant-run to continue, or /ant-continue to advance manually.
```

Autopilot ran two phases without you touching the keyboard. It paused gracefully at the limit instead of running off into the distance. You review what landed, emit any new signals if needed, and decide whether to continue.

---

### The Session Break

Now suppose you close your laptop. The next morning, you open Claude Code in the same directory. You have lost your conversation context. No problem.

```bash
/ant-resume
```

```
Previous colony session detected: "Build a REST API for task management..."

Restoring context...
  Phase: 4/5 complete
  Files created: 14
  Tests passing: 58/58
  Active instincts: 7
  Open flags: 0

Resume from Phase 5: Integration and Polish
  /ant-build 5 to continue
```

The colony reconstructed its full context from the colony state file, active signals, and accumulated instincts. You did not need to re-explain the project or paste your plan. The colony remembered.

---

### Step 13 -- Build Phase 5 (Final)

```bash
/ant-focus "API documentation -- generate OpenAPI spec from handlers"
/ant-build 5
```

```
Deploying worker wave for Phase 5: Integration and Polish...

Workers spawned:
  Chip-44 (Builder)     -> End-to-end test suite
  Chip-99 (Chronicler)  -> API documentation
  Dot-07 (Watcher)      -> Final quality gate

[Chip-44] tests/integration_test.go created -- 22 e2e tests
[Chip-99] docs/api.yaml created -- OpenAPI 3.0 spec
[Chip-99] docs/README.md created -- Getting started guide
[Dot-07] All endpoints documented
[Dot-07] Error responses consistent across handlers
[Dot-07] No hardcoded secrets found

Build complete. 3 files created, 80 tests passing.
Phase 5 status: VERIFIED
```

---

### Step 14 -- Seal the Colony

```bash
/ant-seal
```

```
Colony sealed.

Summary:
  Goal: Build a REST API for task management with user auth, project CRUD, and task tracking
  Phases completed: 5/5
  Files created: 17
  Tests passing: 80/80
  Instincts recorded: 9

Wisdom promotion:
  3 instincts promoted to QUEEN.md (score >= 0.80)
  0 instincts promoted to Hive Brain (score >= 0.90)

Colony status: CROWNED
```

The colony ran a final curation pass. High-confidence instincts were promoted to QUEEN.md -- your personal wisdom file that primes future colonies. The highest-scoring instincts also flowed into the Hive Brain, making them available to other projects on your machine.

---

### Step 15 -- Entomb

```bash
/ant-entomb
```

```
Colony entombed.
  Archive: .aether/chambers/task-api-2026-04-08/
  Colony state: cleared
  Wisdom: preserved in QUEEN.md and Hive Brain
```

The colony's work is archived. The directory is clean. But the knowledge persists -- next time you start a Go API project, the colony will already know that pgxpool works well, versioned migrations prevent drift, and bcrypt cost 12 is your standard.

---

### The Shortcut: Autopilot from Start

If you trust the plan and want hands-off execution, you can skip the manual loop entirely:

```bash
/ant-lay-eggs
/ant-init "Build a REST API for task management"
/ant-plan
/ant-focus "database migrations -- use versioned migrations"
/ant-redirect "No raw SQL in application code"
/ant-run
```

Autopilot runs every remaining phase, pausing only when something needs your attention -- a test failure, a security concern, a blocker it cannot resolve. Fix the issue, run `/ant-run` again, and it resumes.

That is five commands from zero to shipped. The colony handles the rest.
