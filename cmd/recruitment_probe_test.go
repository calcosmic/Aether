package cmd

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

// TestRecruitmentProbeReadsTimeoutEnv proves resolvedRecruitmentProbeTimeout
// follows the resolve-warn-fallback shape resolvePreflightTimeoutMs uses on
// the TypeScript side: a valid Go duration wins, an invalid one falls back
// to the compiled default.
func TestRecruitmentProbeReadsTimeoutEnv(t *testing.T) {
	t.Setenv(recruitmentProbeTimeoutEnv, "2500ms")
	if got := resolvedRecruitmentProbeTimeout(); got != 2500*time.Millisecond {
		t.Fatalf("resolvedRecruitmentProbeTimeout() = %v, want 2.5s", got)
	}

	t.Setenv(recruitmentProbeTimeoutEnv, "not-a-duration")
	if got := resolvedRecruitmentProbeTimeout(); got != recruitmentProbeDefaultTimeout {
		t.Fatalf("resolvedRecruitmentProbeTimeout() with invalid value = %v, want default %v", got, recruitmentProbeDefaultTimeout)
	}

	t.Setenv(recruitmentProbeTimeoutEnv, "")
	if got := resolvedRecruitmentProbeTimeout(); got != recruitmentProbeDefaultTimeout {
		t.Fatalf("resolvedRecruitmentProbeTimeout() with empty value = %v, want default %v", got, recruitmentProbeDefaultTimeout)
	}
}

// TestRecruitmentProbeBoundsWallClock proves a probe whose fake launcher
// never returns (the counterpart to the pre-configureWorkerCommand 60s
// overrun documented on runAvailabilityProbeOnce, pkg/codex/platform_dispatch.go)
// still resolves within a small multiple of a 200ms budget -- the same
// measurement TestRecruitmentDispatchTerminatesWholeProcessGroup already
// uses for dispatchRecruitment's own process-group teardown.
func TestRecruitmentProbeBoundsWallClock(t *testing.T) {
	resetRecruitmentProbeCacheForTest()
	defer resetRecruitmentProbeCacheForTest()

	t.Setenv(recruitmentProbeTimeoutEnv, "200ms")

	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}

	start := time.Now()
	result := probeNativeNestingOnce()
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("probeNativeNestingOnce took %v against a 200ms budget, want well under 2s", elapsed)
	}
	if result.Supported {
		t.Fatalf("expected an unsupported verdict on timeout, got Supported=true (%+v)", result)
	}
	if result.Verdict == "" {
		t.Fatal("expected a non-empty verdict on timeout")
	}
}

// TestRecruitmentProbeRunsOncePerProcess proves two calls to
// probeNativeNestingOnce in one process produce exactly one launch --
// the second reads the cached result, per this plan's own must_have.
func TestRecruitmentProbeRunsOncePerProcess(t *testing.T) {
	resetRecruitmentProbeCacheForTest()
	defer resetRecruitmentProbeCacheForTest()

	var launches int64
	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		atomic.AddInt64(&launches, 1)
		return "NATIVE_NESTING_AVAILABLE=false", nil
	}

	first := probeNativeNestingOnce()
	second := probeNativeNestingOnce()

	if got := atomic.LoadInt64(&launches); got != 1 {
		t.Fatalf("expected exactly 1 launch across two calls, got %d", got)
	}
	if first.ProbedAt != second.ProbedAt {
		t.Fatalf("expected the cached result to be returned byte-identical, got first=%+v second=%+v", first, second)
	}
}

