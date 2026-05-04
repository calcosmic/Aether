package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestE2EInstallSetupUpdateFlow verifies the full lifecycle:
//   - install: populates hub (~/.aether/system/) from package dir (.aether/, .claude/, .opencode/)
//   - setup: syncs hub system files to a target repo's .aether/
//   - update: re-syncs changed hub files to the target repo
//
// This is an end-to-end integration test that exercises all three commands
// in sequence within temp directories, verifying data flows correctly and
// protected directories (data/, dreams/) are never overwritten.
func TestE2EInstallSetupUpdateFlow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	// --- Setup: create temp directories ---
	packageDir := t.TempDir() // simulates the Aether npm package
	homeDir := t.TempDir()    // simulates $HOME
	repoDir := t.TempDir()    // simulates a target repository

	// Create package structure: .aether/ with companion files
	pkgAether := filepath.Join(packageDir, ".aether")
	pkgAetherDocs := filepath.Join(pkgAether, "docs")
	if err := os.MkdirAll(pkgAetherDocs, 0755); err != nil {
		t.Fatalf("failed to create package .aether dirs: %v", err)
	}
	pkgAetherSkills := filepath.Join(pkgAether, "skills", "colony")
	if err := os.MkdirAll(pkgAetherSkills, 0755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(pkgAether, "templates"), 0755); err != nil {
		t.Fatalf("failed to create templates dir: %v", err)
	}

	// Write companion files
	workersV1 := []byte("# Workers v1 - initial version")
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), workersV1, 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAetherDocs, "guide.md"), []byte("# Guide v1"), 0644); err != nil {
		t.Fatalf("failed to write guide.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAetherSkills, "tdd.md"), []byte("# TDD Skill"), 0644); err != nil {
		t.Fatalf("failed to write tdd.md: %v", err)
	}

	// Create .claude/commands/ant/ in package
	pkgCmds := filepath.Join(packageDir, ".claude", "commands", "ant")
	if err := os.MkdirAll(pkgCmds, 0755); err != nil {
		t.Fatalf("failed to create claude commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgCmds, "init.md"), []byte("# Init command"), 0644); err != nil {
		t.Fatalf("failed to write init.md: %v", err)
	}

	// Create .claude/agents/ant/ in package
	pkgAgents := filepath.Join(packageDir, ".claude", "agents", "ant")
	if err := os.MkdirAll(pkgAgents, 0755); err != nil {
		t.Fatalf("failed to create claude agents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAgents, "builder.md"), []byte("# Builder agent"), 0644); err != nil {
		t.Fatalf("failed to write builder.md: %v", err)
	}

	// Create .opencode/commands/ant/ in package
	pkgOCcmds := filepath.Join(packageDir, ".opencode", "commands", "ant")
	if err := os.MkdirAll(pkgOCcmds, 0755); err != nil {
		t.Fatalf("failed to create opencode commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgOCcmds, "init.md"), []byte("# OC Init command"), 0644); err != nil {
		t.Fatalf("failed to write OC init.md: %v", err)
	}

	// Create .opencode/agents/ in package
	pkgOCAgents := filepath.Join(packageDir, ".opencode", "agents")
	if err := os.MkdirAll(pkgOCAgents, 0755); err != nil {
		t.Fatalf("failed to create opencode agents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgOCAgents, "builder.md"), []byte("---\nname: aether-builder\ndescription: \"OpenCode builder agent for the Aether colony framework\"\nmode: subagent\ncolor: \"#ff0000\"\ntools:\n  write: true\n  edit: true\n  bash: true\n  grep: true\n  glob: true\n  task: true\n---\n\n# OC Builder agent\n"), 0644); err != nil {
		t.Fatalf("failed to write OC builder.md: %v", err)
	}

	// ===== STEP 1: Install =====
	t.Run("install", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("install command failed: %v", err)
		}

		// Verify output is valid JSON with ok:true
		output := buf.String()
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("install output is not valid JSON: %v\noutput: %s", err, output)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("install returned ok:false, output: %s", output)
		}

		// Verify hub was created
		hubDir := filepath.Join(homeDir, ".aether")
		if _, err := os.Stat(hubDir); os.IsNotExist(err) {
			t.Fatal("hub directory not created")
		}

		// Verify version.json was created
		versionFile := filepath.Join(hubDir, "version.json")
		if _, err := os.Stat(versionFile); os.IsNotExist(err) {
			t.Fatal("hub version.json not created")
		}

		// Verify system files were synced
		hubSystem := filepath.Join(hubDir, "system")
		hubWorkers := filepath.Join(hubSystem, "workers.md")
		if _, err := os.Stat(hubWorkers); os.IsNotExist(err) {
			t.Fatal("hub system workers.md not created")
		}

		// Verify hub workers content matches v1
		content, err := os.ReadFile(hubWorkers)
		if err != nil {
			t.Fatalf("failed to read hub workers.md: %v", err)
		}
		if string(content) != string(workersV1) {
			t.Errorf("hub workers.md content mismatch\ngot:  %s\nwant: %s", string(content), string(workersV1))
		}

		// Verify wrapper command mirrors were published into the hub locations
		// that `aether update` reads from.
		hubClaudeCmd := filepath.Join(hubSystem, "commands", "claude", "init.md")
		if _, err := os.Stat(hubClaudeCmd); os.IsNotExist(err) {
			t.Error("hub claude command mirror not created")
		}

		hubOpenCodeCmd := filepath.Join(hubSystem, "commands", "opencode", "init.md")
		if _, err := os.Stat(hubOpenCodeCmd); os.IsNotExist(err) {
			t.Error("hub opencode command mirror not created")
		}

		hubOpenCodeAgent := filepath.Join(hubSystem, "agents", "builder.md")
		if _, err := os.Stat(hubOpenCodeAgent); os.IsNotExist(err) {
			t.Error("hub opencode agent mirror not created")
		}

		// Verify claude commands were installed in the flat /ant-* layout.
		claudeCmdDest := filepath.Join(homeDir, ".claude", "commands", "ant-init.md")
		if _, err := os.Stat(claudeCmdDest); os.IsNotExist(err) {
			t.Error("claude commands not installed to home dir")
		}

		// Verify claude agents were installed
		claudeAgentDest := filepath.Join(homeDir, ".claude", "agents", "ant", "builder.md")
		if _, err := os.Stat(claudeAgentDest); os.IsNotExist(err) {
			t.Error("claude agents not installed to home dir")
		}

		// Verify opencode commands were installed
		ocCmdDest := filepath.Join(homeDir, ".config", "opencode", "commands", "ant", "init.md")
		if _, err := os.Stat(ocCmdDest); os.IsNotExist(err) {
			t.Error("opencode commands not installed to home dir")
		}

		// Verify opencode agents were installed
		ocAgentDest := filepath.Join(homeDir, ".config", "opencode", "agents", "builder.md")
		if _, err := os.Stat(ocAgentDest); os.IsNotExist(err) {
			t.Error("opencode agents not installed to home dir")
		}
	})

	// ===== STEP 2: Setup =====
	t.Run("setup", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("setup command failed: %v", err)
		}

		// Verify output is valid JSON
		output := buf.String()
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("setup output is not valid JSON: %v\noutput: %s", err, output)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("setup returned ok:false, output: %s", output)
		}

		// Verify companion file stays global and is not copied to the repo
		repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
		if _, err := os.Stat(repoWorkers); err == nil {
			t.Fatal("repo workers.md should not be created by setup")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo workers.md: %v", err)
		}

		// Verify required directories were created
		for _, dir := range []string{"data", "checkpoints", "locks"} {
			p := filepath.Join(repoDir, ".aether", dir)
			if info, err := os.Stat(p); os.IsNotExist(err) {
				t.Errorf("required dir %s not created", dir)
			} else if err == nil && !info.IsDir() {
				t.Errorf("expected %s to be a directory", dir)
			}
		}

		// Verify .gitignore was created
		gitignore := filepath.Join(repoDir, ".aether", ".gitignore")
		if _, err := os.Stat(gitignore); os.IsNotExist(err) {
			t.Error(".gitignore not created by setup")
		}

		// Verify shipped docs were not copied into the repo
		repoGuide := filepath.Join(repoDir, ".aether", "docs", "guide.md")
		if _, err := os.Stat(repoGuide); err == nil {
			t.Error("docs/guide.md should stay global")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo guide: %v", err)
		}

		// Verify shipped skills were not copied into the repo
		repoSkill := filepath.Join(repoDir, ".aether", "skills", "colony", "tdd.md")
		if _, err := os.Stat(repoSkill); err == nil {
			t.Error("skills/colony/tdd.md should stay global")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo skill: %v", err)
		}
	})

	// ===== STEP 3: Create local user data that must be protected =====
	localDataDir := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(localDataDir, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	userState := `{"goal":"user goal","state":"ACTIVE","important":true}`
	if err := os.WriteFile(filepath.Join(localDataDir, "COLONY_STATE.json"), []byte(userState), 0644); err != nil {
		t.Fatalf("failed to create user state: %v", err)
	}

	// Create local dreams that must be protected
	localDreamsDir := filepath.Join(repoDir, ".aether", "dreams")
	if err := os.MkdirAll(localDreamsDir, 0755); err != nil {
		t.Fatalf("failed to create local dreams dir: %v", err)
	}
	userDream := "# My personal dream - do not overwrite"
	if err := os.WriteFile(filepath.Join(localDreamsDir, "dream1.md"), []byte(userDream), 0644); err != nil {
		t.Fatalf("failed to create user dream: %v", err)
	}

	// ===== STEP 4: Simulate hub update (modify source files) =====
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	workersV2 := []byte("# Workers v2 - updated version")
	if err := os.WriteFile(filepath.Join(hubSystem, "workers.md"), workersV2, 0644); err != nil {
		t.Fatalf("failed to update hub workers.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubSystem, "docs", "guide.md"), []byte("# Guide v2 - updated"), 0644); err != nil {
		t.Fatalf("failed to update hub guide.md: %v", err)
	}

	// Add a new file to hub
	if err := os.WriteFile(filepath.Join(hubSystem, "newfile.md"), []byte("# Brand new file"), 0644); err != nil {
		t.Fatalf("failed to create new hub file: %v", err)
	}

	// Update claude commands in hub system (simulates commands being in hub system)
	hubClaudeCmds := filepath.Join(hubSystem, "commands", "claude")
	if err := os.MkdirAll(hubClaudeCmds, 0755); err != nil {
		t.Fatalf("failed to create hub claude commands dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubClaudeCmds, "updated.md"), []byte("# Updated command"), 0644); err != nil {
		t.Fatalf("failed to write updated command: %v", err)
	}

	// Update rules in hub system
	hubRules := filepath.Join(hubSystem, "rules")
	if err := os.MkdirAll(hubRules, 0755); err != nil {
		t.Fatalf("failed to create hub rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubRules, "test-rule.md"), []byte("# Test Rule"), 0644); err != nil {
		t.Fatalf("failed to write test rule: %v", err)
	}

	// ===== STEP 5: Update =====
	t.Run("update", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		// update uses $HOME and cwd, so set those
		t.Setenv("HOME", homeDir)
		oldDir, _ := os.Getwd()
		if err := os.Chdir(repoDir); err != nil {
			t.Fatalf("failed to chdir to repo: %v", err)
		}
		defer os.Chdir(oldDir)

		rootCmd.SetArgs([]string{"update", "--force"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("update command failed: %v", err)
		}

		// Verify output is valid JSON
		output := buf.String()
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(output), &result); err != nil {
			t.Fatalf("update output is not valid JSON: %v\noutput: %s", err, output)
		}
		if ok, _ := result["ok"].(bool); !ok {
			t.Fatalf("update returned ok:false, output: %s", output)
		}

		// Verify workers.md remains global
		repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
		if _, err := os.Stat(repoWorkers); err == nil {
			t.Fatal("repo workers.md should not be copied by update")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo workers.md after update: %v", err)
		}

		// Verify guide.md remains global
		repoGuide := filepath.Join(repoDir, ".aether", "docs", "guide.md")
		if _, err := os.Stat(repoGuide); err == nil {
			t.Fatal("repo guide.md should not be copied by update")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo guide.md after update: %v", err)
		}

		// Verify arbitrary new system file was not copied
		repoNewFile := filepath.Join(repoDir, ".aether", "newfile.md")
		if _, err := os.Stat(repoNewFile); err == nil {
			t.Error("new file should not be synced to repo")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo newfile: %v", err)
		}

		// Verify updated command was not synced to repo
		repoUpdatedCmd := filepath.Join(repoDir, ".claude", "commands", "ant-updated.md")
		if _, err := os.Stat(repoUpdatedCmd); err == nil {
			t.Error("updated command should stay global")
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repo command: %v", err)
		}

		// Verify rules were synced
		repoRule := filepath.Join(repoDir, ".claude", "rules", "test-rule.md")
		if _, err := os.Stat(repoRule); os.IsNotExist(err) {
			t.Error("rules not synced to repo")
		}

		// ===== PROTECTED DIRS: verify user data is untouched =====

		// Verify COLONY_STATE.json is preserved
		stateContent, err := os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
		if err != nil {
			t.Fatalf("failed to read local COLONY_STATE.json after update: %v", err)
		}
		if string(stateContent) != userState {
			t.Errorf("user COLONY_STATE.json was overwritten during update\ngot:  %s\nwant: %s", string(stateContent), userState)
		}

		// Verify dream file is preserved
		dreamContent, err := os.ReadFile(filepath.Join(localDreamsDir, "dream1.md"))
		if err != nil {
			t.Fatalf("failed to read local dream after update: %v", err)
		}
		if string(dreamContent) != userDream {
			t.Errorf("user dream was overwritten during update\ngot:  %s\nwant: %s", string(dreamContent), userDream)
		}
	})

	// ===== STEP 6: Verify idempotency - running update again skips unchanged =====
	t.Run("update_idempotent", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf

		t.Setenv("HOME", homeDir)
		oldDir, _ := os.Getwd()
		if err := os.Chdir(repoDir); err != nil {
			t.Fatalf("failed to chdir to repo: %v", err)
		}
		defer os.Chdir(oldDir)

		rootCmd.SetArgs([]string{"update"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("second update failed: %v", err)
		}

		// Verify user data is STILL preserved after second update
		stateContent, err := os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
		if err != nil {
			t.Fatalf("failed to read COLONY_STATE.json after second update: %v", err)
		}
		if string(stateContent) != userState {
			t.Errorf("user COLONY_STATE.json was overwritten on second update\ngot:  %s\nwant: %s", string(stateContent), userState)
		}
	})
}

// TestE2EInstallSetupProtectedDirsFromHub verifies that even if the hub's system/
// directory contains data/ or dreams/ files, setup and update never overwrite
// local user data.
func TestE2EInstallSetupProtectedDirsFromHub(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	packageDir := t.TempDir()
	homeDir := t.TempDir()
	repoDir := t.TempDir()

	// Create package with companion files
	pkgAether := filepath.Join(packageDir, ".aether")
	if err := os.MkdirAll(pkgAether, 0755); err != nil {
		t.Fatalf("failed to create package dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgAether, "workers.md"), []byte("# Workers"), 0644); err != nil {
		t.Fatalf("failed to write workers.md: %v", err)
	}

	// Install
	var installBuf bytes.Buffer
	stdout = &installBuf
	rootCmd.SetArgs([]string{"install", "--package-dir", packageDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("install failed: %v", err)
	}

	// Poison the hub system/ with data/ files (simulating corrupted hub)
	hubSystem := filepath.Join(homeDir, ".aether", "system")
	hubDataDir := filepath.Join(hubSystem, "data")
	if err := os.MkdirAll(hubDataDir, 0755); err != nil {
		t.Fatalf("failed to create hub data dir: %v", err)
	}
	hubState := `{"goal":"MALICIOUS","state":"CORRUPTED"}`
	if err := os.WriteFile(filepath.Join(hubDataDir, "COLONY_STATE.json"), []byte(hubState), 0644); err != nil {
		t.Fatalf("failed to create hub state: %v", err)
	}

	// Poison hub system/ with dreams/ files
	hubDreamsDir := filepath.Join(hubSystem, "dreams")
	if err := os.MkdirAll(hubDreamsDir, 0755); err != nil {
		t.Fatalf("failed to create hub dreams dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hubDreamsDir, "evil.md"), []byte("# Evil dream"), 0644); err != nil {
		t.Fatalf("failed to create hub dream: %v", err)
	}

	// Setup with pre-existing user data
	localDataDir := filepath.Join(repoDir, ".aether", "data")
	if err := os.MkdirAll(localDataDir, 0755); err != nil {
		t.Fatalf("failed to create local data dir: %v", err)
	}
	userState := `{"goal":"my goal","state":"ACTIVE"}`
	if err := os.WriteFile(filepath.Join(localDataDir, "COLONY_STATE.json"), []byte(userState), 0644); err != nil {
		t.Fatalf("failed to create user state: %v", err)
	}

	var setupBuf bytes.Buffer
	stdout = &setupBuf
	rootCmd.SetArgs([]string{"setup", "--repo-dir", repoDir, "--home-dir", homeDir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Verify user data was NOT overwritten by setup
	stateContent, err := os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read state after setup: %v", err)
	}
	if string(stateContent) != userState {
		t.Errorf("setup overwrote user COLONY_STATE.json\ngot:  %s\nwant: %s", string(stateContent), userState)
	}

	// dreams/ is local-only scaffold. Hub dreams must never be copied into it.
	if _, err := os.Stat(filepath.Join(repoDir, ".aether", "dreams", "evil.md")); err == nil {
		t.Error("setup copied hub dreams/ content into repo-local state")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat repo dream: %v", err)
	}

	// Now run update with force
	var updateBuf bytes.Buffer
	stdout = &updateBuf
	t.Setenv("HOME", homeDir)
	oldDir, _ := os.Getwd()
	if err := os.Chdir(repoDir); err != nil {
		t.Fatalf("failed to chdir to repo: %v", err)
	}
	defer os.Chdir(oldDir)
	rootCmd.SetArgs([]string{"update", "--force"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("update failed: %v", err)
	}

	// Verify user data is STILL not overwritten even with --force
	stateContent, err = os.ReadFile(filepath.Join(localDataDir, "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("failed to read state after update: %v", err)
	}
	if string(stateContent) != userState {
		t.Errorf("force update overwrote user COLONY_STATE.json\ngot:  %s\nwant: %s", string(stateContent), userState)
	}

	// Verify workers.md is not copied into repos anymore.
	repoWorkers := filepath.Join(repoDir, ".aether", "workers.md")
	if _, err := os.Stat(repoWorkers); err == nil {
		t.Error("workers.md should stay global after setup+update")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat workers.md after setup+update: %v", err)
	}
}
