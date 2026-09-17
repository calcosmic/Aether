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


def inventory(source):
    paths = subprocess.check_output(['git', 'ls-files', '-z'], cwd=source).decode().split('\0')
    production, corpus = {}, {}
    for name in sorted(set(paths)):
        if not name or name.startswith('.planning/'):
            continue
        path = source / name
        data = ('symlink:' + os.readlink(path)).encode() if path.is_symlink() else path.read_bytes()
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


def prepare(args, source, root):
    if root.exists():
        raise RuntimeError('prepare requires a new evidence root; previous evidence is immutable')
    root.mkdir(parents=True)
    manifest = {'schema_version': 'aether-native-regression-preflight/v1', 'passed': False,
                'source': str(source), 'source_revision': output(['git','rev-parse','HEAD'],source),
                'source_status': output(['git','status','--porcelain=v1','--untracked-files=all'],source),
                'source_refs_before': output(['git','show-ref'],source),
                'identity': inventory(source), 'disposed_commits': objects(source), 'lanes': {},
                'native_qualification': 'not attempted; regression preparation is not native proof',
                'prior_npm_incident': 'Partial recovery only: 31 additions, 64 changes, 17 removals remain unestablished; no real-home restoration performed.'}
    try:
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
            entry['build'] = record(lane_root,'build',['go','build','-buildvcs=false','-ldflags',
                '-X github.com/calcosmic/Aether/cmd.Version='+version,'-o',str(binary),'./cmd/aether'],clone,env,300)
            if entry['build']['exit_code']:
                raise RuntimeError('candidate build failed')
            entry['candidate'] = file_ref(binary)
            argv = ['go','test','./cmd'] + (['-race'] if lane == 'race' else []) + ['-run',PREFLIGHT,'-count=1','-timeout','180s','-json']
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
        manifest['error'] = str(exc)
    finally:
        manifest['source_refs_after'] = output(['git','show-ref'],source)
        manifest['source_revision_after'] = output(['git','rev-parse','HEAD'],source)
        if manifest['source_revision_after'] != manifest['source_revision'] or inventory(source) != manifest['identity']:
            manifest['passed'] = False
            manifest['error'] = 'source changed during preparation'
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
