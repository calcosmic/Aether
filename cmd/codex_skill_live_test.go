package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
)

// This is a real-provider opt-in, never a fake dispatcher. Set EVIDENCE_DIR to
// durable storage; the test refuses to substitute temporary proof or a new root.
type codexSkillLiveReceipt struct {
	SchemaVersion            string   `json:"schema_version"`
	SourceRevision           string   `json:"source_revision"`
	SourceStatus             string   `json:"source_status"`
	CandidatePath            string   `json:"candidate_path"`
	CandidateDigest          string   `json:"candidate_sha256"`
	CandidateVersion         string   `json:"candidate_version"`
	ClientPath               string   `json:"client_path"`
	ClientDigest             string   `json:"client_sha256"`
	ClientVersion            string   `json:"client_version"`
	Model                    string   `json:"model"`
	PayloadIdentity          string   `json:"payload_identity"`
	DiscoveryRoot            string   `json:"discovery_root"`
	SkillPath                string   `json:"skill_path"`
	SkillDigest              string   `json:"installed_skill_sha256"`
	LoadedPath               string   `json:"loaded_path,omitempty"`
	LoadedDigest             string   `json:"loaded_sha256,omitempty"`
	SkillLoadEvent           string   `json:"skill_load_event,omitempty"`
	GuideEvent               string   `json:"guide_event,omitempty"`
	GuideCommand             string   `json:"guide_command,omitempty"`
	CandidateResolutionEvent string   `json:"candidate_resolution_event,omitempty"`
	RawEvents                string   `json:"raw_events"`
	RawStderr                string   `json:"raw_stderr"`
	Args                     []string `json:"client_args"`
	ExitStatus               int      `json:"exit_status"`
	MissingSkill             bool     `json:"missing_skill_control"`
	AuthRemoved              bool     `json:"temporary_auth_removed"`
	Outcome                  string   `json:"outcome"`
	Reason                   string   `json:"reason,omitempty"`
}

type codexSkillHostEvent struct {
	Type string `json:"type"`
	Item struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Command  string `json:"command"`
		Output   string `json:"aggregated_output"`
		Status   string `json:"status"`
		ExitCode *int   `json:"exit_code"`
	} `json:"item"`
}

