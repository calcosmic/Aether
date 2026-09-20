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
	"sort"
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
	SourceDigest             string   `json:"source_digest"`
	Scenario                 string   `json:"scenario"`
	SkillName                string   `json:"skill_name"`
	SupportPath              string   `json:"support_path"`
	SupportDigest            string   `json:"support_sha256"`
	SupportEvent             string   `json:"private_support_event,omitempty"`
	RuntimeRoute             string   `json:"route"`
	PayloadPath              string   `json:"payload_path"`
	PayloadVersion           string   `json:"payload_version"`
	CanonicalMenu            []string `json:"canonical_menu"`
	WrongPayload             bool     `json:"wrong_payload_control"`
	RawEventsDigest          string   `json:"raw_events_sha256"`
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
	receipt.SupportEvent, receipt.RuntimeRoute = "", ""
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
	commandName := strings.TrimPrefix(receipt.SkillName, "ant-")
	if commandName == "" { // Compatibility with the Plan 01 parser fixtures.
		commandName = "plan"
	}
	var support []byte
	if receipt.SchemaVersion == "codex-skill-discovery/v2" {
		if err := validateCodexSkillCandidateFiles(receipt, skill); err != nil {
			return err
		}
		support, err = os.ReadFile(receipt.SupportPath)
		if err != nil || lifecycleDigest(support) != receipt.SupportDigest {
			return fmt.Errorf("selected private support unavailable or changed: %v", err)
		}
	}
	guidePattern := regexp.MustCompile(`^(?:AETHER_OUTPUT_MODE=json\s+)?(?:aether|` + regexp.QuoteMeta(receipt.CandidatePath) + `) command-guide ` + regexp.QuoteMeta(commandName) + ` --platform codex$`)
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
		if receipt.SchemaVersion == "codex-skill-discovery/v2" && event.Type == "item.completed" {
			if item.Type == "command_execution" {
				command := unwrapCodexShellCommand(item.Command)
				if !liveSkillReadCommand(item.Command, receipt.SkillPath) && !liveSkillReadCommand(item.Command, receipt.SupportPath) && command != "command -v aether" && !guidePattern.MatchString(command) {
					return fmt.Errorf("unexpected tool command outside the read-only entry probe")
				}
			} else if item.Type != "agent_message" && item.Type != "reasoning" {
				return fmt.Errorf("unexpected host tool %s outside the read-only entry probe", item.Type)
			}
		}
		if event.Type != "item.completed" || item.Type != "command_execution" || item.Status != "completed" || item.ExitCode == nil || *item.ExitCode != 0 {
			continue
		}
		ref := fmt.Sprintf("%s:%d#%s", receipt.RawEvents, line, item.ID)
		if liveSkillReadCommand(item.Command, receipt.SkillPath) && strings.TrimSpace(item.Output) == strings.TrimSpace(string(skill)) {
			receipt.LoadedPath, receipt.LoadedDigest, receipt.SkillLoadEvent = receipt.SkillPath, receipt.SkillDigest, ref
		}
		if len(support) > 0 && liveSkillReadCommand(item.Command, receipt.SupportPath) && strings.TrimSpace(item.Output) == strings.TrimSpace(string(support)) {
			receipt.SupportEvent = ref
		}
		if unwrapCodexShellCommand(item.Command) == "command -v aether" && strings.TrimSpace(item.Output) == receipt.CandidatePath {
			receipt.CandidateResolutionEvent = ref
		}
		// A command merely echoed in prose/output is not an invocation.
		command := unwrapCodexShellCommand(item.Command)
		if guidePattern.MatchString(command) && receipt.SkillLoadEvent != "" && receipt.CandidateResolutionEvent != "" {
			var guide struct {
				OK     bool `json:"ok"`
				Result struct {
					Command    string `json:"command"`
					Platform   string `json:"platform"`
					RunCommand string `json:"run_command"`
				} `json:"result"`
			}
			if json.Unmarshal([]byte(item.Output), &guide) == nil && guide.OK && guide.Result.Command == commandName && guide.Result.Platform == "codex" && guide.Result.RunCommand == runtimeRoute {
				receipt.GuideEvent, receipt.GuideCommand = ref, command
				receipt.RuntimeRoute = guide.Result.RunCommand
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
		return fmt.Errorf("no successful actual command-guide %s --platform codex invocation", commandName)
	}
	if len(support) > 0 && receipt.SupportEvent == "" {
		return fmt.Errorf("no successful host read of the selected private support bytes")
	}
	return nil
}

