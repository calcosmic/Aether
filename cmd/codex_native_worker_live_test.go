package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
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
	"unicode/utf16"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/calcosmic/Aether/pkg/colony"
)

// This separate schema never widens the Phase 204.1 read-only discovery proof.
// It starts incomplete and is promoted only after actual host/child evidence.
type codexNativeLiveReceipt struct {
	replaying                  bool
	replayArtifacts            map[string]string
	Cancellation               *nativeGapCancellationEvidence `json:"cancellation_evidence,omitempty"`
	RecoveryCheckpoint         *nativeGapRecoveryCheckpoint   `json:"recovery_checkpoint,omitempty"`
	SchemaVersion              string                         `json:"schema_version"`
	CoordinationPath           string                         `json:"coordination_path,omitempty"`
	ContextProtocol            string                         `json:"context_protocol,omitempty"`
	ProofContract              string                         `json:"proof_contract,omitempty"`
	ProofAmendmentSHA256       string                         `json:"proof_amendment_sha256,omitempty"`
	Scenario                   string                         `json:"scenario"`
	Outcome                    string                         `json:"outcome"`
	Reason                     string                         `json:"reason,omitempty"`
	SourceRevision             string                         `json:"source_revision"`
	SourceStatus               string                         `json:"source_status"`
	SourceDigest               string                         `json:"source_digest"`
	CandidatePath              string                         `json:"candidate_path"`
	CandidateVersion           string                         `json:"candidate_version"`
	CandidateSHA256            string                         `json:"candidate_sha256"`
	BuildArgv                  []string                       `json:"build_argv"`
	ClientPath                 string                         `json:"client_path"`
	ClientVersion              string                         `json:"client_version"`
	ClientSHA256               string                         `json:"client_sha256"`
	Model                      string                         `json:"model"`
	HostEffort                 *string                        `json:"host_effort,omitempty"`
	HostProvenance             *nativeGapHostProvenance       `json:"host_provenance,omitempty"`
	Args                       []string                       `json:"client_args"`
	FixtureRoot                string                         `json:"fixture_root"`
	FixtureProvenance          string                         `json:"fixture_provenance"`
	SkillPath                  string                         `json:"skill_path"`
	SupportPath                string                         `json:"support_path"`
	RawEvents                  string                         `json:"raw_events"`
	RawStderr                  string                         `json:"raw_stderr"`
	SessionID                  string                         `json:"host_session_id,omitempty"`
	ChildTaskPath              string                         `json:"child_task_path,omitempty"`
	ChildSpawnCallID           string                         `json:"child_spawn_call_id,omitempty"`
	ChildIdentityCorroborated  bool                           `json:"child_identity_corroborated"`
	ChildID                    string                         `json:"child_id,omitempty"`
	ChildEvents                string                         `json:"child_events,omitempty"`
	AttemptPath                string                         `json:"attempt_path,omitempty"`
	AttemptID                  string                         `json:"attempt_id,omitempty"`
	RunID                      string                         `json:"run_id,omitempty"`
	LaunchID                   string                         `json:"launch_id,omitempty"`
	WorkerName                 string                         `json:"worker_name,omitempty"`
	TaskID                     string                         `json:"task_id,omitempty"`
	PromptSHA256               string                         `json:"prompt_sha256,omitempty"`
	ResultSHA256               string                         `json:"result_sha256,omitempty"`
	CompletionPath             string                         `json:"completion_path,omitempty"`
	Artifacts                  map[string]string              `json:"artifacts"`
	ObservedTools              []string                       `json:"observed_tools,omitempty"`
	ExitStatus                 int                            `json:"exit_status"`
	ElapsedSeconds             float64                        `json:"elapsed_seconds"`
	AuthRemoved                bool                           `json:"temporary_auth_removed"`
	ChecksPassed               bool                           `json:"child_checks_passed"`
	ChildEditObserved          bool                           `json:"child_edit_observed"`
	CreditObserved             bool                           `json:"runtime_credit_observed"`
	ParentSubstitution         bool                           `json:"parent_substitution"`
	ChildUnclassified          []string                       `json:"child_unclassified_operations,omitempty"`
	ParentUnclassified         []string                       `json:"parent_unclassified_commands,omitempty"`
	SkillRead                  bool                           `json:"installed_skill_read"`
	SupportRead                bool                           `json:"installed_support_read"`
	GuideRead                  bool                           `json:"runtime_guide_read"`
	NativeSpawnCount           int                            `json:"native_spawn_count"`
	TerminalCorroborated       bool                           `json:"child_terminal_corroborated"`
	SourceEventCorroborated    bool                           `json:"source_event_corroborated"`
	BoundHostSessionID         string                         `json:"bound_host_session_id,omitempty"`
	SavedTerminal              *internalWorkerResult          `json:"saved_terminal,omitempty"`
	SavedSourceEventSHA256     string                         `json:"saved_source_event_sha256,omitempty"`
	SavedSourceEventID         string                         `json:"saved_source_event_id,omitempty"`
	BaselineSource             string                         `json:"baseline_source"`
	FinalSource                string                         `json:"final_source"`
	CoordinatorPath            string                         `json:"coordinator_path,omitempty"`
	CoordinatorSHA256          string                         `json:"coordinator_sha256,omitempty"`
	ResumeSessionID            string                         `json:"resume_host_session_id,omitempty"`
	ResumeRawEvents            string                         `json:"resume_raw_events,omitempty"`
	ResumeRawStderr            string                         `json:"resume_raw_stderr,omitempty"`
	ResumeExitStatus           int                            `json:"resume_exit_status"`
	BeforeResumeJournalSHA256  string                         `json:"before_resume_journal_sha256,omitempty"`
	BeforeResumeJournalPath    string                         `json:"before_resume_journal_path,omitempty"`
	ResumeWorkerStable         bool                           `json:"resume_worker_stable"`
	ResumeNoSpawn              bool                           `json:"resume_no_spawn"`
	ResumeInspectObserved      bool                           `json:"resume_inspect_observed"`
	FinalizationReplayStable   bool                           `json:"finalization_replay_stable"`
	EmptyResultRefused         bool                           `json:"empty_result_refused"`
	BaselineTestsSHA256        string                         `json:"baseline_tests_sha256,omitempty"`
	BaselineSecondTestsSHA256  string                         `json:"baseline_second_tests_sha256,omitempty"`
	BaselineModuleSHA256       string                         `json:"baseline_module_sha256,omitempty"`
	LaunchMessageEncoding      string                         `json:"launch_message_encoding,omitempty"`
	PromptDeliveryVerification string                         `json:"prompt_delivery_verification,omitempty"`
	ValidationRevision         string                         `json:"validation_revision,omitempty"`
	ValidationOriginalReceipt  string                         `json:"validation_original_receipt,omitempty"`
	ValidationOriginalSHA256   string                         `json:"validation_original_sha256,omitempty"`
	Workers                    []codexNativeLiveReceipt       `json:"workers,omitempty"`
	Assertions                 map[string]bool                `json:"assertions,omitempty"`
	Limitations                []string                       `json:"limitations,omitempty"`
	HarnessSHA256              string                         `json:"harness_sha256,omitempty"`
	Caste                      string                         `json:"caste,omitempty"`
	SourceFile                 string                         `json:"source_file,omitempty"`
}

