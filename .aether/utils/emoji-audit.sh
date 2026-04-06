#!/usr/bin/env bash
# emoji-audit.sh — Audit emoji usage across Aether command files
# Checks command files against the canonical emoji reference map defined in
# .aether/skills/colony/colony-visuals/SKILL.md
#
# Usage: bash emoji-audit.sh [repo_root]
#
# Output: JSON compatible with aether-utils.sh subcommand pattern:
#   {"ok": true, "result": {"files_scanned": N, "total_emojis": N,
#     "unmapped": [...], "unused": [...], "usage": {...}}}
#
# Compatible with bash 3.x (macOS system bash).
# Can be sourced by aether-utils.sh or run standalone.

# ---------------------------------------------------------------------------
# _emoji_audit_main — perform the audit and print JSON result
# Uses Python3 for emoji extraction and map logic (handles multi-codepoint sequences)
# ---------------------------------------------------------------------------
_emoji_audit_main() {
    local repo_root="${1:-.}"

    if ! command -v python3 >/dev/null 2>&1; then
        printf '{"ok":false,"error":"python3 is required but not found"}\n'
        return 1
    fi

    python3 - "$repo_root" <<'PYEOF'
import sys
import os
import re
import json
import glob

repo_root = sys.argv[1] if len(sys.argv) > 1 else "."

# ---------------------------------------------------------------------------
# Canonical emoji reference map — must match colony-visuals SKILL.md
# ---------------------------------------------------------------------------
EMOJI_REF_MAP = {
    "\U0001f528":        "Builder ant",           # 🔨
    "\U0001f441\ufe0f": "Watcher ant",            # 👁️
    "\U0001f3b2":        "Chaos ant",             # 🎲
    "\U0001f50d":        "Scout ant",             # 🔍
    "\U0001f3fa":        "Archaeologist / Seal",  # 🏺
    "\U0001f52e":        "Oracle ant",            # 🔮
    "\U0001f3db\ufe0f": "Architect ant",          # 🏛️
    "\U0001f50c":        "Ambassador ant",        # 🔌
    "\U0001f4ca":        "Measurer ant / Status", # 📊
    "\U0001f9ea":        "Probe / Tests",         # 🧪
    "\U0001f504":        "Weaver / Refresh",      # 🔄
    "\U0001f4e6":        "Gatekeeper / Package",  # 📦
    "\U0001f465":        "Auditor",               # 👥
    "\U0001f6a9":        "Flag / Blocker",        # 🚩
    "\U0001f4ad":        "Dream",                 # 💭
    "\U0001f95a":        "Queen / Init",          # 🥚
    "\U0001f4cb":        "Plan / List",           # 📋
    "\u2705":            "Pass / Success",        # ✅
    "\u274c":            "Fail / Error",          # ❌
    "\u26a0\ufe0f":     "Warning",               # ⚠️
    "\u26d4":            "Hard block",            # ⛔
    "\U0001f4be":        "Save / Persist",        # 💾
    "\U0001f3af":        "Focus signal",          # 🎯
    "\U0001f6ab":        "Redirect signal",       # 🚫
    "\U0001f4ac":        "Feedback signal",       # 💬
}

# ---------------------------------------------------------------------------
# Scan command files
# ---------------------------------------------------------------------------
scan_dirs = [
    os.path.join(repo_root, ".claude", "commands", "ant"),
    os.path.join(repo_root, ".opencode", "commands", "ant"),
]

scan_files = []
for d in scan_dirs:
    if os.path.isdir(d):
        scan_files.extend(glob.glob(os.path.join(d, "*.md")))

files_scanned = len(scan_files)

# ---------------------------------------------------------------------------
# Extract emoji sequences from combined content
# Matches base emoji + optional variation selectors, ZWJ sequences
# ---------------------------------------------------------------------------
EMOJI_PATTERN = re.compile(
    r'[\U0001F300-\U0001F9FF\U00002600-\U000027BF\U00002702-\U000027B0'
    r'\U0001FA00-\U0001FA9F\U0001FAA0-\U0001FAFF\U00002300-\U000023FF'
    r'\U00002B00-\U00002BFF\U00003000-\U0000303F'
    r'\U0001F600-\U0001F64F\U0001F680-\U0001F6FF'
    r'\u2300-\u27BF\u2B00-\u2BFF\u2600-\u27FF'
    r'\u2702-\u27B0\u2194-\u21AA\u231A-\u231B\u23E9-\u23F3\u23F8-\u23FA'
    r'\u25AA-\u25FE\u2614-\u2615\u2648-\u2653\u267F\u2693\u26A0-\u26A1'
    r'\u26AA-\u26AB\u26BD-\u26BE\u26C4-\u26C5\u26CE\u26D4\u26EA'
    r'\u26F2-\u26F3\u26F5\u26FA\u26FD\u2702\u2705\u2708-\u270D'
    r'\u270F\u2712\u2714\u2716\u271D\u2721\u2728\u2733-\u2734\u2744'
    r'\u2747\u274C\u274E\u2753-\u2755\u2757\u2763-\u2764\u2795-\u2797'
    r'\u27A1\u27B0\u27BF\u2934-\u2935\u2B05-\u2B07\u2B1B-\u2B1C\u2B50'
    r'\u2B55\u3030\u303D\u3297\u3299]'
    r'[\uFE0E\uFE0F\u20D0-\u20FF\u200D\U0001F3FB-\U0001F3FF]*'
    r'(?:\u200D[\U0001F300-\U0001FFFF\u2600-\u27BF][\uFE0E\uFE0F\u20D0-\u20FF]*)*',
    re.UNICODE
)

found_emojis = set()
for filepath in scan_files:
    try:
        with open(filepath, "r", encoding="utf-8", errors="replace") as fh:
            content = fh.read()
        for m in EMOJI_PATTERN.finditer(content):
            e = m.group(0)
            if e.strip():
                found_emojis.add(e)
    except OSError:
        pass

total_emojis = len(found_emojis)

# ---------------------------------------------------------------------------
# Compute results
# ---------------------------------------------------------------------------
# Normalize ref map keys for lookup (strip variation selectors for comparison)
def normalize(s):
    return s.replace("\ufe0f", "").replace("\ufe0e", "")

ref_normalized = {normalize(k): (k, v) for k, v in EMOJI_REF_MAP.items()}
found_normalized = {normalize(e): e for e in found_emojis}

# unmapped: found in files but not in reference map (by normalized form)
unmapped = sorted([
    raw for norm, raw in found_normalized.items()
    if norm not in ref_normalized
])

# unused: in reference map but not found in files (by normalized form)
unused = sorted([
    canonical for norm, (canonical, concept) in ref_normalized.items()
    if norm not in found_normalized
])

# usage: reference map entries found in files -> concept
usage = {}
for norm, (canonical, concept) in ref_normalized.items():
    if norm in found_normalized:
        usage[canonical] = concept

output = {
    "ok": True,
    "result": {
        "files_scanned": files_scanned,
        "total_emojis": total_emojis,
        "unmapped": unmapped,
        "unused": unused,
        "usage": usage,
    }
}
print(json.dumps(output))
PYEOF
}

# ---------------------------------------------------------------------------
# Entry point when run as a standalone script
# ---------------------------------------------------------------------------
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    _emoji_audit_main "${1:-$(pwd)}"
fi
