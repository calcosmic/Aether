package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// preflightEnvKnobPattern matches every AETHER_PREFLIGHT_*/AETHER_SKIP_PREFLIGHT
// identifier that source code can define or reference.
var preflightEnvKnobPattern = regexp.MustCompile(`\bAETHER_PREFLIGHT[A-Z_]*\b|\bAETHER_SKIP_PREFLIGHT\b`)

// scanPreflightEnvKnobs walks the given repo-root-relative directories
// (non-recursive, matching the plan's literal "cmd/*.go, pkg/codex/*.go,
// .aether/ts-host/src/*.ts" scope), collecting every distinct
// AETHER_PREFLIGHT_*/AETHER_SKIP_PREFLIGHT identifier found in files with
// the given extension. Files ending in "_test<ext>" are skipped -- test-only
// fixtures must not be allowed to satisfy the documentation requirement.
func scanPreflightEnvKnobs(t *testing.T, repoRoot string, dirs []string, ext string) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	for _, dir := range dirs {
		absDir := filepath.Join(repoRoot, dir)
		entries, err := os.ReadDir(absDir)
		if err != nil {
			t.Fatalf("read dir %s: %v", absDir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ext) {
				continue
			}
			if strings.HasSuffix(name, "_test"+ext) {
				continue
			}
			content, err := os.ReadFile(filepath.Join(absDir, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			for _, match := range preflightEnvKnobPattern.FindAllString(string(content), -1) {
				found[match] = true
			}
		}
	}
	return found
}

// TestEveryPreflightEnvKnobIsDocumented fails the moment a new
// AETHER_PREFLIGHT_*/AETHER_SKIP_PREFLIGHT knob ships without a documented
// entry in .aether/docs/preflight-configuration.md (D-05, T-163.2-27).
func TestEveryPreflightEnvKnobIsDocumented(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	knobs := map[string]bool{}
	for k := range scanPreflightEnvKnobs(t, repoRoot, []string{"cmd", "pkg/codex"}, ".go") {
		knobs[k] = true
	}
	for k := range scanPreflightEnvKnobs(t, repoRoot, []string{".aether/ts-host/src"}, ".ts") {
		knobs[k] = true
	}

	if len(knobs) == 0 {
		t.Fatal("scan found zero AETHER_PREFLIGHT_*/AETHER_SKIP_PREFLIGHT identifiers -- the scan pattern itself is broken")
	}

	docPath := filepath.Join(repoRoot, ".aether", "docs", "preflight-configuration.md")
	docBytes, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	doc := string(docBytes)

	var missing []string
	for knob := range knobs {
		if !strings.Contains(doc, knob) {
			missing = append(missing, knob)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("%s is missing documentation for: %s", docPath, strings.Join(missing, ", "))
	}
}

// TestHostsAgreeOnPreflightDefaultBudget parses hostedPreflightTimeout from
// pkg/codex/platform_dispatch.go and PREFLIGHT_DEFAULT_TIMEOUT_MS from
// .aether/ts-host/src/preflight-config.ts and asserts the two values are
// equal in milliseconds. Changing one host's default without the other must
// fail this test (T-163.2-28).
func TestHostsAgreeOnPreflightDefaultBudget(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	goPath := filepath.Join(repoRoot, "pkg", "codex", "platform_dispatch.go")
	goContent, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read %s: %v", goPath, err)
	}
	goMatch := regexp.MustCompile(`hostedPreflightTimeout\s*=\s*(\d+)\s*\*\s*time\.Second`).FindSubmatch(goContent)
	if goMatch == nil {
		t.Fatalf("%s: could not find \"hostedPreflightTimeout = N * time.Second\"", goPath)
	}
	goSeconds, err := strconv.Atoi(string(goMatch[1]))
	if err != nil {
		t.Fatalf("parse Go seconds value %q: %v", goMatch[1], err)
	}
	goMs := goSeconds * 1000

	tsPath := filepath.Join(repoRoot, ".aether", "ts-host", "src", "preflight-config.ts")
	tsContent, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read %s: %v", tsPath, err)
	}
	tsMatch := regexp.MustCompile(`PREFLIGHT_DEFAULT_TIMEOUT_MS\s*=\s*([\d_]+)`).FindSubmatch(tsContent)
	if tsMatch == nil {
		t.Fatalf("%s: could not find \"PREFLIGHT_DEFAULT_TIMEOUT_MS = N\"", tsPath)
	}
	tsMs, err := strconv.Atoi(strings.ReplaceAll(string(tsMatch[1]), "_", ""))
	if err != nil {
		t.Fatalf("parse TS milliseconds value %q: %v", tsMatch[1], err)
	}

	if goMs != tsMs {
		t.Fatalf("preflight default budget drift: %s hostedPreflightTimeout = %dms, %s PREFLIGHT_DEFAULT_TIMEOUT_MS = %dms -- these must agree", goPath, goMs, tsPath, tsMs)
	}
}

