package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSafeIdentifierSegmentRejectsEscapes is the WP-10b unit gate. Swarm ids
// and chamber names become directory names, and one of them is handed straight
// to os.RemoveAll — so an id must never be able to select a directory outside
// its base.
func TestSafeIdentifierSegmentRejectsEscapes(t *testing.T) {
	base := t.TempDir()
	for _, bad := range []string{
		"", " ", ".", "..", "../..", "../../../etc",
		"a/b", `a\b`, "/absolute", "sub/../..", "C:evil",
	} {
		if _, err := safeIdentifierSegment(base, "swarm id", bad); err == nil {
			t.Errorf("safeIdentifierSegment accepted escaping identifier %q", bad)
		}
	}
	for _, good := range []string{"swarm_123", "auth-bug", "Chamber.One", "a.b.c"} {
		got, err := safeIdentifierSegment(base, "swarm id", good)
		if err != nil {
			t.Errorf("safeIdentifierSegment rejected legitimate identifier %q: %v", good, err)
			continue
		}
		if filepath.Dir(got) != base {
			t.Errorf("identifier %q resolved outside its base: %s", good, got)
		}
	}
}

// TestSwarmCleanupRejectsTraversalId is the end-to-end gate: the destructive
// path must refuse a traversing id and leave the filesystem untouched.
func TestSwarmCleanupRejectsTraversalId(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// A victim directory a traversing id would reach: two levels above the
	// swarms base is the store root's parent.
	victim := filepath.Join(store.BasePath(), "victim")
	if err := os.MkdirAll(victim, 0755); err != nil {
		t.Fatalf("create victim dir: %v", err)
	}
	canary := filepath.Join(victim, "keep.txt")
	if err := os.WriteFile(canary, []byte("must survive"), 0644); err != nil {
		t.Fatalf("write canary: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm-cleanup", "--id", "../victim"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-cleanup returned a hard error: %v", err)
	}

	if _, err := os.Stat(canary); err != nil {
		t.Fatalf("traversing swarm id deleted data outside the swarms directory: %v", err)
	}
}

// A legitimate id must still clean up, or the guard has broken the feature.
func TestSwarmCleanupStillRemovesLegitimateSwarm(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	swarmDir := filepath.Join(store.BasePath(), "swarms", "swarm_ok")
	if err := os.MkdirAll(swarmDir, 0755); err != nil {
		t.Fatalf("create swarm dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(swarmDir, "state.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("write swarm state: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm-cleanup", "--id", "swarm_ok"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm-cleanup returned error: %v", err)
	}
	if _, err := os.Stat(swarmDir); !os.IsNotExist(err) {
		t.Fatalf("legitimate swarm directory was not removed (err=%v)", err)
	}
}

// Chamber creation must refuse a traversing name rather than creating a
// directory outside the chambers tree.
func TestChamberCreateRejectsTraversalName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf strings.Builder
	_ = buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"chamber-create", "--name", "../escaped-chamber"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("chamber-create returned a hard error: %v", err)
	}
	// Nothing named escaped-chamber may exist anywhere under the temp root.
	var found string
	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && info.IsDir() && strings.Contains(filepath.Base(path), "escaped-chamber") {
			found = path
		}
		return nil
	})
	if found != "" {
		t.Fatalf("traversing chamber name created a directory outside the chambers tree: %s", found)
	}
}
