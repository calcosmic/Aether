package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This separate schema never widens the Phase 204.1 read-only discovery proof.
// It starts incomplete and is promoted only after actual host/child evidence.
type codexNativeLiveReceipt struct {
	SchemaVersion              string                `json:"schema_version"`
	Scenario                   string                `json:"scenario"`
	Outcome                    string                `json:"outcome"`
	Reason                     string                `json:"reason,omitempty"`
	SourceRevision             string                `json:"source_revision"`
	SourceStatus               string                `json:"source_status"`
	SourceDigest               string                `json:"source_digest"`
	CandidatePath              string                `json:"candidate_path"`
	CandidateVersion           string                `json:"candidate_version"`
	CandidateSHA256            string                `json:"candidate_sha256"`
	BuildArgv                  []string              `json:"build_argv"`
	ClientPath                 string                `json:"client_path"`
	ClientVersion              string                `json:"client_version"`
	ClientSHA256               string                `json:"client_sha256"`
	Model                      string                `json:"model"`
	Args                       []string              `json:"client_args"`
	FixtureRoot                string                `json:"fixture_root"`
	FixtureProvenance          string                `json:"fixture_provenance"`
	SkillPath                  string                `json:"skill_path"`
	SupportPath                string                `json:"support_path"`
	RawEvents                  string                `json:"raw_events"`
	RawStderr                  string                `json:"raw_stderr"`
	SessionID                  string                `json:"host_session_id,omitempty"`
	ChildID                    string                `json:"child_id,omitempty"`
	ChildEvents                string                `json:"child_events,omitempty"`
	AttemptPath                string                `json:"attempt_path,omitempty"`
	AttemptID                  string                `json:"attempt_id,omitempty"`
	RunID                      string                `json:"run_id,omitempty"`
	LaunchID                   string                `json:"launch_id,omitempty"`
	WorkerName                 string                `json:"worker_name,omitempty"`
	TaskID                     string                `json:"task_id,omitempty"`
	PromptSHA256               string                `json:"prompt_sha256,omitempty"`
	ResultSHA256               string                `json:"result_sha256,omitempty"`
	CompletionPath             string                `json:"completion_path,omitempty"`
	Artifacts                  map[string]string     `json:"artifacts"`
	ObservedTools              []string              `json:"observed_tools,omitempty"`
	ExitStatus                 int                   `json:"exit_status"`
	ElapsedSeconds             float64               `json:"elapsed_seconds"`
	AuthRemoved                bool                  `json:"temporary_auth_removed"`
	ChecksPassed               bool                  `json:"child_checks_passed"`
	ChildEditObserved          bool                  `json:"child_edit_observed"`
	CreditObserved             bool                  `json:"runtime_credit_observed"`
	ParentSubstitution         bool                  `json:"parent_substitution"`
	ParentUnclassified         []string              `json:"parent_unclassified_commands,omitempty"`
	SkillRead                  bool                  `json:"installed_skill_read"`
	SupportRead                bool                  `json:"installed_support_read"`
	GuideRead                  bool                  `json:"runtime_guide_read"`
	NativeSpawnCount           int                   `json:"native_spawn_count"`
	TerminalCorroborated       bool                  `json:"child_terminal_corroborated"`
	SourceEventCorroborated    bool                  `json:"source_event_corroborated"`
	BoundHostSessionID         string                `json:"bound_host_session_id,omitempty"`
	SavedTerminal              *internalWorkerResult `json:"saved_terminal,omitempty"`
	SavedSourceEventSHA256     string                `json:"saved_source_event_sha256,omitempty"`
	SavedSourceEventID         string                `json:"saved_source_event_id,omitempty"`
	BaselineSource             string                `json:"baseline_source"`
	FinalSource                string                `json:"final_source"`
	CoordinatorPath            string                `json:"coordinator_path,omitempty"`
	CoordinatorSHA256          string                `json:"coordinator_sha256,omitempty"`
	ResumeSessionID            string                `json:"resume_host_session_id,omitempty"`
	ResumeRawEvents            string                `json:"resume_raw_events,omitempty"`
	ResumeRawStderr            string                `json:"resume_raw_stderr,omitempty"`
	ResumeExitStatus           int                   `json:"resume_exit_status"`
	BeforeResumeJournalSHA256  string                `json:"before_resume_journal_sha256,omitempty"`
	BeforeResumeJournalPath    string                `json:"before_resume_journal_path,omitempty"`
	ResumeWorkerStable         bool                  `json:"resume_worker_stable"`
	ResumeNoSpawn              bool                  `json:"resume_no_spawn"`
	ResumeInspectObserved      bool                  `json:"resume_inspect_observed"`
	FinalizationReplayStable   bool                  `json:"finalization_replay_stable"`
	EmptyResultRefused         bool                  `json:"empty_result_refused"`
	BaselineTestsSHA256        string                `json:"baseline_tests_sha256,omitempty"`
	BaselineModuleSHA256       string                `json:"baseline_module_sha256,omitempty"`
	LaunchMessageEncoding      string                `json:"launch_message_encoding,omitempty"`
	PromptDeliveryVerification string                `json:"prompt_delivery_verification,omitempty"`
	ValidationRevision         string                `json:"validation_revision,omitempty"`
	ValidationOriginalReceipt  string                `json:"validation_original_receipt,omitempty"`
	ValidationOriginalSHA256   string                `json:"validation_original_sha256,omitempty"`
}

