# Phase 110: Go Safety Invariant Verification - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-12
**Phase:** 110-Go Safety Invariant Verification
**Areas discussed:** Validation Strictness, Concurrency Testing, Test Organization

---

## Validation Strictness

### Safety check approach

| Option | Description | Selected |
|--------|-------------|----------|
| Verify existing | Write tests proving Go's existing finalizers, atomic writes, and locking prevent TS from writing state. No new runtime guards. | ✓ |
| Add active guards | Add runtime guards that actively detect and reject non-Go state writes — like a watchdog. | |
| Verify first, guard if needed | Start with verifying existing code, then add guards only if gaps found. | |

**User's choice:** Verify existing (recommended)
**Notes:** Trust Go's existing code and just prove it holds. No new guard infrastructure.

### Verification focus

| Option | Description | Selected |
|--------|-------------|----------|
| Manifest validation | Verify each finalizer rejects malformed manifests (missing fields, bad provenance). Tests Go won't commit garbage. | ✓ |
| Boundary enforcement | Verify TS host literally cannot write to .aether/data/. Tests the boundary itself. | |
| Both | Manifest validation and boundary enforcement — thorough but more test code. | |

**User's choice:** Manifest validation (recommended)
**Notes:** Focus on what Go does with the data TS sends — reject bad manifests.

### Manifest corruption scope

| Option | Description | Selected |
|--------|-------------|----------|
| Common corruption | Missing phase number, invalid version, no provenance timestamp, empty worker list. Covers likely integration bugs. | ✓ |
| Include adversarial | Also test: extremely large manifests, deeply nested JSON, Unicode injection, future-version manifests. | |
| Claude's discretion | Builder picks the right level based on what the finalizer code actually checks. | |

**User's choice:** Common corruption (recommended)
**Notes:** Cover realistic TS→Go integration bugs, not adversarial attacks.

### Test structure per finalizer

| Option | Description | Selected |
|--------|-------------|----------|
| Per-finalizer tests | Each plan, build, continue finalizer gets its own set of validation tests. Clean separation. | ✓ |
| Shared test matrix | One test function running all finalizers through the same validation matrix. Less code but harder to pinpoint failures. | |
| Claude's discretion | Pick based on how existing finalizer tests are organized. | |

**User's choice:** Per-finalizer tests (recommended)

---

## Concurrency Testing

### Concurrency scope

| Option | Description | Selected |
|--------|-------------|----------|
| Normal flow only | Run plan→build→continue through TS host and verify state is correct. Proves integration works. | ✓ |
| Include stress scenarios | Also test concurrent Go+TS finalizer calls, concurrent reads/writes, multiple TS hosts in parallel. | |
| Claude's discretion | Pick scenarios based on what's realistic (TS host is a prototype, likely one at a time). | |

**User's choice:** Normal flow only (recommended)

### Test reuse strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse Phase 108 tests | Run golden workflow tests driven through TS host. If same state transitions happen, invariants hold. | ✓ |
| New dedicated tests | Write new integration tests specifically for TS host→Go lifecycle. More control but duplicates coverage. | |
| Reuse + TS assertions | Run Phase 108 golden tests AND add TS-specific assertions (spawn-log entries, etc.). | |

**User's choice:** Reuse Phase 108 tests (recommended)

### Install/update/publish purity

| Option | Description | Selected |
|--------|-------------|----------|
| Smoke test | Verify install/update/publish have zero code path overlap with TS host. Simple grep + functional test. | ✓ |
| Full tests | Full test suite for install, update, publish with TS host files present. | |
| Claude's discretion | These commands are already pure Go and have never touched TS. | |

**User's choice:** Smoke test install/update (recommended)

---

## Test Organization

### File structure

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated file | One new cmd/safety_invariant_test.go covering all 6 success criteria. Easy to find, clear purpose. | ✓ |
| Extend existing files | Add safety tests to finality_parity_test.go, boundary_contract_test.go, etc. Consolidated but harder to see together. | |
| Claude's discretion | Pick based on what's cleanest. | |

**User's choice:** Dedicated file (recommended)

### Test organization within file

| Option | Description | Selected |
|--------|-------------|----------|
| Per-criterion tests | One TestX per success criterion. 6 tests mapping to 6 criteria. Clear traceability. | ✓ |
| Grouped tests | Fewer, broader tests covering multiple criteria. Less code but harder to identify which invariant failed. | |
| Claude's discretion | Organize in whatever way maps cleanly to the codebase. | |

**User's choice:** Per-criterion tests (recommended)

---

## Claude's Discretion

- Exact test implementation patterns (table-driven, sequential, etc.)
- Which golden tests to reuse and how to adapt for TS host execution
- How to structure the install/update/publish purity check (grep vs Go AST analysis vs import check)
- Error message expectations for rejected manifests
- Whether to use existing test helpers (setupBuildFlowTest, createTestColonyState) or write new ones

## Deferred Ideas

None — discussion stayed within phase scope
