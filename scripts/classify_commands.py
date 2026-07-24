#!/usr/bin/env python3
"""
Enrich cmd/testdata/command_catalog.json with classification and since_version.

Usage:
    python3 scripts/classify_commands.py
"""

import json
import re
import subprocess
import sys
from pathlib import Path

CATALOG_PATH = Path("cmd/testdata/command_catalog.json")
CACHE_PATH = Path(".planning/phases/146-command-classification/git_tag_cache.json")

# Tags to scan, ordered oldest -> newest
TAGS = ["v5.4.0", "v1.10", "v1.11", "v1.12", "v1.13", "v1.14",
        "v1.17", "v1.18", "v1.19", "v1.20", "v1.21"]

# Lifecycle command roots and their known subcommands/finalize variants
LIFECYCLE_ROOTS = {"init", "plan", "build", "continue", "seal", "colonize", "run", "entomb"}
LIFECYCLE_SUBS = {"init-ceremony", "init-research",
                  "plan-finalize", "plan-granularity",
                  "build-finalize",
                  "continue-finalize",
                  "seal-finalize",
                  "colonize-finalize"}

# Alias commands (explicit list from research)
ALIAS_NAMES = {
    "watch",
    "pheromone-export-xml", "pheromone-import-xml",
    "wisdom-export-xml", "wisdom-import-xml",
    "registry-export-xml", "registry-import-xml",
    "colony-archive-xml",
}

# Internal runtime commands
INTERNAL_NAMES = {
    "autofix-checkpoint", "autofix-rollback",
    "error-pattern-check",
}

# Host subcommands that are lifecycle vs utility vs alias
HOST_LIFECYCLE = {"build", "colonize", "continue", "plan", "seal", "lifecycle"}
HOST_UTILITY = {"oracle", "swarm"}
HOST_ALIAS = {"watch"}


def classify(entry: dict) -> str:
    name = entry["name"]
    parent = entry.get("parent_command", "")
    desc = entry.get("short_description", "")

    # Explicit deprecated marker
    if "deprecated" in desc.lower():
        return "deprecated"

    # Internal runtime
    if name in INTERNAL_NAMES or name.startswith("hook-") or name.startswith("autofix-"):
        return "internal_runtime"

    # Alias
    if name in ALIAS_NAMES or "alias" in desc.lower():
        return "alias"

    # Host subcommands
    if parent == "host":
        if name in HOST_LIFECYCLE:
            return "public_lifecycle"
        if name in HOST_UTILITY:
            return "public_utility"
        if name in HOST_ALIAS:
            return "alias"
        return "public_utility"

    # Lifecycle roots and known subcommands
    if name in LIFECYCLE_ROOTS or name in LIFECYCLE_SUBS:
        return "public_lifecycle"

    # Default
    return "public_utility"


def load_cache() -> dict:
    if CACHE_PATH.exists():
        with open(CACHE_PATH, "r") as f:
            return json.load(f)
    return {}


def save_cache(cache: dict) -> None:
    CACHE_PATH.parent.mkdir(parents=True, exist_ok=True)
    with open(CACHE_PATH, "w") as f:
        json.dump(cache, f, indent=2)


def get_commands_at_tag(tag: str, repo_root: Path, catalog_names: set) -> set:
    """Return set of command names present in cmd/*.go at a given git tag."""
    try:
        result = subprocess.run(
            ["git", "ls-tree", "-r", "--name-only", tag, "cmd/"],
            capture_output=True, text=True, cwd=repo_root, check=True
        )
    except subprocess.CalledProcessError:
        return set()

    files = [line for line in result.stdout.splitlines() if line.endswith(".go") and not line.endswith("_test.go")]
    if not files:
        return set()

    found = set()
    # Extract all quoted strings from all files at this tag in one pass
    for f in files:
        try:
            show = subprocess.run(
                ["git", "show", f"{tag}:{f}"],
                capture_output=True, text=True, cwd=repo_root, check=True
            )
        except subprocess.CalledProcessError:
            continue
        content = show.stdout
        # Find all quoted strings in the file
        quoted = set(re.findall(r'["\']([a-zA-Z0-9_-]+)["\']', content))
        # Also match Use: patterns
        use_matches = set(re.findall(r'\bUse:\s*["\']?([a-zA-Z0-9_-]+)["\']?\b', content))
        all_strings = quoted | use_matches
        # Intersect with catalog names
        found.update(all_strings & catalog_names)
    return found


def compute_since_versions(catalog: list, repo_root: Path, cache: dict) -> dict:
    """Return mapping command_name -> earliest tag or 'current'."""
    catalog_names = {entry["name"] for entry in catalog}
    since = {}

    for entry in catalog:
        name = entry["name"]
        if name in cache and not name.startswith("__"):
            since[name] = cache[name]
            continue

        earliest = None
        for tag in TAGS:
            # Check cache for tag-level results
            cache_key = f"__tag_{tag}"
            if cache_key in cache:
                tag_commands = set(cache[cache_key])
            else:
                print(f"  Scanning {tag}...", file=sys.stderr)
                tag_commands = get_commands_at_tag(tag, repo_root, catalog_names)
                cache[cache_key] = sorted(tag_commands)

            if name in tag_commands:
                earliest = tag
                break

        result = earliest if earliest else "current"
        since[name] = result
        cache[name] = result
    return since


def main():
    repo_root = Path(__file__).resolve().parent.parent
    catalog_file = repo_root / CATALOG_PATH

    if not catalog_file.exists():
        print(f"Catalog not found: {catalog_file}", file=sys.stderr)
        sys.exit(1)

    with open(catalog_file, "r") as f:
        catalog = json.load(f)

    # Add classification
    for entry in catalog:
        entry["classification"] = classify(entry)

    # Add since_version
    print("Computing since_version from git tags...", file=sys.stderr)
    cache = load_cache()
    since_map = compute_since_versions(catalog, repo_root, cache)
    for entry in catalog:
        entry["since_version"] = since_map[entry["name"]]

    save_cache(cache)

    with open(catalog_file, "w") as f:
        json.dump(catalog, f, indent=2)

    # Summary
    counts = {}
    for entry in catalog:
        counts[entry["classification"]] = counts.get(entry["classification"], 0) + 1

    print(f"Enriched {len(catalog)} entries.")
    print("Classifications:")
    for cls in ["public_lifecycle", "public_utility", "internal_runtime", "alias", "deprecated"]:
        print(f"  {cls}: {counts.get(cls, 0)}")

    current_count = sum(1 for e in catalog if e["since_version"] == "current")
    print(f"since_version='current': {current_count}")


if __name__ == "__main__":
    main()