// TestRecruitmentProbeDirectoryIsCleanedUp proves the probe's working
// directory is a fresh temp directory outside the repository, and that it is
// removed once the probe returns, on both the success and the timeout path.
func TestRecruitmentProbeDirectoryIsCleanedUp(t *testing.T) {
	t.Run("success path", func(t *testing.T) {
		resetRecruitmentProbeCacheForTest()
		defer resetRecruitmentProbeCacheForTest()

		var recordedDir string
		previous := recruitmentProbeRunner
		defer func() { recruitmentProbeRunner = previous }()
		recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
			recordedDir = dir
			return "NATIVE_NESTING_AVAILABLE=true", nil
		}

		result := probeNativeNestingOnce()
		if recordedDir == "" {
			t.Fatal("fixture is broken: recruitmentProbeRunner was never called")
		}
		if !result.Supported {
			t.Fatalf("expected a supported verdict, got %+v", result)
		}
		if _, err := os.Stat(recordedDir); !os.IsNotExist(err) {
			t.Fatalf("expected probe directory %q to be removed after a successful probe, stat err = %v", recordedDir, err)
		}
	})

	t.Run("timeout path", func(t *testing.T) {
		resetRecruitmentProbeCacheForTest()
		defer resetRecruitmentProbeCacheForTest()

		t.Setenv(recruitmentProbeTimeoutEnv, "50ms")

		var recordedDir string
		previous := recruitmentProbeRunner
		defer func() { recruitmentProbeRunner = previous }()
		recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
			recordedDir = dir
			<-ctx.Done()
			return "", ctx.Err()
		}

		probeNativeNestingOnce()
		if recordedDir == "" {
			t.Fatal("fixture is broken: recruitmentProbeRunner was never called")
		}
		if _, err := os.Stat(recordedDir); !os.IsNotExist(err) {
			t.Fatalf("expected probe directory %q to be removed after a timed-out probe, stat err = %v", recordedDir, err)
		}
	})
}

// TestRecruitmentProbeNeverInfersFromPlatformName proves a probe verdict
// comes only from the launcher's reported output, never from a hardcoded
// platform/version table -- the must_have this plan names explicitly.
// Forcing two different reported outputs on the SAME resolved binary/platform
// must produce two different verdicts; a hardcoded table keyed on platform
// name alone could not do this.
func TestRecruitmentProbeNeverInfersFromPlatformName(t *testing.T) {
	previous := recruitmentProbeRunner
	defer func() { recruitmentProbeRunner = previous }()

	resetRecruitmentProbeCacheForTest()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		return "NATIVE_NESTING_AVAILABLE=true", nil
	}
	supported := probeNativeNestingOnce()
	if !supported.Supported {
		t.Fatalf("expected Supported=true when the launcher reports availability, got %+v", supported)
	}

	resetRecruitmentProbeCacheForTest()
	recruitmentProbeRunner = func(ctx context.Context, binary, dir string) (string, error) {
		return "NATIVE_NESTING_AVAILABLE=false", nil
	}
	unsupported := probeNativeNestingOnce()
	if unsupported.Supported {
		t.Fatalf("expected Supported=false when the launcher reports unavailability, got %+v", unsupported)
	}
}

