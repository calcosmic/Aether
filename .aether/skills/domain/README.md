# Domain Skills

Drop a folder here with a SKILL.md to add custom domain skills.
Skills created with `/ant:skill-create` (Claude Code/OpenCode) or `aether skill-create` (Codex CLI) are placed here automatically.

## Creating a Custom Skill

1. Create a directory: `mkdir ~/.aether/skills/domain/my-skill`
2. Create `SKILL.md` inside it with this frontmatter:

```yaml
---
name: my-skill
description: Use when working with my technology
type: domain
domains: [my-domain, related-domain]
agent_roles: [builder]
detect_files: ["my-config.*"]
detect_packages: ["my-package"]
priority: normal
version: "1.0"
---

Your best practices and guidance here.
```

3. The skill will be auto-detected on the next build

## Important

- Your custom skills are **never overwritten** by `aether update`
- Only skills listed in `.manifest.json` are managed by Aether
- Use `/ant:skill-create "<topic>"` (Claude Code/OpenCode) or `aether skill-create "<topic>"` (Codex CLI) for Oracle-powered skill generation
