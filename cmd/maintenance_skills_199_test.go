package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMaintenanceSkills199(t *testing.T) {
	t.Run("LiveScan", func(t *testing.T) {
		workspace, hub := maintenanceSkillFixture(t)
		alphaPath := filepath.Join(hub, "system", "skills", "colony", "alpha", "SKILL.md")
		writeMaintenanceSkill(t, alphaPath, "alpha", "colony", "shipped", "Builder guidance v1")
		writeMaintenanceSkill(t, filepath.Join(workspace, ".aether", "skills", "domain", "beta", "SKILL.md"), "beta", "domain", "custom", "Domain guidance")

		first := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		if got := stringField(t, first, "operation_id"); got != "skills.inspect" {
			t.Fatalf("operation_id = %q, want skills.inspect", got)
		}
		if got := stringField(t, first, "state_effect"); got != "none" {
			t.Fatalf("state_effect = %q, want none", got)
		}
		alpha := maintenanceSkillEntryByName(t, first, "alpha")
		oldDigest := stringField(t, alpha, "digest")
		for _, field := range []string{"identity", "source_identity", "source", "path", "path_class", "digest", "parse_status"} {
			if strings.TrimSpace(stringField(t, alpha, field)) == "" {
				t.Errorf("alpha entry has empty %s", field)
			}
		}
		if len(oldDigest) != 64 {
			t.Fatalf("digest = %q, want SHA-256 hex", oldDigest)
		}
		metadata := mapField(t, alpha, "metadata")
		if stringField(t, metadata, "name") != "alpha" {
			t.Fatalf("parsed metadata = %#v", metadata)
		}

		writeMaintenanceSkill(t, alphaPath, "alpha", "colony", "shipped", "Builder guidance v2")
		writeMaintenanceSkill(t, filepath.Join(hub, "skills", "domain", "just-added", "SKILL.md"), "just-added", "domain", "custom", "Immediate guidance")
		second := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		if newDigest := stringField(t, maintenanceSkillEntryByName(t, second, "alpha"), "digest"); newDigest == oldDigest {
			t.Fatal("edited skill did not appear immediately in the next live scan")
		}
		_ = maintenanceSkillEntryByName(t, second, "just-added")
		for _, cachePath := range []string{filepath.Join(hub, "skills", "index.json"), filepath.Join(workspace, ".aether", "skills", "index.json")} {
			if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
				t.Fatalf("live scan recreated obsolete cache %s", cachePath)
			}
		}
	})

	t.Run("Diff", func(t *testing.T) {
		workspace, hub := maintenanceSkillFixture(t)
		alphaPath := filepath.Join(hub, "system", "skills", "colony", "alpha", "SKILL.md")
		removedPath := filepath.Join(workspace, ".aether", "skills", "domain", "removed", "SKILL.md")
		writeMaintenanceSkill(t, alphaPath, "alpha", "colony", "shipped", "Alpha v1")
		writeMaintenanceSkill(t, removedPath, "removed", "domain", "custom", "Remove me")
		before := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		alphaBefore := maintenanceSkillEntryByName(t, before, "alpha")
		removedBefore := maintenanceSkillEntryByName(t, before, "removed")

		writeMaintenanceSkill(t, alphaPath, "alpha", "colony", "shipped", "Alpha v2")
		if err := os.Remove(removedPath); err != nil {
			t.Fatal(err)
		}
		writeMaintenanceSkill(t, filepath.Join(hub, "skills", "domain", "added", "SKILL.md"), "added", "domain", "custom", "Add me")
		after := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		alphaAfter := maintenanceSkillEntryByName(t, after, "alpha")
		addedAfter := maintenanceSkillEntryByName(t, after, "added")

		receipts := t.TempDir()
		beforePath := writeMaintenanceSkillReceipt(t, receipts, "before.json", before)
		afterPath := writeMaintenanceSkillReceipt(t, receipts, "after.json", after)
		args := []string{"maintenance", "skills", "diff", "--before", beforePath, "--after", afterPath}
		first := executeMaintenanceSkillsJSON(t, args...)
		second := executeMaintenanceSkillsJSON(t, args...)
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("diff is not deterministic\nfirst:  %#v\nsecond: %#v", first, second)
		}
		if got := stringField(t, first, "state_effect"); got != "none" {
			t.Fatalf("state_effect = %q, want none", got)
		}
		assertMaintenanceSkillDeltaHasIdentity(t, first, "added", stringField(t, addedAfter, "identity"))
		assertMaintenanceSkillDeltaHasIdentity(t, first, "removed", stringField(t, removedBefore, "identity"))
		changed := maintenanceSkillDeltaByIdentity(t, first, "changed", stringField(t, alphaBefore, "identity"))
		if got, want := stringField(t, changed, "before_digest"), stringField(t, alphaBefore, "digest"); got != want {
			t.Fatalf("before_digest = %q, want %q", got, want)
		}
		if got, want := stringField(t, changed, "after_digest"), stringField(t, alphaAfter, "digest"); got != want {
			t.Fatalf("after_digest = %q, want %q", got, want)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		workspace, hub := maintenanceSkillFixture(t)
		invalidPath := filepath.Join(hub, "skills", "domain", "broken", "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(invalidPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(invalidPath, []byte("---\ntype: domain\nsource: custom\n---\nMissing a name\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		result := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		entries := maintenanceSkillEntries(t, result)
		for _, entry := range entries {
			if stringField(t, entry, "parse_status") != "invalid" {
				continue
			}
			if stringField(t, entry, "path") != invalidPath {
				continue
			}
			errors, ok := entry["errors"].([]any)
			if !ok || len(errors) == 0 {
				t.Fatalf("invalid skill omitted parse errors: %#v", entry)
			}
			invalid, ok := result["invalid"].([]any)
			if !ok || len(invalid) != 1 {
				t.Fatalf("receipt invalid list = %#v, want one entry", result["invalid"])
			}
			return
		}
		fatalEntries := entries
		t.Fatalf("invalid skill was omitted from live inventory: %#v", fatalEntries)
	})

	t.Run("ReadOnly", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		workspace, hub := maintenanceSkillFixture(t)
		writeMaintenanceSkill(t, filepath.Join(hub, "system", "skills", "colony", "alpha", "SKILL.md"), "alpha", "colony", "shipped", "Alpha")
		receipt := executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		receiptDir := t.TempDir()
		beforePath := writeMaintenanceSkillReceipt(t, receiptDir, "before.json", receipt)
		afterPath := writeMaintenanceSkillReceipt(t, receiptDir, "after.json", receipt)

		before := fingerprintMaintenanceTrees(t, workspace, hub, receiptDir)
		_ = executeMaintenanceSkillsJSON(t, "maintenance", "skills", "inspect", "--root", workspace, "--hub", hub)
		_ = executeMaintenanceSkillsJSON(t, "maintenance", "skills", "diff", "--before", beforePath, "--after", afterPath)
		after := fingerprintMaintenanceTrees(t, workspace, hub, receiptDir)
		if !reflect.DeepEqual(before, after) {
			t.Fatalf("skill inspection changed bytes or mtimes\nbefore: %#v\nafter:  %#v", before, after)
		}
	})

	t.Run("NoLegacyCLI", func(t *testing.T) {
		for _, banned := range []string{"skill-cache-rebuild", "skill-list", "skill-diff"} {
			for _, command := range rootCmd.Commands() {
				if command.Name() == banned {
					t.Fatalf("retired root command %q was resurrected", banned)
				}
			}
		}
		for _, path := range [][]string{{"maintenance", "skills", "inspect"}, {"maintenance", "skills", "diff"}} {
			command, remaining, err := rootCmd.Find(path)
			if err != nil || len(remaining) != 0 || command.Name() != path[len(path)-1] {
				t.Fatalf("nested maintenance command %v is unavailable: command=%v remaining=%v err=%v", path, command, remaining, err)
			}
		}
	})
}

