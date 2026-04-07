package storage

import (
	"testing"
)

// TestBoundaryGuard_ProtectedPathRejected verifies that CheckWrite returns an error
// when writing to COLONY_STATE.json without explicit Allow().
func TestBoundaryGuard_ProtectedPathRejected(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	err = bg.CheckWrite("COLONY_STATE.json")
	if err == nil {
		t.Fatal("expected error for protected path COLONY_STATE.json, got nil")
	}
}

// TestBoundaryGuard_ProtectedPathAllowed verifies that CheckWrite returns nil
// when writing to a protected path after Allow() was called.
func TestBoundaryGuard_ProtectedPathAllowed(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	bg.Allow("COLONY_STATE.json")
	err = bg.CheckWrite("COLONY_STATE.json")
	if err != nil {
		t.Fatalf("expected nil after Allow(), got: %v", err)
	}
}

// TestBoundaryGuard_NonProtectedPath verifies that CheckWrite returns nil
// for non-protected paths like pheromones.json.
func TestBoundaryGuard_NonProtectedPath(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	err = bg.CheckWrite("pheromones.json")
	if err != nil {
		t.Fatalf("expected nil for non-protected path, got: %v", err)
	}
}

// TestBoundaryGuard_AllProtectedPaths verifies that all protected paths are
// rejected: COLONY_STATE.json, session.json, checkpoints/ prefix, midden/ prefix.
func TestBoundaryGuard_AllProtectedPaths(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	protectedPaths := []string{
		"COLONY_STATE.json",
		"session.json",
		"checkpoints/",
		"checkpoints/auto-20240101-120000.json",
		"midden/",
		"midden/midden.json",
	}

	for _, path := range protectedPaths {
		t.Run(path, func(t *testing.T) {
			err := bg.CheckWrite(path)
			if err == nil {
				t.Errorf("expected error for protected path %q, got nil", path)
			}
		})
	}
}

// TestBoundaryGuard_AllowOnlySpecificPath verifies that Allow() only authorizes
// the specific path, not other protected paths.
func TestBoundaryGuard_AllowOnlySpecificPath(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	bg.Allow("COLONY_STATE.json")

	if err := bg.CheckWrite("COLONY_STATE.json"); err != nil {
		t.Fatalf("COLONY_STATE.json should be allowed: %v", err)
	}

	if err := bg.CheckWrite("session.json"); err == nil {
		t.Fatal("session.json should still be rejected after allowing COLONY_STATE.json")
	}
}

// TestBoundaryGuard_SubdirectoryProtection verifies that writes to subdirectories
// of protected paths (checkpoints/, midden/) are rejected.
func TestBoundaryGuard_SubdirectoryProtection(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	bg := NewBoundaryGuard(s)

	subPaths := []string{
		"checkpoints/snapshot.json",
		"checkpoints/auto-20240101.json",
		"midden/failure-record.json",
	}

	for _, path := range subPaths {
		err := bg.CheckWrite(path)
		if err == nil {
			t.Errorf("expected error for protected subdirectory path %q, got nil", path)
		}
	}
}
