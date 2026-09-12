package cmd

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// --- Task 1: one resolver owns effective scope, expiry and strength ---
// (floatPtr is already declared in cmd/codex_build_test.go)

func TestResolveEffectivePheromonesActiveStrongUnexpired(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "s1", Type: "FOCUS", Active: true,
				CreatedAt: now.Format(time.RFC3339),
				Strength:  floatPtr(0.9),
				Content:   json.RawMessage(`{"text": "in effect"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved signal, got %d", len(resolved))
	}
	if !resolved[0].InEffect {
		t.Errorf("expected active, strong, unexpired signal to be in effect; excluded reason: %q", resolved[0].ExcludedReason)
	}
	if resolved[0].EffectiveStrength <= 0 {
		t.Errorf("expected positive effective strength, got %f", resolved[0].EffectiveStrength)
	}
}

func TestResolveEffectivePheromonesActiveStrongExpiredIsExcluded(t *testing.T) {
	// This is the exact gap NOW-09 (extractSignalTexts) had before this file:
	// active + strong but expired must resolve as NOT in effect.
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour).Format(time.RFC3339)
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "expired", Type: "REDIRECT", Active: true,
				CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
				ExpiresAt: &past,
				Strength:  floatPtr(1.0),
				Content:   json.RawMessage(`{"text": "should not apply"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved signal, got %d", len(resolved))
	}
	if resolved[0].InEffect {
		t.Fatal("expected active-strong-expired signal to be excluded, but it was in effect")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedExpired {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedExpired, resolved[0].ExcludedReason)
	}
}

func TestResolveEffectivePheromonesBelowFloorIsExcluded(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "weak", Type: "FEEDBACK", Active: true,
				CreatedAt: now.Format(time.RFC3339),
				Strength:  floatPtr(pheromoneEffectiveFloor - 0.01),
				Content:   json.RawMessage(`{"text": "too weak"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved signal, got %d", len(resolved))
	}
	if resolved[0].InEffect {
		t.Fatal("expected below-floor signal to be excluded")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedBelowFloor {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedBelowFloor, resolved[0].ExcludedReason)
	}
}

func TestResolveEffectivePheromonesMalformedTimestampNeverDefaultsIntoEffect(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "bad-time", Type: "FOCUS", Active: true,
				CreatedAt: "not-a-timestamp",
				Strength:  floatPtr(1.0),
				Content:   json.RawMessage(`{"text": "malformed created_at"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved signal, got %d", len(resolved))
	}
	if resolved[0].InEffect {
		t.Fatal("expected malformed-timestamp signal to be excluded, never defaulted into effect")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedMalformed {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedMalformed, resolved[0].ExcludedReason)
	}
}

func TestResolveEffectivePheromonesNonFiniteStrengthNeverDefaultsIntoEffect(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "bad-strength", Type: "FOCUS", Active: true,
				CreatedAt: now.Format(time.RFC3339),
				Strength:  floatPtr(math.NaN()),
				Content:   json.RawMessage(`{"text": "malformed strength"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved signal, got %d", len(resolved))
	}
	if resolved[0].InEffect {
		t.Fatal("expected non-finite-strength signal to be excluded, never defaulted into effect")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedMalformed {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedMalformed, resolved[0].ExcludedReason)
	}
}

func TestResolveEffectivePheromonesInactiveIsExcluded(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID: "off", Type: "FOCUS", Active: false,
				CreatedAt: now.Format(time.RFC3339),
				Strength:  floatPtr(1.0),
				Content:   json.RawMessage(`{"text": "inactive"}`),
			},
		},
	}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 || resolved[0].InEffect {
		t.Fatal("expected inactive signal to be excluded")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedInactive {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedInactive, resolved[0].ExcludedReason)
	}
}

func TestResolveEffectivePheromonesTieBreaksOnIdentifierAcrossRepeatedCalls(t *testing.T) {
	now := time.Now().UTC()
	pf := &colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "zzz", Type: "REDIRECT", Active: true, CreatedAt: now.Format(time.RFC3339), Strength: floatPtr(1.0), Content: json.RawMessage(`{"text": "z"}`)},
			{ID: "aaa", Type: "REDIRECT", Active: true, CreatedAt: now.Format(time.RFC3339), Strength: floatPtr(1.0), Content: json.RawMessage(`{"text": "a"}`)},
			{ID: "mmm", Type: "REDIRECT", Active: true, CreatedAt: now.Format(time.RFC3339), Strength: floatPtr(1.0), Content: json.RawMessage(`{"text": "m"}`)},
		},
	}

	for i := 0; i < 10; i++ {
		resolved := resolveEffectivePheromones(pf, now)
		if len(resolved) != 3 {
			t.Fatalf("call %d: expected 3 resolved signals, got %d", i, len(resolved))
		}
		got := []string{resolved[0].Signal.ID, resolved[1].Signal.ID, resolved[2].Signal.ID}
		want := []string{"aaa", "mmm", "zzz"}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("call %d: order = %v, want %v (identifier ascending tie-break)", i, got, want)
			}
		}
	}
}

func TestResolveEffectivePheromonesNilAndEmpty(t *testing.T) {
	if resolveEffectivePheromones(nil, time.Now()) != nil {
		t.Error("expected nil for nil PheromoneFile")
	}
	if resolveEffectivePheromones(&colony.PheromoneFile{}, time.Now()) != nil {
		t.Error("expected nil for empty PheromoneFile")
	}
}

// TestActiveStrongExpiredExcludedByEveryReader is the named regression test
// this plan exists to close: extractSignalTexts previously lacked the
// expiry check filterSignalsForPrompt already had (NOW-09 vs NOW-10). All
// three readers must now agree, via the one resolver.
func TestActiveStrongExpiredExcludedByEveryReader(t *testing.T) {
	now := time.Now().UTC()
	past := now.Add(-1 * time.Hour).Format(time.RFC3339)
	signal := colony.PheromoneSignal{
		ID: "expired-redirect", Type: "REDIRECT", Active: true,
		CreatedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
		ExpiresAt: &past,
		Strength:  floatPtr(1.0),
		Content:   json.RawMessage(`{"text": "expired but strong"}`),
	}
	pf := &colony.PheromoneFile{Signals: []colony.PheromoneSignal{signal}}

	if got := filterSignalsForPrompt(pf.Signals, now); len(got) != 0 {
		t.Errorf("filterSignalsForPrompt: expected 0 signals, got %d", len(got))
	}
	if got := extractSignalTextsFrom(pf, 8); len(got) != 0 {
		t.Errorf("extractSignalTextsFrom: expected 0 texts, got %v", got)
	}
	if signalActiveForPrompt(signal, now) {
		t.Error("signalActiveForPrompt: expected false for an expired signal")
	}

	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer func() { _ = tmpDir }()
	store = s
	if err := s.SaveJSON("pheromones.json", *pf); err != nil {
		t.Fatalf("save pheromones.json: %v", err)
	}
	if got := extractSignalTexts(8); len(got) != 0 {
		t.Errorf("extractSignalTexts: expected 0 texts, got %v", got)
	}
}

// --- Task 2: record where every note came from, quarantine imports ---

func TestLegacySignalReadsUnknownProvenanceAndNotQuarantined(t *testing.T) {
	// A fixture written before Provenance/Quarantined existed: raw JSON with
	// no such keys. Unmarshalling into today's struct must leave both nil,
	// and the resolver must read that as "unknown", "not quarantined".
	raw := `{"signals":[{"id":"legacy1","type":"FOCUS","priority":"normal","source":"cli","created_at":"2026-01-01T00:00:00Z","active":true,"strength":0.9,"content":{"text":"pre-existing signal"}}]}`
	var pf colony.PheromoneFile
	if err := json.Unmarshal([]byte(raw), &pf); err != nil {
		t.Fatalf("unmarshal legacy fixture: %v", err)
	}
	if len(pf.Signals) != 1 {
		t.Fatalf("expected 1 legacy signal, got %d", len(pf.Signals))
	}
	sig := pf.Signals[0]
	if sig.Provenance != nil {
		t.Errorf("expected nil Provenance on a legacy fixture, got %v", *sig.Provenance)
	}
	if sig.Quarantined != nil {
		t.Errorf("expected nil Quarantined on a legacy fixture, got %v", *sig.Quarantined)
	}
	if got := pheromoneSignalProvenance(sig); got != colony.PheromoneProvenanceUnknown {
		t.Errorf("pheromoneSignalProvenance = %q, want %q", got, colony.PheromoneProvenanceUnknown)
	}
	if pheromoneSignalQuarantined(sig) {
		t.Error("expected legacy signal to read as not quarantined")
	}
}

func TestImportedNoteIsQuarantinedAndExcludedFromWorkerBrief(t *testing.T) {
	now := time.Now().UTC()
	imported := colony.PheromoneSignal{
		ID: "imported1", Type: "FOCUS", Active: true,
		CreatedAt:   now.Format(time.RFC3339),
		Strength:    floatPtr(1.0),
		Provenance:  strPtr(colony.PheromoneProvenanceImport),
		Quarantined: boolPtr(true),
		Content:     json.RawMessage(`{"text": "imported from another project"}`),
	}
	pf := &colony.PheromoneFile{Signals: []colony.PheromoneSignal{imported}}

	resolved := resolveEffectivePheromones(pf, now)
	if len(resolved) != 1 {
		t.Fatalf("expected the quarantined note to still appear in resolveEffectivePheromones's output, got %d entries", len(resolved))
	}
	if resolved[0].InEffect {
		t.Fatal("expected quarantined note to be excluded from effect")
	}
	if resolved[0].ExcludedReason != pheromoneExcludedQuarantine {
		t.Errorf("expected exclusion reason %q, got %q", pheromoneExcludedQuarantine, resolved[0].ExcludedReason)
	}
	if resolved[0].Provenance != colony.PheromoneProvenanceImport {
		t.Errorf("expected Provenance %q on the resolved entry, got %q", colony.PheromoneProvenanceImport, resolved[0].Provenance)
	}

	if got := extractSignalTextsFrom(pf, 8); len(got) != 0 {
		t.Errorf("extractSignalTextsFrom: expected quarantined note excluded from worker brief text, got %v", got)
	}
}

func boolPtr(b bool) *bool { return &b }

func TestWritePheromoneSignalStampsProvenanceAndQuarantinesImports(t *testing.T) {
	saveGlobalsCmd(t)
	s, tmpDir := newTestStoreCmd(t)
	defer func() { _ = tmpDir }()
	store = s

	owner, _, err := writePheromoneSignal("FOCUS", "owner note", "", "cli", "", "", 0, nil)
	if err != nil {
		t.Fatalf("writePheromoneSignal (cli/owner): %v", err)
	}
	if owner.Provenance == nil || *owner.Provenance != colony.PheromoneProvenanceOwner {
		t.Errorf("expected provenance %q for source=cli, got %v", colony.PheromoneProvenanceOwner, owner.Provenance)
	}
	if owner.Quarantined != nil && *owner.Quarantined {
		t.Error("expected an owner-provenance note to not be quarantined")
	}

	runtimeSig, _, err := writePheromoneSignal("FEEDBACK", "runtime note", "", "aether continue", "", "", 0, nil)
	if err != nil {
		t.Fatalf("writePheromoneSignal (runtime): %v", err)
	}
	if runtimeSig.Provenance == nil || *runtimeSig.Provenance != colony.PheromoneProvenanceRuntime {
		t.Errorf("expected provenance %q for source='aether continue', got %v", colony.PheromoneProvenanceRuntime, runtimeSig.Provenance)
	}

	imported, _, err := writePheromoneSignal("FOCUS", "imported note", "", "import", "", "", 0, nil)
	if err != nil {
		t.Fatalf("writePheromoneSignal (import): %v", err)
	}
	if imported.Provenance == nil || *imported.Provenance != colony.PheromoneProvenanceImport {
		t.Errorf("expected provenance %q for source=import, got %v", colony.PheromoneProvenanceImport, imported.Provenance)
	}
	if imported.Quarantined == nil || !*imported.Quarantined {
		t.Error("expected an import-provenance note to be quarantined")
	}
}