func TestCodexNativeWorkerFreshHost(t *testing.T) {
	if os.Getenv("AETHER_CODEX_NATIVE_LIVE") != "1" {
		t.Skip("opt-in actual Codex host; no live proof claimed")
	}
	evidenceRoot := os.Getenv("AETHER_CODEX_NATIVE_EVIDENCE_DIR")
	if !filepath.IsAbs(evidenceRoot) {
		t.Fatal("AETHER_CODEX_NATIVE_EVIDENCE_DIR must be a durable absolute path")
	}
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	scenario := os.Getenv("AETHER_CODEX_NATIVE_SCENARIOS")
	runRoot, err := os.MkdirTemp(evidenceRoot, "native-"+scenario+"-")
	if err != nil {
		t.Fatal(err)
	}
	receipt := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v1", Scenario: scenario, Outcome: "incomplete", Reason: "harness did not reach all evidence gates", ExitStatus: -1, Artifacts: map[string]string{}, FixtureProvenance: "Fixture-prepared one-task accepted plan through specification, staged planning coordinator and exact acceptPlanCandidate; no live planning claim."}
	defer func() {
		// Index complete raw captures and installed inputs, but never credential caches.
		_ = filepath.WalkDir(runRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			base := entry.Name()
			if base == "auth.json" || base == "receipt.json" || strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
				return nil
			}
			if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".toml") || strings.HasSuffix(path, ".patch") || strings.HasSuffix(path, ".py") {
				raw, err := os.ReadFile(path)
				if err == nil {
					receipt.Artifacts[path] = lifecycleDigest(raw)
				}
			}
			return nil
		})
		liveSkillWriteJSON(t, filepath.Join(runRoot, "receipt.json"), receipt)
		t.Logf("native receipt: %s", filepath.Join(runRoot, "receipt.json"))
	}()
	fail := func(reason string) { receipt.Reason = reason; t.Fatal(reason) }
	if scenario != "ordinary" && scenario != "early-resume" {
		fail("choose ordinary or early-resume actual-host scenario")
	}
	source := antSkillSourceRoot(t)
	receipt.SourceRevision = strings.TrimSpace(liveSkillCommandOutput(t, source, "git", "rev-parse", "HEAD"))
	receipt.SourceStatus = strings.TrimSpace(liveSkillCommandOutput(t, source, "git", "status", "--short"))
	receipt.SourceDigest = liveSkillSourceIdentity(t, source, runRoot)
	client, err := exec.LookPath("codex")
	if err != nil {
		fail("actual Codex executable unavailable: " + err.Error())
	}
	client, err = filepath.EvalSymlinks(client)
	if err != nil {
		fail(err.Error())
	}
	receipt.ClientPath, receipt.ClientSHA256 = client, liveSkillFileDigest(t, client)
	receipt.ClientVersion = strings.TrimSpace(liveSkillCommandOutput(t, source, client, "--version"))
	receipt.CandidatePath = filepath.Join(runRoot, "bin", "aether")
	if err := os.MkdirAll(filepath.Dir(receipt.CandidatePath), 0700); err != nil {
		fail(err.Error())
	}
	receipt.BuildArgv = []string{"go", "build", "-ldflags", "-X github.com/calcosmic/Aether/cmd.Version=" + readRepoVersion(source), "-o", receipt.CandidatePath, "./cmd/aether"}
	build := exec.Command(receipt.BuildArgv[0], receipt.BuildArgv[1:]...)
	build.Dir = source
	raw, err := build.CombinedOutput()
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build.txt"), raw)
	if err != nil {
		fail("candidate build: " + err.Error())
	}
	receipt.CandidateSHA256 = liveSkillFileDigest(t, receipt.CandidatePath)
	receipt.CandidateVersion = strings.TrimSpace(liveSkillCommandOutput(t, source, receipt.CandidatePath, "version"))
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build-info.txt"), []byte(liveSkillCommandOutput(t, source, "go", "version", "-m", receipt.CandidatePath)))
	fixtureHome, repo := filepath.Join(runRoot, "home"), filepath.Join(runRoot, "repository")
	for _, dir := range []string{fixtureHome, repo} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			fail(err.Error())
		}
	}
	receipt.FixtureRoot = repo
	env := isolatedCodexSkillEnvironment(fixtureHome, filepath.Dir(receipt.CandidatePath))
	cache, err := os.MkdirTemp("", "aether-native-gocache-")
	if err != nil {
		fail(err.Error())
	}
	defer os.RemoveAll(cache)
	env = append(env, "GOCACHE="+cache)
	liveSkillRuntime(t, repo, env, filepath.Join(runRoot, "install.json"), receipt.CandidatePath, "install", "--package-dir", source, "--home-dir", fixtureHome, "--channel", "stable", "--skip-build-binary")
	receipt.SkillPath = filepath.Join(fixtureHome, ".codex", "skills", "aether", "ant-build", "SKILL.md")
	receipt.SupportPath = filepath.Join(fixtureHome, ".codex", "skills", "aether", "support", "aether-colony-build-cycle.md")
	nativePrepareLiveFixture(t, repo, runRoot)
	rawSource, _ := os.ReadFile(filepath.Join(repo, "clamp.go"))
	receipt.BaselineSource = string(rawSource)
	liveSkillWrite(t, filepath.Join(runRoot, "baseline-source.txt"), rawSource)
	coord, err := os.MkdirTemp("", "aether-worker-request-native-live-")
	if err != nil {
		fail(err.Error())
	}
	if err := os.MkdirAll(filepath.Join(runRoot, "coordination"), 0700); err != nil {
		fail(err.Error())
	}
	defer func() {
		_ = filepath.WalkDir(coord, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			data, err := os.ReadFile(path)
			if err == nil {
				liveSkillWrite(t, filepath.Join(runRoot, "coordination", entry.Name()), data)
			}
			return nil
		})
		_ = os.RemoveAll(coord)
	}()
	receipt.CoordinatorPath = filepath.Join(repo, ".aether", "native-fixture-coordinate.py")
	coordinator := strings.NewReplacer("__FIXTURE__", strconv.Quote(repo), "__COORD__", strconv.Quote(coord), "__EARLY__", map[bool]string{true: "True", false: "False"}[scenario == "early-resume"]).Replace(nativeFixtureCoordinator)
	liveSkillWrite(t, receipt.CoordinatorPath, []byte(coordinator))
	receipt.CoordinatorSHA256 = lifecycleDigest([]byte(coordinator))
	receipt.BaselineTestsSHA256 = liveSkillFileDigest(t, filepath.Join(repo, "clamp_test.go"))
	receipt.BaselineModuleSHA256 = liveSkillFileDigest(t, filepath.Join(repo, "go.mod"))
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-init.txt"), "git", "init", "--quiet")
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-add.txt"), "git", "add", "--", "clamp.go", "clamp_test.go", "go.mod", "AGENTS.md", ".gitignore")
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-commit.txt"), "git", "-c", "user.name=Native Fixture", "-c", "user.email=native-fixture@example.invalid", "commit", "-m", "fixture: failing clamp boundary baseline")
	// The baseline must fail before any real child exists.
	baseline := exec.Command("go", "test", "./...")
	baseline.Dir, baseline.Env = repo, env
	raw, err = baseline.CombinedOutput()
	liveSkillWrite(t, filepath.Join(runRoot, "baseline-check.txt"), raw)
	if err == nil {
		fail("baseline check unexpectedly passed; child work would be unprovable")
	}
	authRoot := os.Getenv("AETHER_CODEX_SKILL_AUTH_HOME")
	if authRoot == "" {
		authRoot = os.Getenv("CODEX_HOME")
	}
	if authRoot == "" {
		home, _ := os.UserHomeDir()
		authRoot = filepath.Join(home, ".codex")
	}
	var preferences struct {
		Model  string `toml:"model"`
		Effort string `toml:"model_reasoning_effort"`
	}
	if config, err := os.ReadFile(filepath.Join(authRoot, "config.toml")); err == nil {
		_, _ = toml.Decode(string(config), &preferences)
	}
	receipt.Model = preferences.Model
	auth, err := os.ReadFile(filepath.Join(authRoot, "auth.json"))
	if err != nil {
		fail("file authentication unavailable")
	}
	authPath := filepath.Join(fixtureHome, ".codex", "auth.json")
	if err := os.WriteFile(authPath, auth, 0600); err != nil {
		fail("temporary authentication copy failed")
	}
	defer func() {
		if err := os.Remove(authPath); err != nil && !os.IsNotExist(err) {
			receipt.Reason = "temporary authentication cleanup failed"
			receipt.Outcome = "incomplete"
			t.Error(receipt.Reason)
		} else {
			receipt.AuthRemoved = true
		}
	}()
	prompt := "$ant-build 1\nFixture authorization: execute the already accepted one-task plan in this disposable repository, using the installed skill and its private support. The named native Builder must edit clamp.go through apply_patch and run the full existing fixture tests with go test ./... -json -count=1 itself. Parent may coordinate only, with individual simple shell commands (no shell batching, redirection, Python snippets, or source edits). No commits, extra recruitment, publish, install, authentication reads, or other projects. All routine fixture choices are preauthorized.\n"
	prompt += fmt.Sprintf("The fixture supplies reviewed JSON request plumbing at %s; read it with cat. For each corresponding installed-skill operation call python3 %s <operation> using manifest, reserve, bind <actual-child-ID>, record, inspect, stage, or finalize. It executes one requested Go runtime operation and prints its real response; it never spawns workers, edits source, or normalizes a child result. Use the installed skill for sequencing and actual native spawn_agent/messages. Use this helper for request-file writes instead of authoring scripts. An empty-result invocation must be refused before the valid record. Preserve actual child event evidence and stop if a required host capability is missing.\n", receipt.CoordinatorPath, receipt.CoordinatorPath)
	if scenario == "early-resume" {
		prompt += "Interruption experiment: STOP this parent immediately after one accepted terminal record. Do not stage or finalize. A separate new parent session will finish accounting.\n"
	} else {
		prompt += "Stop after the existing build finalizer; do not run continue.\n"
	}
	liveSkillWrite(t, filepath.Join(runRoot, "prompt.txt"), []byte(prompt))
	args := []string{"exec", "--ignore-user-config", "--ignore-rules", "-s", "workspace-write", "--json", "--color", "never", "-C", repo, "-c", `cli_auth_credentials_store="file"`, "-c", `approval_policy="never"`, "--enable", "multi_agent", "-c", "shell_environment_policy.set.PATH=" + fmt.Sprintf("%q", filepath.Dir(receipt.CandidatePath)+":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin"), "-c", `shell_environment_policy.set.AETHER_OUTPUT_MODE="json"`}
	if preferences.Model != "" {
		args = append(args, "-m", preferences.Model)
	}
	args = append(args, "-c", "shell_environment_policy.set.GOCACHE="+strconv.Quote(cache))
	if preferences.Effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+fmt.Sprintf("%q", preferences.Effort))
	}
	args = append(args, "-")
	receipt.Args = args
	receipt.RawEvents, receipt.RawStderr = filepath.Join(runRoot, "parent-events.jsonl"), filepath.Join(runRoot, "parent-stderr.txt")
	out, err := os.Create(receipt.RawEvents)
	if err != nil {
		fail(err.Error())
	}
	defer out.Close()
	errOut, err := os.Create(receipt.RawStderr)
	if err != nil {
		fail(err.Error())
	}
	defer errOut.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	host := exec.CommandContext(ctx, client, args...)
	host.Dir, host.Env, host.Stdin, host.Stdout, host.Stderr = repo, env, strings.NewReader(prompt), out, errOut
	start := time.Now()
	err = host.Run()
	receipt.ElapsedSeconds = time.Since(start).Seconds()
	receipt.ExitStatus = 0
	if err != nil {
		receipt.ExitStatus = -1
		if exit, ok := err.(*exec.ExitError); ok {
			receipt.ExitStatus = exit.ExitCode()
		}
	}
	_ = out.Close()
	_ = errOut.Close()
	// Only read completed state after the actual host exits; never finish its work.
	nativeCollectLiveEvidence(t, &receipt, runRoot, fixtureHome)
	if scenario == "early-resume" {
		if receipt.ExitStatus != 0 || !receipt.TerminalCorroborated || !receipt.ChildEditObserved || !receipt.ChecksPassed || receipt.CompletionPath != "" || receipt.CreditObserved {
			fail("first parent did not stop at a proved terminal-before-stage boundary")
		}
		before, err := os.ReadFile(receipt.AttemptPath)
		if err != nil {
			fail(err.Error())
		}
		receipt.BeforeResumeJournalPath = filepath.Join(runRoot, "before-resume-attempt.json")
		liveSkillWrite(t, receipt.BeforeResumeJournalPath, before)
		liveSkillWrite(t, filepath.Join(runRoot, "before-resume-source.txt"), []byte(receipt.FinalSource))
		receipt.BeforeResumeJournalSHA256 = lifecycleDigest(before)
		liveSkillWrite(t, filepath.Join(coord, "resume-authorized"), []byte("First actual parent exited; harness verified terminal before staging.\n"))
		resumePrompt := fmt.Sprintf("Resume this disposable native build accounting experiment. A prior actual parent already completed and recorded its one Builder. Do not spawn, contact, rerun, or edit anything for that completed helper. Read installed support %s and the fixture request helper %s. Use individual simple shell commands, and the helper's inspect operation to read the exact retained attempt. Then perform the installed support's stage and finalize operations through the helper; replay finalize once to prove no repeated credit. Parent source edits, new helpers, arbitrary scripts, commits, installs, publish, and continue are forbidden. This is prepared-fixture bridge proof, not live planning.\n", receipt.SupportPath, receipt.CoordinatorPath)
		liveSkillWrite(t, filepath.Join(runRoot, "resume-prompt.txt"), []byte(resumePrompt))
		receipt.ResumeRawEvents, receipt.ResumeRawStderr = filepath.Join(runRoot, "resume-events.jsonl"), filepath.Join(runRoot, "resume-stderr.txt")
		receipt.ResumeExitStatus = nativeRunResumeHost(t, client, args, repo, env, resumePrompt, receipt.ResumeRawEvents, receipt.ResumeRawStderr)
		nativeCollectResumeEvidence(t, &receipt, runRoot, fixtureHome, coord)
	}
	if err := validateCodexNativeLiveReceipt(receipt); err != nil {
		fail(err.Error())
	}
	receipt.Outcome, receipt.Reason = "passed", ""
}

