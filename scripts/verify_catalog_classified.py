#!/usr/bin/env python3
"""Validate that every entry in cmd/testdata/command_catalog.json has required
classification fields.  Exit 0 if all pass, 1 if any fail.

Usage:
    python3 scripts/verify_catalog_classified.py [--strict]
"""

import argparse
import json
import sys
from pathlib import Path

CATALOG_PATH = Path("cmd/testdata/command_catalog.json")

VALID_CLASSIFICATIONS = {
    "public_lifecycle",
    "public_utility",
    "internal_runtime",
    "alias",
    "deprecated",
}

STRICT_RANGES = {
    "public_lifecycle": (10, 25),
    "alias": (5, 15),
    "internal_runtime": (0, 5),
    "deprecated": (0, 10),
}


def validate(catalog: list, strict: bool = False) -> dict:
    total = len(catalog)
    missing_cls = []
    invalid_cls = []
    missing_since = []
    missing_hist = []
    short_hist = []

    for entry in catalog:
        name = entry.get("name", "<unknown>")

        cls = entry.get("classification")
        if cls is None:
            missing_cls.append(name)
        elif cls not in VALID_CLASSIFICATIONS:
            invalid_cls.append((name, cls))

        since = entry.get("since_version")
        if since is None or (isinstance(since, str) and since.strip() == ""):
            missing_since.append(name)

        hist = entry.get("historical_presence")
        if hist is None:
            missing_hist.append(name)
        elif not isinstance(hist, dict) or len(hist) < 5:
            short_hist.append((name, len(hist) if isinstance(hist, dict) else 0))

    counts = {}
    for entry in catalog:
        c = entry.get("classification", "UNKNOWN")
        counts[c] = counts.get(c, 0) + 1

    range_violations = []
    if strict:
        for cls, (lo, hi) in STRICT_RANGES.items():
            cnt = counts.get(cls, 0)
            if cnt < lo or cnt > hi:
                range_violations.append((cls, cnt, lo, hi))

    return {
        "total": total,
        "missing_cls": missing_cls,
        "invalid_cls": invalid_cls,
        "missing_since": missing_since,
        "missing_hist": missing_hist,
        "short_hist": short_hist,
        "counts": counts,
        "range_violations": range_violations,
    }


def report(result: dict, strict: bool) -> bool:
    ok = True
    print(f"Total commands: {result['total']}")
    print(f"  Missing classification: {len(result['missing_cls'])}")
    print(f"  Invalid classification: {len(result['invalid_cls'])}")
    print(f"  Missing since_version: {len(result['missing_since'])}")
    print(f"  Missing historical_presence: {len(result['missing_hist'])}")
    print(f"  historical_presence < 5 keys: {len(result['short_hist'])}")

    if result["missing_cls"]:
        ok = False
        print("\nMissing classification:")
        for name in result["missing_cls"]:
            print(f"  - {name}")

    if result["invalid_cls"]:
        ok = False
        print("\nInvalid classification values:")
        for name, val in result["invalid_cls"]:
            print(f"  - {name}: {val}")

    if result["missing_since"]:
        ok = False
        print("\nMissing since_version:")
        for name in result["missing_since"]:
            print(f"  - {name}")

    if result["missing_hist"]:
        ok = False
        print("\nMissing historical_presence:")
        for name in result["missing_hist"]:
            print(f"  - {name}")

    if result["short_hist"]:
        ok = False
        print("\nShort historical_presence (< 5 version keys):")
        for name, cnt in result["short_hist"]:
            print(f"  - {name}: {cnt} keys")

    print("\nClassification counts:")
    for cls in sorted(result["counts"]):
        print(f"  {cls}: {result['counts'][cls]}")

    if strict and result["range_violations"]:
        ok = False
        print("\nStrict range violations:")
        for cls, cnt, lo, hi in result["range_violations"]:
            print(f"  - {cls}: {cnt} (expected {lo}-{hi})")

    if ok:
        print("\nAll checks passed.")
    else:
        print("\nSome checks failed.")
    return ok


def main():
    parser = argparse.ArgumentParser(description="Verify command catalog classification")
    parser.add_argument("--strict", action="store_true", help="Enforce classification count ranges")
    parser.add_argument("--catalog", type=Path, default=CATALOG_PATH, help="Path to catalog JSON")
    args = parser.parse_args()

    catalog_file = args.catalog
    if not catalog_file.exists():
        print(f"Catalog not found: {catalog_file}", file=sys.stderr)
        sys.exit(1)

    with open(catalog_file, "r") as f:
        catalog = json.load(f)

    result = validate(catalog, strict=args.strict)
    ok = report(result, strict=args.strict)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
