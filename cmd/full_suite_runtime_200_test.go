package cmd

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFullSuiteRuntime200PartitionExactOnce(t *testing.T) {
	discovered := []string{"TestDelta", "TestAlpha", "TestCharlie", "TestBravo"}
	serial := map[string]struct{}{"TestCharlie": {}}
	costs := map[string]time.Duration{
		"TestAlpha":   4 * time.Second,
		"TestBravo":   time.Second,
		"TestCharlie": 9 * time.Second,
		"TestDelta":   3 * time.Second,
	}

	first, err := planFullSuiteLanes(discovered, serial, costs, 2)
	if err != nil {
		t.Fatal(err)
	}
	second, err := planFullSuiteLanes(discovered, serial, costs, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("full-suite partition is not deterministic:\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if err := validateFullSuitePlan(discovered, first); err != nil {
		t.Fatalf("valid exact-once partition rejected: %v", err)
	}
	if len(first) != 3 || first[0].Name != fullSuiteSerialLaneName || !first[0].Serial || !reflect.DeepEqual(first[0].Tests, []string{"TestCharlie"}) {
		t.Fatalf("serial lane = %#v, want the one explicitly classified test", first)
	}

	duplicated := append([]fullSuiteLane(nil), first...)
	duplicated[1].Tests = append(duplicated[1].Tests, "TestCharlie")
	if err := validateFullSuitePlan(discovered, duplicated); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate plan validation error = %v, want explicit duplicate diagnostic", err)
	}
	omitted := append([]fullSuiteLane(nil), first...)
	omitted[0].Tests = nil
	if err := validateFullSuitePlan(discovered, omitted); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("omitted plan validation error = %v, want explicit missing diagnostic", err)
	}
	if _, err := planFullSuiteLanes(append(discovered, "TestAlpha"), serial, costs, 2); err == nil {
		t.Fatal("duplicate discovery entry was silently accepted")
	}
}

func TestFullSuiteRuntime200PreservesFocusedRun(t *testing.T) {
	tests := []struct {
		name string
		in   fullSuiteInvocation
		want bool
	}{
		{name: "exact unfiltered count one", in: fullSuiteInvocation{Count: 1}, want: true},
		{name: "focused selector", in: fullSuiteInvocation{Count: 1, Run: "^TestOne$", RunExplicit: true}},
		{name: "explicit empty selector", in: fullSuiteInvocation{Count: 1, RunExplicit: true}},
		{name: "discovery listing", in: fullSuiteInvocation{Count: 1, List: "^Test", ListExplicit: true}},
		{name: "benchmark", in: fullSuiteInvocation{Count: 1, Bench: ".", BenchExplicit: true}},
		{name: "fuzzing", in: fullSuiteInvocation{Count: 1, Fuzz: "FuzzOne", FuzzExplicit: true}},
		{name: "skip filter", in: fullSuiteInvocation{Count: 1, Skip: "Slow", SkipExplicit: true}},
		{name: "profile output", in: fullSuiteInvocation{Count: 1, Profiled: true}},
		{name: "caller requested repetitions", in: fullSuiteInvocation{Count: 2}},
		{name: "child recursion marker", in: fullSuiteInvocation{Count: 1, ShardMarker: "123:parallel-01"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRunFullSuiteController(tc.in); got != tc.want {
				t.Fatalf("shouldRunFullSuiteController(%+v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFullSuiteRuntime200PropagatesFailure(t *testing.T) {
	lanes := []fullSuiteLane{
		{Name: "parallel-01", Tests: []string{"TestAlpha"}},
		{Name: "parallel-02", Tests: []string{"TestBravo"}},
	}
	runner := func(_ context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
		if request.Lane.Name == "parallel-02" {
			return fullSuiteChildResult{
				Output:   "panic: deliberate child diagnostic\nWARNING: DATA RACE",
				Executed: []string{"TestBravo"},
				Err:      errors.New("exit status 2"),
			}
		}
		return fullSuiteChildResult{Executed: append([]string(nil), request.Lane.Tests...)}
	}

	report, err := runFullSuiteLanes(context.Background(), "/current/cmd.test", lanes, 2, runner)
	if err == nil {
		t.Fatal("child failure was swallowed")
	}
	for _, want := range []string{"parallel-02", "exit status 2", "deliberate child diagnostic", "DATA RACE"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("failure %q omitted actionable child detail %q", err, want)
		}
	}
	if report.Passed {
		t.Fatalf("failed child produced passing report: %+v", report)
	}
}

func TestFullSuiteRuntime200UsesCurrentBinary(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	lanes := []fullSuiteLane{{Name: "parallel-01", Tests: []string{"TestAlpha", "TestBravo"}}}
	var observed fullSuiteChildRequest
	runner := func(_ context.Context, request fullSuiteChildRequest) fullSuiteChildResult {
		observed = request
		return fullSuiteChildResult{Executed: append([]string(nil), request.Lane.Tests...)}
	}
	report, err := runFullSuiteLanes(context.Background(), executable, lanes, 1, runner)
	if err != nil {
		t.Fatal(err)
	}
	if observed.Executable != executable {
		t.Fatalf("child executable = %q, want current race/build-instrumented binary %q", observed.Executable, executable)
	}
	if strings.Contains(strings.Join(observed.Args, " "), "go test") {
		t.Fatalf("child recursively invoked go test: %v", observed.Args)
	}
	if !strings.Contains(strings.Join(observed.Args, " "), "-test.run=^(TestAlpha|TestBravo)$") {
		t.Fatalf("child args do not carry the exact anchored lane selector: %v", observed.Args)
	}
	if report.Discovered != 2 || report.Executed != 2 || !report.Passed {
		t.Fatalf("current-binary report = %+v, want 2/2 passing", report)
	}

	realLane := []fullSuiteLane{{Name: "probe", Tests: []string{"TestFullSuiteRuntime200CurrentBinaryProbe"}}}
	realReport, err := runFullSuiteLanes(context.Background(), executable, realLane, 1, runFullSuiteChildProcess)
	if err != nil {
		t.Fatalf("current test binary could not execute an exact focused child: %v", err)
	}
	if realReport.Discovered != 1 || realReport.Executed != 1 || !realReport.Passed {
		t.Fatalf("real current-binary report = %+v, want 1/1 passing", realReport)
	}
}

func TestFullSuiteRuntime200CurrentBinaryProbe(t *testing.T) {
	if strings.TrimSpace(os.Getenv(fullSuiteShardEnv)) == "" {
		return
	}
}
