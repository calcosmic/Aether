#!/usr/bin/env python3
"""Prepare isolated regression lanes; execute exact suites; replay Go evidence checks.

No receipt is native-host proof. --prepare never executes a broad suite. --run
requires the immutable passing preflight and unchanged source/corpus/toolchain.
All generated files are confined to a new explicit evidence root. Existing run
receipts are never overwritten. A failed check is retained, not waived.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import signal
import shlex
import subprocess
import sys
import time

PREFLIGHT = '^(TestCLIInterruptedBuildResumesThroughForceRedispatch|TestCLIInterruptedBuildReadinessCleanup|TestPackedNPMAncestorIsolation|TestPackedNPMReleaseCandidateContract|TestDisposedBranchesAreGoneFromTheRepository)$'
HARNESS = 'cmd/codex_native_worker_live_test.go'
CONTRACT = 'cmd/codex_native_evidence_test.go'
PROBE = r'''package cmd
import ("os"; "path/filepath"; "testing")
func TestPlan13EvidenceProbe(t *testing.T) {
 root := antSkillSourceRoot(t)
 out := os.Getenv("AETHER_GAP_PROBE_OUT")
 var invocations map[string][]string
 if err := nativeEvidenceReadJSON(filepath.Join(out,"argv.json"), &invocations); err != nil { t.Fatal(err) }
 for name, argv := range invocations { if err := nativeEvidenceRunContract(name,argv); err != nil { t.Fatal(err) } }
 all, err := nativeEvidenceCurrentDiscovery(root)
 if err != nil { t.Fatal(err) }
 liveSkillWriteJSON(t, filepath.Join(out,"discovery.json"), all)
 for _, name := range []string{"focused_normal","focused_race","normal","race"} {
  liveSkillWriteJSON(t,filepath.Join(out,"discovery-"+name+".json"),nativeEvidenceExpectedDiscovery(name,all))
 }
 if receipt := os.Getenv("AETHER_GAP_PROBE_RECEIPT"); receipt != "" {
  var regression nativeEvidenceRegression
  if err := nativeEvidenceReadJSON(receipt,&regression); err != nil { t.Fatal(err) }
  results := map[string]string{}
  for _, name := range []string{"focused_normal","focused_race","normal","race"} {
   run,ok := regression.Runs[name]
   if !ok { results[name]="missing run"; t.Errorf("%s missing",name); continue }
   if err := nativeEvidenceRegressionRunCheck(name,run,all); err != nil {
    results[name]=err.Error(); t.Errorf("%s: %v",name,err)
   } else { results[name]="pass" }
  }
  liveSkillWriteJSON(t,filepath.Join(out,"contract-results.json"),results)
 }
}
'''


def digest(data):
    return 'sha256:' + hashlib.sha256(data).hexdigest()


def file_ref(path):
    return {'path': str(path), 'sha256': digest(Path(path).read_bytes())}


def write_new(path, value):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('x') as stream:
        json.dump(value, stream, indent=2, sort_keys=True)
        stream.write('\n')


def output(argv, cwd=None, env=None):
    return subprocess.check_output(argv, cwd=cwd, env=env, text=True, timeout=90).strip()


def record(root, label, argv, cwd, env, bound):
    """Raw stdout/stderr, exact argv and wall bound; stop only our process group."""
    out, err = root / (label + '.stdout'), root / (label + '.stderr')
    start = time.time()
    timed_out = False
    with out.open('xb') as stdout, err.open('xb') as stderr:
        child = subprocess.Popen(argv, cwd=cwd, env=env, stdout=stdout,
                                 stderr=stderr, start_new_session=True)
        try:
            code = child.wait(timeout=bound)
        except (subprocess.TimeoutExpired, KeyboardInterrupt):
            timed_out = True
            os.killpg(child.pid, signal.SIGTERM)
            try:
                child.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(child.pid, signal.SIGKILL)
                child.wait()
            code = child.returncode
    receipt = {'argv': argv, 'cwd': str(cwd), 'exit_code': code,
               'started_epoch': start, 'elapsed_seconds': time.time()-start,
               'bound_seconds': bound, 'timed_out': timed_out,
               'stdout': file_ref(out), 'stderr': file_ref(err)}
    write_new(root / (label + '.json'), receipt)
    print(label, 'exit', code, flush=True)
    return receipt


def admit_inputs(source, paths):
    """Reject inputs omitted by git diff; inspect ignored embeds without a build."""
    # Exact GSD instruction/dispatch metadata, outside current Go inputs. The
    # embed scan below still refuses these if a source begins consuming them.
    metadata = {'.gsd/.agents/skills/to-prd/SKILL.md', '.gsd/dispatch-isolation-sentinel.json'}
    metadata.update({'.gsd/scratch/blocker1.txt', '.gsd/scratch/decision1.txt', '.gsd/scratch/decision2.txt'})
    metadata.update('.gsd/worktrees/phase-198-3/wave-' + str(n) + '-manifest.json' for n in range(1, 8))
    untracked = subprocess.check_output(
        ['git', 'ls-files', '--others', '--exclude-standard', '-z'], cwd=source).decode().split('\0')
    for name in untracked:
        if name and name not in metadata:
            raise RuntimeError('unadmitted untracked input: ' + name)
    tracked = set(paths)
    metadata_links = {'.claude/skills/to-prd', '.crush/skills/to-prd', 'skills/to-prd'}
    for name in paths:
        if not name:
            continue
        path = source / name
        if path.is_symlink():
            if name not in metadata_links:
                raise RuntimeError('symlink input requires explicit admission: ' + name)
            continue
        if not name.endswith('.go') or not path.exists():
            continue
        for line in path.read_text().splitlines():
            directive = re.match(r'^\s*//go:embed\s+(.+)$', line)
            if not directive:
                continue
            for pattern in shlex.split(directive[1]):
                include_hidden = pattern.startswith('all:')
                pattern = pattern.removeprefix('all:')
                if '..' in Path(pattern).parts or Path(pattern).is_absolute():
                    raise RuntimeError('unsupported embed pattern: ' + pattern)
                matches = list(path.parent.glob(pattern))
                if not matches:
                    raise RuntimeError('missing embed input: ' + pattern)
                for match in matches:
                    members = [match, *match.rglob('*')] if match.is_dir() else [match]
                    consumed = 0
                    for member in members:
                        # Go skips dot/underscore descendants when walking a
                        # matched directory, not explicitly matched files.
                        if member != match and not include_hidden and any(
                                part.startswith(('.', '_')) for part in member.relative_to(match).parts):
                            continue
                        relative = member.relative_to(source).as_posix()
                        if member.is_symlink():
                            raise RuntimeError('symlink embed input: ' + relative)
                        if member.is_file() and relative not in tracked:
                            raise RuntimeError('unadmitted embedded input: ' + relative)
                        consumed += member.is_file()
                    if not consumed:
                        raise RuntimeError('empty embed input: ' + pattern)
    # Ignored compilation files in tracked package directories can still affect
    # Go compilation. Refuse them, including cgo/assembly inputs.
    package_dirs = {str(Path(name).parent) for name in paths if name.endswith('.go')}
    extensions = {'.go', '.s', '.S', '.c', '.h', '.cc', '.cpp', '.cxx', '.m', '.mm', '.f', '.F', '.for', '.f90', '.syso'}
    ignored = subprocess.check_output(['git', 'ls-files', '--others', '--ignored', '--exclude-standard', '-z'], cwd=source).decode().split('\0')
    for name in ignored:
        if name and Path(name).suffix in extensions:
            raise RuntimeError('unadmitted ignored compilation input: ' + name)
    for directory in package_dirs:
        for member in (source / directory).iterdir():
            if member.suffix in extensions and member.relative_to(source).as_posix() not in tracked:
                raise RuntimeError('unadmitted compilation input: ' + str(member))


def inventory(source):
    paths = subprocess.check_output(['git', 'ls-files', '-z'], cwd=source).decode().split('\0')
    admit_inputs(source, paths)
    production, corpus = {}, {}
    for name in sorted(set(paths)):
        if not name or name.startswith('.planning/'):
            continue
        path = source / name
        # An unstaged tracked deletion is part of the patch and must alter identity.
        data = ('symlink:' + os.readlink(path)).encode() if path.is_symlink() else path.read_bytes() if path.exists() else b'absent:tracked-deletion'
        (corpus if name.endswith('_test.go') else production)[name] = digest(data)
    def aggregate(files):
        return digest(''.join(f'{key}\0{files[key]}\n' for key in sorted(files)).encode())
    return {'production': production, 'test_corpus': corpus,
            'production_digest': aggregate(production), 'test_corpus_digest': aggregate(corpus),
            'harness_sha256': digest((source / HARNESS).read_bytes()),
            'contract_sha256': digest((source / CONTRACT).read_bytes())}


def objects(source):
    text = (source / 'cmd/branch_disposition_test.go').read_text()
    block = text.split('var disposedBranches = map[string]string{', 1)[1].split('}', 1)[0]
    hashes = re.findall(r':\s*"([0-9a-f]{40})"', block)
    if len(hashes) != 3:
        raise RuntimeError('disposedBranches contract changed; inspect before preparing')
    return hashes


def selected(source):
    text = (source / CONTRACT).read_text()
    return json.loads(re.search(r'const nativeEvidenceFocusedSelection = ("[^\n]+")', text)[1])


def argv_for(name, source):
    focused = name.startswith('focused_')
    argv = ['go', 'test', './cmd' if focused else './...']
    if name.endswith('race'):
        argv.append('-race')
    if focused:
        argv += ['-run', selected(source)]
    return argv + ['-count=1', '-timeout', '90m', '-json']


def isolated_env(root, toolchain, modules):
    # Construct a child map; never assign HOME/CODEX_HOME in this process.
    env = {key: os.environ[key] for key in ['PATH','LANG','LC_ALL','LC_CTYPE','SystemRoot','SYSTEMROOT','COMSPEC'] if key in os.environ}
    for key, leaf in [('HOME', 'home'), ('USERPROFILE', 'home'), ('CODEX_HOME', 'codex'),
                      ('CLAUDE_CONFIG_DIR', 'claude'), ('XDG_CONFIG_HOME', 'config'),
                      ('TMPDIR', 'tmp'), ('TMP', 'tmp'), ('TEMP', 'tmp'),
                      ('GOCACHE', 'go-cache'), ('GOPATH', 'gopath'), ('npm_config_cache', 'npm-cache')]:
        path = root / leaf
        path.mkdir(parents=True, exist_ok=True)
        env[key] = str(path)
    env.update(GOMODCACHE=str(modules), GOTOOLCHAIN='local', GOFLAGS='', GOPROXY='off',
               GOSUMDB='off', GIT_CONFIG_GLOBAL=os.devnull, GIT_CONFIG_NOSYSTEM='1',
               npm_config_userconfig=str(root/'absent.npmrc'),
               npm_config_globalconfig=str(root/'absent-global.npmrc'), npm_config_offline='true',
               PATH=str(Path(toolchain)/'bin')+os.pathsep+env.get('PATH', ''))
    return env


def probe(root, clone, env, receipt=None):
    root.mkdir(parents=True, exist_ok=False)
    write_new(root/'argv.json', {name:argv_for(name,clone) for name in ['focused_normal','focused_race','normal','race']})
    helper = root / 'probe_test.go'
    helper.write_text(PROBE)
    overlay = root / 'overlay.json'
    write_new(overlay, {'Replace': {str(clone/'cmd/zz_plan13_evidence_probe_test.go'): str(helper)}})
    child = dict(env, AETHER_GAP_PROBE_OUT=str(root))
    if receipt:
        child['AETHER_GAP_PROBE_RECEIPT'] = str(receipt)
    return record(root, 'probe', ['go', 'test', '-overlay', str(overlay), './cmd',
                  '-run', '^TestPlan13EvidenceProbe$', '-count=1', '-timeout', '180s', '-json'],
                  clone, child, 300)


def tree_bytes(path, exclude_toolchains=False):
    total = 0
    def fail(exc):
        raise exc
    for directory, dirs, files in os.walk(path, onerror=fail):
        if exclude_toolchains:
            dirs[:] = [d for d in dirs if not d.startswith('toolchain@') and not (d == 'toolchain' and 'cache/download/golang.org' in directory)]
        for name in files:
            item = Path(directory) / name
            total += item.lstat().st_size
    if not Path(path).exists():
        raise RuntimeError('storage estimate input unavailable: ' + str(path))
    return total


def storage_estimate(source, cache_source):
    receipt = json.loads((source / '.planning/phases/204.2-codex-native-worker-lifecycle/evidence/gap-closure/native-regression.json').read_text())
    ref = receipt['preflight_manifest']
    historical = Path(ref['path'])
    if file_ref(historical) != ref:
        raise RuntimeError('storage baseline identity mismatch')
    old = json.loads(historical.read_text())
    sizes = {'prior_exhausted_free_bytes': old['disk_free_before_copy']}
    missing = []
    measurements = {}
    try:
        # The old cache was deliberately retired. Its immutable retirement
        # receipt preserves a measured working-set floor without requiring it
        # to exist again or charging the shared multi-revision cache as one run.
        retirement_ref = receipt['cache_retirement']
        retirement_path = Path(retirement_ref['path'])
        if file_ref(retirement_path) != retirement_ref:
            raise RuntimeError('cache retirement identity mismatch')
        retirement = json.loads(retirement_path.read_text())
        if retirement['manifest'] != ref or retirement['bytes'] <= 0:
            raise RuntimeError('cache retirement baseline mismatch')
        sizes['prior_compiler_cache'] = retirement['bytes']
        env = json.loads(output(['go', 'env', '-json'], source))
        common = Path(output(['git', 'rev-parse', '--git-common-dir'], source))
        common = common if common.is_absolute() else source / common
        measurements = {'module_copy': str(cache_source), 'toolchain': env['GOROOT'],
                        'git_objects': str(common / 'objects')}
        for key, value in measurements.items():
            path = Path(value)
            sizes[key] = tree_bytes(path, key == 'module_copy')
        paths = subprocess.check_output(['git', 'ls-files', '-z'], cwd=source).decode().split('\0')
        sizes['checkout'] = sum((source / name).lstat().st_size for name in set(paths)
                                if name and ((source / name).exists() or (source / name).is_symlink()))
        candidate = old['lanes']['normal']['candidate']
        if file_ref(Path(candidate['path'])) != candidate:
            raise RuntimeError('historical candidate identity mismatch')
        sizes['candidate_binary'] = Path(candidate['path']).stat().st_size
    except (OSError, RuntimeError, KeyError, ValueError) as exc:
        missing.append(str(exc))
    if missing:
        return {'required_bytes': None, 'required_lower_bound_bytes': sizes['prior_exhausted_free_bytes'],
                'measured_bytes': sizes, 'historical_manifest': ref, 'uncertainties': missing,
                'method': 'Incomplete measurements prohibit admission; prior exhausted free bytes are only a lower bound.'}
    working = max(sizes['prior_compiler_cache'], sizes['checkout'] + sizes['module_copy'] + sizes['toolchain'])
    sizes['standalone_clone'] = sizes['checkout'] + sizes['git_objects']
    # Two caches, race expansion and link scratch, plus one additional working
    # set of headroom beyond the greater of our model and prior exhaustion.
    measured = sizes['module_copy'] + 2*sizes['standalone_clone'] + 4*working + 2*sizes['candidate_binary']
    required = max(measured, sizes['prior_exhausted_free_bytes']) + working
    return {'required_bytes': required, 'measured_bytes': sizes, 'measurement_paths': measurements,
            'compiler_working_set_bytes': working, 'historical_manifest': ref,
            'cache_retirement': retirement_ref,
            'method': 'working=max(retained measured compiler cache, current checkout+modules+toolchain); max(module+two standalone clones+four working sets+two binaries, exhausted prior free bytes)+one working set headroom; conservative estimate, not a guarantee or universal threshold'}


def storage_admit(manifest, root, stage):
    item = {'stage': stage, 'free_bytes': shutil.disk_usage(root).free,
            'required_bytes': manifest['storage_estimate']['required_bytes']}
    manifest.setdefault('storage_admissions', []).append(item)
    item['passed'] = item['required_bytes'] is not None and item['free_bytes'] >= item['required_bytes']
    if item['required_bytes'] is None:
        raise RuntimeError('uncertain storage estimate before ' + stage + ': ' + str(manifest['storage_estimate']))
    if not item['passed']:
        raise RuntimeError('insufficient storage before ' + stage + ': ' + str(item))


def prepare(args, source, root):
    if root.exists():
        raise RuntimeError('prepare requires a new evidence root; previous evidence is immutable')
    root.mkdir(parents=True)
    manifest = {'schema_version': 'aether-native-regression-preflight/v1', 'passed': False,
                'source': str(source), 'source_revision': output(['git','rev-parse','HEAD'],source),
                'source_status': output(['git','status','--porcelain=v1','--untracked-files=all'],source),
                'source_refs_before': output(['git','show-ref'],source),
                'lanes': {},
                'native_qualification': 'not attempted; regression preparation is not native proof',
                'prior_npm_incident': 'Partial recovery only: 31 additions, 64 changes, 17 removals remain unestablished; no real-home restoration performed.'}
    try:
        manifest['identity'] = inventory(source)
        manifest['disposed_commits'] = objects(source)
        patch = subprocess.check_output(['git','diff','--binary','HEAD'],cwd=source)
        (root/'candidate.patch').write_bytes(patch)
        manifest['patch'] = file_ref(root/'candidate.patch')
        toolchain = output(['go','env','GOROOT'],source)
        manifest['go_version'] = output(['go','version'],source)
        manifest['go_env'] = json.loads(output(['go','env','-json'],source))
        manifest['script'] = file_ref(source/'scripts/qualify-native-gap-regression.py')
        manifest['go_executable'] = file_ref(Path(toolchain)/'bin/go')
        manifest['tools'] = {}
        for tool in ['git','node','npm','codex','aether']:
            path = shutil.which(tool)
            item = {'path': path, 'role': 'read-only identity; installed aether is not this candidate'}
            if path:
                item.update(file_ref(Path(path).resolve()))
                # Aether version prints only; no lifecycle/install is called.
                item['version'] = output([path, 'version' if tool == 'aether' else '--version'],source)
            manifest['tools'][tool] = item
        manifest['host_execution'] = {'model': None, 'effort': None, 'tool_schema': None,
                                      'reason': 'No host launched by regression preparation; no invented host configuration.'}
        manifest['installed_assets'] = 'Not installed by this tool; test fixture installations are disposable and asserted by TestPackedNPMReleaseCandidateContract. Native host installed-byte proof belongs to the live qualification.'
        modules = root/'module-cache'
        # A disposable copy prevents even module-cache lock writes in real home.
        cache_source = Path(manifest['go_env']['GOMODCACHE'])
        manifest['storage_estimate'] = storage_estimate(source, cache_source)
        storage_admit(manifest, root, 'module-copy')
        def exclude_unused_toolchain(directory, names):
            # The pinned Go executable above is read-only and GOTOOLCHAIN=local;
            # copying other downloaded toolchains wastes gigabytes, not proof.
            relative = Path(directory).relative_to(cache_source).as_posix()
            if relative == 'golang.org':
                return [name for name in names if name.startswith('toolchain@')]
            if relative == 'cache/download/golang.org':
                return [name for name in names if name == 'toolchain']
            return []
        manifest['disk_free_before_copy'] = shutil.disk_usage(root).free
        shutil.copytree(cache_source, modules, symlinks=True, ignore=exclude_unused_toolchain)
        for lane in ['normal','race']:
            storage_admit(manifest, root, lane + '-clone')
            lane_root = root/lane
            lane_root.mkdir()
            clone = lane_root/'repo'
            env = isolated_env(lane_root,toolchain,modules)
            entry = {'root': str(lane_root), 'clone': str(clone), 'env': env, 'passed': False}
            manifest['lanes'][lane] = entry
            if record(lane_root,'clone',['git','clone','--no-local','--no-hardlinks','--no-checkout',str(source),str(clone)],source,env,180)['exit_code']:
                raise RuntimeError('standalone clone failed')
            if record(lane_root,'checkout',['git','checkout','--detach',manifest['source_revision']],clone,env,90)['exit_code']:
                raise RuntimeError('candidate checkout failed')
            if patch:
                if record(lane_root,'patch',['git','apply','--index','--binary',str(root/'candidate.patch')],clone,env,90)['exit_code']:
                    raise RuntimeError('candidate patch failed')
            for commit in manifest['disposed_commits']:
                if subprocess.run(['git','cat-file','-e',commit+'^{commit}'],cwd=clone,env=env,stderr=subprocess.DEVNULL).returncode:
                    # No refspec destination, no FETCH_HEAD, never a source ref write.
                    if record(lane_root,'import-'+commit,['git','fetch','--no-tags','--no-write-fetch-head',str(source),commit],clone,env,90)['exit_code']:
                        raise RuntimeError('missing disposed commit '+commit)
                if record(lane_root,'object-'+commit,['git','cat-file','-e',commit+'^{commit}'],clone,env,90)['exit_code']:
                    raise RuntimeError('disposed commit verification failed '+commit)
            if (clone/'.git/objects/info/alternates').exists() or not (clone/'.git').is_dir():
                raise RuntimeError('clone is not standalone')
            if inventory(clone) != manifest['identity']:
                raise RuntimeError('clone candidate/corpus mismatch')
            entry['git_dir'] = output(['git','rev-parse','--absolute-git-dir'],clone,env)
            version = json.loads((clone/'.aether/version.json').read_text())['version']
            binary = lane_root/'candidate-aether'
            storage_admit(manifest, root, lane + '-build-link')
            entry['build'] = record(lane_root,'build',['go','build','-buildvcs=false','-ldflags',
                '-X github.com/calcosmic/Aether/cmd.Version='+version,'-o',str(binary),'./cmd/aether'],clone,env,300)
            if entry['build']['exit_code']:
                raise RuntimeError('candidate build failed')
            entry['candidate'] = file_ref(binary)
            argv = ['go','test','./cmd'] + (['-race'] if lane == 'race' else []) + ['-run',PREFLIGHT,'-count=1','-timeout','180s','-json']
            storage_admit(manifest, root, lane + '-fixture-link')
            entry['fixture'] = record(lane_root,'fixture',argv,clone,env,420)
            events = [json.loads(line) for line in (lane_root/'fixture.stdout').read_text().splitlines() if line.strip()]
            terminals = {e.get('Test'):e['Action'] for e in events if e.get('Test') and e['Action'] in ['pass','fail','skip']}
            required = PREFLIGHT[2:-2].split('|')
            entry['fixture_terminals'] = terminals
            entry['passed'] = entry['fixture']['exit_code'] == 0 and all(terminals.get(t) == 'pass' for t in required)
            if entry['passed']:
                entry['discovery_probe'] = probe(lane_root/'discovery',clone,env)
                entry['passed'] = entry['discovery_probe']['exit_code'] == 0
            if inventory(clone) != manifest['identity']:
                raise RuntimeError('fixture changed candidate/corpus')
        manifest['passed'] = all(lane['passed'] for lane in manifest['lanes'].values())
    except Exception as exc:
        manifest['passed'] = False
        manifest['error'] = str(exc)
    finally:
        try:
            manifest['source_refs_after'] = output(['git','show-ref'],source)
            manifest['source_revision_after'] = output(['git','rev-parse','HEAD'],source)
            if manifest['source_revision_after'] != manifest['source_revision'] or inventory(source) != manifest.get('identity'):
                raise RuntimeError('source changed during preparation')
        except Exception as exc:
            manifest['passed'] = False
            if 'error' in manifest:
                manifest.setdefault('secondary_errors', []).append(str(exc))
            else:
                manifest['error'] = str(exc)
        write_new(root/'manifest.json',manifest)
        write_new(root/'manifest.sha256.json',file_ref(root/'manifest.json'))
    return 0 if manifest['passed'] else 1


def validate(source,root):
    manifest = json.loads((root/'manifest.json').read_text())
    if file_ref(root/'manifest.json') != json.loads((root/'manifest.sha256.json').read_text()):
        raise RuntimeError('immutable manifest changed')
    if not manifest['passed']:
        raise RuntimeError('failed preflight prohibits broad runs')
    if digest(Path(__file__).read_bytes()) != manifest['script']['sha256']:
        raise RuntimeError('regression harness changed')
    if inventory(source) != manifest['identity']:
        raise RuntimeError('current candidate/corpus mismatch; prepare a fresh root')
    if file_ref(Path(manifest['go_executable']['path'])) != manifest['go_executable']:
        raise RuntimeError('Go executable changed')
    for tool in manifest['tools'].values():
        if tool.get('sha256') and file_ref(Path(tool['path']))['sha256'] != tool['sha256']:
            raise RuntimeError('pinned tool changed: '+tool['path'])
    for lane in manifest['lanes'].values():
        for result in [lane['fixture'],lane['build'],lane['discovery_probe']]:
            for key in ['stdout','stderr']:
                if file_ref(Path(result[key]['path'])) != result[key]:
                    raise RuntimeError('preflight raw evidence changed')
        clone = Path(lane['clone'])
        if inventory(clone) != manifest['identity']:
            raise RuntimeError('prepared clone candidate/corpus mismatch')
        if file_ref(Path(lane['candidate']['path'])) != lane['candidate']:
            raise RuntimeError('candidate binary changed')
        for commit in manifest['disposed_commits']:
            subprocess.run(['git','cat-file','-e',commit+'^{commit}'],cwd=clone,env=lane['env'],check=True)
    return manifest


def run(args,source,root):
    manifest = validate(source,root)
    runs = {}
    for name in ['focused_normal','focused_race','normal','race']:
        manifest = validate(source,root)
        lane = manifest['lanes']['race' if name.endswith('race') else 'normal']
        clone, lane_root = Path(lane['clone']),Path(lane['root'])
        result = record(root,name,argv_for(name,clone),clone,lane['env'],5500)
        runs[name] = {'argv':result['argv'],'exit_code':result['exit_code'],
                      'raw_json':result['stdout'], 'stderr':result['stderr'],
                      'discovery':file_ref(lane_root/'discovery'/('discovery-'+name+'.json')),
                      'inherited_failures': {}}
        write_new(root/(name+'-run.json'),runs[name])
        validate(source,root)
    receipt = {'schema_version':'aether-native-regression/v1', 'runs':runs,
               **{k:v for k,v in manifest['identity'].items() if k.endswith('digest') or k == 'harness_sha256'},
               'qualification_sha256': None, 'status':'captured; requires diagnostic comparison and actual qualification'}
    write_new(root/'regression.json',receipt)
    return 0 if all(r['exit_code']==0 for r in runs.values()) else 1


def check(args,source,root):
    manifest = validate(source,root)
    receipt = Path(args.receipt).resolve() if args.receipt else root/'regression.json'
    data = json.loads(receipt.read_text())
    for key in ['production_digest','test_corpus_digest','harness_sha256']:
        if data.get(key) != manifest['identity'][key]:
            raise RuntimeError('regression identity mismatch: '+key)
    # Reviewed historical comparisons may be supplied, but exact run/discovery
    # bytes and invocations must still equal the immutable original captures.
    for name, item in data['runs'].items():
        original = json.loads((root/(name+'-run.json')).read_text())
        for key in ['argv','exit_code','raw_json','discovery','stderr']:
            if item.get(key) != original[key]:
                raise RuntimeError('run binding changed: '+name+'/'+key)
        stderr = Path(item['stderr']['path'])
        if file_ref(stderr) != item['stderr'] or stderr.read_bytes().strip():
            raise RuntimeError('stderr requires explicit diagnostic review: '+name)
    lane = manifest['lanes']['normal']
    result = probe(root/('check-'+str(time.time_ns())),Path(lane['clone']),lane['env'],receipt)
    return 0 if result['exit_code'] == 0 else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    modes = parser.add_mutually_exclusive_group(required=True)
    for mode in ['prepare','run','check']:
        modes.add_argument('--'+mode, action='store_true')
    parser.add_argument('--evidence-root', required=True)
    parser.add_argument('--receipt', help='reviewed regression receipt for --check; original captures remain binding')
    args = parser.parse_args()
    source = Path(__file__).resolve().parents[1]
    root = Path(args.evidence_root).expanduser().resolve()
    if not Path(args.evidence_root).is_absolute() or root == source or source in root.parents:
        parser.error('evidence root must be an absolute external disposable directory')
    try:
        return (prepare if args.prepare else run if args.run else check)(args,source,root)
    except Exception as exc:
        print('INCOMPLETE:',exc,file=sys.stderr)
        return 1

if __name__ == '__main__':
    sys.exit(main())
