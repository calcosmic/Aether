# Phase 205: Owner Acceptance and Restoration Seal - Pattern Map

**Mapped:** 2026-09-15
**Files analyzed:** 13 (5 new coverage JSONs + 1 new ratchet test + 4 defect-fix edits + 4 planning docs)
**Analogs found:** 13 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.planning/phases/{200,201,202,203,204}-*/{N}-CLASSIC-COVERAGE.json` | config/data (planning artifact) | batch (static signed ledger) | `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json` | exact |
| `.planning/phases/205-.../205-CLASSIC-COVERAGE.json` | config/data | batch | same as above | exact |
| `cmd/classic_coverage_ratchet_test.go` (generalized validator + master 72-row cross-phase test) | test | batch/CRUD-over-files | `cmd/classic_coverage_199_test.go` | exact |
| `cmd/seal_final_review.go` (`renderSealFinalReviewBrief`) | service/prompt-builder | request-response | `cmd/codex_continue_plan.go` (`continueExternalBriefWithHandoffSchema`, line 332) | exact (same handoff-schema append pattern, different call site) |
| `cmd/entomb_cmd.go` (`addFile`/required-sources table, ~line 367-384) | service (archival/file I/O) | file-I/O | itself, extended with a fallback branch — closest sibling pattern is the `optionalFiles`/`clear` handling already in the same function | role-match (extend same function, no external analog needed) |
| `cmd/seal_confirmation.go` (`runSealPreflightConfirmationGate`) | controller/CLI command | request-response (stdout emission) | `cmd/codex_visuals.go` (`writeVisualOutput`, `shouldRenderVisualOutput`) | exact (same stdout-gating mechanism to reuse/fix) |
| `cmd/queen.go` (`sanitizeQueenInline`) | utility (sanitizer) | transform | `pkg/colony/sanitize.go` (`SanitizeSignalContent`) | exact — this is the sanitizer that should be reused/composed, not a separate analog to copy style from |
| `.claude/commands/ant/seal.md` / `.aether/commands/seal.yaml` (defect 5 wrapper text) | config (wrapper source) | request-response | `.aether/commands/*.yaml` → generated wrapper pattern (any other Aether-managed wrapper with a "Synced by aether update" header) | role-match |
| `.planning/phases/205-.../205-UAT.md` | test/report (planning doc) | batch (verdict recording) | `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` (also `201-UAT.md`, `202-UAT.md` for tri-state/skipped vocabulary) | exact |
| `.planning/phases/205-.../205-BRIEF.md` (one-page owner brief) | config/doc | request-response (owner-read) | no close analog exists; nearest structural kin is a `*-UAT.md` header/frontmatter block for status tracking, and `.planning/field-reports/*.md` for plain-English incident framing | no analog (see below) |
| `.planning/phases/205-.../205-LIMITATIONS-CARD.md` | config/doc | batch (WINDOWS.md rollup) | `.planning/WINDOWS.md` (source table to transcribe from) | role-match |
| Platform honesty cards (OpenCode/Codex, D-13) | doc/report | request-response (one real run each) | `.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md` (plain-English incident/finding format) | role-match |

## Pattern Assignments

### `.planning/phases/{200..205}-*/{N}-CLASSIC-COVERAGE.json` (config/data, batch)

**Analog:** `.planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json`

**Row shape to replicate exactly** (verbatim, one row):
```json
{"type":"GOAL","id":"CAP-006","disposition":"replace-better","modern_home":"cmd/entomb_cmd.go","plan_task":"199-21 Task 1","artifact":"cmd/entomb_cmd.go","proofs":["TestLifecycleCloseout199Entomb"],"status":"PASS","historical_evidence":"Classic entomb evidence retained in 199-CLASSIC-SYNTHESIS.md"}
```
Top-level document shape:
```json
{"version": "1", "rows": [ /* one row object per GOAL/REQ/RESEARCH/CONTEXT id */ ]}
```
Required fields per row (none may be empty/TODO/MISSING/UNSUPPORTED — enforced by `containsClassicCoverage199Placeholder`): `type` (`GOAL|REQ|RESEARCH|CONTEXT`), `id`, `disposition` (`keep-current|restore-modern|replace-better|retire-with-proof`), `modern_home`, `plan_task`, `artifact` (must resolve to a real, non-directory file on disk), `proofs` (non-empty array, each must resolve to a real Go test function name or a `cases.json` id), `status` (must literally be `"PASS"`), `historical_evidence`.

**Row counts per phase (GOAL/CAP rows only, from RESEARCH.md ledger)**: 200→7, 201→8, 202→8, 203→6, 204→7, 205→2 (`CAP-023`, `CAP-054`). `REQ`/`RESEARCH`/`CONTEXT` id lists must be pulled from each phase's own `REQUIREMENTS.md` mapping row and its own `*-CONTEXT.md`/synthesis decision IDs — do not invent them; extract the same way 199's list was built (its 12 REQ / 10 RESEARCH / 17 CONTEXT ids came straight from that phase's own documents).

---

### `cmd/classic_coverage_ratchet_test.go` (test, batch)

**Analog:** `cmd/classic_coverage_199_test.go` (full file read this session)

**Struct + path constant** (lines 1-30):
```go
const classicCoverage199Path = ".planning/phases/199-front-door-and-classic-contract/199-CLASSIC-COVERAGE.json"