// Fixture-only request plumbing, supplied before either measured parent starts.
// Each invocation performs one explicit runtime operation; this code never
// launches a helper, changes the library, or transforms a child result.
const nativeFixtureCoordinator = `import hashlib, json, os, pathlib, subprocess, sys
repo = pathlib.Path(__FIXTURE__)
coord = pathlib.Path(__COORD__)
sessions = pathlib.Path(os.environ["CODEX_HOME"]) / "sessions"
op = sys.argv[1]
def write(name, value):
    (coord / name).write_text(json.dumps(value, indent=2))
def read(name):
    return json.loads((coord / name).read_text())
def runtime(args, label):
    proc = subprocess.run(["aether"] + args, cwd=repo, capture_output=True, text=True,
                          env={**os.environ, "AETHER_OUTPUT_MODE": "json"})
    (coord / (label + ".stdout.json")).write_text(proc.stdout)
    (coord / (label + ".stderr.txt")).write_text(proc.stderr)
    if proc.returncode:
        print(proc.stdout); print(proc.stderr, file=sys.stderr)
        sys.exit(proc.returncode)
    envelope = json.loads(proc.stdout)
    assert envelope["ok"], envelope
    return envelope["result"]
def request(operation, value):
    path = coord / (operation + "-request.json")
    write(path.name, value)
    return runtime(["codex-native-worker", operation, "--request", str(path)], operation)
def minimal():
    manifest = read("manifest.json")
    return {"schema_version": 1, "phase": manifest["phase"], "execution_binding": manifest["execution_binding"]}
def host_session():
    candidates = []
    for path in sessions.rglob("*.jsonl"):
        with path.open() as handle:
            meta = json.loads(handle.readline())["payload"]
        if meta.get("cwd") == str(repo) and not meta.get("parent_thread_id") and meta.get("thread_source") != "subagent":
            candidates.append((path.stat().st_mtime_ns, meta["id"]))
    assert candidates, "Actual parent session metadata unavailable"
    expected = os.environ.get("CODEX_THREAD_ID")
    if expected:
        assert any(item[1] == expected for item in candidates)
        return expected
    return max(candidates)[1]
def latest_native_terminal(session_root, child_id):
    paths = list(session_root.rglob("*" + child_id + ".jsonl"))
    assert len(paths) == 1, "Expected one bound child rollout"
    found = None
    for raw in paths[0].read_bytes().splitlines():
        event = json.loads(raw); payload = event.get("payload", {}); item = payload.get("item", {})
        if event.get("type") == "event_msg" and payload.get("type") == "item_completed" and payload.get("thread_id") == child_id and item.get("type") == "AgentMessage" and item.get("phase") == "final_answer":
            found = (raw, item)
    assert found, "No thread-attributed child terminal event yet"
    raw, item = found
    content = item["content"]
    text = content if isinstance(content, str) else "".join(part.get("text", "") for part in content)
    text = text.strip()
    if text.startswith(chr(96)*3 + "json"): text = text[7:].removesuffix(chr(96)*3).strip()
    return raw, item["id"], json.loads(text)
if op == "manifest":
    assert not (coord / "manifest.json").exists(), "Reuse saved manifest; do not redispatch"
    result = runtime(["build", "1", "--plan-only"], "manifest")
    write("manifest-envelope.json", {"ok": True, "result": result})
    write("manifest.json", result["dispatch_manifest"])
elif op == "reserve":
    manifest = read("manifest.json")
    assert len(manifest["dispatches"]) == 1
    dispatch = manifest["dispatches"][0]
    value = {**minimal(), "worker_name": dispatch["name"], "task_id": dispatch["task_id"],
             "host_session_id": host_session(), "workspace": str(repo), "host_permission": "workspace_write"}
    result = request("reserve", value)
    write("reservation.json", result)
elif op == "bind":
    reserved = read("reservation.json")
    worker = reserved["worker"]; native = worker["native"]
    value = {**read("reserve-request.json"), "launch_id": worker["provider_run_id"], "child_id": sys.argv[2],
             "dispatch_sha256": native["dispatch_sha256"], "prompt_sha256": native["prompt_sha256"]}
    result = request("bind", value)
elif op in ("record", "empty-result"):
    value = read("bind-request.json")
    if op == "empty-result":
        value["result"] = {}
    else:
        raw, event_id, result = latest_native_terminal(sessions, value["child_id"])
        (coord / "child-terminal.jsonl").write_bytes(raw + b"\n")
        value.update(result=result, source_event_id=event_id, source_event_sha256=hashlib.sha256(raw).hexdigest())
    result = request("record", value)
elif op in ("inspect", "stage"):
    if op == "stage" and __EARLY__ and not (coord / "resume-authorized").exists():
        sys.exit("Early-resume fixture: terminal boundary reached. Stop this parent before staging.")
    result = request(op, minimal())
elif op == "finalize":
    staged = read("stage.stdout.json")["result"]
    number = len(list(coord.glob("post-finalize-*-state.json"))) + 1
    result = runtime(["build-finalize", "1", "--completion-file", staged["completion_path"]], "finalize-" + str(number))
    (coord / ("post-finalize-" + str(number) + "-state.json")).write_bytes((repo / ".aether/data/COLONY_STATE.json").read_bytes())
else:
    sys.exit("Unknown coordinator operation")
print(json.dumps(result, indent=2))
`