// Only successful host tool events can prove loading/routing. Assistant prose,
// command text without completion, and knowing a route with no file cannot pass.
func validateCodexSkillLiveEvidence(receipt *codexSkillLiveReceipt, events []byte, skill []byte, runtimeRoute string) error {
	receipt.LoadedPath, receipt.LoadedDigest, receipt.SkillLoadEvent = "", "", ""
	receipt.GuideEvent, receipt.GuideCommand, receipt.CandidateResolutionEvent = "", "", ""
	if receipt.ExitStatus != 0 {
		return fmt.Errorf("client exited %d", receipt.ExitStatus)
	}
	installed, err := os.ReadFile(receipt.SkillPath)
	if err != nil {
		return fmt.Errorf("selected installed skill unavailable: %w", err)
	}
	if !bytes.Equal(installed, skill) || lifecycleDigest(installed) != receipt.SkillDigest {
		return fmt.Errorf("installed skill bytes changed")
	}
	if !pathIsWithin(receipt.DiscoveryRoot, receipt.SkillPath) {
		return fmt.Errorf("loaded skill outside selected discovery root")
	}
	complete := false
	scanner := bufio.NewScanner(bytes.NewReader(events))
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		var event codexSkillHostEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return fmt.Errorf("invalid host event at line %d: %w", line, err)
		}
		if event.Type == "turn.completed" {
			complete = true
		}
		item := event.Item
		if event.Type != "item.completed" || item.Type != "command_execution" || item.Status != "completed" || item.ExitCode == nil || *item.ExitCode != 0 {
			continue
		}
		ref := fmt.Sprintf("%s:%d#%s", receipt.RawEvents, line, item.ID)
		if strings.Contains(item.Command, receipt.SkillPath) && strings.Contains(item.Command, "cat ") && strings.TrimSpace(item.Output) == strings.TrimSpace(string(skill)) {
			receipt.LoadedPath, receipt.LoadedDigest, receipt.SkillLoadEvent = receipt.SkillPath, receipt.SkillDigest, ref
		}
		if strings.Contains(item.Command, "command -v aether") && strings.TrimSpace(item.Output) == receipt.CandidatePath {
			receipt.CandidateResolutionEvent = ref
		}
		// A command merely echoed in prose/output is not an invocation.
		command := unwrapCodexShellCommand(item.Command)
		pattern := `^(?:AETHER_OUTPUT_MODE=json\s+)?(?:aether|` + regexp.QuoteMeta(receipt.CandidatePath) + `) command-guide plan --platform codex$`
		if regexp.MustCompile(pattern).MatchString(command) {
			var guide struct {
				OK     bool `json:"ok"`
				Result struct {
					Command    string `json:"command"`
					Platform   string `json:"platform"`
					RunCommand string `json:"run_command"`
				} `json:"result"`
			}
			if json.Unmarshal([]byte(item.Output), &guide) == nil && guide.OK && guide.Result.Command == "plan" && guide.Result.Platform == "codex" && guide.Result.RunCommand == runtimeRoute {
				receipt.GuideEvent, receipt.GuideCommand = ref, command
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if !complete {
		return fmt.Errorf("no completed host turn")
	}
	if receipt.SkillLoadEvent == "" {
		return fmt.Errorf("no successful host read of the selected installed SKILL.md bytes")
	}
	if receipt.CandidateResolutionEvent == "" {
		return fmt.Errorf("no host evidence resolving aether to the exact candidate")
	}
	if receipt.GuideEvent == "" {
		return fmt.Errorf("no successful actual command-guide plan --platform codex invocation")
	}
	return nil
}

func TestCodexAntSkillLiveEvidence(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ant-plan", "SKILL.md")
	skill := []byte("---\nname: ant-plan\n---\nReal fixture instructions\n")
	writeMaintenanceMutation199File(t, path, skill)
	base := codexSkillLiveReceipt{DiscoveryRoot: root, SkillPath: path, SkillDigest: lifecycleDigest(skill), CandidatePath: "/candidate/aether", RawEvents: "fixture.jsonl"}
	const route = "runtime route fixture"
	commandEvent := func(id, command, output string) string {
		value := map[string]any{"type": "item.completed", "item": map[string]any{"id": id, "type": "command_execution", "command": command, "aggregated_output": output, "exit_code": 0, "status": "completed"}}
		raw, _ := json.Marshal(value)
		return string(raw) + "\n"
	}
	read := commandEvent("load", "cat "+path, string(skill))
	resolve := commandEvent("resolve", "command -v aether", base.CandidatePath+"\n")
	guide := commandEvent("guide", "/bin/zsh -lc 'AETHER_OUTPUT_MODE=json aether command-guide plan --platform codex'", `{"ok":true,"result":{"command":"plan","platform":"codex","run_command":"runtime route fixture"}}`)
	const done = "{\"type\":\"turn.completed\"}\n"
	t.Run("host-events", func(t *testing.T) {
		receipt := base
		if err := validateCodexSkillLiveEvidence(&receipt, []byte(read+resolve+guide+done), skill, route); err != nil {
			t.Fatal(err)
		}
		if receipt.LoadedDigest != base.SkillDigest || receipt.GuideEvent == "" {
			t.Fatal("lost evidence linkage")
		}
	})
	for _, test := range []struct{ name, events string }{
		{"model-claim", `{"type":"item.completed","item":{"type":"agent_message","text":"I loaded ant-plan and ran aether command-guide plan --platform codex"}}` + "\n" + done},
		{"command-without-load", resolve + guide + done},
		{"wrong-candidate", read + guide + done},
		{"no-turn-completion", read + resolve + guide},
		{"echo-not-invocation", read + resolve + strings.Replace(guide, "aether command-guide", "echo aether command-guide", 1) + done},
		{"wrong-loaded-bytes", strings.Replace(read, "Real fixture", "Fake fixture", 1) + resolve + guide + done},
		{"failed-tool", strings.Replace(read, `"exit_code":0`, `"exit_code":1`, 1) + resolve + guide + done},
	} {
		t.Run(test.name, func(t *testing.T) {
			receipt := base
			receipt.SkillLoadEvent, receipt.GuideEvent, receipt.CandidateResolutionEvent = "cached", "cached", "cached"
			if err := validateCodexSkillLiveEvidence(&receipt, []byte(test.events), skill, route); err == nil {
				t.Fatal("accepted unproved claim")
			}
		})
	}
	t.Run("missing-skill-despite-known-route", func(t *testing.T) {
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		receipt := base
		if err := validateCodexSkillLiveEvidence(&receipt, []byte(read+resolve+guide+done), skill, route); err == nil {
			t.Fatal("missing installed skill passed")
		}
	})
}

func unwrapCodexShellCommand(command string) string {
	for _, marker := range []string{" -lc '", " -c '"} {
		if i := strings.Index(command, marker); i >= 0 && strings.HasSuffix(command, "'") {
			return command[i+len(marker) : len(command)-1]
		}
	}
	return command
}

func TestCodexAntSkillFreshHost(t *testing.T) {
	if os.Getenv("AETHER_CODEX_SKILL_LIVE") != "1" {
		t.Skip("opt-in real Codex provider check; no live proof claimed")
	}
	if os.Getenv("AETHER_CODEX_SKILL_LIVE_CASES") != "plan" {
		t.Fatal("this tracer requires AETHER_CODEX_SKILL_LIVE_CASES=plan")
	}
	evidenceRoot := os.Getenv("AETHER_CODEX_SKILL_EVIDENCE_DIR")
	if !filepath.IsAbs(evidenceRoot) {
		t.Fatal("set AETHER_CODEX_SKILL_EVIDENCE_DIR to a durable absolute directory")
	}
	if err := os.MkdirAll(evidenceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	runRoot, err := os.MkdirTemp(evidenceRoot, "fresh-host-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("durable live evidence: %s", runRoot)
	root := antSkillSourceRoot(t)
	client, err := exec.LookPath("codex")
	if err != nil {
		t.Fatal(err)
	}
	client, err = filepath.EvalSymlinks(client)
	if err != nil {
		t.Fatal(err)
	}
	base := codexSkillLiveReceipt{SchemaVersion: "codex-skill-discovery/v1", Outcome: "unavailable", ExitStatus: -1, ClientPath: client}
	base.SourceRevision = strings.TrimSpace(liveSkillCommandOutput(t, root, "git", "rev-parse", "HEAD"))
	base.SourceStatus = strings.TrimSpace(liveSkillCommandOutput(t, root, "git", "status", "--short"))
	base.ClientVersion = strings.TrimSpace(liveSkillCommandOutput(t, root, client, "--version"))
	base.ClientDigest = liveSkillFileDigest(t, client)
	for _, help := range []struct {
		name string
		args []string
	}{{"codex-help.txt", []string{"--help"}}, {"codex-exec-help.txt", []string{"exec", "--help"}}} {
		liveSkillWrite(t, filepath.Join(runRoot, help.name), []byte(liveSkillCommandOutput(t, root, client, help.args...)))
	}
	base.CandidatePath = filepath.Join(runRoot, "bin", "aether")
	if err := os.MkdirAll(filepath.Dir(base.CandidatePath), 0o755); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-ldflags", "-X github.com/calcosmic/Aether/cmd.Version="+readRepoVersion(root), "-o", base.CandidatePath, "./cmd/aether")
	build.Dir = root
	buildOutput, err := build.CombinedOutput()
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build.txt"), buildOutput)
	if err != nil {
		t.Fatalf("candidate build: %v", err)
	}
	base.CandidateDigest = liveSkillFileDigest(t, base.CandidatePath)
	base.CandidateVersion = strings.TrimSpace(liveSkillCommandOutput(t, root, base.CandidatePath, "version"))
	authRoot := os.Getenv("AETHER_CODEX_SKILL_AUTH_HOME")
	if authRoot == "" {
		authRoot = os.Getenv("CODEX_HOME")
	}
	if authRoot == "" {
		hostHome, _ := os.UserHomeDir()
		authRoot = filepath.Join(hostHome, ".codex")
	}
	// Read only non-secret model preferences; never import shared plugins, MCPs,
	// instructions, permissions, or skill configuration into the isolated client.
	var preferences struct {
		Model  string `toml:"model"`
		Effort string `toml:"model_reasoning_effort"`
	}
	if raw, err := os.ReadFile(filepath.Join(authRoot, "config.toml")); err == nil {
		_, _ = toml.Decode(string(raw), &preferences)
	}
	base.Model = preferences.Model
	var receipts []codexSkillLiveReceipt
	for _, missing := range []bool{false, true} {
		name := "plan"
		if missing {
			name = "plan-missing-skill"
		}
		t.Run(name, func(t *testing.T) {
			receipt := base
			receipt.MissingSkill = missing
			caseRoot := filepath.Join(runRoot, name)
			fixtureHome, repo := filepath.Join(caseRoot, "home"), filepath.Join(caseRoot, "repository")
			for _, dir := range []string{fixtureHome, repo} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			codexHome := filepath.Join(fixtureHome, ".codex")
			receipt.DiscoveryRoot = filepath.Join(codexHome, "skills", "aether")
			receipt.SkillPath = filepath.Join(receipt.DiscoveryRoot, "ant-plan", "SKILL.md")
			receipt.RawEvents, receipt.RawStderr = filepath.Join(caseRoot, "events.jsonl"), filepath.Join(caseRoot, "stderr.txt")
			defer func() {
				liveSkillWriteJSON(t, filepath.Join(caseRoot, "receipt.json"), receipt)
				receipts = append(receipts, receipt)
			}()
			env := isolatedCodexSkillEnvironment(fixtureHome, filepath.Dir(base.CandidatePath))
			install := exec.Command(base.CandidatePath, "install", "--package-dir", root, "--home-dir", fixtureHome, "--channel", "stable", "--skip-build-binary")
			install.Dir, install.Env = repo, env
			output, err := install.CombinedOutput()
			liveSkillWrite(t, filepath.Join(caseRoot, "install.json"), output)
			var result struct {
				OK bool `json:"ok"`
			}
			if err != nil || json.Unmarshal(output, &result) != nil || !result.OK {
				receipt.Reason = "candidate install failed"
				t.Fatalf("candidate install failed: %v", err)
			}
			var ownership codexSkillOwnership
			raw, err := os.ReadFile(filepath.Join(receipt.DiscoveryRoot, ".aether-owned.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &ownership); err != nil {
				t.Fatal(err)
			}
			receipt.PayloadIdentity = ownership.PayloadIdentity
			skill, err := os.ReadFile(receipt.SkillPath)
			if err != nil {
				t.Fatal(err)
			}
			receipt.SkillDigest = lifecycleDigest(skill)
			liveSkillWrite(t, filepath.Join(caseRoot, "installed-ant-plan.md"), skill)
			if missing {
				if err := os.Remove(receipt.SkillPath); err != nil {
					t.Fatal(err)
				}
			}
			git := exec.Command("git", "init", "--quiet", repo)
			git.Env = env
			if output, err := git.CombinedOutput(); err != nil {
				t.Fatalf("disposable git init: %v %s", err, output)
			}
			// Supported file credential cache, copied without parsing/logging, then
			// removed even when authentication or discovery fails. No shared write.
			auth, err := os.ReadFile(filepath.Join(authRoot, "auth.json"))
			if err != nil {
				receipt.Reason = "file authentication unavailable"
				t.Fatal(receipt.Reason)
			}
			authPath := filepath.Join(codexHome, "auth.json")
			if err := os.WriteFile(authPath, auth, 0o600); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if err := os.Remove(authPath); err != nil && !os.IsNotExist(err) {
					t.Errorf("temporary auth cleanup: %v", err)
				} else {
					receipt.AuthRemoved = true
				}
			}()
			const prompt = "$ant-plan\nPerform only the selected skill's first read-only command-guide step, then stop. For this discovery check, first read its SKILL.md with a separate cat tool call using the exact path supplied by your discovered skill catalog. Then resolve the runtime with a separate command -v aether tool call, and run the skill's command-guide as a separate shell call in JSON output mode. Do not plan, dispatch, initialize, or mutate colony/project files. If ant-plan is absent from your discovered skills, report unavailable and stop; do not search other roots or reconstruct its instructions. Do not read authentication or other configuration files."
			liveSkillWrite(t, filepath.Join(caseRoot, "prompt.txt"), []byte(prompt))
			args := []string{"exec", "--ignore-user-config", "--ignore-rules", "-s", "read-only", "--json", "--color", "never", "-C", repo,
				"-c", `cli_auth_credentials_store="file"`, "-c", `approval_policy="never"`,
				"-c", "shell_environment_policy.set.PATH=" + fmt.Sprintf("%q", filepath.Dir(base.CandidatePath)+":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin"),
				"-c", `shell_environment_policy.set.AETHER_OUTPUT_MODE="json"`,
			}
			if preferences.Model != "" {
				args = append(args, "-m", preferences.Model)
			}
			if preferences.Effort != "" {
				args = append(args, "-c", "model_reasoning_effort="+fmt.Sprintf("%q", preferences.Effort))
			}
			args = append(args, "-")
			receipt.Args = args
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
			defer cancel()
			clientRun := exec.CommandContext(ctx, client, args...)
			clientRun.Dir, clientRun.Env, clientRun.Stdin = repo, env, strings.NewReader(prompt)
			out, err := os.Create(receipt.RawEvents)
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			errOut, err := os.Create(receipt.RawStderr)
			if err != nil {
				t.Fatal(err)
			}
			defer errOut.Close()
			clientRun.Stdout, clientRun.Stderr = out, errOut
			err = clientRun.Run()
			receipt.ExitStatus = 0
			if err != nil {
				receipt.ExitStatus = -1
				if exit, ok := err.(*exec.ExitError); ok {
					receipt.ExitStatus = exit.ExitCode()
				}
			}
			events, readErr := os.ReadFile(receipt.RawEvents)
			if readErr != nil {
				t.Fatal(readErr)
			}
			validation := validateCodexSkillLiveEvidence(&receipt, events, skill, commandGuideCatalog()["plan"].RunCommand)
			if missing {
				if validation == nil {
					receipt.Outcome = "invalid-control"
					t.Error("missing-skill control passed")
				} else if receipt.ExitStatus == 0 && bytes.Contains(events, []byte(`"type":"turn.completed"`)) {
					receipt.Outcome = "rejected-missing-skill"
					receipt.Reason = validation.Error()
				} else {
					receipt.Reason = "negative control unavailable: " + validation.Error()
					t.Error(receipt.Reason)
				}
			} else if validation != nil {
				receipt.Outcome = "discovery-failed"
				if receipt.ExitStatus != 0 {
					receipt.Outcome = "unavailable"
				}
				receipt.Reason = validation.Error()
				t.Error(receipt.Reason)
			} else {
				receipt.Outcome = "qualified"
			}
		})
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "discovery-tracer.json"), receipts)
}

func isolatedCodexSkillEnvironment(fixtureHome, bin string) []string {
	env := []string{"HOME=" + fixtureHome, "CODEX_HOME=" + filepath.Join(fixtureHome, ".codex"), "XDG_CONFIG_HOME=" + filepath.Join(fixtureHome, ".config"), "PATH=" + bin + ":/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin", "AETHER_OUTPUT_MODE=json", "AETHER_PLATFORM=codex", "AETHER_HIVE_POLICY=off"}
	for _, key := range []string{"TMPDIR", "LANG", "SSL_CERT_FILE", "SSL_CERT_DIR", "CODEX_CA_CERTIFICATE", "HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY"} {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func liveSkillWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func liveSkillWriteJSON(t *testing.T, path string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	liveSkillWrite(t, path, append(raw, '\n'))
}

func liveSkillFileDigest(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	raw, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	return lifecycleDigest(raw)
}

func liveSkillCommandOutput(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	output, err := command.Output()
	if err != nil {
		t.Fatalf("%s %v: %v", name, args, err)
	}
	return string(output)
}