func maintenanceSkillFixture(t *testing.T) (string, string) {
	t.Helper()
	workspace := t.TempDir()
	hub := filepath.Join(t.TempDir(), "hub")
	for _, dir := range []string{filepath.Join(workspace, ".aether", "data"), filepath.Join(workspace, ".aether", "skills"), hub} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("AETHER_ROOT", workspace)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(workspace, ".aether", "data"))
	t.Setenv("AETHER_HUB_DIR", hub)
	return workspace, hub
}

func writeMaintenanceSkill(t *testing.T, path, name, kind, source, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: Test " + name + " skill\ntype: " + kind + "\nsource: " + source + "\nagent_roles: [builder]\nworkflow_triggers: [build]\ntask_keywords: [test]\n---\n" + body + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func executeMaintenanceSkillsJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	resetFlags(rootCmd)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	var out bytes.Buffer
	stdout = &out
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("aether %s: %v\n%s", strings.Join(args, " "), err, out.String())
	}
	var envelope struct {
		OK     bool           `json:"ok"`
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
		t.Fatalf("decode aether %s output %q: %v", strings.Join(args, " "), out.String(), err)
	}
	if !envelope.OK || envelope.Result == nil {
		t.Fatalf("aether %s returned unsuccessful envelope: %s", strings.Join(args, " "), out.String())
	}
	return envelope.Result
}

func maintenanceSkillEntries(t *testing.T, receipt map[string]any) []map[string]any {
	t.Helper()
	raw, ok := receipt["entries"].([]any)
	if !ok {
		t.Fatalf("entries = %T, want array", receipt["entries"])
	}
	entries := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		entry, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("entry = %T, want object", value)
		}
		entries = append(entries, entry)
	}
	return entries
}

func maintenanceSkillEntryByName(t *testing.T, receipt map[string]any, name string) map[string]any {
	t.Helper()
	for _, entry := range maintenanceSkillEntries(t, receipt) {
		metadata := mapField(t, entry, "metadata")
		if value, _ := metadata["name"].(string); value == name {
			return entry
		}
	}
	t.Fatalf("receipt omitted skill %q: %#v", name, receipt["entries"])
	return nil
}

func writeMaintenanceSkillReceipt(t *testing.T, dir, name string, receipt map[string]any) string {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func maintenanceSkillDeltaByIdentity(t *testing.T, result map[string]any, field, identity string) map[string]any {
	t.Helper()
	raw, ok := result[field].([]any)
	if !ok {
		t.Fatalf("%s = %T, want array", field, result[field])
	}
	for _, value := range raw {
		delta, ok := value.(map[string]any)
		if ok && delta["identity"] == identity {
			return delta
		}
	}
	t.Fatalf("%s omitted identity %q: %#v", field, identity, result[field])
	return nil
}

func assertMaintenanceSkillDeltaHasIdentity(t *testing.T, result map[string]any, field, identity string) {
	t.Helper()
	_ = maintenanceSkillDeltaByIdentity(t, result, field, identity)
}
