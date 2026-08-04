package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// resetHivePolicyWarnOnceForTest resets the package-level once-guard on the
// unrecognized-AETHER_HIVE_POLICY-value stderr warning so each subtest can
// independently assert whether the warning fires. Test-only: production code
// never needs to re-arm the guard within a single process.
func resetHivePolicyWarnOnceForTest() {
	hivePolicyWarnOnce = sync.Once{}
}

// TestHiveWorkerReadIsOnByDefaultAndOffSwitchDisablesIt supersedes the
// pre-D-02 "off by default, twice-gated" test. D-02 retired the per-colony
// consent file: AETHER_HIVE_POLICY is now the only control surface, unset
// means promote (worker reads succeed), and explicit "off" is the sole
// compensating control (T-162-03).
func TestHiveWorkerReadIsOnByDefaultAndOffSwitchDisablesIt(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	if err := os.MkdirAll(filepath.Join(hub, "hive"), 0o755); err != nil {
		t.Fatalf("create hive: %v", err)
	}
	data := `{"entries":[{"id":"go-safe","text":"Prefer table tests","domain":"go","source_repo":"one","confidence":0.9}]}`
	if err := os.WriteFile(filepath.Join(hub, "hive", "wisdom.json"), []byte(data), 0o644); err != nil {
		t.Fatalf("write wisdom: %v", err)
	}

	t.Setenv(hivePolicyEnv, "")
	resetFlags(rootCmd)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"hive-read", "--for-worker"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worker hive-read: %v", err)
	}
	workerEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	workerResult := workerEnvelope["result"].(map[string]interface{})
	if workerResult["enabled"] != true || int(workerResult["total"].(float64)) != 1 {
		t.Fatalf("default (unset) worker read should return hive wisdom: %+v", workerResult)
	}

	resetFlags(rootCmd)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"hive-read"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("manual hive-read: %v", err)
	}
	manualEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	manualResult := manualEnvelope["result"].(map[string]interface{})
	if int(manualResult["total"].(float64)) != 1 {
		t.Fatalf("manual hive-read should remain inspectable: %+v", manualResult)
	}

	t.Setenv(hivePolicyEnv, "off")
	resetFlags(rootCmd)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"hive-read", "--for-worker"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("worker hive-read with policy off: %v", err)
	}
	offEnvelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	offResult := offEnvelope["result"].(map[string]interface{})
	if offResult["enabled"] != false || int(offResult["total"].(float64)) != 0 {
		t.Fatalf("AETHER_HIVE_POLICY=off should disable worker read: %+v", offResult)
	}
}

func TestSealLeavesEligibleInstinctsLocalWhenHivePolicyIsOff(t *testing.T) {
	saveGlobals(t)
	t.Setenv(hivePolicyEnv, "off")
	s, _ := newTestStore(t)
	store = s
	t.Setenv("AETHER_HUB_DIR", t.TempDir())
	stdout = &bytes.Buffer{}
	goal := "Keep project knowledge scoped"
	state := colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Done", Status: colony.PhaseCompleted}}},
	}
	instincts := colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{{
		ID: "local-only", Trigger: "test", Action: "Use project-specific convention", Domain: "go", Confidence: 0.95,
	}}}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("save instincts: %v", err)
	}
	if err := completeSealRuntime(state); err != nil {
		t.Fatalf("complete seal: %v", err)
	}
	if !strings.Contains(stdout.(*bytes.Buffer).String(), "Hive auto-promotion is disabled") {
		t.Fatalf("seal did not disclose disabled promotion: %s", stdout.(*bytes.Buffer).String())
	}
	wisdomPath := filepath.Join(resolveHubPath(), "hive", "wisdom.json")
	if data, err := os.ReadFile(wisdomPath); err == nil {
		var wisdom hiveWisdomData
		if json.Unmarshal(data, &wisdom) == nil && len(wisdom.Entries) > 0 {
			t.Fatalf("seal promoted project instinct while policy was off: %+v", wisdom.Entries)
		}
	}
}

func TestAutomaticHivePromoteCommandHonorsSafeDefault(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv(hivePolicyEnv, "off")
	hub := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hub)
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{
		"hive-promote",
		"--automatic",
		"--text", "Prefer the project-specific layout",
		"--domain", "go",
		"--source-repo", "project-one",
		"--confidence", "0.9",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("automatic hive-promote: %v", err)
	}
	envelope := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := envelope["result"].(map[string]interface{})
	if result["promoted"] != false || result["skipped"] != true || result["policy"] != "off" {
		t.Fatalf("automatic promotion did not fail closed: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(hub, "hive", "wisdom.json")); !os.IsNotExist(err) {
		t.Fatalf("automatic promotion wrote global wisdom while disabled: %v", err)
	}
}

