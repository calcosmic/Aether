# Phase 113: Ceremony Narrator - Research

**Researched:** 2026-05-13
**Domain:** TypeScript terminal rendering, Go event emission, YAML template system
**Confidence:** HIGH

## Summary

Phase 113 replaces Go's hardcoded `cmd/codex_visuals.go` rendering with an event-driven TypeScript narrator that consumes Go ceremony events via the Phase 112 event bridge and writes formatted terminal output in real-time. The TS host already has the event bridge (`event-bridge.ts`), caste config loader (`caste-config.ts`), and type definitions (`types.ts`) from Phase 112. This phase adds the narrator module, template system, and three-mode output support.

**Primary recommendation:** Build a `narrator.ts` module that subscribes to the event bridge, loads editable YAML-frontmatter templates from `.aether/templates/ceremony/`, and renders using chalk + figlet + boxen. Keep Go changes minimal — only ensure all visual moments emit events instead of printing directly.

---

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01: Editable Ceremony Templates** — Templates live in `.aether/templates/ceremony/` as YAML frontmatter + markdown body, NOT hardcoded in TypeScript.
- **D-02: Non-TTY Fallback** — Plain text with structure for non-TTY. Colors stripped, emojis preserved, stage separators kept. Use `chalk` with `supports-color` detection.
- **D-03: Direct Terminal Output** — TS host writes ceremony directly to `process.stdout` in real-time. Wrapper still receives final structured result via Go finalizers.
- **D-04: Per-Command Custom Stage Separators** — Each lifecycle command has its own stage separator style configured in `ceremony.yaml` under `stage_separator.<command>` keys.
- **D-05: Three-Mode Output Support** — `json` (machine), `visual` (TTY ANSI), `markdown` (plain text). Mode selected via `AETHER_OUTPUT_MODE` env var.

### Deferred Ideas (OUT OF SCOPE)
- Template hot-reload with chokidar
- Custom user templates in `~/.aether/templates/ceremony/`
- Animated transitions / spinners between stages

---

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| CER-01 | Ceremony banners and art restored to command wrappers (editable markdown, not compiled Go) | Template system with YAML frontmatter + figlet |
| CER-02 | Shared ceremony config in YAML (caste emoji/color/label maps, naming conventions) | `caste-config.ts` already loads `.aether/config/ceremony.yaml` |
| CER-03 | Go ceremony rendering code replaced by event emission (Go emits, wrappers render from templates) | `ceremony_emitter.go` already emits events; need to redirect direct prints in `codex_visuals.go` |
| CER-04 | Crowned Anthill seal ASCII art in editable template | figlet template with configurable font |
| CER-05 | Worker spawn notifications with caste identity frames | Template for spawn frame + caste config accessors |
| CER-06 | Build summary and closeout rituals with template frames | boxen-framed summary template |

---

## Standard Stack

### Core (already in TS host `package.json`)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| chalk | 5.6.2 | ANSI color codes | ESM-native, supports `supports-color` auto-detection, de-facto standard [VERIFIED: npm registry] |
| figlet | 1.11.0 | ASCII banner generation | Only mature Node.js figlet library, supports 200+ fonts [VERIFIED: npm registry] |
| boxen | 8.0.1 | Terminal box frames | Clean API, works with chalk, maintained by Sindre Sorhus [VERIFIED: npm registry] |
| js-yaml | 4.1.0 | YAML parsing for templates and config | Already used by `caste-config.ts`, proven in codebase [VERIFIED: codebase] |
| strip-ansi | 7.2.0 | Strip ANSI codes for non-TTY fallback | Used by chalk ecosystem, lightweight [VERIFIED: npm registry] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| supports-color | 10.2.2 | TTY / color support detection | chalk already bundles it; explicit import only if chalk's built-in detection is insufficient [VERIFIED: npm registry] |

**No additional installs needed** — all required packages are already in `package.json`.

