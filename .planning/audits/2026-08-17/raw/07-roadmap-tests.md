# Raw report 07 — Planning-vs-Reality Divergence + Test/Proof Infrastructure

*Verbatim output of the "Audit roadmap, tests, divergence" investigation, 2026-08-17, branch `oracle-reinstate`. NOTE: claims A3's "v1.0.56 shipped outside planning" and "STATE.md is fiction" were RETRACTED/WEAKENED by the hostile review (see HOSTILE-REVIEW.md §B and raw/10) — the range check was a boundary artifact; v1.0.56 wrote ten decision records and roadmap rows. Read this report with that correction.*

## Task A — Planning-vs-reality divergence

### A1. What the hardening plan asks for
Source: .planning/HARDENING-PLAN.md (approved 2026-08-14, "go ahead, cut them all"). Trigger: the CalVault disaster — 3 workers burned 256,292 tokens in 12.5 minutes to copy 0 of 110 files (:38-49). Every phase REMOVES something: 180 internal references out of other projects (ROADMAP:830-838); 181 reading material by task relevance (:840-849); 182 specialists by what a phase changes (:851-859); 183 worker cap refuses before dispatch (:861-869); 184 one worker owns a chain (:871-878); 185 one honest cost line (:880-887); 175 plain-English dispatch explanation (:735-743); 179 Proof. Cut outright: 176, 177, 178, 174 plans 3-9 (:113-119). Finish line: "When those are true, we stop building" (:229).

### A2. v1.25 "Switch It On"
Source: v1.25-MILESTONE-AUDIT.md (2026-08-03). Premise: capability existed, never wired — "switch on and prove". 99 requirements; at audit 18 satisfied + 14 inserted-phase, 6 partial, 65 unsatisfied, 10 orphaned (:6, :53). Executed & verified: 163, 163.1, 163.2, 164, 165 (:80-89). Executed unverified: 160 (8 plans, no VERIFICATION.md — "exactly the failure mode this project's Definition of Done exists to prevent", :119). 161 descoped; 162 executed after the audit, 6/6 complete 2026-08-04. Never executed: 166-171 (41+ requirements). Superseded at 24% when v1.26 began. Damning process note: the milestone executed against a requirements revision "not yet approved by the user" (:57).

### A3. Do v1.0.54→57 map to planned phases? (SEE CORRECTION ABOVE)
- v1.0.54 (fe0bc982, 2026-08-15) — PLANNED: hardening phases 180-184 exactly as ordered (progress log :255-259; ROADMAP rows; commits f1cf2f40 → 47480c3d). No GSD phase directories for 180-185 — plan/summary/verification machinery skipped per the plan's own "turn the dials down"; named tests exist (verified TestAetherInternalReferencesStayOutOfOtherProjects, TestBuildWorkerCapHonoursVerificationDepth).
- v1.0.55 (6f0428c6, 2026-08-16) — PLANNED by a parallel system: SHIP-PROGRESS.md ("Ralph restoration loop"), incl. Phase 175 ("Plans: shipped outside the phase system, 2026-08-16, SHIP-PROGRESS Item 3" — ROADMAP:743). Also driven by field report 2026-08-16-init-obsidian-vault.md ("A unknown project"; "the CLI reported success while providing no value").
- v1.0.56 (26ec7834) — the original claim "NOT a planned phase anywhere" was WRONG in the strong form: the work wrote after-the-fact decision records into .planning/decisions/ (SEE-03, SEE-07, SEE-12-13, caste-emoji, autopilot-pause, dream-caste, init-preservation), retro-labelled against v1.25's deferred SEE-* requirements; 3fed5f3a "reconcile typed-mode ledger: work had shipped, record was missing"; ROADMAP Progress updated (8f6891ec).
- v1.0.57 (f56add75, 2026-08-17) — fork: zero commits in 26ec7834..f56add75 touch .planning/ (verified). ~13 feature commits trace to an Apple Notes ideas review.
- STATE.md frozen at 2026-08-14 ("Phase 180 is next") while ROADMAP shows 180-184+175 complete — stale (the maintained ledger moved to ROADMAP + decisions/). Genuinely planned-and-undone: Phase 185 (no "tokens (measured)" string in cmd/) and Phase 179.