// TestHostsAgreeOnPreflightRetryAttempts parses hostedPreflightAttempts from
// pkg/codex/platform_dispatch.go and PREFLIGHT_GO_ATTEMPTS from
// .aether/ts-host/src/worker-dispatch.ts and asserts they are equal. The TS
// host sizes its Node-side adapter kill budget as
// resolvePreflightTimeoutMs() x PREFLIGHT_GO_ATTEMPTS + slack; if Go grows a
// third retry attempt without the TS mirror following, Node SIGTERMs the
// adapter mid-retry again (CR-01, the 27-July failure mode).
func TestHostsAgreeOnPreflightRetryAttempts(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	goPath := filepath.Join(repoRoot, "pkg", "codex", "platform_dispatch.go")
	goContent, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("read %s: %v", goPath, err)
	}
	goMatch := regexp.MustCompile(`hostedPreflightAttempts\s*=\s*(\d+)`).FindSubmatch(goContent)
	if goMatch == nil {
		t.Fatalf("%s: could not find \"hostedPreflightAttempts = N\"", goPath)
	}
	goAttempts, err := strconv.Atoi(string(goMatch[1]))
	if err != nil {
		t.Fatalf("parse Go attempts value %q: %v", goMatch[1], err)
	}

	tsPath := filepath.Join(repoRoot, ".aether", "ts-host", "src", "worker-dispatch.ts")
	tsContent, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read %s: %v", tsPath, err)
	}
	tsMatch := regexp.MustCompile(`PREFLIGHT_GO_ATTEMPTS\s*=\s*(\d+)`).FindSubmatch(tsContent)
	if tsMatch == nil {
		t.Fatalf("%s: could not find \"PREFLIGHT_GO_ATTEMPTS = N\"", tsPath)
	}
	tsAttempts, err := strconv.Atoi(string(tsMatch[1]))
	if err != nil {
		t.Fatalf("parse TS attempts value %q: %v", tsMatch[1], err)
	}

	if goAttempts != tsAttempts {
		t.Fatalf("preflight attempts drift: %s hostedPreflightAttempts = %d, %s PREFLIGHT_GO_ATTEMPTS = %d -- these must agree", goPath, goAttempts, tsPath, tsAttempts)
	}
}

