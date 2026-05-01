package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- check-antipattern tests ---

func TestCheckAntipatternCleanFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a clean Go file
	cleanFile := filepath.Join(tmpDir, "clean.go")
	os.WriteFile(cleanFile, []byte("package main\nfunc main() {}\n"), 0644)

	rootCmd.SetArgs([]string{"check-antipattern", "--file", cleanFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["clean"] != true {
		t.Errorf("clean = %v, want true", result["clean"])
	}
}

func TestCheckAntipatternMissingFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"check-antipattern", "--file", "/nonexistent/file.js"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["clean"] != true {
		t.Errorf("clean = %v, want true for missing file", result["clean"])
	}
}

func TestCheckAntipatternExposedSecret(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a file with an exposed secret
	secretFile := filepath.Join(tmpDir, "config.py")
	os.WriteFile(secretFile, []byte("api_key = \"sk-12345abcdef67890\"\n"), 0644)

	rootCmd.SetArgs([]string{"check-antipattern", "--file", secretFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["clean"] == true {
		t.Error("expected clean=false for file with exposed secret, got clean=true")
	}
	criticals := result["critical"].([]interface{})
	if len(criticals) == 0 {
		t.Error("expected at least one critical finding for exposed secret")
	}
}

func TestCheckAntipatternConsoleLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a JS file with console.log (non-test file)
	jsFile := filepath.Join(tmpDir, "app.js")
	os.WriteFile(jsFile, []byte("console.log(\"hello\");\nconst x = 1;\n"), 0644)

	rootCmd.SetArgs([]string{"check-antipattern", "--file", jsFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["clean"] == true {
		t.Error("expected clean=false for file with console.log")
	}
	warnings := result["warnings"].([]interface{})
	if len(warnings) == 0 {
		t.Error("expected at least one warning for console.log")
	}
}

func TestCheckAntipatternBareExcept(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a Python file with bare except
	pyFile := filepath.Join(tmpDir, "handler.py")
	os.WriteFile(pyFile, []byte("try:\n    do_thing()\nexcept:\n    pass\n"), 0644)

	rootCmd.SetArgs([]string{"check-antipattern", "--file", pyFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["clean"] == true {
		t.Error("expected clean=false for file with bare except")
	}
}

func TestCheckAntipatternTodoComments(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a Go file with TODO
	goFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(goFile, []byte("package main\n// TODO: fix this later\nfunc main() {}\n"), 0644)

	rootCmd.SetArgs([]string{"check-antipattern", "--file", goFile})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	warnings := result["warnings"].([]interface{})
	found := false
	for _, w := range warnings {
		warning := w.(map[string]interface{})
		if warning["pattern"] == "todo-comment" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected todo-comment warning")
	}
}

// --- midden-write tests ---

func TestMiddenWrite(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"midden-write", "--category", "build", "--message", "build failed", "--source", "test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["success"] != true {
		t.Errorf("success = %v, want true", result["success"])
	}
	entryID, ok := result["entry_id"].(string)
	if !ok || entryID == "" {
		t.Errorf("entry_id = %v, want non-empty string", result["entry_id"])
	}
	if result["category"] != "build" {
		t.Errorf("category = %v, want build", result["category"])
	}
	if result["midden_total"] != float64(1) {
		t.Errorf("midden_total = %v, want 1", result["midden_total"])
	}

	// Verify the entry was persisted
	var mf colony.MiddenFile
	s.LoadJSON("midden.json", &mf)
	if len(mf.Entries) != 1 {
		t.Fatalf("entries count = %d, want 1", len(mf.Entries))
	}
	if mf.Entries[0].Message != "build failed" {
		t.Errorf("message = %q, want 'build failed'", mf.Entries[0].Message)
	}
	if mf.Entries[0].Category != "build" {
		t.Errorf("category = %q, want 'build'", mf.Entries[0].Category)
	}
	if mf.Entries[0].Reviewed {
		t.Error("reviewed = true, want false")
	}
}

func TestMiddenWriteNoMessage(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"midden-write"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["success"] != true {
		t.Errorf("success = %v, want true", result["success"])
	}
	if result["warning"] != "no_message_provided" {
		t.Errorf("warning = %v, want no_message_provided", result["warning"])
	}
}

func TestMiddenWriteDefaults(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"midden-write", "--message", "something broke"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["category"] != "general" {
		t.Errorf("default category = %v, want general", result["category"])
	}

	// Verify the entry was persisted with defaults
	var mf colony.MiddenFile
	s.LoadJSON("midden.json", &mf)
	if mf.Entries[0].Source != "unknown" {
		t.Errorf("default source = %q, want unknown", mf.Entries[0].Source)
	}
}

// --- signature-match tests ---

func TestSignatureMatchFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a test file with a known pattern
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hello\") }\n"), 0644)

	rootCmd.SetArgs([]string{"signature-match", "--file", testFile, "--pattern", "fmt\\.Println"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["matched"] != true {
		t.Errorf("matched = %v, want true", result["matched"])
	}
	matches := result["matches"].([]interface{})
	if len(matches) == 0 {
		t.Error("expected at least one match")
	}
}

func TestSignatureMatchNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\nfunc main() {}\n"), 0644)

	rootCmd.SetArgs([]string{"signature-match", "--file", testFile, "--pattern", "nonexistent_pattern_xyz"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["matched"] != false {
		t.Errorf("matched = %v, want false", result["matched"])
	}
}

func TestSignatureMatchMissingFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"signature-match", "--file", "/nonexistent/file.go", "--pattern", "test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["matched"] != false {
		t.Errorf("matched = %v, want false for missing file", result["matched"])
	}
}

func TestSignatureMatchInvalidRegex(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n"), 0644)

	rootCmd.SetArgs([]string{"signature-match", "--file", testFile, "--pattern", "[invalid"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid regex, got: %v", env["ok"])
	}
}

// --- signature-scan tests ---

func TestSignatureScanFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create signatures.json
	s.SaveJSON("signatures.json", map[string]interface{}{
		"signatures": []interface{}{
			map[string]interface{}{
				"name":                 "test-sig",
				"pattern_string":       "fmt.Println",
				"confidence_threshold": 0.8,
			},
		},
	})

	// Create target file with the pattern
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hi\") }\n"), 0644)

	rootCmd.SetArgs([]string{"signature-scan", "--file", testFile, "--name", "test-sig"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["found"] != true {
		t.Errorf("found = %v, want true", result["found"])
	}
}

func TestSignatureScanNoSignaturesFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte("package main\n"), 0644)

	rootCmd.SetArgs([]string{"signature-scan", "--file", testFile, "--name", "test-sig"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["found"] != false {
		t.Errorf("found = %v, want false when no signatures file", result["found"])
	}
}

// --- privacy-scan tests ---

func TestPrivacyScan_BlocksAPIKey(t *testing.T) {
	result := privacyScan("my secret key is sk-abc123def456ghi789jkl012mno345")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing sk- API key")
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f, "api_key") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding containing 'api_key', got: %v", result.Findings)
	}
}

