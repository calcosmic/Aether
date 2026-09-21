package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestCodexBuildGuideNativeOptInIsUnchanged locks the parked native Codex
// build guide byte-for-byte behind its opt-in. The digest below was captured
// by running a throwaway test at unmodified HEAD (commit 96ff6f45), before
// any Phase-204.2-parking edit existed, via:
//
//	go test ./cmd -count=1 -run TestZZZCaptureNativeBuildGuideDigest -v
//
// which logged the hex SHA-256 of
// json.Marshal(codexNativeBuildCommandGuide(commandGuideCatalog()["build"]))
// and its byte length (18458).
func TestCodexBuildGuideNativeOptInIsUnchanged(t *testing.T) {
	t.Setenv(codexNativeBuildOptInEnv, "1")
	def := commandGuideCatalog()["build"]
	adapted := adaptCommandGuideDefinitionForPlatform("build", "codex", def)
	native := codexNativeBuildCommandGuide(def)
	if !reflect.DeepEqual(adapted, native) {
		t.Fatalf("opt-in codex build guide diverged from the native guide")
	}

	b, err := json.Marshal(native)
	if err != nil {
		t.Fatalf("marshal native guide: %v", err)
	}
	if len(b) <= 1000 {
		t.Fatalf("marshaled native guide unexpectedly short: %d bytes", len(b))
	}
	sum := sha256.Sum256(b)
	got := hex.EncodeToString(sum[:])
	// Re-pinned in 1.0.88. The one and only change to the parked native guide
	// was removing the `AETHER_FORCE_COLOR=1 ` prefix from its two ceremony
	// render commands (git diff of cmd/command_guide.go shows nothing else):
	// forced colour put escape codes into screens a chat now shows to the
	// owner. Previous digest: a00b386a91ae0773dfda16ad9eb8427a713c444a942c56b466f65ac3f5d3da81.
	const wantDigest = "69e79d0415a6226e5789dcd8abe1bb05b957d0080918edfd96f9d7e8a7430bed"
	if got != wantDigest {
		t.Fatalf("native build guide changed since it was parked: digest = %s, want %s", got, wantDigest)
	}
}

const codexDirectBuildRunCommand = "AETHER_OUTPUT_MODE=visual aether build <phase>"
const codexNativeBuildRunCommand = "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by codex-native-worker stage>"

// TestCodexBuildGuideDefaultsToDirectRoute asserts the proven direct-dispatch
// route is the Codex default, the parked native bridge is reachable only
// through the exact opt-in value, and the opt-in never leaks into the
// Claude/OpenCode guides.
func TestCodexBuildGuideDefaultsToDirectRoute(t *testing.T) {
	t.Setenv(codexNativeBuildOptInEnv, "")
	guide, err := buildCommandGuide("build", "codex")
	if err != nil {
		t.Fatal(err)
	}
	var parts []string
	parts = append(parts, guide.Intent)
	parts = append(parts, guide.PreSteps...)
	parts = append(parts, guide.RunCommand)
	parts = append(parts, guide.PostSteps...)
	parts = append(parts, guide.DriftGuards...)
	parts = append(parts, guide.RawBypass)
	text := strings.Join(parts, "\n")

	for _, forbidden := range []string{"codex-native-worker", "spawn_agent", "--completion-file", "build-completion-stage", "aether spawn-log", "worker.native"} {
		if strings.Contains(text, forbidden) {
			t.Errorf("default codex build guide still contains %q", forbidden)
		}
	}
	if guide.RunCommand != codexDirectBuildRunCommand {
		t.Errorf("RunCommand = %q, want %q", guide.RunCommand, codexDirectBuildRunCommand)
	}
	if !strings.Contains(text, "Do not pass `--plan-only`") {
		t.Error("default guide missing the --plan-only instruction")
	}
	if !strings.Contains(text, "blocker_advisory") {
		t.Error("default guide missing blocker_advisory guidance")
	}
	if len(guide.PostSteps) == 0 || !strings.Contains(guide.PostSteps[len(guide.PostSteps)-1], "Route first to `aether continue`") {
		t.Errorf("last PostStep does not route to continue: %v", guide.PostSteps)
	}

	table := map[string]bool{"": false, "0": false, "true": false, "yes": false, "1": true, " 1 ": true}
	for value, wantNative := range table {
		value, wantNative := value, wantNative
		t.Run("env="+value, func(t *testing.T) {
			t.Setenv(codexNativeBuildOptInEnv, value)
			g, err := buildCommandGuide("build", "codex")
			if err != nil {
				t.Fatal(err)
			}
			want := codexDirectBuildRunCommand
			if wantNative {
				want = codexNativeBuildRunCommand
			}
			if g.RunCommand != want {
				t.Errorf("env %q: RunCommand = %q, want %q", value, g.RunCommand, want)
			}
		})
	}

	t.Setenv(codexNativeBuildOptInEnv, "")
	claudeDefault, err := buildCommandGuide("build", "claude")
	if err != nil {
		t.Fatal(err)
	}
	opencodeDefault, err := buildCommandGuide("build", "opencode")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(codexNativeBuildOptInEnv, "1")
	claudeNative, err := buildCommandGuide("build", "claude")
	if err != nil {
		t.Fatal(err)
	}
	opencodeNative, err := buildCommandGuide("build", "opencode")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(claudeDefault, claudeNative) {
		t.Error("claude build guide changed with the codex-only native opt-in")
	}
	if !reflect.DeepEqual(opencodeDefault, opencodeNative) {
		t.Error("opencode build guide changed with the codex-only native opt-in")
	}
}

