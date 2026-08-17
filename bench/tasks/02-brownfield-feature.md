# Task 02: Brownfield feature touching several files

## Substrate

`zod` (TypeScript), pinned commit `ca42965df46b2f7e2747db29c40a26bcb32a51d5`
(tag `v3.23.8`; see `bench/tasks/substrate.md` for the clone command and
neutrality evidence).

## The task

**This prompt text is handed to all three lanes verbatim. It is written the
way a real developer would be briefed, and contains none of either system's
internal vocabulary.**

> Add a new string-validation rule called `lowercase()` to this library's
> string schema. When a user calls `.lowercase()` on a string schema, the
> resulting schema should only accept strings that are already entirely
> lowercase (a string equal to its own `.toLowerCase()` result counts as
> lowercase; an empty string counts as lowercase). Calling `.parse()` or
> `.safeParse()` with a string containing any uppercase letter should fail
> validation with a clear message saying the input must be lowercase.
> Follow the existing pattern this library already uses for its other
> simple string checks (for example how it already validates a string is
> trimmed, or matches a fixed prefix) so the new rule behaves consistently
> with the rest of the library — same error code, same options for a custom
> message, same way of composing with other checks like `.min()` on the
> same schema.

## Setup

None. The harness clones the substrate repository at its pinned commit and
runs `npm install` once; the system under test starts from an unmodified
checkout.

## Done means

A correct solution requires real edits in at least three existing files,
because the feature does not exist yet at any single point in the codebase
— it must be defined as a new validation-check kind, registered in the
library's issue-reporting type, and given a human-readable failure message:

- `src/types.ts` gains a new check kind for the string schema (alongside the
  existing check kinds like `trim`, `toLowerCase`, `startsWith`), the
  validation logic that runs it during parsing, and a public
  `.lowercase(...)` builder method on the string schema class that a caller
  can chain the same way they chain `.trim()` or `.min()` today.
- `src/ZodError.ts` gains the new check's name to the union type that lists
  every possible string-validation kind an issue can report (the same union
  that already lists `"email"`, `"trim"`, and the other existing kinds).
- `src/locales/en.ts` gains a case producing a specific, readable message
  for the new failure (not the generic fallback message every unhandled
  case gets), matching the style of the messages already written for the
  library's other string-validation failures.

Observable end state, checkable without asking either system whether it
succeeded:
- The full existing test suite (`npm test`) still exits 0 — no pre-existing
  test broke.
- A schema built by calling `.lowercase()` on a string schema rejects a
  string containing an uppercase character and accepts a string with none.
- `git diff --name-only` against the pinned commit shows changes in all
  three files named above (not just one of them).

## Allowlist

`bench/tasks/allowlists/02-brownfield-feature.txt`

## Per-lane invocation

- **GSD:** `/gsd-execute-phase` against a plan produced for this feature,
  followed by `/gsd-verify-work`.
- **Aether interactive:** `/ant-build <phase>` against a phase describing
  this feature, followed by `/ant-continue`.
- **Aether autopilot:** `/ant-run` against a colony initialized with this
  task as its goal.
