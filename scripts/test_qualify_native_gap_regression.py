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

    def test_ignored_new_package_and_whitespace_embed(self):
        self.put('.gitignore', 'newpkg/\n*.ignored\n')
        self.put('newpkg/new.go', 'package newpkg')
        with self.assertRaisesRegex(RuntimeError, 'unadmitted'):
            q.inventory(self.repo)
        (self.repo / 'newpkg/new.go').unlink()
        self.put('main.go', '\t//go:embed\tassets\n')
        self.put('assets/new.ignored', 'asset')
        with self.assertRaisesRegex(RuntimeError, 'unadmitted'):
            q.inventory(self.repo)

    def test_ignored_explicit_imports_refuse_before_preparation(self):
        self.put('.gitignore', '.hidden/\n_hidden/\ntestdata/\n')
        self.put('go.mod', 'module example.test/fixture\n\ngo 1.26\n')
        subprocess.run(['git', 'add', 'go.mod'], cwd=self.repo, check=True)
        for directory in ['.hidden', '_hidden', 'testdata/helper']:
            with self.subTest(directory=directory):
                self.put('main.go', 'package main\nimport _ "example.test/fixture/' + directory + '"\nfunc main() {}\n')
                name = directory + '/helper.go'
                self.put(name, 'package helper\n')
                with self.assertRaisesRegex(RuntimeError, 'unadmitted ignored compilation'):
                    q.inventory(self.repo)
                with tempfile.TemporaryDirectory() as external:
                    root = Path(external) / 'preparation'
                    with patch.object(q, 'output', return_value='fixture'), patch.object(q.shutil, 'copytree') as copy, patch.object(q, 'record') as launch:
                        self.assertEqual(q.prepare(None, self.repo, root), 1)
                        receipt = q.json.loads((root / 'manifest.json').read_text())
                        self.assertFalse(receipt['passed'])
                        self.assertIn('unadmitted ignored compilation', receipt['error'])
                        copy.assert_not_called()
                        launch.assert_not_called()
                (self.repo / name).unlink()

    def test_ignored_addition_inside_tracked_hidden_package_refuses(self):
        self.put('.gitignore', '.scratch/\n')
        self.put('.scratch/tracked.go', 'package scratch')
        subprocess.run(['git', 'add', '-f', '.scratch/tracked.go'], cwd=self.repo, check=True)
        self.put('.scratch/generated.go', 'package scratch')
        with self.assertRaisesRegex(RuntimeError, 'unadmitted ignored compilation'):
            q.inventory(self.repo)

    def test_validate_source_and_both_prepared_lanes_before_launch(self):
        self.put('.gitignore', '.hidden/\n_hidden/\ntestdata/\n')
        with tempfile.TemporaryDirectory() as external:
            root = Path(external)
            binary = root / 'binary'
            binary.write_text('binary')
            log = root / 'log'
            log.write_text('retained output')
            result = {'stdout': q.file_ref(log), 'stderr': q.file_ref(log)}
            lanes = {}
            for name in ['normal', 'race']:
                clone = root / name
                q.shutil.copytree(self.repo, clone)
                lanes[name] = {'clone': str(clone), 'root': str(root), 'env': {}, 'fixture': result, 'build': result, 'discovery_probe': result, 'candidate': q.file_ref(binary)}
            manifest = {'passed': True, 'script': q.file_ref(Path(q.__file__)), 'identity': q.inventory(self.repo), 'go_executable': q.file_ref(binary), 'tools': {}, 'lanes': lanes, 'disposed_commits': []}
            q.write_new(root / 'manifest.json', manifest)
            q.write_new(root / 'manifest.sha256.json', q.file_ref(root / 'manifest.json'))
            self.assertEqual(manifest, q.validate(self.repo, root))
            for repo in [self.repo, root / 'normal', root / 'race']:
                for name in ['new.go', 'new_test.go', 'assets/new.txt', '.hidden/helper.go', '_hidden/helper.go', 'testdata/helper/helper.go']:
                    with self.subTest(repo=repo, name=name):
                        target = repo / name
                        target.parent.mkdir(parents=True, exist_ok=True)
                        target.write_text('unadmitted')
                        with patch.object(q, 'record') as launch:
                            with self.assertRaisesRegex(RuntimeError, 'unadmitted'):
                                q.run(None, self.repo, root)
                            launch.assert_not_called()
                        target.unlink()

    def test_symlink_refused(self):
        (self.repo / 'main.go').unlink()
        (self.repo / 'main.go').symlink_to('/etc/hosts')
        with self.assertRaisesRegex(RuntimeError, 'symlink'):
            q.inventory(self.repo)

    def test_normal_embed_skips_hidden_descendants_but_all_consumes_them(self):
        self.put('.gitignore', '.DS_Store\n_hidden/\n')
        before = q.inventory(self.repo)
        self.put('assets/.DS_Store', 'Finder metadata')
        self.put('assets/_hidden/nested.txt', 'hidden asset')
        self.assertEqual(before, q.inventory(self.repo))
        self.put('main.go', '//go:embed all:assets\n')
        with self.assertRaisesRegex(RuntimeError, 'unadmitted embedded'):
            q.inventory(self.repo)

    def test_explicit_hidden_embed_match_is_consumed(self):
        self.put('.gitignore', '.DS_Store\n')
        self.put('assets/.DS_Store', 'explicitly consumed')
        self.put('main.go', '//go:embed assets/.DS_Store\n')
        with self.assertRaisesRegex(RuntimeError, 'unadmitted embedded'):
            q.inventory(self.repo)

    def test_hidden_only_directory_cannot_satisfy_normal_embed(self):
        self.put('.gitignore', '.DS_Store\n')
        self.put('assets/.DS_Store', 'ignored Finder metadata')
        (self.repo / 'assets/a.txt').unlink()
        with self.assertRaisesRegex(RuntimeError, 'empty embed input'):
            q.inventory(self.repo)

    def test_storage_uses_current_measurements_without_retired_cache(self):
        history = self.repo / 'old-manifest.json'
        candidate = self.repo / 'candidate'
        candidate.write_text('binary')
        history.write_text(q.json.dumps({'disk_free_before_copy': 1000, 'lanes': {'normal': {'candidate': q.file_ref(candidate)}}}))
        retirement = self.repo / 'retirement.json'
        retirement.write_text(q.json.dumps({'bytes': 100, 'manifest': q.file_ref(history)}))
        self.put('.planning/phases/204.2-codex-native-worker-lifecycle/evidence/gap-closure/native-regression.json',
                 q.json.dumps({'preflight_manifest': q.file_ref(history), 'cache_retirement': q.file_ref(retirement)}))
        for name in ['modules/data', 'current-cache/data', 'toolchain/data']:
            self.put(name, 'measurable bytes')
        env = {'GOCACHE': str(self.repo / 'current-cache'), 'GOROOT': str(self.repo / 'toolchain')}
        def answer(argv, *args):
            return q.json.dumps(env) if argv[0] == 'go' else '.git'
        with patch.object(q, 'output', side_effect=answer):
            estimate = q.storage_estimate(self.repo, self.repo / 'modules')
        self.assertGreater(estimate['required_bytes'], 1000)
        self.assertEqual(estimate['measured_bytes']['toolchain'], 16)
        self.assertEqual(estimate['measured_bytes']['prior_compiler_cache'], 100)
        self.assertEqual(estimate['historical_manifest'], q.file_ref(history))
        retirement.write_text('tampered historical measurement')
        with patch.object(q, 'output', side_effect=answer):
            refused = q.storage_estimate(self.repo, self.repo / 'modules')
        self.assertIsNone(refused['required_bytes'])
        self.assertIn('cache retirement identity mismatch', refused['uncertainties'][0])

    def test_unreadable_tree_cannot_be_zero_capacity(self):
        def denied(path, onerror=None, **kwargs):
            onerror(PermissionError('cannot measure tree'))
            return iter(())
        with patch.object(q.os, 'walk', side_effect=denied):
            with self.assertRaisesRegex(PermissionError, 'cannot measure'):
                q.tree_bytes(self.repo)

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

    def test_uncertain_capacity_cannot_pass(self):
        manifest = {'storage_estimate': {'required_bytes': None, 'uncertainties': ['missing measured cache']}}
        with patch.object(q.shutil, 'disk_usage', return_value=types.SimpleNamespace(free=10**15)):
            with self.assertRaisesRegex(RuntimeError, 'uncertain storage'):
                q.storage_admit(manifest, self.repo, 'module-copy')
        self.assertFalse(manifest['storage_admissions'][0]['passed'])

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