// TestCodexAntBuildSkillPayloadFollowsNativeOptIn asserts the generated
// public Codex skill payload for ant-build follows the same opt-in, while
// every other generated file is unaffected.
func TestCodexAntBuildSkillPayloadFollowsNativeOptIn(t *testing.T) {
	root := antSkillSourceRoot(t)

	t.Setenv(codexNativeBuildOptInEnv, "")
	direct, err := buildCodexSkillPayload(root)
	if err != nil {
		t.Fatalf("default payload: %v", err)
	}
	t.Setenv(codexNativeBuildOptInEnv, "1")
	native, err := buildCodexSkillPayload(root)
	if err != nil {
		t.Fatalf("opt-in payload: %v", err)
	}

	find := func(t *testing.T, p codexSkillPayload, rel string) codexSkillPayloadFile {
		t.Helper()
		for _, f := range p.Files {
			if f.RelativePath == rel {
				return f
			}
		}
		t.Fatalf("payload missing file %s", rel)
		return codexSkillPayloadFile{}
	}

	directBuild := find(t, direct, "ant-build/SKILL.md")
	nativeBuild := find(t, native, "ant-build/SKILL.md")
	if strings.Contains(string(directBuild.Content), "codex-native-worker") {
		t.Error("default ant-build/SKILL.md still contains native strings")
	}
	if !strings.Contains(string(directBuild.Content), "## Runtime Command\n`"+codexDirectBuildRunCommand+"`") {
		t.Errorf("default ant-build/SKILL.md missing direct Runtime Command:\n%s", directBuild.Content)
	}
	if !strings.Contains(string(nativeBuild.Content), "codex-native-worker reserve") {
		t.Error("opt-in ant-build/SKILL.md missing native strings")
	}

	if len(direct.Files) != len(native.Files) {
		t.Fatalf("file count changed between opt-in states: default=%d opt-in=%d", len(direct.Files), len(native.Files))
	}
	checked := 0
	for _, f := range direct.Files {
		if f.RelativePath == "ant-build/SKILL.md" {
			continue
		}
		checked++
		other := find(t, native, f.RelativePath)
		if f.SHA256 != other.SHA256 || !bytes.Equal(f.Content, other.Content) {
			t.Errorf("%s changed between opt-in states", f.RelativePath)
		}
	}
	if checked != 11 {
		t.Fatalf("expected exactly eleven other files, checked %d", checked)
	}
}

