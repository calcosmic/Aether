# Phase 160: Fail Loudly - Pattern Map

**Mapped:** 2026-07-27
**Files analyzed:** 15 (8 new, 7 modified)
**Analogs found:** 15 / 15 (all have at least a partial match; none required inventing a pattern from scratch)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `cmd/policy_schema_test.go` | test (static/doc) | file-I/O (read+parse YAML, assert fields) | `control-ts/tests/schemas/policy.schema.test.ts` (spec to port) + `cmd/claudemd_verification_depth_test.go` (Go file-read-and-assert style) | role-match (cross-language port) |
| generalized command-call execution audit test | test (integration) | batch (enumerate + execute) | `cmd/security_gate_drift_test.go` (execution style) + `cmd/cli_flag_audit_test.go` (regex/enumeration style, has the exact gap to close) | exact (both explicitly named by RESEARCH.md as the two halves to merge) |
| invariant/count test for `2>/dev/null` in live wrapper dirs (LOUD-06) | test (invariant) | batch (grep-count) | `cmd/codex_build_test.go::TestBuildWorkerBriefIsMostlyTask` (proportion/invariant style) | role-match (style exemplar, not a grep test itself) |
| static-doc test for LOUD-07 doc claims | test (static/doc) | file-I/O (read + string assertions) | `cmd/claudemd_verification_depth_test.go` | exact |
| live-path gate test for antipattern gate (LOUD-02) | test (integration) | request-response (gate check invocation) | `cmd/gate.go` producer functions `checkNoCriticalFlags`/`checkAllTasksCompleted` (no dedicated test file for these; see below) + `cmd/security_gate_drift_test.go` (fixture/setup pattern for antipattern scanning) | role-match |
| `.aether/commands/unblock.yaml` | config (command source) | request-response | `.aether/commands/preferences.yaml` | exact |
| `.claude/commands/ant/unblock.md` | config (wrapper) | request-response | `.claude/commands/ant/preferences.md` | exact |
| `.opencode/commands/ant/unblock.md` | config (wrapper) | request-response | `.opencode/commands/ant/preferences.md` | exact |
| `.aether/docs/retired-tests-ledger.md` | doc | — | `.aether/docs/known-issues.md` | role-match |
| debug-artifact test(s) in `pkg/codex/` for timeout/non-zero-exit (D-03) | test (integration + unit) | event-driven (subprocess failure paths) | `pkg/codex/platform_dispatch_test.go::TestWriteHostedWorkerOutputDebugRedactsProviderOutput` (unit call site) + `TestInvokeHostedWorkerEnvVarOverride`/`NoEnvVarOverride` (fake-shell-script integration style) | exact |
| `cmd/gate.go` (add `checkAntiPatternGate` producer) | service/gate producer | CRUD (reads state, produces gateCheck) | `checkNoCriticalFlags` (`cmd/gate.go:323-354`), `checkAllTasksCompleted` (`cmd/gate.go:357-411`) | exact |
| `cmd/security_cmds.go` (extract shared scan function) | service (refactor) | transform | existing `checkAntipatternCmd.RunE` body itself (`cmd/security_cmds.go:35-219`) — the extraction target | exact (self-referential; no external analog needed) |
| `cmd/maintenance.go` (`dataCleanCmd` — add worker-debug retention) | service (maintenance) | batch (prune-by-age/count) | `backupPruneGlobalCmd` (`cmd/maintenance.go:95-169`, cap-based prune) and `tempCleanCmd` (`cmd/maintenance.go:171-214`, age-based prune) | exact |
| `pkg/codex/platform_dispatch.go` (`writeHostedWorkerOutputDebug` new call sites) | service | event-driven | the two existing call sites at `pkg/codex/platform_dispatch.go:1032` and `:1040` | exact |
| `cmd/codex_build_worktree.go` (tracking-root resolution for debug artifacts) | service | file-I/O | `workerTrackingRoot` (`pkg/codex/worker.go:648-653`), already used at `pkg/codex/platform_dispatch.go:977` for process tracking | exact |

## Pattern Assignments

### `cmd/policy_schema_test.go` (test, file-I/O)

**Analog (spec to port):** `control-ts/tests/schemas/policy.schema.test.ts` (121 lines, full file read)
**Analog (Go idiom for reading a project file and asserting on it):** `cmd/claudemd_verification_depth_test.go`

