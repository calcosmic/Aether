---
colony_prime_version: "1.0"
section_templates:
  state:
    header: "## Colony State\n\n"
    goal_format: "Goal: %s\n"
    state_format: "State: %s\n"
    phase_format: "Phase: %d\n"
    phase_name_format: "Phase Name: %s\n"
    task_format: "  - [%s] %s\n"
    tasks_header: "Tasks:\n"
    parallel_mode_format: "Parallel Mode: %s\n"
  review_depth:
    header: "## Review Depth\n\n"
    light_text: "Light review -- core verification only"
    standard_text: "Standard review -- watcher and probe verification"
    heavy_text: "Heavy review -- full quality gauntlet"
    default_text: "Standard review -- watcher and probe verification"
  pheromones:
    header: "## Pheromone Signals\n\n"
    lifecycle_context: ""
    signal_format: "- [%s] %s\n"
  instincts:
    header: "## Active Instincts\n\n"
    instinct_format: "- [%s] %s (confidence: %.2f)\n"
  decisions:
    header: "## Key Decisions\n\n"
    decision_format: "- Phase %d: %s — %s\n"
  learnings:
    header: "## Phase Learnings\n\n"
    phase_header_format: "### Phase %d: %s\n"
    learning_format: "  - %s [%s]\n"
  worker_handoffs:
    header: "## Previous Worker Handoffs\n\n"
    worker_header_format: "### %s\n"
    status_format: "- Status: %s; verification: %s\n"
    summary_format: "- Summary: %s\n"
    list_format: "- %s: %s\n"
  hive_wisdom:
    header: "## HIVE WISDOM (Cross-Colony Patterns)\n\n"
    entry_format: "- %s\n"
  learned_memory:
    header: "## LEARNED MEMORY (Verified Outcomes)\n\n"
    entry_format: "- [Phase %d] %s (confidence: %.0f%%, classification: %s)\n"
  global_queen_md:
    header: "## GLOBAL QUEEN WISDOM (Cross-Colony)\n\n"
    entry_format: "- %s\n"
  user_preferences:
    header: "## USER PREFERENCES\n\n"
    entry_format: "- %s\n"
  prior_reviews:
    header: "## Prior Reviews\n\n"
    domain_format: "- %s (%d open): %s\n"
    domain_count_only_format: "- %s (%d open)\n"
  local_queen_wisdom:
    header: "## LOCAL QUEEN WISDOM (Repo-Specific)\n\n"
    entry_format: "- %s\n"
  clarified_intent:
    header: "## CLARIFIED INTENT\n\n"
  blockers:
    header: "## Active Blockers\n\n"
    blocker_format: "- %s\n"
  medic_health:
    header: "## Colony Health Issues\n\n"
    scan_timestamp_format: "Last scan: %s\n\n"
    issue_format: "- [%s] %s"
    issue_file_format: " (%s)"
---

# Colony Prime Prompt Templates

This file contains the editable prompt section templates used by `colony-prime` when assembling worker context. Each section has a header and optional format strings that control how colony state, signals, instincts, and other data are rendered into the prompt.

## How to Edit Safely

- Keep YAML frontmatter valid. The Go runtime parses everything between the first `---` and the closing `---`.
- Do not remove required fields (e.g., `header`) from a section unless you intend to fall back to the hardcoded default.
- Format strings use Go `fmt.Sprintf` verbs (e.g., `%s`, `%d`, `%.2f`). Keep them compatible.
- If the file is missing or malformed, the runtime falls back to hardcoded templates built into the binary.

## Sections

| Section | Description |
|---------|-------------|
| `state` | Colony goal, state, phase, tasks, parallel mode |
| `review_depth` | Verification depth label (light / standard / heavy) |
| `pheromones` | Active pheromone signals (FOCUS, REDIRECT, FEEDBACK) |
| `instincts` | Learned instincts with confidence scores |
| `decisions` | Key decisions recorded per phase |
| `learnings` | Phase-scoped learnings |
| `worker_handoffs` | Previous worker handoff records |
| `hive_wisdom` | Cross-colony wisdom from the Hive Brain |
| `learned_memory` | Verified durable learning entries |
| `global_queen_md` | Cross-colony QUEEN.md wisdom |
| `user_preferences` | User preference lines from QUEEN.md |
| `prior_reviews` | Open review findings from domain ledgers |
| `local_queen_wisdom` | Repo-specific QUEEN.md wisdom |
| `clarified_intent` | Clarified intent lines from pending decisions |
| `blockers` | Active blocker flags |
| `medic_health` | Critical colony health issues from last medic scan |