func nativeFixtureCommand(t *testing.T, repo string, env []string, log, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir, command.Env = repo, env
	raw, err := command.CombinedOutput()
	liveSkillWrite(t, log, raw)
	if err != nil {
		t.Fatalf("fixture %s: %v (raw %s)", name, err, log)
	}
}

func TestCodexNativeCoordinatorTerminalSelection(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "coordinator.py")
	script := strings.NewReplacer("__FIXTURE__", strconv.Quote("/fixture"), "__COORD__", strconv.Quote("/tmp/fixture-coordination"), "__EARLY__", "True").Replace(nativeFixtureCoordinator)
	liveSkillWrite(t, scriptPath, []byte(script))
	// Execute only the real coordinator's pure selection function, without
	// running a host, runtime, or provider in this deterministic parser test.
	probe := `import ast,json,pathlib,sys
tree=ast.parse(pathlib.Path(sys.argv[1]).read_text())
function=next(node for node in tree.body if isinstance(node,ast.FunctionDef) and node.name=="latest_native_terminal")
exec(compile(ast.Module(body=[function],type_ignores=[]),sys.argv[1],"exec"))
raw,event_id,result=latest_native_terminal(pathlib.Path(sys.argv[2]),"child")
print(json.dumps({"event_id":event_id,"result":result}))
`
	for _, tc := range []struct {
		name, first, last string
		pass              bool
	}{
		{"waiting_then_valid", "Waiting for bound release.", `{"status":"completed"}`, true},
		{"valid_then_malformed", `{"status":"completed"}`, "Malformed latest result", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := t.TempDir()
			first := nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "AgentMessage", "id": "earlier", "phase": "final_answer", "content": []any{map[string]any{"text": tc.first}}})
			last := nativeEvidenceEvent(t, "item_completed", "child", map[string]any{"type": "AgentMessage", "id": "latest", "phase": "final_answer", "content": []any{map[string]any{"text": tc.last}}})
			liveSkillWrite(t, filepath.Join(events, "rollout-child.jsonl"), append(first, last...))
			out, err := exec.Command("python3", "-c", probe, scriptPath, events).CombinedOutput()
			if tc.pass && (err != nil || !strings.Contains(string(out), `"event_id": "latest"`)) {
				t.Fatalf("latest corrected terminal unavailable: %v %s", err, out)
			}
			if !tc.pass && err == nil {
				t.Fatalf("malformed latest reused older success: %s", out)
			}
		})
	}
}

func TestCodexNativeFixtureCoordinatorBoundary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coordinator.py")
	script := strings.NewReplacer("__FIXTURE__", strconv.Quote("/fixture"), "__COORD__", strconv.Quote("/tmp/fixture-coordination"), "__EARLY__", "True").Replace(nativeFixtureCoordinator)
	check := exec.Command("python3", "-c", "import ast,sys; ast.parse(sys.stdin.read())")
	check.Stdin = strings.NewReader(script)
	if out, err := check.CombinedOutput(); err != nil {
		t.Fatalf("fixture coordinator syntax: %v %s", err, out)
	}
	liveSkillWrite(t, path, []byte(script))
	r := codexNativeLiveReceipt{CoordinatorPath: path, CoordinatorSHA256: lifecycleDigest([]byte(script))}
	allowed := []string{"python3 " + path + " inspect", "python3 " + path + " bind child-123", "aether command-guide build --platform codex", "git diff --check", "cat /installed/SKILL.md", `jq '{workers: .workers | length}' /tmp/manifest.json`, `rg --files "$CODEX_HOME/sessions"`}
	for _, command := range allowed {
		if !nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}) {
			t.Fatalf("safe fixture coordination refused: %s", command)
		}
	}
	for _, command := range []string{"python3 " + path + " inspect extra", "python3 -c 'exec(open(\"" + path + "\").read())'", "git diff --output=/fixture/clamp.go", "cat x > clamp.go", "python3 " + path + " inspect; echo ok", `rg --files "$(touch clamp.go)"`, `jq '.' manifest.json > clamp.go`} {
		if nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}) {
			t.Fatalf("unknown parent operation accepted: %s", command)
		}
	}
	liveSkillWrite(t, path, []byte(script+"\nopen('/fixture/clamp.go','w').write('parent')\n"))
	if nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", allowed[0]}) {
		t.Fatal("modified coordinator accepted")
	}
}

func nativeRunResumeHost(t *testing.T, client string, args []string, repo string, env []string, prompt, events, stderrPath string) int {
	t.Helper()
	out, err := os.Create(events)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	stderrFile, err := os.Create(stderrPath)
	if err != nil {
		t.Fatal(err)
	}
	defer stderrFile.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	host := exec.CommandContext(ctx, client, args...)
	host.Dir, host.Env, host.Stdin, host.Stdout, host.Stderr = repo, env, strings.NewReader(prompt), out, stderrFile
	if err := host.Run(); err != nil {
		if exit, ok := err.(*exec.ExitError); ok {
			return exit.ExitCode()
		}
		return -1
	}
	return 0
}

