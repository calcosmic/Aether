## Command Reference

Aether provides 45 slash commands organized into seven categories. Each command is invoked via `/ant:<name>` in your Claude session. This section is a complete quick-reference for every command, including syntax, description, and key options.

---

### Setup and Getting Started

These commands set up, initialize, and drive the core colony workflow from first use through phase completion.

| Command | Description |
|---------|-------------|
| `/ant:lay-eggs` | Set up Aether in this repo -- creates `.aether/` with all system files, templates, and utilities. Run once per repo. |
| `/ant:init "<goal>"` | Initialize a colony with a goal. Scans the repo, generates a charter for approval, creates colony state. Supports `--no-visual`. |
| `/ant:colonize` | Survey the codebase with 4 parallel scouts, producing 7 territory documents (PROVISIONS, TRAILS, BLUEPRINT, CHAMBERS, DISCIPLINES, SENTINEL-PROTOCOLS, PATHOGENS). Flags: `--no-visual`, `--force-resurvey`. |
| `/ant:plan` | Generate or display a project plan. Uses an iterative research loop (scout + planner per iteration) to reach a confidence target. Flags: `--fast`, `--balanced`, `--deep`, `--exhaustive`, `--target <N>`, `--max-iterations <N>`, `--accept`, `--no-visual`. |
| `/ant:build <phase>` | Execute a phase with parallel workers. Loads and runs 5 build playbooks sequentially (prep, context, wave, verify, complete). Self-organizing emergence. |
| `/ant:continue` | Verify completed build, reconcile state, and advance to the next phase. Runs 4 continue playbooks (verify, gates, advance, finalize). Enforces quality gates. |
| `/ant:run` | Autopilot mode -- chains build and continue across multiple phases automatically. Pauses on failures, blockers, or replan triggers. Flags: `--max-phases N`, `--replan-interval N`, `--continue`, `--dry-run`, `--headless`, `--verbose`. |

---

### Pheromone Signals

Pheromones are the colony's guidance system. They inject signals that workers sense and respond to, without hard-coding instructions. Signals decay over time: FOCUS 30 days, REDIRECT 60 days, FEEDBACK 90 days.

| Command | Description |
|---------|-------------|
| `/ant:focus "<area>"` | Emit a FOCUS signal to guide colony attention toward an area. Priority: normal. Strength: 0.8. Flag: `--ttl <value>` (default: `phase_end`). |
| `/ant:redirect "<pattern>"` | Emit a REDIRECT signal to warn the colony away from a pattern. Priority: high. Strength: 0.9. Flag: `--ttl <value>` (default: `phase_end`). |
| `/ant:feedback "<note>"` | Emit a FEEDBACK signal with gentle guidance. Priority: low. Strength: 0.7. Creates a colony instinct. Flag: `--ttl <value>` (default: `phase_end`). |
| `/ant:pheromones [subcommand]` | View and manage active pheromone signals. Subcommands: `all` (default), `focus`, `redirect`, `feedback`, `clear`, `expire <id>`. Flag: `--no-visual`. |
| `/ant:export-signals [path]` | Export colony pheromone signals to portable XML format. Default output: `.aether/exchange/pheromones.xml`. Requires `xmllint`. |
| `/ant:import-signals <file> [colony]` | Import pheromone signals from another colony's XML export. Second argument is an optional colony name prefix to prevent ID collisions. Requires `xmllint`. |

---

### Status and Monitoring

These commands provide visibility into colony state, phase progress, flags, event history, and live activity.

| Command | Description |
|---------|-------------|
| `/ant:status` | Colony dashboard at a glance -- goal, phase/task progress bars, focus and constraint counts, instincts, flags, milestone, vital signs, memory health, pheromone summary, data safety. |
| `/ant:phase [N\|list]` | View phase details (tasks, dependencies, success criteria) for a specific phase by number, or `list`/`all` for a summary of all phases grouped by status. |
| `/ant:flags` | List project flags (blockers, issues, notes). Flags: `--all`, `--type <blocker\|issue\|note>`, `--phase N`, `--resolve <id> "<msg>"`, `--ack <id>`. |
| `/ant:flag "<title>"` | Create a new flag. Flags: `--type <blocker\|issue\|note>` (default: `issue`), `--phase N`. Blockers prevent phase advancement until resolved. |
| `/ant:history` | Browse colony event history. Flags: `--type <TYPE>`, `--since <DATE>`, `--until <DATE>`, `--limit N` (default: 10). Dates accept ISO format or relative values like `1d`, `2h`. |
| `/ant:watch` | Set up a tmux session with a 4-pane live dashboard (status, progress, spawn tree, activity log). Requires tmux. |
| `/ant:maturity` | View colony maturity journey through 6 milestones (First Mound, Open Chambers, Brood Stable, Ventilated Nest, Sealed Chambers, Crowned Anthill) with ASCII art anthill and progress bar. |
| `/ant:memory-details` | Drill-down view of colony memory -- wisdom entries by category from QUEEN.md, pending promotions, deferred proposals, and recent failures from the midden. |

---

### Session Management

Manage session state for handoff between conversations, so you can safely `/clear` and resume later.

| Command | Description |
|---------|-------------|
| `/ant:pause-colony` | Save colony state and create a handoff document at `.aether/HANDOFF.md`. Optionally suggests committing uncommitted work. Flag: `--no-visual`. |
| `/ant:resume-colony` | Full session restore from pause -- loads state, displays pheromones with strength bars, phase progress, survey freshness, and handoff context. Clears paused state and removes HANDOFF.md. Flag: `--no-visual`. |
| `/ant:resume` | Quick session restore after `/clear` or new session. Detects codebase drift, computes next-step guidance, and displays a compact dashboard with memory health. Includes blocking guards for missing plans or interrupted builds. |