---

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Go Runtime                               │
│  cmd/ceremony_emitter.go  ──►  events.Bus.Publish()             │
│  (already emits events)                                          │
└────────────────────────────┬────────────────────────────────────┘
                             │ JSONL event bus
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                     TypeScript Host                              │
│                                                                  │
│  ┌──────────────┐    ┌─────────────┐    ┌──────────────────┐   │
│  │ event-bridge │───►│  narrator   │───►│ process.stdout   │   │
│  │ (Phase 112)  │    │ (new)       │    │ (real-time)      │   │
│  └──────────────┘    └──────┬──────┘    └──────────────────┘   │
│                             │                                    │
│                    ┌────────┴────────┐                          │
│                    ▼                 ▼                          │
│            ┌─────────────┐   ┌─────────────┐                   │
│            │ template-   │   │ caste-      │                   │
│            │ loader      │   │ config      │                   │
│            │ (new)       │   │ (Phase 112) │                   │
│            └─────────────┘   └─────────────┘                   │
│                    ▲                 ▲                          │
│                    │                 │                          │
│            .aether/templates/   .aether/config/                 │
│            ceremony/*.md        ceremony.yaml                   │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
.aether/ts-host/src/
├── event-bridge.ts       # (existing) Event consumption
├── caste-config.ts       # (existing) Caste config loader
├── types.ts              # (existing) CeremonyEvent, CeremonyPayload
├── narrator.ts           # (NEW) Main narrator: subscribes to bridge, dispatches to renderers
├── template-loader.ts    # (NEW) Load YAML-frontmatter templates from .aether/templates/ceremony/
├── renderers/
│   ├── visual.ts         # (NEW) ANSI output: chalk + figlet + boxen
│   ├── markdown.ts       # (NEW) Plain text output: stripped ANSI, preserved structure
│   └── json.ts           # (NEW) Passthrough / no-op (Go handles json mode)
└── index.ts              # (existing) Wire narrator into lifecycle

.aether/templates/ceremony/   # (NEW)
├── banner-build-start.md
├── banner-seal-complete.md
├── spawn-frame.md
├── stage-separator.md
├── build-summary.md
└── closeout-ritual.md
```

### Pattern 1: Event-to-Render Dispatch
**What:** The narrator receives a `CeremonyEvent`, looks up the topic in a handler map, and calls the appropriate renderer with the payload.
**When to use:** All ceremony events follow this pattern.
**Example:**
```typescript
// Source: 113-CONTEXT.md D-03
const handlers: Record<string, (payload: CeremonyPayload) => string> = {
  "ceremony.build.spawn": renderSpawnFrame,
  "ceremony.build.wave.start": renderWaveStart,
  "ceremony.build.wave.end": renderWaveEnd,
  "ceremony.chamber.seal": renderSealBanner,
};

function onEvent(event: CeremonyEvent): void {
  const render = handlers[event.topic];
  if (render) {
    const output = render(event.payload);
    process.stdout.write(output + "\n");
  }
}
```

### Pattern 2: Template with YAML Frontmatter
**What:** Each template file has YAML frontmatter defining variables (figlet font, emoji, title) and a markdown body with `{variable}` substitution.
**When to use:** Banners, spawn frames, and any reusable ceremony text.
**Example:**
```markdown
---
figlet_font: Standard
emoji: "🔨"
title: "BUILD"
---

{banner}

── {stage} ──

{content}
```

### Pattern 3: Three-Mode Output
**What:** The narrator selects a renderer backend based on `AETHER_OUTPUT_MODE` and TTY detection.
**When to use:** Every output path.
**Selection logic:**
```typescript
// Source: 113-CONTEXT.md D-05
function selectRenderer(): Renderer {
  const mode = process.env.AETHER_OUTPUT_MODE ?? "visual";
  if (mode === "json") return jsonRenderer; // passthrough
  if (mode === "markdown") return markdownRenderer;
  if (mode === "visual" && process.stdout.isTTY) return visualRenderer;
  return markdownRenderer; // non-TTY fallback
}
```

### Anti-Patterns to Avoid
- **Hardcoding templates in TypeScript:** Defeats D-01. Templates must be editable files.
- **Writing to `.aether/data/` from TS host:** Boundary violation. The narrator is read-only.
- **Replacing Go's `outputOK` / `outputError` JSON paths:** Go still owns machine-readable output. The narrator only handles visual presentation.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ANSI color detection | Custom TTY checks | chalk (built-in `supports-color`) | Handles `NO_COLOR`, `FORCE_COLOR`, CI environments, Windows terminals |
| ASCII banner generation | Custom character art | figlet | 200+ fonts, battle-tested, supports vertical/horizontal layout |
| Terminal box drawing | Manual unicode borders | boxen | Handles padding, border styles, color integration, terminal width |
| YAML parsing | Custom regex | js-yaml | Already in codebase, handles edge cases, safe load |
| Template substitution | `String.prototype.replace` loop | Simple regex with fallback defaults | Sufficient for `{var:default}` syntax; no need for full templating engine |

**Key insight:** The ceremony narrator is a presentation layer, not a framework. Use mature, focused libraries for each concern rather than building a custom rendering engine.

---

## Common Pitfalls

### Pitfall 1: Go Direct Rendering Still Active
**What goes wrong:** If `codex_visuals.go` functions like `renderBanner`, `renderStageMarker`, or `renderInitVisual` continue to be called directly, the terminal gets duplicate output — once from Go, once from TS.
**Why it happens:** Go commands call `outputWorkflow(result, visual)` which prints visual output before returning JSON. The TS host then receives the same event and prints again.
**How to avoid:** Audit every call site of `outputWorkflow`, `emitVisualProgress`, and `writeVisualOutput` in `cmd/`. Replace direct visual prints with `emitBuildCeremony` or `emitLifecycleCeremony` calls. The Go command should return JSON only; visual output becomes the TS host's responsibility.
**Warning signs:** Golden tests show duplicate banners or stage markers.

### Pitfall 2: Non-TTY Emoji Width Miscalculation
**What goes wrong:** Terminal emulators count emoji as 2 columns, but some libraries (boxen, figlet) may miscalculate width when emojis are present, causing misaligned boxes.
**Why it happens:** Unicode width detection is inconsistent across terminals and libraries.
**How to avoid:** Use `strip-ansi` before measuring width for boxen. Test spawn frames with widest emoji (`👁️` is 2 columns). Consider padding adjustments.
**Warning signs:** Box borders don't align on the right side.

### Pitfall 3: Template File Missing in Production
**What goes wrong:** If `.aether/templates/ceremony/` is not distributed by `aether publish`, the TS host crashes on missing templates.
**Why it happens:** Templates are new files not yet in the publish manifest.
**How to avoid:** Include inline fallback templates in `template-loader.ts` (similar to `DEFAULT_CEREMONY_CONFIG` in `caste-config.ts`). If a template file is missing, use the fallback.
**Warning signs:** `ENOENT` errors when running `aether build` after `aether update`.

### Pitfall 4: Chalk ESM Import Issues
**What goes wrong:** chalk 5.x is ESM-only. If the TS host build uses CommonJS output, `require("chalk")` fails.
**Why it happens:** `package.json` has `"type": "module"` and chalk 5 is ESM-native, but build misconfiguration can still cause issues.
**How to avoid:** Verify `tsconfig.build.json` outputs ESM. The existing `package.json` already has `"type": "module"` and chalk 5.6.2 is installed — this should work. Just don't downgrade to chalk 4.
**Warning signs:** `Error [ERR_REQUIRE_ESM]: require() of ES Module` at runtime.

---

## Code Examples

### Verified patterns from official sources:

#### Chalk with conditional color
```typescript
// Source: chalk official docs (via Context7 / npm)
import chalk from "chalk";

const c = chalk.supportsColor ? chalk.hex("#FFD700") : { bold: (s: string) => s };
console.log(c.bold("Builder"));
```

#### Figlet banner generation
```typescript
// Source: figlet npm README
import figlet from "figlet";

const banner = figlet.textSync("BUILD", { font: "Standard" });
console.log(chalk.cyan(banner));
```

#### Boxen frame
```typescript
// Source: boxen npm README
import boxen from "boxen";

const box = boxen("Build complete!", {
  padding: 1,
  margin: 1,
  borderStyle: "round",
  borderColor: "green",
});
console.log(box);
```

#### YAML frontmatter parsing
```typescript
// Source: js-yaml docs + 113-CONTEXT.md D-01
import yaml from "js-yaml";

function parseTemplate(raw: string): { frontmatter: Record<string, unknown>; body: string } {
  const match = raw.match(/^---\n([\s\S]*?)\n---\n([\s\S]*)$/);
  if (!match) throw new Error("Invalid template: missing YAML frontmatter");
  return {
    frontmatter: yaml.load(match[1]) as Record<string, unknown>,
    body: match[2].trim(),
  };
}
```

#### Variable substitution with fallback
```typescript
// Source: 113-CONTEXT.md Specific Ideas
function substitute(template: string, vars: Record<string, string>): string {
  return template.replace(/\{(\w+)(?::([^}]*))?\}/g, (_match, key, fallback) => {
    return vars[key] ?? fallback ?? "";
  });
}
```

---

## Go Changes Needed

### Minimal changes to `cmd/codex_visuals.go`
The following functions currently print directly and need to emit events instead (or have their callers updated):

| Function | Current Behavior | Needed Change |
|----------|-----------------|---------------|
| `outputWorkflow(result, visual)` | Prints visual then returns JSON | Remove visual print; Go returns JSON only. TS host handles visual via event. |
| `emitVisualProgress(visual)` | Prints progress string | Replace with `emitBuildCeremony(events.CeremonyTopicBuildToolUse, ...)` |
| `renderInitVisual(...)` | Returns full init banner string | Move to template; Go emits `ceremony.build.prewave` with context |
| `renderBanner(emoji, title)` | Returns banner string | Template-driven in TS host |
| `renderStageMarker(title)` | Returns stage separator | Template-driven in TS host |
| `renderAetherWordmark()` | Returns ASCII wordmark | Keep in Go for `aether version` only; ceremony uses figlet templates |

### No changes to `cmd/ceremony_emitter.go`
This file already emits all required events. The narrator consumes them.

### Potential new topics needed
- `ceremony.build.summary` — for build closeout / summary frame (CER-06)
- `ceremony.command.start` — for command-level banner (CER-01)

**Confidence:** MEDIUM — exact topic list depends on which Go render functions are replaced. The existing 29 topics cover most lifecycle moments.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Go `codex_visuals.go` hardcodes all rendering | Event-driven TS host with editable templates | Phase 113 (v1.17) | Non-technical users can customize ceremony without code changes |
| Single visual mode | Three-mode output (json/visual/markdown) | Phase 113 (v1.17) | CI logs are readable, machine parsing still works |
| Go owns presentation + safety | Go owns safety, TS host owns soul | v1.17 roadmap | Separation of concerns aligned with philosophy |

**Deprecated/outdated:**
- Direct `fmt.Fprint(w, visual)` in Go commands: replaced by event emission
- Hardcoded `aetherWordmark` in Go: replaced by figlet templates

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `chalk 5.6.2` ESM works with existing `tsconfig.build.json` | Standard Stack | Build fails at runtime; fix by verifying ESM output |
| A2 | `figlet` fonts "Standard" and any others used are available at runtime | Template System | Banner falls back to plain text; acceptable degradation |
| A3 | `boxen` handles emoji width correctly on macOS and Linux terminals | Common Pitfalls | Misaligned boxes; fix with manual width adjustment |
| A4 | All Go visual output can be replaced by events without breaking `aether version` or non-ceremony commands | Go Changes Needed | Some commands (like `version`) may still need Go-side rendering |

---

## Open Questions (RESOLVED)

1. **Which Go commands besides `build`, `plan`, `continue`, `colonize`, `seal` need ceremony events?** — RESOLVED: Only lifecycle commands (build/plan/continue/colonize/seal) are migrated. `aether version` and other non-lifecycle commands keep Go-side rendering. Verified by audit of `cmd/codex_visuals.go` render* call sites in Plan 113-02.

2. **Should the TS host narrator run as a singleton per command, or per lifecycle session?** — RESOLVED: Per-command narrator instance. Matches event bridge lifecycle (one bridge per command invocation). Plan 113-02 Task 2 implements this.

3. **How are templates distributed via `aether publish`?** — RESOLVED: `DEFAULT_TEMPLATES` inline fallback in `template-loader.ts` ensures templates work even if `.aether/templates/ceremony/` is missing from publish manifest. Plan 113-01 Task 1 includes this safety net.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | TS host runtime | ✓ | >=20 (package.json engines) | — |
| chalk | ANSI colors | ✓ | 5.6.2 (in package.json) | — |
| figlet | ASCII banners | ✓ | 1.11.0 (in package.json) | Plain text fallback |
| boxen | Terminal frames | ✓ | 8.0.1 (in package.json) | Plain text fallback |
| js-yaml | YAML parsing | ✓ | 4.1.0 (in package.json) | — |
| strip-ansi | ANSI stripping | ✓ | 7.2.0 (in package.json) | — |

**Missing dependencies with no fallback:** None.

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Node.js built-in test runner (`node:test`) via tsx |
| Config file | none — see Wave 0 |
| Quick run command | `npm test` |
| Full suite command | `npm test` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CER-01 | Banner renders from template | unit | `npm test -- test/narrator.test.ts` | ❌ Wave 0 |
| CER-02 | Caste config loads emoji/color/label | unit | `npm test -- test/caste-config.test.ts` | ✅ (Phase 112) |
| CER-03 | Go emits event instead of printing | integration | `go test ./cmd/...` | ❌ Wave 0 |
| CER-04 | Seal figlet banner matches template | unit | `npm test -- test/narrator.test.ts` | ❌ Wave 0 |
| CER-05 | Spawn frame includes caste identity | unit | `npm test -- test/narrator.test.ts` | ❌ Wave 0 |
| CER-06 | Build summary renders as boxen frame | unit | `npm test -- test/narrator.test.ts` | ❌ Wave 0 |

### Wave 0 Gaps
- [ ] `test/narrator.test.ts` — covers CER-01, CER-04, CER-05, CER-06
- [ ] `test/template-loader.test.ts` — covers template parsing and fallback
- [ ] `test/renderers.test.ts` — covers visual, markdown, and json renderers

---

## Security Domain

This phase is purely presentation-layer code. It does not handle authentication, session management, cryptography, or user input validation. The only security-relevant concern is the boundary contract (TS host must not write to `.aether/data/`), which is already enforced by `boundary-reference.ts`.

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes (templates) | js-yaml safeLoad, path validation before file read |
| V6 Cryptography | no | — |

**Known threat pattern:** Path traversal via malicious template path. Mitigation: restrict template loading to `.aether/templates/ceremony/` subdirectory, reject `..` segments.

---

## Sources

### Primary (HIGH confidence)
- `cmd/ceremony_emitter.go` — Event emission API and payload shapes
- `cmd/codex_visuals.go` — Current Go rendering logic (first 500 lines reviewed)
- `pkg/events/ceremony.go` — Go ceremony topic constants and payload struct
- `.aether/ts-host/src/event-bridge.ts` — Event consumption API
- `.aether/ts-host/src/caste-config.ts` — Caste config loader with typed accessors
- `.aether/ts-host/src/types.ts` — CeremonyEvent, CeremonyPayload, CeremonyTopic types
- `.aether/ts-host/package.json` — Installed dependency versions
- `.aether/config/ceremony.yaml` — Shared ceremony config

### Secondary (MEDIUM confidence)
- `113-CONTEXT.md` — Implementation decisions and specific ideas
- npm registry versions for chalk, figlet, boxen, supports-color, js-yaml

### Tertiary (LOW confidence)
- None — all claims verified against codebase or npm registry

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already installed and verified
- Architecture: HIGH — event bridge and caste config already proven in Phase 112
- Pitfalls: MEDIUM-HIGH — based on codebase analysis, but full `codex_visuals.go` audit needed
- Go changes: MEDIUM — exact scope depends on audit of all render call sites

**Research date:** 2026-05-13
**Valid until:** 2026-06-13 (stable stack, low churn expected)