func nativeCollectResumeEvidence(t *testing.T, r *codexNativeLiveReceipt, runRoot, fixtureHome, coord string) {
	t.Helper()
	r.ResumeWorkerStable, r.ResumeNoSpawn, r.ResumeInspectObserved, r.FinalizationReplayStable = false, false, false, false
	raw, err := os.ReadFile(r.ResumeRawEvents)
	if err != nil {
		return
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "thread.started" {
			r.ResumeSessionID = e.ThreadID
		}
	}
	if r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID {
		return
	}
	resumed := *r
	resumed.SessionID, resumed.NativeSpawnCount, resumed.ParentSubstitution = r.ResumeSessionID, 0, false
	found := false
	_ = filepath.WalkDir(filepath.Join(fixtureHome, ".codex", "sessions"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, r.ResumeSessionID+".jsonl") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var first nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &first) != nil || first.Type != "session_meta" || first.Payload.ID != r.ResumeSessionID {
			return nil
		}
		found = true
		nativeInspectParentEvents(&resumed, raw)
		return nil
	})
	r.ResumeNoSpawn = found && resumed.NativeSpawnCount == 0
	r.ParentSubstitution = r.ParentSubstitution || resumed.ParentSubstitution
	r.ResumeInspectObserved = resumed.ResumeInspectObserved
	beforeRaw, err := os.ReadFile(r.BeforeResumeJournalPath)
	if err != nil || lifecycleDigest(beforeRaw) != r.BeforeResumeJournalSHA256 {
		return
	}
	var before buildAttemptRecord
	if json.Unmarshal(beforeRaw, &before) != nil || before.CompletionPath != "" || len(before.WorkerRuns) != 1 {
		return
	}
	beforeWorker := before.WorkerRuns[0]
	if beforeWorker.Native == nil || beforeWorker.Native.LaunchState != "terminal" || beforeWorker.ResultSHA256 != r.ResultSHA256 {
		return
	}
	sourceBefore, err := os.ReadFile(filepath.Join(filepath.Dir(r.BeforeResumeJournalPath), "before-resume-source.txt"))
	if err != nil {
		return
	}
	nativeCollectLiveEvidence(t, r, runRoot, fixtureHome)
	afterRaw, err := os.ReadFile(r.AttemptPath)
	if err != nil {
		return
	}
	var after buildAttemptRecord
	if json.Unmarshal(afterRaw, &after) != nil {
		return
	}
	beforeWorkers, _ := json.Marshal(before.WorkerRuns)
	afterWorkers, _ := json.Marshal(after.WorkerRuns)
	r.ResumeWorkerStable = before.ID == after.ID && before.RunID == after.RunID && bytes.Equal(beforeWorkers, afterWorkers) && string(sourceBefore) == r.FinalSource
	first, err1 := os.ReadFile(filepath.Join(coord, "post-finalize-1-state.json"))
	second, err2 := os.ReadFile(filepath.Join(coord, "post-finalize-2-state.json"))
	var final1, final2 struct {
		OK     bool `json:"ok"`
		Result struct {
			Idempotent *bool `json:"idempotent"`
		} `json:"result"`
	}
	out1, err3 := os.ReadFile(filepath.Join(coord, "finalize-1.stdout.json"))
	out2, err4 := os.ReadFile(filepath.Join(coord, "finalize-2.stdout.json"))
	if err1 == nil && err2 == nil && err3 == nil && err4 == nil && json.Unmarshal(out1, &final1) == nil && json.Unmarshal(out2, &final2) == nil {
		r.FinalizationReplayStable = bytes.Equal(first, second) && final1.OK && final2.OK && final1.Result.Idempotent != nil && !*final1.Result.Idempotent && final2.Result.Idempotent != nil && *final2.Result.Idempotent
	}
}

