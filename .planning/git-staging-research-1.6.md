## 6. Commit Message Conventions for Colony Commits

### Current State

The Aether colony currently uses a single hardcoded commit message format for checkpoints:

```
aether-checkpoint: pre-phase-$PHASE_NUMBER
```

This is generated in `build.md` Step 3 via:
```bash
git add -A && git commit --allow-empty -m "aether-checkpoint: pre-phase-$PHASE_NUMBER"
```

There are no standardized formats for progress commits, milestone commits, or fix commits. These are left to whatever the AI agent or user produces ad hoc.

---

### Existing Commit Style Analysis

Analyzing the last 80 commits in the Aether repository reveals three distinct patterns:

**1. Conventional Commits (dominant pattern, ~60% of commits)**
```
feat: add emoji styling to all ant: commands
fix: correct copyright year in MIT License
docs: update README for v1.0.0 stable release
chore: prepare for v1.0.0 stable release
refactor(37-04): reduce aether-utils.sh from 317 to 85 lines
feat(ant): v2.0 - nested spawning, visual improvements, flagging system
docs(v5.1): complete milestone audit - all requirements satisfied
```

**2. Aether checkpoint format (~15% of commits)**
```
aether-checkpoint: pre-phase-1
aether-checkpoint: pre-phase-2
aether-checkpoint: pre-phase-3
checkpoint: pre-emoji-update
```

**3. Unstructured/ad hoc (~25% of commits)**
```
Update spawn output format: emoji adjacent to ant name
Match command output emojis to description styling
sync OpenCode commands with latest Claude Code versions
2.4.2
Remove disclaimer from README
```

**Key observations:**
- The user's stated preference is "imperative mood, under 72 characters, focused on why not what."
- Conventional commits with scoped phases (e.g., `feat(33-02):`) appeared heavily during structured development sprints.
- Checkpoint messages use `aether-checkpoint:` as a namespace prefix.
- Some commits use phase numbers as scopes (e.g., `refactor(33-04):`) making them traceable to specific build phases.
- Milestone commits (e.g., `chore: complete v5.1 milestone`) use standard conventional commit types.

---

### Industry Context: How AI Tools Handle Commit Messages

**Aider:** Uses Conventional Commits by default. Imperative mood. Customizable via `--commit-prompt`. Adds `Co-authored-by` trailer via `--attribute-co-authored-by`.

**Claude Code:** Follows whatever conventions are in CLAUDE.md or the project's git history. Adds `Co-Authored-By: Claude` trailer by default. No fixed format imposed.

**Cursor/Copilot:** Generates Conventional Commits style by default. Cursor uses staged diff analysis.

**Windsurf:** Single-click commit message generation. Conventional Commits format.

**Common theme:** The entire AI tooling ecosystem has converged on Conventional Commits as the default format, with imperative mood and optional scopes.

---

### Proposed Format Options

The Aether colony has four commit types that need distinct formats:
1. **Checkpoint** -- safety rollback point before a phase begins
2. **Progress** -- after a phase's work is done (code changes applied)
3. **Milestone** -- after a phase passes all verification gates
4. **Fix** -- after swarm resolves a bug

---

#### Option A: Conventional Commits with `aether` Scope

Adapts standard Conventional Commits by using `aether` as the scope namespace with a sub-scope for the phase.

| Commit Type | Format | Example |
|-------------|--------|---------|
| Checkpoint | `chore(aether): checkpoint pre-phase-N` | `chore(aether): checkpoint pre-phase-3` |
| Progress | `feat(aether-N): <description>` | `feat(aether-2): implement signal schema unification` |
| Milestone | `chore(aether): complete phase N — <name>` | `chore(aether): complete phase 3 — signal unification` |
| Fix | `fix(aether-N): <description>` | `fix(aether-2): resolve missing signal paths` |

**Strengths:**
- Fully compatible with Conventional Commits tooling (commitlint, standard-version, semantic-release)
- Machine-parseable with existing regex patterns
- Scoped phases enable `git log --grep="aether-2"` filtering
- Familiar to anyone who knows Conventional Commits

**Weaknesses:**
- Checkpoint commits being `chore` may feel wrong semantically; they are operational, not maintenance
- `aether-N` scope overloads the scope field (scope is usually a code module, not a workflow phase)
- Progress commits using `feat` may not always be features (could be refactors, docs, etc.)

**Variation:** Progress commits could use their natural type:
```
refactor(aether-2): consolidate signal paths
docs(aether-2): update architecture documentation
feat(aether-2): add checkpoint messaging
```

---

#### Option B: Namespaced Prefix Format

Uses `aether-<category>:` as a simple, consistent prefix. Prioritizes readability and colony identity.

| Commit Type | Format | Example |
|-------------|--------|---------|
| Checkpoint | `aether-checkpoint: pre-phase-N` | `aether-checkpoint: pre-phase-3` |
| Progress | `aether-progress: phase N — <description>` | `aether-progress: phase 2 — implement signal schema` |
| Milestone | `aether-milestone: phase N complete — <name>` | `aether-milestone: phase 3 complete — signal unification` |
| Fix | `aether-fix: phase N — <description>` | `aether-fix: phase 2 — resolve missing signal paths` |

**Strengths:**
- Immediately obvious which commits are colony-generated vs. human-authored
- Easy grep: `git log --grep="aether-checkpoint"`, `git log --grep="aether-milestone"`
- Already partially in use (`aether-checkpoint:` exists in the codebase)
- Very clean in `git log --oneline` output
- No ambiguity about commit type; the category IS the type

