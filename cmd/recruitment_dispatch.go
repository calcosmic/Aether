package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

// recruitRecruitTimeoutEnv is the environment variable resolvedRecruitmentTimeout
// reads: AETHER_RECRUIT_TIMEOUT, in Go duration syntax (e.g. "5m"). BIO-03's
// own requirement is a CONFIGURABLE, bounded timeout -- the folded todo
// (2026-08-01-ts-host-preflight-hardcoded-timeout.md) names a hardcoded
// probe timeout as the counter-example this must not repeat.
const recruitRecruitTimeoutEnv = "AETHER_RECRUIT_TIMEOUT"

// defaultRecruitmentTimeout is used when AETHER_RECRUIT_TIMEOUT is absent or
// unparsable, mirroring the platform dispatcher's own 10-minute default
// (.aether/ts-host/src/platform-dispatcher.ts:172).
const defaultRecruitmentTimeout = 10 * time.Minute

// resolvedRecruitmentTimeout reads AETHER_RECRUIT_TIMEOUT and falls back to
// defaultRecruitmentTimeout when the value is absent, unparsable, or
// non-positive -- D-19 fail-closed in spirit: a malformed value never
// silently grows the timeout past the safe default.
func resolvedRecruitmentTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv(recruitRecruitTimeoutEnv))
	if raw == "" {
		return defaultRecruitmentTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultRecruitmentTimeout
	}
	return d
}

// resolvedRecruitmentBinary resolves which executable dispatchRecruitment
// launches. AETHER_RECRUIT_BINARY is a dedicated override this tracer's own
// tests use to substitute a deterministic child; production dispatch, absent
// that override, resolves through the SAME platform resolution the codex
// worker dispatcher already uses (AETHER_CODEX_PATH, default "codex") --
// per this plan's instruction to resolve through the existing resolution
// rather than inventing a second one.
func resolvedRecruitmentBinary() string {
	if v := strings.TrimSpace(os.Getenv("AETHER_RECRUIT_BINARY")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("AETHER_CODEX_PATH")); v != "" {
		return v
	}
	return "codex"
}

// recruitmentDispatchArgv builds the child's argv. AETHER_RECRUIT_ARGS is a
// dedicated, test-only override, newline-delimited (never space-delimited --
// a test driving a shell child needs a single arg that itself contains
// spaces, e.g. `sh -c "sleep 5 & exit 0"`). Production dispatch, absent that
// override, builds a minimal argv directly from the intent's own fields.
func recruitmentDispatchArgv(intent recruitmentIntent) []string {
	if raw, ok := os.LookupEnv("AETHER_RECRUIT_ARGS"); ok {
		if strings.TrimSpace(raw) == "" {
			return nil
		}
		return strings.Split(raw, "\n")
	}
	return []string{"--caste", intent.Caste, "--objective", intent.Objective}
}

// recruitmentAdapterKind is the closed enum of dispatch mechanisms a
// recruitment can be carried by (SYN-203-04). recruitmentAdapterRootMediated
// is the declared v1 default and, per the synthesis ruling on SYN-203-04
// (native nesting is a probed, non-launch-blocking enhancement only), the
// only mechanism this phase's dispatch path actually uses; native-bind
// exists as a fully-tested, available choice for a future plan that adds a
// requested-adapter field to recruitmentIntent.
type recruitmentAdapterKind string

const (
	recruitmentAdapterRootMediated recruitmentAdapterKind = "root-mediated"
	recruitmentAdapterNativeBind   recruitmentAdapterKind = "native-bind"
)

// recruitmentAdapterKinds returns every declared adapter kind -- the same
// completeness convention ColonyLiveEpisodeKinds() uses (pkg/events/colony_live.go),
// and for the same reason: TestEveryDeclaredAdapterCanDispatch derives its
// inventory from this function's RUNTIME output, never a hand-typed list,
// so a kind declared here with no dispatch branch fails that test by name.
func recruitmentAdapterKinds() []recruitmentAdapterKind {
	return []recruitmentAdapterKind{
		recruitmentAdapterRootMediated,
		recruitmentAdapterNativeBind,
	}
}

