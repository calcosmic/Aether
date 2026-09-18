#!/usr/bin/env python3
"""Offline, fixture-only evidence storage. No host qualification or live collection.

capture --root EXPORT_ROOT --output NEW_DIR --kind synthetic|unqualified
        --fixture NAME --format json|jsonl --file RELATIVE_PATH [--file ...]
verify --output DIR --manifest-sha256 DIGEST

Keep the returned manifest digest outside the capture directory. It pins the
inventory, not host authenticity. Exit 0 means storage/replay succeeded only;
ready_for_adapter is always false. Exit 1 leaves a partial capture for inspection.
No resume/overwrite, external reference fetching, decompression or host decoding.
Limits: 16 MiB/file, 64 MiB/run, 128 files, 10,000 records, 64 JSON levels.
These storage quotas do not change the downstream 2 MiB context proof limit.
"""

import argparse
import hashlib
import json
import os
from pathlib import PurePosixPath
import re
import stat
import sys
import uuid
from datetime import datetime, timezone
from contextlib import contextmanager

SCHEMA = "aether-scoped-collector/v1"
MAX_FILE = 16 * 1024 * 1024
MAX_TOTAL = 64 * 1024 * 1024
MAX_FILES = 128
MAX_RECORDS = 10000
MAX_MANIFEST = 1024 * 1024
OBLIGATIONS = ("settings", "tools", "context", "answer_ack", "cancelled", "no_write_interval")


class Refused(ValueError):
    pass


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encode(value):
    return (json.dumps(value, sort_keys=True, ensure_ascii=False, allow_nan=False) + "\n").encode()


def relative(name):
    if not isinstance(name, str) or not name or "\\" in name or "\x00" in name:
        raise Refused("invalid relative path")
    path = PurePosixPath(name)
    if path.is_absolute() or any(p in ("", ".", "..") for p in name.split("/")):
        raise Refused("path escape or noncanonical path")
    if len(path.parts) > 64 or len(name) > 4096:
        raise Refused("path quota exceeded")
    return path.parts


def fingerprint(info):
    return (info.st_dev, info.st_ino, info.st_size, info.st_mtime_ns, info.st_ctime_ns)


def check_directory_links(links):
    for parent, name, child in links:
        actual = os.fstat(child)
        named = os.stat(name, dir_fd=parent, follow_symlinks=False)
        if not stat.S_ISDIR(named.st_mode) or (actual.st_dev, actual.st_ino) != (named.st_dev, named.st_ino):
            raise Refused("directory changed during operation")


@contextmanager
def pinned_directory(path, create=False):
    # Only Apple's root-owned standard aliases are canonicalized. Arbitrary
    # symlink components, including the final component with '/', are refused.
    path = os.fspath(path)
    if ".." in path.split("/"):
        raise Refused("noncanonical directory path")
    absolute = os.path.abspath(path)
    if sys.platform == "darwin":
        for alias in ("/tmp", "/var", "/etc"):
            if (absolute == alias or absolute.startswith(alias + "/")) and os.path.realpath(alias) == "/private" + alias:
                absolute = "/private" + absolute
                break
    parts = relative(absolute.lstrip("/"))
    handles = [os.open("/", os.O_RDONLY | os.O_DIRECTORY)]
    links = []
    try:
        for index, part in enumerate(parts):
            parent = handles[-1]
            creating = create and index == len(parts) - 1
            if creating:
                parent_info = os.fstat(parent)
                if parent_info.st_uid != os.getuid() or parent_info.st_mode & 0o022:
                    raise Refused("output parent must be owned and not group/world writable")
                os.mkdir(part, 0o700, dir_fd=parent)
                os.fsync(parent)
            named = os.stat(part, dir_fd=parent, follow_symlinks=False)
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
            handles.append(child)
            actual = os.fstat(child)
            if not stat.S_ISDIR(named.st_mode) or (actual.st_dev, actual.st_ino) != (named.st_dev, named.st_ino):
                raise Refused("directory replaced before open")
            if creating and (stat.S_IMODE(actual.st_mode) != 0o700 or actual.st_uid != os.getuid()):
                raise Refused("output directory is not private")
            links.append((parent, part, child))
        yield handles[-1]
        check_directory_links(links)
        if create and stat.S_IMODE(os.fstat(handles[-1]).st_mode) != 0o700:
            raise Refused("output directory permissions changed")
    finally:
        for handle in reversed(handles):
            os.close(handle)


