# Command Playbooks

This directory contains split execution playbooks for high-complexity commands.

## Build

- `build-prep.md` - Version/state checks, argument validation, checkpoint setup
- `build-context.md` - Colony context loading, survey, archaeology, suggestions
- `build-wave.md` - Swarm initialization, wave execution, result processing
- `build-verify.md` - Watcher/measurer/chaos verification and flag creation
- `build-complete.md` - Synthesis, handoff/context update, display, session update

## Continue

- `continue-verify.md` - State loading and verification-loop setup
- `continue-gates.md` - Enforcement, anti-pattern, security, quality, runtime, flags gates
- `continue-advance.md` - State advancement, pheromone updates, learning proposal checks
- `continue-finalize.md` - Handoff/changelog/commit, completion display, session update

## Canonical Sources

- `build-full.md` - Full original build command prior to split
- `continue-full.md` - Full original continue command prior to split