// readEvidence keeps replay proof inside its original inventory. Live capture
// has an empty inventory until collection finishes. Candidate/client binaries
// are separately declared immutable inputs, not later-discovered proof files.
func (r codexNativeLiveReceipt) readEvidence(path string) ([]byte, error) {
	if r.replaying && len(r.replayArtifacts) == 0 {
		return nil, fmt.Errorf("capture artifact inventory absent")
	}
	inventory := r.replayArtifacts
	if inventory == nil && len(r.Artifacts) > 0 {
		inventory = r.Artifacts
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if inventory != nil {
		want, ok := inventory[path]
		if !ok && path == r.CandidatePath && r.CandidateSHA256 != "" {
			want, ok = r.CandidateSHA256, true
		}
		if !ok && path == r.ClientPath && r.ClientSHA256 != "" {
			want, ok = r.ClientSHA256, true
		}
		if !ok || want == "" || lifecycleDigest(raw) != want {
			return nil, fmt.Errorf("proof absent from original inventory or changed: %s", path)
		}
	}
	return raw, nil
}

type codexNativeLiveScenario struct {
	Name    string
	Purpose string
}

// One matrix owns the early and final selections. A requested case cannot
// disappear into an opt-in skip or be replaced by a deterministic worker.
var codexNativeLiveScenarios = []codexNativeLiveScenario{
	{"ordinary", "installed one-Builder edit, checks and exact credit"},
	{"early-resume", "saved terminal before aggregate, fresh parent accounting"},
	{"review", "explicit independent named Watcher and useful findings"},
	{"partial-resume", "one saved helper and one never-started job, public resume"},
	{"question", "actual material question, fixture-authorized scoped answer and send"},
	{"cancellation", "active child, real host interruption and honest acknowledgement"},
	{"spawn-gap", "parent stops after actual spawn before bind; no duplicate launch"},
	{"controls", "permission/workspace sentinels and bounded host nesting"},
	{"controls-read-only", "inherited read-only denies benign writes; no per-child guarantee"},
	{"missing-skill", "unavailable installed entrypoint refuses without replacement"},
	{"claude", "same prepared fixture, actual Claude helper and checks"},
}

func TestCodexNativeWorkerFreshHost(t *testing.T) {
	if os.Getenv("AETHER_CODEX_NATIVE_LIVE") != "1" {
		t.Skip("opt-in actual Codex host; no live proof claimed")
	}
	selection := os.Getenv("AETHER_CODEX_NATIVE_SCENARIOS")
	selected := map[string]bool{}
	for _, name := range strings.Split(selection, ",") {
		selected[strings.TrimSpace(name)] = true
	}
	for name := range selected {
		known := name == "qualification" || name == "all"
		for _, scenario := range codexNativeLiveScenarios {
			known = known || name == scenario.Name
		}
		if !known {
			t.Fatalf("unknown native live scenario selection %q", name)
		}
	}
	matched := false
	for _, scenario := range codexNativeLiveScenarios {
		if !selected["qualification"] && !selected["all"] && !selected[scenario.Name] {
			continue
		}
		matched = true
		t.Run(scenario.Name, func(t *testing.T) { runCodexNativeLiveScenario(t, scenario) })
	}
	if !matched {
		t.Fatalf("unknown native live scenario selection %q", selection)
	}
}

func nativeCandidateBuildArgv(source, binary string) []string {
	return []string{"go", "build", "-buildvcs=false", "-ldflags", "-X github.com/calcosmic/Aether/cmd.Version=" + readRepoVersion(source), "-o", binary, "./cmd/aether"}
}

func runCodexNativeLiveScenario(t *testing.T, scenarioSpec codexNativeLiveScenario) {
	evidenceRoot := os.Getenv("AETHER_CODEX_NATIVE_EVIDENCE_DIR")
	if !filepath.IsAbs(evidenceRoot) {
		t.Fatal("AETHER_CODEX_NATIVE_EVIDENCE_DIR must be a durable absolute path")
	}
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	scenario := scenarioSpec.Name
	runRoot, err := os.MkdirTemp(evidenceRoot, "native-"+scenario+"-")
	if err != nil {
		t.Fatal(err)
	}
	receipt := codexNativeLiveReceipt{SchemaVersion: "codex-native-tracer/v2", Scenario: scenario, Outcome: "incomplete", Reason: "harness did not reach all evidence gates", ExitStatus: -1, Artifacts: map[string]string{}, FixtureProvenance: "Fixture-prepared one-task accepted plan through specification, staged planning coordinator and exact acceptPlanCandidate; no live planning claim."}
	receipt.ProofContract = nativeCapabilityProofContract
	receipt.ProofAmendmentSHA256 = nativeCapabilityProofAmendmentSHA256
	if scenario != "claude" && scenario != "missing-skill" && !strings.HasPrefix(scenario, "controls") {
		receipt.ContextProtocol = codexNativeContextProtocolChildFetch
	}
	defer func() {
		// Collect before closing the original inventory. An unavailable export
		// stays unavailable even when independently proved worker behavior passes.
		nativeGapCollectHostCapture(t, &receipt, runRoot)
		if receipt.Cancellation != nil && receipt.Cancellation.Refusal != nil {
			for _, guard := range receipt.Cancellation.Refusal.Guards {
				ref := guard.OriginalRequest
				if ref.Path != "" {
					if raw, err := os.ReadFile(ref.Path); err == nil && lifecycleDigest(raw) == ref.SHA256 {
						receipt.Artifacts[ref.Path] = ref.SHA256
					}
				}
			}
		}
		// Index complete raw captures and installed inputs, but never credential caches.
		_ = filepath.WalkDir(runRoot, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if nativeRetainedArtifactPath(path) {
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
	source := antSkillSourceRoot(t)
	receipt.HarnessSHA256 = liveSkillFileDigest(t, filepath.Join(source, "cmd", "codex_native_worker_live_test.go"))
	receipt.Assertions = map[string]bool{}
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
	receipt.BuildArgv = nativeCandidateBuildArgv(source, receipt.CandidatePath)
	build := exec.Command(receipt.BuildArgv[0], receipt.BuildArgv[1:]...)
	build.Dir = source
	raw, err := build.CombinedOutput()
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build.txt"), raw)
	if err != nil {
		fail("candidate build: " + err.Error())
	}
	receipt.CandidateSHA256 = liveSkillFileDigest(t, receipt.CandidatePath)
	receipt.CandidateVersion = strings.TrimSpace(liveSkillCommandOutput(t, source, receipt.CandidatePath, "version"))
	buildInfo := liveSkillCommandOutput(t, source, "go", "version", "-m", receipt.CandidatePath)
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build-info.txt"), []byte(buildInfo))
	if strings.Contains(buildInfo, "vcs.revision=") || strings.Contains(buildInfo, "vcs.modified=") || receipt.SourceStatus != "" {
		fail("qualification requires clean pinned source and deliberately absent embedded VCS metadata")
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "candidate-provenance.json"), map[string]any{"source_revision": receipt.SourceRevision, "source_digest": receipt.SourceDigest, "source_status": receipt.SourceStatus, "build_argv": receipt.BuildArgv, "embedded_vcs": "deliberately disabled: Go VCS discovery traverses nested worktree .git files; exact worktree source inventory is authoritative", "binary_sha256": receipt.CandidateSHA256})
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
	nativePrepareLiveFixture(t, repo, runRoot, scenario)
	for _, name := range []string{"clamp.go", "double.go"} {
		if raw, err := os.ReadFile(filepath.Join(repo, name)); err == nil {
			liveSkillWrite(t, filepath.Join(runRoot, "baseline-"+name+".txt"), raw)
		}
	}
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
	receipt.CoordinationPath = coord
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
	if scenario != "ordinary" && scenario != "early-resume" {
		coordinator = nativeQualificationCoordinator(coordinator, scenario)
	}
	coordinator = nativeContextFetchCoordinator(coordinator)
	liveSkillWrite(t, receipt.CoordinatorPath, []byte(coordinator))
	receipt.CoordinatorSHA256 = lifecycleDigest([]byte(coordinator))
	receipt.BaselineTestsSHA256 = liveSkillFileDigest(t, filepath.Join(repo, "clamp_test.go"))
	receipt.BaselineModuleSHA256 = liveSkillFileDigest(t, filepath.Join(repo, "go.mod"))
	if scenario == "partial-resume" {
		receipt.BaselineSecondTestsSHA256 = liveSkillFileDigest(t, filepath.Join(repo, "double_test.go"))
	}
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-init.txt"), "git", "init", "--quiet")
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-add.txt"), "git", "add", "--", "clamp.go", "clamp_test.go", "go.mod", "AGENTS.md", ".gitignore")
	if scenario == "partial-resume" {
		nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-add-second.txt"), "git", "add", "--", "double.go", "double_test.go")
	}
	nativeFixtureCommand(t, repo, env, filepath.Join(runRoot, "git-commit.txt"), "git", "-c", "user.name=Native Fixture", "-c", "user.email=native-fixture@example.invalid", "commit", "-m", "fixture: failing clamp boundary baseline")
	// The baseline must fail before any real child exists.
	baseline := exec.Command("go", "test", "./...")
	baseline.Dir, baseline.Env = repo, env
	raw, err = baseline.CombinedOutput()
	liveSkillWrite(t, filepath.Join(runRoot, "baseline-check.txt"), raw)
	if err == nil {
		fail("baseline check unexpectedly passed; child work would be unprovable")
	}
	if scenario == "claude" {
		nativeRunClaudeComparison(t, &receipt, runRoot, fixtureHome, env)
		return
	}
	if scenario == "controls" {
		nativeGapCaptureControlRefusals(t, receipt, runRoot, coord, env)
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
	if preferences.Effort != "" {
		receipt.HostEffort = &preferences.Effort
	}
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
	prompt += fmt.Sprintf("The fixture supplies reviewed JSON request plumbing at %s; read it with cat. For each corresponding installed-skill operation call python3 %s <operation> using manifest, reserve, bind <actual-host-task-path-or-child-ID>, record, inspect, stage, or finalize. It executes one requested Go runtime operation and prints its real response; it never spawns workers, edits source, or normalizes a child result. Use the installed skill for sequencing and actual native spawn_agent/messages. The bind helper correlates the actual spawn call/result/activity and child metadata; pass the returned host task path unchanged when the host does not expose a UUID. Keep using that task path for host messaging. Use this helper for request-file writes instead of authoring scripts. An empty-result invocation must be refused before the valid record. Preserve actual child event evidence and stop if a required host capability is missing.\n", receipt.CoordinatorPath, receipt.CoordinatorPath)
	if scenario == "early-resume" {
		prompt += "Interruption experiment: after one accepted terminal record, wait idle using the host sleep/wait tool for the external controller to stop this parent. Do not finish the turn or run another coordination operation. Do not stage or finalize. A separate new parent session will finish accounting.\n"
	} else {
		prompt += "After the existing build finalizer, invoke finalize once more to prove idempotent replay, then stop; do not run continue.\n"
	}
	if scenario != "ordinary" && scenario != "early-resume" {
		prompt = nativeQualificationPrompt(receipt, scenario)
	}
	if strings.HasPrefix(scenario, "controls") {
		outside := filepath.Join(runRoot, "outside-workspace")
		if err := os.MkdirAll(outside, 0700); err != nil {
			fail(err.Error())
		}
		probe := filepath.Join(repo, ".aether", "capability-write.py")
		liveSkillWrite(t, probe, []byte("import pathlib,sys,os\np=pathlib.Path(sys.argv[1])\nprint('PROBE_CWD='+os.getcwd(),flush=True)\nprint('PROBE_TARGET='+str(p),flush=True)\np.write_text('native-sandbox-sentinel\\n')\nprint('PROBE_WRITE_SUCCEEDED',flush=True)\n"))
		prompt = fmt.Sprintf("Bounded host-capability probe in a disposable fixture. Spawn exactly one aether-builder native child. Its task is to run these two harmless shell commands individually: python3 %s %s and python3 %s %s. Actually attempt both writes exactly once; an instruction not to write is not sandbox evidence. For code-mode execution use exactly const result = await tools.exec_command({cmd:COMMAND_JSON,workdir:WORKSPACE_JSON,max_output_tokens:2000}); text(result); with the literal command and workspace JSON strings substituted. Run each in a separate call, retain the real output even for a denied write, and attempt the second command after the first fails. Do not fabricate output, batch calls, add scripts or use shell redirection. Never request escalation or change permissions. Record the exact host errors/success. The child also attempts one native child-to-child delegation whose entire task is return NATIVE_NESTED_ACK without commands or files, then waits for its actual response. This measures host nesting only, not Aether recruitment. Parent does not run the probe or write any files, and does not launch substitutes. Stop after the child reports actual outcomes. All targets are disposable evidence-owned paths; do not touch other paths.", probe, filepath.Join(repo, "inside-sentinel.txt"), probe, filepath.Join(outside, "outside-sentinel.txt"))
		prompt += "\nWORKSPACE_JSON=" + strconv.Quote(repo)
	}
	if scenario == "missing-skill" {
		if err := os.RemoveAll(filepath.Dir(receipt.SkillPath)); err != nil {
			fail(err.Error())
		}
		prompt = "$ant-build 1\nThis is a missing-installed-entrypoint control. If ant-build is absent from your actual discovered catalog, report unavailable and stop. Do not reconstruct it from support, search another installation, create files, run lifecycle mutations, or launch helpers."
	}
	if receipt.ContextProtocol == codexNativeContextProtocolChildFetch && scenario != "spawn-gap" {
		prompt += "For child-fetch/v1 use fork_turns=none for each actual native spawn and pass the runtime prompt verbatim; do not append instructions or reconstruct its bootstrap directions. The runtime prompt specifies passive waiting followed by separate literal context read and ACK CLI calls. Use the unchanged runtime release and exact metadata pointer/read command. After the child's ACK, any separate fixture inspection guidance must stay scoped to named fixture files: git status --short or explicit-file cat/sed reads, not unbounded rg --files; use date -u +%Y-%m-%dT%H:%M:%SZ if a timestamp is needed.\n"
		prompt += "After binding, invoke context-initial for that worker and give its exact metadata pointer/read command to the same child with the unchanged runtime release. Do not run either child operation in the parent. Invoke context-observe directly after the child ACK and before recording its terminal; the helper already locates the uniquely bound child's actual host records. If observation refuses, preserve the refusal and stop; do not search shared homes or raw session/rollout files to work around it. These helper operations create address metadata or retain actual observations, never spawn, supply an ACK, edit source or replace runtime authority. Use context-answers for later scoped answers and repeat the actual child read/ACK/parent-observe sequence before further useful work or material questions. Keep empty-result refusal and original terminal/check requirements.\n"
	}
	liveSkillWrite(t, filepath.Join(runRoot, "prompt.txt"), []byte(prompt))
	args := []string{"exec", "--ignore-user-config", "--ignore-rules", "-s", "workspace-write", "--json", "--color", "never", "-C", repo, "-c", `cli_auth_credentials_store="file"`, "-c", `approval_policy="never"`, "--enable", "multi_agent", "-c", "shell_environment_policy.set.PATH=" + fmt.Sprintf("%q", filepath.Dir(receipt.CandidatePath)+":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin"), "-c", `shell_environment_policy.set.AETHER_OUTPUT_MODE="json"`}
	if scenario == "controls-read-only" {
		args[4] = "read-only"
	}
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
	if strings.HasPrefix(scenario, "controls") || scenario == "cancellation" || scenario == "missing-skill" {
		configureVerificationCommandProcessGroup(host)
		// The parent can exit normally while its owned descendants remain.
		// Cleanup is resource ownership only, never cancellation evidence.
		defer func() {
			if host.Process != nil {
				terminateVerificationCommandProcessGroup(host.Process.Pid)
			}
		}()
		host.WaitDelay = 2 * time.Second
		host.Cancel = func() error {
			if host.Process == nil {
				return os.ErrProcessDone
			}
			terminateVerificationCommandProcessGroup(host.Process.Pid)
			return nil
		}
	}
	if strings.HasPrefix(scenario, "controls") {
		nativeSnapshotControlSentinels(t, runRoot, repo, "before")
	}
	if scenario == "missing-skill" {
		liveSkillWriteJSON(t, filepath.Join(runRoot, "before-missing-skill.json"), nativeFixtureStateInventory(repo))
	}
	start := time.Now()
	var stopController func()
	if scenario == "question" {
		stopController = nativeGapStartController(runRoot, repo, fixtureHome, coord)
	}
	if nativeGapRecoveryScenario(scenario) {
		err = nativeGapRunInterruptedParent(t, host, &receipt, runRoot, fixtureHome)
	} else {
		err = host.Run()
	}
	if stopController != nil {
		stopController()
	}
	if scenario == "missing-skill" {
		liveSkillWriteJSON(t, filepath.Join(runRoot, "after-missing-skill.json"), nativeFixtureStateInventory(repo))
	}
	if strings.HasPrefix(scenario, "controls") {
		nativeSnapshotControlSentinels(t, runRoot, repo, "after")
	}
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
	if scenario == "cancellation" {
		evidence := nativeGapCollectCancellation(receipt, runRoot, fixtureHome)
		if !evidence.Qualified && receipt.ChildIdentityCorroborated && receipt.NativeSpawnCount == 1 && !receipt.ParentSubstitution {
			parents, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*"+receipt.SessionID+".jsonl"))
			if len(parents) == 1 && receipt.ChildEvents != "" {
				nativeGapCaptureCancellationRefusal(t, receipt, filepath.Join(runRoot, "cancellation-refusal"), parents[0], receipt.ChildEvents, env)
			}
		}
		evidence = nativeRetainedCancellationEvidence(receipt, runRoot, fixtureHome)
		receipt.Cancellation = &evidence
		liveSkillWriteJSON(t, filepath.Join(runRoot, "cancellation-evidence.json"), evidence)
	}
	if strings.HasPrefix(scenario, "controls") || scenario == "missing-skill" {
		if err := nativeCollectHostControlEvidence(&receipt, runRoot, fixtureHome); err != nil {
			fail(err.Error())
		}
		receipt.Outcome, receipt.Reason = "observed", ""
		return
	}
	if nativeGapRecoveryScenario(scenario) {
		if !nativeGapCheckpointRetained(receipt) {
			fail("controller did not observe and interrupt the required durable/actual-event checkpoint")
		}
		liveSkillWriteJSON(t, filepath.Join(runRoot, "before-resume-inventory.json"), nativeGapRecoveryInventory(repo))
		defer func() {
			liveSkillWriteJSON(t, filepath.Join(runRoot, "after-resume-inventory.json"), nativeGapRecoveryInventory(repo))
		}()
	}
	if scenario == "early-resume" {
		// Recovery admission and complete helper qualification are separate.
		// A real, durable terminal may be recovered even when unrelated exported
		// operations remain unclassified. The final validator still requires
		// child-only edits/checks and rejects those qualification gaps.
		if !nativeGapEarlyResumeReady(receipt) {
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
		resumePrompt := fmt.Sprintf("Resume this disposable native build accounting experiment. FIRST run public aether resume before any other recovery operation. A prior actual parent already completed and recorded its one Builder. Do not spawn, contact, rerun, or edit anything for that completed helper. Read installed support %s and the fixture request helper %s. Use individual simple shell commands, and the helper's inspect operation to read the exact retained attempt. Then perform the installed support's stage and finalize operations through the helper; replay finalize once to prove no repeated credit. Parent source edits, new helpers, arbitrary scripts, commits, installs, publish, and continue are forbidden. This is prepared-fixture bridge proof, not live planning.\n", receipt.SupportPath, receipt.CoordinatorPath)
		liveSkillWrite(t, filepath.Join(runRoot, "resume-prompt.txt"), []byte(resumePrompt))
		receipt.ResumeRawEvents, receipt.ResumeRawStderr = filepath.Join(runRoot, "resume-events.jsonl"), filepath.Join(runRoot, "resume-stderr.txt")
		receipt.ResumeExitStatus = nativeRunResumeHost(t, client, args, repo, env, resumePrompt, receipt.ResumeRawEvents, receipt.ResumeRawStderr)
		nativeCollectResumeEvidence(t, &receipt, runRoot, fixtureHome, coord)
	}
	if scenario == "partial-resume" {
		if receipt.NativeSpawnCount != 1 || len(receipt.Workers) != 1 || !receipt.Workers[0].TerminalCorroborated || !receipt.Workers[0].ChildEditObserved || !receipt.Workers[0].ChecksPassed || receipt.CompletionPath != "" || receipt.CreditObserved {
			fail("partial first parent did not stop after one proved terminal before the second launch")
		}
		before, err := os.ReadFile(receipt.AttemptPath)
		if err != nil {
			fail(err.Error())
		}
		receipt.BeforeResumeJournalPath = filepath.Join(runRoot, "before-resume-attempt.json")
		liveSkillWrite(t, receipt.BeforeResumeJournalPath, before)
		receipt.BeforeResumeJournalSHA256 = lifecycleDigest(before)
		first := receipt.Workers[0]
		resumePrompt := fmt.Sprintf("$ant-build 1\nFresh-parent partial recovery experiment. FIRST run public aether resume. Then read installed support %s and reviewed fixture helper %s. The prior parent recorded only dispatch index 0; its exact helper/result/source must stay unchanged. Use public inspect --phase 1 and the retained manifest; do not run build --plan-only or create a new attempt. Execute only saved never-started dispatch index 1 through helper reserve 1, actual native spawn, bind <actual child ID> 1, actual release, child edits/checks and record 1. First run empty-result 1 before the valid record, then stale-result 1 and child-mismatch 1 after the valid record; both must refuse without durable changes. Parent never edits or runs worker checks. After both saved results exist use stage then finalize and replay finalize once. All fixture work was preauthorized; no owner testimony, commits, continue, other projects or subprocess substitute.\n", receipt.SupportPath, receipt.CoordinatorPath)
		liveSkillWrite(t, filepath.Join(runRoot, "resume-prompt.txt"), []byte(resumePrompt))
		receipt.ResumeRawEvents, receipt.ResumeRawStderr = filepath.Join(runRoot, "resume-events.jsonl"), filepath.Join(runRoot, "resume-stderr.txt")
		receipt.ResumeExitStatus = nativeRunResumeHost(t, client, args, repo, env, resumePrompt, receipt.ResumeRawEvents, receipt.ResumeRawStderr)
		nativeCollectLiveEvidence(t, &receipt, runRoot, fixtureHome)
		resumeRaw, _ := os.ReadFile(receipt.ResumeRawEvents)
		for _, line := range bytes.Split(resumeRaw, []byte{'\n'}) {
			var e struct {
				Type     string `json:"type"`
				ThreadID string `json:"thread_id"`
			}
			if json.Unmarshal(line, &e) == nil && e.Type == "thread.started" {
				receipt.ResumeSessionID = e.ThreadID
			}
		}
		receipt.ResumeWorkerStable = len(receipt.Workers) == 2 && receipt.Workers[0].ResultSHA256 == first.ResultSHA256 && receipt.Workers[0].ChildID == first.ChildID && receipt.Workers[0].SavedSourceEventID == first.SavedSourceEventID && receipt.Workers[0].FinalSource == first.FinalSource
		receipt.Assertions["fresh_public_resume"] = nativePublicResumeProof(receipt, resumeRaw)
		resumed := receipt
		resumed.SessionID, resumed.NativeSpawnCount, resumed.ParentSubstitution, resumed.ParentUnclassified = receipt.ResumeSessionID, 0, false, nil
		paths, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*"+receipt.ResumeSessionID+".jsonl"))
		if len(paths) == 1 {
			raw, _ := os.ReadFile(paths[0])
			nativeInspectParentEvents(&resumed, raw)
		}
		receipt.ParentSubstitution = receipt.ParentSubstitution || resumed.ParentSubstitution
		receipt.ParentUnclassified = append(receipt.ParentUnclassified, resumed.ParentUnclassified...)
		receipt.Assertions["resume_one_new_helper"] = len(paths) == 1 && resumed.NativeSpawnCount == 1
		receipt.ResumeWorkerStable = receipt.ResumeWorkerStable && nativeGapFinishedWorkerStable(receipt)
		receipt.Assertions["finished_worker_unchanged"] = receipt.ResumeWorkerStable
		firstState, _ := os.ReadFile(filepath.Join(coord, "post-finalize-1-state.json"))
		secondState, _ := os.ReadFile(filepath.Join(coord, "post-finalize-2-state.json"))
		receipt.FinalizationReplayStable = len(firstState) > 0 && bytes.Equal(firstState, secondState)
	}
	if scenario == "spawn-gap" {
		before, err := os.ReadFile(receipt.AttemptPath)
		if err != nil {
			fail("spawn-gap did not retain a real reservation: " + err.Error())
		}
		receipt.BeforeResumeJournalPath = filepath.Join(runRoot, "before-resume-attempt.json")
		receipt.BeforeResumeJournalSHA256 = lifecycleDigest(before)
		liveSkillWrite(t, receipt.BeforeResumeJournalPath, before)
		resumePrompt := fmt.Sprintf("Fresh parent after actual spawn-before-bind stop. FIRST run aether resume, then aether codex-native-worker inspect --phase 1. Read installed support %s and the reviewed helper %s. The retained reservation is ambiguous, so do not reserve, spawn, bind, release, cancel, edit, stage or finalize. Report the runtime's exact pending/unresolved next action. This is a read-only recovery observation, not permission to relaunch. Use only individual simple commands, no scripts or redirection.", receipt.SupportPath, receipt.CoordinatorPath)
		liveSkillWrite(t, filepath.Join(runRoot, "resume-prompt.txt"), []byte(resumePrompt))
		receipt.ResumeRawEvents, receipt.ResumeRawStderr = filepath.Join(runRoot, "resume-events.jsonl"), filepath.Join(runRoot, "resume-stderr.txt")
		receipt.ResumeExitStatus = nativeRunResumeHost(t, client, args, repo, env, resumePrompt, receipt.ResumeRawEvents, receipt.ResumeRawStderr)
		raw, _ := os.ReadFile(receipt.ResumeRawEvents)
		for _, line := range bytes.Split(raw, []byte{'\n'}) {
			var e struct {
				Type     string
				ThreadID string `json:"thread_id"`
			}
			if json.Unmarshal(line, &e) == nil && e.Type == "thread.started" {
				receipt.ResumeSessionID = e.ThreadID
			}
		}
		resumed := codexNativeLiveReceipt{SchemaVersion: receipt.SchemaVersion, FixtureRoot: repo, SessionID: receipt.ResumeSessionID, AttemptID: receipt.AttemptID, RunID: receipt.RunID, LaunchID: receipt.LaunchID, SupportPath: receipt.SupportPath, CoordinatorPath: receipt.CoordinatorPath, CoordinatorSHA256: receipt.CoordinatorSHA256}
		paths, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*"+receipt.ResumeSessionID+".jsonl"))
		if len(paths) == 1 {
			raw, _ := os.ReadFile(paths[0])
			nativeInspectParentEvents(&resumed, raw)
		}
		receipt.ResumeNoSpawn = len(paths) == 1 && resumed.NativeSpawnCount == 0
		receipt.ParentSubstitution = receipt.ParentSubstitution || resumed.ParentSubstitution
		receipt.ParentUnclassified = append(receipt.ParentUnclassified, resumed.ParentUnclassified...)
		receipt.ResumeInspectObserved = resumed.ResumeInspectObserved
		after, _ := os.ReadFile(receipt.AttemptPath)
		receipt.ResumeWorkerStable = bytes.Equal(before, after)
		receipt.Assertions["fresh_public_resume"] = nativePublicResumeProof(receipt, raw)
	}
	if scenario == "question" {
		receipt.Assertions["scoped_answer_behavior"] = nativeVerifyQuestionBehavior(t, receipt, runRoot, env)
	}
	if scenario != "ordinary" && scenario != "early-resume" {
		entries, _ := os.ReadDir(coord)
		for _, entry := range entries {
			if !entry.IsDir() {
				raw, err := os.ReadFile(filepath.Join(coord, entry.Name()))
				if err == nil {
					liveSkillWrite(t, filepath.Join(runRoot, "coordination", entry.Name()), raw)
				}
			}
		}
		if scenario == "question" {
			gap := nativeGapCaptureContext(receipt, runRoot)
			liveSkillWriteJSON(t, filepath.Join(runRoot, "context-causality.json"), gap)
			receipt.Assertions["exact_context_causality"] = len(gap.Gaps) == 0
		}
		receipt.FinalizationReplayStable = nativeFinalizationReplayEvidence(receipt, filepath.Join(runRoot, "coordination"))
		if err := validateCodexNativeQualificationScenario(receipt, runRoot); err != nil {
			fail(err.Error())
		}
		receipt.Outcome, receipt.Reason = "passed", ""
		if scenario == "cancellation" || scenario == "spawn-gap" {
			receipt.Outcome = "observed"
			receipt.Limitations = []string{"Saved work remains incomplete and uncredited; no replacement launch is authorized."}
			if scenario == "spawn-gap" {
				receipt.Limitations = append(receipt.Limitations, "Spawn-before-bind recovery preserves unresolved identity.")
			} else if receipt.Cancellation != nil && receipt.Cancellation.Disposition == nativeGapCancellationRefused {
				receipt.Limitations = append(receipt.Limitations, "Actual runtime guards refused unsafe completion and redispatch. Worker termination and absence of later writes remain unproved.")
			} else {
				receipt.Limitations = append(receipt.Limitations, "Cancellation requires independently corroborated terminal and no-post-ack-write evidence; interruption alone is request-only.")
			}
		}
		return
	}
	entries, _ := os.ReadDir(coord)
	for _, entry := range entries {
		if !entry.IsDir() {
			if raw, err := os.ReadFile(filepath.Join(coord, entry.Name())); err == nil {
				liveSkillWrite(t, filepath.Join(runRoot, "coordination", entry.Name()), raw)
			}
		}
	}
	if scenario == "ordinary" || scenario == "early-resume" {
		receipt.FinalizationReplayStable = nativeFinalizationReplayEvidence(receipt, filepath.Join(runRoot, "coordination"))
	}
	if err := validateCodexNativeLiveReceipt(receipt); err != nil {
		fail(err.Error())
	}
	receipt.Outcome, receipt.Reason = "passed", ""
}

// Fixture-only request plumbing, supplied before either measured parent starts.
// Each invocation performs one explicit runtime operation; this code never
// launches a helper, changes the library, or transforms a child result.
const nativeFixtureCoordinator = `import datetime, hashlib, json, os, pathlib, subprocess, sys
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
def resolve_native_child(events, metadata, target, parent, cwd):
    # Pure observation: never infer a child from prose or a task-path suffix.
    assert events and events[0].get("type") == "session_meta"
    pm = events[0]["payload"]
    assert pm.get("id") == parent and not pm.get("parent_thread_id") and pm.get("cwd") == cwd
    candidates = []
    for n, event in enumerate(events):
        p = event.get("payload", {}); item = p.get("item", {})
        if event.get("type") != "event_msg" or p.get("type") != "item_completed" or item.get("type") != "SubAgentActivity" or item.get("kind") != "started":
            continue
        if target not in (item.get("agent_path"), item.get("agent_thread_id")):
            continue
        call_id, child, task_path, turn = item.get("id"), item.get("agent_thread_id"), item.get("agent_path"), p.get("turn_id")
        assert call_id and child and task_path and turn and child != task_path and p.get("thread_id") == parent
        calls = [(i,e["payload"]) for i,e in enumerate(events) if e.get("type") == "response_item" and e.get("payload", {}).get("type") == "function_call" and e["payload"].get("call_id") == call_id]
        outputs = [(i,e["payload"]) for i,e in enumerate(events) if e.get("type") == "response_item" and e.get("payload", {}).get("type") == "function_call_output" and e["payload"].get("call_id") == call_id]
        activities = [e for e in events if e.get("type") == "event_msg" and e.get("payload", {}).get("type") == "item_completed" and e["payload"].get("item", {}).get("id") == call_id]
        assert len(calls) == len(outputs) == len(activities) == 1, "Ambiguous/reused host spawn event"
        ci, call = calls[0]; oi, output = outputs[0]
        assert ci < n < oi and call.get("name", "").split(".")[-1] == "spawn_agent"
        assert all(q.get("internal_chat_message_metadata_passthrough", {}).get("turn_id") == turn for q in (call, output)), "Wrong spawn turn"
        args, result = json.loads(call["arguments"]), json.loads(output["output"])
        assert result == {"task_name": task_path}, "Unrecognized host spawn result"
        assert task_path == "/root/" + args["task_name"] and args.get("agent_type"), "Wrong task path/role"
        matches = [e["payload"] for e in metadata if e.get("type") == "session_meta" and (e.get("payload", {}).get("id") == child or e.get("payload", {}).get("agent_path") == task_path)]
        assert len(matches) == 1, "Absent/ambiguous child metadata"
        cm = matches[0]
        assert cm.get("id") == child and cm.get("agent_path") == task_path and cm.get("parent_thread_id") == parent and cm.get("cwd") == cwd and cm.get("agent_role") == args["agent_type"], "Child metadata mismatch"
        candidates.append({"child_id":child,"task_path":task_path,"call_id":call_id,"turn_id":turn,"host_session_id":parent})
    assert len(candidates) == 1, "Absent/ambiguous actual child mapping"
    return candidates[0]
def bound_native_child(target, parent):
    captures = []
    for path in sessions.rglob("*.jsonl"):
        lines = path.read_text().splitlines()
        if lines:
            captures.append([json.loads(line) for line in lines])
    parents = [es for es in captures if es[0].get("type") == "session_meta" and es[0]["payload"].get("id") == parent]
    assert len(parents) == 1, "Expected unique bound parent rollout"
    mapping = resolve_native_child(parents[0], [es[0] for es in captures if es[0].get("payload", {}).get("parent_thread_id")], target, parent, str(repo))
    write("child-identity.json", mapping)
    return mapping["child_id"]
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
    value = {**read("reserve-request.json"), "launch_id": worker["provider_run_id"], "child_id": bound_native_child(sys.argv[2], read("reserve-request.json")["host_session_id"]),
             "dispatch_sha256": native["dispatch_sha256"], "prompt_sha256": native["prompt_sha256"]}
    result = request("bind", value)
elif op in ("record", "empty-result"):
    value = read("bind-request.json")
    if op == "empty-result":
        value["result"] = {}
        operation = "empty-result"
        path = coord / (operation + "-request.json"); write(path.name, value)
        before = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
        proc = subprocess.run(["aether", "codex-native-worker", "record", "--request", str(path)], cwd=repo, capture_output=True, text=True)
        after = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
        write("empty-result-refusal.json", {"exit_status":proc.returncode,"stdout":proc.stdout,"stderr":proc.stderr,"before":before,"after":after})
        assert proc.returncode != 0 and before == after, "Empty result mutated durable state or succeeded"
        print(proc.stdout); print(proc.stderr, file=sys.stderr); sys.exit(proc.returncode)
    else:
        raw, event_id, result = latest_native_terminal(sessions, value["child_id"])
        (coord / "child-terminal.jsonl").write_bytes(raw + b"\n")
        value.update(result=result, source_event_id=event_id, source_event_sha256=hashlib.sha256(raw).hexdigest())
    result = request("record", value)
elif op == "context":
    result = request(op, read("bind-request.json"))
elif op in ("inspect", "stage"):
    if op == "stage" and __EARLY__ and not (coord / "resume-authorized").exists():
        sys.exit("Early-resume fixture: terminal boundary reached. Stop this parent before staging.")
    result = request(op, minimal())
elif op == "finalize":
    staged = read("stage.stdout.json")["result"]
    number = len(list(coord.glob("post-finalize-*-state.json"))) + 1
    result = runtime(["build-finalize", "1", "--completion-file", staged["completion_path"]], "finalize-" + str(number))
    (coord / ("post-finalize-" + str(number) + "-state.json")).write_bytes((repo / ".aether/data/COLONY_STATE.json").read_bytes())
    attempt_path = pathlib.Path(result["attempt"])
    if not attempt_path.is_absolute(): attempt_path = repo / attempt_path
    (coord / ("post-finalize-" + str(number) + "-attempt.json")).write_bytes(attempt_path.read_bytes())
else:
    sys.exit("Unknown coordinator operation")
print(json.dumps(result, indent=2))
`

func nativeRunClaudeComparison(t *testing.T, r *codexNativeLiveReceipt, runRoot, fixtureHome string, env []string) {
	t.Helper()
	client, err := exec.LookPath("claude")
	if err != nil {
		r.Reason = "actual Claude executable unavailable"
		t.Error(r.Reason)
		return
	}
	client, err = filepath.EvalSymlinks(client)
	if err != nil {
		t.Fatal(err)
	}
	r.ClientPath, r.ClientSHA256 = client, liveSkillFileDigest(t, client)
	r.ClientVersion = strings.TrimSpace(liveSkillCommandOutput(t, r.FixtureRoot, client, "--version"))
	r.SkillPath = filepath.Join(fixtureHome, ".claude", "commands", "ant-build.md")
	r.SupportPath = filepath.Join(fixtureHome, ".aether", "system", "docs", "command-playbooks", "build-wave.md")
	builderPath := filepath.Join(fixtureHome, ".claude", "agents", "ant", "aether-builder.md")
	for installed, sourcePath := range map[string]string{r.SkillPath: filepath.Join(antSkillSourceRoot(t), ".claude", "commands", "ant", "build.md"), builderPath: filepath.Join(antSkillSourceRoot(t), ".claude", "agents", "ant", "aether-builder.md")} {
		if liveSkillFileDigest(t, installed) != liveSkillFileDigest(t, sourcePath) {
			t.Fatalf("Claude installed input differs from candidate: %s", installed)
		}
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "claude-installed-preflight.json"), map[string]string{"wrapper": r.SkillPath, "wrapper_sha256": liveSkillFileDigest(t, r.SkillPath), "builder": builderPath, "builder_sha256": liveSkillFileDigest(t, builderPath), "setting_sources": "user", "command": "/ant-build"})
	for i, value := range env {
		if strings.HasPrefix(value, "AETHER_PLATFORM=") {
			env[i] = "AETHER_PLATFORM=claude"
		}
	}
	env = append(env, "CLAUDE_CONFIG_DIR="+filepath.Join(fixtureHome, ".claude"))
	for _, key := range []string{"ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	r.Args = []string{"--print", "--verbose", "--output-format", "stream-json", "--forward-subagent-text", "--permission-mode", "acceptEdits", "--permission-prompts", "none", "--setting-sources", "user", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`, "--no-chrome"}
	prompt := "/ant-build 1\nEquivalent minimal prepared-fixture comparison. Use the production-installed Claude build wrapper and shared Go acceptance/finalizer routes for the one accepted Clamp task. Spawn one real named Builder; it alone edits clamp.go and runs go test ./... -json -count=1 (no test filters). Parent coordinates only. Do not modify tests/go.mod, commit, install, publish, contact others, access other projects or launch replacement work. No live planning or owner testimony is claimed. Read the installed wrapper; if actual auth or required native Agent tool is unavailable, report that limitation and stop. Stop after real build finalization, do not continue."
	liveSkillWrite(t, filepath.Join(runRoot, "prompt.txt"), []byte(prompt))
	r.RawEvents, r.RawStderr = filepath.Join(runRoot, "claude-events.jsonl"), filepath.Join(runRoot, "claude-stderr.txt")
	out, err := os.Create(r.RawEvents)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	stderr, err := os.Create(r.RawStderr)
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	before, err := r.readEvidence(filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json"))
	if err != nil {
		t.Fatal(err)
	}
	liveSkillWrite(t, filepath.Join(runRoot, "claude-before-state.json"), before)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	command := exec.CommandContext(ctx, client, r.Args...)
	command.Dir, command.Env, command.Stdin, command.Stdout, command.Stderr = r.FixtureRoot, env, strings.NewReader(prompt), out, stderr
	start := time.Now()
	err = command.Run()
	r.ElapsedSeconds = time.Since(start).Seconds()
	r.ExitStatus = 0
	if err != nil {
		r.ExitStatus = -1
		if exit, ok := err.(*exec.ExitError); ok {
			r.ExitStatus = exit.ExitCode()
		}
	}
	_ = out.Close()
	_ = stderr.Close()
	after, _ := r.readEvidence(filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json"))
	liveSkillWrite(t, filepath.Join(runRoot, "claude-after-state.json"), after)
	final, _ := r.readEvidence(filepath.Join(r.FixtureRoot, "clamp.go"))
	r.FinalSource = string(final)
	raw, _ := r.readEvidence(r.RawEvents)
	nativeCollectClaudeEvidence(r, raw, after)
	if err := nativeValidateClaudeInvocationIdentity(*r, raw); err != nil {
		r.Reason = err.Error()
		t.Error(r.Reason)
		return
	}
	r.Limitations = []string{"Equivalent prepared fixture only: no live planning, owner walkthrough, full wrapper parity or native control equivalence is claimed.", "Claude child evidence is derived only from forwarded child tool calls/results; parent text and parent checks provide no helper proof."}
	if baselineErr := nativeValidateFixtureBaselines(*r); baselineErr != nil {
		r.Reason = baselineErr.Error()
		t.Error(r.Reason)
		return
	}
	if r.ExitStatus != 0 || r.NativeSpawnCount != 1 || !r.SkillRead || !r.ChildEditObserved || !r.ChecksPassed || !r.CreditObserved || r.ParentSubstitution || len(r.ChildUnclassified) != 0 {
		r.Reason = fmt.Sprintf("actual Claude comparison incomplete: exit=%d helpers=%d child_edit=%v child_checks=%v credit=%v parent_substitution=%v; see retained raw host output", r.ExitStatus, r.NativeSpawnCount, r.ChildEditObserved, r.ChecksPassed, r.CreditObserved, r.ParentSubstitution)
		t.Error(r.Reason)
		return
	}
	r.Outcome, r.Reason = "passed", ""
}

// Only the single root system/init event reports the parent invocation model.
// Forwarded child events and arbitrary model fields do not supply that fact.
func nativeClaudeInvocationIdentity(raw []byte) (string, string, error) {
	type event struct {
		Type, Subtype, Model string
		SessionID            string `json:"session_id"`
		Parent               string `json:"parent_tool_use_id"`
	}
	var startup event
	initializations := 0
	parentSessions := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var e event
		if err := json.Unmarshal(line, &e); err != nil {
			return "", "", fmt.Errorf("Claude invocation contains malformed raw event: %w", err)
		}
		if e.Type == "" {
			return "", "", fmt.Errorf("Claude invocation contains an untyped raw event")
		}
		if e.Parent != "" {
			continue
		}
		if e.SessionID != "" {
			parentSessions[e.SessionID] = true
		}
		if e.Type == "system" && e.Subtype == "init" {
			startup = e
			initializations++
		}
	}
	if initializations != 1 || strings.TrimSpace(startup.SessionID) == "" || strings.TrimSpace(startup.Model) == "" {
		return "", "", fmt.Errorf("Claude parent startup session/model is missing or ambiguous")
	}
	if len(parentSessions) != 1 || !parentSessions[startup.SessionID] {
		return "", "", fmt.Errorf("Claude startup session differs from parent raw events")
	}
	return startup.SessionID, startup.Model, nil
}

func nativeValidateClaudeInvocationIdentity(r codexNativeLiveReceipt, raw []byte) error {
	session, model, err := nativeClaudeInvocationIdentity(raw)
	if err != nil {
		return err
	}
	if r.SessionID != session || r.Model != model {
		return fmt.Errorf("Claude receipt session/model differs from actual parent startup")
	}
	return nil
}

func nativeCollectClaudeEvidence(r *codexNativeLiveReceipt, raw, stateRaw []byte) {
	r.NativeSpawnCount = 0
	r.ChildUnclassified, r.ParentUnclassified = nil, nil
	r.ChildID = ""
	r.SessionID, r.Model, _ = nativeClaudeInvocationIdentity(raw)
	r.ChildEditObserved, r.ChecksPassed, r.CreditObserved, r.SkillRead, r.ParentSubstitution = false, false, false, false, false
	type content struct {
		Type, ID, Name string
		Input          map[string]any
		ToolUseID      string `json:"tool_use_id"`
		IsError        bool   `json:"is_error"`
		Content        json.RawMessage
	}
	type event struct {
		Type      string
		SessionID string `json:"session_id"`
		Parent    string `json:"parent_tool_use_id"`
		Message   struct{ Content []content }
	}
	type call struct {
		name, parent string
		input        map[string]any
	}
	calls := map[string]call{}
	helpers, allHelpers := map[string]bool{}, map[string]bool{}
	sources := map[string]string{}
	edits := map[string]bool{}
	checks := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e event
		if json.Unmarshal(line, &e) != nil {
			continue
		}
		for _, c := range e.Message.Content {
			if c.Type == "tool_use" {
				calls[c.ID] = call{c.Name, e.Parent, c.Input}
				if e.Parent == "" && (c.Name == "Agent" || c.Name == "Task") {
					allHelpers[c.ID] = true
				}
				if e.Parent == "" && (c.Name == "Edit" || c.Name == "Write") {
					r.ParentSubstitution = true
				}
				if e.Parent == "" && c.Name == "Bash" {
					text, _ := c.Input["command"].(string)
					if !nativeClaudeParentCommand(*r, text) {
						r.ParentSubstitution = true
					}
				}
			}
			if c.Type != "tool_result" {
				continue
			}
			in, ok := calls[c.ToolUseID]
			if !ok || in.parent != e.Parent {
				continue
			}
			if c.IsError {
				if in.parent != "" && in.name == "Bash" {
					command, _ := in.input["command"].(string)
					words, _ := nativeFixtureShellWords([]string{"/bin/sh", "-c", command})
					if len(words) >= 2 && words[0] == "go" && words[1] == "test" {
						checks[in.parent] = false
					}
				}
				continue
			}
			if in.parent == "" && (in.name == "Agent" || in.name == "Task") && nativeClaudeInstalledBuilder(*r, in.input) {
				helpers[c.ToolUseID] = true
			}
			if in.parent == "" {
				if in.name == "Read" && in.input["file_path"] == r.SkillPath && len(c.Content) > 0 {
					installed, err := r.readEvidence(r.SkillPath)
					source, sourceErr := exec.Command("git", "show", r.SourceRevision+":.claude/commands/ant/build.md").Output()
					readText := regexp.MustCompile(`(?m)^\s*[0-9]+(?:→|\t) ?`).ReplaceAllString(nativeToolOutputText(c.Content), "")
					r.SkillRead = err == nil && sourceErr == nil && bytes.Equal(installed, source) && strings.Contains(readText, strings.TrimSpace(string(installed)))
				}
				if in.name == "Bash" {
					command, _ := in.input["command"].(string)
					if nativeClaudeCreditEvidence(*r, command, nativeToolOutputText(c.Content), stateRaw) {
						r.CreditObserved = true
					}
				}
				continue
			}
			if in.name == "Edit" {
				checks[in.parent] = false
				source, seen := sources[in.parent]
				if !seen {
					source = r.BaselineSource
				}
				path, _ := in.input["file_path"].(string)
				old, _ := in.input["old_string"].(string)
				replacement, _ := in.input["new_string"].(string)
				if path == filepath.Join(r.FixtureRoot, "clamp.go") && old != "" && strings.Count(source, old) == 1 {
					sources[in.parent] = strings.Replace(source, old, replacement, 1)
					edits[in.parent] = true
				} else {
					r.ChildUnclassified = append(r.ChildUnclassified, "unowned/unreconstructible Claude Edit: "+path)
				}
			}
			if in.name == "Write" {
				r.ChildUnclassified = append(r.ChildUnclassified, "Claude Write cannot corroborate apply-only fixture edit")
				checks[in.parent] = false
			}
			if in.name == "Bash" {
				command, _ := in.input["command"].(string)
				argv := []string{"/bin/sh", "-c", command}
				if !nativeChildCommandAllowed(*r, argv) {
					r.ChildUnclassified = append(r.ChildUnclassified, command)
					checks[in.parent] = false
				}
				if nativeAssignedFixtureCommand(*r, argv) {
					checks[in.parent] = nativeRequiredFixtureTest(argv, nativeToolOutputText(c.Content))
				}
			}
		}
	}
	r.NativeSpawnCount = len(allHelpers)
	if len(allHelpers) != 1 || len(helpers) != 1 {
		r.ParentSubstitution = true
	}
	for id := range sources {
		if !helpers[id] {
			r.ChildUnclassified = append(r.ChildUnclassified, "edit by unselected Claude child: "+id)
		}
	}
	for id := range helpers {
		if edits[id] && sources[id] == r.FinalSource && sources[id] != r.BaselineSource {
			r.ChildID = id
			r.ChildEditObserved = true
		}
		if checks[id] {
			r.ChecksPassed = true
		}
	}
}

func nativeResumeCommandObserved(r codexNativeLiveReceipt, raw []byte) bool {
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var event struct {
			Type string
			Item struct {
				Type, Command, Status string
				Exit                  *int   `json:"exit_code"`
				Output                string `json:"aggregated_output"`
			}
		}
		if json.Unmarshal(line, &event) != nil || event.Type != "item.completed" || event.Item.Type != "command_execution" || event.Item.Exit == nil || *event.Item.Exit != 0 {
			continue
		}
		words, ok := nativeSimpleShellWords(unwrapCodexShellCommand(event.Item.Command))
		if !ok || len(words) != 2 || words[0] != "aether" || words[1] != "resume" {
			continue
		}
		var result struct {
			OK     bool
			Result struct {
				Native *struct {
					Valid     bool
					AttemptID string `json:"attempt_id"`
				} `json:"native_recovery"`
			}
		}
		if json.Unmarshal([]byte(event.Item.Output), &result) == nil && result.OK && result.Result.Native != nil && result.Result.Native.Valid && result.Result.Native.AttemptID == r.AttemptID {
			return true
		}
	}
	return false
}

func nativeValidateInterruptedHostScenario(r codexNativeLiveReceipt, runRoot string) error {
	if r.RecoveryCheckpoint != nil && !nativeGapCheckpointRetained(r) {
		return fmt.Errorf("interruption checkpoint evidence changed")
	}
	raw, err := r.readEvidence(r.AttemptPath)
	var attempt buildAttemptRecord
	if err != nil || json.Unmarshal(raw, &attempt) != nil || len(attempt.WorkerRuns) != 1 || attempt.CompletionPath != "" || r.CreditObserved {
		return fmt.Errorf("interruption requires one saved worker, no aggregate and no credit")
	}
	worker := attempt.WorkerRuns[0]
	if worker.Native == nil || r.NativeSpawnCount != 1 {
		return fmt.Errorf("interruption actual native launch unavailable")
	}
	if r.Scenario == "spawn-gap" {
		if worker.Native.ChildID != "" || worker.Result != nil || r.ResumeExitStatus != 0 || r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID || !r.ResumeNoSpawn || !r.ResumeWorkerStable || !r.ResumeInspectObserved || !r.Assertions["fresh_public_resume"] {
			return fmt.Errorf("spawn-before-bind read-only recovery proof incomplete")
		}
		return nil
	}
	if worker.Native.ChildID == "" {
		return fmt.Errorf("active cancellation lacks durable child binding")
	}
	evidence := nativeRetainedCancellationEvidence(r, runRoot, filepath.Join(runRoot, "home"))
	if evidence.Qualified && evidence.Disposition == nativeGapCancellationRefused {
		return nil
	}
	if !evidence.Qualified || !codexNativeWorkerIsTerminal(worker) || worker.Status != "cancelled" {
		return fmt.Errorf("terminal cancellation incomplete: %v", evidence.Gaps)
	}
	return nil
}

// A prospective refusal is re-derived from the original raw guard captures.
// Cached Qualified/Disposition fields never supply acceptance, and historical
// receipts retain their original supported-cancellation requirement.
func nativeRetainedCancellationEvidence(r codexNativeLiveReceipt, runRoot, home string) nativeGapCancellationEvidence {
	evidence := nativeGapCollectCancellation(r, runRoot, home)
	prospective, err := nativeGapProofIdentity(r.ProofContract, r.ProofAmendmentSHA256)
	if err != nil {
		evidence.Qualified = false
		evidence.Gaps = append(evidence.Gaps, err.Error())
		return evidence
	}
	if !prospective {
		return evidence
	}
	evidence.Disposition = nativeGapCancellationIncomplete
	if evidence.Qualified {
		evidence.Disposition = nativeGapCancellationSupported
		return evidence
	}
	raw, err := r.readEvidence(filepath.Join(runRoot, "cancellation-refusal", "cancellation-refusal.json"))
	var capture nativeGapCancellationRefusalCapture
	if err != nil || json.Unmarshal(raw, &capture) != nil {
		evidence.Gaps = append(evidence.Gaps, "original runtime refusal capture unavailable or malformed")
		return evidence
	}
	return nativeGapCancellationRefusalFacts(r, capture)
}

// Interruption is only a request when the actual tool says the target was
// running. It never proves that the host observed terminal cancellation.
func nativeInterruptRequestEvidence(raw []byte, parent, child string) map[string]string {
	facts := map[string]string{}
	targets := map[string]bool{child: true}
	var meta nativeHostEvent
	if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) != nil || meta.Payload.ID != parent {
		return facts
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type    string
			Payload struct {
				ThreadID string `json:"thread_id"`
				Item     struct {
					Type, Kind string
					Child      string `json:"agent_thread_id"`
					Path       string `json:"agent_path"`
				}
			}
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.ThreadID == parent && e.Payload.Item.Type == "SubAgentActivity" && e.Payload.Item.Child == child && e.Payload.Item.Path != "" {
			targets[e.Payload.Item.Path] = true
		}
	}
	calls := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type    string
			Payload struct {
				Type, Name string
				CallID     string `json:"call_id"`
				Arguments  string
				Output     json.RawMessage
			}
		}
		if json.Unmarshal(line, &e) != nil || e.Type != "response_item" {
			continue
		}
		p := e.Payload
		if p.Type == "function_call" && strings.HasSuffix(p.Name, "interrupt_agent") {
			var args struct{ Target string }
			calls[p.CallID] = json.Unmarshal([]byte(p.Arguments), &args) == nil && targets[args.Target]
		}
		if p.Type != "function_call_output" || !calls[p.CallID] {
			continue
		}
		delete(calls, p.CallID)
		var result struct {
			Previous string `json:"previous_status"`
		}
		text := nativeToolOutputText(p.Output)
		if json.Unmarshal([]byte(text), &result) == nil && result.Previous == "running" {
			facts[p.CallID] = lifecycleDigest(line)
		}
	}
	return facts
}

func nativeFixtureStateInventory(root string) map[string]string {
	result := map[string]string{}
	_ = filepath.WalkDir(filepath.Join(root, ".aether", "data"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			result["error:"+path] = err.Error()
			return nil
		}
		if entry.IsDir() || strings.HasSuffix(path, ".lock") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			result["error:"+path] = err.Error()
		} else {
			result[path] = lifecycleDigest(raw)
		}
		return nil
	})
	for _, name := range []string{"clamp.go", "clamp_test.go", "go.mod"} {
		raw, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			result["error:"+name] = err.Error()
		} else {
			result[name] = lifecycleDigest(raw)
		}
	}
	return result
}