type classicCoverage199Document struct {
	Version string                  `json:"version"`
	Rows    []classicCoverage199Row `json:"rows"`
}

type classicCoverage199Row struct {
	Type               string   `json:"type"`
	ID                 string   `json:"id"`
	Disposition        string   `json:"disposition"`
	ModernHome         string   `json:"modern_home"`
	PlanTask           string   `json:"plan_task"`
	Artifact           string   `json:"artifact"`
	Proofs             []string `json:"proofs"`
	Status             string   `json:"status"`
	HistoricalEvidence string   `json:"historical_evidence"`
}
```
**Generalize this struct as-is** (field names must stay identical across all six new files — CONTEXT.md's own warning: "the new ratchet must not duplicate a second vocabulary"). Parameterize only the file path and the `expected` ID map per phase.

**Exact-sets validator to generalize** (lines 152-176, full function read):
```go
func validateClassicCoverage199ExactSets(document classicCoverage199Document) error {
	expected := map[string][]string{
		"GOAL":     {"CAP-006", "CAP-007", /* ... 34 total */},
		"REQ":      {"SYNTH-01", "CEC-01", /* ... 12 total */},
		"RESEARCH": {"SYN-199-01", /* ... 10 total */},
		"CONTEXT":  {"D-01", /* ... 17 total */},
	}
	seen := make(map[string]map[string]int, len(expected))
	for group := range expected { seen[group] = map[string]int{} }
	for _, row := range document.Rows {
		if _, ok := seen[row.Type]; !ok {
			return fmt.Errorf("unknown coverage type %q", row.Type)
		}
		seen[row.Type][row.ID]++
	}
	for group, ids := range expected {
		if len(seen[group]) != len(ids) {
			return fmt.Errorf("%s has %d unique rows, want %d", group, len(seen[group]), len(ids))
		}
		for _, id := range ids {
			if seen[group][id] != 1 {
				return fmt.Errorf("%s/%s has %d rows, want exactly one", group, id, seen[group][id])
			}
		}
	}
	if len(document.Rows) != 73 { /* per-phase total, not 73 for 200-205 */
		return fmt.Errorf("coverage has %d rows, want 73", len(document.Rows))
	}
	return nil
}
```

**Required-fields + disposition-enum validator** (same file, `validateClassicCoverage199RequiredFields`):
```go
func validateClassicCoverage199RequiredFields(document classicCoverage199Document) error {
	for _, row := range document.Rows {
		for field, value := range map[string]string{
			"id": row.ID, "disposition": row.Disposition, "modern_home": row.ModernHome,
			"plan_task": row.PlanTask, "artifact": row.Artifact, "status": row.Status,
			"historical_evidence": row.HistoricalEvidence,
		} {
			if strings.TrimSpace(value) == "" || containsClassicCoverage199Placeholder(value) {
				return fmt.Errorf("%s/%s has invalid %s %q", row.Type, row.ID, field, value)
			}
		}
		if len(row.Proofs) == 0 {
			return fmt.Errorf("%s/%s has no proof", row.Type, row.ID)
		}
		if row.Disposition != "keep-current" && row.Disposition != "restore-modern" && row.Disposition != "replace-better" && row.Disposition != "retire-with-proof" {
			return fmt.Errorf("%s/%s has unsupported disposition %q", row.Type, row.ID, row.Disposition)
		}
	}
	return nil
}
```

**The AST-based proof-resolution ratchet — never hand-roll a "known good" test list** (`buildClassicCoverage199ProofIndex`, lines 279-330):
```go
func validateClassicCoverage199ProofResolution(document classicCoverage199Document) error {
	root := findTestModuleRootForClassicCoverage199()
	index, err := buildClassicCoverage199ProofIndex(root)
	if err != nil { return err }
	for _, row := range document.Rows {
		artifact := filepath.Join(root, filepath.FromSlash(row.Artifact))
		if info, err := os.Stat(artifact); err != nil || info.IsDir() {
			return fmt.Errorf("%s/%s artifact %q does not resolve", row.Type, row.ID, row.Artifact)
		}
		for _, proof := range row.Proofs {
			if !index.has(proof) {
				return fmt.Errorf("%s/%s proof %q does not resolve", row.Type, row.ID, proof)
			}
		}
	}
	return nil
}
```
`buildClassicCoverage199ProofIndex` walks `cmd/*_test.go` with `go/parser`, collecting top-level `*ast.FuncDecl` names (rejects comment/string decoys) plus every `case.id` from `cmd/testdata/classic-contract/v1/cases.json`. Reuse this exact function (parameterize by root only — the walk logic and case-ID collection do not need to change per phase).

**Passing-status + audit-doc cross-check** (`validateClassicCoverage199PassingStatus`, `validateClassicCoverage199Audit`): every row's `status` must be `"PASS"`; a companion `{N}-CLASSIC-COVERAGE.md` must render a `| ID | disposition | ` row for every JSON row and a `## GOAL` / `## REQ` / `## RESEARCH` / `## CONTEXT` heading per group — copy this rendering-cross-check pattern for each new phase's `.md` companion.

**Master cross-phase ratchet (new, PROOF-02's literal enforcement point):** a single test unioning `type: GOAL` rows across all six `{N}-CLASSIC-COVERAGE.json` files, asserting the union equals exactly `CAP-001..CAP-072` with zero duplicates and zero gaps. This is new code with no direct analog — build it as a thin wrapper that calls each phase's own `loadClassicCoverageDocument`+`validateClassicCoverageExactSets` (generalized names) and then does one more union check across all six.

---

### `cmd/seal_final_review.go` — `renderSealFinalReviewBrief` (Defect 1: missing handoff schema sentence)

**Analog:** `cmd/codex_continue_plan.go:325-333` (full function read)
```go
// (plannedContinueReviewDispatches, plannedContinueWatcherDispatch,
// cmd/codex_continue.go) already states this schema via a SEPARATE channel
// (AssembleHostedPrompt + renderResponseContract) -- appending it here,
// rather than inside renderCodexContinueReviewBrief/renderCodexContinueWatcherBrief
// themselves (shared by both paths), is what keeps a native-Codex continue
// worker from seeing it twice (D-06).
func continueExternalBriefWithHandoffSchema(rendered string) string {
	return rendered + fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected. %s\n", codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance)
}
```
**Fix pattern:** `cmd/seal_final_review.go:151-181` (`renderSealFinalReviewBrief`, full function read this session) currently has no equivalent append. Add a sibling function (e.g. `sealExternalBriefWithHandoffSchema`) that appends the identical `fmt.Sprintf("\nYour final result's handoff object must include %s. An empty handoff is rejected. %s\n", codex.HandoffFieldsSummary, codex.HandoffOpenDecisionsGuidance)` string, called at the same call-site position the continue path uses it (immediately before the brief is returned to the dispatch caller) — reuse `codex.HandoffFieldsSummary` / `codex.HandoffOpenDecisionsGuidance` directly, do not restate the schema text inline (single source of truth, per the "one instruction, one source" pattern already established for the recruit invitation in Phase 203).

**Error handling / validation context:** the same `IsEmptyWorkerHandoff` guard (used by both continue and seal finalize paths) is what actually rejects an empty handoff — the fix's proof-test should assert the brief string now contains `codex.HandoffFieldsSummary`, mirroring however the continue path's own wiring test asserts it (grep `TestContinueLaneWorkersAreToldHowToAskForHelp`-style naming convention for the sibling seal test, e.g. `TestSealFinalReviewBriefCarriesHandoffSchema`).

---

### `cmd/entomb_cmd.go` — required-sources table (Defect 2: hard-required `.aether/HANDOFF.md`)

**Analog:** the function's own existing `optionalFiles`/`clear`-flag handling pattern (same file, `addFile` closure, lines ~346-364) and the `required` slice literal (lines ~367-384, full block read):
```go
required := []struct {
	logical  string
	archived string
	kind     string
	actual   string
	clear    bool
}{
	{".aether/data/COLONY_STATE.json", "COLONY_STATE.json", "state", filepath.Join(input.DataRoot, "COLONY_STATE.json"), false},
	...
	{".aether/HANDOFF.md", "HANDOFF.md", "tombstone_input", filepath.Join(input.Root, ".aether", "HANDOFF.md"), false},
}
for _, item := range required {
	if err := addFile(item.logical, item.archived, item.kind, item.actual, item.clear); err != nil {
		return entombPreflight{}, err
	}
}
```
**Fix pattern:** `addFile` currently wraps any read failure as a hard `required source %q: %w` error with no fallback (confirmed: no `optionalFiles`/synthesized-fallback branch exists anywhere in the file for this entry). The fix should move `.aether/HANDOFF.md` out of the flat `required` slice into a small helper (e.g. `addRequiredOrSynthesized`) that, on read failure for this one logical path only, synthesizes a minimal tombstone-input document from `COLONY_STATE.json` + `outcome` (both already loaded in this same function scope) rather than failing the whole entomb — the surrounding `verifyEntombClosureArtifact` call and `preflight.Sources = append(...)` structure stay identical to every other entry, only the failure branch for this one row changes. Companion fix: `cmd/session_flow_cmds.go:680`'s resume transaction (`tx.DeclareRemoval(lifecycleTransactionRootRepository, filepath.Join(".aether", "HANDOFF.md"))`) should be paired with a rewrite step so seal/entomb after a resume finds a fresh file rather than none at all — check whether seal already has a natural point to call `writeLocalQueenText`-style regeneration (mirrors the "single source rewrites, single source reads" discipline used for `sanitizeQueenInline`/`appendEntriesToQueenSection` below).

---

### `cmd/seal_confirmation.go` — `runSealPreflightConfirmationGate` (Defect 4: JSON-mode stdout pollution)

**Analog:** `cmd/codex_visuals.go` `writeVisualOutput` / `shouldRenderVisualOutput` — the existing gate mechanism that should be consulted, not duplicated.

**Current (buggy) code, full function read** (`cmd/seal_confirmation.go:342-364`):
```go
func runSealPreflightConfirmationGate(preflight SealPreflight) (bool, map[string]interface{}) {
	visualFprint(stdout, renderSealPreflightCard(preflight))
	question := SealConfirmationCopy(preflight)
	recordedAnswer := loadSealConfirmationRecordedAnswer(store, question)
	named := make([]string, 0, len(preflight.UnresolvedItems))
	for _, item := range preflight.UnresolvedItems {
		named = append(named, item.Summary)
	}
	decision := decideSealConfirmation(sealConfirmationInput{NamedProblems: named, RecordedAnswer: recordedAnswer})
	if decision.Proceed {
		return true, nil
	}
	...
	visualFprint(stdout, renderSealPreflightConfirmationQuestionVisual(preflight, next))
	return false, map[string]interface{}{ ... }
}
```
**Root cause:** `writeVisualOutput` / `visualFprint` writes unconditionally regardless of `AETHER_OUTPUT_MODE`; `shouldRenderVisualOutput(w)` only gates platform-hint *command translation*, never whether text is written at all. **Fix pattern:** gate both `visualFprint(stdout, ...)` calls in this function behind the same JSON-mode detection `outputOK`/`outputError` already use elsewhere in this file (grep the file for how JSON-mode is detected before `outputOK` is called at the end of `sealFinalizeCmd.RunE`) — if JSON output mode is active, suppress the visual card write (or route it to stderr) so `--completion-file`/JSON consumers get pure JSON on stdout, matching how other JSON-mode commands in `cmd/` already separate human card output from machine output.

---

### `cmd/queen.go` — `sanitizeQueenInline` (Defect 6: no content-safety filter on promoted lessons)

**Analog:** `pkg/colony/sanitize.go` `SanitizeSignalContent` (full relevant block read, lines 1-40+):
```go
const maxSignalContentLength = 500

func SanitizeSignalContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if len(content) > maxSignalContentLength {
		return "", fmt.Errorf("content exceeds maximum length of %d characters (%d)", maxSignalContentLength, len(content))
	}
	if findings := DetectPromptIntegrityFindings(content); len(findings) > 0 {
		first := findings[0]
		switch first.Kind {
		case "xml_tag":
			return "", fmt.Errorf("content contains XML structural tags which are not allowed")
		case "prompt_injection":
			return "", fmt.Errorf("content contains prompt injection patterns which are not allowed")
		case "shell_injection":
			return "", fmt.Errorf("%s", first.Message)
		...
		}
	}
	...
}
```
**Current (buggy) code** — `cmd/queen.go` (full function read):
```go
func sanitizeQueenInline(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(strings.Join(strings.Fields(value), " "))
}
```
Called from `writeSealReusableLessonsToQueen` (`cmd/seal_final_review.go:823-849`, full function read):
```go
func writeSealReusableLessonsToQueen(phase int, lessons []string) (int, string) {
	...
	for _, lesson := range lessons {
		lesson = sanitizeQueenInline(lesson)
		if lesson == "" { continue }
		entry := fmt.Sprintf("- %s (seal review phase %d, %s)", lesson, phase, time.Now().UTC().Format("2006-01-02"))
		if !isEntryInText(text, entry) { entries = append(entries, entry) }
	}
	...
	text = appendEntriesToQueenSection(text, "Wisdom", entries)
	if err := writeLocalQueenText(text); err != nil { return 0, err.Error() }
	return len(entries), ""
}
```
**Fix pattern:** compose `SanitizeSignalContent` (or its underlying `DetectPromptIntegrityFindings`) into the whitespace-collapse `sanitizeQueenInline` already does, and additionally reject/skip a lesson whose content matches credential/secret patterns (e.g. `.env`, `credentials`, `password`, `secret`, `api_key` — reuse whatever pattern list `DetectPromptIntegrityFindings` or the shell-injection detector already carries rather than inventing a new regex list) before appending to `entries` in `writeSealReusableLessonsToQueen`. A rejected lesson should be dropped silently (same as an empty string is dropped today), not surfaced as an error to the seal caller — mirror the existing `if lesson == "" { continue }` short-circuit shape.

---

### `.claude/commands/ant/seal.md` / `.aether/commands/seal.yaml` (Defect 5: wrapper vs. runtime drift)

**No safe copy-paste analog** — this is a wording fix, not a new mechanism. `sealReviewRequiredCastes` (`cmd/seal_final_review.go:51-63`) delegates to `queenSealReviewSpecs(state, phase, depth)`, which is the actual source of truth for which castes dispatch. The fix is to read that function's real logic (flagged as an open question in RESEARCH.md — not yet read this session) and rewrite the wrapper's flat "dispatches: Gatekeeper, Auditor, and Probe" sentence in `.aether/commands/seal.yaml` (the source of truth) to describe the real conditional logic in plain English, then regenerate/hand-sync the `.claude/` and `.opencode/` mirrors identically (per the wrapper-triplet-byte-identical project memory). **Anti-pattern:** do not hand-edit `.claude/commands/ant/seal.md` alone.

---

### `.planning/phases/205-.../205-UAT.md` (test/report, batch)

**Analog:** `.planning/phases/199-front-door-and-classic-contract/199-UAT.md` (frontmatter + block shape, full header read), extended per `202-UAT.md`'s `result: skipped` precedent for the tri-state/conditional vocabulary.

**Frontmatter + block shape to extend:**
```markdown
---
status: complete
phase: 199-front-door-and-classic-contract
source: [199-01-SUMMARY.md, ...]
started: 2026-09-06T20:51:26Z
updated: 2026-09-06T22:48:18Z
---

## Current Test

[testing complete]

## Tests

### 1. Guided Help and Current Standing
expected: <plain-English description of what should happen>
result: pass
```
**Extension for 205 (D-09, D-19):** `result:` must support `pass` / `fail` / a free-text "worked, but …" phrase in the owner's own words (tri-state per D-09), plus the exact literal phrase `not exercised by the owner; undo path proven only by the program's own tests` for journey 10 if no bad learned change arises (D-19), and plain `not exercised` for journey 5 if nothing breaks (D-17). Add one closing block for the overall verdict (D-09's "feels like Aether again and I trust it to operate" — yes/no), which has no precedent in prior `*-UAT.md` files — model it as a final `### Overall Verdict` block with the same `expected:`/`result:` shape rather than inventing new frontmatter keys.

## Shared Patterns

### Coverage-JSON schema consistency (PROOF-02 / avoid a second vocabulary)
**Source:** `cmd/classic_coverage_199_test.go` field names (`type`, `id`, `disposition`, `modern_home`, `plan_task`, `artifact`, `proofs`, `status`, `historical_evidence`)
**Apply to:** all six new/extended `{N}-CLASSIC-COVERAGE.json` files and their validator tests. Never paraphrase field names, even though the 200-204 synthesis docs' own "CAP row routing table" prose columns use different, informal names (`Ledger hypothesis`, `Independently confirmed disposition`, `Evidence`) — that prose is narrative justification only; the machine-signed row must use the exact schema above.

### AST-based proof resolution (never hand-type a "known good" test list)
**Source:** `buildClassicCoverage199ProofIndex` (`cmd/classic_coverage_199_test.go:279-330`)
**Apply to:** every phase's proof-resolution validator — a `proofs` entry must resolve against a real parsed `*ast.FuncDecl` or a real `cases.json` id, never a string that merely looks like a test name.

### Handoff schema single-source append
**Source:** `codex.HandoffFieldsSummary` / `codex.HandoffOpenDecisionsGuidance` (used identically by `cmd/codex_continue_plan.go:332` and `cmd/codex_build.go:4704`)
**Apply to:** the seal brief fix (Defect 1) — reuse the same two constants and the same `fmt.Sprintf` shape rather than writing new handoff-schema prose for seal.

### Wrapper-runtime contract (fixes must land in Go/YAML source, never a lone `.claude/` edit)
**Source:** CLAUDE.md "Wrapper-Runtime Contract"; `.aether/commands/*.yaml` → generated `.claude/`/`.opencode/` mirrors
**Apply to:** Defects 1, 3, 5 — any wrapper-text change must originate in `.aether/commands/seal.yaml`/`entomb.yaml`, with `.claude/commands/ant/*.md` and `.opencode/commands/ant/*.md` kept byte-identical per the wrapper-triplet convention (project memory: "three hand-edited copies per command, parity test-enforced, no generator; edit all three identically").

### Content sanitization for anything promoted into an owner-facing file
**Source:** `pkg/colony/sanitize.go` `SanitizeSignalContent` / `DetectPromptIntegrityFindings`
**Apply to:** Defect 6's `sanitizeQueenInline` fix — compose with this existing detector rather than writing a new regex list from scratch.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `205-BRIEF.md` (one-page owner brief, D-18) | doc | request-response (owner-read, no command names) | No prior Aether artifact is "one page, task-only, no commands" — nearest kin (UAT frontmatter, field reports) both name commands/tests explicitly. Per RESEARCH.md's Open Question 2, default to plain planning-doc markdown; build fresh rather than force-fitting an existing template. |
| `205-LIMITATIONS-CARD.md` (D-15) | doc | batch (WINDOWS.md transcription) | No prior "limitations card" artifact exists in this repo; closest source is `.planning/WINDOWS.md` itself (to transcribe from, not to pattern the document's shape on) — build fresh, staying in planning records per D-16, and skip voice-corpus registration per RESEARCH.md Assumption A3 unless the planner decides to render it via CLI. |
| Platform honesty cards (OpenCode, Codex, D-13) | doc | request-response (one real run each) | No existing per-platform "honest capability card" exists; closest analog in tone only is `.planning/field-reports/2026-09-14-cosmic-seal-entomb-lifecycle.md` (plain-English findings format) — build fresh from an actual single run of each platform's real entry points. |

## Metadata

**Analog search scope:** `cmd/*_test.go`, `cmd/seal_*.go`, `cmd/entomb_cmd.go`, `cmd/queen.go`, `cmd/codex_continue_plan.go`, `cmd/codex_visuals.go`, `pkg/colony/sanitize.go`, `.planning/phases/199-front-door-and-classic-contract/`, `.planning/phases/{200-204}-*/`
**Files scanned:** ~14 (all read directly this session or in the required-reading pass)
**Pattern extraction date:** 2026-09-15