func TestPrivacyScan_BlocksKeyPrefix(t *testing.T) {
	result := privacyScan("deploy key: key-abc123def456")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing key- prefix")
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f, "api_key_prefix") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding containing 'api_key_prefix', got: %v", result.Findings)
	}
}

func TestPrivacyScan_BlocksBearerToken(t *testing.T) {
	result := privacyScan("Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing Bearer token")
	}
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f, "bearer_token") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected finding containing 'bearer_token', got: %v", result.Findings)
	}
}

func TestPrivacyScan_BlocksPrivateKey(t *testing.T) {
	result := privacyScan("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQ\n-----END RSA PRIVATE KEY-----")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing RSA private key")
	}
}

func TestPrivacyScan_BlocksECPrivateKey(t *testing.T) {
	result := privacyScan("-----BEGIN EC PRIVATE KEY-----\nMHQCAQEE\n-----END EC PRIVATE KEY-----")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing EC private key")
	}
}

func TestPrivacyScan_BlocksOpenSSHKey(t *testing.T) {
	result := privacyScan("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC\n-----END OPENSSH PRIVATE KEY-----")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing OpenSSH private key")
	}
}

func TestPrivacyScan_BlocksPassword(t *testing.T) {
	result := privacyScan("database password='supersecretvalue'")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing long password")
	}
}

