## 5. Natural Commit Points in Aether's Colony Lifecycle

### 5.1 Lifecycle Diagram

```
USER ACTION          COLONY STATE          STATE CHANGES               NATURAL COMMIT POINT?
─────────────────────────────────────────────────────────────────────────────────────────────

/ant:init "goal"     ─── INIT ───────────  .aether/data/COLONY_STATE.json created    [A] POST-INIT
                         │                 .aether/data/constraints.json created
                         │                 State: null → READY
                         ▼
/ant:plan            ─── PLAN ───────────  .aether/data/COLONY_STATE.json updated    [B] POST-PLAN
                         │  ↻ loop        .aether/data/watch-status.txt written
                         │  (research +   .aether/data/watch-progress.txt written
                         │   planning     State: READY → PLANNING → READY
                         │   iterations)  plan.phases populated
                         ▼
/ant:build N         ─── BUILD ──────────  git add -A && git commit (checkpoint)     [C] PRE-BUILD (existing)
                         │                 COLONY_STATE: state → EXECUTING
                         │                 Workers spawn, create/modify USER CODE
                         │                 .aether/data/spawn-tree.txt updated
                         │                 .aether/data/activity.log appended
                         │                 State: READY → EXECUTING
                         ▼                                                           [D] POST-BUILD
                                          (user code files created/modified)
                                          (test files created)

/ant:continue        ─── CONTINUE ───────  Verification loop runs (6 phases)         [E] POST-VERIFY
                         │                 Spawn gate checked
                         │                 Anti-pattern gate checked
                         │                 TDD gate checked
                         │                 Runtime verification (user prompt)
                         │                 Flags gate checked
                         │                 COLONY_STATE: phase marked completed       [F] POST-ADVANCE
                         │                 COLONY_STATE: current_phase incremented
                         │                 memory.phase_learnings appended
                         │                 memory.instincts updated
                         │                 CHANGELOG.md appended
                         │                 State: EXECUTING → READY
                         ▼
                     ─── (next phase or COMPLETE) ──────────────                     [G] PROJECT-COMPLETE
                         │                 .aether/data/completion-report.md written
                         ▼

/ant:swarm "bug"     ─── SWARM ──────────  git stash (autofix-checkpoint)            [H] PRE-SWARM (existing)
                         │                 4 scouts spawn, investigate
                         │                 Code changes applied
                         │                 Rollback if fix fails
                         ▼                                                           [I] POST-SWARM-FIX

/ant:pause-colony    ─── PAUSE ──────────  .aether/HANDOFF.md written                [J] SESSION-PAUSE

/ant:resume-colony   ─── RESUME ─────────  (read-only, no state mutations)           [K] SESSION-RESUME

/clear               ─── SESSION CLEAR ──  (no state changes; persistence via files) [L] PRE-CLEAR
```

### 5.2 Assessment of Each Commit Point

Each point is scored 1-5 across four dimensions:

| Dimension | Description | 1 (low) | 5 (high) |
|-----------|-------------|---------|----------|
| **Value** | How valuable is this snapshot for the user's git history? | Trivial / internal-only state | Major feature milestone; user would want this in history |
| **Risk** | What's lost if we don't commit here and something goes wrong? | Nothing meaningful; easy to recreate | Significant user code; hard to reconstruct |
| **Disruption** | How disruptive would a commit prompt be at this moment? | Seamless; user is already pausing | Breaks concentration; mid-flow interruption |
| **Clarity** | How clear/meaningful would a commit message be? | Vague; "stuff changed" | Precise; maps to a deliverable or decision |

**Composite Score:** `Value * (6 - Disruption)` — higher is better. This rewards high-value points that are low-disruption.

