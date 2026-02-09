# Changelog

All notable changes to the Aether Colony project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2025-02-09

### First Stable Release 🎉

Aether Colony is a multi-agent system using ant colony intelligence for Claude Code and OpenCode. Workers self-organize via pheromone signals to complete complex tasks autonomously.

### Features
- **Command output styling** — Visual consistency with emoji sandwich styling across all ant commands
- **20 ant commands** for autonomous project planning, building, and management
- **Multi-agent emergence** — Workers spawn sub-workers to handle complex tasks
- **Pheromone signals** — FOCUS, REDIRECT, and FEEDBACK for colony guidance
- **Project flags** — Blockers, issues, and notes for tracking impediments
- **State persistence** — Session handoff with pause/resume colony capability

### Architecture
- Queen ant orchestrates via pheromone signals
- Worker castes: Builder, Scout, Watcher, Architect, Route-Setter
- Pure emergence — no hardcoded implementations, workers discover solutions
- TDD enforcement with independent verification
- Git checkpoint/rollback for safety

### Previous Development Versions
Versions 2.0.0–2.4.2 were development/testing releases. This 1.0.0 is the first stable production release.