def read_owned(root_fd, name, limit):
    """Walk using no-follow directory handles; refuse races and special files."""
    parts = relative(name)
    handles = [os.dup(root_fd)]
    links = []
    try:
        for part in parts[:-1]:
            parent = handles[-1]
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
            handles.append(child)
            links.append((parent, part, child))
        parent = handles[-1]
        fd = os.open(parts[-1], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        try:
            before = os.fstat(fd)
            if not stat.S_ISREG(before.st_mode) or before.st_nlink != 1:
                raise Refused("source must be a regular unlinked-to-other-paths file")
            if before.st_size > limit:
                raise Refused("byte quota exceeded")
            chunks, size = [], 0
            while True:
                chunk = os.read(fd, min(65536, limit + 1 - size))
                if not chunk:
                    break
                chunks.append(chunk)
                size += len(chunk)
                if size > limit:
                    raise Refused("byte quota exceeded")
            after = os.fstat(fd)
            named = os.stat(parts[-1], dir_fd=parent, follow_symlinks=False)
            if fingerprint(before) != fingerprint(after) or fingerprint(after) != fingerprint(named) or size != before.st_size:
                raise Refused("source changed during import")
            check_directory_links(links)
            return b"".join(chunks)
        finally:
            os.close(fd)
    finally:
        for handle in reversed(handles):
            os.close(handle)


def save(root_fd, name, data):
    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600, dir_fd=root_fd)
    try:
        with os.fdopen(fd, "wb", closefd=False) as stream:
            stream.write(data)
            stream.flush()
            os.fsync(fd)
    finally:
        os.close(fd)
    os.fsync(root_fd)


def pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise Refused("duplicate JSON key")
        result[key] = value
    return result


def load_json(raw):
    def invalid(_):
        raise Refused("nonfinite JSON value")
    try:
        return json.loads(raw.decode("utf-8"), object_pairs_hook=pairs, parse_constant=invalid)
    except (UnicodeError, ValueError, RecursionError) as exc:
        raise Refused("invalid or ambiguous JSON") from exc


def inspect_value(value, files, depth=0):
    if depth > 64:
        raise Refused("JSON nesting quota exceeded")
    if isinstance(value, dict):
        for key, item in value.items():
            if key in ("body_ref", "$ref"):
                # Reference syntax requires a vendor adapter. Only exact,
                # allowlisted local paths are accepted; never dereference here.
                relative(item)
                if item not in files:
                    raise Refused("external or unlisted reference")
            inspect_value(item, files, depth + 1)
    elif isinstance(value, list):
        for item in value:
            inspect_value(item, files, depth + 1)


def framed_records(raw, framing):
    if framing == "json":
        yield 0, raw
    elif framing == "jsonl":
        if not raw.endswith(b"\n"):
            raise Refused("unterminated JSONL record")
        offset = 0
        while offset < len(raw):
            end = raw.find(b"\n", offset) + 1
            # LF is the only separator; JSON's whitespace rules allow CRLF.
            yield offset, raw[offset:end]
            offset = end
    else:
        raise Refused("unknown framing")


def index_records(raw, framing, files):
    spans = []
    for offset, record in framed_records(raw, framing):
        if len(spans) >= MAX_RECORDS:
            raise Refused("record quota exceeded")
        if not record.strip():
            raise Refused("empty JSONL record")
        value = load_json(record)
        if not isinstance(value, dict):
            raise Refused("record must be a JSON object")
        inspect_value(value, files)
        spans.append({"offset": offset, "length": len(record)})
    return spans


def unavailable():
    return [{"host": host, "obligation": item, "status": "unavailable"}
            for host in ("codex", "claude") for item in OBLIGATIONS]


def capture(root, output, files, framing, kind, fixture):
    if kind not in ("synthetic", "unqualified") or not fixture or len(fixture) > 256:
        raise Refused("explicit bounded fixture identity and evidence kind required")
    if not 0 < len(files) <= MAX_FILES or len(set(files)) != len(files):
        raise Refused("file quota or duplicate input")
    for name in files:
        relative(name)
    with pinned_directory(root) as root_fd:
        with pinned_directory(output, create=True) as out_fd:
            run = {"run_id": str(uuid.uuid4()), "fixture": fixture, "kind": kind,
                   "started_at": datetime.now(timezone.utc).isoformat()}
            save(out_fd, "STARTED.json", encode(run))
            artifacts, total, count = [], 0, 0
            try:
                for sequence, name in enumerate(files, 1):
                    raw = read_owned(root_fd, name, min(MAX_FILE, MAX_TOTAL - total))
                    stored = f"raw-{sequence:04d}.bin"
                    save(out_fd, stored, raw)  # Original bytes precede parsing.
                    spans = index_records(raw, framing, set(files))
                    total += len(raw)
                    count += len(spans)
                    if count > MAX_RECORDS:
                        raise Refused("run record quota exceeded")
                    artifacts.append({"source_path": name, "stored_path": stored,
                                      "size": len(raw), "sha256": digest(raw),
                                      "sequence": sequence, "framing": framing, "spans": spans})
                manifest = {"schema_version": SCHEMA, "run": run, "artifacts": artifacts,
                            "status": "captured", "ready_for_adapter": False,
                            "origin": "unverified", "coverage": "unavailable",
                            "obligations": unavailable(), "errors": []}
                encoded = encode(manifest)
                if len(encoded) > MAX_MANIFEST:
                    raise Refused("manifest quota exceeded")
                save(out_fd, "manifest.json", encoded)
                return {"status": "captured", "manifest_sha256": digest(encoded),
                        "ready_for_adapter": False}
            except (Exception, KeyboardInterrupt):
                # STARTED without a valid pinned manifest is already partial,
                # including when disk failure prevents this best-effort marker.
                try:
                    save(out_fd, "PARTIAL.json", encode({"status": "partial", "ready_for_adapter": False}))
                except OSError:
                    pass
                raise