// TestNoHardcodedPreflightBudgetRemains proves the folded todo's complaint
// (a hardcoded 20s preflight timeout) stays fixed in both hosts, including
// the compiled dist bundle a downstream repo actually runs (T-163.2-28).
func TestNoHardcodedPreflightBudgetRemains(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	checkNoLiteral := func(relPath string, literals []string) {
		t.Helper()
		absPath := filepath.Join(repoRoot, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			t.Fatalf("read %s: %v", absPath, err)
		}
		text := string(content)
		for _, literal := range literals {
			if strings.Contains(text, literal) {
				t.Fatalf("%s still contains the stale preflight literal %q -- this is the folded todo's complaint, made permanent", absPath, literal)
			}
		}
	}

	checkNoLiteral(filepath.Join(".aether", "ts-host", "src", "platform-dispatcher.ts"), []string{"20_000", "20000"})
	checkNoLiteral(filepath.Join(".aether", "ts-host", "dist", "platform-dispatcher.js"), []string{"20_000", "20000"})
	checkNoLiteral(filepath.Join(".aether", "ts-host", "src", "preflight-config.ts"), []string{"20_000", "20000"})
	checkNoLiteral(filepath.Join(".aether", "ts-host", "dist", "preflight-config.js"), []string{"20_000", "20000"})

	workerGoPath := filepath.Join(repoRoot, "pkg", "codex", "worker.go")
	content, err := os.ReadFile(workerGoPath)
	if err != nil {
		t.Fatalf("read %s: %v", workerGoPath, err)
	}
	if strings.Contains(string(content), "20 * time.Second") {
		t.Fatalf("%s still contains a hardcoded \"20 * time.Second\" literal -- the preflight budget must come from resolvedPreflightTimeout", workerGoPath)
	}
}

// TestAuthFailureRegexMatchesAcrossHosts asserts the Go trust-window
// invalidation vocabulary (providerAuthFailureRegexp in
// cmd/preflight_cache.go) and the TS isAuthError classifier
// (.aether/ts-host/src/worker-dispatch.ts) use byte-identical pattern
// bodies, enforcing the "keep the two patterns byte-identical" comment the
// code demands (WR-03). If they drift, the two hosts disagree on which
// failures clear the trust window: one host re-probes while the other
// trusts a stale success for up to an hour after an auth lapse.
func TestAuthFailureRegexMatchesAcrossHosts(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	// Go side: use the compiled regexp itself (not a source scrape) so the
	// comparison covers the bytes the runtime actually matches with.
	goPattern := providerAuthFailureRegexp.String()
	goBody := strings.TrimPrefix(goPattern, "(?i)")
	if goBody == goPattern {
		t.Fatalf("providerAuthFailureRegexp %q lost its (?i) case-insensitivity flag", goPattern)
	}

	tsPath := filepath.Join(repoRoot, ".aether", "ts-host", "src", "worker-dispatch.ts")
	tsContent, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("read %s: %v", tsPath, err)
	}
	tsMatch := regexp.MustCompile(`(?s)export function isAuthError.*?return /(.+?)/(\w*)\.test\(message\)`).FindSubmatch(tsContent)
	if tsMatch == nil {
		t.Fatalf("%s: could not find the isAuthError regex literal", tsPath)
	}
	tsBody := string(tsMatch[1])
	tsFlags := string(tsMatch[2])
	if !strings.Contains(tsFlags, "i") {
		t.Fatalf("%s: isAuthError regex lost its /i case-insensitivity flag (flags = %q)", tsPath, tsFlags)
	}

	if goBody != tsBody {
		t.Fatalf("auth-failure regex drift -- the two hosts no longer agree on which failures clear the trust window:\n  Go (cmd/preflight_cache.go): %q\n  TS (%s): %q", goBody, tsPath, tsBody)
	}
}

// TestSkipNoticeWordingMatchesAcrossHosts extracts the skip notice line from
// cmd/preflight_cache.go (via preflightSkipNoticeLine, the function that
// actually produces the CLI/JSON bytes) and from .aether/ts-host/src/host.ts
// and asserts the two strings are identical (D-06, T-163.2-27).
func TestSkipNoticeWordingMatchesAcrossHosts(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	goNotice := preflightSkipNoticeLine()

	hostPath := filepath.Join(repoRoot, ".aether", "ts-host", "src", "host.ts")
	content, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("read %s: %v", hostPath, err)
	}
	match := regexp.MustCompile(`"(Warning: preflight skipped via [^"]*)\\n"`).FindSubmatch(content)
	if match == nil {
		t.Fatalf("%s: could not find the skip notice string literal", hostPath)
	}
	tsNotice := string(match[1])

	if goNotice != tsNotice {
		t.Fatalf("skip notice wording drift:\n  Go: %q\n  TS: %q", goNotice, tsNotice)
	}
}
