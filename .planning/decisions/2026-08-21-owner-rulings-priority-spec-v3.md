# Owner Rulings — Priority Implementation Spec v3 (2026-08-21)

Source: the ten decisions (D1–D10) requested by
`.aether/dreams/AETHER_PRIORITY_SPEC_V3_ANALYSIS_2026-08-21.md` §26.
D1 and D2 were ruled directly by the owner this evening; D3–D10 record the
audit's recommended answers, adopted as defaults the owner may reverse at any
time.

## Ruled by the owner

- **D1 — Document authority: "The spec should have an order of importance."**
  v3 is neither frozen until Phase 192 nor adopted verbatim as a master plan.
  It becomes a priority-ordered backlog, merged with the remaining approved
  programme (185, 186-07, 192) into one ranked list the owner blesses and
  work proceeds top-down. The Phase-192 stop rule is superseded by that
  ranked list.

- **D2 — Automatic model routing: APPROVED (reverses the prior rejection).**
  The owner explicitly chose "allow it automatically": the system may route
  routine work to cheaper models on its own. Engineering sequencing: the
  honest cost line (Phase 185) lands first so routing decisions are measured
  and visible; routing always displays its reasoning per dispatch and remains
  overridable per build. Prior ruling ("no automatic model selection",
  recorded pre-2026-08-21) is superseded.

## Recommended answers adopted as defaults (reversible)

- **D3 — Strategy authority:** the Queen/host *proposes*; the Go runtime
  validates, persists, and finalizes. No host-side state ownership.
- **D4 — Safety precedence:** runtime safety/security/capability constraints
  outrank SPEC, preferences, Skills, and learned policy. Unconditional.
- **D5 — Canonical command identity:** platform-neutral action ID with
  per-host projections and aliases; slash spellings are projections.
- **D6 — No-change semantics:** result status `completed_no_change`, with
  disposition `verified_existing` when existing behavior was proven. One
  terminal state machine, not two.
- **D7 — Quota status semantics:** worker result `interrupted` (with reason);
  workstream/colony budget state `SUSPENDED`.
- **D8 — Review taxonomy:** policy axis `none|deterministic_only|final_only|adaptive`;
  depth axis `quick|light|standard|heavy|adversarial`. Independent axes.
- **D9 — Legacy colonies:** compatibility read/continue plus an explicit
  generated draft-SPEC proposal; never silently approve one.
- **D10 — Artifact retention/security:** content hashes, restrictive local
  permissions, configurable retention, secret scanning/redaction before
  persistence; never commit subscription transcripts by default.