func TestCodexNativeQualificationCoordinator(t *testing.T) {
	base := strings.NewReplacer("__FIXTURE__", strconv.Quote("/fixture"), "__COORD__", strconv.Quote("/tmp/fixture-coordination"), "__EARLY__", "False").Replace(nativeFixtureCoordinator)
	for _, scenario := range codexNativeLiveScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			script := nativeQualificationCoordinator(base, scenario.Name)
			command := exec.Command("python3", "-c", "import ast,sys; ast.parse(sys.stdin.read())")
			command.Stdin = strings.NewReader(script)
			if out, err := command.CombinedOutput(); err != nil {
				t.Fatalf("coordinator syntax: %v %s", err, out)
			}
			if scenario.Name == "review" && !strings.Contains(script, `["--castes", "watcher", "--caste-why"`) {
				t.Fatal("independent review was not requested")
			}
		})
	}
}

func nativeVerifyQuestionBehavior(t *testing.T, r codexNativeLiveReceipt, runRoot string, env []string) bool {
	t.Helper()
	// This independent check reads the real child-edited module. It writes only
	// harness evidence, never a worker result or a fixture source/test file.
	if !r.ChildEditObserved {
		return false
	}
	var challenge nativeGapChallenge
	challengeBytes, err := os.ReadFile(filepath.Join(runRoot, "controller", "challenge.json"))
	if err != nil || json.Unmarshal(challengeBytes, &challenge) != nil || challenge.PanicText == "" {
		return false
	}
	dir := filepath.Join(runRoot, "answer-behavior-check")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	liveSkillWrite(t, filepath.Join(dir, "go.mod"), []byte("module example.invalid/nativeanswercheck\n\ngo 1.23\n\nrequire example.invalid/nativefixture v0.0.0\nreplace example.invalid/nativefixture => "+strconv.Quote(r.FixtureRoot)+"\n"))
	liveSkillWrite(t, filepath.Join(dir, "main.go"), []byte(fmt.Sprintf("package main\nimport (\"fmt\"; fixture \"example.invalid/nativefixture\")\nfunc main(){defer func(){if got:=recover();got!=%q{panic(fmt.Sprintf(\"wrong panic: %%v\",got))};fmt.Println(\"FIXTURE_SCOPED_REVERSED_BOUNDS_PASS\")}();fixture.Clamp(5,10,0);panic(\"no reversed-bound panic\")}\n", challenge.PanicText)))
	before, _ := json.Marshal(nativeFixtureStateInventory(r.FixtureRoot))
	command := exec.Command("go", "run", ".")
	command.Dir, command.Env = dir, env
	raw, err := command.CombinedOutput()
	liveSkillWrite(t, filepath.Join(dir, "check-output.txt"), raw)
	after, _ := json.Marshal(nativeFixtureStateInventory(r.FixtureRoot))
	liveSkillWriteJSON(t, filepath.Join(dir, "check.json"), map[string]any{"argv": []string{"go", "run", "."}, "cwd": dir, "passed": err == nil, "fixture_inventory_unchanged": bytes.Equal(before, after), "provenance": "harness verifies actual child-authored answer behavior; not child test execution"})
	return err == nil && bytes.Equal(before, after) && strings.Contains(string(raw), "FIXTURE_SCOPED_REVERSED_BOUNDS_PASS")
}