// TestCodexPlanOnlyContextProtocolRequiresNativeOptIn asserts the
// child-fetch/v1 manifest protocol pin (cmd/codex_build.go) and the provider
// lane check it triggers (validateBuildWorkerProviderLane,
// cmd/build_worker_run.go) both stay gated on the same opt-in.
func TestCodexPlanOnlyContextProtocolRequiresNativeOptIn(t *testing.T) {
	cases := []struct {
		name         string
		platform     string
		optIn        string
		wantProtocol string
		wantLaneErr  bool
	}{
		{"codex-default", "codex", "", "", false},
		{"codex-optin", "codex", "1", codexNativeContextProtocolChildFetch, true},
		{"claude-optin", "claude", "1", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			root := setupExternalBuildAttemptTest(t)
			t.Setenv("AETHER_ACTIVE_PLATFORM", tc.platform)
			t.Setenv(codexNativeBuildOptInEnv, tc.optIn)
			result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
			if err != nil {
				t.Fatal(err)
			}
			manifest := result["dispatch_manifest"].(codexBuildManifest)
			if manifest.ContextProtocol != tc.wantProtocol {
				t.Fatalf("ContextProtocol = %q, want %q", manifest.ContextProtocol, tc.wantProtocol)
			}
			_, attempt, ok := loadLatestBuildAttempt(1)
			if !ok {
				t.Fatal("no durable build attempt after plan-only")
			}
			laneErr := validateBuildWorkerProviderLane(attempt)
			if tc.wantLaneErr && laneErr == nil {
				t.Error("expected validateBuildWorkerProviderLane to refuse generic dispatch, got nil")
			}
			if !tc.wantLaneErr && laneErr != nil {
				t.Errorf("expected validateBuildWorkerProviderLane to allow generic dispatch, got %v", laneErr)
			}
		})
	}
}

// TestCodexAntSkillSourceCheckFollowsNativeOptIn asserts `source-check`
// passes in both opt-in states (contracts adjust with the same switch), and
// still catches a stale native-generated shim checked under the default
// (direct) env.
func TestCodexAntSkillSourceCheckFollowsNativeOptIn(t *testing.T) {
	antPayloadEnvironment(t)
	root := minimalSourceCheckRoot(t)
	original := sourceCheckCodexShims
	t.Cleanup(func() { sourceCheckCodexShims = original })

	for _, env := range []string{"", "1"} {
		env := env
		t.Run("clean/env="+env, func(t *testing.T) {
			sourceCheckCodexShims = original
			t.Setenv(codexNativeBuildOptInEnv, env)
			resetFlags(rootCmd)
			output, err := antPayloadCommand(t, "source-check", "--root", root, "--json")
			var result sourceCheckResult
			if decodeErr := json.Unmarshal([]byte(output), &result); decodeErr != nil {
				t.Fatalf("checker output: %v", decodeErr)
			}
			if err != nil || !result.OK {
				t.Fatalf("valid surface refused under env=%q: %v %+v", env, err, result.Issues)
			}
		})
	}

	t.Run("stale-native-shim-under-default-env", func(t *testing.T) {
		t.Setenv(codexNativeBuildOptInEnv, "1")
		nativeShims := codexCommandSkillShims()
		t.Setenv(codexNativeBuildOptInEnv, "")
		sourceCheckCodexShims = func() []codexSkillShim { return nativeShims }
		resetFlags(rootCmd)
		output, err := antPayloadCommand(t, "source-check", "--root", root, "--json")
		var result sourceCheckResult
		if decodeErr := json.Unmarshal([]byte(output), &result); decodeErr != nil {
			t.Fatalf("checker output: %v", decodeErr)
		}
		if err == nil || result.OK {
			t.Fatalf("stale native shim under default env should fail source-check: %v %+v", err, result)
		}
		found := false
		for _, issue := range result.Issues {
			if issue.Area == "codex_skills" && issue.Path == "system/codex-skills/ant-build/SKILL.md" && strings.Contains(issue.Message, "invalid runtime route") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected invalid runtime route issue on ant-build/SKILL.md, got: %+v", result.Issues)
		}
	})
}
