<div align="center">

<img src="https://raw.githubusercontent.com/calcosmic/Aether/main/assets/banner/banner.jpg" alt="Aether Banner" width="100%" />

<img src="https://raw.githubusercontent.com/calcosmic/Aether/main/assets/logo/logo.jpg" alt="Aether Logo" width="140" />

# Aether

**Artificial Ecology for Thought and Emergent Reasoning**

[![GitHub release](https://img.shields.io/github/v/release/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/releases)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-7B3FE4?style=flat-square)](https://github.com/calcosmic/Aether/blob/main/LICENSE)
[![GitHub stars](https://img.shields.io/github/stars/calcosmic/Aether.svg?style=flat-square)](https://github.com/calcosmic/Aether/stargazers)

```bash
npx --yes aether-colony@latest
```

</div>

`aether-colony` is the low-friction npm bootstrap for Aether. It is not a second runtime. It downloads the matching published Go `aether` binary for your platform, installs it into a stable local directory, and then hands off to the real CLI.

The npm package version intentionally matches the published Go release version.
There is one public Aether version, not one version for npm and another for the
runtime.

## What is Aether?

If Claude Code, OpenCode, and Codex are the workers, Aether is the colony.

Aether is a Go runtime plus companion-file system for AI software delivery. It orchestrates planning, build, verification, recovery, and update workflows across the supported platforms so the system stays aligned instead of drifting into disconnected agent sessions.

## Aether is right for you if

- You want the easiest first install path
- You want the published Go runtime, not a separate Node runtime
- You want one install path that can hand off to the real `aether` CLI immediately
- You want the npm package version and the Aether release version to stay identical

## What happens on first run

1. The wrapper resolves the matching Aether release for your platform.
2. It downloads and verifies the release archive from GitHub Releases.
3. It installs the binary locally.
4. It runs `aether install` so the hub and companion files are populated.

## Quick start

```bash
npx --yes aether-colony@latest
```

## Hand off to the real CLI

```bash
npx --yes aether-colony@latest -- status
npx --yes aether-colony@latest -- update --force --download-binary
npx --yes aether-colony@latest -- init "Build feature X"
```

## Important distinction

- `aether-colony` is the bootstrap and discovery path.
- The real runtime is the Go `aether` binary.
- After the first install, users should normally run `aether ...` directly.
- `aether-colony@1.0.19` is expected to bootstrap Aether `1.0.19`.

## Source and docs

- GitHub: https://github.com/calcosmic/Aether
- Install guide: https://github.com/calcosmic/Aether#-install
- Release notes: https://github.com/calcosmic/Aether/releases