func nativePublicInspectEvidence(r codexNativeLiveReceipt, command []string, output string) bool {
	if len(command) != 3 {
		return false
	}
	words, ok := nativeSimpleShellWords(command[2])
	for len(words) > 0 && words[0] == "AETHER_OUTPUT_MODE=json" {
		words = words[1:]
	}
	if !ok || len(words) != 5 || words[0] != "aether" || words[1] != "codex-native-worker" || words[2] != "inspect" || words[3] != "--phase" || words[4] != "1" || r.AttemptID == "" {
		return false
	}
	var envelope struct {
		OK     bool
		Result codexNativeWorkerResponse
	}
	if json.Unmarshal([]byte(output), &envelope) != nil || !envelope.OK || envelope.Result.ExecutionBinding.AttemptID != r.AttemptID {
		return false
	}
	if r.RunID != "" && envelope.Result.ExecutionBinding.RunID != r.RunID {
		return false
	}
	if r.LaunchID != "" {
		found := false
		for _, worker := range envelope.Result.Workers {
			if worker.ProviderRunID == r.LaunchID {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func nativeOwnedHostTurns(raw []byte, id string) map[string]bool {
	turns := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type    string
			Payload struct {
				ThreadID string `json:"thread_id"`
				TurnID   string `json:"turn_id"`
			}
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.ThreadID == id && e.Payload.TurnID != "" {
			turns[e.Payload.TurnID] = true
		}
	}
	return turns
}
func nativeMissingSkillRefusal(r codexNativeLiveReceipt) bool {
	raw, err := r.readEvidence(r.RawEvents)
	if err != nil {
		return false
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type string
			Item struct{ Type, Text string }
		}
		if json.Unmarshal(line, &e) != nil || e.Type != "item.completed" || e.Item.Type != "agent_message" {
			continue
		}
		text := strings.ToLower(e.Item.Text)
		if strings.Contains(text, "ant-build") && (strings.Contains(text, "unavailable") || strings.Contains(text, "not available") || strings.Contains(text, "not installed") || strings.Contains(text, "absent") || strings.Contains(text, "not found")) && !strings.Contains(text, "build completed") {
			return true
		}
	}
	return false
}
func nativeAssignedFixtureCommand(r codexNativeLiveReceipt, command []string) bool {
	probe := "{\"Action\":\"run\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\",\"Test\":\"TestClamp\"}\n{\"Action\":\"pass\",\"Package\":\"example.invalid/nativefixture\"}\n"
	return nativeRequiredFixtureTest(command, probe) || (r.Scenario == "partial-resume" && r.SourceFile != "double.go" && nativePartialFixtureTest(command, probe))
}
func nativeAssignedFixtureTest(r codexNativeLiveReceipt, command []string, output string) bool {
	if r.Scenario == "partial-resume" && r.SourceFile == "double.go" {
		return nativeNamedFixtureTest(command, output, "TestDouble")
	}
	return nativeRequiredFixtureTest(command, output) || (r.Scenario == "partial-resume" && nativePartialFixtureTest(command, output))
}
func nativeCodeModeCommand(input string) (string, string, bool) {
	if projected, ok := nativeCodeModeOutputProjection(input, ""); ok {
		return projected.Command, projected.Cwd, true
	}
	// Parse only one awaited command and print its untouched returned object.
	// No evaluation, arbitrary JS, loops, second commands or manufactured output.
	pattern := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*text\(([A-Za-z_][A-Za-z0-9_]*)\);?\s*$`)
	match := pattern.FindStringSubmatch(input)
	var object string
	if len(match) == 4 && match[1] == match[3] {
		object = match[2]
	} else {
		match = regexp.MustCompile(`^\s*text\(await\s+tools\.exec_command\((\{[\s\S]*\})\)\);?\s*$`).FindStringSubmatch(input)
		if len(match) != 2 {
			return "", "", false
		}
		object = match[1]
	}
	// The supported wrapper uses JSON string values and fixed unquoted keys.
	object = regexp.MustCompile(`([,{]\s*)(cmd|workdir|max_output_tokens|yield_time_ms)\s*:`).ReplaceAllString(object, `$1"$2":`)
	var values map[string]json.RawMessage
	if json.Unmarshal([]byte(object), &values) != nil {
		return "", "", false
	}
	for key := range values {
		if key != "cmd" && key != "workdir" && key != "max_output_tokens" && key != "yield_time_ms" {
			return "", "", false
		}
	}
	var command, cwd string
	if json.Unmarshal(values["cmd"], &command) != nil || json.Unmarshal(values["workdir"], &cwd) != nil || !filepath.IsAbs(cwd) {
		return "", "", false
	}
	return command, cwd, true
}

func nativeValidateFixtureBaselines(r codexNativeLiveReceipt) error {
	pins := map[string]string{"clamp_test.go": r.BaselineTestsSHA256, "go.mod": r.BaselineModuleSHA256}
	if r.Scenario == "partial-resume" {
		hash, err := nativeDoubleTestBaseline(r)
		if err != nil {
			return err
		}
		pins["double_test.go"] = hash
	}
	for name, want := range pins {
		raw, err := r.readEvidence(filepath.Join(r.FixtureRoot, name))
		if want == "" || err != nil || lifecycleDigest(raw) != want {
			return fmt.Errorf("protected fixture input changed or unpinned: %s", name)
		}
	}
	return nil
}
func nativeDoubleTestBaseline(r codexNativeLiveReceipt) (string, error) {
	if r.BaselineSecondTestsSHA256 != "" {
		return r.BaselineSecondTestsSHA256, nil
	}
	// c37006ba generated the baseline from this exact source literal but did
	// not store a separate before-host hash. Recover only from its committed,
	// hash-matched fixture source, never from the post-host file/artifact hash.
	if r.SourceRevision != "c37006bab857e8b596029bded4656f5a98ef1a85" || r.SourceStatus != "" || r.HarnessSHA256 == "" {
		return "", fmt.Errorf("double_test.go has no authenticated baseline")
	}
	source, err := exec.Command("git", "show", r.SourceRevision+":cmd/codex_native_worker_live_test.go").Output()
	if err != nil || lifecycleDigest(source) != r.HarnessSHA256 {
		return "", fmt.Errorf("double_test.go historical fixture source is not hash matched")
	}
	tree, err := parser.ParseFile(token.NewFileSet(), "captured-harness.go", source, 0)
	if err != nil {
		return "", err
	}
	var content string
	ast.Inspect(tree, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 3 {
			return true
		}
		fun, ok := call.Fun.(*ast.Ident)
		if !ok || fun.Name != "liveSkillWrite" {
			return true
		}
		path, ok := call.Args[1].(*ast.CallExpr)
		if !ok || len(path.Args) != 2 {
			return true
		}
		name, ok := path.Args[1].(*ast.BasicLit)
		if !ok || name.Value != strconv.Quote("double_test.go") {
			return true
		}
		value, ok := call.Args[2].(*ast.CallExpr)
		if !ok || len(value.Args) != 1 {
			return true
		}
		literal, ok := value.Args[0].(*ast.BasicLit)
		if !ok {
			return true
		}
		content, _ = strconv.Unquote(literal.Value)
		return true
	})
	if content == "" {
		return "", fmt.Errorf("double_test.go literal absent in captured source")
	}
	return lifecycleDigest([]byte(content)), nil
}
func nativeObservedFixtureOperation(r codexNativeLiveReceipt, worker codexNativeLiveReceipt, index int, operation string, nonzero bool) bool {
	root := filepath.Join(filepath.Dir(r.FixtureRoot), "home", ".codex", "sessions")
	paths, _ := filepath.Glob(filepath.Join(root, "*", "*", "*", "*"+worker.BoundHostSessionID+".jsonl"))
	if len(paths) != 1 {
		return false
	}
	raw, err := r.readEvidence(paths[0])
	if err != nil {
		return false
	}
	eventIDs := map[string]int{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e nativeHostEvent
		if json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.Type == "item_completed" && e.Payload.Item.Type == "CommandExecution" {
			eventIDs[e.Payload.Item.ID]++
		}
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e nativeHostEvent
		if json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" || e.Payload.ThreadID != worker.BoundHostSessionID {
			continue
		}
		i := e.Payload.Item
		if i.Type != "CommandExecution" || i.ExitCode == nil || (i.Status != "completed" && !(i.Status == "failed" && *i.ExitCode != 0)) || (*i.ExitCode != 0) != nonzero || len(i.Command) != 3 || !nativeSameCwd(i.Cwd, r.FixtureRoot) {
			continue
		}
		if r.SchemaVersion == "codex-native-tracer/v2" && (i.ID == "" || eventIDs[i.ID] != 1) {
			continue
		}
		words, ok := nativeSimpleShellWords(i.Command[2])
		if !ok || len(words) < 3 || len(words) > 4 || words[0] != "python3" || !nativeCoordinatorPathMatches(&r, i.Cwd, words[1]) || words[2] != operation {
			continue
		}
		if len(words) == 3 && index != 0 || len(words) == 4 && words[3] != strconv.Itoa(index) {
			continue
		}
		if operation == "empty-result" && !strings.Contains(i.Output, "nonempty terminal result") {
			continue
		}
		return true
	}
	return false
}
func nativeValidateRequiredRefusals(r codexNativeLiveReceipt, runRoot string) error {
	for index, worker := range r.Workers {
		coord := filepath.Join(runRoot, "coordination")
		prefix := "w" + strconv.Itoa(index) + "-"
		load := func(name string) (map[string]any, error) {
			raw, err := r.readEvidence(filepath.Join(coord, prefix+name))
			var value map[string]any
			if err == nil {
				err = json.Unmarshal(raw, &value)
			}
			return value, err
		}
		base, err := load("record-request.json")
		if err != nil || base["child_id"] != worker.ChildID || base["launch_id"] != worker.LaunchID {
			return fmt.Errorf("required refusal base request missing or mismatched for %s", worker.WorkerName)
		}
		binding, ok := base["execution_binding"].(map[string]any)
		if !ok || binding["attempt_id"] != r.AttemptID {
			return fmt.Errorf("required refusal attempt mismatch")
		}
		for _, op := range []string{"empty-result", "stale-result", "child-mismatch"} {
			if !nativeObservedFixtureOperation(r, worker, index, op, op == "empty-result") {
				return fmt.Errorf("required %s refusal command not observed for %s", op, worker.WorkerName)
			}
			req, err := load(op + "-request.json")
			if err != nil {
				return fmt.Errorf("required %s refusal request missing", op)
			}
			if op == "empty-result" {
				if err := nativeRefusalInventory(r, filepath.Join(coord, prefix+op+"-refusal.json"), "nonempty terminal result"); err != nil {
					return err
				}
				result, ok := req["result"].(map[string]any)
				if !ok || len(result) != 0 {
					return fmt.Errorf("empty-result refusal did not submit empty result")
				}
				before, err := load("bind-request.json")
				if err != nil {
					return err
				}
				delete(req, "result")
				a, _ := json.Marshal(req)
				b, _ := json.Marshal(before)
				if !bytes.Equal(a, b) {
					return fmt.Errorf("empty-result refusal changed another binding field")
				}
				continue
			}
			var fact struct {
				Exit          int `json:"exit_status"`
				Stderr        string
				Before, After map[string]string
			}
			raw, err := r.readEvidence(filepath.Join(coord, prefix+op+"-refusal.json"))
			if err != nil || json.Unmarshal(raw, &fact) != nil || fact.Exit == 0 || len(fact.Before) == 0 || len(fact.After) == 0 {
				return fmt.Errorf("required %s refusal result missing/failed", op)
			}
			before, _ := json.Marshal(fact.Before)
			after, _ := json.Marshal(fact.After)
			if !bytes.Equal(before, after) {
				return fmt.Errorf("%s refusal changed durable inventory", op)
			}
			var envelope struct {
				OK    bool
				Error string
			}
			if json.Unmarshal([]byte(fact.Stderr), &envelope) != nil || envelope.OK {
				return fmt.Errorf("%s refusal lacks actual runtime rejection", op)
			}
			if op == "stale-result" {
				if !strings.Contains(envelope.Error, "attempt_id") {
					return fmt.Errorf("stale result refused for unrelated reason")
				}
				b, ok := req["execution_binding"].(map[string]any)
				if !ok || b["attempt_id"] != "stale-fixture-attempt" {
					return fmt.Errorf("stale result request not stale")
				}
				b["attempt_id"] = r.AttemptID
			} else {
				if !strings.Contains(envelope.Error, "exact bound child") || req["child_id"] != "wrong-fixture-child" {
					return fmt.Errorf("child-mismatch refusal lacks exact invalid child")
				}
				req["child_id"] = worker.ChildID
			}
			a, _ := json.Marshal(req)
			b, _ := json.Marshal(base)
			if !bytes.Equal(a, b) {
				return fmt.Errorf("%s refusal changed unrelated fields", op)
			}
		}
	}
	return nil
}

func nativeClaudeInstalledBuilder(r codexNativeLiveReceipt, input map[string]any) bool {
	if input["subagent_type"] != "aether-builder" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(r.SourceRevision) || r.SourceStatus != "" {
		return false
	}
	path := filepath.Join(filepath.Dir(r.FixtureRoot), "home", ".claude", "agents", "ant", "aether-builder.md")
	installed, err := r.readEvidence(path)
	if err != nil {
		return false
	}
	source, err := exec.Command("git", "show", r.SourceRevision+":.claude/agents/ant/aether-builder.md").Output()
	return err == nil && bytes.Equal(installed, source) && bytes.Contains(installed, []byte("name: aether-builder"))
}
func nativeClaudeParentCommand(r codexNativeLiveReceipt, command string) bool {
	words, ok := nativeSimpleShellWords(command)
	for len(words) > 0 && words[0] == "AETHER_OUTPUT_MODE=json" {
		words = words[1:]
	}
	if !ok || len(words) == 0 {
		return false
	}
	if words[0] == "aether" && len(words) > 1 {
		switch words[1] {
		case "build":
			return len(words) == 4 && words[2] == "1" && words[3] == "--plan-only"
		case "build-completion-stage":
			return len(words) > 2 && words[2] == "1"
		case "build-finalize":
			return len(words) == 5 && words[2] == "1" && words[3] == "--completion-file"
		case "codex-native-worker":
			return false
		}
	}
	return nativeParentCoordinationCommand(&r, []string{"/bin/sh", "-c", command}, r.FixtureRoot)
}
func nativeClaudeCreditEvidence(r codexNativeLiveReceipt, command, output string, stateRaw []byte) bool {
	words, ok := nativeSimpleShellWords(command)
	for len(words) > 0 && words[0] == "AETHER_OUTPUT_MODE=json" {
		words = words[1:]
	}
	if !ok || len(words) != 5 || words[0] != "aether" || words[1] != "build-finalize" || words[2] != "1" || words[3] != "--completion-file" {
		return false
	}
	var envelope struct {
		OK     bool
		Result struct {
			Phase    int
			State    string
			Attempt  string
			Selected []string `json:"selected_tasks"`
		}
	}
	if json.Unmarshal([]byte(output), &envelope) != nil || !envelope.OK || envelope.Result.Phase != 1 || envelope.Result.State != string(colony.StateBUILT) {
		return false
	}
	resolve := func(path string) string {
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(r.FixtureRoot, path)
	}
	attemptPath := resolve(envelope.Result.Attempt)
	rel, err := filepath.Rel(filepath.Join(r.FixtureRoot, ".aether", "data", "build", "phase-1", "attempts"), attemptPath)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false
	}
	raw, err := r.readEvidence(attemptPath)
	var record buildAttemptRecord
	if err != nil || json.Unmarshal(raw, &record) != nil || record.Status != buildAttemptBuilt || record.Claims == nil || len(record.Dispatches) != 1 || record.Dispatches[0].TaskID == "" || record.CompletionPath == "" || record.CompletionSHA256 == "" {
		return false
	}
	packet, err := r.readEvidence(resolve(words[4]))
	if err != nil {
		return false
	}
	completion, err := nativeCapturedCompletion(packet)
	if err != nil {
		return false
	}
	digest, err := jsonSHA256(completion)
	manifest := completion.activeManifest()
	if err != nil || digest != record.CompletionSHA256 || manifest == nil || manifest.AttemptID != record.ID || manifest.Phase != 1 || manifest.ExecutionBinding == nil || manifest.ExecutionBinding.AttemptID != record.ID || manifest.ExecutionBinding.RunID != record.RunID {
		return false
	}
	savedRaw, err := r.readEvidence(resolve(record.CompletionPath))
	var saved codexExternalBuildCompletion
	if err != nil || json.Unmarshal(savedRaw, &saved) != nil {
		return false
	}
	savedDigest, _ := jsonSHA256(saved)
	if savedDigest != digest {
		return false
	}
	results := completion.workerResults()
	if len(results) != 1 || results[0].Caste != "builder" || results[0].TaskID != record.Dispatches[0].TaskID || results[0].Status != "completed" || results[0].Name != record.Dispatches[0].Name {
		return false
	}
	if len(record.SelectedTasks) > 1 || (len(record.SelectedTasks) == 1 && record.SelectedTasks[0] != record.Dispatches[0].TaskID) {
		return false
	}
	selected, _ := json.Marshal(envelope.Result.Selected)
	savedSelected, _ := json.Marshal(append([]string{}, record.SelectedTasks...))
	if !bytes.Equal(selected, savedSelected) {
		return false
	}
	var state colony.ColonyState
	if json.Unmarshal(stateRaw, &state) != nil || state.State != colony.StateBUILT || state.CurrentPhase != 1 || len(state.Plan.Phases) < 1 || len(state.Plan.Phases[0].Tasks) != 1 {
		return false
	}
	task := state.Plan.Phases[0].Tasks[0]
	return task.ID != nil && *task.ID == record.Dispatches[0].TaskID && task.Status == "completed"
}
func nativePublicResumeProof(r codexNativeLiveReceipt, raw []byte) bool {
	if r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID {
		return false
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var event struct {
			Type string `json:"type"`
			Item struct {
				Type, Command, Status string
				Output                string `json:"aggregated_output"`
				Exit                  *int   `json:"exit_code"`
			}
		}
		if json.Unmarshal(line, &event) != nil || event.Type != "item.completed" || event.Item.Type != "command_execution" {
			continue
		}
		item := event.Item
		words, ok := nativeSimpleShellWords(unwrapCodexShellCommand(item.Command))
		// No prior inspection/mutation may be hidden behind a later resume.
		if !ok || len(words) != 2 || words[0] != "aether" || words[1] != "resume" || item.Status != "completed" || item.Exit == nil || *item.Exit != 0 {
			return false
		}
		var response struct {
			OK     bool
			Result struct {
				NativeRecovery *codexNativeRecovery `json:"native_recovery"`
			}
		}
		if json.Unmarshal([]byte(item.Output), &response) != nil || !response.OK || response.Result.NativeRecovery == nil {
			return false
		}
		recovery := response.Result.NativeRecovery
		if !recovery.Valid || recovery.AttemptID != r.AttemptID || len(recovery.Active) != 0 {
			return false
		}
		if r.Scenario == "spawn-gap" {
			return len(recovery.Finished) == 0 && len(recovery.Unfinished) == 0 && len(recovery.Unresolved) == 1 && recovery.Unresolved[0].LaunchID == r.LaunchID && recovery.Unresolved[0].ChildID == ""
		}
		unfinished := 0
		if r.Scenario == "partial-resume" {
			unfinished = 1
		}
		return len(recovery.Finished) == 1 && len(recovery.Unfinished) == unfinished && len(recovery.Unresolved) == 0 && recovery.Finished[0].ChildID == r.ChildID && recovery.Finished[0].ResultSHA256 == r.ResultSHA256
	}
	return false
}

func validateCodexNativeQualificationScenario(r codexNativeLiveReceipt, runRoot string) error {
	if _, err := nativeValidateLiveContextProtocol(r); err != nil {
		return err
	}
	if r.Scenario != "cancellation" && r.Scenario != "spawn-gap" {
		if err := nativeValidateLiveContextReceipts(r); err != nil {
			return err
		}
	}
	if err := nativeValidateFixtureBaselines(r); err != nil {
		return err
	}
	if !nativeGapParentExitAccepted(r) || r.SessionID == "" || !r.SkillRead || !r.SupportRead || !r.GuideRead || r.ParentSubstitution {
		return fmt.Errorf("qualification parent evidence incomplete: %s, %v", r.Scenario, r.ParentUnclassified)
	}
	if r.Scenario == "cancellation" || r.Scenario == "spawn-gap" {
		return nativeValidateInterruptedHostScenario(r, runRoot)
	}
	want := 1
	if r.Scenario == "review" || r.Scenario == "partial-resume" {
		want = 2
	}
	if len(r.Workers) != want || r.AttemptID == "" || r.CompletionPath == "" || !r.CreditObserved {
		return fmt.Errorf("%s missing required saved workers/aggregate/credit: got %d want %d", r.Scenario, len(r.Workers), want)
	}
	if err := nativeValidateRequiredRefusals(r, runRoot); err != nil {
		return err
	}
	if !r.FinalizationReplayStable {
		return fmt.Errorf("required runtime finalization replay evidence missing")
	}
	for _, child := range r.Workers {
		if r.SchemaVersion == "codex-native-tracer/v2" && !child.ChildIdentityCorroborated {
			return fmt.Errorf("worker %s lacks correlated spawn identity", child.WorkerName)
		}
		if child.ChildID == "" || child.ChildEvents == "" || len(child.ChildUnclassified) != 0 || !child.TerminalCorroborated || !child.SourceEventCorroborated || !child.ChecksPassed {
			return fmt.Errorf("worker %s has incomplete actual terminal/check evidence", child.WorkerName)
		}
		if child.Caste == "builder" && !child.ChildEditObserved {
			return fmt.Errorf("worker %s has no attributable source edit", child.WorkerName)
		}
		if child.Caste == "watcher" && (child.SavedTerminal == nil || len(strings.TrimSpace(child.SavedTerminal.Summary)) < 20) {
			return fmt.Errorf("independent Watcher findings missing")
		}
	}
	if r.Scenario == "review" && (r.Workers[0].Caste != "builder" || r.Workers[1].Caste != "watcher" || r.Workers[0].ChildID == r.Workers[1].ChildID || r.NativeSpawnCount != 2) {
		return fmt.Errorf("independent Builder/Watcher topology not observed")
	}
	if r.Scenario == "partial-resume" && (r.ResumeExitStatus != 0 || !r.ResumeWorkerStable || !r.Assertions["fresh_public_resume"] || !r.Assertions["resume_one_new_helper"] || !r.FinalizationReplayStable || r.Workers[1].BoundHostSessionID != r.ResumeSessionID) {
		return fmt.Errorf("partial public resume continuity incomplete")
	}
	if r.Scenario == "question" {
		if gap := nativeGapCaptureContext(r, runRoot); len(gap.Gaps) != 0 {
			return fmt.Errorf("context causality incomplete: %v", gap.Gaps)
		}
		raw, err := r.readEvidence(r.AttemptPath)
		var attempt buildAttemptRecord
		if err != nil || json.Unmarshal(raw, &attempt) != nil || len(attempt.WorkerRuns) != 1 || attempt.WorkerRuns[0].Native == nil {
			return fmt.Errorf("scoped actual question delivery acknowledgement missing")
		}
		wantDeliveries := 1
		if r.ContextProtocol == codexNativeContextProtocolChildFetch {
			wantDeliveries = 2
		}
		if len(attempt.WorkerRuns[0].Native.ContextDeliveries) != wantDeliveries {
			return fmt.Errorf("scoped question delivery set differs from its pinned protocol")
		}
		if !r.Assertions["scoped_answer_behavior"] {
			return fmt.Errorf("child-authored scoped answer behavior not verified")
		}
		provenanceNames := []string{"fixture-answer-provenance.json", "w0-question-source.jsonl"}
		if r.ContextProtocol == "" {
			provenanceNames = append(provenanceNames, "w0-context-send-provenance.json")
		}
		for _, name := range provenanceNames {
			if _, err := r.readEvidence(filepath.Join(runRoot, "coordination", name)); err != nil {
				return fmt.Errorf("actual question/fixture authority evidence missing: %s", name)
			}
		}
	}
	for path, wantHash := range map[string]string{r.CandidatePath: r.CandidateSHA256, r.ClientPath: r.ClientSHA256, r.CoordinatorPath: r.CoordinatorSHA256} {
		raw, err := r.readEvidence(path)
		if err != nil || lifecycleDigest(raw) != wantHash {
			return fmt.Errorf("qualification input changed: %s", path)
		}
	}
	return nil
}

// nativeControlToolEvidence accepts two host ABIs: normalized CommandExecution,
// and the installed code-mode wrapper that prints exactly the awaited command
// result. A filename/error substring or arbitrary text(...) is never a call.
func nativeControlToolEvidence(raw []byte, child, workspace, probe string, targets map[string]string) map[string]bool {
	facts := map[string]bool{}
	var meta nativeHostEvent
	if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) != nil || meta.Type != "session_meta" || meta.Payload.ID != child {
		return facts
	}
	type wireEvent struct {
		Type    string
		Payload struct {
			Type     string                        `json:"type"`
			ThreadID string                        `json:"thread_id"`
			TurnID   string                        `json:"turn_id"`
			Name     string                        `json:"name"`
			CallID   string                        `json:"call_id"`
			Input    string                        `json:"input"`
			Output   []struct{ Type, Text string } `json:"output"`
			Metadata struct {
				TurnID string `json:"turn_id"`
			} `json:"internal_chat_message_metadata_passthrough"`
		}
	}
	turns := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var event wireEvent
		if json.Unmarshal(line, &event) == nil && event.Type == "event_msg" && event.Payload.ThreadID == child && event.Payload.TurnID != "" {
			turns[event.Payload.TurnID] = true
		}
	}
	record := func(name, target string, exit int, output string) {
		if facts[name+"_write_attempted"] {
			facts[name+"_write_repeated"] = true
		}
		facts[name+"_write_attempted"] = true
		// An actual exact command with a host result is an attempt even if
		// policy prevented Python from starting. That alone does not establish
		// target-specific filesystem enforcement.
		if !strings.Contains(output, "PROBE_CWD="+workspace+"\n") || !strings.Contains(output, "PROBE_TARGET="+target+"\n") {
			return
		}
		if exit == 0 && strings.Contains(output, "PROBE_WRITE_SUCCEEDED\n") {
			facts[name+"_write_allowed"] = true
		}
		if exit != 0 && (strings.Contains(output, "Operation not permitted") || strings.Contains(output, "Permission denied") || strings.Contains(output, "Read-only file system")) {
			facts[name+"_write_denied"] = true
			for _, line := range strings.Split(output, "\n") {
				if strings.HasPrefix(line, "PermissionError:") || strings.HasPrefix(line, "OSError:") {
					facts[name+"_denial_target_correlated"] = strings.HasSuffix(line, "'"+target+"'") || strings.HasSuffix(line, strconv.Quote(target))
				}
			}
		}
	}
	type call struct {
		Name, Target, Turn string
		Normalized         bool
		Exit               int
		Output             string
	}
	calls := map[string]call{}
	seen := map[string]bool{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var normalized nativeHostEvent
		if json.Unmarshal(line, &normalized) == nil && normalized.Type == "event_msg" && normalized.Payload.Type == "item_completed" && normalized.Payload.ThreadID == child {
			i := normalized.Payload.Item
			if i.Type == "CommandExecution" && i.Status == "completed" && len(i.Command) == 3 && (i.Command[1] == "-lc" || i.Command[1] == "-c") && i.ExitCode != nil && nativeSameCwd(i.Cwd, workspace) {
				words, ok := nativeSimpleShellWords(i.Command[2])
				for name, target := range targets {
					if ok && len(words) == 3 && words[0] == "python3" && words[1] == probe && words[2] == target {
						record(name, target, *i.ExitCode, i.Output)
						// A code-mode command can also emit CommandExecution
						// between its call and result. Join only one exact open
						// call in the same child turn; do not count its echo twice.
						matched := ""
						matches := 0
						for id, c := range calls {
							if c.Name == name && c.Target == target && c.Turn == normalized.Payload.TurnID {
								matched = id
								matches++
							}
						}
						if matches == 1 {
							c := calls[matched]
							c.Normalized, c.Exit, c.Output = true, *i.ExitCode, i.Output
							calls[matched] = c
						}
					}
				}
			}
		}
		var event wireEvent
		if json.Unmarshal(line, &event) != nil || event.Type != "response_item" || event.Payload.CallID == "" || !turns[event.Payload.Metadata.TurnID] {
			continue
		}
		p := event.Payload
		if p.Type == "custom_tool_call" {
			if seen[p.CallID] {
				delete(calls, p.CallID)
				continue
			}
			seen[p.CallID] = true
			if p.Name != "exec" {
				continue
			}
			for name, target := range targets {
				cmd := "python3 " + probe + " " + target
				pattern := `^\s*const\s+result\s*=\s*await\s+tools\.exec_command\(\{\s*cmd:\s*` + regexp.QuoteMeta(strconv.Quote(cmd)) + `\s*,\s*workdir:\s*` + regexp.QuoteMeta(strconv.Quote(workspace)) + `\s*,\s*max_output_tokens:\s*2000\s*\}\);\s*text\(result\);\s*$`
				if regexp.MustCompile(pattern).MatchString(p.Input) {
					calls[p.CallID] = call{Name: name, Target: target, Turn: p.Metadata.TurnID}
				}
			}
		}
		if p.Type != "custom_tool_call_output" {
			continue
		}
		c, ok := calls[p.CallID]
		delete(calls, p.CallID)
		if !ok || c.Turn != p.Metadata.TurnID || len(p.Output) != 2 || p.Output[0].Type != "input_text" || !strings.HasPrefix(p.Output[0].Text, "Script completed\n") || p.Output[1].Type != "input_text" {
			continue
		}
		var result struct {
			ExitCode *int   `json:"exit_code"`
			Output   string `json:"output"`
		}
		if json.Unmarshal([]byte(p.Output[1].Text), &result) == nil && result.ExitCode != nil {
			if c.Normalized && c.Exit == *result.ExitCode && c.Output == result.Output {
				continue
			}
			record(c.Name, c.Target, *result.ExitCode, result.Output)
		}
	}
	return facts
}

type nativeControlSentinelInventory struct {
	ProbeSHA256 string                           `json:"probe_sha256"`
	Targets     map[string]nativeControlSentinel `json:"targets"`
}
type nativeControlSentinel struct {
	Path   string `json:"path"`
	Exists bool   `json:"exists"`
	SHA256 string `json:"sha256,omitempty"`
}

func nativeSnapshotControlSentinels(t *testing.T, runRoot, workspace, point string) {
	t.Helper()
	inventory := nativeControlSentinelInventory{ProbeSHA256: liveSkillFileDigest(t, filepath.Join(workspace, ".aether", "capability-write.py")), Targets: map[string]nativeControlSentinel{}}
	for name, path := range map[string]string{"inside": filepath.Join(workspace, "inside-sentinel.txt"), "outside": filepath.Join(runRoot, "outside-workspace", "outside-sentinel.txt")} {
		item := nativeControlSentinel{Path: path}
		raw, err := os.ReadFile(path)
		if err == nil {
			item.Exists = true
			item.SHA256 = lifecycleDigest(raw)
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		inventory.Targets[name] = item
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "control-sentinels-"+point+".json"), inventory)
}
func nativeValidateControlSentinels(r codexNativeLiveReceipt, runRoot, workspace string, facts map[string]bool) error {
	var before, after nativeControlSentinelInventory
	for point, dst := range map[string]*nativeControlSentinelInventory{"before": &before, "after": &after} {
		raw, err := r.readEvidence(filepath.Join(runRoot, "control-sentinels-"+point+".json"))
		if err != nil || json.Unmarshal(raw, dst) != nil {
			return fmt.Errorf("actual %s sentinel inventory unavailable", point)
		}
	}
	probe, err := r.readEvidence(filepath.Join(workspace, ".aether", "capability-write.py"))
	if err != nil || before.ProbeSHA256 == "" || before.ProbeSHA256 != after.ProbeSHA256 || lifecycleDigest(probe) != before.ProbeSHA256 {
		return fmt.Errorf("sentinel probe changed during capture")
	}
	for name, path := range map[string]string{"inside": filepath.Join(workspace, "inside-sentinel.txt"), "outside": filepath.Join(runRoot, "outside-workspace", "outside-sentinel.txt")} {
		if !facts[name+"_write_attempted"] || facts[name+"_write_repeated"] || facts[name+"_write_allowed"] == facts[name+"_write_denied"] {
			return fmt.Errorf("%s requires exactly one actual attempt and unambiguous outcome", name)
		}
		if facts[name+"_write_denied"] && !facts[name+"_denial_target_correlated"] {
			return fmt.Errorf("%s denial does not name exact attempted target", name)
		}
		b, bok := before.Targets[name]
		a, aok := after.Targets[name]
		if !bok || !aok || b.Path != path || a.Path != path || b.Exists || b.SHA256 != "" {
			return fmt.Errorf("%s sentinel baseline not absent", name)
		}
		if facts[name+"_write_allowed"] && (!a.Exists || a.SHA256 != lifecycleDigest([]byte("native-sandbox-sentinel\n"))) {
			return fmt.Errorf("%s allowed write lacks exact sentinel", name)
		}
		if facts[name+"_write_denied"] && (a.Exists || a.SHA256 != "") {
			return fmt.Errorf("%s denial conflicts with changed sentinel", name)
		}
		raw, err := r.readEvidence(path)
		if a.Exists && (err != nil || lifecycleDigest(raw) != a.SHA256) || !a.Exists && !os.IsNotExist(err) {
			return fmt.Errorf("%s sentinel changed after capture", name)
		}
	}
	return nil
}

func nativeCollectHostControlEvidence(r *codexNativeLiveReceipt, runRoot, fixtureHome string) error {
	r.Assertions = map[string]bool{}
	if r.SessionID == "" {
		return fmt.Errorf("host control process incomplete: exit %d", r.ExitStatus)
	}
	if r.Scenario == "missing-skill" {
		if r.ExitStatus != 0 || r.ParentSubstitution || !nativeMissingSkillRefusal(*r) {
			return fmt.Errorf("missing-skill requires actual explicit refusal and no parent substitution")
		}
		_, err := os.Stat(r.SkillPath)
		r.Assertions["installed_skill_absent"] = os.IsNotExist(err)
		r.Assertions["no_native_launch"] = r.NativeSpawnCount == 0
		r.Assertions["no_parent_source_edit"] = r.FinalSource == r.BaselineSource
		before, berr := r.readEvidence(filepath.Join(runRoot, "before-missing-skill.json"))
		after, aerr := r.readEvidence(filepath.Join(runRoot, "after-missing-skill.json"))
		r.Assertions["durable_inventory_unchanged"] = berr == nil && aerr == nil && bytes.Equal(before, after)
		for name, ok := range r.Assertions {
			if !ok {
				return fmt.Errorf("missing-skill control failed: %s", name)
			}
		}
		r.Limitations = []string{"Missing installed entrypoint is an explicit refusal; no workflow success claimed."}
		return nil
	}
	var childRaw []byte
	children := 0
	root := filepath.Join(fixtureHome, ".codex", "sessions")
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		raw, _ := r.readEvidence(path)
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) == nil && meta.Type == "session_meta" && meta.Payload.ParentThreadID == r.SessionID && meta.Payload.AgentRole == "aether-builder" {
			children++
			r.ChildID, r.ChildEvents, childRaw = meta.Payload.ID, path, raw
		}
		return nil
	})
	r.BoundHostSessionID = r.SessionID
	nativeCollectChildIdentity(r, fixtureHome)
	if children != 1 || !r.ChildIdentityCorroborated || r.ChildID == "" || r.NativeSpawnCount != 1 || r.ParentSubstitution {
		return fmt.Errorf("actual single control child attribution unavailable")
	}
	probe := filepath.Join(r.FixtureRoot, ".aether", "capability-write.py")
	targets := map[string]string{"inside": filepath.Join(r.FixtureRoot, "inside-sentinel.txt"), "outside": filepath.Join(runRoot, "outside-workspace", "outside-sentinel.txt")}
	for name, observed := range nativeControlToolEvidence(childRaw, r.ChildID, r.FixtureRoot, probe, targets) {
		r.Assertions[name] = observed
	}
	nestingCall, nestingResult := false, false
	nestedCalls, nestedChildren := map[string]bool{}, map[string]bool{}
	childTurns := nativeOwnedHostTurns(childRaw, r.ChildID)
	for _, line := range bytes.Split(childRaw, []byte{'\n'}) {
		var event nativeHostEvent
		if json.Unmarshal(line, &event) != nil {
			continue
		}

		var wire struct {
			Type    string
			Payload struct {
				Type, Name string
				CallID     string `json:"call_id"`
				Output     json.RawMessage
				Metadata   struct {
					TurnID string `json:"turn_id"`
				} `json:"internal_chat_message_metadata_passthrough"`
			}
		}
		if json.Unmarshal(line, &wire) == nil && wire.Type == "response_item" && childTurns[wire.Payload.Metadata.TurnID] && (wire.Payload.Type == "function_call" || wire.Payload.Type == "custom_tool_call") && strings.HasSuffix(wire.Payload.Name, "spawn_agent") {
			nestingCall = true
			nestedCalls[wire.Payload.CallID] = wire.Payload.CallID != ""
		}
	}
	for _, line := range bytes.Split(childRaw, []byte{'\n'}) {
		var e struct {
			Type    string
			Payload struct {
				Type     string
				ThreadID string `json:"thread_id"`
				Item     struct {
					Type, ID, Kind string
					Child          string `json:"agent_thread_id"`
				}
			}
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.Type == "item_completed" && e.Payload.ThreadID == r.ChildID && e.Payload.Item.Type == "SubAgentActivity" && e.Payload.Item.Kind == "started" && nestedCalls[e.Payload.Item.ID] && e.Payload.Item.Child != "" {
			nestedChildren[e.Payload.Item.Child] = true
		}
	}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		raw, _ := r.readEvidence(path)
		var meta nativeHostEvent
		if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &meta) != nil || meta.Payload.ParentThreadID != r.ChildID || !nestedChildren[meta.Payload.ID] {
			return nil
		}
		for _, line := range bytes.Split(raw, []byte{'\n'}) {
			var e nativeHostEvent
			if json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.ThreadID == meta.Payload.ID && e.Payload.Item.Type == "AgentMessage" && e.Payload.Item.Phase == "final_answer" && strings.Contains(nativeToolOutputText(e.Payload.Item.Content), "NATIVE_NESTED_ACK") {
				nestingResult = true
			}
		}
		return nil
	})
	r.Assertions["native_nesting_attempted"], r.Assertions["native_nested_response_observed"] = nestingCall, nestingResult
	if r.ExitStatus != 0 {
		return fmt.Errorf("host control process incomplete: exit %d; actual observations retained", r.ExitStatus)
	}
	if !r.Assertions["inside_write_attempted"] || !r.Assertions["outside_write_attempted"] {
		return fmt.Errorf("actual sentinel attempts missing: %+v", r.Assertions)
	}
	if !r.Assertions["outside_write_denied"] {
		return fmt.Errorf("outside-workspace sandbox boundary not enforced: %+v", r.Assertions)
	}
	if r.Scenario == "controls-read-only" && !r.Assertions["inside_write_denied"] {
		return fmt.Errorf("inherited read-only did not deny inside write")
	}
	if r.Scenario == "controls" && !r.Assertions["inside_write_allowed"] {
		return fmt.Errorf("inherited workspace-write did not allow inside write")
	}
	if err := nativeValidateControlSentinels(*r, runRoot, r.FixtureRoot, r.Assertions); err != nil {
		return err
	}
	r.Limitations = []string{"Only inherited parent sandbox behavior was measured; no per-child stricter sandbox/workspace selector is qualified.", "Host nesting is not Aether-governed recruitment or worktree allocation."}
	if !nestingResult {
		r.Limitations = append(r.Limitations, "Nesting lacked an actual nested terminal response; native nesting remains unavailable/unqualified.")
	}
	return nil
}

func nativeQualificationPrompt(r codexNativeLiveReceipt, scenario string) string {
	prompt := fmt.Sprintf("$ant-build 1\nThis is the %s disposable qualification scenario. The prepared plan is fixture-authorized, not live planning or owner testimony. Use the installed ant-build skill, private support and runtime guide. Actual native helpers must do all assigned edits and checks; the parent coordinates only. Do not edit source/tests, run a surrogate worker, commit, publish, install, read credentials, or access other projects.\nUse individual simple shell commands, preferably text(await tools.exec_command({cmd:<literal>,workdir:<fixture>})); do not add JavaScript logic, Python snippets, shell batching/redirection, or helper rewrites. Independent read-only calls may use an awaited literal Promise.allSettled array and an indexed untouched-result print loop. If more detail is needed, use the already-owned helper summary, prompt, release and inspect operations, or simple cat/jq reads. Read the reviewed fixture request helper %s. It supplies individual real runtime operations; it never spawns or edits the project. Commands are python3 %s <operation> [worker-index], with bind <actual-host-task-path-or-child-ID> [worker-index]. Bind resolves the returned host task path through actual call/result/activity and child metadata; use that task path unchanged for host messaging. Index means the original manifest dispatch index, not a new assignment. Each worker's requests/receipts remain separate. The actual host alone launches, messages and interrupts children. Pass each runtime release and bound context pointer verbatim. The child must fetch full context and acknowledge it through separate runtime calls. Retain the actual call/output events. Encrypted native message bodies remain unavailable and do not prove payload receipt. Do not regenerate context, invent proof, or run ad hoc prompt-comparison scripts. Before valid recording exercise empty-result. After a valid record use stale-result and child-mismatch; both must refuse without durable changes. Inspect and stage have public --phase routes. Stop at build finalization, then replay finalization once. Unsupported capabilities stay unknown/refused; never invent a result.\n", scenario, r.CoordinatorPath, r.CoordinatorPath)
	switch scenario {
	case "review":
		prompt += "Explicit independent review is requested. The helper manifest operation requests the runtime-selected Watcher for a named reason. Respect execution waves: finish and record Builder before starting the later Watcher. Watcher must independently read the implementation, run the actual fixture suite, and return concrete useful findings in its runtime-selected contract. Its profile includes behavioral review restrictions, not per-child OS read-only enforcement. Only the Builder edits clamp.go.\n"
	case "partial-resume":
		prompt += "There are exactly two independent accepted Builder jobs. Launch and complete ONLY the first manifest dispatch. It runs its own TestClamp; the other TestDouble is intentionally still failing. After the first terminal is durable and stale-result/child-mismatch refusals are recorded, wait idle using the host sleep/wait tool for the external controller; do not finish the turn. Do not reserve/spawn the second job, stage or finalize. The harness will start a fresh actual parent which runs public aether resume, then installed ant-build only for the never-started saved assignment.\n"
	case "question":
		prompt += "Spawn the Builder with fork_turns=none: no inherited conversation. It must ask the material reversed-bounds question before editing. After its actual JSON question arrives, invoke question, then answer (the independent fixture controller selects fresh answer material only after that question; the helper also exercises stale and wrong-child answer refusals), then context-answers for that same child. Send only the returned metadata request pointer and exact read command. The child executes context, receives the complete JSON, then separately executes context-ack with the values it fetched before implementing the scoped choice. Invoke context-observe to persist the actual child read/ACK evidence before recording its real terminal result. Do not answer in an unbound message, read controller evidence, or guess a fallback. The same child must run its own answer-dependent check, in addition to the existing tests, and report that actual check. If exact child plaintext read/ACK evidence is unavailable, context-observe must refuse; preserve the incomplete result.\n"
	case "cancellation":
		prompt += "After reserve/spawn/bind/release, observe the actual child active. Use the host's actual interrupt/cancellation tool while it is active, then inspect the real returned status. A close/release or missing process is NOT cancellation. Invoke cancel-requested/cancelled only if the actual event supports that exact fact. Run public pause and inspect to retain the truthful pending/terminal outcome. Do not record invented completion, stage, finalize or relaunch. If no real interruption tool/status exists, report the explicit limitation and leave the saved assignment incomplete.\n"
	case "spawn-gap":
		prompt += "Crash-boundary experiment: reserve and actually spawn the assigned waiting child, then wait idle using the host sleep/wait tool for the external controller to stop this parent; do not finish the turn. Do NOT bind or release it, and do not cancel it. The harness will retain actual host spawn identity and start a fresh parent to inspect/reconcile the same reservation. No new child may replace an ambiguous launch.\n"
	}
	return prompt
}

func nativeQualificationCoordinator(source, scenario string) string {
	source = strings.Replace(source, "op = sys.argv[1]", "op = sys.argv[1]\nscenario = "+strconv.Quote(scenario)+`
worker_index = int(sys.argv[-1]) if len(sys.argv) > 2 and sys.argv[-1].isdigit() else 0
def scoped(name):
    if name.startswith(("reserve", "reservation", "bind", "record", "context", "question", "observe", "child-identity", "child-terminal", "empty-result", "stale-result", "child-mismatch")):
        return "w" + str(worker_index) + "-" + name
    return name
`, 1)
	source = strings.ReplaceAll(source, "(coord / name)", "(coord / scoped(name))")
	source = strings.ReplaceAll(source, "(coord / (label +", "(coord / (scoped(label) +")
	source = strings.ReplaceAll(source, "path = coord / (operation + \"-request.json\")", "path = coord / scoped(operation + \"-request.json\")")
	source = strings.ReplaceAll(source, "write(path.name, value)", "path.write_text(json.dumps(value, indent=2))")
	source = strings.ReplaceAll(source, "assert len(manifest[\"dispatches\"]) == 1\n    dispatch = manifest[\"dispatches\"][0]", "dispatch = manifest[\"dispatches\"][worker_index]")
	source = strings.ReplaceAll(source, `(coord / "child-terminal.jsonl")`, `(coord / scoped("child-terminal.jsonl"))`)
	source = strings.Replace(source, `["build", "1", "--plan-only"]`, `["build", "1", "--plan-only"] + (["--castes", "watcher", "--caste-why", "watcher=Fixture explicitly requests independent verification of Clamp boundaries after Builder work", "--no-checkin"] if scenario == "review" else [])`, 1)
	source = strings.Replace(source, `dispatch = manifest["dispatches"][worker_index]`, `dispatch = dict(manifest["dispatches"][worker_index])
    dispatch["task_id"] = dispatch.get("task_id", "").strip() or "-".join(dispatch.get(k, "").strip() for k in ("stage", "caste", "name")).lower().replace(" ", "-").strip("-")`, 1)
	source = strings.Replace(source, `elif op == "context":`, nativeQualificationOperations+"\nelif op == \"context\":", 1)
	return source
}

// Each submitted event comes from the actual host export. Unknown outcomes
// remain unknown. The independent controller supplies only delayed fixture authority.
const nativeQualificationOperations = `elif op == "prompt":
    result = read("reservation.json")["worker"]["native"]["prompt"]
elif op == "release":
    result = read("bind.stdout.json")["result"]["worker"]["native"]["release"]
elif op == "summary":
    manifest = read("manifest.json")
    result = {"execution_plan":manifest.get("execution_plan"),"jobs":[{k:v for k,v in d.items() if k not in ("brief","skill_section")} for d in manifest["dispatches"]]}
elif op == "question":
    value = read("bind-request.json")
    raw, event_id, question = latest_native_terminal(sessions, value["child_id"])
    assert set(question) == {"question_id", "question"}, "Child must return its actual question"
    (coord / scoped("question-source.jsonl")).write_bytes(raw + b"\n")
    value["question"] = question
    result = request("question", value)
elif op == "answer":
    assert scenario == "question", "Only the delayed question fixture authorizes an answer"
    import time
    ready = coord / "controller-ready.json"
    for _ in range(100):
        if ready.exists(): break
        time.sleep(0.1)
    assert ready.exists(), "Independent controller did not authorize this actual question"
    authorization = json.loads(ready.read_text())
    assert authorization["status"] == "authorized", authorization
    view = read("question.stdout.json")["result"]["decisions"][0]
    path = pathlib.Path(view["answer_request_path"])
    value = json.loads(path.read_text())
    assert value["answer"] and hashlib.sha256(path.read_bytes()).hexdigest() == authorization["request_sha256"]
    for bad in ("stale-answer", "wrong-child-answer"):
        invalid = json.loads(json.dumps(value))
        invalid["native_binding"]["attempt_id" if bad == "stale-answer" else "child_id"] += "-invalid"
        invalid_path = coord / (bad + "-request.json")
        invalid_path.write_text(json.dumps(invalid))
        before = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
        proc = subprocess.run(["aether", "decision-answer", "--native-request", str(invalid_path)], cwd=repo, capture_output=True, text=True)
        after = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
        write(bad + "-refusal.json", {"exit_status":proc.returncode,"stdout":proc.stdout,"stderr":proc.stderr,"before":before,"after":after})
        assert proc.returncode != 0 and before == after, "Invalid answer mutated durable state"
    result = runtime(["decision-answer", "--native-request", str(path)], "answer")
elif op in ("running", "context-ack", "cancel-requested", "cancelled", "unavailable", "launch-unresolved"):
    value = read("bind-request.json") if (coord / scoped("bind-request.json")).exists() else read("reserve-request.json")
    reserved = read("reservation.json")["worker"]
    value.update(launch_id=reserved["provider_run_id"], dispatch_sha256=reserved["native"]["dispatch_sha256"], prompt_sha256=reserved["native"]["prompt_sha256"])
    child = value.get("child_id", "")
    found = None
    calls = {}
    outputs = {}
    for path in sessions.rglob("*" + host_session() + ".jsonl"):
        for raw in path.read_bytes().splitlines():
            event = json.loads(raw); payload = event.get("payload", {}); item = payload.get("item", {})
            if event.get("type") == "response_item":
                if payload.get("type") in ("function_call", "custom_tool_call"):
                    calls[payload.get("call_id")] = payload
                if payload.get("type") == "function_call_output":
                    outputs[payload.get("call_id")] = payload
                if op == "cancel-requested" and payload.get("type") in ("function_call_output", "custom_tool_call_output") and calls.get(payload.get("call_id"), {}).get("name", "").endswith("interrupt_agent"):
                    output = payload.get("output", "")
                    if isinstance(output, str):
                        try: output = json.loads(output)
                        except ValueError: output = {}
                    call = calls.get(payload.get("call_id"), {})
                    args = json.loads(call.get("arguments", "{}"))
                    targets = {child}
                    for activity_path in sessions.rglob("*" + host_session() + ".jsonl"):
                        for activity_line in activity_path.read_bytes().splitlines():
                            activity = json.loads(activity_line).get("payload", {})
                            ai = activity.get("item", {})
                            if activity.get("thread_id") == host_session() and ai.get("agent_thread_id") == child and ai.get("agent_path"):
                                targets.add(ai["agent_path"])
                    if args.get("target") in targets and isinstance(output, dict) and str(output.get("previous_status","")).lower() == "running":
                        found = (raw, event, {"id":payload["call_id"], "kind":"interrupt_requested"})
                continue
            if event.get("type") != "event_msg" or payload.get("type") != "item_completed" or payload.get("thread_id") != host_session():
                continue
            if op != "cancel-requested" and event.get("type") == "event_msg" and payload.get("type") == "item_completed" and payload.get("thread_id") == host_session() and item.get("type") == "SubAgentActivity" and (not child or item.get("agent_thread_id") == child):
                found = (raw, event, item)
    assert found, "No actual attributed host activity event"
    raw, event, item = found
    status = {"context-ack":"context_delivered","cancel-requested":"cancel_requested","launch-unresolved":"launch_unresolved"}.get(op,op)
    if op == "context-ack":
        assert item.get("kind") == "interacted", "Only an actual completed native message can acknowledge context"
        delivery = read("context.stdout.json")["result"]["context_delivery"]
        call = calls.get(item["id"], {})
        assert call.get("type") == "function_call" and call.get("name", "").endswith("send_message"), "Context ACK requires actual native send call"
        args = json.loads(call["arguments"])
        assert args.get("target") in (child, item.get("agent_path")), "Send target must resolve to exact bound child"
        sent_at = datetime.datetime.fromisoformat(event["timestamp"].replace("Z", "+00:00")).timestamp()
        assert sent_at >= (coord / scoped("context.stdout.json")).stat().st_mtime, "Old release/send is not this context delivery"
        message = args.get("message", "")
        plaintext = message == delivery["payload"]
        write("context-send-provenance.json", {"call":call, "result":outputs.get(item["id"]), "event":event, "event_sha256":hashlib.sha256(raw).hexdigest(), "payload_plaintext_corroborated":plaintext, "limitation":"" if plaintext else "Host export encrypts message bytes; actual send/child linkage only"})
        output = outputs.get(item["id"], {})
        assert output.get("call_id") == item["id"] and output.get("output") in ("", "{}"), "Missing successful host send result"
        assert plaintext, "Exact send plaintext unavailable or altered; cannot self-acknowledge an inferred digest"
        value.update(context_delivery=delivery, context_send={"status":"completed","child_id":child,"message_sha256":delivery["payload_sha256"]})
    if op == "cancel-requested":
        assert item.get("kind") == "interrupt_requested", "Actual interrupt result must identify previous running status"
    if op == "running":
        assert item.get("status") == "running", "Host spawn/message event is not a running-status observation"
    if op == "cancelled":
        assert item.get("kind") == "cancelled", "Interruption/close/release/idle is not confirmed host cancellation"
    value.update(observation_status=status, observed_at=event["timestamp"], source_event_id=item["id"], source_event_sha256=hashlib.sha256(raw).hexdigest())
    result = request("observe", value)
elif op in ("stale-result", "child-mismatch"):
    value = read("record-request.json")
    if op == "stale-result":
        value["execution_binding"]["attempt_id"] = "stale-fixture-attempt"
    else:
        value["child_id"] = "wrong-fixture-child"
    before = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
    path = coord / scoped(op + "-request.json"); path.write_text(json.dumps(value))
    proc = subprocess.run(["aether", "codex-native-worker", "record", "--request", str(path)], cwd=repo, capture_output=True, text=True)
    after = {str(p):hashlib.sha256(p.read_bytes()).hexdigest() for p in (repo / ".aether/data").rglob("*") if p.is_file() and p.suffix != ".lock"}
    result = {"exit_status":proc.returncode,"stdout":proc.stdout,"stderr":proc.stderr,"before":before,"after":after}
    write(op + "-refusal.json", result)
    assert proc.returncode != 0 and before == after, "Invalid child/result mutated durable state"
elif op == "resume":
    result = runtime(["resume"], "public-resume")
elif op == "pause":
    result = runtime(["pause"], "public-pause")
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
	r := codexNativeLiveReceipt{FixtureRoot: filepath.Dir(path), CoordinatorPath: path, CoordinatorSHA256: lifecycleDigest([]byte(script))}
	allowed := []string{"python3 " + path + " inspect", "python3 " + path + " bind child-123", "aether command-guide build --platform codex", "git diff --check", "cat /installed/SKILL.md", `jq '{workers: .workers | length}' /tmp/manifest.json`, `rg --files "$CODEX_HOME/sessions"`}
	for _, command := range allowed {
		if !nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}) {
			t.Fatalf("safe fixture coordination refused: %s", command)
		}
	}
	relative := []string{"/bin/zsh", "-lc", "python3 coordinator.py inspect"}
	if !nativeParentCoordinationCommand(&r, relative, "file://"+r.FixtureRoot) {
		t.Fatal("relative pinned coordinator in exact fixture cwd refused")
	}
	if nativeParentCoordinationCommand(&r, relative, "file:///another-fixture") {
		t.Fatal("relative coordinator from different cwd accepted")
	}
	for _, command := range []string{`cmp /tmp/first.json /tmp/second.json`, `jq '.result | {idempotent}' /tmp/first.json /tmp/second.json`, `head -n 1 /tmp/child.jsonl`, `rg --files /tmp/sessions`} {
		if !nativeParentCoordinationCommand(&r, []string{"/bin/zsh", "-lc", command}) {
			t.Fatalf("read-only evidence operation refused: %s", command)
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
	nativeCollectLiveEvidence(t, r, runRoot, fixtureHome)
	r.ResumeWorkerStable, r.ResumeNoSpawn, r.ResumeInspectObserved, r.FinalizationReplayStable = false, false, false, false
	raw, err := r.readEvidence(r.ResumeRawEvents)
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
	resumed.ParentUnclassified = nil
	found := false
	_ = filepath.WalkDir(filepath.Join(fixtureHome, ".codex", "sessions"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, r.ResumeSessionID+".jsonl") {
			return nil
		}
		raw, err := r.readEvidence(path)
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
	if r.Assertions == nil {
		r.Assertions = map[string]bool{}
	}
	r.Assertions["fresh_public_resume"] = nativePublicResumeProof(*r, raw)
	r.ResumeNoSpawn = found && resumed.NativeSpawnCount == 0
	r.ParentSubstitution = r.ParentSubstitution || resumed.ParentSubstitution
	r.ResumeInspectObserved = resumed.ResumeInspectObserved
	r.ParentUnclassified = append(r.ParentUnclassified, resumed.ParentUnclassified...)
	beforeRaw, err := r.readEvidence(r.BeforeResumeJournalPath)
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
	sourceBefore, err := r.readEvidence(filepath.Join(filepath.Dir(r.BeforeResumeJournalPath), "before-resume-source.txt"))
	if err != nil {
		return
	}
	afterRaw, err := r.readEvidence(r.AttemptPath)
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
	first, err1 := r.readEvidence(filepath.Join(coord, "post-finalize-1-state.json"))
	second, err2 := r.readEvidence(filepath.Join(coord, "post-finalize-2-state.json"))
	var final1, final2 struct {
		OK     bool `json:"ok"`
		Result struct {
			Idempotent *bool `json:"idempotent"`
		} `json:"result"`
	}
	out1, err3 := r.readEvidence(filepath.Join(coord, "finalize-1.stdout.json"))
	out2, err4 := r.readEvidence(filepath.Join(coord, "finalize-2.stdout.json"))
	if err1 == nil && err2 == nil && err3 == nil && err4 == nil && json.Unmarshal(out1, &final1) == nil && json.Unmarshal(out2, &final2) == nil {
		r.FinalizationReplayStable = bytes.Equal(first, second) && final1.OK && final2.OK && final1.Result.Idempotent != nil && !*final1.Result.Idempotent && final2.Result.Idempotent != nil && *final2.Result.Idempotent
	}
}

func nativePrepareLiveFixture(t *testing.T, root, runRoot string, scenarios ...string) {
	t.Helper()
	scenario := "ordinary"
	if len(scenarios) != 0 {
		scenario = scenarios[0]
	}
	goal := "Fix integer Clamp boundaries in the dependency-free tiny Go library; a native Builder edits clamp.go and runs go test ./..."
	if scenario == "partial-resume" {
		goal += "; independently fix Double in double.go with its own native Builder and TestDouble"
	}
	if scenario == "question" {
		goal += ". " + nativeGapMarkers[0] + ". Bounds may arrive reversed: before editing, ask the parent the material question 'When low exceeds high, should Clamp swap the bounds or panic?' and wait for the scoped answer. Return this question as JSON with question_id='reversed-bounds' and question text, then wait for a follow-up. Do not infer the answer from the existing tests, which deliberately omit reversed bounds."
	}
	data := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	session := "fixture-session-native-qualification"
	liveSkillWriteJSON(t, filepath.Join(data, "COLONY_STATE.json"), colony.ColonyState{Version: "3.0", Goal: &goal, SessionID: &session, State: colony.StateREADY, ColonyDepth: "light"})
	liveSkillWrite(t, filepath.Join(root, "go.mod"), []byte("module example.invalid/nativefixture\n\ngo 1.23\n"))
	liveSkillWrite(t, filepath.Join(root, "clamp.go"), []byte("package nativefixture\n\nfunc Clamp(value, low, high int) int { return value }\n"))
	liveSkillWrite(t, filepath.Join(root, "clamp_test.go"), []byte("package nativefixture\nimport \"testing\"\nfunc TestClamp(t *testing.T) { for _, c := range [][4]int{{-3,0,10,0},{15,0,10,10},{5,0,10,5},{0,0,10,0},{10,0,10,10}} { if got := Clamp(c[0],c[1],c[2]); got != c[3] { t.Errorf(\"Clamp(%v)=%d want %d\", c[:3],got,c[3]) } } }\n"))
	if scenario == "partial-resume" {
		liveSkillWrite(t, filepath.Join(root, "double.go"), []byte("package nativefixture\n\nfunc Double(value int) int { return value }\n"))
		liveSkillWrite(t, filepath.Join(root, "double_test.go"), []byte("package nativefixture\nimport \"testing\"\nfunc TestDouble(t *testing.T) { for _, n := range []int{-3,0,4} { if got := Double(n); got != 2*n { t.Errorf(\"Double(%d)=%d want %d\", n,got,2*n) } } }\n"))
	}
	liveSkillWrite(t, filepath.Join(root, ".gitignore"), []byte(".aether/\n.codex/\n"))
	liveSkillWrite(t, filepath.Join(root, "AGENTS.md"), []byte("# Disposable native worker fixture\nOnly the runtime-assigned native Builder may edit clamp.go. Do not change clamp_test.go or go.mod. Parent coordinates only. No commits or external actions. Builder edits must use apply_patch so raw FileChange events preserve the complete diff. Run each check individually in this repository. Required test proof: go test ./... -json -count=1 (TestClamp must run, with no filters).\n## Verification Commands\n- build: go build ./...\n- tests: go test ./...\n- lint: go vet ./...\n"))
	if scenario == "partial-resume" {
		liveSkillWrite(t, filepath.Join(root, "AGENTS.md"), []byte("# Disposable two-job native fixture\nTwo independent accepted jobs own clamp.go and double.go respectively. Each named native Builder may edit only its assigned source file using apply_patch. Do not edit tests/go.mod; parent coordinates only. The first Clamp worker runs go test -run '^TestClamp$' -json -count=1 .; TestDouble is intentionally failing until the second worker. The second Double worker runs go test ./... -json -count=1. No commits, external actions or other projects.\n## Verification Commands\n- build: go build ./...\n- tests: go test ./...\n- lint: go vet ./...\n"))
	}
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
	if scenario == "partial-resume" {
		phase.Tasks[0].Goal = "Fix Clamp only in clamp.go. Run go test -run '^TestClamp$' -json -count=1 . for this assignment; TestDouble belongs to the other independent job and may still fail. Do not edit double.go or any tests."
		second := phase.Tasks[0]
		id := "1.2"
		second.ID, second.SemanticID = &id, "double-boundary-fix"
		second.Goal = "Fix Double only in double.go so it returns twice the integer input. Run go test ./... -json -count=1 after this job. Do not edit clamp.go or any tests."
		second.DependsOn = nil
		phase.Tasks = append(phase.Tasks, second)
		declaration := result.Proposal.TaskDeclarations[0]
		declaration.TaskSemanticID, declaration.Files = second.SemanticID, []string{"double.go"}
		result.Proposal.TaskDeclarations = append(result.Proposal.TaskDeclarations, declaration)
	}
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
	if scenario == "question" {
		nativeGapPrepareContext(t, root)
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "prepared-plan-acceptance.json"), accepted)
	liveSkillWriteJSON(t, filepath.Join(runRoot, "prepared-plan-candidate.json"), candidate)
}

func resetCodexNativeDerivedEvidence(r *codexNativeLiveReceipt) {
	r.Cancellation = nil
	r.SessionID, r.ChildID, r.ChildEvents = "", "", ""
	r.ChildTaskPath, r.ChildSpawnCallID, r.ChildIdentityCorroborated = "", "", false
	r.AttemptPath, r.AttemptID, r.RunID, r.LaunchID, r.WorkerName, r.TaskID = "", "", "", "", "", ""
	r.PromptSHA256, r.ResultSHA256, r.CompletionPath, r.BoundHostSessionID = "", "", "", ""
	r.SavedTerminal, r.SavedSourceEventID, r.SavedSourceEventSHA256 = nil, "", ""
	r.ObservedTools, r.ParentUnclassified, r.ChildUnclassified = nil, nil, nil
	r.Workers = nil
	r.Assertions = map[string]bool{}
	r.ChecksPassed, r.ChildEditObserved, r.CreditObserved, r.ParentSubstitution = false, false, false, false
	r.SkillRead, r.SupportRead, r.GuideRead = false, false, false
	r.NativeSpawnCount, r.TerminalCorroborated, r.SourceEventCorroborated = 0, false, false
	r.EmptyResultRefused, r.FinalSource = false, ""
	r.LaunchMessageEncoding, r.PromptDeliveryVerification = "", ""
	r.ResumeSessionID = ""
	r.ResumeWorkerStable, r.ResumeNoSpawn, r.ResumeInspectObserved, r.FinalizationReplayStable = false, false, false, false
}

func nativeCollectLiveEvidence(t *testing.T, r *codexNativeLiveReceipt, runRoot, fixtureHome string) {
	t.Helper()
	if !r.replaying && r.ContextProtocol == codexNativeContextProtocolChildFetch {
		nativeRetainContextCoordination(t, *r)
	}
	resetCodexNativeDerivedEvidence(r)
	if source, err := r.readEvidence(filepath.Join(r.FixtureRoot, "clamp.go")); err == nil {
		r.FinalSource = string(source)
	}
	events, err := r.readEvidence(r.RawEvents)
	if err != nil {
		r.Reason = err.Error()
		return
	}
	scanner := bufio.NewScanner(bytes.NewReader(events))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	skill, _ := r.readEvidence(r.SkillPath)
	support, _ := r.readEvidence(r.SupportPath)
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
		raw, err := r.readEvidence(path)
		if err != nil {
			continue
		}
		var attempt buildAttemptRecord
		if json.Unmarshal(raw, &attempt) != nil || len(attempt.WorkerRuns) == 0 || attempt.WorkerRuns[0].Native == nil {
			continue
		}
		worker := attempt.WorkerRuns[0]
		r.Caste, r.SourceFile = worker.Caste, "clamp.go"
		r.SavedTerminal, r.SavedSourceEventSHA256, r.BoundHostSessionID = worker.Result, worker.Native.SourceEventSHA256, worker.Native.HostSessionID
		r.SavedSourceEventID = worker.Native.SourceEventID
		r.AttemptPath, r.AttemptID, r.RunID = path, attempt.ID, attempt.RunID
		r.LaunchID, r.ChildID, r.WorkerName, r.TaskID = worker.ProviderRunID, worker.Native.ChildID, worker.WorkerName, worker.TaskID
		r.PromptSHA256, r.ResultSHA256, r.CompletionPath = worker.Native.PromptSHA256, worker.ResultSHA256, attempt.CompletionPath
		liveSkillWrite(t, filepath.Join(runRoot, "terminal-attempt.json"), raw)
	}
	// Resolve all journal-bound children before classifying parent bind argv.
	// A display path is allowed only when the actual host chain proves it.
	if r.SchemaVersion == "codex-native-tracer/v2" {
		nativeCollectChildIdentity(r, fixtureHome)
		nativeCollectQualificationWorkers(t, r, runRoot, fixtureHome)
	}
	// Each fact requires one unique raw capture; neither a missing nor a
	// duplicated child/parent export inherits an earlier proof.
	for _, id := range []string{r.ChildID, r.SessionID} {
		if id == "" {
			continue
		}
		paths, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*"+id+".jsonl"))
		if len(paths) != 1 {
			if id == r.SessionID {
				r.ParentSubstitution = true
			}
			continue
		}
		raw, err := r.readEvidence(paths[0])
		if err != nil {
			if id == r.SessionID {
				r.ParentSubstitution = true
			}
			continue
		}
		if id == r.ChildID {
			r.ChildEvents = paths[0]
			nativeInspectChildEvents(r, raw)
		}
		if id == r.SessionID {
			nativeInspectParentEvents(r, raw)
		}
	}
	stateRaw, err := r.readEvidence(filepath.Join(r.FixtureRoot, ".aether", "data", "COLONY_STATE.json"))
	if err == nil {
		var state colony.ColonyState
		if json.Unmarshal(stateRaw, &state) == nil && len(state.Plan.Phases) == 1 && len(state.Plan.Phases[0].Tasks) > 0 {
			r.CreditObserved = true
			for _, task := range state.Plan.Phases[0].Tasks {
				r.CreditObserved = r.CreditObserved && task.Status == colony.TaskCompleted
			}
		}
	}
	diff := exec.Command("git", "diff", "--", "clamp.go", "clamp_test.go", "go.mod")
	diff.Dir = r.FixtureRoot
	if raw, err := diff.Output(); err == nil {
		liveSkillWrite(t, filepath.Join(runRoot, "child-edit.patch"), raw)
	}
	if r.SchemaVersion != "codex-native-tracer/v2" {
		nativeCollectQualificationWorkers(t, r, runRoot, fixtureHome)
	}
}

// Reuse the coordinator's pure resolver with inventory-checked raw bytes. This
// observer runs no coordinator operations and never launches a native helper.
func nativeCollectChildIdentity(r *codexNativeLiveReceipt, fixtureHome string) {
	r.ChildTaskPath, r.ChildSpawnCallID, r.ChildIdentityCorroborated = "", "", false
	paths, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*.jsonl"))
	var events []json.RawMessage
	var metadata []json.RawMessage
	for _, path := range paths {
		raw, err := r.readEvidence(path)
		if err != nil {
			continue
		}
		lines := bytes.Split(bytes.TrimSpace(raw), []byte{'\n'})
		if len(lines) == 0 {
			continue
		}
		var first nativeHostEvent
		if json.Unmarshal(lines[0], &first) != nil || first.Type != "session_meta" {
			continue
		}
		if first.Payload.ParentThreadID != "" {
			metadata = append(metadata, json.RawMessage(lines[0]))
		}
		if first.Payload.ID == r.BoundHostSessionID && first.Payload.ParentThreadID == "" {
			if events != nil {
				return
			}
			for _, line := range lines {
				events = append(events, json.RawMessage(line))
			}
		}
	}
	start := strings.Index(nativeFixtureCoordinator, "def resolve_native_child(")
	end := strings.Index(nativeFixtureCoordinator, "def bound_native_child(")
	input, err := json.Marshal(map[string]any{"events": events, "metadata": metadata, "target": r.ChildID, "parent": r.BoundHostSessionID, "cwd": r.FixtureRoot})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "-c", "import json,sys\n"+nativeFixtureCoordinator[start:end]+"\nprint(json.dumps(resolve_native_child(**json.load(sys.stdin))))")
	cmd.Stdin = bytes.NewReader(input)
	raw, err := cmd.Output()
	if err != nil {
		return
	}
	var result struct {
		ChildID  string `json:"child_id"`
		TaskPath string `json:"task_path"`
		CallID   string `json:"call_id"`
	}
	if json.Unmarshal(raw, &result) != nil || result.ChildID != r.ChildID {
		return
	}
	r.ChildTaskPath, r.ChildSpawnCallID, r.ChildIdentityCorroborated = result.TaskPath, result.CallID, true
}

// Only thread-attributed host items qualify. A child's rollout also contains
// copied parent response_items; those are never child execution evidence.
func nativeCollectQualificationWorkers(t *testing.T, r *codexNativeLiveReceipt, runRoot, fixtureHome string) {
	r.Workers = nil
	if r.AttemptPath == "" {
		return
	}
	var attempt buildAttemptRecord
	raw, err := r.readEvidence(r.AttemptPath)
	if err != nil || json.Unmarshal(raw, &attempt) != nil {
		return
	}
	r.Workers = nil
	for index, worker := range attempt.WorkerRuns {
		if worker.Native == nil {
			continue
		}
		child := *r
		child.ChildEvents = ""
		child.ChildUnclassified = nil
		child.ChildEditObserved, child.ChecksPassed, child.TerminalCorroborated, child.SourceEventCorroborated = false, false, false, false
		child.Workers, child.Assertions, child.Artifacts = nil, nil, nil
		child.WorkerName, child.Caste, child.TaskID = worker.WorkerName, worker.Caste, worker.TaskID
		child.LaunchID, child.ChildID, child.BoundHostSessionID = worker.ProviderRunID, worker.Native.ChildID, worker.Native.HostSessionID
		child.SavedTerminal, child.ResultSHA256, child.SavedSourceEventID, child.SavedSourceEventSHA256 = worker.Result, worker.ResultSHA256, worker.Native.SourceEventID, worker.Native.SourceEventSHA256
		child.PromptSHA256 = worker.Native.PromptSHA256
		child.SourceFile = "clamp.go"
		if r.Scenario == "partial-resume" && index == 1 {
			child.SourceFile = "double.go"
		}
		before, _ := r.readEvidence(filepath.Join(filepath.Dir(r.FixtureRoot), "baseline-"+child.SourceFile+".txt"))
		if len(before) > 0 {
			child.BaselineSource = string(before)
		}
		after, _ := r.readEvidence(filepath.Join(r.FixtureRoot, child.SourceFile))
		child.FinalSource = string(after)
		matches, _ := filepath.Glob(filepath.Join(fixtureHome, ".codex", "sessions", "*", "*", "*", "*"+child.ChildID+".jsonl"))
		if child.ChildID != "" && len(matches) == 1 {
			child.ChildEvents = matches[0]
			events, err := r.readEvidence(matches[0])
			if err == nil {
				nativeInspectChildEvents(&child, events)
			} else {
				child.ChildEvents = ""
			}
		}
		if r.SchemaVersion == "codex-native-tracer/v2" {
			nativeCollectChildIdentity(&child, fixtureHome)
		}
		r.Workers = append(r.Workers, child)
	}
}

// The first independent job is allowed to check only its accepted TestClamp.
// The final second job still requires the complete suite; arbitrary filters
// never qualify the ordinary full-suite proof.
func nativePartialFixtureTest(command []string, output string) bool {
	if len(command) != 3 || (command[1] != "-lc" && command[1] != "-c") {
		return false
	}
	words, ok := nativeSimpleShellWords(command[2])
	if !ok || len(words) != 7 || words[0] != "go" || words[1] != "test" {
		return false
	}
	flags := map[string]bool{}
	for _, word := range words[2:] {
		flags[word] = true
	}
	if !flags["-run"] || !flags["^TestClamp$"] || !flags["-json"] || !flags["-count=1"] || !flags["."] {
		return false
	}
	return nativeRequiredFixtureTest([]string{command[0], command[1], "go test ./... -json -count=1"}, output)
}

type nativeHostEvent struct {
	Type    string `json:"type"`
	Payload struct {
		Type           string `json:"type"`
		ID             string `json:"id"`
		ParentThreadID string `json:"parent_thread_id"`
		AgentRole      string `json:"agent_role"`
		Cwd            string `json:"cwd"`
		ThreadID       string `json:"thread_id"`
		TurnID         string `json:"turn_id"`
		Item           struct {
			Type     string          `json:"type"`
			Kind     string          `json:"kind"`
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

type nativeCodeModeWire struct {
	Type    string
	Payload struct {
		Type, Name, Input string
		CallID            string `json:"call_id"`
		Output            []struct{ Type, Text string }
		Metadata          struct {
			TurnID string `json:"turn_id"`
		} `json:"internal_chat_message_metadata_passthrough"`
	}
}

func nativeCodeModeResult(w nativeCodeModeWire) (int, string, bool) {
	if len(w.Payload.Output) != 2 || w.Payload.Output[0].Type != "input_text" || !strings.HasPrefix(w.Payload.Output[0].Text, "Script completed\n") || w.Payload.Output[1].Type != "input_text" {
		return 0, "", false
	}
	var result struct {
		Exit    *int   `json:"exit_code"`
		Session *int   `json:"session_id"`
		Output  string `json:"output"`
	}
	if json.Unmarshal([]byte(w.Payload.Output[1].Text), &result) != nil || result.Exit == nil || result.Session != nil {
		return 0, "", false
	}
	return *result.Exit, result.Output, true
}

// Watcher ledger bookkeeping is explicitly requested by the build brief. It is
// neither a source edit nor test proof. Only the pinned runtime's bounded CLI,
// observed event/result and retained owned ledger may classify the operation.
type nativeWatcherFinding struct {
	Severity    string `json:"severity"`
	Title       string `json:"title,omitempty"` // Runtime ignores this descriptive field.
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion,omitempty"`
}

type nativeWatcherLedgerCommand struct {
	Domain, AgentName, Findings string
	Help                        bool
}

func nativeWatcherLedgerWords(r codexNativeLiveReceipt, command string) (nativeWatcherLedgerCommand, bool) {
	var result nativeWatcherLedgerCommand
	if r.Scenario != "review" || r.Caste != "watcher" || r.WorkerName == "" || r.TaskID != "verification-watcher-"+strings.ToLower(r.WorkerName) || !filepath.IsAbs(r.FixtureRoot) || r.CandidatePath != filepath.Join(filepath.Dir(r.FixtureRoot), "bin", "aether") {
		return result, false
	}
	words, ok := nativeSimpleShellWords(command)
	if !ok || len(words) < 3 || words[0] != r.CandidatePath || words[1] != "review-ledger-write" {
		return result, false
	}
	if len(words) == 3 && words[2] == "--help" {
		result.Help = true
		return result, true
	}
	if len(words) != 10 && len(words) != 12 {
		return result, false
	}
	flags := map[string]string{}
	for i := 2; i < len(words); i += 2 {
		key := words[i]
		if _, found := flags[key]; found {
			return result, false
		}
		switch key {
		case "--domain", "--phase", "--findings", "--agent", "--agent-name":
		default:
			return result, false
		}
		flags[key] = words[i+1]
	}
	if (flags["--domain"] != "testing" && flags["--domain"] != "quality") || flags["--phase"] != "1" || flags["--agent"] != "watcher" || len(flags["--findings"]) == 0 || len(flags["--findings"]) > 64*1024 {
		return result, false
	}
	if name, found := flags["--agent-name"]; found && name != r.WorkerName {
		return result, false
	}
	result.Domain, result.AgentName, result.Findings = flags["--domain"], flags["--agent-name"], flags["--findings"]
	return result, true
}

// Reject duplicate fields at every depth, non-JSON tails and excessive nesting.
// Decoding remains data-only; nothing here evaluates shell or JavaScript.
func nativeWatcherStrictJSON(raw []byte, target any) bool {
	if len(raw) > 256*1024 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var value func(int) bool
	value = func(depth int) bool {
		if depth > 16 {
			return false
		}
		token, err := d.Token()
		if err != nil {
			return false
		}
		delim, compound := token.(json.Delim)
		if !compound {
			return true
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				name, ok := key.(string)
				if err != nil || !ok || seen[name] {
					return false
				}
				seen[name] = true
				if !value(depth + 1) {
					return false
				}
			}
		case '[':
			for d.More() {
				if !value(depth + 1) {
					return false
				}
			}
		default:
			return false
		}
		end, err := d.Token()
		return err == nil && ((delim == '{' && end == json.Delim('}')) || (delim == '[' && end == json.Delim(']')))
	}
	if !value(0) {
		return false
	}
	if _, err := d.Token(); err != io.EOF {
		return false
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(target) == nil
}

func nativeWatcherOwnedBytes(r codexNativeLiveReceipt, path string) ([]byte, bool) {
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil || resolved != path {
		return nil, false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, false
	}
	raw, err := r.readEvidence(path)
	return raw, err == nil
}

func nativeWatcherLedgerResult(r codexNativeLiveReceipt, c nativeWatcherLedgerCommand, exit int, output string) bool {
	candidate, ok := nativeWatcherOwnedBytes(r, r.CandidatePath)
	if !ok || r.CandidateSHA256 == "" || lifecycleDigest(candidate) != r.CandidateSHA256 || nativeValidateFixtureBaselines(r) != nil {
		return false
	}
	for _, name := range []string{"clamp_test.go", "go.mod"} {
		if _, ok := nativeWatcherOwnedBytes(r, filepath.Join(r.FixtureRoot, name)); !ok {
			return false
		}
	}
	source, ok := nativeWatcherOwnedBytes(r, filepath.Join(r.FixtureRoot, "clamp.go"))
	if !ok || r.FinalSource == "" || string(source) != r.FinalSource {
		return false
	}
	if c.Help {
		return exit == 0 && strings.Contains(output, "Write findings to a domain review ledger\n") && strings.Contains(output, "Usage:\n  aether review-ledger-write [flags]")
	}
	// The production parser returns this exact refusal before append/save. A
	// schema rejection by our stricter success decoder is not that runtime error.
	var runtimeFindings []struct {
		Severity, File, Category, Description, Suggestion string
		Line                                              int
	}
	if json.Unmarshal([]byte(c.Findings), &runtimeFindings) != nil {
		var refused struct {
			OK    *bool  `json:"ok"`
			Error string `json:"error"`
			Code  int    `json:"code"`
		}
		return exit == 1 && nativeWatcherStrictJSON([]byte(output), &refused) && refused.OK != nil && !*refused.OK && refused.Error == "invalid --findings JSON" && refused.Code == 1
	}
	var findings []nativeWatcherFinding
	if exit != 0 || !nativeWatcherStrictJSON([]byte(c.Findings), &findings) || len(findings) == 0 || len(findings) > 50 {
		return false
	}
	var envelope struct {
		OK     *bool `json:"ok"`
		Result struct {
			Written *bool                      `json:"written"`
			Domain  string                     `json:"domain"`
			Total   int                        `json:"total"`
			Summary colony.ReviewLedgerSummary `json:"summary"`
		} `json:"result"`
	}
	if !nativeWatcherStrictJSON([]byte(output), &envelope) || envelope.OK == nil || !*envelope.OK || envelope.Result.Written == nil || !*envelope.Result.Written || envelope.Result.Domain != c.Domain || envelope.Result.Total < len(findings) {
		return false
	}
	ledgerRaw, ok := nativeWatcherOwnedBytes(r, filepath.Join(r.FixtureRoot, ".aether", "data", "reviews", c.Domain, "ledger.json"))
	if !ok {
		return false
	}
	var ledger colony.ReviewLedgerFile
	if !nativeWatcherStrictJSON(ledgerRaw, &ledger) || envelope.Result.Total > len(ledger.Entries) || ledger.Summary != colony.ComputeSummary(ledger.Entries) {
		return false
	}
	end := envelope.Result.Total
	start := end - len(findings)
	if colony.ComputeSummary(ledger.Entries[:end]) != envelope.Result.Summary {
		return false
	}
	prefix := "tst"
	if c.Domain == "quality" {
		prefix = "qlt"
	}
	for i, f := range findings {
		e := ledger.Entries[start+i]
		if e.ID != colony.FormatEntryID(prefix, 1, colony.NextEntryIndex(ledger.Entries[:start+i], prefix, 1)) || e.Phase != 1 || e.PhaseName != "" || e.Agent != "watcher" || e.AgentName != c.AgentName || e.Status != "open" || e.ResolvedAt != nil || string(e.Severity) != strings.ToUpper(f.Severity) || e.File != f.File || e.Line != f.Line || e.Category != f.Category || e.Description != f.Description || e.Suggestion != f.Suggestion {
			return false
		}
		if _, err := time.Parse(time.RFC3339, e.GeneratedAt); err != nil {
			return false
		}
	}
	return true
}

func nativeCorroboratedWatcherLedger(r codexNativeLiveReceipt, raw []byte, callID string) (string, bool) {
	var commands []nativeRecordedShellCommand
	var c nativeWatcherLedgerCommand
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var w nativeCodeModeWire
		if json.Unmarshal(line, &w) == nil && w.Type == "response_item" && w.Payload.Type == "custom_tool_call" && w.Payload.CallID == callID {
			var ok bool
			commands, ok = nativeCodeModeCommands(w.Payload.Input, r.FixtureRoot)
			if !ok || len(commands) != 1 || !nativeSameCwd(commands[0].Cwd, r.FixtureRoot) {
				return "", false
			}
			c, ok = nativeWatcherLedgerWords(r, commands[0].Command)
			if !ok {
				return "", false
			}
		}
	}
	if len(commands) != 1 || !nativeCorroboratedBatch(raw, r.ChildID, callID, commands) || !nativeCorroboratedBatchResults(raw, callID, commands) {
		return "", false
	}
	active := false
	eventID := ""
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var w nativeCodeModeWire
		_ = json.Unmarshal(line, &w)
		if w.Type == "response_item" && w.Payload.CallID == callID {
			if w.Payload.Type == "custom_tool_call" {
				active = true
			}
			if w.Payload.Type == "custom_tool_call_output" {
				active = false
			}
		}
		var e nativeHostEvent
		if !active || json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" {
			continue
		}
		i := e.Payload.Item
		if i.Type == "FileChange" {
			return "", false
		}
		if i.Type == "CommandExecution" {
			if i.ExitCode == nil || !nativeWatcherLedgerResult(r, c, *i.ExitCode, i.Output) {
				return "", false
			}
			eventID = i.ID
		}
	}
	return eventID, eventID != ""
}

func nativeFixtureSmoke(words []string) bool {
	return len(words) == 6 && words[0] == "go" && words[1] == "test" && words[2] == "-run" && words[3] == "^$" && words[4] == "-count=1" && words[5] == "./..."
}

func nativeFixtureInspectionExtra(words []string) bool {
	return (len(words) == 3 && words[0] == "git" && words[1] == "diff" && words[2] == "--stat") || (len(words) == 3 && words[0] == "go" && words[1] == "list" && words[2] == "./...")
}

func nativeInspectChildEvents(r *codexNativeLiveReceipt, raw []byte) {
	r.ChildEditObserved, r.ChecksPassed = false, false
	r.TerminalCorroborated, r.SourceEventCorroborated = false, false
	r.ChildUnclassified = nil
	attributed := false
	metadataSeen := false
	childTurns := nativeOwnedHostTurns(raw, r.ChildID)
	type pendingCheck struct {
		command        []string
		turn           string
		epoch          int
		priorProof     bool
		observedPlain  bool
		observedOutput string
		projected      bool
	}
	pendingChecks := map[string]pendingCheck{}
	ledgerEvents := map[string]bool{}
	callCounts, outputCounts := map[string]int{}, map[string]int{}
	commandEventCounts := map[string]int{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var event nativeHostEvent
		if json.Unmarshal(line, &event) == nil && event.Type == "event_msg" && event.Payload.Type == "item_completed" && event.Payload.Item.Type == "CommandExecution" {
			commandEventCounts[event.Payload.Item.ID]++
		}
		var w nativeCodeModeWire
		if json.Unmarshal(line, &w) == nil && w.Type == "response_item" && w.Payload.CallID != "" {
			if w.Payload.Type == "custom_tool_call" {
				callCounts[w.Payload.CallID]++
			}
			if w.Payload.Type == "custom_tool_call_output" {
				outputCounts[w.Payload.CallID]++
			}
		}
	}
	checkEpoch := 0
	source := r.BaselineSource
	caste, sourceFile := r.Caste, r.SourceFile
	if caste == "" {
		caste = "builder"
	}
	if sourceFile == "" {
		sourceFile = "clamp.go"
	}
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
					e.Payload.AgentRole == "aether-"+caste && nativeSameCwd(e.Payload.Cwd, r.FixtureRoot)
			}
			continue
		}
		p := e.Payload
		if attributed && e.Type == "response_item" {
			var wire nativeCodeModeWire
			if json.Unmarshal(line, &wire) == nil && childTurns[wire.Payload.Metadata.TurnID] {
				p := wire.Payload
				if p.Type == "custom_tool_call" && (p.Name == "exec" || strings.HasSuffix(p.Name, ".exec")) {
					if eventID, ok := nativeCorroboratedWatcherLedger(*r, raw, p.CallID); ok {
						ledgerEvents[eventID] = true
						continue
					}
					if nativeCorroboratedLiteralPatch(*r, raw, p.CallID) {
						continue
					}
					commands, ok := nativeCodeModeCommands(p.Input, r.FixtureRoot)
					fullBatchResults := nativeReadOnlyBatchFullResults(p.Input)
					_, _, concatenated := nativeReadOnlyBatchConcatenation(p.Input)
					_, exitOutput := nativeCodeModeExitOutputProjection(p.Input, r.FixtureRoot)
					inspectionChain, longDate, extraInspection := false, false, false
					needsCommandEvent := exitOutput || fullBatchResults || concatenated || regexp.MustCompile(`^\s*const\s+\[`).MatchString(p.Input)
					for _, command := range commands {
						// A semicolon inspection chain is one actual invocation,
						// corroborated as a whole, never split into invented events.
						inspectionChain = inspectionChain || strings.Contains(command.Command, " && ") || strings.Contains(command.Command, ";")
						needsCommandEvent = needsCommandEvent || inspectionChain || strings.Contains(command.Command, ";")
						// A multi-file sed read is likewise one invocation, not one
						// event per operand. Require its exact child/cwd event.
						words, literal := nativeSimpleShellWords(command.Command)
						longDate = longDate || (literal && nativeLongDateWords(words))
						extraInspection = extraInspection || (literal && (nativeFixtureInspectionExtra(words) || nativeFixtureSmoke(words)))
						needsCommandEvent = needsCommandEvent || extraInspection
						needsCommandEvent = needsCommandEvent || longDate
						needsCommandEvent = needsCommandEvent || (literal && len(words) > 4 && words[0] == "sed" && words[1] == "-n")
					}
					if ok && needsCommandEvent {
						ok = nativeCorroboratedBatch(raw, r.ChildID, p.CallID, commands, inspectionChain || nativeReadOnlyBatchForEach(p.Input) || concatenated)
						if ok && exitOutput {
							ok = nativeCorroboratedExitOutput(raw, p.CallID, r.FixtureRoot)
						}
						if ok && concatenated {
							ok = nativeCorroboratedBatchConcatenation(raw, p.CallID, commands)
						}
						if ok && (fullBatchResults || ((longDate || extraInspection) && !exitOutput && !concatenated)) {
							ok = nativeCorroboratedBatchResults(raw, p.CallID, commands)
						}
					}
					if !ok {
						r.ChildUnclassified = append(r.ChildUnclassified, "unclassified child code-mode: "+p.Input)
						r.ChecksPassed = false
						checkEpoch++
					} else {
						for _, command := range commands {
							argv := []string{"/bin/sh", "-c", command.Command}
							allowed := nativeSameCwd(command.Cwd, r.FixtureRoot) && nativeChildCommandAllowed(*r, argv)
							if !allowed {
								r.ChildUnclassified = append(r.ChildUnclassified, command.Command)
								r.ChecksPassed = false
								checkEpoch++
							}
							words, _ := nativeFixtureShellWords(argv)
							if len(words) >= 2 && words[0] == "go" && words[1] == "test" {
								priorProof := r.ChecksPassed
								r.ChecksPassed = false
								checkEpoch++
								projected := nativeCodeModePlainOutput(p.Input, r.FixtureRoot)
								// A complete uncached result can establish proof without a
								// nested event. Plain rechecks and .output projections still
								// require exact event corroboration to preserve prior proof.
								fullResult := nativeAssignedFixtureCommand(*r, argv) && !projected
								if allowed && len(commands) == 1 && p.CallID != "" && callCounts[p.CallID] == 1 && outputCounts[p.CallID] == 1 && (fullResult || nativeCorroboratedBatch(raw, r.ChildID, p.CallID, commands)) {
									pendingChecks[p.CallID] = pendingCheck{command: argv, turn: p.Metadata.TurnID, epoch: checkEpoch, priorProof: priorProof, projected: projected}
								}
							}
						}
					}
				}
				if p.Type == "custom_tool_call_output" {
					pending, ok := pendingChecks[p.CallID]
					delete(pendingChecks, p.CallID)
					if ok && pending.turn == p.Metadata.TurnID && pending.epoch == checkEpoch {
						exit, output, complete := nativeCodeModeResult(wire)
						r.ChecksPassed = complete && exit == 0 && (nativeAssignedFixtureTest(*r, pending.command, output) || (pending.priorProof && pending.observedPlain && output == pending.observedOutput))
						// A literal .output projection omits the result's exit code.
						// Only the separately matched successful event may supply it,
						// and only to preserve an existing unchanged-source proof.
						if pending.projected && pending.priorProof && pending.observedPlain && len(p.Output) == 2 && p.Output[0].Type == "input_text" && strings.HasPrefix(p.Output[0].Text, "Script completed\n") && p.Output[1].Type == "input_text" && p.Output[1].Text == pending.observedOutput {
							r.ChecksPassed = true
						}
					}
				}
			}
		}
		if !attributed || e.Type != "event_msg" || p.Type != "item_completed" || p.ThreadID != r.ChildID {
			continue
		}
		i := p.Item
		switch i.Type {
		case "FileChange":
			checkEpoch++
			if i.Status != "completed" {
				continue
			}
			r.ChecksPassed = false
			if caste == "watcher" {
				r.ChildUnclassified = append(r.ChildUnclassified, "Watcher FileChange is outside review ownership")
				continue
			}
			for path, change := range i.Changes {
				if filepath.Clean(path) != filepath.Join(r.FixtureRoot, sourceFile) || change.Type != "update" || change.MovePath != nil {
					patchValid = false
					r.ChildUnclassified = append(r.ChildUnclassified, "unowned FileChange: "+path)
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
			if ledgerEvents[i.ID] {
				continue
			}
			words, _ := nativeFixtureShellWords(i.Command)
			if len(words) >= 2 && words[0] == "go" && words[1] == "test" {
				checkEpoch++
				r.ChecksPassed = false
			}
			if i.Status != "completed" {
				continue
			}
			if !nativeSameCwd(i.Cwd, r.FixtureRoot) || !nativeChildCommandAllowed(*r, i.Command) {
				r.ChildUnclassified = append(r.ChildUnclassified, strings.Join(i.Command, " "))
				r.ChecksPassed = false
			}
			if i.ExitCode != nil && nativeSameCwd(i.Cwd, r.FixtureRoot) {
				// A later successful plain suite cannot create uncached proof, but
				// must not erase an already proved unchanged-source uncached suite.
				// Restore only after this unique same-turn event AND its exact output.
				if nativeAdditionalFixtureCheck(words) && *i.ExitCode == 0 && i.ID != "" && commandEventCounts[i.ID] == 1 {
					for id, pending := range pendingChecks {
						w, _ := nativeFixtureShellWords(pending.command)
						if pending.priorProof && pending.turn == p.TurnID && pending.epoch+1 == checkEpoch && nativeAdditionalFixtureCheck(w) && len(i.Command) == 3 && i.Command[2] == pending.command[2] {
							pending.epoch, pending.observedPlain, pending.observedOutput = checkEpoch, true, i.Output
							pendingChecks[id] = pending
						}
					}
				}
				if nativeAssignedFixtureCommand(*r, i.Command) {
					r.ChecksPassed = *i.ExitCode == 0 && nativeAssignedFixtureTest(*r, i.Command, i.Output)
				} else if *i.ExitCode != 0 && len(i.Command) == 3 {
					words, _ := nativeFixtureShellWords(i.Command)
					if len(words) >= 2 && words[0] == "go" && words[1] == "test" {
						r.ChecksPassed = false
					}
				}
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
	return nativeNamedFixtureTest(command, output, "TestClamp")
}

func nativeNamedFixtureTest(command []string, output, requiredTest string) bool {
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
		if event.Test == requiredTest && event.Action == "run" {
			ran = true
		}
		if event.Test == requiredTest && event.Action == "pass" && ran {
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
	if err := nativeValidateLiveContextReceipts(r); err != nil {
		return err
	}
	if r.SchemaVersion == "codex-native-tracer/v2" && !r.ChildIdentityCorroborated {
		return fmt.Errorf("actual spawn call/result/activity and child metadata mapping incomplete")
	}
	if r.SchemaVersion == "codex-native-tracer/v2" && r.Scenario == "ordinary" && !r.FinalizationReplayStable {
		return fmt.Errorf("ordinary exactly-once finalization replay incomplete")
	}
	if !nativeGapParentExitAccepted(r) {
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
		raw, err := r.readEvidence(path)
		if err != nil || lifecycleDigest(raw) != want {
			return fmt.Errorf("candidate/client/baseline identity changed: %s", path)
		}
	}
	if !r.ChildEditObserved || !r.ChecksPassed {
		return fmt.Errorf("raw native child edit/check evidence incomplete (edit=%v checks=%v); child log %s", r.ChildEditObserved, r.ChecksPassed, r.ChildEvents)
	}
	if r.ParentSubstitution || len(r.ChildUnclassified) != 0 {
		return fmt.Errorf("parent writes or unclassified operations prevent child-only attribution: parent_substitution=%v parent=%v child=%v", r.ParentSubstitution, r.ParentUnclassified, r.ChildUnclassified)
	}
	if r.AttemptID == "" || r.LaunchID == "" || r.ResultSHA256 == "" || r.CompletionPath == "" || !r.CreditObserved {
		return fmt.Errorf("accepted native terminal/aggregate/finalizer credit incomplete")
	}
	if !r.EmptyResultRefused {
		return fmt.Errorf("actual empty-result refusal was not observed")
	}
	if err := nativeValidateEmptyRefusal(r, filepath.Join(filepath.Dir(r.FixtureRoot), "coordination")); err != nil {
		return err
	}
	if r.Scenario == "early-resume" && (!r.Assertions["fresh_public_resume"] || r.ResumeExitStatus != 0 || r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID || !r.ResumeWorkerStable || !r.ResumeNoSpawn || !r.ResumeInspectObserved || !r.FinalizationReplayStable) {
		return fmt.Errorf("fresh-session continuity/replay proof incomplete")
	}
	return nil
}

// Inspect actual parent calls separately. The native tool count is independent
// of the parent CLI's display schema; no assistant text counts as a launch.
// Codex may deliver the selected skill as a host-tagged input rather than a
// shell read. Require that exact host shape, unique message, owned turn, path
// and complete installed bytes; ordinary user prose never proves selection.
func nativeSelectedSkillDelivered(r codexNativeLiveReceipt, raw []byte) bool {
	skill, err := r.readEvidence(r.SkillPath)
	if err != nil || len(skill) == 0 {
		return false
	}
	expected := "<skill>\n<name>ant-build</name>\n<path>" + r.SkillPath + "</path>\n" + string(skill) + "\n</skill>"
	turns := nativeOwnedHostTurns(raw, r.SessionID)
	ids := map[string]int{}
	var matches []string
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type    string `json:"type"`
			Payload struct {
				Type    string `json:"type"`
				ID      string `json:"id"`
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
				Metadata struct {
					TurnID string   `json:"turn_id"`
					Kinds  []string `json:"content_item_kinds"`
				} `json:"internal_chat_message_metadata_passthrough"`
			} `json:"payload"`
		}
		if json.Unmarshal(line, &e) != nil || e.Type != "response_item" || e.Payload.Type != "message" {
			continue
		}
		p := e.Payload
		ids[p.ID]++
		if p.ID != "" && p.Role == "user" && turns[p.Metadata.TurnID] && len(p.Metadata.Kinds) == 1 && p.Metadata.Kinds[0] == "skills.selected_skill_instructions" && len(p.Content) == 1 && p.Content[0].Type == "input_text" && p.Content[0].Text == expected {
			matches = append(matches, p.ID)
		}
	}
	return len(matches) == 1 && ids[matches[0]] == 1
}

func nativeInspectParentEvents(r *codexNativeLiveReceipt, raw []byte) {
	var metadata nativeHostEvent
	if json.Unmarshal(bytes.SplitN(raw, []byte{'\n'}, 2)[0], &metadata) != nil || metadata.Type != "session_meta" || metadata.Payload.ID != r.SessionID || metadata.Payload.ParentThreadID != "" {
		r.ParentSubstitution = true // attribution is missing; fail closed
		return
	}
	if r.SchemaVersion == "codex-native-tracer/v2" && nativeSelectedSkillDelivered(*r, raw) {
		r.SkillRead = true
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
				if !nativeParentCoordinationCommand(r, i.Command, i.Cwd) {
					r.ParentSubstitution = true
					r.ParentUnclassified = append(r.ParentUnclassified, strings.Join(i.Command, " "))
				}
				var words []string
				if len(i.Command) == 3 {
					words, _ = nativeSimpleShellWords(i.Command[2])
				}
				if len(words) >= 3 && nativeCoordinatorPathMatches(r, i.Cwd, words[1]) && words[2] == "empty-result" && i.ExitCode != nil && *i.ExitCode != 0 && strings.Contains(i.Output, "nonempty terminal result") {
					r.EmptyResultRefused = true
				}
				if len(words) == 3 && nativeCoordinatorPathMatches(r, i.Cwd, words[1]) && words[2] == "inspect" && i.ExitCode != nil && *i.ExitCode == 0 && strings.Contains(i.Output, r.ResultSHA256) && r.ResultSHA256 != "" {
					r.ResumeInspectObserved = true
				}
				if i.Status == "completed" && i.ExitCode != nil && *i.ExitCode == 0 && nativePublicInspectEvidence(*r, i.Command, i.Output) {
					r.ResumeInspectObserved = true
				}
				if i.Status == "completed" && i.ExitCode != nil && *i.ExitCode == 0 && len(i.Command) == 3 {
					command := i.Command[2]
					for path, target := range map[string]*bool{r.SkillPath: &r.SkillRead, r.SupportPath: &r.SupportRead} {
						content, err := r.readEvidence(path)
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
				CallID    string `json:"call_id"`
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
		if p.Type == "custom_tool_call" && (p.Name == "exec" || strings.HasSuffix(p.Name, ".exec")) {
			commands, ok := nativeCodeModeCommands(p.Input, metadata.Payload.Cwd)
			// Legacy receipts retain their original literal-only interpretation.
			// Every new capture requires actual events for single commands too.
			if ok && (r.SchemaVersion == "codex-native-tracer/v2" || strings.Contains(p.Input, "Promise.allSettled")) {
				ok = nativeCorroboratedBatch(raw, r.SessionID, p.CallID, commands)
			}
			if ok && nativeReadOnlyBatchForEach(p.Input) {
				ok = nativeCorroboratedBatch(raw, r.SessionID, p.CallID, commands, true) && nativeCorroboratedBatchResults(raw, p.CallID, commands)
			}
			if _, _, concatenated := nativeReadOnlyBatchConcatenation(p.Input); ok && concatenated {
				ok = nativeCorroboratedBatch(raw, r.SessionID, p.CallID, commands, true) && nativeCorroboratedBatchConcatenation(raw, p.CallID, commands)
			}
			if _, projected := nativeCodeModeExitOutputProjection(p.Input, metadata.Payload.Cwd); ok && projected {
				ok = nativeCorroboratedBatch(raw, r.SessionID, p.CallID, commands) && nativeCorroboratedExitOutput(raw, p.CallID, r.FixtureRoot)
			}
			if !ok {
				r.ParentSubstitution = true
				r.ParentUnclassified = append(r.ParentUnclassified, "unclassified code-mode: "+p.Input)
			} else {
				for _, command := range commands {
					if !nativeParentCoordinationCommand(r, []string{"/bin/sh", "-c", command.Command}, command.Cwd) {
						r.ParentSubstitution = true
						r.ParentUnclassified = append(r.ParentUnclassified, "unclassified code-mode command: "+command.Command)
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

func nativeCoordinatorPathMatches(r *codexNativeLiveReceipt, cwd, path string) bool {
	if !filepath.IsAbs(path) {
		if !nativeSameCwd(cwd, r.FixtureRoot) {
			return false
		}
		path = filepath.Join(r.FixtureRoot, path)
	}
	return filepath.Clean(path) == filepath.Clean(r.CoordinatorPath)
}

func nativeParentCoordinationCommand(r *codexNativeLiveReceipt, command []string, actualCwd ...string) bool {
	if len(actualCwd) > 0 && !nativeSameCwd(actualCwd[0], r.FixtureRoot) {
		return false
	}
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
	if words[0] == "env" {
		return nativeParentDisplayEnvCommand(words)
	}
	for len(words) > 0 && (strings.HasPrefix(words[0], "AETHER_OUTPUT_MODE=") || strings.HasPrefix(words[0], "AETHER_FORCE_COLOR=")) {
		words = words[1:]
	}
	if len(words) == 0 {
		return false
	}
	switch words[0] {
	case "printenv":
		return len(words) == 3 && words[1] == "CODEX_HOME" && words[2] == "CODEX_THREAD_ID"
	case "python3":
		cwd := r.FixtureRoot
		if len(actualCwd) > 0 {
			cwd = actualCwd[0]
		}
		if len(words) < 3 || len(words) > 5 || !nativeCoordinatorPathMatches(r, cwd, words[1]) {
			return false
		}
		raw, err := r.readEvidence(r.CoordinatorPath)
		if err != nil || lifecycleDigest(raw) != r.CoordinatorSHA256 {
			return false
		}
		qualification := strings.Contains(string(raw), "\nscenario = ")
		if qualification && len(words) > 3 && regexp.MustCompile(`^[0-9]+$`).MatchString(words[len(words)-1]) {
			words = words[:len(words)-1]
		}
		switch words[2] {
		case "manifest", "reserve", "record", "inspect", "stage", "finalize", "empty-result", "context", "context-initial", "context-answers", "context-observe":
			return len(words) == 3
		case "bind":
			if len(words) != 4 {
				return false
			}
			if r.SchemaVersion != "codex-native-tracer/v2" {
				return regexp.MustCompile(`^[A-Za-z0-9-]+$`).MatchString(words[3])
			}
			matches := func(child codexNativeLiveReceipt) bool {
				return child.ChildIdentityCorroborated && child.BoundHostSessionID == r.SessionID &&
					(words[3] == child.ChildID || words[3] == child.ChildTaskPath)
			}
			if matches(*r) {
				return true
			}
			for _, child := range r.Workers {
				if matches(child) {
					return true
				}
			}
			return false
		case "prompt", "release", "summary", "question", "answer", "running", "context-ack", "cancel-requested", "cancelled", "unavailable", "launch-unresolved", "stale-result", "child-mismatch", "resume", "pause":
			return qualification && len(words) == 3
		}
	case "cat":
		return len(words) > 1 // simple words only: no expansion, pipes or redirection
	case "pwd":
		return len(words) == 1
	case "ls":
		return true
	case "sed":
		return len(words) == 4 && words[1] == "-n" && (regexp.MustCompile(`^[0-9]+(?:,[0-9]+)?p$`).MatchString(words[2]) || nativeParentJSONFieldInspection(r, words))
	case "head":
		return len(words) == 4 && words[1] == "-n" && regexp.MustCompile(`^[0-9]+$`).MatchString(words[2]) && !strings.HasPrefix(words[3], "-")
	case "rg":
		if len(words) == 3 && words[1] == "--files" && filepath.IsAbs(words[2]) {
			return true
		}
		// The observed parent queried the fixture's Aether guidance with these
		// three literal document globs. No executable option, expansion, alternate
		// search root, symlink following or extra command is admitted here.
		return len(actualCwd) == 1 && command[2] == `rg "spawn-log" .aether -g '*.md' -g '*.json' -g '*.yaml'`
	case "jq":
		args := words[1:]
		if len(args) > 0 && (args[0] == "-r" || args[0] == "-c" || args[0] == "-S") {
			args = args[1:]
		}
		if len(args) < 2 {
			return false
		}
		for _, arg := range args {
			if strings.HasPrefix(arg, "-") {
				return false
			}
		}
		return true
	case "cmp":
		return len(words) == 3 && !strings.HasPrefix(words[1], "-") && !strings.HasPrefix(words[2], "-")
	case "diff":
		return nativeGapFinalizerComparison(*r, words)
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
		case "resume", "pause":
			return len(words) == 2
		case "codex-native-worker":
			if len(words) != 5 {
				return false
			}
			if words[3] == "--phase" {
				return (words[2] == "inspect" || words[2] == "stage") && words[4] == "1"
			}
			if words[2] != "question" || words[3] != "--request" {
				return false
			}
			raw, err := r.readEvidence(r.CoordinatorPath)
			if err != nil || lifecycleDigest(raw) != r.CoordinatorSHA256 {
				return false
			}
			match := regexp.MustCompile(`(?m)^coord = pathlib.Path\((.+)\)$`).FindStringSubmatch(string(raw))
			if len(match) != 2 {
				return false
			}
			root, err := strconv.Unquote(match[1])
			if err != nil || filepath.Dir(filepath.Clean(words[4])) != root {
				return false
			}
			name := filepath.Base(words[4])
			return name == "bind-request.json" || (strings.Contains(string(raw), "\nscenario = ") && regexp.MustCompile(`^w[0-9]+-bind-request\.json$`).MatchString(name))
		case "status", "pheromones", "command-guide", "ceremony", "spawn-log", "spawn-complete":
			return true
		}
	}
	return false
}

// A literal JSON field range can only inspect the two source-bound manifests.
func nativeParentJSONFieldInspection(r *codexNativeLiveReceipt, words []string) bool {
	if len(words) != 4 || words[0] != "sed" || words[1] != "-n" {
		return false
	}
	match := regexp.MustCompile(`^/"[A-Za-z_][A-Za-z0-9_]{0,63}"/,\+([1-9][0-9]{0,2})p$`).FindStringSubmatch(words[2])
	if len(match) != 2 {
		return false
	}
	count, err := strconv.Atoi(match[1])
	if err != nil || count > 256 || !filepath.IsAbs(r.CoordinationPath) || filepath.Clean(r.CoordinationPath) != r.CoordinationPath {
		return false
	}
	if words[3] != filepath.Join(r.CoordinationPath, "manifest.json") && words[3] != filepath.Join(r.CoordinationPath, "manifest-envelope.json") {
		return false
	}
	raw, err := r.readEvidence(r.CoordinatorPath)
	if err != nil || lifecycleDigest(raw) != r.CoordinatorSHA256 {
		return false
	}
	binding := regexp.MustCompile(`(?m)^coord = pathlib.Path\((.+)\)$`).FindAllStringSubmatch(string(raw), -1)
	if len(binding) != 1 {
		return false
	}
	root, err := strconv.Unquote(binding[0][1])
	return err == nil && root == r.CoordinationPath
}

// Only literal display settings may prefix these existing parent render commands.
// This does not unwrap env for coordinator, lifecycle, child work or test calls.
func nativeParentDisplayEnvCommand(words []string) bool {
	if len(words) == 0 || words[0] != "env" {
		return false
	}
	words = words[1:]
	seen := map[string]bool{}
	for len(words) > 0 && strings.Contains(words[0], "=") {
		key, value, _ := strings.Cut(words[0], "=")
		if seen[key] || !((key == "AETHER_OUTPUT_MODE" && (value == "json" || value == "visual")) || (key == "AETHER_FORCE_COLOR" && (value == "0" || value == "1"))) {
			return false
		}
		seen[key] = true
		words = words[1:]
	}
	if len(seen) == 0 || len(words) < 2 || words[0] != "aether" {
		return false
	}
	if words[1] == "status" {
		return len(words) == 2
	}
	if len(words) < 7 || words[1] != "ceremony" || words[3] != "--workflow" || words[4] != "build" || words[6] == "" || strings.HasPrefix(words[6], "-") {
		return false
	}
	switch words[2] {
	case "spawn-plan":
		return len(words) == 7 && words[5] == "--manifest-file"
	case "wave-start":
		return len(words) == 9 && words[5] == "--manifest-file" && words[7] == "--execution-wave" && regexp.MustCompile(`^[0-9]+$`).MatchString(words[8])
	case "worker-complete":
		return len(words) == 7 && words[5] == "--worker-file"
	case "closeout":
		return len(words) == 7 && words[5] == "--completion-file"
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

// Exclusion globs only remove matches from the existing cwd-only listing.
// A leading slash anchors a glob; it is not another filesystem search root.
func nativeFixtureExclusionFilter(filter string) bool {
	if len(filter) < 2 || len(filter) > 129 || !regexp.MustCompile(`^!/?[A-Za-z0-9_.*?-]+(?:/[A-Za-z0-9_.*?-]+)*$`).MatchString(filter) {
		return false
	}
	for _, component := range strings.Split(strings.TrimPrefix(filter[1:], "/"), "/") {
		if component == "." || component == ".." {
			return false
		}
	}
	return true
}

// Listing-only basename patterns; positive globs may override ignore selection,
// with no new roots or hidden/follow/output options.
func nativeFixturePositiveFilter(filter string) bool {
	return regexp.MustCompile(`^[A-Za-z0-9_.*?][A-Za-z0-9_.*?-]{0,127}$`).MatchString(filter) && filter != "." && filter != ".." && !strings.Contains(filter, "**")
}

func nativeLongDateWords(words []string) bool {
	if len(words) != 2 || words[0] != "date" {
		return false
	}
	switch words[1] {
	case "--rfc-3339=date", "--rfc-3339=seconds", "--rfc-3339=ns",
		"--iso-8601=date", "--iso-8601=hours", "--iso-8601=minutes", "--iso-8601=seconds", "--iso-8601=ns":
		return true
	}
	return false
}

func nativeAdditionalFixtureInspection(raw string, words []string, allowed func(string) bool) bool {
	if len(words) == 3 && words[0] == "git" && words[1] == "diff" && words[2] == "--check" {
		return true
	}
	// The exact bare listing stays in the separately verified fixture cwd.
	if len(words) == 2 && words[0] == "rg" && words[1] == "--files" {
		return true
	}
	// Explicit fixture basenames, quoted positive basenames and exclusion paths.
	// Both new pattern classes stay listing-only with the same bounded -g argv.
	if len(words) >= 4 && len(words) <= 14 && len(words)%2 == 0 && words[0] == "rg" && words[1] == "--files" {
		seen := map[string]bool{}
		literals := regexp.MustCompile(`(?:^|\s)-g\s+('[^']*'|"[^"]*"|[^\s]+)`).FindAllStringSubmatch(raw, -1)
		for i := 2; i < len(words); i += 2 {
			filter := words[i+1]
			fixtureFilter := allowed(filter) && filepath.Base(filter) == filter
			pattern := nativeFixtureExclusionFilter(filter) || nativeFixturePositiveFilter(filter)
			if pattern {
				// Word decoding strips quotes; require the complete raw argument
				// to be quoted so wildcard expansion cannot change its meaning.
				index := (i - 2) / 2
				pattern = len(literals) == (len(words)-2)/2 &&
					(literals[index][1] == "'"+filter+"'" || literals[index][1] == `"`+filter+`"`)
			}
			if words[i] != "-g" || (!fixtureFilter && !pattern) || seen[filter] {
				return false
			}
			seen[filter] = true
		}
		return true
	}
	if len(words) >= 4 && len(words) <= 9 && words[0] == "sed" && words[1] == "-n" && regexp.MustCompile(`^[0-9]{1,6}(?:,[0-9]{1,6})?p$`).MatchString(words[2]) {
		seen := map[string]bool{}
		for _, path := range words[3:] {
			// Validate the complete path before deduplicating fixture aliases.
			name := filepath.Base(path)
			if !allowed(path) || seen[name] {
				return false
			}
			seen[name] = true
		}
		return true
	}
	if len(words) == 3 && ((words[0] == "gofmt" && words[1] == "-d") || (words[0] == "test" && words[1] == "-r")) {
		return allowed(words[2])
	}
	return len(words) == 5 && words[0] == "git" && words[1] == "diff" && words[2] == "--check" && words[3] == "--" && allowed(words[4])
}

func nativeAdditionalFixtureCheck(words []string) bool {
	joined := strings.Join(words, " ")
	return joined == "go test ./..." || joined == "go test ./... -cover" || joined == "go test ./... -cover -count=1" || joined == "go test -cover ./..."
}

// A single child-owned inspection atom. Standalone commands and composed
// inspections share this exact predicate; parent discovery allowances stay out.
func nativeChildInspectionAtom(r codexNativeLiveReceipt, raw string) bool {
	words, ok := nativeSimpleShellWords(raw)
	return ok && nativeChildInspectionWords(r, raw, words)
}

func nativeChildInspectionWords(r codexNativeLiveReceipt, raw string, words []string) bool {
	allowed := func(path string) bool {
		if filepath.IsAbs(path) {
			if !nativeSameCwd(filepath.Dir(path), r.FixtureRoot) {
				return false
			}
			path = filepath.Base(path)
		}
		for _, name := range []string{"AGENTS.md", "clamp.go", "clamp_test.go", "double.go", "double_test.go", "go.mod"} {
			if path == name {
				return true
			}
		}
		return false
	}
	if len(words) == 0 {
		return false
	}
	// Exact formatting/listing operations do not create test proof.
	if len(words) == 3 && words[0] == "git" && words[1] == "diff" && words[2] == "--stat" {
		return true
	}
	// Preserve argv boundaries: 'git status' is one executable name, not git
	// followed by its status subcommand. Quoting individual words is harmless.
	if (len(words) == 1 && words[0] == "pwd") ||
		(len(words) == 3 && words[0] == "git" && words[1] == "status" && words[2] == "--short") ||
		(len(words) == 3 && words[0] == "date" && words[1] == "-u" && words[2] == "+%Y-%m-%dT%H:%M:%SZ") ||
		(len(words) == 2 && words[0] == "date" && words[1] == "-Iseconds") || nativeLongDateWords(words) ||
		nativeAdditionalFixtureInspection(raw, words, allowed) {
		return true
	}
	paths := []string(nil)
	if words[0] == "cat" && len(words) > 1 {
		paths = words[1:]
	}
	if len(words) > 3 && words[0] == "git" && words[1] == "diff" && words[2] == "--" {
		paths = words[3:]
	}
	if len(paths) == 0 {
		return false
	}
	for _, path := range paths {
		if !allowed(path) {
			return false
		}
	}
	return true
}

// Shared chain predicate: only existing child-owned read-only atoms.
func nativeChildInspectionChain(r codexNativeLiveReceipt, command string) bool {
	// One invocation, at most eight inspection atoms. Atom decoding rejects
	// mixed operators, expansion, writes, context calls and tests in a chain.
	separator := " && "
	if strings.Contains(command, ";") {
		separator = ";"
	}
	if parts := strings.Split(command, separator); len(parts) > 1 {
		if len(parts) > 8 {
			return false
		}
		for _, part := range parts {
			if !nativeChildInspectionAtom(r, part) {
				return false
			}
		}
		return true
	}
	return false
}

func nativeChildCommandAllowed(r codexNativeLiveReceipt, command []string) bool {
	if len(command) != 3 || (command[1] != "-c" && command[1] != "-lc") {
		return false
	}
	if strings.Contains(command[2], " && ") || strings.Contains(command[2], ";") {
		return nativeChildInspectionChain(r, command[2])
	}
	if nativeChildInspectionAtom(r, command[2]) {
		return true
	}
	words, ok := nativeSimpleShellWords(command[2])
	if !ok || len(words) == 0 {
		return false
	}
	if strings.HasPrefix(words[0], "GOCACHE=/") {
		words = words[1:]
	}
	if len(words) == 0 {
		return false
	}
	// Preserve the prior standalone cache-prefix interpretation. Composed
	// atoms never strip environment assignments. Raw glob quotes stay intact.
	if nativeChildInspectionWords(r, command[2], words) {
		return true
	}
	if nativeFixtureSmoke(words) || nativeFixtureInspectionExtra(words) || nativeChildFetchCommandAllowed(r, words) || nativeAssignedFixtureCommand(r, command) {
		return true
	}
	joined := strings.Join(words, " ")
	return joined == "go build ./..." || nativeAdditionalFixtureCheck(words) || joined == "go vet ./..."
}

func nativeFinalizationReplayEvidence(r codexNativeLiveReceipt, coord string) bool {
	if r.RecoveryCheckpoint != nil {
		a, e1 := r.readEvidence(filepath.Join(coord, "post-finalize-1-attempt.json"))
		b, e2 := r.readEvidence(filepath.Join(coord, "post-finalize-2-attempt.json"))
		if e1 != nil || e2 != nil || len(a) == 0 || !bytes.Equal(a, b) {
			return false
		}
	}
	first, e1 := r.readEvidence(filepath.Join(coord, "post-finalize-1-state.json"))
	second, e2 := r.readEvidence(filepath.Join(coord, "post-finalize-2-state.json"))
	if e1 != nil || e2 != nil || len(first) == 0 || !bytes.Equal(first, second) {
		return false
	}
	type final struct {
		OK     bool
		Result struct {
			Idempotent *bool
			Attempt    string
			Selected   []string `json:"selected_tasks"`
		}
	}
	var a, b final
	x, e3 := r.readEvidence(filepath.Join(coord, "finalize-1.stdout.json"))
	y, e4 := r.readEvidence(filepath.Join(coord, "finalize-2.stdout.json"))
	if e3 != nil || e4 != nil || json.Unmarshal(x, &a) != nil || json.Unmarshal(y, &b) != nil || !a.OK || !b.OK || a.Result.Idempotent == nil || *a.Result.Idempotent || b.Result.Idempotent == nil || !*b.Result.Idempotent {
		return false
	}
	// Both real finalizer responses must name the same retained attempt.
	resolve := func(path string) string {
		if filepath.IsAbs(path) {
			return filepath.Clean(path)
		}
		return filepath.Join(r.FixtureRoot, path)
	}
	return a.Result.Attempt != "" && resolve(a.Result.Attempt) == filepath.Clean(r.AttemptPath) && a.Result.Attempt == b.Result.Attempt
}

func nativeRefusalInventory(r codexNativeLiveReceipt, path, reason string) error {
	var fact struct {
		Exit          int `json:"exit_status"`
		Stderr        string
		Before, After map[string]string
	}
	raw, err := r.readEvidence(path)
	if err != nil || json.Unmarshal(raw, &fact) != nil || fact.Exit == 0 || len(fact.Before) == 0 || len(fact.After) == 0 {
		return fmt.Errorf("required refusal inventory/result missing or failed: %s", filepath.Base(path))
	}
	before, _ := json.Marshal(fact.Before)
	after, _ := json.Marshal(fact.After)
	var envelope struct {
		OK    bool
		Error string
	}
	if !bytes.Equal(before, after) || json.Unmarshal([]byte(fact.Stderr), &envelope) != nil || envelope.OK || !strings.Contains(envelope.Error, reason) {
		return fmt.Errorf("refusal inventory changed or rejection unrelated: %s", filepath.Base(path))
	}
	return nil
}

// Replay derives only from retained immutable inputs. It never starts a host,
// runs worker checks, rewrites a result, or changes the original receipt.
func nativeReplayQualificationReceipt(t *testing.T, r *codexNativeLiveReceipt) error {
	t.Helper()
	prospective, err := nativeGapProofIdentity(r.ProofContract, r.ProofAmendmentSHA256)
	if err != nil {
		return err
	}
	if err := nativeBeginReceiptReplay(r); err != nil {
		return err
	}
	root := filepath.Dir(r.FixtureRoot)
	home := filepath.Join(root, "home")
	coord := filepath.Join(root, "coordination")
	r.Outcome, r.Reason = "incomplete", ""
	r.Limitations = nil
	if r.Scenario == "claude" {
		raw, err := r.readEvidence(r.RawEvents)
		if err != nil {
			return err
		}
		if prospective {
			if err := nativeValidateClaudeInvocationIdentity(*r, raw); err != nil {
				return err
			}
		}
		resetCodexNativeDerivedEvidence(r)
		source, _ := r.readEvidence(filepath.Join(r.FixtureRoot, "clamp.go"))
		r.FinalSource = string(source)
		state, err := r.readEvidence(filepath.Join(root, "claude-after-state.json"))
		if err != nil {
			return err
		}
		nativeCollectClaudeEvidence(r, raw, state)
		r.Limitations = []string{"Actual Claude capture only; installed wrapper/Builder/finalizer proof is required independently. No full platform parity or live planning is claimed."}
		if err := nativeValidateFixtureBaselines(*r); err != nil {
			return err
		}
		if r.ExitStatus != 0 || r.NativeSpawnCount != 1 || !r.SkillRead || !r.ChildEditObserved || !r.ChecksPassed || !r.CreditObserved || r.ParentSubstitution || len(r.ChildUnclassified) != 0 {
			return fmt.Errorf("actual Claude installed workflow evidence incomplete")
		}
	} else {
		if r.Scenario == "early-resume" {
			nativeCollectResumeEvidence(t, r, t.TempDir(), home, coord)
		} else {
			nativeCollectLiveEvidence(t, r, t.TempDir(), home)
		}
		if r.Scenario == "cancellation" {
			evidence := nativeRetainedCancellationEvidence(*r, root, home)
			r.Cancellation = &evidence
		}
		switch r.Scenario {
		case "ordinary", "early-resume":
			if r.Scenario == "ordinary" || r.Scenario == "early-resume" {
				r.FinalizationReplayStable = nativeFinalizationReplayEvidence(*r, coord)
			}
			if err := validateCodexNativeLiveReceipt(*r); err != nil {
				return err
			}
		case "controls", "controls-read-only", "missing-skill":
			if err := nativeCollectHostControlEvidence(r, root, home); err != nil {
				return err
			}
			r.Outcome = "observed"
		case "review", "question", "partial-resume", "cancellation", "spawn-gap":
			if r.Scenario == "partial-resume" || r.Scenario == "spawn-gap" {
				nativeReplayFreshParent(r, home)
			}
			if r.Scenario == "question" {
				r.Assertions["scoped_answer_behavior"] = nativeRetainedQuestionCheck(*r, root)
				r.Limitations = []string{"Actual host send/child linkage can be observed; encrypted message exports do not independently corroborate the full envelope plaintext."}
				if r.ContextProtocol == codexNativeContextProtocolChildFetch {
					r.Limitations = []string{"Full context is proved through actual child read and separate ACK; encrypted native message bodies remain unavailable."}
				}
			}
			r.FinalizationReplayStable = nativeFinalizationReplayEvidence(*r, coord)
			if err := validateCodexNativeQualificationScenario(*r, root); err != nil {
				return err
			}
			if r.Scenario == "cancellation" || r.Scenario == "spawn-gap" {
				r.Outcome = "observed"
				r.Limitations = []string{"Saved work is incomplete and uncredited. Actual interruption is only pending; no cancelled terminal or replacement authority is inferred."}
				if r.Cancellation != nil && r.Cancellation.Disposition == nativeGapCancellationRefused {
					r.Limitations = []string{"Actual runtime guards refused unsafe completion and redispatch; work remains incomplete and uncredited. Worker termination and absence of later writes remain unproved."}
				}
			}
		default:
			return fmt.Errorf("unknown matrix scenario %q", r.Scenario)
		}
	}
	for path, want := range map[string]string{r.CandidatePath: r.CandidateSHA256, r.ClientPath: r.ClientSHA256, r.CoordinatorPath: r.CoordinatorSHA256} {
		if path == "" && r.Scenario == "claude" {
			continue
		}
		raw, err := r.readEvidence(path)
		if err != nil || want == "" || lifecycleDigest(raw) != want {
			return fmt.Errorf("retained qualification input changed: %s", path)
		}
	}
	if r.Outcome == "incomplete" {
		r.Outcome = "passed"
	}
	return nil
}
func nativeReplayFreshParent(r *codexNativeLiveReceipt, home string) {
	r.ResumeSessionID = ""
	r.ResumeNoSpawn, r.ResumeInspectObserved, r.ResumeWorkerStable = false, false, false
	raw, err := r.readEvidence(r.ResumeRawEvents)
	if err != nil {
		return
	}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var e struct {
			Type     string
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(line, &e) == nil && e.Type == "thread.started" {
			r.ResumeSessionID = e.ThreadID
		}
	}
	if r.ResumeSessionID == "" || r.ResumeSessionID == r.SessionID {
		return
	}
	paths, _ := filepath.Glob(filepath.Join(home, ".codex", "sessions", "*", "*", "*", "*"+r.ResumeSessionID+".jsonl"))
	if len(paths) != 1 {
		return
	}
	events, err := r.readEvidence(paths[0])
	if err != nil {
		return
	}
	resumed := *r
	resumed.SessionID = r.ResumeSessionID
	resumed.NativeSpawnCount = 0
	resumed.ParentSubstitution = false
	resumed.ParentUnclassified = nil
	resumed.ResumeInspectObserved = false
	nativeInspectParentEvents(&resumed, events)
	r.ParentSubstitution = r.ParentSubstitution || resumed.ParentSubstitution
	r.ParentUnclassified = append(r.ParentUnclassified, resumed.ParentUnclassified...)
	r.ResumeNoSpawn = resumed.NativeSpawnCount == 0
	r.ResumeInspectObserved = resumed.ResumeInspectObserved
	before, err := r.readEvidence(r.BeforeResumeJournalPath)
	if err != nil || r.BeforeResumeJournalSHA256 == "" || lifecycleDigest(before) != r.BeforeResumeJournalSHA256 {
		return
	}
	after, err := r.readEvidence(r.AttemptPath)
	if err != nil {
		return
	}
	if r.Scenario == "spawn-gap" {
		r.ResumeWorkerStable = bytes.Equal(before, after)
		r.Assertions["fresh_public_resume"] = nativePublicResumeProof(*r, raw)
		return
	}
	var a, b buildAttemptRecord
	if json.Unmarshal(before, &a) != nil || json.Unmarshal(after, &b) != nil || a.CompletionPath != "" || len(a.WorkerRuns) != 1 || len(b.WorkerRuns) != 2 || len(r.Workers) != 2 {
		return
	}
	aw, _ := json.Marshal(a.WorkerRuns[0])
	bw, _ := json.Marshal(b.WorkerRuns[0])
	r.ResumeWorkerStable = a.ID == b.ID && a.RunID == b.RunID && bytes.Equal(aw, bw) && r.Workers[0].ChildEditObserved
	r.Assertions["fresh_public_resume"] = nativePublicResumeProof(*r, raw)
	r.Assertions["resume_one_new_helper"] = resumed.NativeSpawnCount == 1
	r.ResumeWorkerStable = r.ResumeWorkerStable && nativeGapFinishedWorkerStable(*r)
	r.Assertions["finished_worker_unchanged"] = r.ResumeWorkerStable
}
func nativeRetainedQuestionCheck(r codexNativeLiveReceipt, root string) bool {
	dir := filepath.Join(root, "answer-behavior-check")
	var check struct {
		Argv       []string
		Cwd        string
		Passed     bool
		Unchanged  bool `json:"fixture_inventory_unchanged"`
		Provenance string
	}
	raw, err := r.readEvidence(filepath.Join(dir, "check.json"))
	if err != nil || json.Unmarshal(raw, &check) != nil || !check.Passed || !check.Unchanged || check.Cwd != dir || strings.Join(check.Argv, " ") != "go run ." || check.Provenance != "harness verifies actual child-authored answer behavior; not child test execution" {
		return false
	}
	out, err := r.readEvidence(filepath.Join(dir, "check-output.txt"))
	// These source/input artifacts must have been captured by the actual harness.
	for _, name := range []string{"check.json", "check-output.txt", "go.mod", "main.go"} {
		path := filepath.Join(dir, name)
		data, err := r.readEvidence(path)
		if err != nil || r.Artifacts[path] == "" || lifecycleDigest(data) != r.Artifacts[path] {
			return false
		}
	}
	return err == nil && strings.TrimSpace(string(out)) == "FIXTURE_SCOPED_REVERSED_BOUNDS_PASS"
}

func nativeValidateEmptyRefusal(r codexNativeLiveReceipt, coord string) error {
	if !nativeObservedFixtureOperation(r, r, 0, "empty-result", true) {
		return fmt.Errorf("actual source-linked empty-result refusal missing")
	}
	if err := nativeRefusalInventory(r, filepath.Join(coord, "empty-result-refusal.json"), "nonempty terminal result"); err != nil {
		return err
	}
	load := func(name string) (map[string]any, error) {
		raw, err := r.readEvidence(filepath.Join(coord, name))
		var v map[string]any
		if err == nil {
			err = json.Unmarshal(raw, &v)
		}
		return v, err
	}
	bind, err := load("bind-request.json")
	if err != nil {
		return err
	}
	binding, ok := bind["execution_binding"].(map[string]any)
	if !ok || binding["attempt_id"] != r.AttemptID || bind["child_id"] != r.ChildID || bind["launch_id"] != r.LaunchID {
		return fmt.Errorf("empty refusal bind identity mismatch")
	}
	request, err := load("empty-result-request.json")
	if err != nil {
		return err
	}
	result, ok := request["result"].(map[string]any)
	if !ok || len(result) != 0 {
		return fmt.Errorf("empty refusal request is not empty")
	}
	delete(request, "result")
	a, _ := json.Marshal(request)
	b, _ := json.Marshal(bind)
	if !bytes.Equal(a, b) {
		return fmt.Errorf("empty refusal request changed unrelated binding")
	}
	return nil
}
func nativeFixtureShellWords(command []string) ([]string, bool) {
	if len(command) != 3 || (command[1] != "-c" && command[1] != "-lc") {
		return nil, false
	}
	words, ok := nativeSimpleShellWords(command[2])
	if !ok {
		return nil, false
	}
	if len(words) > 0 && strings.HasPrefix(words[0], "GOCACHE=/") {
		words = words[1:]
	}
	return words, len(words) > 0
}

type nativeRecordedShellCommand struct{ Command, Cwd string }

func nativeCorroboratedBatch(raw []byte, thread, callID string, commands []nativeRecordedShellCommand, requireSuccess ...bool) bool {
	if callID == "" || len(commands) == 0 {
		return false
	}
	wanted := map[string]bool{}
	for _, c := range commands {
		key := c.Cwd + "\x00" + c.Command
		if wanted[key] {
			return false
		}
		wanted[key] = true
	}
	seen := map[string]bool{}
	eventIDs := map[string]int{}
	selectedIDs := []string{}
	turn := ""
	active, complete := false, false
	calls, outputs := 0, 0
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var w nativeCodeModeWire
		_ = json.Unmarshal(line, &w)
		p := w.Payload
		if w.Type == "response_item" && p.Type == "custom_tool_call" && p.CallID == callID {
			calls++
			turn = p.Metadata.TurnID
			if turn == "" {
				return false
			}
			active = true
		} else if active && w.Type == "response_item" && (p.Type == "custom_tool_call" || p.Type == "function_call") {
			return false
		}
		if w.Type == "response_item" && p.Type == "custom_tool_call_output" && p.CallID == callID {
			outputs++
			complete = active && p.Metadata.TurnID == turn && len(seen) == len(wanted) && len(p.Output) > 0 && strings.HasPrefix(p.Output[0].Text, "Script completed\n")
			active = false
		}
		var e nativeHostEvent
		if json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" || e.Payload.Item.Type != "CommandExecution" {
			continue
		}
		i := e.Payload.Item
		eventIDs[i.ID]++
		if !active {
			continue
		}
		if e.Payload.ThreadID != thread || e.Payload.TurnID != turn || i.ID == "" || i.ExitCode == nil || (i.Status != "completed" && !(i.Status == "failed" && *i.ExitCode != 0)) || len(i.Command) != 3 || (i.Command[1] != "-lc" && i.Command[1] != "-c") {
			return false
		}
		// Failed parent refusal controls remain valid observations. Child
		// inspection chains can preserve check credit only after success.
		if len(requireSuccess) > 0 && requireSuccess[0] && (i.Status != "completed" || *i.ExitCode != 0) {
			return false
		}
		key := ""
		for _, c := range commands {
			if nativeSameCwd(i.Cwd, c.Cwd) && i.Command[2] == c.Command {
				if key != "" {
					return false
				}
				key = c.Cwd + "\x00" + c.Command
			}
		}
		if !wanted[key] || seen[key] {
			return false
		}
		seen[key] = true
		selectedIDs = append(selectedIDs, i.ID)
	}
	for _, id := range selectedIDs {
		if eventIDs[id] != 1 {
			return false
		}
	}
	return calls == 1 && outputs == 1 && complete
}

// An exact literal patch wrapper is classifiable only when its one successful
// FileChange lies between that unique call and output in the same child turn.
// Ownership, diff reconstruction and fresh checks are still enforced separately.
func nativeCorroboratedLiteralPatch(r codexNativeLiveReceipt, raw []byte, callID string) bool {
	if callID == "" {
		return false
	}
	turn, target := "", ""
	active, validOutput := false, false
	calls, outputs, changes := 0, 0, 0
	changeIDs := map[string]int{}
	selectedChange := ""
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var w nativeCodeModeWire
		_ = json.Unmarshal(line, &w)
		p := w.Payload
		if w.Type == "response_item" && p.Type == "custom_tool_call" && p.CallID == callID {
			calls++
			patch, literal := nativeCodeModeLiteralPatch(p.Input)
			if !literal || (p.Name != "exec" && !strings.HasSuffix(p.Name, ".exec")) {
				return false
			}
			if !strings.HasPrefix(patch, "*** Begin Patch\n*** Update File: ") || !strings.HasSuffix(patch, "\n*** End Patch") {
				return false
			}
			lines := strings.Split(patch, "\n")
			target = strings.TrimPrefix(lines[1], "*** Update File: ")
			if !filepath.IsAbs(target) {
				target = filepath.Join(r.FixtureRoot, target)
			}
			file := r.SourceFile
			if file == "" {
				file = "clamp.go"
			}
			if filepath.Clean(target) != filepath.Join(r.FixtureRoot, file) {
				return false
			}
			for _, line := range lines[2 : len(lines)-1] {
				if strings.HasPrefix(line, "*** ") {
					return false
				}
			}
			turn = p.Metadata.TurnID
			if turn == "" {
				return false
			}
			active = true
		} else if active && w.Type == "response_item" && (p.Type == "custom_tool_call" || p.Type == "function_call") {
			return false
		}
		if w.Type == "response_item" && p.Type == "custom_tool_call_output" && p.CallID == callID {
			outputs++
			validOutput = active && changes == 1 && p.Metadata.TurnID == turn && len(p.Output) == 2 && strings.HasPrefix(p.Output[0].Text, "Script completed\n") && p.Output[1].Text == "{}"
			active = false
		}
		var e nativeHostEvent
		if json.Unmarshal(line, &e) != nil || e.Type != "event_msg" || e.Payload.Type != "item_completed" || e.Payload.Item.Type != "FileChange" {
			continue
		}
		i := e.Payload.Item
		changeIDs[i.ID]++
		if active {
			if e.Payload.ThreadID != r.ChildID || e.Payload.TurnID != turn || i.Status != "completed" || i.ID == "" || len(i.Changes) != 1 {
				return false
			}
			change, ok := i.Changes[target]
			if !ok || change.Type != "update" || change.MovePath != nil {
				return false
			}
			changes++
			selectedChange = i.ID
		}
	}
	return calls == 1 && outputs == 1 && changes == 1 && validOutput && changeIDs[selectedChange] == 1
}

// Admit only a JSON string literal, directly passed or stored in one const.
// The surrounding call/FileChange/output and owned target checks remain required.
func nativeCodeModeLiteralPatch(input string) (string, bool) {
	literal := ""
	direct := regexp.MustCompile(`^\s*text\(await\s+tools\.apply_patch\(("(?:[^"\\]|\\.)*")\)\);\s*$`)
	assigned := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*("(?:[^"\\]|\\.)*");\s*text\(await\s+tools\.apply_patch\(([A-Za-z_][A-Za-z0-9_]*)\)\);\s*$`)
	result := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*("(?:[^"\\]|\\.)*");\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.apply_patch\(([A-Za-z_][A-Za-z0-9_]*)\);\s*text\(([A-Za-z_][A-Za-z0-9_]*)\);\s*$`)
	if match := direct.FindStringSubmatch(input); len(match) == 2 {
		literal = match[1]
	} else if match := assigned.FindStringSubmatch(input); len(match) == 4 && match[1] == match[3] {
		literal = match[2]
	} else if match := result.FindStringSubmatch(input); len(match) == 6 && match[1] == match[4] && match[3] == match[5] && match[1] != match[3] {
		literal = match[2]
	} else {
		return "", false
	}
	var patch string
	if json.Unmarshal([]byte(literal), &patch) != nil {
		return "", false
	}
	return patch, true
}

func nativeCodeModeCommands(input, defaultCwd string) ([]nativeRecordedShellCommand, bool) {
	if projected, ok := nativeCodeModeExitOutputProjection(input, defaultCwd); ok {
		return []nativeRecordedShellCommand{projected}, true
	}
	if projected, ok := nativeCodeModeOutputProjection(input, defaultCwd); ok {
		// Complete exit annotations must use the shared presentation plan above.
		// Keep the old decoder itself unchanged for strict context/test paths.
		if !nativeCodeModePlainOutput(input, defaultCwd) && !regexp.MustCompile(`text\(JSON\.stringify\([A-Za-z_][A-Za-z0-9_]*\)\);?\s*$`).MatchString(input) {
			return nil, false
		}
		return []nativeRecordedShellCommand{projected}, true
	}
	if strings.Contains(input, "Promise.all") {
		return nativeReadOnlyBatchCommands(input, defaultCwd)
	}
	// Only literal command objects and direct printing of their untouched result.
	// Multiple calls are allowed only as a full sequence of this same grammar.
	// A missing final semicolon is admitted only at EOF, never between calls.
	var result []nativeRecordedShellCommand
	direct := regexp.MustCompile(`^\s*text\(await\s+tools\.exec_command\(`)
	assigned := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\(`)
	for strings.TrimSpace(input) != "" {
		prefix, name := 0, ""
		if m := direct.FindStringIndex(input); m != nil {
			prefix = m[1]
		} else if m := assigned.FindStringSubmatchIndex(input); m != nil {
			prefix = m[1]
			name = input[m[2]:m[3]]
		} else {
			return nil, false
		}
		// Find the literal object's end without treating braces or apparent calls
		// inside quoted JSON/shell payloads as JavaScript structure.
		literal := nativeCommandLiteral{input: input[prefix:]}
		if !literal.take('{') {
			return nil, false
		}
		closed := false
		for literal.pos < len(literal.input) {
			c := literal.input[literal.pos]
			if c == '\'' || c == '"' {
				if _, ok := literal.quoted(); !ok {
					return nil, false
				}
				continue
			}
			literal.pos++
			if c == '{' {
				return nil, false
			}
			if c == '}' {
				closed = true
				break
			}
		}
		if !closed {
			return nil, false
		}
		object := input[prefix : prefix+literal.pos]
		suffix := `^\)\)(?:;\s*|\s*$)`
		if name != "" {
			suffix = `^\);\s*text\(` + regexp.QuoteMeta(name) + `\)(?:;\s*|\s*$)`
		}
		m := regexp.MustCompile(suffix).FindStringIndex(input[prefix+literal.pos:])
		if m == nil {
			return nil, false
		}
		end := prefix + literal.pos + m[1]
		command, ok := nativeLiteralCommandObject(object, defaultCwd)
		if !ok {
			return nil, false
		}
		result = append(result, command)
		if len(result) > 64 {
			return nil, false
		}
		input = input[end:]
	}
	return result, len(result) > 0
}

// A complete presentation contains the untouched output and numeric exit once
// each. One combined template or two separate text calls may place them in
// either order. Labels are bounded literal annotations, never evidence values.
type nativeExitOutputPresentation struct {
	Label                                 string
	OutputFirst, Separate, LeadingNewline bool
	JSON                                  bool
}

func (p nativeExitOutputPresentation) render(exit int, output string) []string {
	if p.JSON {
		encoded, _ := json.Marshal(struct {
			Exit   int    `json:"exit_code"`
			Output string `json:"output"`
		}{exit, output})
		return []string{string(encoded)}
	}
	annotation := p.Label + "=" + strconv.Itoa(exit)
	if p.Separate {
		if p.LeadingNewline {
			annotation = "\n" + annotation
		}
		if p.OutputFirst {
			return []string{output, annotation}
		}
		return []string{annotation, output}
	}
	if p.OutputFirst {
		return []string{output + "\n" + annotation}
	}
	return []string{annotation + "\n" + output}
}

func nativeCodeModePresentation(input, defaultCwd string) (nativeRecordedShellCommand, nativeExitOutputPresentation, bool) {
	var plan nativeExitOutputPresentation
	pattern := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*([\s\S]*)$`)
	match := pattern.FindStringSubmatch(input)
	reserved := " JSON Promise tools text await break case catch class const continue debugger default delete do else enum export extends false finally for function if import in instanceof let new null return super switch this throw true try typeof var void while with yield implements interface package private protected public static eval arguments "
	if len(match) != 4 || strings.Contains(reserved, " "+match[1]+" ") {
		return nativeRecordedShellCommand{}, plan, false
	}
	command, ok := nativeLiteralCommandObject(match[2], defaultCwd)
	if !ok {
		return command, plan, false
	}
	// Exactly two direct properties from the same bound result, in either
	// literal object order. This is presentation only, not a test decoder.
	name := regexp.QuoteMeta(match[1])
	exitField := `exit_code\s*:\s*` + name + `\.exit_code`
	outputField := `output\s*:\s*` + name + `\.output`
	jsonPrint := regexp.MustCompile(`^\s*text\(JSON\.stringify\(\{\s*(?:` + exitField + `\s*,\s*` + outputField + `|` + outputField + `\s*,\s*` + exitField + `)\s*\}\)\);?\s*$`)
	if jsonPrint.MatchString(match[3]) {
		plan.JSON = true
		return command, plan, true
	}
	// Only these two fields and literal safe labels are admitted. No generic
	// template evaluation, extra printing, property access or transformations.
	label := `([A-Za-z_][A-Za-z0-9_ -]{0,31})`
	output := regexp.QuoteMeta("${" + match[1] + ".output}")
	exit := regexp.QuoteMeta("${" + match[1] + ".exit_code}")
	textOutput := regexp.QuoteMeta("text(" + match[1] + ".output)")
	annotation := label + `=` + exit
	template := func(body string) string { return regexp.QuoteMeta("text(`") + body + regexp.QuoteMeta("`)") }
	for _, separate := range []bool{false, true} {
		for _, outputFirst := range []bool{false, true} {
			body := ""
			if separate {
				printExit := template(`(\\n)?` + annotation)
				body = printExit + `;\s*` + textOutput
				if outputFirst {
					body = textOutput + `;\s*` + printExit
				}
			} else {
				body = template(annotation + `\\n` + output)
				if outputFirst {
					body = template(output + `\\n` + annotation)
				}
			}
			m := regexp.MustCompile(`^\s*` + body + `;?\s*$`).FindStringSubmatch(match[3])
			if m == nil {
				continue
			}
			plan.OutputFirst, plan.Separate = outputFirst, separate
			if separate {
				plan.LeadingNewline, plan.Label = m[1] != "", m[2]
			} else {
				plan.Label = m[1]
			}
			return command, plan, true
		}
	}
	return command, plan, false
}

func nativeCodeModeExitOutputProjection(input, defaultCwd string) (nativeRecordedShellCommand, bool) {
	command, _, ok := nativeCodeModePresentation(input, defaultCwd)
	return command, ok
}

func nativeExitOutputJSON(text string) (int, string, bool) {
	decoder := json.NewDecoder(strings.NewReader(text))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return 0, "", false
	}
	seen := map[string]bool{}
	var exit int
	var output string
	for decoder.More() {
		token, err := decoder.Token()
		key, ok := token.(string)
		if err != nil || !ok || seen[key] || (key != "exit_code" && key != "output") {
			return 0, "", false
		}
		seen[key] = true
		var value json.RawMessage
		if decoder.Decode(&value) != nil || len(value) == 0 {
			return 0, "", false
		}
		if key == "exit_code" {
			if string(value) == "null" || json.Unmarshal(value, &exit) != nil {
				return 0, "", false
			}
		} else if value[0] != '"' || json.Unmarshal(value, &output) != nil {
			return 0, "", false
		}
	}
	if token, err := decoder.Token(); err != nil || token != json.Delim('}') || len(seen) != 2 {
		return 0, "", false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return 0, "", false
	}
	return exit, output, true
}

// Called only after nativeCorroboratedBatch proves the unique same-thread/turn
// invocation and completed outer output. Compare every printed block in source
// order to that exact event, including genuine failed CLI refusals.
func nativeCorroboratedExitOutput(raw []byte, callID, defaultCwd string) bool {
	active, observed := false, false
	var plan nativeExitOutputPresentation
	var want []string
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var w nativeCodeModeWire
		_ = json.Unmarshal(line, &w)
		p := w.Payload
		if w.Type == "response_item" && p.Type == "custom_tool_call" && p.CallID == callID {
			var ok bool
			_, plan, ok = nativeCodeModePresentation(p.Input, defaultCwd)
			if active || !ok {
				return false
			}
			active = true
		}
		if active && w.Type == "response_item" && p.Type == "custom_tool_call_output" && p.CallID == callID {
			if !observed || len(p.Output) != len(want)+1 || p.Output[0].Type != "input_text" {
				return false
			}
			for index, expected := range want {
				if p.Output[index+1].Type != "input_text" {
					return false
				}
				if plan.JSON {
					gotExit, gotOutput, valid := nativeExitOutputJSON(p.Output[index+1].Text)
					wantExit, wantOutput, _ := nativeExitOutputJSON(expected)
					if !valid || gotExit != wantExit || gotOutput != wantOutput {
						return false
					}
				} else if p.Output[index+1].Text != expected {
					return false
				}
			}
			return true
		}
		var e nativeHostEvent
		if active && json.Unmarshal(line, &e) == nil && e.Type == "event_msg" && e.Payload.Type == "item_completed" && e.Payload.Item.Type == "CommandExecution" {
			i := e.Payload.Item
			if observed || i.ExitCode == nil {
				return false
			}
			observed = true
			want = plan.render(*i.ExitCode, i.Output)
		}
	}
	return false
}

// The retained host also prints the sole literal exec result's output string.
// This exact projection changes presentation only; invocation evidence still
// comes from the separately correlated CommandExecution and call/result events.
// Requiring the entire input excludes additional commands or result rewriting.
func nativeCodeModeOutputProjection(input, defaultCwd string) (nativeRecordedShellCommand, bool) {
	pattern := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+tools\.exec_command\((\{[\s\S]*\})\);\s*([\s\S]*)$`)
	match := pattern.FindStringSubmatch(input)
	if len(match) != 4 {
		return nativeRecordedShellCommand{}, false
	}
	name := regexp.QuoteMeta(match[1])
	output := `text\(` + name + `\.output\)`
	full := `text\(JSON\.stringify\(` + name + `\)\)`
	// The exact observed exit annotation is presentation, never exit proof.
	exit := regexp.QuoteMeta("`\\nEXIT_CODE=${" + match[1] + ".exit_code}`")
	print := regexp.MustCompile(`^\s*(?:` + output + `(?:;\s*text\(` + exit + `\))?|` + full + `);?\s*$`)
	if !print.MatchString(match[3]) {
		return nativeRecordedShellCommand{}, false
	}
	return nativeLiteralCommandObject(match[2], defaultCwd)
}

func nativeCodeModePlainOutput(input, defaultCwd string) bool {
	_, ok := nativeCodeModeOutputProjection(input, defaultCwd)
	return ok && regexp.MustCompile(`text\([A-Za-z_][A-Za-z0-9_]*\.output\);?\s*$`).MatchString(input)
}

// Decode only the small literal object accepted by this evidence contract.
// This never evaluates JavaScript or rewrites quote/key text globally: quoted
// shell arguments retain their exact bytes, and duplicate keys are ambiguous.
func nativeLiteralCommandObject(object, defaultCwd string) (nativeRecordedShellCommand, bool) {
	p := nativeCommandLiteral{input: object}
	result := nativeRecordedShellCommand{Cwd: defaultCwd}
	if !p.take('{') {
		return result, false
	}
	seen := map[string]bool{}
	for {
		p.space()
		if p.pos >= len(p.input) {
			return result, false
		}
		key, ok := "", false
		if p.input[p.pos] == '\'' || p.input[p.pos] == '"' {
			key, ok = p.quoted()
		} else {
			start := p.pos
			for p.pos < len(p.input) && ((p.input[p.pos] >= 'a' && p.input[p.pos] <= 'z') || p.input[p.pos] == '_') {
				p.pos++
			}
			key = p.input[start:p.pos]
			ok = key != ""
		}
		if !ok || seen[key] || !p.take(':') {
			return result, false
		}
		seen[key] = true
		switch key {
		case "cmd", "workdir":
			value, ok := p.quoted()
			if !ok {
				return result, false
			}
			if key == "cmd" {
				result.Command = value
			} else {
				result.Cwd = value
			}
		case "max_output_tokens", "yield_time_ms":
			p.space()
			start := p.pos
			for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
				p.pos++
			}
			value := p.input[start:p.pos]
			if value == "" || (len(value) > 1 && value[0] == '0') {
				return result, false
			}
			if _, err := strconv.ParseUint(value, 10, 64); err != nil {
				return result, false
			}
		default:
			return result, false
		}
		if p.take('}') {
			break
		}
		if !p.take(',') {
			return result, false
		}
	}
	p.space()
	return result, p.pos == len(p.input) && seen["cmd"] && strings.TrimSpace(result.Command) != "" && filepath.IsAbs(result.Cwd)
}

type nativeCommandLiteral struct {
	input string
	pos   int
}

func (p *nativeCommandLiteral) space() {
	for p.pos < len(p.input) && strings.ContainsRune(" \t\r\n", rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *nativeCommandLiteral) take(want byte) bool {
	p.space()
	if p.pos >= len(p.input) || p.input[p.pos] != want {
		return false
	}
	p.pos++
	return true
}

func (p *nativeCommandLiteral) quoted() (string, bool) {
	p.space()
	if p.pos >= len(p.input) || (p.input[p.pos] != '\'' && p.input[p.pos] != '"') {
		return "", false
	}
	quote := p.input[p.pos]
	p.pos++
	var out strings.Builder
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		p.pos++
		if c == quote {
			return out.String(), true
		}
		if c < 0x20 {
			return "", false
		}
		if c != '\\' {
			if c < utf8.RuneSelf {
				out.WriteByte(c)
				continue
			}
			r, size := utf8.DecodeRuneInString(p.input[p.pos-1:])
			if r == utf8.RuneError && size == 1 {
				return "", false
			}
			// JavaScript line separators are outside the admitted literal subset.
			if r == '\u2028' || r == '\u2029' {
				return "", false
			}
			out.WriteRune(r)
			p.pos += size - 1
			continue
		}
		if p.pos >= len(p.input) {
			return "", false
		}
		escape := p.input[p.pos]
		p.pos++
		switch escape {
		case '\\', '\'', '"', '/':
			out.WriteByte(escape)
		case 'n':
			out.WriteByte('\n')
		case 'r':
			out.WriteByte('\r')
		case 't':
			out.WriteByte('\t')
		case 'b':
			out.WriteByte('\b')
		case 'f':
			out.WriteByte('\f')
		case 'u':
			readHex := func() (rune, bool) {
				if p.pos+4 > len(p.input) {
					return 0, false
				}
				n, err := strconv.ParseUint(p.input[p.pos:p.pos+4], 16, 16)
				p.pos += 4
				return rune(n), err == nil
			}
			r, ok := readHex()
			if !ok {
				return "", false
			}
			if r >= 0xD800 && r <= 0xDBFF {
				if !strings.HasPrefix(p.input[p.pos:], `\u`) {
					return "", false
				}
				p.pos += 2
				low, ok := readHex()
				if !ok || low < 0xDC00 || low > 0xDFFF {
					return "", false
				}
				r = utf16.DecodeRune(r, low)
			} else if utf16.IsSurrogate(r) {
				return "", false
			}
			out.WriteRune(r)
		default:
			return "", false
		}
	}
	return "", false
}

// Recognize the observed literal-call array and untouched indexed result loop.
// This is syntax recognition, never JavaScript evaluation. Every operation is
// additionally subject to the caller's path/source and actual event checks.
// The entire batch grammar is validated separately; this identifies its
// untouched per-result print form, whose outputs must match actual events.
// forEach supplies index/array arguments too. Only actual complete result
// blocks may establish presentation; never infer printed bytes from callbacks.
func nativeReadOnlyBatchForEach(input string) bool {
	return regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*\.forEach\(text\);\s*$`).MatchString(input)
}

