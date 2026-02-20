# Verification Discipline

## The Iron Law

```
NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE
```

If you haven't run the verification command in this message, you cannot claim it passes.

## The Gate Function

```
BEFORE claiming any status or expressing satisfaction:

1. IDENTIFY: What command proves this claim?
2. RUN: Execute the FULL command (fresh, complete)
3. READ: Full output, check exit code, count failures
4. VERIFY: Does output confirm the claim?
   - If NO: State actual status with evidence
   - If YES: State claim WITH evidence
5. ONLY THEN: Make the claim

Skip any step = lying, not verifying
```

## Common Failures

| Claim | Requires | Not Sufficient |
|-------|----------|----------------|
| Tests pass | Test command output: 0 failures | Previous run, "should pass" |
| Build succeeds | Build command: exit 0 | Linter passing, logs look good |
| Bug fixed | Test original symptom: passes | Code changed, assumed fixed |
| Task complete | Success criteria verified | "I wrote the code" |
| Phase ready | All tasks verified | Prime Worker reports success |

## Red Flags - STOP

When you catch yourself:
- Using "should", "probably", "seems to"
- Expressing satisfaction before verification ("Great!", "Done!")
- About to report completion without running checks
- Trusting spawn reports without independent verification
- Relying on partial verification
- **ANY wording implying success without having run verification**

## Rationalization Prevention

| Excuse | Reality |
|--------|---------|
| "Should work now" | RUN the verification |
| "I'm confident" | Confidence ≠ evidence |
| "Just this once" | No exceptions |
| "Spawn said success" | Verify independently |
| "Partial check is enough" | Partial proves nothing |

## Verification Patterns

**Tests:**
```
✅ [Run test command] → [See: 34/34 pass] → "All tests pass"
❌ "Should pass now" / "Looks correct"
```

**Build:**
```
✅ [Run build] → [See: exit 0] → "Build succeeds"
❌ "Code compiles" (without running build)
```

**Task Completion:**
```
✅ Re-read success criteria → Verify each with evidence → Report with proof
❌ "Tasks done, phase complete"
```

**Spawn Verification:**
```
✅ Spawn reports success → Check files exist → Run tests → Report actual state
❌ Trust spawn report blindly
```

## Phase Advancement Gate

Before `/ant:continue` advances to next phase:

1. **Build Check**: Run project build command (if exists)
2. **Test Check**: Run test suite, capture pass/fail counts
3. **Success Criteria**: Line-by-line verification of phase criteria
4. **Evidence Required**: Each criterion needs specific proof

```
Phase {N} Verification
======================

Build: {command} → {exit code}
Tests: {command} → {pass}/{total} ({fail} failures)

Success Criteria:
  ✅ Criterion 1: {evidence}
  ✅ Criterion 2: {evidence}
  ❌ Criterion 3: {what's missing}

Verdict: PASS / FAIL
```

If ANY criterion fails verification, phase cannot advance.

## Why This Matters

- False completion claims break trust
- Unverified code ships bugs
- Time wasted on rework from premature advancement
- Colony integrity depends on honest reporting

**Evidence before claims, always.**
