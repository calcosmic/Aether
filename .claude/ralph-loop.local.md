---
active: true
iteration: 1
session_id: fa760d23-0970-4d2a-9a50-97cf20dcfdc0
max_iterations: 40
completion_promise: "AETHER-RESTORED-HARDENED"
started_at: "2026-08-16T15:18:27Z"
---

You are finishing Aether: a restoration and hardening pass, NOT feature development. The goal is the v5.4.0-era experience — a colony you can watch work — running on the modern Go runtime, with every lifecycle command provably working end-to-end.

BINDING LAW — NO NEW THINGS: every change must be one of (a) making an existing documented behaviour actually work, (b) surfacing data the runtime already computes but never shows, or (c) retiring dead weight per an existing spec. If a change is none of those three, do not make it. No cost/token/price features of any kind — Phase 185 is dropped by owner decision.

CONTEXT TO READ EVERY ROUND: .planning/ROADMAP.md (phase specs and success criteria), .planning/PROJECT.md lines 260-300 (unchecked requirements), .aether/docs/PARITY_CLASSIC_VS_GO.md (classic-vs-Go map — GAP and DEGRADED rows are your restoration targets), .planning/field-reports/2026-08-16-init-obsidian-vault.md. Consult classic behaviour directly with git show v5.4.0:<path> when unsure what 'good' looked like. Maintain .planning/SHIP-PROGRESS.md as your memory (git add -f): read it first, update it every round.

BINDING RULE from CLAUDE.md: a requirement is satisfied only when a command exists that someone can run, and that command FAILS when the requirement is unmet. This repo's history is 18 of 25 milestones re-restoring things previously marked done, and its core disease is machinery that exists but was never switched on. Verify by EXECUTION, never by grep. Every piece of wiring gets a named test that fails if the wiring is removed.

WORK ITEMS, strict priority order — ship each fully before starting the next, so hitting the round cap still leaves the most valuable work done:

1. LIFECYCLE PROOF (PROJECT.md WORKFLOW-01..09): for each of colonize, plan, build, continue, run, swarm, seal, entomb — audit whether executable end-to-end evidence exists (a test that runs the real command against a fixture repo and fails when the workflow is broken). Many already have coverage from phases 165/172; close only the gaps. Tick each WORKFLOW requirement in PROJECT.md ONLY with the name of the test that proves it. Also RUNTIME-01 (no stale decision/session leakage between colonies) and RUNTIME-02 (worker results are never silently lost) — both need a failing-test demonstration, then the fix.

2. INIT HARDENING (field report): knowledge-repo detection in cmd/init_research.go (markdown/source counting in the walk ~line 1890, .obsidian//.logseq/ detection, knowledge_base class in classifyDirs ~line 375 ONLY when zero languages detected; that class suppresses CI/LICENSE/README/formatter pheromones ~line 1331 and swaps risks to content-loss/broken-links ~line 1707); fix 'A unknown project' at ~line 1637 and the five 'No X detected' filler lines; low-signal branch in .aether/commands/init.yaml + all three init wrapper mirrors (the .opencode copy differs — edit by hand, never overwrite); durable-state marker .aether/WHAT-IS-THIS.md written by ensureRepoLocalScaffold (cmd/platform_sync.go ~line 765).

3. TIE IT TOGETHER (spirit of Phase 168 'Your Eyes Back' + Phase 175, per their ROADMAP.md specs): the runtime already computes colony vital signs (colony-vital-signs), next-step guidance (nextCommandFromState, closeoutNextCommand, continueNextCommandForAssessment), and per-caste dispatch rationale (carried in the dispatch manifest, never rendered). Wire these into the commands a user actually types: status/build/continue output shows colony health and the next suggested step; the Dispatch stage prints one plain-English clause per selected caste plus a SHORT clause (count or notable few, never a 27-row table) for castes considered and not called; when the runtime restores a safety caste or trims to the cap, the output names the caste and reason; the run summary distinguishes a worker that found something from one that came back clean. Reuse the computed strings — a render test must fail if the manifest rationale is removed. Do NOT reimplement any of it.

4. ORPHAN RECLAMATION (spirit of Phase 170, hardening only): pick the highest-value entries in cmd/testdata/orphan_allowlist.json that block the ratchet from tightening. For each: wire it to a real caller with a test, or retire it following the repo's existing retirement pattern, or record a dated disposition note. Never delete anything that is not already recorded as an orphan. Shrink the allowlist; never grow it.

5. RELEASE READINESS: bump the version (a republished version number proves nothing), update CHANGELOG.md, verify goreleaser check and aether integrity pass. DO NOT run aether publish or git push — shipping is the operator's call.

LANDMINES: gofmt -w touched files; go vet ./... clean; refresh goldens only when your change caused the mismatch (-update-golden on TestAuditCatalogGolden / TestRegressionSnapshot / TestPlatformParityGolden); some wrappers are SHA-256 pinned in cmd/lifecycle_wrapper_contract_test.go — update the hash in the same commit with the reason; new subcommands must be referenced from a wrapper or doc in the same commit or the reachability ratchet fails; TestBuildWorkerBriefIsMostlyTask is a proportion invariant — do not bloat briefs; wrapper mirrors must stay consistent but .opencode copies differ in content — edit, never blind-copy; do not touch .aether/data/, .aether/dreams/, or the oracle-reinstate work already on this branch.

DO NOT: add features, add any cost/token display, revive cut phases 174(03-09)/176/177/178, attempt Phase 179 (requires real downstream repos and the operator), publish, push, or weaken existing tests.

VERIFY before claiming done, all green in the SAME round: go vet ./... clean; go test ./... -count=1 fully green; goreleaser check passes; aether integrity passes; every claimed item in .planning/SHIP-PROGRESS.md names its proving test; PROJECT.md requirements ticked only where a named executable check exists. Commit in logical units, imperative messages under 72 chars. Only then output <promise>AETHER-RESTORED-HARDENED</promise>
