#!/usr/bin/env python3
"""Build historical reliability matrix from git tags."""
import json, os, re, subprocess, sys
CATALOG_PATH = "cmd/testdata/command_catalog.json"
CACHE_PATH = "scripts/.reliability_cache.json"
TAGS = ["v5.4.0", "v1.10", "v1.11", "v1.12", "v1.13", "v1.14", "v1.17", "v1.18", "v1.19", "v1.20", "v1.21"]

def load_cache():
    if os.path.exists(CACHE_PATH):
        with open(CACHE_PATH) as f:
            return json.load(f)
    return {}

def save_cache(data):
    os.makedirs(os.path.dirname(CACHE_PATH), exist_ok=True)
    with open(CACHE_PATH, "w") as f:
        json.dump(data, f, indent=2)

def tag_exists(tag):
    try:
        subprocess.check_output(["git", "rev-parse", f"refs/tags/{tag}"], stderr=subprocess.DEVNULL)
        return True
    except subprocess.CalledProcessError:
        return False

def list_cmd_files(tag):
    try:
        out = subprocess.check_output(["git", "ls-tree", "-r", "--name-only", tag, "--", "cmd/"], text=True, stderr=subprocess.DEVNULL)
        return [l for l in out.strip().split("
") if l.endswith(".go") and not l.endswith("_test.go")]
    except subprocess.CalledProcessError:
        return []

def extract_commands_from_file(tag, filepath):
    commands = set()
    try:
        content = subprocess.check_output(["git", "show", f"{tag}:{filepath}"], text=True, stderr=subprocess.DEVNULL)
    except subprocess.CalledProcessError:
        return commands
    for line in content.split("
"):
        m = re.search(r'Use:\s*"([^"]+)"', line)
        if m:
            commands.add(m.group(1).split()[0])
    return commands

def scrape_tag(tag):
    files = list_cmd_files(tag)
    commands = set()
    for filepath in files:
        commands |= extract_commands_from_file(tag, filepath)
    return commands

def build_presence_matrix(cache):
    presence = {}
    for tag in TAGS:
        if tag in cache:
            present_set = set(cache[tag])
        elif tag_exists(tag):
            print(f"  Scanning {tag}...", file=sys.stderr)
            present_set = scrape_tag(tag)
            cache[tag] = sorted(present_set)
        else:
            print(f"Warning: tag {tag} not found", file=sys.stderr)
            continue
        presence[tag] = present_set
    return presence

def main():
    with open(CATALOG_PATH) as f:
        catalog = json.load(f)
    cache = load_cache()
    presence = build_presence_matrix(cache)
    save_cache(cache)
    for entry in catalog:
        name = entry["name"]
        hp = {}
        true_count = 0
        for tag in TAGS:
            if tag in presence:
                present = name in presence[tag]
                hp[tag] = present
                if present:
                    true_count += 1
            else:
                hp[tag] = False
        entry["historical_presence"] = hp
        entry["stability_score"] = round(true_count / len(TAGS) * 100)
    with open(CATALOG_PATH, "w") as f:
        json.dump(catalog, f, indent=2)
        f.write("
")
    print(f"Enriched {len(catalog)} entries.")

if __name__ == "__main__":
    main()
