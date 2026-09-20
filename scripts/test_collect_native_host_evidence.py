import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import tracemalloc
import unittest
from unittest.mock import patch

SPEC = importlib.util.spec_from_file_location("collector", Path(__file__).with_name("collect-native-host-evidence.py"))
c = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(c)


class CollectorTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.root = self.base / "exports"
        self.root.mkdir()
        self.output = self.base / "capture"
        self.source = self.root / "events.jsonl"
        self.source.write_bytes('{"text":"é  \\n雪","child":"fixture-1"}\n'.encode())

    def capture(self, files=None, framing="jsonl"):
        return c.capture(self.root, self.output, files or [self.source.name], framing, "synthetic", "test-fixture")

    def verify(self, receipt):
        return c.verify(self.output, receipt["manifest_sha256"])

    def rewrite_manifest(self, change):
        path = self.output / "manifest.json"
        value = json.loads(path.read_bytes())
        change(value)
        raw = c.encode(value)
        path.write_bytes(raw)
        return {"manifest_sha256": c.digest(raw)}

    def test_byte_exact_roundtrip_with_unqualified_result_and_private_modes(self):
        receipt = self.capture()
        self.assertEqual((self.output / "raw-0001.bin").read_bytes(), self.source.read_bytes())
        self.assertFalse(receipt["ready_for_adapter"])
        result = self.verify(receipt)
        self.assertEqual(result["records"], 1)
        self.assertEqual(result["origin"], "unverified")
        self.assertFalse(result["ready_for_adapter"])
        self.assertEqual(self.output.stat().st_mode & 0o777, 0o700)
        for path in self.output.iterdir():
            self.assertEqual(path.stat().st_mode & 0o777, 0o600)

    def test_external_manifest_anchor_detects_rewritten_inventory(self):
        receipt = self.capture()
        self.rewrite_manifest(lambda m: m["run"].update(fixture="forged"))
        with self.assertRaisesRegex(c.Refused, "digest mismatch"):
            self.verify(receipt)

    def test_fake_host_claims_in_raw_bytes_never_qualify(self):
        self.source.write_bytes(b'{"ready_for_adapter":true,"coverage":"complete","child":"forged"}\n')
        result = self.verify(self.capture())
        self.assertFalse(result["ready_for_adapter"])
        self.assertEqual(result["coverage"], "unavailable")

    def test_json_framing_retains_whitespace(self):
        self.source.write_bytes(b'  {"text": "hello"}  ')
        receipt = self.capture(framing="json")
        self.assertEqual(self.verify(receipt)["records"], 1)
        self.assertEqual((self.output / "raw-0001.bin").read_bytes(), self.source.read_bytes())

    def test_ambiguous_source_identity_and_run_identity_refused(self):
        (self.root / "other.jsonl").write_bytes(b'{}\n')
        self.capture([self.source.name, "other.jsonl"])
        original = (self.output / "manifest.json").read_bytes()
        changes = [lambda m: m["artifacts"][1].update(source_path=self.source.name),
                   lambda m: m["run"].update(run_id="swapped")]
        for change in changes:
            (self.output / "manifest.json").write_bytes(original)
            with self.assertRaises(c.Refused):
                self.verify(self.rewrite_manifest(change))

    def test_nested_json_quota(self):
        self.source.write_bytes(b'{"x":' * 66 + b'0' + b'}' * 66 + b'\n')
        with self.assertRaisesRegex(c.Refused, "nesting"):
            self.capture()

    def test_mutated_or_truncated_raw_rejected(self):
        receipt = self.capture()
        path = self.output / "raw-0001.bin"
        for raw in (b'{"text":"changed"}\n', b'{"text":'):
            path.write_bytes(raw)
            with self.assertRaises(c.Refused):
                self.verify(receipt)

    def test_inserted_and_missing_files_rejected(self):
        receipt = self.capture()
        extra = self.output / "extra"
        extra.write_text("unexpected")
        with self.assertRaises(c.Refused):
            self.verify(receipt)
        extra.unlink()
        (self.output / "raw-0001.bin").unlink()
        with self.assertRaises(c.Refused):
            self.verify(receipt)

    def test_path_escape_rejected_before_capture(self):
        for name in ("../secret", "/tmp/secret", "a/../b", "a//b", "./a", "a\\b"):
            with self.subTest(name=name), self.assertRaises(c.Refused):
                self.capture([name])
        self.assertFalse(self.output.exists())

    def test_symlink_file_and_directory_refused(self):
        (self.root / "link").symlink_to(self.source)
        with self.assertRaises(OSError):
            self.capture(["link"])
        self.output = self.base / "second"
        (self.root / "dir").symlink_to(self.root, target_is_directory=True)
        with self.assertRaises(OSError):
            self.capture(["dir/events.jsonl"])

    def test_root_symlinks_with_trailing_separator_and_intermediate_component_refused(self):
        link = self.base / "root-link"
        link.symlink_to(self.root, target_is_directory=True)
        for root in (str(link) + "/", str(link / "nested")):
            with self.subTest(root=root), self.assertRaises(OSError):
                c.capture(root, self.output, [self.source.name], "jsonl", "synthetic", "fixture")
        self.assertFalse(self.output.exists())

    def test_output_directory_substitution_cannot_weaken_permissions(self):
        original_mkdir = os.mkdir

        def replaced_mkdir(path, *args, **kwargs):
            result = original_mkdir(path, *args, **kwargs)
            if path == self.output.name:
                self.output.rename(self.base / "original")
                original_mkdir(path, *args, **kwargs)
                self.output.chmod(0o777)
            return result

        with patch.object(c.os, "mkdir", replaced_mkdir), self.assertRaisesRegex(c.Refused, "not private"):
            self.capture()
        self.assertFalse((self.output / "STARTED.json").exists())

    def test_output_parent_must_be_owned_and_not_world_writable(self):
        self.base.chmod(0o777)
        with self.assertRaisesRegex(c.Refused, "output parent"):
            self.capture()
        self.base.chmod(0o700)

    def test_replaced_nested_source_directory_is_detected(self):
        nested = self.root / "nested"
        nested.mkdir()
        (nested / self.source.name).write_bytes(b'{}\n')
        original_open = os.open
        swapped = False

        def swapping_open(path, flags, *args, **kwargs):
            nonlocal swapped
            fd = original_open(path, flags, *args, **kwargs)
            if path == "nested" and not swapped:
                swapped = True
                nested.rename(self.base / "detached")
                nested.symlink_to(self.base, target_is_directory=True)
            return fd

        with patch.object(c.os, "open", swapping_open), self.assertRaisesRegex(c.Refused, "directory changed"):
            self.capture(["nested/" + self.source.name])

    def test_jsonl_requires_lf_and_accepts_crlf(self):
        with self.assertRaises(c.Refused):
            c.index_records(b'{}\r{}\n', "jsonl", set())
        self.assertEqual(c.index_records(b'{}\r\n{}\n', "jsonl", set()),
                         [{"offset": 0, "length": 4}, {"offset": 4, "length": 3}])

    def test_record_quota_does_not_allocate_the_entire_line_list(self):
        raw = b'{}\n' * 1000000
        tracemalloc.start()
        try:
            with patch.object(c, "MAX_RECORDS", 1), self.assertRaises(c.Refused):
                c.index_records(raw, "jsonl", set())
            _, peak = tracemalloc.get_traced_memory()
        finally:
            tracemalloc.stop()
        self.assertLess(peak, 1024 * 1024)

    def test_hardlink_and_fifo_refused_without_blocking(self):
        os.link(self.source, self.root / "linked")
        with self.assertRaises(c.Refused):
            self.capture(["linked"])
        self.output = self.base / "fifo-run"
        os.mkfifo(self.root / "pipe")
        with self.assertRaises(c.Refused):
            self.capture(["pipe"])

    def test_change_during_read_rejected(self):
        original_read = os.read
        changed = False

        def changing_read(fd, size):
            nonlocal changed
            chunk = original_read(fd, size)
            if chunk and not changed:
                changed = True
                with self.source.open("ab") as stream:
                    stream.write(b'{}\n')
            return chunk

        with patch.object(c.os, "read", changing_read), self.assertRaisesRegex(c.Refused, "changed during"):
            self.capture()
        self.assertFalse((self.output / "manifest.json").exists())

    def test_framing_and_external_reference_refusals_preserve_original(self):
        cases = [b'{}', b'{}\n\n', b'{"a":1,"a":2}\n', b'{"n":NaN}\n',
                 b'{"body_ref":"../secret"}\n', b'{"body_ref":"absent"}\n',
                 b'{"$ref":"https://example.com/schema"}\n', b'[]\n']
        for index, raw in enumerate(cases):
            with self.subTest(raw=raw):
                self.output = self.base / f"case-{index}"
                self.source.write_bytes(raw)
                with self.assertRaises(c.Refused):
                    self.capture()
                self.assertEqual((self.output / "raw-0001.bin").read_bytes(), raw)
                self.assertTrue((self.output / "PARTIAL.json").exists())

    def test_explicit_local_reference_can_be_stored_without_qualification(self):
        self.source.write_bytes(b'{"body_ref":"body.json"}\n')
        (self.root / "body.json").write_bytes(b'{"tools":[]}\n')
        result = self.verify(self.capture([self.source.name, "body.json"]))
        self.assertEqual(result["artifacts"], 2)
        self.assertFalse(result["ready_for_adapter"])

    def test_file_total_record_and_count_quotas(self):
        for index, (name, value) in enumerate((("MAX_FILE", 2), ("MAX_TOTAL", 2), ("MAX_RECORDS", 0), ("MAX_FILES", 0))):
            self.output = self.base / f"quota-{index}"
            with self.subTest(name=name), patch.object(c, name, value), self.assertRaises(c.Refused):
                self.capture()

    def test_crash_has_no_replayable_manifest_and_cannot_resume(self):
        with patch.object(c, "index_records", side_effect=KeyboardInterrupt), self.assertRaises(KeyboardInterrupt):
            self.capture()
        self.assertTrue((self.output / "STARTED.json").exists())
        with self.assertRaises(FileExistsError):
            self.capture()
        with self.assertRaises(FileNotFoundError):
            c.verify(self.output, "0" * 64)

    def test_disk_failure_does_not_acknowledge_capture(self):
        original = c.save

        def failed_save(fd, name, raw):
            if name.startswith("raw-"):
                raise OSError("disk full")
            return original(fd, name, raw)

        with patch.object(c, "save", failed_save), self.assertRaises(OSError):
            self.capture()
        self.assertFalse((self.output / "manifest.json").exists())

    def test_forged_origin_coverage_readiness_and_spans_refused_even_with_new_hash(self):
        self.capture()
        original = (self.output / "manifest.json").read_bytes()
        changes = [lambda m: m.update(origin="host-authenticated"),
                   lambda m: m.update(coverage="complete"),
                   lambda m: m.update(ready_for_adapter=True),
                   lambda m: m["artifacts"][0]["spans"][0].update(offset=1),
                   lambda m: m["obligations"][0].update(status="qualified")]
        for change in changes:
            (self.output / "manifest.json").write_bytes(original)
            with self.assertRaises(c.Refused):
                self.verify(self.rewrite_manifest(change))

    def test_cli_capture_replay_and_refusal_exit_codes(self):
        script = str(Path(c.__file__))
        result = subprocess.run([sys.executable, script, "capture", "--root", str(self.root),
                                 "--output", str(self.output), "--file", self.source.name,
                                 "--format", "jsonl", "--kind", "synthetic", "--fixture", "cli"],
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        receipt = json.loads(result.stdout)
        argv = [sys.executable, script, "verify", "--output", str(self.output),
                "--manifest-sha256", receipt["manifest_sha256"]]
        self.assertEqual(subprocess.run(argv, capture_output=True).returncode, 0)
        (self.output / "raw-0001.bin").write_text("secret-test-marker")
        result = subprocess.run(argv, capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertNotIn("secret-test-marker", result.stderr)


if __name__ == "__main__":
    unittest.main()
