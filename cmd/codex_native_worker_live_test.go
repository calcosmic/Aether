package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
	BaselineTestsSHA256        string                `json:"baseline_tests_sha256,omitempty"`
	BaselineModuleSHA256       string                `json:"baseline_module_sha256,omitempty"`
	LaunchMessageEncoding      string                `json:"launch_message_encoding,omitempty"`
	PromptDeliveryVerification string                `json:"prompt_delivery_verification,omitempty"`
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
			if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".jsonl") || strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md") || strings.HasSuffix(path, ".toml") || strings.HasSuffix(path, ".patch") {
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
	if scenario != "ordinary" {
		fail("early-resume is gated on a passing ordinary native tracer; no resumed host has been run")
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
	liveSkillRuntime(t, repo, env, filepath.Join(runRoot, "install.json"), receipt.CandidatePath, "install", "--package-dir", source, "--home-dir", fixtureHome, "--channel", "stable", "--skip-build-binary")
	receipt.SkillPath = filepath.Join(fixtureHome, ".codex", "skills", "aether", "ant-build", "SKILL.md")
	receipt.SupportPath = filepath.Join(fixtureHome, ".codex", "skills", "aether", "support", "aether-colony-build-cycle.md")
	nativePrepareLiveFixture(t, repo, runRoot)
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
	prompt := "$ant-build 1\nFixture authorization: execute the already accepted one-task plan in this disposable repository. Use the installed skill and its private support. The named native Builder must make the clamp.go edit and run go test ./... itself; the parent may coordinate runtime commands but cannot edit library or test files, commit, recruit, publish, install or touch another project. All routine fixture choices are preauthorized. Preserve complete native parent/child event evidence in this isolated home, including terminal response identity. If the native tools, named agent, permissions or raw child evidence are unavailable, report the exact limitation and stop without substituting a subprocess worker. Do not read authentication files. Stop after the existing build finalizer; do not run continue.\n"
	liveSkillWrite(t, filepath.Join(runRoot, "prompt.txt"), []byte(prompt))
	args := []string{"exec", "--ignore-user-config", "--ignore-rules", "-s", "workspace-write", "--json", "--color", "never", "-C", repo, "-c", `cli_auth_credentials_store="file"`, "-c", `approval_policy="never"`, "--enable", "multi_agent", "-c", "shell_environment_policy.set.PATH=" + fmt.Sprintf("%q", filepath.Dir(receipt.CandidatePath)+":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin"), "-c", `shell_environment_policy.set.AETHER_OUTPUT_MODE="json"`}
	if preferences.Model != "" {
		args = append(args, "-m", preferences.Model)
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
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
	if err := validateCodexNativeLiveReceipt(receipt); err != nil {
		fail(err.Error())
	}
	receipt.Outcome, receipt.Reason = "passed", ""
}

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
	liveSkillWrite(t, filepath.Join(root, "AGENTS.md"), []byte("# Disposable native worker fixture\nOnly the runtime-assigned native Builder may edit clamp.go. Do not change clamp_test.go or go.mod. Parent coordinates only. No commits or external actions.\n## Verification Commands\n- build: go build ./...\n- tests: go test ./...\n- lint: go vet ./...\n"))
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

func nativeInspectChildEvents(r *codexNativeLiveReceipt, raw []byte) {
	// A returned function/custom-tool output must corroborate each call. Prose
	// mentioning a command never counts. Actual host names are retained above.
	calls := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	for scanner.Scan() {
		var hostEvent struct {
			Type    string `json:"type"`
			Payload struct {
				Type     string `json:"type"`
				ThreadID string `json:"thread_id"`
				Item     struct {
					Type     string                     `json:"type"`
					Status   string                     `json:"status"`
					Changes  map[string]json.RawMessage `json:"changes"`
					Command  []string                   `json:"command"`
					ExitCode *int                       `json:"exit_code"`
					Output   string                     `json:"aggregated_output"`
				} `json:"item"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &hostEvent) == nil && hostEvent.Type == "event_msg" && hostEvent.Payload.Type == "item_completed" && hostEvent.Payload.ThreadID == r.ChildID {
			item := hostEvent.Payload.Item
			if item.Type == "FileChange" && item.Status == "completed" {
				for path := range item.Changes {
					if path == filepath.Join(r.FixtureRoot, "clamp.go") {
						r.ChildEditObserved = true
					}
				}
			}
			if item.Type == "CommandExecution" && item.Status == "completed" && item.ExitCode != nil && *item.ExitCode == 0 && strings.Contains(strings.Join(item.Command, " "), "go test ./...") && strings.Contains(item.Output, "ok") {
				r.ChecksPassed = true
			}
		}
		// Source events may be host AgentMessage envelopes or response items.
		if strings.TrimPrefix(lifecycleDigest(scanner.Bytes()), "sha256:") == strings.TrimPrefix(r.SavedSourceEventSHA256, "sha256:") || strings.TrimPrefix(lifecycleDigest(append(append([]byte(nil), scanner.Bytes()...), byte(10))), "sha256:") == strings.TrimPrefix(r.SavedSourceEventSHA256, "sha256:") {
			r.SourceEventCorroborated = true
		}
		var event struct {
			Type    string `json:"type"`
			Payload struct {
				Type      string          `json:"type"`
				Name      string          `json:"name"`
				CallID    string          `json:"call_id"`
				Arguments string          `json:"arguments"`
				Input     string          `json:"input"`
				Output    json.RawMessage `json:"output"`
				Role      string          `json:"role"`
				Content   []struct {
					Text string `json:"text"`
				} `json:"content"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &event) != nil || event.Type != "response_item" {
			continue
		}
		p := event.Payload
		if strings.TrimPrefix(lifecycleDigest(scanner.Bytes()), "sha256:") == strings.TrimPrefix(r.SavedSourceEventSHA256, "sha256:") || strings.TrimPrefix(lifecycleDigest(append(append([]byte(nil), scanner.Bytes()...), byte(10))), "sha256:") == strings.TrimPrefix(r.SavedSourceEventSHA256, "sha256:") {
			r.SourceEventCorroborated = true
		}
		if p.Type == "message" && p.Role == "assistant" && r.SavedTerminal != nil {
			for _, part := range p.Content {
				text := strings.TrimSpace(part.Text)
				text = strings.TrimPrefix(text, "```json")
				text = strings.TrimSuffix(text, "```")
				var result internalWorkerResult
				if json.Unmarshal([]byte(strings.TrimSpace(text)), &result) == nil {
					result.Status = strings.ToLower(strings.TrimSpace(result.Status))
					digest, _ := jsonSHA256(&result)
					if digest == r.ResultSHA256 {
						r.TerminalCorroborated = true
					}
				}
			}
		}
		switch p.Type {
		case "function_call", "custom_tool_call":
			calls[p.CallID] = p.Name + "\n" + p.Arguments + p.Input
		case "function_call_output", "custom_tool_call_output":
			call := calls[p.CallID]
			output := nativeToolOutputText(p.Output)
			if strings.Contains(call, "apply_patch") && strings.Contains(call, "clamp.go") && strings.Contains(output, "Success") {
				r.ChildEditObserved = true
			}
			if strings.Contains(call, "go test ./...") && (strings.Contains(output, "Process exited with code 0") || strings.Contains(output, `"exit_code":0`) || strings.Contains(output, `"exit_code": 0`)) && strings.Contains(output, "ok") {
				r.ChecksPassed = true
			}
		}
	}
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
	for path, want := range map[string]string{r.CandidatePath: r.CandidateSHA256, r.ClientPath: r.ClientSHA256, filepath.Join(r.FixtureRoot, "clamp_test.go"): r.BaselineTestsSHA256, filepath.Join(r.FixtureRoot, "go.mod"): r.BaselineModuleSHA256} {
		raw, err := os.ReadFile(path)
		if err != nil || lifecycleDigest(raw) != want {
			return fmt.Errorf("candidate/client/baseline identity changed: %s", path)
		}
	}
	if !r.ChildEditObserved || !r.ChecksPassed {
		return fmt.Errorf("raw native child edit/check evidence incomplete (edit=%v checks=%v); child log %s", r.ChildEditObserved, r.ChecksPassed, r.ChildEvents)
	}
	if r.ParentSubstitution {
		return fmt.Errorf("parent performed the promised child work")
	}
	if r.AttemptID == "" || r.LaunchID == "" || r.ResultSHA256 == "" || r.CompletionPath == "" || !r.CreditObserved {
		return fmt.Errorf("accepted native terminal/aggregate/finalizer credit incomplete")
	}
	return nil
}

// Inspect actual parent calls separately. The native tool count is independent
// of the parent CLI's display schema; no assistant text counts as a launch.
func nativeInspectParentEvents(r *codexNativeLiveReceipt, raw []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	scanner.Buffer(make([]byte, 65536), 16<<20)
	spawns := 0
	for scanner.Scan() {
		var host struct {
			Type    string `json:"type"`
			Payload struct {
				Type     string `json:"type"`
				ThreadID string `json:"thread_id"`
				Item     struct {
					Type    string                     `json:"type"`
					Status  string                     `json:"status"`
					Changes map[string]json.RawMessage `json:"changes"`
				} `json:"item"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &host) == nil && host.Type == "event_msg" && host.Payload.Type == "item_completed" && host.Payload.ThreadID == r.SessionID && host.Payload.Item.Type == "FileChange" && host.Payload.Item.Status == "completed" {
			for path := range host.Payload.Item.Changes {
				if path == filepath.Join(r.FixtureRoot, "clamp.go") || path == filepath.Join(r.FixtureRoot, "clamp_test.go") {
					r.ParentSubstitution = true
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
