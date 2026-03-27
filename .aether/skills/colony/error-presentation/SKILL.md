---
name: error-presentation
description: Use when an operation fails, an error occurs, or a warning needs to be communicated to the user
type: colony
domains: [errors, ux, formatting, diagnostics]
agent_roles: [builder, watcher, scout]
priority: normal
version: "1.0"
---

# Error Presentation

## Purpose

Errors deserve the same visual care as successes. When something goes wrong, the user should see a clear, formatted explanation -- not a raw stack trace or cryptic error code. This skill ensures error output matches the quality of success output.

## Error Output Format

Every error must follow this four-part structure:

### 1. Error Banner

Use the standard spaced-letter banner format:

```
━━ E R R O R ━━
```

For warnings (non-fatal issues):

```
━━ W A R N I N G ━━
```

### 2. Plain-English Description

Explain what went wrong in one or two sentences that a non-technical person could understand.

Good: "The build failed because a test file could not be found."
Bad: "ENOENT: no such file or directory, open '/path/to/test.spec.js'"

Never show raw error messages, stack traces, file paths, or error codes to the user. If technical details are needed for debugging, include them in a collapsed or secondary section, not the main message.

### 3. What Is Being Done

Tell the user what the system is doing about the error:

- "Retrying with an alternative approach..."
- "Logging this failure for colony learning."
- "Rolling back to the last known good state."

If nothing can be done automatically, say so clearly: "This requires manual intervention."

### 4. User Action

End with specific, actionable steps the user can take:

```
What you can do:
- Run /ant:status to see the current state
- Run /ant:build 3 to retry this phase
- Run /ant:flag "blocked on X" to mark a blocker
```

Always provide at least one actionable command. Never end an error with just "Something went wrong."

## Error Grouping

When multiple related errors occur (e.g., 5 tests fail in the same module), group them:

```
━━ E R R O R ━━
3 tests failed in the authentication module:
- Login with expired token
- Password reset flow
- Session refresh

Common cause: The auth service mock is not configured.
```

Never dump errors one by one when they share a root cause. Grouping helps the user see the pattern.

## Error Severity Levels

| Level | Banner | When |
|-------|--------|------|
| Error | `━━ E R R O R ━━` | Operation failed, cannot proceed |
| Warning | `━━ W A R N I N G ━━` | Something is off but work continues |
| Notice | `━━ N O T I C E ━━` | Informational, no action needed |

Use the right level. Do not call everything an error -- warnings and notices exist for a reason.

## Anti-Patterns

- Showing raw JSON error responses to the user.
- Displaying file paths without explaining what the file is.
- Using technical jargon: "mutex deadlock", "race condition", "null pointer".
- Ending error output without a Next Up or action block.
- Formatting errors as plain text when successes get rich formatting.
