# Planning Discipline

## Purpose

Write comprehensive implementation plans with bite-sized tasks, assuming workers have zero context. Each task should be one action (2-5 minutes of work).

## Core Principles

- **DRY** - Don't Repeat Yourself
- **YAGNI** - You Aren't Gonna Need It
- **TDD** - Test-Driven Development (test first, always)
- **Frequent Commits** - One commit per feature/fix

## Bite-Sized Task Granularity

**Each step is ONE action:**

```
"Write the failing test" - step
"Run it to make sure it fails" - step
"Implement the minimal code to make the test pass" - step
"Run the tests and make sure they pass" - step
"Commit" - step
```

**NOT acceptable:**
- "Implement authentication with tests" (too big)
- "Add validation" (too vague)
- "Set up the project" (needs breakdown)

## Task Structure

Each task should include:

### 1. Files Section

```markdown
**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`
```

Always use exact paths. No "somewhere in src/" ambiguity.

### 2. Steps with TDD Flow

```markdown
**Step 1: Write the failing test**

\`\`\`python
def test_specific_behavior():
    result = function(input)
    assert result == expected
\`\`\`

**Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

**Step 3: Write minimal implementation**

\`\`\`python
def function(input):
    return expected
\`\`\`

**Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

**Step 5: Commit**

\`\`\`bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
\`\`\`
```

### 3. Expected Output

For every command, specify expected output:

```markdown
Run: `npm test`
Expected: 47 passing, 0 failing
```

Not just "run tests" - specify what success looks like.

## Phase Structure

```markdown
### Phase N: [Component Name]

**Goal:** [One sentence describing what this phase achieves]

**Dependencies:** [What must exist before this phase]

**Tasks:**

#### Task N.1: [Specific action]

**Files:**
- Create: `src/components/Button.tsx`
- Test: `tests/components/Button.test.tsx`

**Steps:**
(TDD steps as above)

**Success Criteria:**
- Tests pass for Button component
- Component renders without errors
```

## Quality Checks

Before a plan is complete, verify:

1. **Exact paths** - Every file reference is complete path
2. **Complete code** - No "add appropriate code" placeholders
3. **Expected outputs** - Every command has expected result
4. **TDD flow** - Test before implementation for each feature
5. **No assumptions** - Worker could execute with zero context
6. **Commit points** - Clear where to commit

## Integration with Colony

The Route-Setter Ant follows this discipline when generating plans.

Each task in `plan.phases[N].tasks` should be:
- Executable in 2-5 minutes
- Self-contained with file paths
- Verifiable with specific success criteria

## Red Flags

**Never in a plan:**
- "Add tests" (which tests? for what?)
- "Implement feature" (what specifically?)
- "Update file" (which file? what changes?)
- "Handle edge cases" (which cases?)

**Always in a plan:**
- Exact file paths
- Complete code snippets
- Exact commands to run
- Expected outputs/results
- One action per step

## Why This Matters

- Workers can execute without asking questions
- Progress is measurable (step by step)
- Debugging is easier (clear steps to trace)
- Commits are atomic and meaningful
- Quality is built in (TDD from start)
