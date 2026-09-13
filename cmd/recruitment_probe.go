package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// recruitmentProbeTimeoutEnv is the environment variable
// resolvedRecruitmentProbeTimeout reads: AETHER_RECRUIT_PROBE_TIMEOUT, in Go
// duration syntax (e.g. "10s"). This is a DIFFERENT knob than
// AETHER_PREFLIGHT_TIMEOUT (the model round-trip / auth readiness budget,
// pkg/codex/platform_dispatch.go) -- this probe answers a different question
// ("can a helper nest natively here?"), not provider readiness, so it is not
// a duplicate of an existing setting the way a second AETHER_PROBE_TIMEOUT
// would have been (see cmd/recruitment-probe.ts's own deviation note).
const recruitmentProbeTimeoutEnv = "AETHER_RECRUIT_PROBE_TIMEOUT"

// recruitmentProbeDefaultTimeout is used when AETHER_RECRUIT_PROBE_TIMEOUT is
// absent, unparsable, or non-positive.
const recruitmentProbeDefaultTimeout = 10 * time.Second

// recruitmentProbeWarnedValuesMu guards recruitmentProbeWarnedValues, the
// once-per-distinct-bad-value dedupe resolvedRecruitmentProbeTimeout uses --
// mirroring resolvePreflightTimeoutMs's own warnedTimeoutValues Set
// (.aether/ts-host/src/preflight-config.ts) so a broken env var produces one
// loud warning, not one per probe call.
var (
	recruitmentProbeWarnedValuesMu sync.Mutex
	recruitmentProbeWarnedValues   = map[string]bool{}
)

// resolvedRecruitmentProbeTimeout reads AETHER_RECRUIT_PROBE_TIMEOUT and
// falls back to recruitmentProbeDefaultTimeout when the value is absent,
// unparsable, or non-positive -- the same resolve-warn-fallback shape
// resolvePreflightTimeoutMs uses on the TypeScript side.
func resolvedRecruitmentProbeTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(recruitmentProbeTimeoutEnv))
	if raw == "" {
		return recruitmentProbeDefaultTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		warnInvalidRecruitmentProbeTimeout(raw)
		return recruitmentProbeDefaultTimeout
	}
	return d
}

// warnInvalidRecruitmentProbeTimeout prints one stderr warning per distinct
// bad AETHER_RECRUIT_PROBE_TIMEOUT value seen by this process -- repeating
// the same line on every recruitment would drown the dispatch output it is
// trying to protect.
func warnInvalidRecruitmentProbeTimeout(raw string) {
	recruitmentProbeWarnedValuesMu.Lock()
	defer recruitmentProbeWarnedValuesMu.Unlock()
	if recruitmentProbeWarnedValues[raw] {
		return
	}
	recruitmentProbeWarnedValues[raw] = true
	fmt.Fprintf(os.Stderr, "Warning: %s=%q is not a positive Go duration (expected e.g. \"10s\", \"1500ms\") -- falling back to %s\n", recruitmentProbeTimeoutEnv, raw, recruitmentProbeDefaultTimeout)
}

// recruitmentProbeResult is a single native-nesting probe run's outcome,
// cached at most once per process. Every recorded field is evidence the
// probe itself gathered -- Verdict is never inferred from Platform, a
// version string, or a hardcoded table (this plan's own must_have).
type recruitmentProbeResult struct {
	Platform  string
	Binary    string
	BudgetMs  int64
	Supported bool
	Verdict   string
	ElapsedMs int64
	ProbedAt  time.Time
}

// recruitmentProbePrompt asks the resolved platform CLI, in its own child
// process, whether an agent-spawning tool (e.g. Task) appears in ITS OWN
// available tool list. The child's plain-text answer is the only thing this
// probe reads to decide Supported -- never the platform name or a version
// string.
const recruitmentProbePrompt = "List your available tools, then reply with exactly one line and nothing else: NATIVE_NESTING_AVAILABLE=true if an agent-spawning tool (for example, a Task tool) appears in your own available tool list, otherwise NATIVE_NESTING_AVAILABLE=false.\n"

// recruitmentProbeArgv builds the probe child's argv. AETHER_RECRUIT_PROBE_ARGS
// is a dedicated, test-only override (newline-delimited, mirroring
// recruitmentDispatchArgv's own AETHER_RECRUIT_ARGS convention) so a test can
// drive a deterministic fake binary without depending on an installed
// platform CLI. Production probing, absent that override, reuses the same
// minimal non-interactive codex exec flags dispatchRecruitment's own
// production path assumes.
func recruitmentProbeArgv() []string {
	if raw, ok := os.LookupEnv("AETHER_RECRUIT_PROBE_ARGS"); ok {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		return strings.Split(raw, "\n")
	}
	return []string{
		"--sandbox", "read-only",
		"--ask-for-approval", "never",
		"exec",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
	}
}

