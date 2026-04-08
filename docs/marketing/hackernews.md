# Hacker News Post

**Title:** Show HN: Aether -- Multi-agent dev framework modeled on ant colonies

---

Aether is an open-source multi-agent framework where 24 specialized workers self-organize around development goals. Written in Go, MIT licensed.

The architecture is modeled on ant colony behavior:

- **Pheromone signals** replace prompt engineering. You emit lightweight directives (FOCUS, REDIRECT, FEEDBACK) that guide agent behavior without micromanaging. Signals decay over time, so stale guidance doesn't persist.

- **Compounding memory** replaces stateless sessions. Agents develop instincts (learned patterns with confidence scores) that accumulate into collective wisdom over a project's lifetime.

- **Specialized castes** replace the single-assistant model. Builders implement. Watchers monitor quality. Scouts research. Chaos agents stress-test. Colonizers explore codebases. Each has distinct boundaries and protocols.

- **Self-organization** replaces manual orchestration. You set a goal, the colony plans phases, distributes work across castes, and self-verifies results.

The framework includes 28 built-in skills (code generation, testing, deployment, research), an autopilot mode for hands-off execution, and a session recovery system for long-running projects.

Architecture details, caste protocols, and signal system are documented in the repo. Happy to answer questions about design decisions.

github.com/calcosmic/Aether
aetherantcolony.com
Install: `go install github.com/calcosmic/Aether@latest`

---

*Word count: 188*
