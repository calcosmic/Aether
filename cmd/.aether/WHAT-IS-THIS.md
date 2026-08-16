# What is this directory?

This is Aether's colony state for this repository — durable working memory,
not a build cache. Deleting it destroys any colony running here: its goal,
phase plan, learned lessons, and steering signals.

Safe to delete: ts-host/node_modules/ only (npm packages, ~60 MB — Aether
reinstalls them on demand).

Everything else here should be treated like your project's own files.
Managed by the aether CLI (https://github.com/calcosmic/Aether).