| # | Point | What Changed | Value | Risk | Disruption | Clarity | Composite | Classification |
|---|-------|-------------|-------|------|------------|---------|-----------|----------------|
| A | POST-INIT | `.aether/data/` files (COLONY_STATE, constraints) | 1 | 1 | 2 | 2 | 4 | Safety |
| B | POST-PLAN | `.aether/data/` files (plan phases populated) | 2 | 2 | 2 | 3 | 8 | Progress |
| C | PRE-BUILD | Everything (git add -A) — existing checkpoint | 3 | 4 | 1 | 3 | 15 | Safety |
| D | POST-BUILD | **User code + tests + .aether state** | 4 | 5 | 3 | 4 | 12 | Progress |
| E | POST-VERIFY | Same as D but now verified by Watcher | 4 | 5 | 2 | 4 | 16 | Progress |
| **F** | **POST-ADVANCE** | **User code + verified + learnings + CHANGELOG** | **5** | **5** | **1** | **5** | **25** | **Milestone** |
| G | PROJECT-COMPLETE | Completion report + all accumulated work | 5 | 5 | 1 | 5 | 25 | Milestone |
| H | PRE-SWARM | Stash (existing checkpoint) | 2 | 4 | 1 | 2 | 10 | Safety |
| I | POST-SWARM-FIX | Bug fix code changes | 3 | 4 | 2 | 4 | 12 | Progress |
| J | SESSION-PAUSE | .aether/HANDOFF.md + any accumulated state | 3 | 3 | 1 | 3 | 15 | Safety |
| K | SESSION-RESUME | (nothing changed; read-only) | 0 | 0 | 3 | 1 | 0 | None |
| L | PRE-CLEAR | Whatever is unsaved at that moment | 2 | 3 | 2 | 2 | 8 | Safety |

### 5.3 Detailed Assessment Rationale

#### [A] POST-INIT — Value: 1, Risk: 1, Disruption: 2, Clarity: 2

**What changed:** Only `.aether/data/COLONY_STATE.json` (fresh skeleton with goal) and `.aether/data/constraints.json` (empty). No user code touched.

**Value is low** because the state file is trivially regenerable by running `/ant:init` again. No intellectual work product exists yet.

**Risk is low** because losing this state means re-running one command with the same goal string. Cost: seconds.

**Disruption is low** because the user just typed a command and is seeing the output banner. They're in a "setup" mental mode, not a flow state.

**Clarity is moderate-low** because "Initialize colony with goal X" is meaningful but trivial. It's a glorified config write.

**Classification: Safety.** If committed at all, this protects the initial state but has almost no user value.

#### [B] POST-PLAN — Value: 2, Risk: 2, Disruption: 2, Clarity: 3

**What changed:** `COLONY_STATE.json` now has `plan.phases` populated with the full project plan (phases, tasks, success criteria, confidence scores). Watch files updated. Activity logs written.