**What the TS spec asserts today** (full content read, lines 1-121) — 8 `it()` blocks, one per policy YAML file, each reading a fixture, parsing with `PolicySchema.parse`, and asserting specific field values:
```typescript
// control-ts/tests/schemas/policy.schema.test.ts:8-16
it("validates model-routing.yaml", () => {
  const text = readFileSync(fixturePath("policies", "model-routing.yaml"), "utf8");
  const data = parse(text);
  const result = PolicySchema.parse(data);
  expect(result.model_routing.default_provider).toBe("anthropic");
  expect(result.memory_rules.max_learnings).toBe(100);
  expect(result.skill_creation.allowed).toBe(true);
  expect(result.safety_gates.security_scan).toBe(true);
});
```
Required-field surface across all 8 blocks (port these field checks, not the Zod schema itself — RESEARCH.md's Don't-Hand-Roll table is explicit that a full typed loader is Phase 161/MODEL-01's job, not this phase's):
`model_routing.default_provider`, `memory_rules.max_learnings`, `memory_rules.learning_retention_days`, `memory_rules.instinct_cap`, `memory_rules.event_cap`, `skill_creation.allowed`, `skill_creation.max_skills_per_colony`, `skill_creation.require_wisdom_threshold`, `safety_gates.security_scan`, `safety_gates.chaos_scan`, `safety_gates.auditor_score_threshold`, `dispatch_contract.max_workers_per_phase`, `dispatch_contract.spawn_depth_limits`, `dispatch_contract.timeout_defaults`, `pheromone_lifecycle.default_ttl_days`, `pheromone_lifecycle.auto_expire_on_phase_end`, `pheromone_lifecycle.max_active_signals`, `pheromone_lifecycle.strength_decay_per_day`, `signal_rules.auto_emit_on_phase_complete`, `signal_rules.max_feedback_per_phase`, `signal_rules.hard_constraint_prefixes` (must contain `[error-pattern]`, `[redirect]`), `autopilot.replan_interval`, `autopilot.pause_conditions` (must contain `test_failure`, `critical_chaos`, `security_gate_failure`, `quality_gate_failure`, `runtime_verification_needed`).

Note: the TS test reads from `control-ts/tests/fixtures/policies/*.yaml` (fixture copies), **not** the live `colony/policies/*.yaml`. RESEARCH.md's own confidence note flags these as possibly drifted — the new Go test should read the live `colony/policies/*.yaml` files directly (`cmd/policy_loader.go` already has `loadYAMLPolicy(path string, out interface{})` as the generic YAML reader to reuse), which is a strict improvement over testing a fixture copy.

**Go file-read-and-assert idiom to copy structurally** (`cmd/claudemd_verification_depth_test.go:11-16`):
```go
func TestCLAUDEMDVerificationDepthClaims(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)
	// ... strings.Contains assertions ...
}
```
For YAML parsing use `map[string]interface{}` (loose decode, matching the "narrow test, not a typed loader" framing) — unmarshal via `gopkg.in/yaml.v3` or whatever `cmd/policy_loader.go` already imports (read that file's import block before writing this test to match the exact YAML library already in use).

---

### Generalized command-call execution audit test (LOUD-03/04/05)

**Analogs:** `cmd/security_gate_drift_test.go` (execution style, full 142-line file read above) and `cmd/cli_flag_audit_test.go` (enumeration style + the exact positional-arg blind spot, full 189-line file read above).

**Execution-style pattern to generalize** (`cmd/security_gate_drift_test.go:18-65`, `TestCheckAntipatternAcceptsPlaybookInvocationForm`):
```go
func TestCheckAntipatternAcceptsPlaybookInvocationForm(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	target := filepath.Join(tmpDir, "config.go")
	os.WriteFile(target, []byte(`...fixture...`), 0644)

	renderedCommandExitCode.Store(0) // rootCmd.Execute() alone does not reset it
	rootCmd.SetArgs([]string{"check-antipattern", target})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("playbook-form invocation failed: %v", err)
	}
	if code := int(renderedCommandExitCode.Load()); code != 0 { ... }
	env := parseEnvelope(t, buf.String())
	// ... assert envelope fields ...
}
```
This is the pattern for the "curated subset of read-only, safe-to-execute commands" half of the audit (Open Question 1 recommendation (b)): drive `rootCmd.Execute()` for real against a `newTestStore(t)` fixture, reset globals with `saveGlobals(t)`/`resetRootCmd(t)`, and parse the JSON envelope with the existing `parseEnvelope(t, ...)` helper (defined elsewhere in `cmd`, already used by `security_gate_drift_test.go`).

**Enumeration-style pattern with the exact gap to close** (`cmd/cli_flag_audit_test.go:32,117` — see full excerpt already quoted in RESEARCH.md § Code Examples; reproduced here for the planner):
```go
// Outer regex only captures a trailing run of --flag tokens
re := regexp.MustCompile(`aether\s+([\w][\w-]*)\s+((?:--[\w][\w-]*(?:=\S*|\s+\S*)?\s*)*)`)
// ...
// Inner regex only ever looks at --flag tokens inside that captured group
flagRe := regexp.MustCompile(`--([\w][\w-]*)`)
flagMatches := flagRe.FindAllStringSubmatch(flagsStr, -1)
```
`TestCLIFlagAudit` already walks `../.claude/commands/ant/`, `../.opencode/commands/ant/`, `../.aether/docs/command-playbooks/`, builds a `registered map[string]map[string]bool` of subcommand→flags from `rootCmd.Commands()`, and has a `skipSubcommands` allowlist for markdown-only pseudo-commands. The new test should extend or sit alongside this exact scanning skeleton, adding: (1) a raw remainder-of-line capture and positional-token count per call, (2) a `map[string]int` (or direct `cmd.Args(cmd, syntheticArgs)` invocation, since `cobra.PositionalArgs` is `func(cmd *Command, args []string) error`) to check positional-arg shape per command. `TestGatekeeperPlaybooksUsePositionalForm` (`cmd/security_gate_drift_test.go:124-142`) is the smallest existing example of pinning a playbook's exact call text against a `findRepoRoot()`-resolved path — reuse `findRepoRoot()` rather than re-deriving repo-root logic (also see `repoRootForCommandSourceTest()` in `cmd/command_source_hygiene_test.go:157-191` as a second, slightly more defensive repo-root-finder if `findRepoRoot()` proves insufficient for this test's directory set).