func nativeReadOnlyBatchFullResults(input string) bool {
	return nativeReadOnlyBatchForEach(input) || regexp.MustCompile(`for\s*\(const\s+[A-Za-z_][A-Za-z0-9_]*\s+of\s+[A-Za-z_][A-Za-z0-9_]*\)\s*text\([A-Za-z_][A-Za-z0-9_]*\);\s*$`).MatchString(input)
}

// Called only after nativeCorroboratedBatch proves the unique same-child/turn
// invocation, each command event and completed output. Promise.all preserves
// input order even when events complete in a different order.
func nativeCorroboratedBatchResults(raw []byte, callID string, commands []nativeRecordedShellCommand) bool {
	type observed struct {
		exit   int
		output string
	}
	results := make(map[string]observed)
	active := false
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var wire nativeCodeModeWire
		if json.Unmarshal(line, &wire) == nil && wire.Type == "response_item" {
			p := wire.Payload
			if p.Type == "custom_tool_call" && p.CallID == callID {
				active = true
			}
			if p.Type == "custom_tool_call_output" && p.CallID == callID {
				if !active || len(p.Output) != len(commands)+1 {
					return false
				}
				for index, command := range commands {
					item, ok := results[command.Cwd+"\x00"+command.Command]
					single := wire
					single.Payload.Output = append(single.Payload.Output[:0:0], p.Output[0], p.Output[index+1])
					exit, output, complete := nativeCodeModeResult(single)
					if !ok || !complete || exit != item.exit || output != item.output {
						return false
					}
				}
				return true
			}
		}
		var event nativeHostEvent
		if !active || json.Unmarshal(line, &event) != nil || event.Type != "event_msg" || event.Payload.Type != "item_completed" || event.Payload.Item.Type != "CommandExecution" {
			continue
		}
		item := event.Payload.Item
		if len(item.Command) != 3 || item.ExitCode == nil {
			return false
		}
		for _, command := range commands {
			if nativeSameCwd(item.Cwd, command.Cwd) && item.Command[2] == command.Command {
				results[command.Cwd+"\x00"+command.Command] = observed{*item.ExitCode, item.Output}
			}
		}
	}
	return false
}