**Value is moderate-low** because the plan is the product of AI iteration and can be regenerated (though it takes time and the exact plan won't be identical). The plan itself lives in `.aether/data/` and doesn't represent user-authored code.

**Risk is moderate-low** because `/ant:plan` can regenerate a plan. However, if the user spent 10-20 minutes in the planning loop providing input at stall points, losing that iteration is annoying.

**Disruption is low** because the plan display ends with "Safe to /clear, then run /ant:build" — the user is explicitly at a pause point between planning and building.

**Clarity is moderate** because "Generate N-phase plan for X" is clear but doesn't represent a code deliverable.

**Classification: Progress.** Records AI-generated planning work. Opt-in at most.

#### [C] PRE-BUILD — Value: 3, Risk: 4, Disruption: 1, Clarity: 3

**What changed:** This is the EXISTING checkpoint. It runs `git add -A && git commit --allow-empty` before any build work begins. It captures the state of the entire working tree as it was before the colony's workers start modifying files.

**Value is moderate** because this snapshot is the "last known good" state. If the build corrupts something, this is the rollback target. It captures user code that existed before the build, which may represent hours of prior work.

**Risk is high** because without this checkpoint, a failed build has no rollback target. Workers may modify dozens of files. The `--allow-empty` means it always creates a restore point even for incremental rebuilds.

**Disruption is very low** because the commit happens automatically as part of `/ant:build` startup. The user invoked the build and expects "things to happen." A checkpoint is invisible infrastructure.

**Clarity is moderate** because "aether-checkpoint: pre-phase-N" is descriptive for its purpose (rollback) but isn't meaningful as a project history entry. It's operational, not narrative.

**Classification: Safety.** This is the textbook safety commit. Task 1.4 correctly identified that implicit consent is acceptable here. However, per R1 from 1.4, `git stash --include-untracked` would be less intrusive.

#### [D] POST-BUILD — Value: 4, Risk: 5, Disruption: 3, Clarity: 4

**What changed:** This is the moment after `/ant:build` completes (Step 7 output displayed). Workers have created/modified user code files, written tests, updated `.aether/data/` state. The user sees the build summary but has NOT yet verified the work.

**Value is high** because real code now exists — potentially dozens of files across the project. This represents substantial AI work product.

**Risk is very high** because this code exists only in the working tree. A `git checkout .` or session crash could lose everything. No checkpoint has been created AFTER the build work (only the PRE-build checkpoint exists).

**Disruption is moderate** because the user is reading the build summary and deciding what to do next. They're in an evaluative state, but they may want to immediately run `/ant:continue` or test the code. A commit prompt here interrupts the natural flow of "build -> verify -> advance."

**Clarity is good** because "Phase N: {name} — build complete (pending verification)" clearly describes the state.

**Classification: Progress.** This is unverified AI-generated code. Per Task 1.4's classification, this should be opt-in only. The user hasn't reviewed or tested the code yet.

#### [E] POST-VERIFY — Value: 4, Risk: 5, Disruption: 2, Clarity: 4

**What changed:** The verification loop has passed (build, types, lint, tests, security, diff review all green). The Watcher independently confirmed the work. Anti-pattern gate passed. TDD gate passed. The code is now machine-verified but not yet user-confirmed at runtime.

**Value is high** because the code has passed automated verification. This is a meaningful quality gate.

**Risk is very high** because all the work from POST-BUILD still exists only in the working tree, and now we additionally know it's verified-good. Losing this state means re-building AND re-verifying.

**Disruption is low** because the user is about to be prompted for runtime verification (Step 1.9). They're already in an interactive, evaluative mode. The verification report is displayed and the user is pausing to review it.

**Clarity is good** because "Phase N: {name} — all verification gates passed" is precise and evidence-backed.

**Classification: Progress.** Close to Milestone, but the user hasn't confirmed runtime behavior yet.

#### [F] POST-ADVANCE — Value: 5, Risk: 5, Disruption: 1, Clarity: 5 (STRONGEST)

**What changed:** The user has confirmed runtime verification (Step 1.9: "Yes, tested and working"). All gates passed. The phase is now marked completed in COLONY_STATE.json. Learnings extracted. Instincts updated. CHANGELOG.md appended. Phase counter advanced.

**Value is maximum** because this is a verified, user-approved milestone. The code works, the tests pass, the user confirmed it runs. The state includes both the deliverable (user code) and the metadata (learnings, changelog). This is the closest analog to what a human developer would naturally commit: "Feature X done and tested."

**Risk is maximum** because losing this state means losing verified work AND the learnings/instincts that inform future phases. The state transition (EXECUTING -> READY) is meaningful; reverting it requires a full rebuild + re-verify cycle.

**Disruption is minimal** because the `continue` command ends with "State persisted — safe to /clear, then run /ant:build {next}". The user is explicitly at a boundary between phases. They expect to pause here. A commit prompt ("Commit Phase N?") is completely natural.

**Clarity is maximum** because the commit message writes itself: "Phase N: {name} — {one-line summary from CHANGELOG entry}". This follows the user's git rules (imperative mood, under 72 chars, focused on "why"). The evidence supporting the commit is abundant (verification results, test counts, success criteria).

**Classification: Milestone.** This is the canonical milestone commit. It maps 1:1 to a completed, verified, user-approved unit of work.

#### [G] PROJECT-COMPLETE — Value: 5, Risk: 5, Disruption: 1, Clarity: 5

**What changed:** All phases completed. Completion report written. Colony learnings summarized. This is the terminal state.

**Value is maximum** because the entire project goal has been achieved. The completion report is a one-time artifact.

**Risk is maximum** because this represents all accumulated work. However, if POST-ADVANCE commits were made for each phase, the incremental risk here is lower (only the final phase's advance + completion report).

**Disruption is minimal** because the project is done. The user sees the celebration banner and is in a natural "wrap up" state.

**Clarity is maximum** because "Complete: {goal}" is as clear as it gets.

**Classification: Milestone.** However, if each phase was committed at POST-ADVANCE, this is effectively the same as the last phase's POST-ADVANCE with an added completion report. May be redundant.

#### [H] PRE-SWARM — Value: 2, Risk: 4, Disruption: 1, Clarity: 2

**What changed:** `autofix-checkpoint` creates a git stash (already the preferred mechanism per Task 1.4). This captures the state before the swarm attempts fixes.

**Value is low-moderate** because the current state may include broken code (the user is invoking swarm because something is wrong). Snapshotting broken code has limited narrative value.

**Risk is high** because the swarm will apply code changes. If the fix makes things worse, this stash is the rollback target. Without it, the user is stuck with a bad fix on top of a bug.

**Disruption is minimal** because it happens automatically as part of `/ant:swarm` startup. Like PRE-BUILD, it's invisible infrastructure.

**Clarity is low** because "pre-swarm checkpoint" is operational, not narrative.

**Classification: Safety.** Already correctly implemented as a stash.

#### [I] POST-SWARM-FIX — Value: 3, Risk: 4, Disruption: 2, Clarity: 4

**What changed:** The swarm applied a fix and it passed verification (build + tests). Code files modified with the bug fix.

**Value is moderate** because the fix is verified working, but it will be subsumed by the next POST-ADVANCE commit when the phase eventually completes.

**Risk is high** because the fix exists only in the working tree. If the user closes the session, the fix persists on disk but is uncommitted.

**Disruption is low** because the swarm just reported success and suggests `/ant:continue` as next step. The user is at a natural pause.

**Clarity is good** because "Fix: {bug description}" is clear and follows conventional commit conventions.

**Classification: Progress.** A verified fix is valuable, but it's an incremental step toward phase completion.

#### [J] SESSION-PAUSE — Value: 3, Risk: 3, Disruption: 1, Clarity: 3

**What changed:** `.aether/HANDOFF.md` written with session context. This is explicitly a "save your work" moment.

**Value is moderate** because the handoff document itself is ephemeral (it's overwritten on next pause). The real value is in ensuring any uncommitted user code changes are captured.

**Risk is moderate** because the user explicitly signaled "I'm leaving." If they don't come back to this exact session/machine, uncommitted changes could be lost. The state files persist on disk, but user code changes may not be pushed anywhere.

**Disruption is minimal** because the user is actively pausing. They expect "cleanup" activities. A commit prompt is perfectly natural here: "Commit current work before pausing?"

**Clarity is moderate** because "Session pause: Phase N in progress" is clear enough but isn't tied to a specific deliverable.

**Classification: Safety.** This is a session boundary commit. Its purpose is protecting work in transit, not recording a milestone.

#### [K] SESSION-RESUME — Not a commit point

**What changed:** Nothing. `/ant:resume-colony` is read-only. It reads HANDOFF.md and COLONY_STATE.json, displays context, and suggests next actions. No files are written or modified.

**Classification: None.** No state changes means no commit value.

#### [L] PRE-CLEAR — Value: 2, Risk: 3, Disruption: 2, Clarity: 2

**What changed:** This is a hypothetical commit point before the user runs `/clear`. Currently, `/clear` is a session-level operation with no Aether hook — there is no way for Aether to intercept it.

**Value is low-moderate** because the state at `/clear` time is arbitrary. The user might be mid-thought, mid-build, or at a clean boundary.

**Risk is moderate** because `/clear` destroys the conversation context but NOT the files on disk. State files and code changes persist. The risk is mainly to unsaved in-flight decisions that exist only in the conversation.

**Disruption is low-moderate** because the user chose to clear. However, Aether cannot currently intercept `/clear`, so any commit prompt would need to be advisory ("remember to commit before clearing") rather than interactive.

**Clarity is low** because the state at clear-time is unpredictable.

**Classification: Safety.** Advisory only — Aether cannot intercept `/clear`.

### 5.4 Ranked Recommendations

Sorted by composite score (Value * (6 - Disruption)):

| Rank | Point | Composite | Classification | Recommendation |
|------|-------|-----------|----------------|----------------|
| **1** | **F: POST-ADVANCE** | **25** | **Milestone** | **Prompt user for commit. This is THE commit point.** |
| **2** | **G: PROJECT-COMPLETE** | **25** | **Milestone** | Prompt user for commit (may be redundant with F if per-phase commits are made). |
| 3 | E: POST-VERIFY | 16 | Progress | Optional. Offer only if user has opted into progress commits. |
| 4 | C: PRE-BUILD | 15 | Safety | Keep as-is but downgrade to `git stash --include-untracked`. |
| 5 | J: SESSION-PAUSE | 15 | Safety | Add commit prompt: "Commit work before pausing?" |
| 6 | D: POST-BUILD | 12 | Progress | Do NOT auto-commit. Unverified AI code. |
| 7 | I: POST-SWARM-FIX | 12 | Progress | Optional progress commit after verified fix. |
| 8 | H: PRE-SWARM | 10 | Safety | Keep as-is (already uses stash). |
| 9 | B: POST-PLAN | 8 | Progress | Low priority. Plan is regenerable. |
| 10 | L: PRE-CLEAR | 8 | Safety | Advisory only (cannot intercept). |
| 11 | A: POST-INIT | 4 | Safety | Not worth committing. Trivially regenerable. |
| 12 | K: SESSION-RESUME | 0 | None | Not a commit point. |

### 5.5 The STRONGEST Commit Point: POST-ADVANCE (F)

**POST-ADVANCE is the single strongest natural commit point in Aether's colony lifecycle.**

Rationale across all four dimensions:

1. **Value (5/5):** This is the moment where a verified, user-approved phase becomes a permanent part of the project. The code works (tests pass), the user confirmed it runs (runtime verification gate), the quality was independently validated (Watcher), and the colony has extracted learnings. The CHANGELOG has been updated. This is the atomic unit of "done" in the colony model.

2. **Risk (5/5):** Without a commit here, the verified phase exists only as uncommitted files on disk. The state transition is complex (phase marked complete, learnings extracted, instincts updated, phase counter advanced) and expensive to reconstruct. If a subsequent build phase fails or the user's machine crashes, this verified work could be lost or muddled with later changes.

3. **Disruption (1/5):** The `continue` command explicitly tells the user "State persisted — safe to /clear, then run /ant:build {next}." This is a designed pause point. The user is transitioning between phases. They expect to stop, review, and then move on. A commit prompt here is like asking "Save your game?" at a save point — it's expected, welcome, and takes 2 seconds.

4. **Clarity (5/5):** The commit message is directly derivable from the phase data:
   - Pattern: `Phase {id}: {name} — {summary}`
   - Example: `Phase 2: Authentication — Add JWT auth with refresh tokens`
   - This perfectly follows the user's git rules: imperative mood, under 72 chars, focused on "why."
   - The commit body could include: success criteria met, test count, verification evidence.

**Why POST-ADVANCE beats POST-BUILD or POST-VERIFY:**
- POST-BUILD has unverified code (Disruption: 3, user hasn't reviewed)
- POST-VERIFY has machine-verified but not user-confirmed code
- POST-ADVANCE has the user's explicit "Yes, tested and working" stamp
- POST-ADVANCE also includes extracted learnings and CHANGELOG, making it a richer, more complete snapshot

**Why POST-ADVANCE beats PRE-BUILD:**
- PRE-BUILD is a safety checkpoint (backward-looking, preserving old state)
- POST-ADVANCE is a milestone (forward-looking, recording new achievement)
- Users want their git history to tell a story of progress, not a story of "about to do something"

**Implementation at POST-ADVANCE:**

The prompt should appear in `continue.md` Step 3 (Display Result), after the phase advancement display and before the "Next Steps" section:

```
Commit Phase {id}?
  1. Yes — "Phase {id}: {name} — {summary}"
  2. Yes — custom message
  3. No — I'll commit later
```

If the user selects (1), run:
```bash
git add -A && git commit -m "Phase {id}: {name} — {summary}"
```

If (2), let them type a message. If (3), skip silently.

This respects the user's git rules ("do not commit unless explicitly asked") because the user explicitly says "yes."

### 5.6 Classification Summary

```
SAFETY COMMITS (protect existing work, implicit consent OK):
  C: PRE-BUILD ........... git stash --include-untracked (downgrade from commit)
  H: PRE-SWARM ........... git stash (already correct)
  J: SESSION-PAUSE ....... prompt for commit (natural pause)
  L: PRE-CLEAR ........... advisory only (cannot intercept)
  A: POST-INIT ........... skip (trivially regenerable)

PROGRESS COMMITS (record AI work, opt-in only):
  D: POST-BUILD .......... do not auto-commit; unverified
  E: POST-VERIFY ......... optional; offer if user opts in
  I: POST-SWARM-FIX ...... optional; verified fix
  B: POST-PLAN ........... low priority; plan is regenerable

MILESTONE COMMITS (verified + approved, prompted):
  F: POST-ADVANCE ........ THE primary commit point (prompt user)
  G: PROJECT-COMPLETE .... final commit (prompt user)

NOT A COMMIT POINT:
  K: SESSION-RESUME ...... read-only, no state changes
```