**No existing test drives `cmd.Args` validators synthetically today** — this is a genuinely new technique for this test (not present in any analog), so the planner should budget explicit design time for it rather than assuming it can be copied verbatim.

---

### Invariant/count test for `2>/dev/null` in live wrapper dirs (LOUD-06)

**Style exemplar (not a grep test — the proportion/invariant testing style to imitate):** `cmd/codex_build_test.go:3182-3221`, `TestBuildWorkerBriefIsMostlyTask`

```go
// TestBuildWorkerBriefIsMostlyTask is the regression lock that matters. It
// asserts a proportion rather than the presence of any one section, so any
// future addition that pushes framework scaffolding past half the prompt fails
// here regardless of what that addition is called.
func TestBuildWorkerBriefIsMostlyTask(t *testing.T) {
	...
	taskChars := 0
	for _, section := range splitBriefSections(brief) {
		switch section.Name {
		case "Assignment", "Phase Objective", ...:
			taskChars += section.Chars
		}
	}
	share := float64(taskChars) / float64(len(brief)) * 100
	if share < 40 {
		t.Errorf("task-relevant content is %.1f%% of the worker brief (%d of %d chars); framework scaffolding now outweighs the task",
			share, taskChars, len(brief))
	}
}
```
The transferable idea: assert a **count/proportion invariant**, with a failure message that states the measured value, not just pass/fail. For LOUD-06 (Open Question 7's recommendation), the new test should `grep`-equivalent (in Go: read every `.md` file under `.claude/commands/ant/` and `.opencode/commands/ant/`, `strings.Count(content, "2>/dev/null")`, sum) and assert the total equals exactly 4 — not "contains at most 4" and not a hardcoded list of the 4 filenames, so any *new* suppression site trips the test regardless of where it lands. Use the directory-walking style already established in `cmd/cli_flag_audit_test.go:71-83` (`os.ReadDir`, filter `.md`, `os.ReadFile`) for the file-collection half; there is no need for a new directory-walk pattern.

---

### Static-doc test for LOUD-07 doc claims

**Analog:** `cmd/claudemd_verification_depth_test.go` (full file read above, 92 lines)

Both `TestCLAUDEMDVerificationDepthClaims` (asserts specific strings ARE present) and `TestCLAUDEMDNoOldModes` (asserts specific strings are ABSENT — direct structural match for LOUD-07's need) are relevant:
```go
// cmd/claudemd_verification_depth_test.go:77-91 — the "must NOT claim X" half,
// structurally identical to what LOUD-07 needs for CLAUDE.md/AGENTS.md/
// structural-learning-stack.md's present-tense consolidation claims.
func TestCLAUDEMDNoOldModes(t *testing.T) {
	data, err := os.ReadFile("../CLAUDE.md")
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "`fast`: low-risk work; light continue verification") {
		t.Error("CLAUDE.md still contains old 'fast' mode description")
	}
	if strings.Contains(content, "watcher subprocess skipped") {
		t.Error("CLAUDE.md still contains 'watcher subprocess skipped'")
	}
}
```
For LOUD-07, read all three files (`../CLAUDE.md`, `../AGENTS.md`, `../.aether/docs/structural-learning-stack.md`) and assert the specific present-tense phrases RESEARCH.md already located are gone:
- `structural-learning-stack.md:14` — "The stack runs automatically at phase-end and seal"
- `structural-learning-stack.md:207` — "Runs at the end of every phase (`/ant-continue`)"
- `structural-learning-stack.md:227` — the matching table row
- `AGENTS.md:899` — "Lifecycle integration: phase-end at `aether continue`, full at `aether seal`"
- `CLAUDE.md:838` — the stale claim contradicting `CLAUDE.md:37`

Match the exact failure-message style (`t.Errorf`/`t.Error` naming the file and the exact stale phrase, not a generic "doc claim wrong").

---

### Live-path gate test for antipattern gate (LOUD-02)

**Analog for the gate-producer function shape:** `cmd/gate.go:323-354` (`checkNoCriticalFlags`) and `cmd/gate.go:357-411` (`checkAllTasksCompleted`) — both already read in full above.

```go
// cmd/gate.go:322-354 — the exact producer-function shape to copy for
// checkAntiPatternGate(files []string) gateCheck
func checkNoCriticalFlags() gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{Name: "no_critical_flags", Passed: true, Detail: "no state file found, skipping flag check"}
	}
	criticalCount := 0
	for _, record := range state.Errors.Records {
		if strings.EqualFold(record.Severity, "CRITICAL") {
			criticalCount++
		}
	}
	if criticalCount > 0 {
		return gateCheck{Name: "no_critical_flags", Passed: false, Detail: fmt.Sprintf("%d critical error record(s) found", criticalCount)}
	}
	return gateCheck{Name: "no_critical_flags", Passed: true, Detail: "no critical flags"}
}
```
`gateCheck{Name, Passed, Detail}` is the return shape every producer function uses — `checkAntiPatternGate` should follow it exactly. The classification map entry already exists and needs no changes (`cmd/gate.go:621`, `softBlock` tier, rationale "Critical patterns are actionable but non-blocking when addressed") and the auto-resolve threshold already exists (`cmd/gate.go:715`, `0.0`).

**Analog for the fixture/setup pattern in the new integration test** (`cmd/security_gate_drift_test.go:18-33`, already quoted above): `saveGlobals(t)`, `resetRootCmd(t)`, `newTestStore(t)`, a fixture file containing a hardcoded credential (`var apiKey = "sk-live-4f9a8b7c6d"`), then assert on both the CLI envelope AND (new, for this gate test) on the `gateCheck.Passed` value returned by the new producer function when called directly, plus an end-to-end assertion that the continue pipeline actually invokes it (per RESEARCH.md's Pitfall 1 — a unit test of the producer function alone is not sufficient; the plan must also prove the continue pipeline calls it, e.g. by driving the actual gate-evaluation path used at `cmd/codex_continue.go` and checking `checkAntiPatternGate` appears in the resulting gate results/blockers).

**Shared scan-function extraction target** (`cmd/security_cmds.go:35-219`, `checkAntipatternCmd.RunE` — full body already read above): extract the scanning logic (everything from the language-specific `switch ext` block through the `secretRe`/`todoRe` common-pattern checks, i.e. roughly `cmd/security_cmds.go:70-211`) into a new unexported function, e.g. `scanFileForAntipatterns(filePath string) (criticals, warnings []AntipatternFinding, err error)`, callable both from `checkAntipatternCmd.RunE` and from `checkAntiPatternGate`. This is explicitly named in RESEARCH.md's Don't-Hand-Roll table as required so "the CLI contract and the gate check can never drift apart."

---

### `.aether/commands/unblock.yaml`, `.claude/commands/ant/unblock.md`, `.opencode/commands/ant/unblock.md` (LOUD-08)

**Analog (full files, already read above):** `.aether/commands/preferences.yaml` (8 lines), `.claude/commands/ant/preferences.md` (40 lines). `.opencode/commands/ant/preferences.md` is confirmed identical in line count (40) to the Claude mirror — read it too before writing the OpenCode wrapper to confirm content parity, since `TestClaudeOpenCodeAgentContentParity`-style tests exist elsewhere in this repo for agent bodies and a similar expectation likely applies to command wrappers (verify during planning whether a parity test already covers command wrapper bodies, not just agent bodies).

**YAML source structure to copy exactly** (`.aether/commands/preferences.yaml:1-8`):
```yaml
name: ant-preferences
description: "🧠 Add or list user preferences in hub QUEEN.md"
source_of_truth: "Use the Go `aether` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether preferences $ARGUMENTS"
guardrails:
  - "Do not write colony state files, session files, or pheromone files by hand from this command spec."
  - "If docs and runtime disagree, runtime wins."
```
For `unblock.yaml`, the runtime command is already known from `cmd/unblock_cmd.go:12-19,132-134` — flags are `--phase` (int, default 0/current phase), `--fixer-mode` (string, default `"propose"`, values `full`/`propose`/`advise`), `--dispatch` (bool). Suggested `runtime.command`: `"AETHER_OUTPUT_MODE=visual aether unblock $ARGUMENTS"` (matching the `$ARGUMENTS`-passthrough convention `preferences.yaml` uses, since `unblock` takes flags rather than a single positional string — verify against a second analog with flag-style args if one exists, e.g. check `.aether/commands/redirect.yaml` or similar during planning if `preferences.yaml`'s single-positional-arg shape doesn't map cleanly).

**Markdown wrapper header and structure to copy exactly** (`.claude/commands/ant/preferences.md:1-6`, header format enforced by `cmd/command_source_hygiene_test.go:12,14` — the `generatedCommandHeaderPattern` regex):
```markdown
<!-- Aether-managed: runtime spec at .aether/commands/preferences.yaml. Synced by aether update. -->
---
name: ant-preferences
description: "🧠 Add or list user preferences in hub QUEEN.md"
---
```
The header line MUST match `^<!-- Aether-managed: runtime spec at (\.aether/commands/[^ ]+\.yaml)\. Synced by aether update\. -->$` exactly (verified regex from `cmd/command_source_hygiene_test.go:12`) or `TestCommandWrappersReferenceRealYamlSources` fails. The rest of `preferences.md` (Steps: Validate / Route / Confirm, "Why the CLI" callout, "Next steps" footer) is the narrative structure to imitate for `unblock.md`, substituting `unblock`'s actual flags/behavior (phase gate recovery summary + optional Fixer dispatch) for preferences' add/list behavior.

**Codex note:** no `.codex/agents/` or `.codex/CODEX.md` wrapper is needed — confirmed by grep that `.codex/CODEX.md` has zero references to "unblock" today (nothing stale to fix), consistent with CLAUDE.md's Platform Policy that Codex gets no wrapper markdown (Go runtime only). `cmd/unblock_cmd.go` already works as a plain `aether unblock` CLI invocation for Codex users.

**Goldens to regenerate** (RESEARCH.md § Code Examples, already verified to exist): `cmd/testdata/command_catalog.json` (checked by `cmd/audit_catalog_test.go:12`), `cmd/testdata/parity_snapshot.json` (checked by `cmd/parity_test.go:159`). Find the `-update-golden` invocation convention by grepping these two test files for the flag name before running it (RESEARCH.md notes `cmd/testdata/regression_snapshot.json`'s `command_count` field should NOT change, since `unblock` is already a registered Go subcommand — only the wrapper markdown is new).

---

### `.aether/docs/retired-tests-ledger.md` (RETIRE-04)

**Analog:** `.aether/docs/known-issues.md` (full file read above, 51 lines) — closest in tone and per-entry structure (short heading, `**Area:**`/`**Impact:**`/`**Mitigation:**` fields).

```markdown
### Post-launch provider/API/auth failures can look like worker parse failures

- **Area:** Real worker dispatch through hosted platforms after the worker process starts
- **Impact:** ...
- **Mitigation:** ...
```
No existing "deleted-test ledger" pattern exists anywhere in the repo (confirmed by RESEARCH.md's own search — the closest thing, `review-ledger`, is a different system for phase-review findings, not deleted tests). Recommended shape (per RESEARCH.md Open Question 4): one entry per deleted test file, adapting `known-issues.md`'s field style to: **Original path**, **What it covered**, **Disposition** (`dead-with-no-replacement` | `recovered-by:<test>`), **Removed in commit**. First two entries to seed the ledger with (both already identified by CONTEXT.md/RESEARCH.md):
1. `control-ts/tests/schemas/policy.schema.test.ts` → disposition `recovered-by:cmd/policy_schema_test.go`
2. `.aether/ts-host/test/playbook-loader.test.ts` (already deleted in commit `b2b41486`) → disposition `dead-with-no-replacement`, retroactive entry.

`.aether/docs/migration-map.md` was also considered as an analog (table-heavy, per-milestone structure) but is a poor structural fit — it documents forward migration plans with Requirements/Risk tables, not a flat append-only record of removed test files. Use `known-issues.md`'s narrative-entry style, not `migration-map.md`'s table style.

---

### Debug-artifact test(s) in `pkg/codex/` for timeout/non-zero-exit (D-03)

**Analog (unit test calling the debug-write function directly):** `pkg/codex/platform_dispatch_test.go:1077-1113`, `TestWriteHostedWorkerOutputDebugRedactsProviderOutput`

```go
func TestWriteHostedWorkerOutputDebugRedactsProviderOutput(t *testing.T) {
	root := t.TempDir()
	rel := writeHostedWorkerOutputDebug(
		root, "opencode",
		WorkerConfig{WorkerName: "Forge-2", Caste: "builder", TaskID: "2.2", AgentName: "aether-builder"},
		[]string{"run", "--agent", "build", "prompt with sk-proj-prompt-secret"},
		"stdout sk-proj-stdout-secret token=stdout-secret",
		"stderr ghp_stderr_secret secret=stderr-secret",
		fmt.Errorf("parse failed sk-proj-error-secret"),
	)
	if rel == "" { t.Fatal("expected debug artifact path") }
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	// ... assert secrets absent, assert redaction marker and stdout_bytes/stderr_bytes present ...
}
```
This is the exact pattern for a unit test asserting D-03/D-04's new debug-artifact content once the two new call sites exist — same direct-call, `t.TempDir()`, read-back-and-assert style.

**Analog (integration test driving the real subprocess dispatch path with a fake shell script) — this is what proves the NEW timeout/non-zero-exit call sites actually fire, not just that `writeHostedWorkerOutputDebug` works in isolation:** `pkg/codex/platform_dispatch_test.go:784-834`, `TestInvokeHostedWorkerEnvVarOverride`
```go
scriptPath := filepath.Join(dir, "fake-opencode.sh")
script := `#!/bin/sh
env | grep -i AETHER > "$ENV_CAPTURE_PATH"
cat <<'EOF'
{"type":"message.part.updated","part":{"type":"text","text":"..."}}
EOF
`
os.WriteFile(scriptPath, []byte(script), 0755)
invoker := &OpenCodeDispatcher{binaryName: scriptPath}
_, err := invoker.Invoke(t.Context(), WorkerConfig{ ... Root: dir })
```
For D-03's new tests, adapt this fake-shell-script technique: (1) a non-zero-exit test — a script that does `exit 1` (or writes garbage and exits nonzero) — and (2) a timeout test — a script that `sleep`s past a short `AETHER_*_TIMEOUT`-equivalent test override (check `WorkerConfig`/dispatch context for how timeout is set in tests; the timeout path is detected via `ctx.Err() == context.DeadlineExceeded` at `pkg/codex/platform_dispatch.go:1013`). After invoking, assert a debug artifact now exists under `filepath.Join(dir, ".aether", "data", "worker-debug")` (the fixed relative path `writeHostedWorkerOutputDebug` always writes to, per `pkg/codex/platform_dispatch.go:1224`) — this is the concrete proof that the two new call sites (currently missing per RESEARCH.md's Code Examples gap analysis at `pkg/codex/platform_dispatch.go:1013-1022`) were added correctly.

**Gap to fix, exact insertion points** (already fully diagnosed in RESEARCH.md § Code Examples "Debug-artifact insertion points for D-03" — reproduced here for completeness since it is the direct implementation target, not just a test target):
```go
// pkg/codex/platform_dispatch.go:1013-1018 — timeout path, currently NO debug write
if ctx.Err() == context.DeadlineExceeded {
	...
	return WorkerResult{..., Error: fmt.Errorf("worker timeout after %v", reportedTimeout)}, nil
	// gap: no writeHostedWorkerOutputDebug call here
}
if waitErr != nil {
	// pkg/codex/platform_dispatch.go:1020-1022 — non-zero exit path, currently NO debug write
	return WorkerResult{..., Error: classifyHostedExecutionError(...)}, nil
	// gap: no writeHostedWorkerOutputDebug call here
}
```
Compare to the two existing correct call sites at `pkg/codex/platform_dispatch.go:1032` and `:1040` (both already quoted in full in the Shared Patterns section below). Per RESEARCH.md's Pitfall 5, **both new call sites must pass `workerTrackingRoot(config)`, not `config.Root`** — see Shared Patterns.

---

### `cmd/maintenance.go` (`dataCleanCmd` — worker-debug retention, D-04)

**Analog for age-based pruning:** `tempCleanCmd` (`cmd/maintenance.go:171-214`, full file already read above):
```go
cutoff := time.Now().Add(-7 * 24 * time.Hour)
cleaned := 0
for _, entry := range entries {
	if entry.IsDir() { continue }
	info, err := entry.Info()
	if err != nil { continue }
	if info.ModTime().Before(cutoff) {
		os.Remove(filepath.Join(tempDir, entry.Name()))
		cleaned++
	}
}
```
**Analog for cap-based pruning (oldest-first):** `backupPruneGlobalCmd` (`cmd/maintenance.go:95-169`):
```go
sort.Slice(files, func(i, j int) bool { return files[i].modTime.Before(files[j].modTime) })
if len(files) <= cap {
	outputOK(map[string]interface{}{"pruned": 0, "kept": len(files)})
	return nil
}
pruneCount := len(files) - cap
for i := 0; i < pruneCount; i++ {
	os.Remove(filepath.Join(backupDir, files[i].name))
}
```
D-04 wants both: "50 files or 14 days" — combine both analog patterns (age cutoff first via the `tempCleanCmd` style, then a count cap via the `backupPruneGlobalCmd` style, or vice versa) as a new step inside `dataCleanCmd.RunE` (`cmd/maintenance.go:18-93`) alongside the existing `pheromones.json` pruning, targeting `.aether/data/worker-debug/*.json`. Follow `dataCleanCmd`'s existing `confirm`/dry-run flag convention (`cmd/maintenance.go:28`, `dataCleanCmd.Flags().Bool("confirm", false, ...)`) — dry-run must report the count that *would* be pruned without deleting, matching the existing `reportedRemoved` dry-run branch at `cmd/maintenance.go:80-84`. This directly matters for CLAUDE.md's Definition of Done corollary: "An inspection or `--dry-run` command must not mutate state" — do not let the new worker-debug pruning step delete files when `--confirm` is absent.

**Wiring point:** `rootCmd.AddCommand(dataCleanCmd)` already exists at `cmd/maintenance.go:244` — no new command registration needed, only new logic inside the existing `RunE`.

---

## Shared Patterns

### Gate-check producer function shape (D-01 gate classification)
**Source:** `cmd/gate.go:322-411` (`checkNoCriticalFlags`, `checkAllTasksCompleted`)
**Apply to:** `checkAntiPatternGate` (the only new gate producer this phase needs, per Open Question 3's recommendation — the other five audited calls are enrichment, not gates, and get no new gate-classification entries)
```go
func checkNoCriticalFlags() gateCheck {
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return gateCheck{Name: "no_critical_flags", Passed: true, Detail: "no state file found, skipping flag check"}
	}
	// ... build detail message from a count, return gateCheck{Name, Passed, Detail} ...
}
```
Classification is already wired: `cmd/gate.go:621` (`"anti_pattern": {softBlock, "..."}`) and `cmd/gate.go:715` (auto-resolve threshold `0.0`). No changes needed to the classification map itself — only a producer function that populates a `GateCheckResult` named `"anti_pattern"`, which today nothing in the repo produces (verified by RESEARCH.md's exhaustive grep).

### Worktree-safe root resolution for any new filesystem write during worker dispatch
**Source:** `pkg/codex/worker.go:648-653`
```go
func workerTrackingRoot(config WorkerConfig) string {
	if root := strings.TrimSpace(config.TrackingRoot); root != "" {
		return root
	}
	return config.Root
}
```
**Apply to:** every `writeHostedWorkerOutputDebug` call site — the two existing ones at `pkg/codex/platform_dispatch.go:1032,1040` currently pass `config.Root` (the worktree path in worktree mode, which `finalizeBuildWorktree` at `cmd/codex_build_worktree.go:452` deletes before anyone can read the debug artifact — RESEARCH.md's Pitfall 5), and the two new D-03 call sites must not repeat this mistake. All four call sites (existing two + new two) should pass `workerTrackingRoot(config)`.

### Debug-artifact write chokepoint (redaction/sanitization already built in — do not duplicate)
**Source:** `pkg/codex/platform_dispatch.go:1219-1253`, `writeHostedWorkerOutputDebug`
```go
func writeHostedWorkerOutputDebug(root, label string, config WorkerConfig, args []string, stdoutText, stderrText string, cause error) string {
	...
	debugDir := filepath.Join(root, ".aether", "data", "worker-debug")
	...
	payload := map[string]interface{}{
		"created_at": now.Format(time.RFC3339Nano),
		"platform": strings.TrimSpace(label),
		"worker_name": strings.TrimSpace(config.WorkerName),
		"caste": strings.TrimSpace(config.Caste),
		"task_id": strings.TrimSpace(config.TaskID),
		"agent_name": strings.TrimSpace(config.AgentName),
		"args": safeHostedWorkerArgs(args),
		"stdout_bytes": len(stdoutText),
		"stderr_bytes": len(stderrText),
		"stdout_excerpt": workerOutputExcerpt(stdoutText),
		"stderr_excerpt": workerOutputExcerpt(stderrText),
		"error": sanitizeWorkerDiagnosticOutput(cause.Error()),
	}
	...
}
```
**Apply to:** D-03's requirement to add `duration`, `exit_code`, and provider-session-id fields — extend this `payload` map (do not create a second debug-writing function). `exit_code` is derivable from `waitErr` via `exec.ExitError.ExitCode()` in the non-zero-exit branch; a provider-session-id candidate already exists on `config` as `config.ProviderRunID` (already used for process tracking at `pkg/codex/platform_dispatch.go:978`) — reuse it rather than inventing a new field name. `sanitizeWorkerDiagnosticOutput` and `workerOutputExcerpt` are already called internally, so no new sanitization logic is needed for the new fields as long as the call sites route through this same function.

### Command wrapper generation chain (LOUD-08)
**Source:** `.aether/commands/preferences.yaml` + `.claude/commands/ant/preferences.md` + `.opencode/commands/ant/preferences.md`
**Apply to:** all three new `unblock` files. Enforcement test: `cmd/command_source_hygiene_test.go:12,14` (`generatedCommandHeaderPattern`, `TestCommandWrappersReferenceRealYamlSources`) — the header line format is non-negotiable and machine-checked.

### Gate/enrichment classification (D-01)
**Source:** `cmd/gate.go:591-627` (`GateClassificationTier` enum: `hardBlock`/`softBlock`/`advisory`, and the `gateClassifications` map)
**Apply to:** only `check-antipattern`/`anti_pattern` needs a new *producer*; the map entry already exists. Do not create new classification entries for `print-next-up`, `verify-claims`, `state-checkpoint`, `generate-progress-bar`, `skill-detect` — per Open Question 3's recommendation, these five have no live gate call site to classify; their "loud failure" requirement is satisfied purely by fixing their dead-doc call sites so the new audit test passes.

## No Analog Found

None. Every file this phase creates or modifies has at least a role-match analog somewhere in the current tree. The two weakest matches are:

| File | Role | Data Flow | Reason for weaker match |
|------|------|-----------|--------------------------|
| generalized command-call execution audit test's positional-arg-shape checker (the `cobra.Args`-synthetic-invocation half) | test (integration) | batch | No existing test in the repo invokes a `cobra.PositionalArgs` validator synthetically against a parsed token count — `cmd/cli_flag_audit_test.go` only ever checks flag names, never positional shape. The planner should treat this specific sub-piece as new design, not a copy job, while still reusing the surrounding file-scanning and subcommand-registry-building code from `cli_flag_audit_test.go`. |
| `.aether/docs/retired-tests-ledger.md` | doc | — | No "deleted-test ledger" has ever existed in this repo (confirmed by RESEARCH.md's search); `known-issues.md` is a tonal/structural analog (short entries with labeled fields) but tracks *live* limitations, not *removed* tests — the planner has explicit discretion here per CONTEXT.md, so this is expected, not a gap to fill by force-fitting a different existing doc's structure. |

## Metadata

**Analog search scope:** `cmd/` (all `*_test.go` and the relevant non-test `.go` files named in CONTEXT.md/RESEARCH.md), `pkg/codex/` (`platform_dispatch.go`, `platform_dispatch_test.go`, `worker.go`, `worker_diagnostics_test.go`), `.aether/commands/`, `.claude/commands/ant/`, `.opencode/commands/ant/`, `.aether/docs/`, `control-ts/tests/schemas/`.
**Files scanned:** ~20 read in full or targeted excerpt; grep sweeps across `cmd/*_test.go`, `pkg/codex/*_test.go`, `.aether/docs/*.md`.
**Pattern extraction date:** 2026-07-27