def verify(output, expected_digest):
    if not re.fullmatch(r"[0-9a-f]{64}", expected_digest):
        raise Refused("external manifest digest required")
    with pinned_directory(output) as fd:
        raw_manifest = read_owned(fd, "manifest.json", MAX_MANIFEST)
        if digest(raw_manifest) != expected_digest:
            raise Refused("manifest digest mismatch")
        manifest = load_json(raw_manifest)
        if (not isinstance(manifest, dict) or manifest.get("schema_version") != SCHEMA or manifest.get("status") != "captured"
                or manifest.get("ready_for_adapter") is not False
                or manifest.get("origin") != "unverified" or manifest.get("coverage") != "unavailable"
                or manifest.get("obligations") != unavailable() or manifest.get("errors") != []):
            raise Refused("unsupported qualification or partial manifest")
        run = manifest["run"]
        if run["kind"] not in ("synthetic", "unqualified") or load_json(read_owned(fd, "STARTED.json", MAX_MANIFEST)) != run:
            raise Refused("run identity mismatch")
        artifacts = manifest["artifacts"]
        if not 0 < len(artifacts) <= MAX_FILES:
            raise Refused("file quota exceeded")
        names = [a["source_path"] for a in artifacts]
        if len(set(names)) != len(names):
            raise Refused("ambiguous source identity")
        for name in names:
            relative(name)
        expected = {"STARTED.json", "manifest.json"} | {f"raw-{i:04d}.bin" for i in range(1, len(artifacts) + 1)}
        if set(os.listdir(fd)) != expected:
            raise Refused("inventory insertion, omission or partial capture")
        total, count = 0, 0
        for sequence, artifact in enumerate(artifacts, 1):
            if artifact["sequence"] != sequence or artifact["stored_path"] != f"raw-{sequence:04d}.bin":
                raise Refused("ambiguous inventory ordering")
            raw = read_owned(fd, artifact["stored_path"], min(MAX_FILE, MAX_TOTAL - total))
            spans = index_records(raw, artifact["framing"], set(names))
            if len(raw) != artifact["size"] or digest(raw) != artifact["sha256"] or spans != artifact["spans"]:
                raise Refused("raw bytes or source pointer mismatch")
            total += len(raw)
            count += len(spans)
            if count > MAX_RECORDS:
                raise Refused("run record quota exceeded")
        if set(os.listdir(fd)) != expected:
            raise Refused("inventory changed during replay")
        return {"status": "replayed", "artifacts": len(artifacts), "records": count,
                "ready_for_adapter": False, "origin": "unverified", "coverage": "unavailable"}


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    commands = parser.add_subparsers(dest="command", required=True)
    cap = commands.add_parser("capture")
    cap.add_argument("--root", required=True)
    cap.add_argument("--output", required=True)
    cap.add_argument("--file", action="append", required=True)
    cap.add_argument("--format", choices=("json", "jsonl"), required=True)
    cap.add_argument("--kind", choices=("synthetic", "unqualified"), required=True)
    cap.add_argument("--fixture", required=True)
    replay = commands.add_parser("verify")
    replay.add_argument("--output", required=True)
    replay.add_argument("--manifest-sha256", required=True)
    args = parser.parse_args()
    try:
        if args.command == "capture":
            result = capture(args.root, args.output, args.file, args.format, args.kind, args.fixture)
        else:
            result = verify(args.output, args.manifest_sha256)
        print(json.dumps(result, sort_keys=True))
        return 0
    except (OSError, ValueError, KeyError, TypeError, RecursionError) as exc:
        # Do not emit raw source content or arbitrary credential-bearing paths.
        print(json.dumps({"status": "refused", "error_type": type(exc).__name__, "ready_for_adapter": False}), file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
