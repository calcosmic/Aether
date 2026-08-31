package cmd

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
)

func phaseScopedPreflightDispatch(root string, phase int, workflow string) []codex.WorkerDispatch {
	return []codex.WorkerDispatch{{
		ID:         "phase-preflight",
		WorkerName: "Builder-Phase",
		Caste:      "builder",
		TaskID:     "phase-preflight",
		TaskBrief:  "Prove readiness is shared only inside one phase",
		Root:       root,
		Wave:       1,
		Workflow:   workflow,
		Phase:      phase,
	}}
}

func TestPreflightSuccessIsScopedToWorkerDispatchPhase(t *testing.T) {
	withTestPreflightStore(t)
	withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	root := t.TempDir()
	for _, workflow := range []string{"build", "build", "continue"} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 4, workflow)); err != nil {
			t.Fatalf("phase 4 %s preflight = %v, want nil", workflow, err)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("phase 4 build/continue probes = %d, want 1", fake.calls)
	}

	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 5, "build")); err != nil {
		t.Fatalf("phase 5 build preflight = %v, want nil", err)
	}
	if fake.calls != 2 {
		t.Fatalf("probes after entering phase 5 = %d, want 2; changing phase must beat the old one-hour TTL", fake.calls)
	}

	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, 5, "continue")); err != nil {
		t.Fatalf("phase 5 continue preflight = %v, want nil", err)
	}
	if fake.calls != 2 {
		t.Fatalf("phase 5 build/continue probes = %d, want 2 total", fake.calls)
	}
}

func TestPreflightPhaseZeroRetainsUnscopedTTLBehavior(t *testing.T) {
	withTestPreflightStore(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	now := time.Now().UTC()
	for i := 0; i < 2; i++ {
		status, _ := gatedProviderPreflight(context.Background(), fake, codex.PlatformClaude, t.TempDir(), now)
		if !status.Available {
			t.Fatalf("unscoped preflight %d reported unavailable", i+1)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("phase 0 probes = %d, want 1 inside the TTL", fake.calls)
	}
}

func TestPreflightInvalidationClearsEveryPhaseScope(t *testing.T) {
	withTestPreflightStore(t)
	withCapturedGateStderr(t)

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	root := t.TempDir()
	for _, phase := range []int{4, 5} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, phase, "build")); err != nil {
			t.Fatalf("warm phase %d preflight = %v, want nil", phase, err)
		}
	}
	if fake.calls != 2 {
		t.Fatalf("warm probes = %d, want 2 (one per phase)", fake.calls)
	}

	results := []codex.DispatchResult{{
		WorkerName: "Builder-Phase",
		Error:      errors.New("authentication failed"),
	}}
	if !invalidatePreflightCacheOnProviderFailure(codex.PlatformClaude, results) {
		t.Fatal("provider auth failure did not trigger invalidation")
	}

	for _, phase := range []int{4, 5} {
		if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(root, phase, "continue")); err != nil {
			t.Fatalf("phase %d preflight after invalidation = %v, want nil", phase, err)
		}
	}
	if fake.calls != 4 {
		t.Fatalf("probes after platform invalidation = %d, want 4; every cached phase scope must be removed", fake.calls)
	}
}

func TestPreflightOldSchemaCannotWarmEveryPhase(t *testing.T) {
	s := withTestPreflightStore(t)
	withCapturedGateStderr(t)
	now := time.Now().UTC()

	legacy := preflightCacheFile{
		SchemaVersion: 1,
		Entries: map[string]preflightCacheEntry{
			string(codex.PlatformClaude): {
				Platform:  string(codex.PlatformClaude),
				Available: true,
				CheckedAt: now.Format(time.RFC3339),
			},
		},
	}
	if err := s.SaveJSON(preflightCachePathRel, &legacy); err != nil {
		t.Fatalf("write legacy cache: %v", err)
	}

	fake := &gatePreflightFake{platform: codex.PlatformClaude, available: true}
	if err := preflightWorkerProvider(context.Background(), fake, phaseScopedPreflightDispatch(t.TempDir(), 4, "build")); err != nil {
		t.Fatalf("phase 4 preflight over legacy cache = %v, want nil", err)
	}
	if fake.calls != 1 {
		t.Fatalf("phase 4 probes over legacy cache = %d, want 1; a platform-only success cannot authorize every phase", fake.calls)
	}

	var upgraded preflightCacheFile
	if err := s.LoadJSON(preflightCachePathRel, &upgraded); err != nil {
		t.Fatalf("load upgraded cache: %v", err)
	}
	if upgraded.SchemaVersion != 2 {
		t.Fatalf("cache schema_version = %d, want 2", upgraded.SchemaVersion)
	}
}

func TestInternalWorkerAdapterPreflightCLIUsesPhaseScope(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s := withTestPreflightStore(t)
	forceJSONOutputModeForTest(t)

	var outBuf, errBuf bytes.Buffer
	stdout = &outBuf
	stderr = &errBuf
	rootCmd.SetArgs([]string{"internal-worker-adapter", "--preflight", "--simulate", "--phase", "7"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("internal-worker-adapter --preflight --phase 7 = %v; stderr=%s", err, errBuf.String())
	}

	var cache preflightCacheFile
	if err := s.LoadJSON(preflightCachePathRel, &cache); err != nil {
		t.Fatalf("load phase-scoped adapter cache: %v", err)
	}
	key := preflightCacheKey(codex.PlatformFake, 7)
	entry, ok := cache.Entries[key]
	if !ok {
		t.Fatalf("adapter cache keys = %v, want %q", cache.Entries, key)
	}
	if entry.Phase != 7 || !entry.Available {
		t.Fatalf("adapter phase entry = %+v, want successful phase 7 readiness", entry)
	}
	if _, ok := cache.Entries[preflightCacheKey(codex.PlatformFake, 0)]; ok {
		t.Fatal("--phase 7 warmed the unscoped phase-0 cache")
	}
}

func TestInternalWorkerAdapterPreflightCLIRejectsInvalidPhaseBeforeProbe(t *testing.T) {
	for _, phase := range []string{"-1", "not-a-number"} {
		t.Run(phase, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			s := withTestPreflightStore(t)
			forceJSONOutputModeForTest(t)
			if err := s.SaveJSON(preflightCachePathRel, &preflightCacheFile{
				SchemaVersion: preflightCacheSchemaVersion,
				Entries:       map[string]preflightCacheEntry{},
			}); err != nil {
				t.Fatalf("seed empty preflight cache: %v", err)
			}

			var outBuf, errBuf bytes.Buffer
			stdout = &outBuf
			stderr = &errBuf
			rootCmd.SetArgs([]string{"internal-worker-adapter", "--preflight", "--simulate", "--phase", phase})
			if err := rootCmd.Execute(); err == nil {
				t.Fatalf("--phase %q succeeded, want validation error", phase)
			}

			var cache preflightCacheFile
			if err := s.LoadJSON(preflightCachePathRel, &cache); err != nil {
				t.Fatalf("load cache after invalid phase: %v", err)
			}
			if len(cache.Entries) != 0 {
				t.Fatalf("--phase %q probed before validation; cache=%+v", phase, cache.Entries)
			}
		})
	}
}