func TestPrivacyScan_BlocksPasswd(t *testing.T) {
	result := privacyScan("user passwd='mylongpassword'")
	if !result.Blocked {
		t.Error("expected blocked=true for content containing long passwd")
	}
}

func TestPrivacyScan_RedactsUserPath(t *testing.T) {
	content := "The project is at /Users/john/projects/myapp/src/main.go"
	result := privacyScan(content)
	if result.Blocked {
		t.Error("expected blocked=false for content with only home paths")
	}
	if !strings.Contains(result.Clean, "[REDACTED_PATH]") {
		t.Errorf("expected Clean to contain [REDACTED_PATH], got: %s", result.Clean)
	}
	if strings.Contains(result.Clean, "/Users/john") {
		t.Errorf("expected Clean to NOT contain /Users/john, got: %s", result.Clean)
	}
}

func TestPrivacyScan_RedactsHomePath(t *testing.T) {
	content := "deploying from /home/user/code/repo"
	result := privacyScan(content)
	if result.Blocked {
		t.Error("expected blocked=false for content with only /home paths")
	}
	if !strings.Contains(result.Clean, "[REDACTED_PATH]") {
		t.Errorf("expected Clean to contain [REDACTED_PATH], got: %s", result.Clean)
	}
}

func TestPrivacyScan_RedactsTildePath(t *testing.T) {
	content := "check ~/Documents/file.txt for details"
	result := privacyScan(content)
	if result.Blocked {
		t.Error("expected blocked=false for content with only tilde paths")
	}
	if !strings.Contains(result.Clean, "[REDACTED_PATH]") {
		t.Errorf("expected Clean to contain [REDACTED_PATH], got: %s", result.Clean)
	}
}

func TestPrivacyScan_CleanContent(t *testing.T) {
	content := "hello world, nothing to see here"
	result := privacyScan(content)
	if result.Blocked {
		t.Error("expected blocked=false for clean content")
	}
	if result.Clean != content {
		t.Errorf("expected Clean to equal original content, got: %s", result.Clean)
	}
}

func TestPrivacyScan_SecretsTakePrecedence(t *testing.T) {
	content := "api_key sk-abc123def456ghi789jkl and the project is at /Users/john/projects/myapp"
	result := privacyScan(content)
	if !result.Blocked {
		t.Error("expected blocked=true when content has both secrets and paths (secrets take precedence)")
	}
	// When blocked, Clean should be empty (write rejected)
	if result.Clean != "" {
		t.Errorf("expected Clean to be empty when blocked, got: %s", result.Clean)
	}
}

func TestPrivacyScan_EnvFilePatterns(t *testing.T) {
	// Test .env
	result := privacyScan("load config from .env")
	if !result.Blocked {
		t.Error("expected blocked=true for content referencing .env")
	}
	// Test .env.local
	result = privacyScan("use .env.local for local overrides")
	if !result.Blocked {
		t.Error("expected blocked=true for content referencing .env.local")
	}
	// Test credentials.json
	result = privacyScan("fetch credentials from credentials.json")
	if !result.Blocked {
		t.Error("expected blocked=true for content referencing credentials.json")
	}
}

func TestPrivacyScan_AllowsShortPasswords(t *testing.T) {
	result := privacyScan("password='abc'")
	if result.Blocked {
		t.Error("expected blocked=false for short passwords (< 8 chars)")
	}
}

func TestPrivacyScan_CLICommand(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"privacy-scan", "--content", "hello world"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["blocked"] != false {
		t.Errorf("blocked = %v, want false for clean content", result["blocked"])
	}
}

func TestPrivacyScan_CLICommandBlocksSecret(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"privacy-scan", "--content", "secret key sk-abc123def456ghi789jkl012mno345"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["blocked"] != true {
		t.Errorf("blocked = %v, want true for secret content", result["blocked"])
	}
}
