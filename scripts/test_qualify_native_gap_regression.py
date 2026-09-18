import importlib.util
import subprocess
import tempfile
import types
import errno
import unittest
from pathlib import Path
from unittest.mock import patch

spec = importlib.util.spec_from_file_location('qualification', Path(__file__).with_name('qualify-native-gap-regression.py'))
q = importlib.util.module_from_spec(spec)
spec.loader.exec_module(q)


class AdmissionTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.repo = Path(self.tmp.name)
        subprocess.run(['git', 'init', '-q', str(self.repo)], check=True)
        for name in [q.HARNESS, q.CONTRACT, 'main.go', 'assets/a.txt']:
            self.put(name, '//go:embed assets\n' if name == 'main.go' else 'baseline')
        self.put('.gitignore', '*.ignored\n')
        subprocess.run(['git', 'add', '.'], cwd=self.repo, check=True)

    def put(self, name, data):
        path = self.repo / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(data)

    def test_untracked_inputs_rejected(self):
        for name in ['extra.go', 'extra_test.go', 'assets/new.txt', 'unknown.txt', 'assets/new.ignored']:
            with self.subTest(name=name):
                self.put(name, 'new')
                with self.assertRaisesRegex(RuntimeError, 'unadmitted'):
                    q.inventory(self.repo)
                (self.repo / name).unlink()

    def test_only_explicit_metadata_excluded(self):
        before = q.inventory(self.repo)
        self.put('.gsd/.agents/skills/to-prd/SKILL.md', 'metadata')
        self.assertEqual(before, q.inventory(self.repo))
        self.put('.gsd/hidden.go', 'package hidden')
        with self.assertRaises(RuntimeError):
            q.inventory(self.repo)

    def test_symlink_refused(self):
        (self.repo / 'main.go').unlink()
        (self.repo / 'main.go').symlink_to('/etc/hosts')
        with self.assertRaisesRegex(RuntimeError, 'symlink'):
            q.inventory(self.repo)

    def test_deletion_is_pinned(self):
        before = q.inventory(self.repo)
        (self.repo / q.HARNESS).unlink()
        with self.assertRaises((RuntimeError, FileNotFoundError)):
            q.inventory(self.repo)
        self.put(q.HARNESS, 'baseline')
        (self.repo / 'assets/a.txt').unlink()
        with self.assertRaisesRegex(RuntimeError, 'embed'):
            q.inventory(self.repo)

    def test_each_launch_revalidates(self):
        manifest = {'lanes': {'normal': {'clone': str(self.repo), 'root': str(self.repo), 'env': {}}}}
        with patch.object(q, 'validate', side_effect=[manifest, RuntimeError('unadmitted lane')]), patch.object(q, 'record') as launch:
            with self.assertRaisesRegex(RuntimeError, 'unadmitted lane'):
                q.run(None, self.repo, self.repo)
            launch.assert_not_called()

    def test_capacity_refusal_retained_without_copy(self):
        root = self.repo / 'evidence'
        # Preparation metadata queries are mocked; storage admission itself is real.
        def answer(argv, *args):
            if argv == ['go', 'env', '-json']:
                return '{"GOMODCACHE":"/unused"}'
            return 'fixture'
        estimate = {'required_bytes': 100, 'basis': 'measured fixture'}
        with patch.object(q, 'inventory', return_value={}), patch.object(q, 'objects', return_value=[]), patch.object(q, 'output', side_effect=answer), patch.object(q, 'file_ref', return_value={}), patch.object(q.shutil, 'which', return_value=None), patch.object(q.subprocess, 'check_output', return_value=b''), patch.object(q, 'storage_estimate', return_value=estimate), patch.object(q.shutil, 'disk_usage', return_value=types.SimpleNamespace(free=1)), patch.object(q.shutil, 'copytree') as copy, patch.object(q, 'record') as launch:
            self.assertEqual(q.prepare(None, self.repo, root), 1)
            manifest = q.json.loads((root / 'manifest.json').read_text())
            self.assertFalse(manifest['passed'])
            self.assertIn('insufficient storage', manifest['error'])
            copy.assert_not_called()
            launch.assert_not_called()

    def test_enospc_survives_later_inventory_failure(self):
        root = self.repo / 'evidence'
        def answer(argv, *args):
            return '{"GOMODCACHE":"/unused"}' if argv == ['go', 'env', '-json'] else 'fixture'
        with patch.object(q, 'inventory', side_effect=[{}, RuntimeError('later inventory failure')]), patch.object(q, 'objects', return_value=[]), patch.object(q, 'output', side_effect=answer), patch.object(q, 'file_ref', return_value={}), patch.object(q.shutil, 'which', return_value=None), patch.object(q.subprocess, 'check_output', return_value=b''), patch.object(q, 'storage_estimate', return_value={'required_bytes': 1}), patch.object(q.shutil, 'disk_usage', return_value=types.SimpleNamespace(free=100)), patch.object(q.shutil, 'copytree', side_effect=OSError(errno.ENOSPC, 'original copy exhaustion')), patch.object(q, 'record') as launch:
            self.assertEqual(q.prepare(None, self.repo, root), 1)
            manifest = q.json.loads((root / 'manifest.json').read_text())
            self.assertIn('original copy exhaustion', manifest['error'])
            self.assertIn('later inventory failure', str(manifest['secondary_errors']))
            self.assertFalse(manifest['passed'])
            launch.assert_not_called()


if __name__ == '__main__':
    unittest.main()