// TestRecruitmentAdapterChoice proves chooseRecruitmentAdapter's own
// decision rules in isolation: root-mediated is the declared default unless
// native-bind is explicitly requested; native-bind requires BOTH an
// explicit request and a probe reporting Supported; a request with no
// probe result at all (the zero value) refuses rather than silently
// defaulting to native; and two calls against an identical probe state
// select the same adapter.
func TestRecruitmentAdapterChoice(t *testing.T) {
	t.Run("root-mediated is the default when native-bind is not requested", func(t *testing.T) {
		kind, decision := chooseRecruitmentAdapter(recruitmentProbeResult{}, recruitmentAdapterRootMediated)
		if !decision.Allowed {
			t.Fatalf("expected root-mediated to be allowed by default, got %+v", decision)
		}
		if kind != recruitmentAdapterRootMediated {
			t.Fatalf("expected root-mediated, got %q", kind)
		}
	})

	t.Run("root-mediated wins even when the probe reports native nesting available", func(t *testing.T) {
		probe := recruitmentProbeResult{Platform: "codex", Supported: true, Verdict: "available"}
		kind, decision := chooseRecruitmentAdapter(probe, recruitmentAdapterRootMediated)
		if !decision.Allowed || kind != recruitmentAdapterRootMediated {
			t.Fatalf("expected root-mediated (the declared default) even with native nesting available, got kind=%q decision=%+v", kind, decision)
		}
	})

	t.Run("native-bind is chosen only when explicitly requested and the probe reports supported", func(t *testing.T) {
		probe := recruitmentProbeResult{Platform: "codex", Supported: true, Verdict: "available"}
		kind, decision := chooseRecruitmentAdapter(probe, recruitmentAdapterNativeBind)
		if !decision.Allowed {
			t.Fatalf("expected native-bind to be allowed when requested and supported, got %+v", decision)
		}
		if kind != recruitmentAdapterNativeBind {
			t.Fatalf("expected native-bind, got %q", kind)
		}
	})

	t.Run("native-bind requested but the probe reports unsupported refuses with reason platform", func(t *testing.T) {
		probe := recruitmentProbeResult{Platform: "codex", Supported: false, Verdict: "unavailable"}
		kind, decision := chooseRecruitmentAdapter(probe, recruitmentAdapterNativeBind)
		if decision.Allowed {
			t.Fatalf("expected native-bind to be refused when the probe reports unsupported, got kind=%q", kind)
		}
		if decision.Reason != "platform" {
			t.Fatalf("expected refusal reason %q, got %q", "platform", decision.Reason)
		}
		if kind != "" {
			t.Fatalf("expected an empty adapter kind on refusal, got %q", kind)
		}
	})

	t.Run("native-bind requested with no probe result at all fails closed, never silently defaults to native", func(t *testing.T) {
		kind, decision := chooseRecruitmentAdapter(recruitmentProbeResult{}, recruitmentAdapterNativeBind)
		if decision.Allowed {
			t.Fatalf("expected a refusal with no probe result recorded, got kind=%q", kind)
		}
		if decision.Reason != "platform" {
			t.Fatalf("expected refusal reason %q, got %q", "platform", decision.Reason)
		}
		if kind == recruitmentAdapterNativeBind {
			t.Fatal("must never silently default to native-bind with no probe result")
		}
	})

	t.Run("two runs with an identical probe state select the same adapter", func(t *testing.T) {
		probe := recruitmentProbeResult{Platform: "codex", Supported: true, Verdict: "available", BudgetMs: 5000}
		firstKind, firstDecision := chooseRecruitmentAdapter(probe, recruitmentAdapterNativeBind)
		secondKind, secondDecision := chooseRecruitmentAdapter(probe, recruitmentAdapterNativeBind)
		if firstKind != secondKind {
			t.Fatalf("expected identical adapter choice across repeated calls, got %q then %q", firstKind, secondKind)
		}
		if firstDecision != secondDecision {
			t.Fatalf("expected identical decision across repeated calls, got %+v then %+v", firstDecision, secondDecision)
		}
	})
}

// recruitmentAdapterKindIdentifierNames maps each declared
// recruitmentAdapterKind value to the Go identifier name that declares it,
// so TestEveryDeclaredAdapterCanDispatch can cross-check
// recruitmentAdapterKinds()'s RUNTIME output (never a hand-typed list of
// kinds) against the identifiers actually used as case labels in
// chooseRecruitmentAdapter's dispatch switch.
var recruitmentAdapterKindIdentifierNames = map[recruitmentAdapterKind]string{
	recruitmentAdapterRootMediated: "recruitmentAdapterRootMediated",
	recruitmentAdapterNativeBind:   "recruitmentAdapterNativeBind",
}