// TestHiveRuntimePolicyDefault pins the D-02 single-switch resolution table:
// unset means promote (on, full), explicit "off" still means off, and
// unrecognized values fail safe to off (Pitfall 4 / RESEARCH.md A3).
func TestHiveRuntimePolicyDefault(t *testing.T) {
	tests := []struct {
		name  string
		value string
		unset bool
		want  hiveRuntimePolicy
	}{
		{name: "unset", unset: true, want: hivePolicyPromote},
		{name: "empty string", value: "", want: hivePolicyPromote},
		{name: "off", value: "off", want: hivePolicyOff},
		{name: "OFF uppercase", value: "OFF", want: hivePolicyOff},
		{name: "off with whitespace", value: "  off  ", want: hivePolicyOff},
		{name: "read", value: "read", want: hivePolicyRead},
		{name: "inject", value: "inject", want: hivePolicyRead},
		{name: "on", value: "on", want: hivePolicyRead},
		{name: "promote", value: "promote", want: hivePolicyPromote},
		{name: "full", value: "full", want: hivePolicyPromote},
		{name: "unrecognized value", value: "raed", want: hivePolicyOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				// t.Setenv cannot express "unset" — restore manually.
				prior, hadPrior := os.LookupEnv(hivePolicyEnv)
				if err := os.Unsetenv(hivePolicyEnv); err != nil {
					t.Fatalf("unsetenv: %v", err)
				}
				t.Cleanup(func() {
					if hadPrior {
						os.Setenv(hivePolicyEnv, prior)
					}
				})
			} else {
				t.Setenv(hivePolicyEnv, tt.value)
			}

			resetHivePolicyWarnOnceForTest()

			if got := currentHiveRuntimePolicy(); got != tt.want {
				t.Errorf("currentHiveRuntimePolicy() = %q, want %q", got, tt.want)
			}
		})
	}

	t.Run("unset enables automatic read and promotion", func(t *testing.T) {
		prior, hadPrior := os.LookupEnv(hivePolicyEnv)
		if err := os.Unsetenv(hivePolicyEnv); err != nil {
			t.Fatalf("unsetenv: %v", err)
		}
		t.Cleanup(func() {
			if hadPrior {
				os.Setenv(hivePolicyEnv, prior)
			}
		})
		resetHivePolicyWarnOnceForTest()

		if !automaticHiveReadEnabled() {
			t.Error("automaticHiveReadEnabled() = false, want true with AETHER_HIVE_POLICY unset (D-01/D-02: on by default)")
		}
		if !automaticHivePromotionEnabled() {
			t.Error("automaticHivePromotionEnabled() = false, want true with AETHER_HIVE_POLICY unset (D-01/D-02: on by default)")
		}
	})
}

// TestHiveRuntimePolicyUnrecognizedWarns pins T-162-04's compensating
// control: an unrecognized AETHER_HIVE_POLICY value must be discoverable,
// not a silent fail-safe.
func TestHiveRuntimePolicyUnrecognizedWarns(t *testing.T) {
	captureStderr := func(t *testing.T, fn func()) string {
		t.Helper()
		orig := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		os.Stderr = w
		defer func() { os.Stderr = orig }()

		fn()

		w.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatalf("copy stderr: %v", err)
		}
		return buf.String()
	}

	t.Run("unrecognized value warns exactly once per process", func(t *testing.T) {
		t.Setenv(hivePolicyEnv, "raed")
		resetHivePolicyWarnOnceForTest()

		output := captureStderr(t, func() {
			currentHiveRuntimePolicy()
			currentHiveRuntimePolicy()
			currentHiveRuntimePolicy()
		})

		count := strings.Count(output, "AETHER_HIVE_POLICY")
		if count != 1 {
			t.Errorf("stderr warning count = %d, want exactly 1 across 3 lookups:\n%s", count, output)
		}
		if !strings.Contains(output, "raed") {
			t.Errorf("stderr warning does not name the offending value:\n%s", output)
		}
	})

	t.Run("recognized value writes nothing", func(t *testing.T) {
		t.Setenv(hivePolicyEnv, "off")
		resetHivePolicyWarnOnceForTest()

		output := captureStderr(t, func() {
			currentHiveRuntimePolicy()
		})

		if output != "" {
			t.Errorf("stderr = %q, want empty output for a recognized value", output)
		}
	})
}