// nativeNestingReportedAvailable reads the probe child's plain-text answer.
// This is the ONLY place Supported is decided, and it never consults the
// platform name, a version string, or a hardcoded supported-platforms table.
func nativeNestingReportedAvailable(output string) bool {
	return strings.Contains(output, "NATIVE_NESTING_AVAILABLE=true")
}

// recruitmentProbeRunner launches the probe subprocess exactly once per
// call. Declared as a package-level var, mirroring hostedPreflightTimeout /
// recruitmentDispatchArgv's own "declared as a var so tests can override"
// discipline, so a test can substitute a deterministic, counting fake
// without depending on an installed platform CLI or a real model round-trip.
var recruitmentProbeRunner = defaultRecruitmentProbeRunner

func defaultRecruitmentProbeRunner(ctx context.Context, binary, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, recruitmentProbeArgv()...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(recruitmentProbePrompt)
	// The single deadline-vs-process-group fix this task requires -- the
	// exact mistake runAvailabilityProbeOnce's own comment documents
	// (pkg/codex/platform_dispatch.go): a bare exec.CommandContext cancels
	// only the direct child, and Run() then blocks until every inherited
	// pipe closes. codex.ConfigureWorkerCommand is the already-tested fix.
	codex.ConfigureWorkerCommand(cmd)
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

// recruitmentProbeState pairs a sync.Once with the result it guards, so
// probeNativeNestingOnce runs the underlying probe at most once per
// process per resolved platform.
type recruitmentProbeState struct {
	once   sync.Once
	result recruitmentProbeResult
}

var (
	recruitmentProbeStatesMu sync.Mutex
	recruitmentProbeStates   = map[string]*recruitmentProbeState{}
)

// probeNativeNestingOnce answers "can a helper nest natively here?" by
// trying it, cheaply, once, somewhere safe. The FIRST call per resolved
// platform launches a real (or test-substituted) probe process; every
// subsequent call in this process reads the cached result. No other file
// may launch this probe (cmd/recruitment_dispatch.go reads the cached
// result only).
func probeNativeNestingOnce() recruitmentProbeResult {
	platform := strings.TrimSpace(string(codex.DetectActivePlatform()))
	if platform == "" || platform == string(codex.PlatformUnknown) {
		platform = "unknown"
	}

	recruitmentProbeStatesMu.Lock()
	state, ok := recruitmentProbeStates[platform]
	if !ok {
		state = &recruitmentProbeState{}
		recruitmentProbeStates[platform] = state
	}
	recruitmentProbeStatesMu.Unlock()

	state.once.Do(func() {
		state.result = runRecruitmentProbe(platform)
	})
	return state.result
}

// resetRecruitmentProbeCacheForTest clears the whole per-platform probe
// cache. Test-only -- lets a test exercise both the first (real launch) and
// the cached (no launch) call deterministically.
func resetRecruitmentProbeCacheForTest() {
	recruitmentProbeStatesMu.Lock()
	defer recruitmentProbeStatesMu.Unlock()
	recruitmentProbeStates = map[string]*recruitmentProbeState{}
}

// runRecruitmentProbe performs one bounded, isolated probe run: a fresh
// temp directory outside the repository as the child's working directory
// (removed on every exit path), the resolved budget, and one retry --
// only on a timeout, matching availabilityProbeAttempts's own documented
// reasoning (pkg/codex/platform_dispatch.go). A missing binary or a refused
// credential fails immediately with no retry.
func runRecruitmentProbe(platform string) recruitmentProbeResult {
	binary := resolvedRecruitmentBinary()
	budget := resolvedRecruitmentProbeTimeout()
	probedAt := time.Now().UTC()

	dir, dirErr := os.MkdirTemp("", "aether-recruit-probe-*")
	if dirErr != nil {
		return recruitmentProbeResult{
			Platform: platform,
			Binary:   binary,
			BudgetMs: budget.Milliseconds(),
			Verdict:  "probe-setup-failed",
			ProbedAt: probedAt,
		}
	}
	defer os.RemoveAll(dir)

	start := time.Now()
	var output string
	var runErr error
	var timedOut bool
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), budget)
		output, runErr = recruitmentProbeRunner(ctx, binary, dir)
		timedOut = ctx.Err() == context.DeadlineExceeded
		cancel()
		if runErr == nil || !timedOut {
			break
		}
	}
	elapsed := time.Since(start)

	supported := false
	verdict := "unavailable"
	switch {
	case timedOut:
		verdict = "timeout"
	case runErr != nil:
		verdict = "probe-failed"
	default:
		supported = nativeNestingReportedAvailable(output)
		if supported {
			verdict = "available"
		}
	}

	return recruitmentProbeResult{
		Platform:  platform,
		Binary:    binary,
		BudgetMs:  budget.Milliseconds(),
		Supported: supported,
		Verdict:   verdict,
		ElapsedMs: elapsed.Milliseconds(),
		ProbedAt:  probedAt,
	}
}