// TestEveryDeclaredAdapterCanDispatch is an AST-derived inventory of
// recruitmentAdapterKinds() cross-checked against chooseRecruitmentAdapter's
// own dispatch switch in cmd/recruitment_dispatch.go: it fails by name when
// a declared kind has no branch, and it fails by name when the switch has a
// branch for a kind recruitmentAdapterKinds() does not declare.
func TestEveryDeclaredAdapterCanDispatch(t *testing.T) {
	declared := recruitmentAdapterKinds()
	if len(declared) == 0 {
		t.Fatal("fixture is broken: recruitmentAdapterKinds() returned no declared adapter kinds")
	}
	for _, kind := range declared {
		if _, ok := recruitmentAdapterKindIdentifierNames[kind]; !ok {
			t.Fatalf("recruitmentAdapterKinds() returned %q, which has no registered identifier name in recruitmentAdapterKindIdentifierNames -- update this test", kind)
		}
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "recruitment_dispatch.go", nil, 0)
	if err != nil {
		t.Fatalf("parse recruitment_dispatch.go: %v", err)
	}

	var switchStmt *ast.SwitchStmt
	ast.Inspect(file, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Name.Name != "chooseRecruitmentAdapter" {
			return true
		}
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if sw, ok := inner.(*ast.SwitchStmt); ok && switchStmt == nil {
				switchStmt = sw
			}
			return true
		})
		return false
	})
	if switchStmt == nil {
		t.Fatal("chooseRecruitmentAdapter has no switch statement over the declared adapter kinds -- cannot verify branch completeness")
	}

	branchNames := map[string]bool{}
	for _, stmt := range switchStmt.Body.List {
		clause, ok := stmt.(*ast.CaseClause)
		if !ok {
			continue
		}
		for _, expr := range clause.List {
			if ident, ok := expr.(*ast.Ident); ok {
				branchNames[ident.Name] = true
			}
		}
	}

	for _, kind := range declared {
		name := recruitmentAdapterKindIdentifierNames[kind]
		if !branchNames[name] {
			t.Errorf("declared adapter kind %q (%s) has no branch in chooseRecruitmentAdapter's dispatch switch", kind, name)
		}
	}

	declaredNames := map[string]bool{}
	for _, kind := range declared {
		declaredNames[recruitmentAdapterKindIdentifierNames[kind]] = true
	}
	for name := range branchNames {
		if !declaredNames[name] {
			t.Errorf("chooseRecruitmentAdapter's dispatch switch has a branch for %q, which recruitmentAdapterKinds() does not declare", name)
		}
	}
}

// TestRecruitmentDispatchResultNamesOneAdapter drives a real dispatchRecruitment
// call and asserts the returned recruitmentDispatchResult always carries
// exactly one declared adapter kind -- the dispatch-record half of this
// plan's "every dispatch names which mechanism carried it" must_have.
// Threading AdapterKind onto the durable recruitmentResult (cmd/recruitment_result.go)
// is 203-07's file to own in this wave (see this plan's own SUMMARY for the
// cross-plan wiring note); this test proves the half owned here.
func TestRecruitmentDispatchResultNamesOneAdapter(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	t.Setenv("AETHER_RECRUIT_BINARY", os.Args[0])
	t.Setenv("AETHER_RECRUIT_ARGS", "-test.run=^TestRecruitmentTracerEndToEnd$")
	t.Setenv("AETHER_RECRUIT_TIMEOUT", "30s")
	t.Setenv("AETHER_RECRUIT_CHILD", "1")

	intent := recruitmentIntent{
		SchemaVersion: recruitmentSchemaVersion,
		ParentName:    "A1",
		Caste:         "builder",
		Objective:     "prove adapter naming",
		Reason:        "test",
		Workspace:     tmpDir,
	}

	result, err := dispatchRecruitment(intent, "AdapterFixture-1")
	if err != nil {
		t.Fatalf("dispatchRecruitment returned an unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("dispatchRecruitment returned a nil result")
	}
	found := false
	for _, kind := range recruitmentAdapterKinds() {
		if result.AdapterKind == kind {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected AdapterKind to be one of %v, got %q", recruitmentAdapterKinds(), result.AdapterKind)
	}
}