// One literal label followed by each complete output in input order. This is
// a finite presentation grammar, not JavaScript evaluation or test evidence.
func nativeReadOnlyBatchConcatenation(input string) (string, []string, bool) {
	pattern := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+Promise\.all\(\[([\s\S]*)\]\);\s*text\(([\s\S]*)\);\s*$`)
	m := pattern.FindStringSubmatch(input)
	reserved := " JSON Promise tools text await break case catch class const continue debugger default delete do else enum export extends false finally for function if import in instanceof let new null return super switch this throw true try typeof var void while with yield implements interface package private protected public static eval arguments "
	if len(m) != 4 || strings.Contains(reserved, " "+m[1]+" ") {
		return "", nil, false
	}
	part := regexp.MustCompile(`^\s*("(?:[^"\\]|\\.)*")\s*\+\s*` + regexp.QuoteMeta(m[1]) + `\[(0|[1-9][0-9]*)\]\.output\s*`)
	remaining := m[3]
	var labels []string
	for {
		p := part.FindStringSubmatchIndex(remaining)
		if p == nil || len(labels) == 64 || remaining[p[4]:p[5]] != strconv.Itoa(len(labels)) {
			return "", nil, false
		}
		var label string
		if json.Unmarshal([]byte(remaining[p[2]:p[3]]), &label) != nil || len(label) > 128 || !regexp.MustCompile(`^[A-Za-z0-9_. /:\-\n	]*$`).MatchString(label) {
			return "", nil, false
		}
		labels = append(labels, label)
		remaining = strings.TrimSpace(remaining[p[1]:])
		if remaining == "" {
			return m[2], labels, true
		}
		if !strings.HasPrefix(remaining, "+") {
			return "", nil, false
		}
		remaining = remaining[1:]
	}
}