// chooseRecruitmentAdapter decides which adapter carries one recruitment.
// Candidates are evaluated in the fixed order recruitmentAdapterKinds()
// declares (root-mediated then native-bind), so the same probe state always
// selects the same adapter. root-mediated is the declared default and wins
// unless native-bind was explicitly requested; native-bind is chosen only
// when it was requested AND probe.Supported is true. Any other native-bind
// request -- including one carrying the zero-value recruitmentProbeResult,
// i.e. no probe result recorded at all -- refuses with reason "platform"
// rather than silently defaulting to native (this plan's own must_have).
func chooseRecruitmentAdapter(probe recruitmentProbeResult, requested recruitmentAdapterKind) (recruitmentAdapterKind, recruitmentDecisionResult) {
	for _, candidate := range recruitmentAdapterKinds() {
		switch candidate {
		case recruitmentAdapterRootMediated:
			if requested != recruitmentAdapterNativeBind {
				return recruitmentAdapterRootMediated, recruitmentDecisionResult{
					Allowed: true,
					Reason:  "default",
					Detail:  "root-mediated subprocess dispatch (declared default, SYN-203-04)",
				}
			}
		case recruitmentAdapterNativeBind:
			if requested != recruitmentAdapterNativeBind {
				continue
			}
			if probe.Supported {
				return recruitmentAdapterNativeBind, recruitmentDecisionResult{
					Allowed: true,
					Reason:  "probed",
					Detail:  fmt.Sprintf("native nesting probe reported available on %q", probe.Platform),
				}
			}
			detail := fmt.Sprintf("native-bind requested but platform %q does not support native nesting (probe verdict: %s)", probe.Platform, probe.Verdict)
			if strings.TrimSpace(probe.Platform) == "" {
				detail = "native-bind requested but no probe result was recorded; unsupported nesting assumed"
			}
			return "", recruitmentDecisionResult{Allowed: false, Reason: "platform", Detail: detail}
		}
	}
	return "", recruitmentDecisionResult{Allowed: false, Reason: "platform", Detail: "no adapter candidate matched the request"}
}

// recruitmentDispatchResult is dispatchRecruitment's own outcome shape --
// deliberately smaller than recruitmentResult (cmd/recruitment_result.go):
// this is what actually happened to the process, before that outcome is
// bound into the exactly-once ledger.
type recruitmentDispatchResult struct {
	TerminalStatus string
	Summary        string
	// AdapterKind names the mechanism that carried this dispatch
	// (SYN-203-04's "every dispatch names which mechanism carried it").
	// Threading this onto the durable recruitmentResult
	// (cmd/recruitment_result.go) is a follow-up wiring step owned by
	// whichever plan owns that file in this wave -- see this plan's own
	// SUMMARY for the cross-plan note.
	AdapterKind recruitmentAdapterKind
}

// dispatchRecruitment runs the admitted child in a leased workspace under a
// bounded, process-group-safe timeout. The workspace lease is validated
// against the colony root through validateSpendContainedPath -- the same
// containment boundary the spend subsystem already uses (cmd/spend_session_capture.go)
// -- so a caller cannot point a recruitment at a path outside the colony;
// that refusal happens here, before any process is started.
func dispatchRecruitment(intent recruitmentIntent, childName string) (*recruitmentDispatchResult, error) {
	root := repoRootFromStore(store)
	workspace, err := validateSpendContainedPath(root, intent.Workspace, "recruitment workspace")
	if err != nil {
		return nil, fmt.Errorf("recruitment workspace refused: %w", err)
	}

	binary := resolvedRecruitmentBinary()
	argv := recruitmentDispatchArgv(intent)

	ctx, cancel := context.WithTimeout(context.Background(), resolvedRecruitmentTimeout())
	defer cancel()

	execCmd := exec.CommandContext(ctx, binary, argv...)
	execCmd.Dir = workspace
	// The recruited child knows it was recruited, and by whom -- a
	// legitimate, intentionally observable side effect of this dispatch,
	// not a test-only seam.
	execCmd.Env = append(os.Environ(),
		"AETHER_RECRUIT_CHILD=1",
		"AETHER_RECRUIT_PARENT="+intent.ParentName,
		"AETHER_RECRUIT_CASTE="+intent.Caste,
		"AETHER_RECRUIT_CHILD_NAME="+childName,
	)
	// The single deadline-vs-process-group fix this task requires: a bare
	// exec.CommandContext cancels only the direct child, and Run() then
	// blocks until every inherited pipe closes -- runAvailabilityProbeOnce's
	// own comment (pkg/codex/platform_dispatch.go) documents exactly this
	// mistake. codex.ConfigureWorkerCommand is the already-tested fix
	// (process group, signal-the-whole-group Cancel, WaitDelay backstop)
	// exported for exactly this kind of caller outside pkg/codex.
	codex.ConfigureWorkerCommand(execCmd)

	var output strings.Builder
	execCmd.Stdout = &output
	execCmd.Stderr = &output

	runErr := execCmd.Run()

	status := "completed"
	if runErr != nil {
		if ctx.Err() == context.DeadlineExceeded {
			status = "timeout"
		} else {
			status = "failed"
		}
	}
	summary := strings.TrimSpace(output.String())
	if summary == "" && runErr != nil {
		summary = runErr.Error()
	}

	// recruitmentIntent carries no requested-adapter field yet (that is a
	// future plan's job, once native-bind has a real caller) -- until then,
	// every dispatch explicitly requests the synthesis-declared default so
	// the mechanism is always named on the record by going THROUGH
	// chooseRecruitmentAdapter, never by hand-writing the literal string.
	// Because the default branch never reads probe, this never launches
	// probeNativeNestingOnce -- a plain recruitment pays nothing for a probe
	// it never needed (SYN-203-05's own must_have).
	adapter, _ := chooseRecruitmentAdapter(recruitmentProbeResult{}, recruitmentAdapterRootMediated)

	return &recruitmentDispatchResult{TerminalStatus: status, Summary: summary, AdapterKind: adapter}, nil
}