func nativePrepareLiveFixture(t *testing.T, root, runRoot string) {
	t.Helper()
	goal := "Fix integer Clamp boundaries in the dependency-free tiny Go library; a native Builder edits clamp.go and runs go test ./..."
	data := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	liveSkillWriteJSON(t, filepath.Join(data, "COLONY_STATE.json"), colony.ColonyState{Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "light"})
	liveSkillWrite(t, filepath.Join(root, "go.mod"), []byte("module example.invalid/nativefixture\n\ngo 1.23\n"))
	liveSkillWrite(t, filepath.Join(root, "clamp.go"), []byte("package nativefixture\n\nfunc Clamp(value, low, high int) int { return value }\n"))
	liveSkillWrite(t, filepath.Join(root, "clamp_test.go"), []byte("package nativefixture\nimport \"testing\"\nfunc TestClamp(t *testing.T) { for _, c := range [][4]int{{-3,0,10,0},{15,0,10,10},{5,0,10,5},{0,0,10,0},{10,0,10,10}} { if got := Clamp(c[0],c[1],c[2]); got != c[3] { t.Errorf(\"Clamp(%v)=%d want %d\", c[:3],got,c[3]) } } }\n"))
	liveSkillWrite(t, filepath.Join(root, ".gitignore"), []byte(".aether/\n.codex/\n"))
	liveSkillWrite(t, filepath.Join(root, "AGENTS.md"), []byte("# Disposable native worker fixture\nOnly the runtime-assigned native Builder may edit clamp.go. Do not change clamp_test.go or go.mod. Parent coordinates only. No commits or external actions. Builder edits must use apply_patch so raw FileChange events preserve the complete diff. Run each check individually in this repository. Required test proof: go test ./... -json -count=1 (TestClamp must run, with no filters).\n## Verification Commands\n- build: go build ./...\n- tests: go test ./...\n- lint: go vet ./...\n"))
	draftReq := specificationTestDraftRequest(t, colony.SpecScopeWholeGoal)
	draftReq.Scope.GoalID, draftReq.Scope.SessionID = "goal-200", "session-200"
	for _, items := range [][]specificationItemInput{draftReq.Outcomes, draftReq.IncludedBehaviors, draftReq.Requirements} {
		for i := range items {
			items[i].Description = goal
		}
	}
	draftReq.Exclusions[0].Description = "Do not change tests, dependencies or files outside the disposable project."
	draftReq.AcceptanceChecks[0].Description = "Clamp returns the low/high bound outside the interval and preserves in-range values."
	draftReq.AcceptanceChecks[0].Verification = "go test ./..."
	draftReq.AffectedPublicPaths[0].Path = "Clamp"
	draft, err := createSpecificationDraft(root, draftReq, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := approveSpecification(root, specificationApprovalRequest{RevisionID: draft.Revision.ID, RevisionContentHash: draft.Revision.ContentHash, ApprovalToken: specificationApprovalToken(draft.Specification.ID, draft.Revision.ID, draft.Revision.ContentHash), ApprovedBy: "fixture:predeclared-authorization", ApprovedAt: draftReq.CreatedAt.Add(time.Minute)}, specificationMutationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	approvalHash, _ := jsonSHA256(*approved.Revision.Approval)
	state := mustReadSpecificationTestState(t, root)
	hash, _ := planStateHash(state.Plan)
	baseID, hash := planningBaseRevisionIdentity(state.Plan, hash)
	binding := planningStageSpecificationBinding{RevisionID: approved.Revision.ID, ContentHash: approved.Revision.ContentHash, Status: colony.SpecStatusApproved, ApprovalReceiptID: approved.Revision.Approval.ID, ApprovalReceiptHash: approvalHash}
	manifest, result := planningRouteStageTestRun(t, root, approved.Revision, binding, baseID, hash, "native-tracer-fixture")
	phase := &result.Proposal.Phases[0]
	phase.Name = "Clamp boundary fix"
	phase.Description = goal
	phase.Tasks[0].Goal = goal
	criterion := "go test ./... passes the existing boundary cases"
	phase.SuccessCriteria = []string{criterion}
	phase.EvidenceRequirements = []colony.CriterionEvidenceRequirement{{Criterion: criterion, Checks: []string{"tests"}}}
	phase.Tasks[0].SuccessCriteria = phase.SuccessCriteria
	phase.Tasks[0].EvidenceRequirements = phase.EvidenceRequirements
	result.Proposal.TaskDeclarations[0].Files = []string{"clamp.go"}
	planningRouteStageSetPolicy(t, root, manifest.RunID, 70, 6)
	coordinated, err := coordinatePlanningRouteStage(root, manifest, planningRouteStageTestBytes(t, result))
	if err != nil {
		t.Fatal(err)
	}
	if coordinated.Candidate == nil {
		t.Fatal("fixture did not produce a candidate")
	}
	candidate := *coordinated.Candidate
	accepted, err := acceptPlanCandidate(root, planCandidateTestAcceptanceRequest(candidate), planCandidateAcceptanceOptions{AcceptedBy: "fixture:predeclared-authorization", AcceptedAt: candidate.CreatedAt.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "prepared-plan-acceptance.json"), accepted)
	liveSkillWriteJSON(t, filepath.Join(runRoot, "prepared-plan-candidate.json"), candidate)
}

func nativeCollectLiveEvidence(t *testing.T, r *codexNativeLiveReceipt, runRoot, fixtureHome string) {
	t.Helper()
	if source, err := os.ReadFile(filepath.Join(r.FixtureRoot, "clamp.go")); err == nil {
		r.FinalSource = string(source)
	}
	events, err := os.ReadFile(r.RawEvents)
	if err != nil {
		r.Reason = err.Error()
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(events))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	skill, _ := os.ReadFile(r.SkillPath)
	support, _ := os.ReadFile(r.SupportPath)
	seen := map[string]bool{}
	for scanner.Scan() {
		var e map[string]any
		if json.Unmarshal(scanner.Bytes(), &e) != nil {
			continue
		}
		if e["type"] == "thread.started" {
			r.SessionID, _ = e["thread_id"].(string)
		}
		if item, ok := e["item"].(map[string]any); ok {
			if e["type"] == "item.completed" {
				kind, _ := item["type"].(string)
				tool, _ := item["tool"].(string)
				if kind == "collab_tool_call" || kind == "collab_agent_tool_call" {
					if tool == "spawn_agent" {
						r.NativeSpawnCount++
					}
				}
				command, _ := item["command"].(string)
				output, _ := item["aggregated_output"].(string)
				status, _ := item["status"].(string)
				exit, _ := item["exit_code"].(float64)
				if kind == "command_execution" && status == "completed" && exit == 0 {
					if nativeLiveReadCommand(command, r.SkillPath) && len(skill) > 0 && strings.Contains(output, strings.TrimSpace(string(skill))) {
						r.SkillRead = true
					}
					if nativeLiveReadCommand(command, r.SupportPath) && len(support) > 0 && strings.Contains(output, strings.TrimSpace(string(support))) {
						r.SupportRead = true
					}
					if strings.Contains(command, "command-guide build --platform codex") && strings.Contains(output, "codex-native-worker reserve") {
						r.GuideRead = true
					}
				}
			}
			kind, _ := item["type"].(string)
			if !seen[kind] {
				r.ObservedTools = append(r.ObservedTools, kind)
				seen[kind] = true
			}
		}
	}
	attemptFiles, _ := filepath.Glob(filepath.Join(r.FixtureRoot, ".aether", "data", "build", "phase-1", "attempts", "*.json"))
	for _, path := range attemptFiles {
		if strings.HasSuffix(path, ".completion.json") {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var attempt buildAttemptRecord
		if json.Unmarshal(raw, &attempt) != nil || len(attempt.WorkerRuns) != 1 || attempt.WorkerRuns[0].Native == nil {
			continue
		}
		worker := attempt.WorkerRuns[0]
		r.SavedTerminal, r.SavedSourceEventSHA256, r.BoundHostSessionID = worker.Result, worker.Native.SourceEventSHA256, worker.Native.HostSessionID
		r.SavedSourceEventID = worker.Native.SourceEventID
		r.AttemptPath, r.AttemptID, r.RunID = path, attempt.ID, attempt.RunID
		r.LaunchID, r.ChildID, r.WorkerName, r.TaskID = worker.ProviderRunID, worker.Native.ChildID, worker.WorkerName, worker.TaskID
		r.PromptSHA256, r.ResultSHA256, r.CompletionPath = worker.Native.PromptSHA256, worker.ResultSHA256, attempt.CompletionPath
		liveSkillWrite(t, filepath.Join(runRoot, "terminal-attempt.json"), raw)
	}
	// Codex's persisted session logs, not parent summaries, identify child tools.
	_ = filepath.WalkDir(filepath.Join(fixtureHome, ".codex", "sessions"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if r.ChildID != "" && strings.Contains(filepath.Base(path), r.ChildID) {
			r.ChildEvents = path
			nativeInspectChildEvents(r, raw)
		}
		if r.SessionID != "" && strings.Contains(filepath.Base(path), r.SessionID) {
			nativeInspectParentEvents(r, raw)
		}
		return nil
	})
	stateRaw, err := os.ReadFile(filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json"))
	if err == nil {
		var state colony.ColonyState
		if json.Unmarshal(stateRaw, &state) == nil && len(state.Plan.Phases) == 1 && len(state.Plan.Phases[0].Tasks) == 1 {
			r.CreditObserved = state.Plan.Phases[0].Tasks[0].Status == colony.TaskCompleted
		}
	}
	diff := exec.Command("git", "diff", "--", "clamp.go", "clamp_test.go", "go.mod")
	diff.Dir = r.FixtureRoot
	if raw, err := diff.Output(); err == nil {
		liveSkillWrite(t, filepath.Join(runRoot, "child-edit.patch"), raw)
	}
}

// Only thread-attributed host items qualify. A child's rollout also contains
// copied parent response_items; those are never child execution evidence.
type nativeHostEvent struct {
	Type    string `json:"type"`
	Payload struct {
		Type           string `json:"type"`
		ID             string `json:"id"`
		ParentThreadID string `json:"parent_thread_id"`
		AgentRole      string `json:"agent_role"`
		Cwd            string `json:"cwd"`
		ThreadID       string `json:"thread_id"`
		Item           struct {
			Type     string          `json:"type"`
			ID       string          `json:"id"`
			Phase    string          `json:"phase"`
			Status   string          `json:"status"`
			Command  []string        `json:"command"`
			Cwd      string          `json:"cwd"`
			ExitCode *int            `json:"exit_code"`
			Output   string          `json:"aggregated_output"`
			Content  json.RawMessage `json:"content"`
			Changes  map[string]struct {
				Type     string  `json:"type"`
				Diff     string  `json:"unified_diff"`
				MovePath *string `json:"move_path"`
			} `json:"changes"`
		} `json:"item"`
	} `json:"payload"`
}

func nativeInspectChildEvents(r *codexNativeLiveReceipt, raw []byte) {
	r.ChildEditObserved, r.ChecksPassed = false, false
	r.TerminalCorroborated, r.SourceEventCorroborated = false, false
	attributed := false
	metadataSeen := false
	source := r.BaselineSource
	edits, patchValid := 0, source != ""
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	for scanner.Scan() {
		line := scanner.Bytes()
		var e nativeHostEvent
		if json.Unmarshal(line, &e) != nil {
			continue
		}
		if e.Type == "session_meta" {
			if !metadataSeen {
				metadataSeen = true
				attributed = e.Payload.ID == r.ChildID && e.Payload.ParentThreadID == r.BoundHostSessionID &&
					e.Payload.AgentRole == "aether-builder" && nativeSameCwd(e.Payload.Cwd, r.FixtureRoot)
			}
			continue
		}
		p := e.Payload
		if !attributed || e.Type != "event_msg" || p.Type != "item_completed" || p.ThreadID != r.ChildID {
			continue
		}
		i := p.Item
		switch i.Type {
		case "FileChange":
			if i.Status != "completed" {
				continue
			}
			for path, change := range i.Changes {
				if filepath.Clean(path) != filepath.Join(r.FixtureRoot, "clamp.go") || change.Type != "update" || change.MovePath != nil {
					patchValid = false
					continue
				}
				edits++
				var err error
				source, err = nativeApplyUnifiedDiff(source, change.Diff)
				if err != nil {
					patchValid = false
				}
			}
		case "CommandExecution":
			if i.Status == "completed" && i.ExitCode != nil && *i.ExitCode == 0 &&
				nativeSameCwd(i.Cwd, r.FixtureRoot) && nativeRequiredFixtureTest(i.Command, i.Output) {
				r.ChecksPassed = true
			}
		case "AgentMessage":
			if i.Phase != "final_answer" || i.ID != r.SavedSourceEventID || r.SavedTerminal == nil {
				continue
			}
			hash := strings.TrimPrefix(r.SavedSourceEventSHA256, "sha256:")
			if hash != strings.TrimPrefix(lifecycleDigest(line), "sha256:") &&
				hash != strings.TrimPrefix(lifecycleDigest(append(append([]byte(nil), line...), '\n')), "sha256:") {
				continue
			}
			message := strings.TrimSpace(nativeToolOutputText(i.Content))
			message = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(message, "\x60\x60\x60json"), "\x60\x60\x60"))
			result, err := normalizeCodexNativeResult([]byte(message))
			if err != nil {
				continue
			}
			digest, _ := jsonSHA256(&result)
			if digest == r.ResultSHA256 {
				r.TerminalCorroborated, r.SourceEventCorroborated = true, true
			}
		}
	}
	r.ChildEditObserved = attributed && patchValid && edits > 0 && source == r.FinalSource && source != r.BaselineSource
}

func nativeSameCwd(got, want string) bool {
	if strings.HasPrefix(got, "file://") {
		u, err := url.Parse(got)
		if err != nil || u.Host != "" {
			return false
		}
		got = u.Path
	}
	return filepath.Clean(got) == filepath.Clean(want)
}

// Only the complete uncached fixture suite qualifies. Test event JSON proves
// TestClamp and its package passed; a shell exit or "ok" substring cannot.
func nativeRequiredFixtureTest(command []string, output string) bool {
	if len(command) != 3 || (command[1] != "-lc" && command[1] != "-c") {
		return false
	}
	words, ok := nativeSimpleShellWords(command[2])
	if !ok {
		return false
	}
	if len(words) > 0 && strings.HasPrefix(words[0], "GOCACHE=/") {
		words = words[1:]
	}
	if len(words) != 5 || words[0] != "go" || words[1] != "test" {
		return false
	}
	flags := map[string]bool{}
	for _, word := range words[2:] {
		if flags[word] {
			return false
		}
		flags[word] = true
	}
	if !flags["./..."] || !flags["-json"] || !flags["-count=1"] {
		return false
	}
	ran, passed, pkg := false, false, false
	for _, line := range strings.Split(output, "\n") {
		var event struct{ Action, Package, Test string }
		if json.Unmarshal([]byte(line), &event) != nil {
			continue
		}
		if event.Package != "example.invalid/nativefixture" {
			return false
		}
		if event.Action == "fail" || event.Action == "skip" {
			return false
		}
		if event.Test == "TestClamp" && event.Action == "run" {
			ran = true
		}
		if event.Test == "TestClamp" && event.Action == "pass" && ran {
			passed = true
		}
		if event.Test == "" && event.Action == "pass" && passed {
			pkg = true
		}
	}
	return ran && passed && pkg
}

// Unknown shell syntax is incomplete proof, never an assumption of safety.
func nativeSimpleShellWords(command string) ([]string, bool) {
	var words []string
	var word strings.Builder
	quote := rune(0)
	for _, c := range strings.TrimSpace(command) {
		if quote == '\'' {
			if c == '\'' {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		if quote == '"' {
			if c == '"' {
				quote = 0
				continue
			}
			if strings.ContainsRune("$\x60\\\n\r", c) {
				return nil, false
			}
			word.WriteRune(c)
			continue
		}
		if strings.ContainsRune("$\x60\\\n\r;|&<>", c) {
			return nil, false
		}
		if c == '\'' || c == '"' {
			if quote == 0 {
				quote = c
			} else if quote == c {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		if (c == ' ' || c == '\t') && quote == 0 {
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
		} else {
			word.WriteRune(c)
		}
	}
	if quote != 0 {
		return nil, false
	}
	if word.Len() > 0 {
		words = append(words, word.String())
	}
	return words, len(words) > 0
}

// Replay every child patch against the recorded baseline, checking old lines.
// The final file must be exactly reconstructible from these child-only edits.
func nativeApplyUnifiedDiff(source, patch string) (string, error) {
	old := strings.Split(strings.TrimSuffix(source, "\n"), "\n")
	lines := strings.Split(strings.TrimSuffix(patch, "\n"), "\n")
	header := regexp.MustCompile(`^@@ -([0-9]+)(?:,([0-9]+))? \+([0-9]+)(?:,([0-9]+))? @@`)
	var out []string
	pos, hunks := 0, 0
	for n := 0; n < len(lines); {
		m := header.FindStringSubmatch(lines[n])
		if m == nil {
			return "", fmt.Errorf("unsupported patch header")
		}
		offset, _ := strconv.Atoi(m[1])
		offset--
		if offset < pos || offset > len(old) {
			return "", fmt.Errorf("invalid patch offset")
		}
		out = append(out, old[pos:offset]...)
		pos = offset
		hunks++
		n++
		for n < len(lines) && !strings.HasPrefix(lines[n], "@@") {
			line := lines[n]
			n++
			if line == "" {
				return "", fmt.Errorf("empty patch line")
			}
			switch line[0] {
			case ' ', '-':
				if pos >= len(old) || old[pos] != line[1:] {
					return "", fmt.Errorf("child patch does not match baseline")
				}
				if line[0] == ' ' {
					out = append(out, line[1:])
				}
				pos++
			case '+':
				out = append(out, line[1:])
			default:
				return "", fmt.Errorf("unsupported patch operation")
			}
		}
	}
	if hunks == 0 {
		return "", fmt.Errorf("no patch hunks")
	}
	out = append(out, old[pos:]...)
	return strings.Join(out, "\n") + "\n", nil
}

func validateCodexNativeLiveReceipt(r codexNativeLiveReceipt) error {
	if r.ExitStatus != 0 {
		return fmt.Errorf("actual Codex parent exited %d; see %s", r.ExitStatus, r.RawStderr)
	}
	if r.SessionID == "" || r.ChildID == "" || r.ChildEvents == "" {
		return fmt.Errorf("actual native parent/child identity or raw child events unavailable; observed tool types: %v", r.ObservedTools)
	}
	if !r.SkillRead || !r.SupportRead || !r.GuideRead {
		return fmt.Errorf("installed skill/support/guide reads not corroborated (skill=%v support=%v guide=%v)", r.SkillRead, r.SupportRead, r.GuideRead)
	}
	if r.NativeSpawnCount != 1 || r.BoundHostSessionID != r.SessionID {
		return fmt.Errorf("native spawn/session binding incomplete (spawns=%d actual=%s bound=%s)", r.NativeSpawnCount, r.SessionID, r.BoundHostSessionID)
	}
	if !r.TerminalCorroborated || !r.SourceEventCorroborated {
		return fmt.Errorf("saved result/source event not corroborated by the raw child terminal")
	}
	for path, want := range map[string]string{r.CandidatePath: r.CandidateSHA256, r.ClientPath: r.ClientSHA256, r.CoordinatorPath: r.CoordinatorSHA256, filepath.Join(r.FixtureRoot, "clamp_test.go"): r.BaselineTestsSHA256, filepath.Join(r.FixtureRoot, "go.mod"): r.BaselineModuleSHA256} {
		raw, err := os.ReadFile(path)
		if err != nil || lifecycleDigest(raw) != want {
			return fmt.Errorf("candidate/client/baseline identity changed: %s", path)
		}
	}
	if !r.ChildEditObserved || !r.ChecksPassed {
		return fmt.Errorf("raw native child edit/check evidence incomplete (edit=%v checks=%v); child log %s", r.ChildEditObserved, r.ChecksPassed, r.ChildEvents)
	}
	if r.ParentSubstitution {
		return fmt.Errorf("parent writes or unclassified operations prevent child-only attribution: %v", r.ParentUnclassified)
	}
	if r.AttemptID == "" || r.LaunchID == "" || r.ResultSHA256 == "" || r.CompletionPath == "" || !r.CreditObserved {
		return fmt.Errorf("accepted native terminal/aggregate/finalizer credit incomplete")
	}
	if !r.EmptyResultRefused {
		return fmt.Errorf("actual empty-result refusal was not observed")
	}
	if r.Scenario == "early-resume" && (r.ResumeExitStatus != 0 || r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID || !r.ResumeWorkerStable || !r.ResumeNoSpawn || !r.ResumeInspectObserved || !r.FinalizationReplayStable) {
		return fmt.Errorf("fresh-session continuity/replay proof incomplete")
	}
	return nil
}

// Inspect actual parent calls separately. The native tool count is independent
// of the parent CLI's display schema; no assistant text counts as a launch.
func nativeInspectParentEvents(r *codexNativeLiveReceipt, raw []byte) {
	var metadata nativeHostEvent
	if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &metadata) != nil || metadata.Type != "session_meta" || metadata.Payload.ID != r.SessionID || metadata.Payload.ParentThreadID != "" {
		r.ParentSubstitution = true // attribution is missing; fail closed
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	spawns := 0
	for scanner.Scan() {
		var host nativeHostEvent
		if json.Unmarshal(scanner.Bytes(), &host) == nil && host.Type == "event_msg" && host.Payload.Type == "item_completed" && host.Payload.ThreadID == r.SessionID {
			i := host.Payload.Item
			if i.Type == "FileChange" && i.Status == "completed" {
				r.ParentSubstitution = true
			}
			if i.Type == "CommandExecution" {
				if !nativeParentCoordinationCommand(r, i.Command) {
					r.ParentSubstitution = true
					r.ParentUnclassified = append(r.ParentUnclassified, strings.Join(i.Command, " "))
				}
				var words []string
				if len(i.Command) == 3 {
					words, _ = nativeSimpleShellWords(i.Command[2])
				}
				if len(words) == 3 && words[1] == r.CoordinatorPath && words[2] == "empty-result" && i.ExitCode != nil && *i.ExitCode != 0 && strings.Contains(i.Output, "nonempty terminal result") {
					r.EmptyResultRefused = true
				}
				if len(words) == 3 && words[1] == r.CoordinatorPath && words[2] == "inspect" && i.ExitCode != nil && *i.ExitCode == 0 && strings.Contains(i.Output, r.ResultSHA256) && r.ResultSHA256 != "" {
					r.ResumeInspectObserved = true
				}
				if i.Status == "completed" && i.ExitCode != nil && *i.ExitCode == 0 && len(i.Command) == 3 {
					command := i.Command[2]
					for path, target := range map[string]*bool{r.SkillPath: &r.SkillRead, r.SupportPath: &r.SupportRead} {
						content, err := os.ReadFile(path)
						if err == nil && nativeLiveReadCommand(command, path) && strings.Contains(i.Output, strings.TrimSpace(string(content))) {
							*target = true
						}
					}
					if strings.Contains(command, "command-guide build --platform codex") && strings.Contains(i.Output, "codex-native-worker reserve") {
						r.GuideRead = true
					}
				}
			}
		}
		var e struct {
			Type    string `json:"type"`
			Payload struct {
				Type      string `json:"type"`
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
				Input     string `json:"input"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &e) != nil || e.Type != "response_item" {
			continue
		}
		p := e.Payload
		if p.Type != "function_call" && p.Type != "custom_tool_call" {
			continue
		}
		if p.Name == "spawn_agent" || strings.HasSuffix(p.Name, ".spawn_agent") {
			spawns++
			var args struct {
				Message string `json:"message"`
			}
			if json.Unmarshal([]byte(p.Arguments), &args) == nil {
				if strings.HasPrefix(args.Message, "gAAAA") {
					r.LaunchMessageEncoding = "host-encrypted"
					r.PromptDeliveryVerification = "Exact plaintext unavailable in exported launch event; runtime-issued prompt bytes/digest retained. Full context delivery qualification remains Plan 03/07."
				} else {
					r.LaunchMessageEncoding = "plaintext"
					if lifecycleDigest([]byte(args.Message)) == r.PromptSHA256 {
						r.PromptDeliveryVerification = "launch message equals runtime prompt digest"
					}
				}
			}
		}
		text := p.Arguments + p.Input
		// Match a source write target, not an unrelated temporary request write
		// batched with a summary that happens to mention clamp.go.
		if nativeParentSourceWrite(text) {
			r.ParentSubstitution = true
		}
	}
	if spawns > 0 {
		r.NativeSpawnCount = spawns
	}
}

func nativeParentCoordinationCommand(r *codexNativeLiveReceipt, command []string) bool {
	if len(command) != 3 || (command[1] != "-lc" && command[1] != "-c") {
		return false
	}
	// Observed host discovery spelling: a fixed read-only lookup, with no
	// caller-provided expression or writable target substituted into the shell.
	if command[2] == "rg --files \"$CODEX_HOME/sessions\"" {
		return true
	}
	words, ok := nativeSimpleShellWords(command[2])
	if !ok {
		return false
	}
	for len(words) > 0 && (strings.HasPrefix(words[0], "AETHER_OUTPUT_MODE=") || strings.HasPrefix(words[0], "AETHER_FORCE_COLOR=")) {
		words = words[1:]
	}
	if len(words) == 0 {
		return false
	}
	switch words[0] {
	case "python3":
		if len(words) < 3 || len(words) > 4 || words[1] != r.CoordinatorPath {
			return false
		}
		raw, err := os.ReadFile(r.CoordinatorPath)
		if err != nil || lifecycleDigest(raw) != r.CoordinatorSHA256 {
			return false
		}
		switch words[2] {
		case "manifest", "reserve", "record", "inspect", "stage", "finalize", "empty-result":
			return len(words) == 3
		case "bind":
			return len(words) == 4 && regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(words[3])
		}
	case "cat":
		return len(words) > 1 // simple words only: no expansion, pipes or redirection
	case "pwd":
		return len(words) == 1
	case "ls":
		return true
	case "sed":
		return len(words) == 4 && words[1] == "-n" && regexp.MustCompile(`^[0-9]+(?:,[0-9]+)?p$`).MatchString(words[2])
	case "head":
		return len(words) == 4 && words[1] == "-n" && regexp.MustCompile(`^[0-9]+$`).MatchString(words[2]) && !strings.HasPrefix(words[3], "-")
	case "jq":
		args := words[1:]
		if len(args) > 0 && (args[0] == "-r" || args[0] == "-c" || args[0] == "-S") {
			args = args[1:]
		}
		return len(args) == 2 && !strings.HasPrefix(args[0], "-") && !strings.HasPrefix(args[1], "-")
	case "git":
		if len(words) == 2 {
			return words[1] == "status" || words[1] == "diff"
		}
		if len(words) == 3 && ((words[1] == "status" && words[2] == "--short") || (words[1] == "diff" && words[2] == "--check")) {
			return true
		}
		return len(words) > 3 && words[1] == "diff" && words[2] == "--"
	case "aether":
		if len(words) < 2 {
			return false
		}
		switch words[1] {
		case "status", "pheromones", "command-guide", "ceremony", "spawn-log", "spawn-complete":
			return true
		}
	}
	return false
}

func nativeParentSourceWrite(text string) bool {
	return regexp.MustCompile(`(?m)(?:\*\*\* (?:Update|Add) File: [^\n]*clamp(?:_test)?\.go|>\s*['"]?(?:[^\s'"]*/)?clamp(?:_test)?\.go|(?:Path|open)\(['"](?:[^'"\n]*/)?clamp(?:_test)?\.go['"]\)(?:\.write_text|\.write_bytes)|open\(['"](?:[^'"\n]*/)?clamp(?:_test)?\.go['"],\s*['"]w)`).MatchString(text)
}

// A single successful cat may read both installed files. Match shell words,
// then require the exact installed bytes in its completed host output.
func nativeLiveReadCommand(command, path string) bool {
	command = unwrapCodexShellCommand(command)
	if liveSkillReadCommand(command, path) {
		return true
	}
	if !strings.HasPrefix(command, "cat ") {
		return false
	}
	for _, word := range strings.Fields(command[4:]) {
		if strings.Trim(word, "\"'") == path {
			return true
		}
	}
	return false
}

func nativeToolOutputText(raw json.RawMessage) string {
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var out strings.Builder
		for _, part := range parts {
			out.WriteString(part.Text)
			out.WriteByte(10)
		}
		return out.String()
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text
	}
	return string(raw)
}
