package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runAdvancedCommandsCLI runs `aether advanced-commands <args...>` in-process
// and returns the parsed {"ok":true,"result":...} envelope's result field
// (or fails the test if the command did not report ok:true).
func runAdvancedCommandsCLI(t *testing.T, args ...string) map[string]interface{} {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	store = nil
	var out bytes.Buffer
	stdout = &out
	stderr = &out
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs(append([]string{"advanced-commands"}, args...))
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("advanced-commands %v: %v (output: %s)", args, err, out.String())
	}
	var envelope struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("advanced-commands %v: parse output %q: %v", args, out.String(), err)
	}
	if !envelope.OK {
		t.Fatalf("advanced-commands %v: reported ok:false, output: %s", args, out.String())
	}
	return envelope.Result
}

// TestAdvancedSettingIsMachineWideAndWorksWithNoProject proves A-01: the
// setting lives with the owner's preferences on his machine, is readable
// and writable from a folder with no Aether project set up in it, and never
// creates a project-scoped .aether directory as a side effect of reading or
// writing it.
func TestAdvancedSettingIsMachineWideAndWorksWithNoProject(t *testing.T) {
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	workDir := t.TempDir()
	t.Chdir(workDir)

	// No .aether directory anywhere before we start.
	if _, err := os.Stat(filepath.Join(workDir, ".aether")); !os.IsNotExist(err) {
		t.Fatalf("precondition: workDir already has .aether: %v", err)
	}

	// get, with nothing stored yet: false, source "default".
	result := runAdvancedCommandsCLI(t, "get")
	if result["value"] != false {
		t.Errorf("get (unset) value = %v, want false", result["value"])
	}
	if result["source"] != "default" {
		t.Errorf("get (unset) source = %v, want %q", result["source"], "default")
	}

	// set on, then get: true, source "stored".
	runAdvancedCommandsCLI(t, "set", "on")
	result = runAdvancedCommandsCLI(t, "get")
	if result["value"] != true {
		t.Errorf("get (after set on) value = %v, want true", result["value"])
	}
	if result["source"] != "stored" {
		t.Errorf("get (after set on) source = %v, want %q", result["source"], "stored")
	}

	// set off: returns to false.
	runAdvancedCommandsCLI(t, "set", "off")
	result = runAdvancedCommandsCLI(t, "get")
	if result["value"] != false {
		t.Errorf("get (after set off) value = %v, want false", result["value"])
	}

	// The working directory never grew a project .aether directory across
	// any of the four calls above -- this is a machine-wide setting, not a
	// project one.
	if _, err := os.Stat(filepath.Join(workDir, ".aether")); !os.IsNotExist(err) {
		t.Errorf("workDir grew a .aether directory: %v", err)
	}

	// The setting was genuinely written to the hub, not just held in
	// process memory.
	prefsPath := filepath.Join(hubDir, hubPreferencesFileName)
	if _, err := os.Stat(prefsPath); err != nil {
		t.Errorf("expected %s to exist after set: %v", prefsPath, err)
	}
}

// TestAdvancedSettingReaderNeverFails proves readAdvancedCommandsSetting
// returns (false, "default") and no panic/error for every degraded input
// named in the plan's acceptance criteria.
func TestAdvancedSettingReaderNeverFails(t *testing.T) {
	t.Run("hub path empty", func(t *testing.T) {
		t.Setenv("AETHER_HUB_DIR", "")
		t.Setenv("HOME", "")
		on, source := readAdvancedCommandsSetting()
		if on || source != "default" {
			t.Errorf("got (%v, %q), want (false, \"default\")", on, source)
		}
	})

	t.Run("file absent", func(t *testing.T) {
		hubDir := t.TempDir()
		t.Setenv("AETHER_HUB_DIR", hubDir)
		on, source := readAdvancedCommandsSetting()
		if on || source != "default" {
			t.Errorf("got (%v, %q), want (false, \"default\")", on, source)
		}
	})

	t.Run("file contains invalid JSON", func(t *testing.T) {
		hubDir := t.TempDir()
		t.Setenv("AETHER_HUB_DIR", hubDir)
		if err := os.WriteFile(filepath.Join(hubDir, hubPreferencesFileName), []byte("{not json"), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		on, source := readAdvancedCommandsSetting()
		if on || source != "default" {
			t.Errorf("got (%v, %q), want (false, \"default\")", on, source)
		}
	})

	t.Run("file carries an unknown schema version", func(t *testing.T) {
		hubDir := t.TempDir()
		t.Setenv("AETHER_HUB_DIR", hubDir)
		body := `{"schema_version":"something-else/v9","show_all_commands":true}`
		if err := os.WriteFile(filepath.Join(hubDir, hubPreferencesFileName), []byte(body), 0644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		on, source := readAdvancedCommandsSetting()
		if on || source != "default" {
			t.Errorf("got (%v, %q), want (false, \"default\")", on, source)
		}
	})
}

// TestAdvancedSettingPreservesUnrelatedPreferences proves
// writeAdvancedCommandsSetting merges into preferences.json rather than
// overwriting the whole object -- an unrelated key already present must
// survive a set call.
func TestAdvancedSettingPreservesUnrelatedPreferences(t *testing.T) {
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	prefsPath := filepath.Join(hubDir, hubPreferencesFileName)
	unrelated := `{"schema_version":"aether-preferences/v1","favorite_caste":"builder"}`
	if err := os.WriteFile(prefsPath, []byte(unrelated), 0644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := writeAdvancedCommandsSetting(true); err != nil {
		t.Fatalf("writeAdvancedCommandsSetting: %v", err)
	}

	raw, err := os.ReadFile(prefsPath)
	if err != nil {
		t.Fatalf("read preferences: %v", err)
	}
	var parsed map[string]interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parse preferences: %v", err)
	}
	if parsed["favorite_caste"] != "builder" {
		t.Errorf("unrelated key favorite_caste = %v, want %q (it must survive a set call)", parsed["favorite_caste"], "builder")
	}
	if parsed["show_all_commands"] != true {
		t.Errorf("show_all_commands = %v, want true", parsed["show_all_commands"])
	}
}

// TestAdvancedCommandsSetRejectsUnrecognisedWord proves an unrecognised word
// to `set` reports the two words it accepts and changes nothing on disk.
func TestAdvancedCommandsSetRejectsUnrecognisedWord(t *testing.T) {
	hubDir := t.TempDir()
	t.Setenv("AETHER_HUB_DIR", hubDir)

	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	store = nil
	var out bytes.Buffer
	stdout = &out
	stderr = &out
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&out)
	rootCmd.SetArgs([]string{"advanced-commands", "set", "maybe"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("advanced-commands set maybe: %v", err)
	}
	combined := out.String()
	if !strings.Contains(combined, "on") || !strings.Contains(combined, "off") {
		t.Errorf("error output does not name both accepted words: %s", combined)
	}
	if _, err := os.Stat(filepath.Join(hubDir, hubPreferencesFileName)); !os.IsNotExist(err) {
		t.Errorf("an unrecognised word must change nothing on disk, but preferences.json now exists (err=%v)", err)
	}
}