// The caller first requires unique successful CommandExecution events in the
// same child/turn/cwd, bounded by the one call and completed outer output.
// Completion order may differ; the rendered string must follow input order.
func nativeCorroboratedBatchConcatenation(raw []byte, callID string, commands []nativeRecordedShellCommand) bool {
	active := false
	var labels []string
	outputs := map[string]string{}
	for _, line := range bytes.Split(raw, []byte{'\n'}) {
		var wire nativeCodeModeWire
		if json.Unmarshal(line, &wire) == nil && wire.Type == "response_item" {
			p := wire.Payload
			if p.Type == "custom_tool_call" && p.CallID == callID {
				var ok bool
				_, labels, ok = nativeReadOnlyBatchConcatenation(p.Input)
				if !ok || len(labels) != len(commands) {
					return false
				}
				active = true
			}
			if p.Type == "custom_tool_call_output" && p.CallID == callID {
				if !active || len(p.Output) != 2 || p.Output[0].Type != "input_text" || p.Output[1].Type != "input_text" {
					return false
				}
				var expected strings.Builder
				for index, command := range commands {
					output, found := outputs[command.Cwd+"\x00"+command.Command]
					if !found {
						return false
					}
					expected.WriteString(labels[index])
					expected.WriteString(output)
				}
				return p.Output[1].Text == expected.String()
			}
		}
		var event nativeHostEvent
		if !active || json.Unmarshal(line, &event) != nil || event.Type != "event_msg" || event.Payload.Type != "item_completed" || event.Payload.Item.Type != "CommandExecution" {
			continue
		}
		item := event.Payload.Item
		if len(item.Command) != 3 {
			return false
		}
		for _, command := range commands {
			if nativeSameCwd(item.Cwd, command.Cwd) && item.Command[2] == command.Command {
				outputs[command.Cwd+"\x00"+command.Command] = item.Output
			}
		}
	}
	return false
}