---

### Lifecycle

These commands manage the beginning and end of a colony's life.

| Command | Description |
|---------|-------------|
| `/ant:seal` | Seal the colony with the Crowned Anthill milestone ceremony. Promotes colony wisdom to QUEEN.md, spawns a Sage for analytics, a Chronicler for documentation audit, exports XML archives, and writes CROWNED-ANTHILL.md. Flags: `--no-visual`. |
| `/ant:entomb` | Archive a sealed colony into `.aether/chambers/`. Requires the colony to be sealed first. Copies all colony data, exports XML archives, records in eternal memory, and resets colony state for a fresh start. Flag: `--no-visual`. |
| `/ant:update` | Update Aether system files from the global hub. Uses a transactional updater with checkpoint creation, safe sync, and automatic rollback on failure. Flag: `--force`. |

---

### Advanced

Power-user commands for deep research, philosophical exploration, resilience testing, and specialized analysis.

| Command | Description |
|---------|-------------|
| `/ant:swarm "<bug>"` | Deploy 4 parallel scouts (Archaeologist, Pattern Hunter, Error Analyst, Web Researcher) to investigate and fix stubborn bugs. Cross-compares findings, ranks solutions by confidence, applies the best fix, and auto-rolls back on failure. No arguments shows a real-time swarm display. |
| `/ant:oracle` | Deep research agent using an iterative RALF loop. Guided by a research wizard (topic, template, depth, confidence, scope, strategy). Subcommands: `stop`, `status`, `promote`. Flags: `--force-research`, `--no-visual`. |
| `/ant:dream` | The Dreamer -- a philosophical wanderer that observes the codebase and writes 5-8 dream observations to `.aether/dreams/`. Categories: musing, observation, concern, emergence, archaeology, prophecy, undercurrent. May suggest pheromones. Flag: `--no-visual`. |
| `/ant:interpret [date]` | The Interpreter -- grounds dreams in reality by validating each dream observation against the actual codebase. Rates each dream as confirmed, partially confirmed, unconfirmed, or refuted. Can inject pheromones or add items to TO-DOS based on findings. |
| `/ant:chaos <target>` | The Chaos Ant -- resilience tester that probes 5 categories (edge cases, boundary conditions, error handling, state corruption, unexpected inputs) for a given file, module, or feature. Produces a structured report with severity ratings and reproduction steps. Auto-creates blocker flags for critical/high findings. Flag: `--no-visual`. |
| `/ant:archaeology <path>` | The Archaeologist -- git historian that excavates commit history for a file or directory. Analyzes authorship, churn, tech debt markers, dead code candidates, and stability. Produces a full archaeology report with tribal knowledge extraction. Flag: `--no-visual`. |
| `/ant:organize` | Codebase hygiene report -- spawns an archivist to scan for stale files, dead code patterns, and orphaned configs. Report-only (no files modified). Output saved to `.aether/data/hygiene-report.md`. Flag: `--no-visual`. |
| `/ant:council` | Convene a council for intent clarification. Presents multi-choice questions about project direction, quality priorities, or constraints, then translates answers into FOCUS, REDIRECT, and FEEDBACK pheromone signals. Supports `--deliberate "<proposal>"` mode for Advocate/Challenger/Sage structured debate. Flag: `--no-visual`. |

---

### Utilities

Maintenance, introspection, and convenience commands for operating the colony system.

| Command | Description |
|---------|-------------|
| `/ant:data-clean` | Scan and remove test/synthetic artifacts from colony data files (pheromones.json, QUEEN.md, learning-observations.json, midden.json, spawn-tree.txt, constraints.json). Runs a dry-run first, then asks for confirmation. |
| `/ant:help` | Display the full system overview -- all commands by category, typical workflow, worker castes, and how the colony lifecycle, pheromone system, and colony memory work. |
| `/ant:insert-phase` | Insert a corrective phase into the active plan after the current phase. Collects a brief description of what is not working and what the fix should accomplish. |
| `/ant:migrate-state` | One-time migration from v1 (6-file) state format to v2.0 (consolidated single-file) format. Creates backups in `.aether/data/backup-v1/`. Safe to run multiple times. |
| `/ant:patrol` | Comprehensive pre-seal colony audit -- verifies plan vs codebase reality, documentation accuracy, unresolved issues, test coverage, and colony health. Produces a completion report at `.aether/data/completion-report.md` with a seal-readiness recommendation. Flag: `--no-visual`. |
| `/ant:preferences` | Add or list user preferences in the hub `~/.aether/QUEEN.md`. Use `--list` to view current preferences, or provide text to add a new one. |
| `/ant:quick "<question>"` | Fast scout query for quick answers about the codebase without build ceremony. Spawns a single scout agent. No state changes, no verification, no checkpoints. |
| `/ant:run` | Autopilot mode -- see Setup and Getting Started above. Listed in both categories because it is the primary execution driver. |
| `/ant:skill-create "<topic>"` | Create a custom domain skill via Oracle mini-research and a guided wizard. Researches best practices, asks about focus area and experience level, then generates a SKILL.md in `~/.aether/skills/domain/`. |
| `/ant:tunnels [chamber] [chamber2]` | Browse archived colonies in `.aether/chambers/`. No arguments shows a timeline. One argument shows the seal document for that chamber. Two arguments compares chambers side-by-side with growth metrics and pheromone trail diffs. |
| `/ant:verify-castes` | Display the colony caste system -- all 24 castes with their model slot assignments (opus, sonnet, inherit), system status (utils, proxy health), and current model configuration. |