### A4. Uncommitted working tree = v1.0.58 prep (hub-leak fix)
hubExcludeFiles map + three excluded dirs; two-sided exclusion (never copied AND stale leaked copies deleted on next publish); TestHubPublishExcludesPrivateColonyFiles (install_cmd_test.go:1007); .aether/registry.json untracked+ignored; version 1.0.58; honest CHANGELOG. Coherent, single-purpose. (Committed shortly after as c6bac0cb.)

## Task B — Test and proof infrastructure

### B1. Test suite shape
- 464 tracked *_test.go files (365 cmd/, 99 pkg/) vs 372 non-test .go files; 4,423 `func Test` functions (3,451 in cmd/).
- Wiring/invariant flavor: ~152 test functions (~3.4%) named wiring/orphan/ratchet/parity/golden/contract/caller/invariant — a floor; many invariant tests named behaviorally.
- **E2E with stubbed LLM: YES.** TestFullLifecycleInDownstreamRepo (cmd/e2e_lifecycle_test.go:148) runs init→plan→build→continue→seal→entomb in a temp downstream git repo with FakeInvoker (header :21-26). Compiled-binary blackbox harness (cmd/blackbox_harness_test.go): TestCLICompiledInstallToSealJourney (:993), TestCLIInterruptedBuildResumesThroughForceRedispatch (:608).
- Release validation: `aether integrity` (integrity_cmd.go:35). CI: full tests, race, goreleaser snapshot, binary smoke, packed-npm staged install, parity tests, catalog classification, named "Verify subcommand wiring and CLI flag contracts" step (ci.yml:99). release.yml gates on tests+race (:116,119) + TestStagedGoReleaserArtifactsInstallThroughPackedNPM (:154).

### B2. Benchmark/eval infrastructure: effectively NONE
Exactly 9 `func Benchmark` functions, all micro-performance (context assembly, stream multiplexer, terminal display). Nothing measures autonomous success rate, intervention count, token cost per outcome, or Aether-vs-baseline. Token capture exists at plumbing level (spend_session_capture.go, 174 plans 1-2) but no user-facing spend command and no cost line (no "tokens (measured)" string — Phase 185 unstarted).

### B3. Phase 179 "Proof" — promised vs existing
Promises (ROADMAP:891-902): three real tasks on a cheap model with interventions counted in a committed artifact; full downstream lifecycle with Aether's own git status clean; before/after tokens "measured by `aether spend`" recorded even if unfavourable; cold-session resume with only /ant-resume; a dated retune decision consuming the measurement. What exists: nothing. No phases/179-* directory. PROOF-01 unticked (SHIP-PROGRESS:31 left it for the operator). Criterion 3 names `aether spend`, a command that does not exist — written before 174's cut, never reconciled. 179 has survived two rescopes as "the finish line" while six releases shipped around it.

### B4. Honest verdict
Partially provable — and the partial is real, but stops exactly where it matters. Proven mechanically on every CI run: the state machine and CLI lifecycle end-to-end with stubbed workers, through the compiled binary, including interruption/resume and install-through-npm — by this project's history of phantom completions, a genuine achievement. NOT proven anywhere: the golden path with a real model doing real work. Every orchestration-quality claim is validated only against fixtures and fakes. The only real-world datapoints committed are failures (CalVault; the 2026-08-16 init walk-away). The one instrument designed to convert real usage into committed evidence — Phase 179 — is the oldest unstarted item on the roadmap. "A gate that gets ignored without anyone noticing is the failure mode this project keeps rediscovering" (STATE.md:109) — happening to the hardening plan itself: 185 and 179 not true, building did not stop. *(Hostile review nuance: 180-184+175 were executed under the plan; the deviation is one feature day jumping the declared order.)*