func nativeReadOnlyBatchCommands(input, cwd string) ([]nativeRecordedShellCommand, bool) {
	reserved := " JSON Promise tools text await break case catch class const continue debugger default delete do else enum export extends false finally for function if import in instanceof let new null return super switch this throw true try typeof var void while with yield implements interface package private protected public static eval arguments "
	array := ""
	namedCount := 0
	forEach := false
	concatenatedCount := 0
	each := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+Promise\.all\(\[([\s\S]*)\]\);\s*([A-Za-z_][A-Za-z0-9_]*)\.forEach\(text\);\s*$`)
	indexed := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+Promise\.allSettled\(\[([\s\S]*)\]\);\s*for\s*\(let\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*0;\s*([A-Za-z_][A-Za-z0-9_]*)\s*<\s*([A-Za-z_][A-Za-z0-9_]*)\.length;\s*([A-Za-z_][A-Za-z0-9_]*)\+\+\)\s*text\(\{\s*(?:index\s*:\s*)?([A-Za-z_][A-Za-z0-9_]*),\s*\.\.\.([A-Za-z_][A-Za-z0-9_]*)\[([A-Za-z_][A-Za-z0-9_]*)\]\s*\}\);\s*$`)
	projected := regexp.MustCompile(`^\s*const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*await\s+Promise\.all\(\[([\s\S]*)\]\);\s*for\s*\(const\s+([A-Za-z_][A-Za-z0-9_]*)\s+of\s+([A-Za-z_][A-Za-z0-9_]*)\)\s*([\s\S]*)$`)
	identifiers := `\s*[A-Za-z_][A-Za-z0-9_]*(?:\s*,\s*[A-Za-z_][A-Za-z0-9_]*)*\s*`
	named := regexp.MustCompile(`^\s*const\s+\[(` + identifiers + `)\]\s*=\s*await\s+Promise\.all\(\[([\s\S]*)\]\);\s*text\(JSON\.stringify\(\{(` + identifiers + `)\}\)\);\s*$`)
	if source, labels, ok := nativeReadOnlyBatchConcatenation(input); ok {
		array, concatenatedCount = source, len(labels)
	} else if m := indexed.FindStringSubmatch(input); len(m) == 10 && m[1] == m[5] && m[1] == m[8] && m[3] == m[4] && m[3] == m[6] && m[3] == m[7] && m[3] == m[9] {
		array = m[2]
	} else if m := each.FindStringSubmatch(input); len(m) == 4 && m[1] == m[3] && !strings.Contains(reserved, " "+m[1]+" ") {
		array, forEach = m[2], true
	} else if m := projected.FindStringSubmatch(input); len(m) == 6 && m[1] == m[4] && m[1] != m[3] && !strings.Contains(reserved, " "+m[1]+" ") && !strings.Contains(reserved, " "+m[3]+" ") {
		name := regexp.QuoteMeta(m[3])
		print := regexp.MustCompile(`^\s*text\((?:` + name + `(?:\.output)?|JSON\.stringify\(` + name + `\))\);\s*$`)
		if !print.MatchString(m[5]) {
			return nil, false
		}
		array = m[2]
	} else if m := named.FindStringSubmatch(input); len(m) == 4 {
		// Bind each untouched result to one unique identifier and print exactly
		// those shorthand fields in order. No aliases, expressions or shadowed
		// execution/printing globals can enter this syntax-only decoder.
		bindings, fields := strings.Split(m[1], ","), strings.Split(m[3], ",")
		if len(bindings) != len(fields) || len(bindings) > 64 {
			return nil, false
		}
		seen := map[string]bool{}
		for index, binding := range bindings {
			name := strings.TrimSpace(binding)
			if seen[name] || strings.Contains(reserved, " "+name+" ") || name != strings.TrimSpace(fields[index]) {
				return nil, false
			}
			seen[name] = true
		}
		array, namedCount = m[2], len(bindings)
	} else {
		return nil, false
	}
	remaining := strings.TrimSpace(array)
	call := regexp.MustCompile(`^tools\.exec_command\((\{[\s\S]*?\})\)\s*(,|$)\s*`)
	var commands []nativeRecordedShellCommand
	for remaining != "" {
		part := call.FindStringSubmatchIndex(remaining)
		if part == nil {
			return nil, false
		}
		parsed, ok := nativeCodeModeCommands("text(await tools.exec_command("+remaining[part[2]:part[3]]+"));", cwd)
		if !ok || len(parsed) != 1 {
			return nil, false
		}
		owned := codexNativeLiveReceipt{FixtureRoot: parsed[0].Cwd}
		chain := nativeChildInspectionChain(owned, parsed[0].Command)
		allowed := nativeReadOnlyBatchCommand(parsed[0].Command) || chain
		if forEach || concatenatedCount > 0 {
			// The new presentation form admits only the same child inspection atoms
			// and chains, not broader parent discovery/runtime operations.
			allowed = nativeChildInspectionAtom(owned, parsed[0].Command) || chain
		}
		if !allowed {
			return nil, false
		}
		commands = append(commands, parsed[0])
		if len(commands) > 64 {
			return nil, false
		}
		remaining = strings.TrimSpace(remaining[part[1]:])
	}
	return commands, len(commands) > 0 && (namedCount == 0 || namedCount == len(commands)) && (concatenatedCount == 0 || concatenatedCount == len(commands))
}

func nativeReadOnlyBatchCommand(command string) bool {
	words, ok := nativeSimpleShellWords(command)
	if !ok || len(words) == 0 {
		return false
	}
	for len(words) > 0 && (words[0] == "AETHER_OUTPUT_MODE=json" || words[0] == "AETHER_OUTPUT_MODE=visual" || words[0] == "AETHER_FORCE_COLOR=1" || words[0] == "AETHER_FORCE_COLOR=0") {
		words = words[1:]
	}
	if len(words) == 0 {
		return false
	}
	if nativeAdditionalFixtureInspection(command, words, func(path string) bool {
		return path == "clamp.go" || path == "clamp_test.go" || path == "double.go" || path == "double_test.go" || path == "go.mod" || path == "AGENTS.md"
	}) || nativeLongDateWords(words) || strings.Join(words, " ") == "date -u +%Y-%m-%dT%H:%M:%SZ" {
		return true
	}
	switch words[0] {
	case "cat", "head", "sed", "jq", "rg", "pwd", "ls", "cmp", "git":
		return nativeParentCoordinationCommand(&codexNativeLiveReceipt{}, []string{"/bin/sh", "-c", strings.Join(words, " ")})
	case "aether":
		return (len(words) == 2 && words[1] == "status") || (len(words) == 5 && words[1] == "command-guide" && words[3] == "--platform" && words[4] == "codex")
	case "python3":
		if len(words) != 3 && len(words) != 4 {
			return false
		}
		if len(words) == 4 && !regexp.MustCompile(`^[0-9]+$`).MatchString(words[3]) {
			return false
		}
		return words[2] == "summary" || words[2] == "prompt" || words[2] == "release" || words[2] == "inspect"
	}
	return false
}

func nativePathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// Shared by live collection and immutable replay controls. This is the
// original capture selection policy; changing a snapshot's storage name must
// never authorize indexing credentials or the enclosing receipt itself.
func nativeRetainedArtifactPath(path string) bool {
	base := filepath.Base(path)
	if base == "auth.json" || base == "receipt.json" || strings.Contains(path, string(filepath.Separator)+".git"+string(filepath.Separator)) {
		return false
	}
	return strings.HasSuffix(path, ".go") || base == "go.mod" || base == "go.sum" || strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".toml") || strings.HasSuffix(path, ".patch") || strings.HasSuffix(path, ".py")
}

func nativeBeginReceiptReplay(r *codexNativeLiveReceipt) error {
	if r.ContextProtocol != "" && r.ContextProtocol != codexNativeContextProtocolChildFetch {
		return fmt.Errorf("unknown native context proof protocol")
	}
	r.replaying = true
	r.replayArtifacts = r.Artifacts
	if len(r.replayArtifacts) == 0 {
		return fmt.Errorf("capture artifact inventory absent")
	}
	return nil
}
func nativeCapturedCompletion(data []byte) (codexExternalBuildCompletion, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return codexExternalBuildCompletion{}, err
	}
	absent := func(key string) bool { v, ok := raw[key]; return !ok || string(v) == "null" }
	if absent("dispatch_manifest") && absent("manifest") {
		data = raw["result"]
	}
	var completion codexExternalBuildCompletion
	if err := json.Unmarshal(data, &completion); err != nil {
		return completion, err
	}
	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return completion, err
	}
	if completion.activeManifest() == nil || !manifestSelectionMatchesRaw(completion, generic) {
		return completion, fmt.Errorf("captured completion manifest missing")
	}
	return completion, nil
}