**Weaknesses:**
- Not compatible with Conventional Commits tooling (commitlint will reject these)
- Loses semantic versioning integration (no `feat`/`fix` for semver bumps)
- Creates a parallel convention that diverges from the rest of the project's commit style
- Does not encode whether the underlying work was a feature, refactor, docs, etc.

---

#### Option C: Hybrid — Conventional Commits with Colony Trailer

Uses standard Conventional Commits for the subject line, with a structured trailer block for colony metadata. This keeps the commit message compatible with all tooling while embedding colony context in the body.

| Commit Type | Format | Example |
|-------------|--------|---------|
| Checkpoint | `chore: aether checkpoint pre-phase-N` | `chore: aether checkpoint pre-phase-3` |
| Progress | `<type>: <description>` + trailer | `feat: implement signal schema unification` |
| Milestone | `chore: aether phase N complete` + trailer | `chore: aether phase 3 complete` |
| Fix | `fix: <description>` + trailer | `fix: resolve missing signal paths in build.md` |

With body/trailer:
```
feat: implement signal schema unification

Aether-Phase: 2
Aether-Type: progress
Aether-Build: 2024-07-15T10:30:00Z
Co-Authored-By: Claude <noreply@anthropic.com>
```

```
chore: aether checkpoint pre-phase-3

Aether-Phase: 3
Aether-Type: checkpoint
Aether-Ref: abc1234
```

**Strengths:**
- Fully Conventional Commits compatible; all tooling works
- Rich metadata in trailers (parseable with `git log --format="%(trailers)"`)
- Progress commits use their natural type (`feat`, `fix`, `refactor`, `docs`)
- Clean `git log --oneline` output matches the rest of the project
- Colony metadata is structured but hidden from the one-line view
- Git trailers are a first-class git feature, not a custom hack

**Weaknesses:**
- More complex to implement (multi-line commit messages)
- Checkpoint commits in `git log --oneline` look like normal `chore:` commits; need `--grep` to distinguish
- Trailers require multi-line `git commit -m` or heredoc syntax
- Developers unfamiliar with git trailers may not know to look for metadata

---

### Comparison Matrix

| Criterion | Option A: Conv. + Scope | Option B: Namespace Prefix | Option C: Conv. + Trailer |
|-----------|------------------------|---------------------------|--------------------------|
| **Readability** | Good — familiar format | Excellent — instantly clear | Good — clean one-line |
| **Parseability** | Excellent — standard regex | Good — simple grep | Excellent — trailers API |
| **git log cleanliness** | Good — consistent | Good — visually distinct | Excellent — blends in |
| **Conv. Commits compat** | Full | None | Full |
| **Semver integration** | Yes | No | Yes |
| **Colony identity** | Moderate — scope only | Strong — prefix is branded | Moderate — in trailer |
| **Implementation ease** | Easy — single line | Easiest — already half done | Medium — needs heredoc |
| **Checkpoint visibility** | Low — looks like chore | High — distinct prefix | Low — looks like chore |
| **Phase traceability** | Good — scope grep | Good — includes phase N | Excellent — trailer field |
| **Mixing with human commits** | Blends in | Stands out | Blends in |

---

### Recommendation

**Primary: Option B (Namespaced Prefix) for colony-automated commits.**

Rationale:

1. **It is already partially adopted.** The existing `aether-checkpoint: pre-phase-N` format is in use and hardcoded into `build.md`. Extending this pattern to `aether-progress:`, `aether-milestone:`, and `aether-fix:` is the lowest-friction path.

2. **Colony commits should be visually distinct from human commits.** When a developer runs `git log --oneline`, they should immediately see which commits were machine-generated colony operations vs. intentional human work. Option B achieves this; Options A and C deliberately blend in, which is a disadvantage for an autonomous multi-agent system.

3. **The Conventional Commits compatibility trade-off is acceptable.** Colony checkpoints and milestones are operational markers, not feature development. They should NOT trigger semver bumps or changelog entries. Keeping them outside the Conventional Commits namespace is actually correct — `feat:` and `fix:` should be reserved for actual features and fixes that matter to end users.

4. **Simple grep filtering is essential for automation.** The colony system itself needs to parse its own commits (e.g., finding the last checkpoint for rollback). `git log --grep="aether-checkpoint"` is simpler and more reliable than parsing scopes from Conventional Commits.

**Secondary consideration:** For progress commits where the colony does real feature/fix work that should appear in changelogs, consider using standard Conventional Commits (`feat:`, `fix:`) with an `Aether-Phase: N` trailer. This gives the best of both worlds: the progress work gets proper semver treatment, while colony operational commits stay in their own namespace.

**Proposed final convention:**

```
# Operational (colony namespace — never triggers semver/changelog)
aether-checkpoint: pre-phase-3
aether-milestone: phase 3 complete — signal unification

# Substantive work (conventional commits — triggers semver/changelog as appropriate)
feat: implement signal schema unification
fix: resolve missing signal paths in build.md
refactor: consolidate utility scripts

# Fix commits from swarm (conventional commits with colony context)
fix: resolve missing signal paths in build.md
```

This hybrid approach uses Option B for operational markers and standard Conventional Commits for actual code changes — matching the project's existing mixed style.