func liveSkillReadCommand(command, path string) bool {
	command = unwrapCodexShellCommand(command)
	return command == "cat "+path || command == "cat '"+path+"'" || command == `cat "`+path+`"`
}

func validateCodexSkillCandidateFiles(receipt *codexSkillLiveReceipt, skill []byte) error {
	for path, digest := range map[string]string{receipt.CandidatePath: receipt.CandidateDigest, receipt.ClientPath: receipt.ClientDigest} {
		data, err := os.ReadFile(path)
		if err != nil || lifecycleDigest(data) != digest {
			return fmt.Errorf("candidate/client identity mismatch: %s", path)
		}
	}
	raw, err := os.ReadFile(filepath.Join(receipt.PayloadPath, "manifest.json"))
	if err != nil {
		return err
	}
	var payload codexSkillPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	for i := range payload.Files {
		payload.Files[i].Content, err = os.ReadFile(filepath.Join(receipt.PayloadPath, payload.Files[i].RelativePath))
		if err != nil {
			return err
		}
	}
	if err := validateCodexSkillPayload(payload); err != nil {
		return err
	}
	if codexSkillPayloadIdentity(payload) != receipt.PayloadIdentity || payload.SourceVersion != receipt.PayloadVersion {
		return fmt.Errorf("published payload identity mismatch")
	}
	matchedSkill, matchedSupport := false, false
	for _, file := range payload.Files {
		if filepath.Join(receipt.DiscoveryRoot, file.RelativePath) == receipt.SkillPath {
			matchedSkill = bytes.Equal(file.Content, skill) && file.SHA256 == receipt.SkillDigest
		}
		if filepath.Join(receipt.DiscoveryRoot, file.RelativePath) == receipt.SupportPath {
			matchedSupport = file.SHA256 == receipt.SupportDigest
		}
	}
	if !matchedSkill || !matchedSupport || receipt.SkillPath != filepath.Join(receipt.DiscoveryRoot, receipt.SkillName, "SKILL.md") {
		return fmt.Errorf("selected skill/support does not match candidate payload")
	}
	state, err := readLifecycleFileState(filepath.Join(receipt.DiscoveryRoot, ".aether-owned.json"))
	if err != nil {
		return err
	}
	owner, err := readCodexOwnership(state)
	if err != nil || owner.PayloadIdentity != receipt.PayloadIdentity {
		return fmt.Errorf("installed ownership does not match published payload: %v", err)
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

func TestCodexAntSkillFreshHost(t *testing.T) { runCodexAntSkillFreshHost(t, "clean") }

func TestCodexAntSkillFreshHostUpgrade(t *testing.T) { runCodexAntSkillFreshHost(t, "upgrade") }

func runCodexAntSkillFreshHost(t *testing.T, scenario string) {
	if os.Getenv("AETHER_CODEX_SKILL_LIVE") != "1" {
		t.Skip("opt-in real Codex provider check; no live proof claimed")
	}
	if os.Getenv("AETHER_CODEX_SKILL_LIVE_CASES") != "all" {
		t.Fatal("this tracer requires AETHER_CODEX_SKILL_LIVE_CASES=all")
	}
	evidenceRoot := os.Getenv("AETHER_CODEX_SKILL_EVIDENCE_DIR")
	if !filepath.IsAbs(evidenceRoot) {
		t.Fatal("set AETHER_CODEX_SKILL_EVIDENCE_DIR to a durable absolute directory")
	}
	if err := os.MkdirAll(evidenceRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	runRoot, err := os.MkdirTemp(evidenceRoot, "fresh-host-"+scenario+"-")
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
	base := codexSkillLiveReceipt{SchemaVersion: "codex-skill-discovery/v2", Scenario: scenario, Outcome: "unavailable", ExitStatus: -1, ClientPath: client}
	base.SourceRevision = strings.TrimSpace(liveSkillCommandOutput(t, root, "git", "rev-parse", "HEAD"))
	base.SourceStatus = strings.TrimSpace(liveSkillCommandOutput(t, root, "git", "status", "--short"))
	base.ClientVersion = strings.TrimSpace(liveSkillCommandOutput(t, root, client, "--version"))
	base.ClientDigest = liveSkillFileDigest(t, client)
	if base.ClientVersion != "codex-cli 0.154.0" {
		t.Fatalf("unqualified client version: %s", base.ClientVersion)
	}
	base.SourceDigest = liveSkillSourceIdentity(t, root, runRoot)
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
	liveSkillWrite(t, filepath.Join(runRoot, "candidate-build-info.txt"), []byte(liveSkillCommandOutput(t, root, "go", "version", "-m", base.CandidatePath)))
	// Publish into a separate disposable home. Upgrade reads this exact hub via
	// the supported hub override; the legacy consumer starts without ant files.
	publisher := filepath.Join(runRoot, "publisher")
	if err := os.MkdirAll(publisher, 0700); err != nil {
		t.Fatal(err)
	}
	publisherEnv := isolatedCodexSkillEnvironment(publisher, filepath.Dir(base.CandidatePath))
	liveSkillRuntime(t, publisher, publisherEnv, filepath.Join(runRoot, "publish-via-install.json"), base.CandidatePath, "install", "--package-dir", root, "--home-dir", publisher, "--channel", "stable", "--skip-build-binary")
	hub := filepath.Join(publisher, ".aether")
	published := antReadPublished(t, hub)
	base.PayloadIdentity, base.PayloadVersion = codexSkillPayloadIdentity(published), published.SourceVersion
	base.PayloadPath = filepath.Join(hub, "system", "codex-skills")
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
	defer func() { liveSkillWriteJSON(t, filepath.Join(runRoot, "fresh-codex-skills.json"), receipts) }()
	// This independent test inventory is shared with the registered guide tests,
	// never derived from the production generator under test.
	type liveCase struct{ command, support, route, control string }
	var cases []liveCase
	for _, want := range codexAntGuideExpectations {
		cases = append(cases, liveCase{want.command, want.support, want.runtime, ""})
	}
	for _, control := range []string{"missing-skill", "wrong-payload"} {
		want := codexAntGuideExpectations[4]
		cases = append(cases, liveCase{want.command, want.support, want.runtime, control})
	}
	for _, test := range cases {
		name := test.command
		if test.control != "" {
			name += "-" + test.control
		}
		t.Run(name, func(t *testing.T) {
			receipt := base
			receipt.SkillName = "ant-" + test.command
			receipt.MissingSkill, receipt.WrongPayload = test.control == "missing-skill", test.control == "wrong-payload"
			caseRoot := filepath.Join(runRoot, name)
			fixtureHome, repo := filepath.Join(caseRoot, "home"), filepath.Join(caseRoot, "repository")
			for _, dir := range []string{fixtureHome, repo} {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			codexHome := filepath.Join(fixtureHome, ".codex")
			receipt.DiscoveryRoot = filepath.Join(codexHome, "skills", "aether")
			receipt.SkillPath = filepath.Join(receipt.DiscoveryRoot, receipt.SkillName, "SKILL.md")
			receipt.RawEvents, receipt.RawStderr = filepath.Join(caseRoot, "events.jsonl"), filepath.Join(caseRoot, "stderr.txt")
			defer func() {
				liveSkillWriteJSON(t, filepath.Join(caseRoot, "receipt.json"), receipt)
				receipts = append(receipts, receipt)
			}()
			env := isolatedCodexSkillEnvironment(fixtureHome, filepath.Dir(base.CandidatePath))
			git := exec.Command("git", "init", "--quiet", repo)
			git.Env = env
			if output, err := git.CombinedOutput(); err != nil {
				t.Fatalf("disposable git init: %v %s", err, output)
			}
			if scenario == "upgrade" {
				// The registered updater requires the existing platform-home roots
				// of an installed system; only its legacy Codex files are seeded.
				for _, path := range []string{filepath.Join(fixtureHome, ".claude"), filepath.Join(fixtureHome, ".config", "opencode")} {
					if err := os.MkdirAll(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
				antSeedLegacy(t, receipt.DiscoveryRoot)
				liveSkillWriteJSON(t, filepath.Join(caseRoot, "legacy-before.json"), antSnapshot(t, receipt.DiscoveryRoot))
				env = append(env, "AETHER_HUB_DIR="+hub)
				liveSkillRuntime(t, repo, env, filepath.Join(caseRoot, "update.json"), base.CandidatePath, "update")
			} else {
				liveSkillRuntime(t, repo, env, filepath.Join(caseRoot, "install.json"), base.CandidatePath, "install", "--package-dir", root, "--home-dir", fixtureHome, "--channel", "stable", "--skip-build-binary")
			}
			liveSkillAssertMenu(t, &receipt)
			var ownership codexSkillOwnership
			raw, err := os.ReadFile(filepath.Join(receipt.DiscoveryRoot, ".aether-owned.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &ownership); err != nil {
				t.Fatal(err)
			}
			if receipt.PayloadIdentity != ownership.PayloadIdentity {
				t.Fatal("installed and published payload identities differ")
			}
			receipt.SupportPath = filepath.Join(receipt.DiscoveryRoot, "support", test.support+".md")
			receipt.SupportDigest = liveSkillFileDigest(t, receipt.SupportPath)
			skill, err := os.ReadFile(receipt.SkillPath)
			if err != nil {
				t.Fatal(err)
			}
			receipt.SkillDigest = lifecycleDigest(skill)
			liveSkillWrite(t, filepath.Join(caseRoot, "installed-"+receipt.SkillName+".md"), skill)
			if receipt.MissingSkill {
				if err := os.Remove(receipt.SkillPath); err != nil {
					t.Fatal(err)
				}
			}
			if receipt.WrongPayload {
				// Only the negative control edits installed bytes; preserve the
				// expected candidate digest so plausible instructions cannot pass.
				liveSkillWrite(t, receipt.SkillPath, append(append([]byte(nil), skill...), []byte("\nWrong-payload negative control.\n")...))
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
			prompt := fmt.Sprintf("$%s\nPerform only the selected skill's first read-only command-guide step and read its referenced private support, then stop. First read its SKILL.md with a separate cat tool call using the exact path supplied by your discovered skill catalog. Resolve the runtime with a separate command -v aether tool call. Run the skill's command-guide as a separate shell call in JSON output mode. Read its private support with a separate cat call using the absolute path resolved relative to that SKILL.md. Report the guide's next runtime route without executing it. Do not ask owner questions, dispatch helpers, initialize, or mutate colony/project files. If %s is absent from your discovered skills, report unavailable and stop; do not search other roots or reconstruct its instructions. Do not read authentication or other configuration files.", receipt.SkillName, receipt.SkillName)
			liveSkillWrite(t, filepath.Join(caseRoot, "prompt.txt"), []byte(prompt))
			args := []string{"exec", "--ephemeral", "--ignore-user-config", "--ignore-rules", "-s", "read-only", "--json", "--color", "never", "-C", repo,
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
			repositoryBefore := antSnapshot(t, repo)
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
			receipt.RawEventsDigest = lifecycleDigest(events)
			antAssertSnapshot(t, repo, repositoryBefore)
			validation := validateCodexSkillLiveEvidence(&receipt, events, skill, test.route)
			if test.control != "" {
				if receipt.WrongPayload {
					// Require the live process actually to load the altered bytes and
					// execute the guide, then reject them against the candidate.
					altered := mustReadLifecycleFixtureFile(t, receipt.SkillPath)
					observed := receipt
					observed.SchemaVersion = "codex-skill-discovery/v1"
					observed.SkillDigest = lifecycleDigest(altered)
					if err := validateCodexSkillLiveEvidence(&observed, events, altered, test.route); err != nil {
						receipt.Reason = "wrong-payload control did not exercise host loading: " + err.Error()
						t.Fatal(receipt.Reason)
					}
				}
				if validation == nil {
					receipt.Outcome = "invalid-control"
					t.Error("negative control passed")
				} else if receipt.ExitStatus == 0 && bytes.Contains(events, []byte(`"type":"turn.completed"`)) {
					receipt.Outcome = "rejected-" + test.control
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

// The hash list includes all tracked production inputs, including embed assets;
// tests and planning prose are separate and cannot silently change a candidate.
func liveSkillSourceIdentity(t *testing.T, root, runRoot string) string {
	t.Helper()
	files := strings.Split(liveSkillCommandOutput(t, root, "git", "ls-files", "-z"), "\x00")
	sort.Strings(files)
	identities := map[string]string{}
	var aggregate strings.Builder
	for _, path := range files {
		if path == "" || strings.HasPrefix(path, ".planning/") || strings.HasSuffix(path, "_test.go") {
			continue
		}
		fullPath := filepath.Join(root, path)
		info, err := os.Lstat(fullPath)
		if err != nil {
			t.Fatal(err)
		}
		var data []byte
		if info.Mode()&os.ModeSymlink != 0 {
			var target string
			target, err = os.Readlink(fullPath)
			data = []byte("symlink:" + target)
		} else {
			data, err = os.ReadFile(fullPath)
		}
		if err != nil {
			t.Fatal(err)
		}
		digest := lifecycleDigest(data)
		identities[path] = digest
		fmt.Fprintf(&aggregate, "%s\x00%s\n", path, digest)
	}
	liveSkillWriteJSON(t, filepath.Join(runRoot, "source-files.json"), identities)
	liveSkillWrite(t, filepath.Join(runRoot, "source.patch"), []byte(liveSkillCommandOutput(t, root, "git", "diff", "HEAD", "--", "cmd", "pkg", ".aether", "go.mod", "go.sum")))
	return lifecycleDigest([]byte(aggregate.String()))
}

func liveSkillRuntime(t *testing.T, dir string, env []string, log, candidate string, args ...string) {
	t.Helper()
	command := exec.Command(candidate, args...)
	command.Dir, command.Env = dir, env
	output, err := command.CombinedOutput()
	liveSkillWrite(t, log, output)
	status := 0
	if err != nil {
		status = -1
		if exit, ok := err.(*exec.ExitError); ok {
			status = exit.ExitCode()
		}
	}
	liveSkillWriteJSON(t, log+".receipt.json", map[string]any{"argv": append([]string{candidate}, args...), "cwd": dir, "exit_code": status, "sha256": lifecycleDigest(output)})
	var result struct {
		OK bool `json:"ok"`
	}
	if err != nil || json.Unmarshal(output, &result) != nil || !result.OK {
		t.Fatalf("candidate %v failed: %v; raw output %s", args, err, log)
	}
}

func liveSkillAssertMenu(t *testing.T, receipt *codexSkillLiveReceipt) {
	t.Helper()
	var got, want []string
	if err := filepath.WalkDir(receipt.DiscoveryRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Name() == "SKILL.md" {
			rel, err := filepath.Rel(receipt.DiscoveryRoot, path)
			if err != nil {
				return err
			}
			got = append(got, filepath.ToSlash(rel))
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range codexAntGuideExpectations {
		want = append(want, "ant-"+expected.command+"/SKILL.md")
	}
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("canonical menu differs: %v", got)
	}
	receipt.CanonicalMenu = got
}

func liveSkillToolEvent(id, command, output string) string {
	raw, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]any{"id": id, "type": "command_execution", "command": command, "aggregated_output": output, "exit_code": 0, "status": "completed"}})
	return string(raw) + "\n"
}

func TestCodexAntSkillReceiptValidation(t *testing.T) {
	if executable, err := os.Executable(); err == nil {
		t.Logf("executed Go test binary %s %s", executable, liveSkillFileDigest(t, executable))
	} else {
		t.Fatal(err)
	}
	if path := os.Getenv("AETHER_CODEX_SKILL_RECEIPT_PATH"); path != "" {
		t.Run("recorded-capture", func(t *testing.T) { liveSkillValidateRecorded(t, path) })
	}
	for _, want := range codexAntGuideExpectations {
		t.Run(want.command, func(t *testing.T) {
			home := t.TempDir()
			payload := antPayload(t)
			if result := syncCodexSkillsFromPayload(payload, home); len(result.errors) > 0 {
				t.Fatal(result.errors)
			}
			root := filepath.Join(home, ".codex", "skills", "aether")
			published := filepath.Join(home, "published")
			for _, file := range payload.Files {
				writeMaintenanceMutation199File(t, filepath.Join(published, file.RelativePath), file.Content)
			}
			liveSkillWriteJSON(t, filepath.Join(published, "manifest.json"), payload)
			candidate, client := filepath.Join(home, "aether"), filepath.Join(home, "codex")
			liveSkillWrite(t, candidate, []byte("candidate parser fixture"))
			liveSkillWrite(t, client, []byte("client parser fixture"))
			base := codexSkillLiveReceipt{SchemaVersion: "codex-skill-discovery/v2", DiscoveryRoot: root, SkillName: "ant-" + want.command,
				SkillPath: filepath.Join(root, "ant-"+want.command, "SKILL.md"), SupportPath: filepath.Join(root, "support", want.support+".md"),
				CandidatePath: candidate, CandidateDigest: liveSkillFileDigest(t, candidate), ClientPath: client, ClientDigest: liveSkillFileDigest(t, client),
				PayloadPath: published, PayloadIdentity: codexSkillPayloadIdentity(payload), PayloadVersion: payload.SourceVersion, RawEvents: "parser-fixture.jsonl"}
			skill := mustReadLifecycleFixtureFile(t, base.SkillPath)
			support := mustReadLifecycleFixtureFile(t, base.SupportPath)
			base.SkillDigest, base.SupportDigest = lifecycleDigest(skill), lifecycleDigest(support)
			read := liveSkillToolEvent("skill", "cat "+base.SkillPath, string(skill))
			private := liveSkillToolEvent("support", "cat "+base.SupportPath, string(support))
			resolve := liveSkillToolEvent("resolve", "command -v aether", candidate+"\n")
			guideJSON, _ := json.Marshal(map[string]any{"ok": true, "result": map[string]string{"command": want.command, "platform": "codex", "run_command": want.runtime}})
			guide := liveSkillToolEvent("guide", "AETHER_OUTPUT_MODE=json aether command-guide "+want.command+" --platform codex", string(guideJSON))
			const done = "{\"type\":\"turn.completed\"}\n"
			events := read + resolve + guide + private + done
			t.Run("valid", func(t *testing.T) {
				r := base
				if err := validateCodexSkillLiveEvidence(&r, []byte(events), skill, want.runtime); err != nil {
					t.Fatal(err)
				}
				if r.SupportEvent == "" || r.RuntimeRoute != want.runtime {
					t.Fatal("missing proof links")
				}
			})
			for _, test := range []struct {
				name, events string
				mutate       func(*codexSkillLiveReceipt)
			}{
				{"no-private-read", read + resolve + guide + done, nil},
				{"echoed-file", strings.Replace(events, "cat ", "echo cat ", 1), nil},
				{"wrong-route", strings.Replace(events, "run_command", "untrusted_route", 1), nil},
				{"incomplete", strings.TrimSuffix(events, done), nil},
				{"extra-runtime-call", read + resolve + guide + private + liveSkillToolEvent("extra", "aether build 1", "") + done, nil},
				{"guide-before-load", guide + read + resolve + private + done, nil},
				{"wrong-candidate", events, func(r *codexSkillLiveReceipt) { r.CandidateDigest = "sha256:wrong" }},
				{"wrong-payload", events, func(r *codexSkillLiveReceipt) { r.PayloadIdentity = "sha256:wrong" }},
				{"wrong-support", events, func(r *codexSkillLiveReceipt) { r.SupportDigest = "sha256:wrong" }},
				{"global-duplicate", events, func(r *codexSkillLiveReceipt) { r.DiscoveryRoot = filepath.Join(home, "other") }},
				{"failed-client", events, func(r *codexSkillLiveReceipt) { r.ExitStatus = 1 }},
			} {
				t.Run(test.name, func(t *testing.T) {
					r := base
					r.SkillLoadEvent, r.GuideEvent, r.SupportEvent = "stale", "stale", "stale"
					if test.mutate != nil {
						test.mutate(&r)
					}
					if err := validateCodexSkillLiveEvidence(&r, []byte(test.events), skill, want.runtime); err == nil {
						t.Fatal("unproved receipt accepted")
					}
				})
			}
			t.Run("missing-skill", func(t *testing.T) {
				if err := os.Remove(base.SkillPath); err != nil {
					t.Fatal(err)
				}
				r := base
				if err := validateCodexSkillLiveEvidence(&r, []byte(events), skill, want.runtime); err == nil {
					t.Fatal("missing file passed with cached events")
				}
			})
		})
	}
}

func liveSkillValidateRecorded(t *testing.T, path string) {
	t.Helper()
	var bundle struct {
		Cases []codexSkillLiveReceipt `json:"cases"`
	}
	if err := json.Unmarshal(mustReadLifecycleFixtureFile(t, path), &bundle); err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{}
	for _, scenario := range []string{"clean", "upgrade"} {
		for _, want := range codexAntGuideExpectations {
			wanted[scenario+"/ant-"+want.command] = true
		}
		for _, control := range []string{"missing-skill", "wrong-payload"} {
			wanted[scenario+"/ant-plan/"+control] = true
		}
	}
	if len(bundle.Cases) != len(wanted) {
		t.Fatalf("recorded cases: %d, want %d", len(bundle.Cases), len(wanted))
	}
	for _, receipt := range bundle.Cases {
		key := receipt.Scenario + "/" + receipt.SkillName
		if receipt.MissingSkill {
			key += "/missing-skill"
		}
		if receipt.WrongPayload {
			key += "/wrong-payload"
		}
		if !wanted[key] {
			t.Fatalf("unexpected/duplicate capture %s", key)
		}
		delete(wanted, key)
		if receipt.SchemaVersion != "codex-skill-discovery/v2" || receipt.SourceDigest == "" || receipt.ClientVersion != "codex-cli 0.154.0" || !receipt.AuthRemoved {
			t.Fatalf("incomplete capture identity: %s", key)
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.Dir(receipt.DiscoveryRoot)), "auth.json")); !os.IsNotExist(err) {
			t.Fatalf("temporary auth still present: %s", key)
		}
		events := mustReadLifecycleFixtureFile(t, receipt.RawEvents)
		if lifecycleDigest(events) != receipt.RawEventsDigest {
			t.Fatalf("changed raw events: %s", key)
		}
		skill := mustReadLifecycleFixtureFile(t, filepath.Join(filepath.Dir(receipt.RawEvents), "installed-"+receipt.SkillName+".md"))
		route := ""
		for _, want := range codexAntGuideExpectations {
			if receipt.SkillName == "ant-"+want.command {
				route = want.runtime
			}
		}
		copy := receipt
		err := validateCodexSkillLiveEvidence(&copy, events, skill, route)
		if receipt.MissingSkill || receipt.WrongPayload {
			if err == nil || receipt.ExitStatus != 0 || !strings.HasPrefix(receipt.Outcome, "rejected-") || !bytes.Contains(events, []byte(`"type":"turn.completed"`)) {
				t.Fatalf("invalid recorded negative control %s: %v", key, err)
			}
		} else if err != nil || receipt.Outcome != "qualified" || copy.LoadedDigest != receipt.LoadedDigest || copy.SupportEvent != receipt.SupportEvent || copy.GuideEvent != receipt.GuideEvent || copy.RuntimeRoute != receipt.RuntimeRoute {
			t.Fatalf("unqualified or changed positive capture %s: %v", key, err)
		}
	}
}
