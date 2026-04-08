package exchange

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func strPtr(s string) *string { return &s }

// TestExportPheromonesRoundTrip exports signals then imports them back,
// verifying all fields match.
func TestExportPheromonesRoundTrip(t *testing.T) {
	signals := []colony.PheromoneSignal{
		{
			ID:        "sig_focus_001",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "user",
			CreatedAt: "2026-04-01T10:00:00Z",
			ExpiresAt: strPtr("2026-04-30T10:00:00Z"),
			Active:    true,
			Content:   json.RawMessage(`{"text":"Focus on security patterns"}`),
		},
		{
			ID:        "sig_redirect_002",
			Type:      "REDIRECT",
			Priority:  "high",
			Source:    "system",
			CreatedAt: "2026-04-01T10:01:00Z",
			ExpiresAt: nil,
			Active:    true,
			Content:   json.RawMessage(`{"text":"Avoid using grep without -F flag"}`),
		},
		{
			ID:        "sig_feedback_003",
			Type:      "FEEDBACK",
			Priority:  "low",
			Source:    "builder",
			CreatedAt: "2026-04-01T10:02:00Z",
			ExpiresAt: strPtr("phase_end"),
			Active:    false,
			Content:   json.RawMessage(`{"text":"Prefer table-driven tests"}`),
		},
	}

	data, err := ExportPheromones(signals)
	if err != nil {
		t.Fatalf("ExportPheromones failed: %v", err)
	}

	// Must have XML header
	if !strings.HasPrefix(string(data), "<?xml") {
		t.Errorf("exported data missing XML header")
	}

	// Import back
	imported, err := ImportPheromones(data)
	if err != nil {
		t.Fatalf("ImportPheromones failed: %v", err)
	}

	if len(imported) != len(signals) {
		t.Fatalf("expected %d signals, got %d", len(signals), len(imported))
	}

	for i, got := range imported {
		want := signals[i]
		if got.ID != want.ID {
			t.Errorf("signal[%d].ID = %q, want %q", i, got.ID, want.ID)
		}
		if got.Type != want.Type {
			t.Errorf("signal[%d].Type = %q, want %q", i, got.Type, want.Type)
		}
		if got.Priority != want.Priority {
			t.Errorf("signal[%d].Priority = %q, want %q", i, got.Priority, want.Priority)
		}
		if got.Source != want.Source {
			t.Errorf("signal[%d].Source = %q, want %q", i, got.Source, want.Source)
		}
		if got.CreatedAt != want.CreatedAt {
			t.Errorf("signal[%d].CreatedAt = %q, want %q", i, got.CreatedAt, want.CreatedAt)
		}
		if got.Active != want.Active {
			t.Errorf("signal[%d].Active = %v, want %v", i, got.Active, want.Active)
		}

		// Compare ExpiresAt
		if want.ExpiresAt == nil {
			if got.ExpiresAt != nil && *got.ExpiresAt != "" {
				t.Errorf("signal[%d].ExpiresAt = %q, want nil", i, *got.ExpiresAt)
			}
		} else {
			if got.ExpiresAt == nil {
				t.Errorf("signal[%d].ExpiresAt = nil, want %q", i, *want.ExpiresAt)
			} else if *got.ExpiresAt != *want.ExpiresAt {
				t.Errorf("signal[%d].ExpiresAt = %q, want %q", i, *got.ExpiresAt, *want.ExpiresAt)
			}
		}

		// Compare Content text
		var wantContent map[string]string
		if err := json.Unmarshal(want.Content, &wantContent); err != nil {
			t.Fatalf("unmarshal want content: %v", err)
		}
		var gotContent map[string]string
		if err := json.Unmarshal(got.Content, &gotContent); err != nil {
			t.Fatalf("unmarshal got content: %v", err)
		}
		if gotContent["text"] != wantContent["text"] {
			t.Errorf("signal[%d].Content.text = %q, want %q", i, gotContent["text"], wantContent["text"])
		}
	}
}

// TestExportPheromonesEmpty verifies that an empty slice produces valid XML.
func TestExportPheromonesEmpty(t *testing.T) {
	data, err := ExportPheromones([]colony.PheromoneSignal{})
	if err != nil {
		t.Fatalf("ExportPheromones failed: %v", err)
	}

	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("missing XML header")
	}

	// Should unmarshal cleanly
	imported, err := ImportPheromones(data)
	if err != nil {
		t.Fatalf("ImportPheromones failed: %v", err)
	}
	if len(imported) != 0 {
		t.Errorf("expected 0 signals, got %d", len(imported))
	}
}

// TestExportPheromonesAllTypes verifies FOCUS, REDIRECT, FEEDBACK all export correctly.
func TestExportPheromonesAllTypes(t *testing.T) {
	types := []string{"FOCUS", "REDIRECT", "FEEDBACK"}
	signals := make([]colony.PheromoneSignal, len(types))
	for i, typ := range types {
		signals[i] = colony.PheromoneSignal{
			ID:        "sig_" + typ,
			Type:      typ,
			Priority:  "normal",
			Source:    "test",
			CreatedAt: "2026-04-01T10:00:00Z",
			Active:    true,
			Content:   json.RawMessage(`{"text":"test content"}`),
		}
	}

	data, err := ExportPheromones(signals)
	if err != nil {
		t.Fatalf("ExportPheromones failed: %v", err)
	}

	xmlStr := string(data)
	for _, typ := range types {
		if !strings.Contains(xmlStr, `type="`+typ+`"`) {
			t.Errorf("exported XML missing type=%q attribute", typ)
		}
	}
}

// TestExportPheromonesAttributes verifies XML has correct attribute names.
func TestExportPheromonesAttributes(t *testing.T) {
	signals := []colony.PheromoneSignal{
		{
			ID:        "sig_001",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "user",
			CreatedAt: "2026-04-01T10:00:00Z",
			Active:    true,
			Content:   json.RawMessage(`{"text":"test"}`),
		},
	}

	data, err := ExportPheromones(signals)
	if err != nil {
		t.Fatalf("ExportPheromones failed: %v", err)
	}

	xmlStr := string(data)
	// Check root element
	if !strings.Contains(xmlStr, `<pheromones `) {
		t.Errorf("missing <pheromones> root element")
	}
	// Check signal attributes
	requiredAttrs := []string{
		`id="sig_001"`,
		`type="FOCUS"`,
		`priority="normal"`,
		`source="user"`,
		`created_at="2026-04-01T10:00:00Z"`,
		`active="true"`,
	}
	for _, attr := range requiredAttrs {
		if !strings.Contains(xmlStr, attr) {
			t.Errorf("exported XML missing attribute: %s", attr)
		}
	}
}

// TestImportPheromonesFromShellXML loads the testdata pheromones.xml file
// and verifies the signal data matches expectations.
func TestImportPheromonesFromShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "testdata/pheromones.xml")

	signals, err := ImportPheromones(xmlData)
	if err != nil {
		t.Fatalf("ImportPheromones failed: %v", err)
	}

	if len(signals) != 3 {
		t.Fatalf("expected 3 signals, got %d", len(signals))
	}

	// Verify first signal
	if signals[0].ID != "sig_focus_001" {
		t.Errorf("signal[0].ID = %q, want %q", signals[0].ID, "sig_focus_001")
	}
	if signals[0].Type != "FOCUS" {
		t.Errorf("signal[0].Type = %q, want %q", signals[0].Type, "FOCUS")
	}
	if signals[0].Priority != "normal" {
		t.Errorf("signal[0].Priority = %q, want %q", signals[0].Priority, "normal")
	}
	if signals[0].Source != "user" {
		t.Errorf("signal[0].Source = %q, want %q", signals[0].Source, "user")
	}
	if !signals[0].Active {
		t.Errorf("signal[0].Active = false, want true")
	}

	// Check content
	var content map[string]string
	if err := json.Unmarshal(signals[0].Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if content["text"] != "Focus on security patterns" {
		t.Errorf("signal[0].content.text = %q, want %q", content["text"], "Focus on security patterns")
	}

	// Verify second signal (REDIRECT)
	if signals[1].Type != "REDIRECT" {
		t.Errorf("signal[1].Type = %q, want %q", signals[1].Type, "REDIRECT")
	}
	if signals[1].Priority != "high" {
		t.Errorf("signal[1].Priority = %q, want %q", signals[1].Priority, "high")
	}

	// Verify third signal (FEEDBACK)
	if signals[2].Type != "FEEDBACK" {
		t.Errorf("signal[2].Type = %q, want %q", signals[2].Type, "FEEDBACK")
	}
}

// TestImportPheromonesMalformed verifies that broken XML returns an error.
func TestImportPheromonesMalformed(t *testing.T) {
	_, err := ImportPheromones([]byte("this is not xml at all"))
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

// TestImportPheromonesFromRealShellXML tests importing the actual shell-produced XML.
func TestImportPheromonesFromRealShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "../../.aether/exchange/pheromones.xml")

	signals, err := ImportPheromones(xmlData)
	if err != nil {
		t.Fatalf("ImportPheromones failed on real shell XML: %v", err)
	}

	if len(signals) == 0 {
		t.Fatal("expected at least 1 signal from real shell XML")
	}

	// Verify the first signal has required fields populated
	s := signals[0]
	if s.ID == "" {
		t.Error("signal.ID is empty")
	}
	if s.Type == "" {
		t.Error("signal.Type is empty")
	}
	if s.Priority == "" {
		t.Error("signal.Priority is empty")
	}
}

// TestExportPheromonesXMLEncoding verifies the XML is well-formed.
func TestExportPheromonesXMLEncoding(t *testing.T) {
	signals := []colony.PheromoneSignal{
		{
			ID:        "sig_001",
			Type:      "FOCUS",
			Priority:  "normal",
			Source:    "user",
			CreatedAt: "2026-04-01T10:00:00Z",
			Active:    true,
			Content:   json.RawMessage(`{"text":"Content with <special> & chars"}`),
		},
	}

	data, err := ExportPheromones(signals)
	if err != nil {
		t.Fatalf("ExportPheromones failed: %v", err)
	}

	// Must be valid XML
	var phXML PheromoneXML
	if err := xml.Unmarshal(data, &phXML); err != nil {
		t.Fatalf("exported XML is not valid: %v", err)
	}

	// Content should be properly escaped
	xmlStr := string(data)
	if !strings.Contains(xmlStr, "special") {
		t.Error("content text not found in XML output")
	}
}

// TestExportWisdomRoundTrip tests wisdom export then import round-trip.
func TestExportWisdomRoundTrip(t *testing.T) {
	entries := []WisdomEntry{
		{
			ID:         "phil_001",
			Category:   "philosophy",
			Confidence: 0.85,
			Domain:     "testing",
			Source:     "instinct",
			CreatedAt:  "2026-04-01T12:00:00Z",
			Content:    "Always write tests first",
		},
		{
			ID:         "pat_001",
			Category:   "pattern",
			Confidence: 0.70,
			Domain:     "architecture",
			Source:     "observation",
			CreatedAt:  "2026-04-01T12:01:00Z",
			Content:    "Prefer composition over inheritance",
		},
	}

	data, err := ExportWisdom(entries, 0.0, "test-colony")
	if err != nil {
		t.Fatalf("ExportWisdom failed: %v", err)
	}

	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("missing XML header")
	}

	imported, err := ImportWisdom(data)
	if err != nil {
		t.Fatalf("ImportWisdom failed: %v", err)
	}

	if len(imported) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(imported))
	}

	if imported[0].Category != "philosophy" {
		t.Errorf("entry[0].Category = %q, want philosophy", imported[0].Category)
	}
	if imported[0].Confidence != 0.85 {
		t.Errorf("entry[0].Confidence = %f, want 0.85", imported[0].Confidence)
	}
	if imported[0].Content != "Always write tests first" {
		t.Errorf("entry[0].Content = %q, want %q", imported[0].Content, "Always write tests first")
	}

	if imported[1].Category != "pattern" {
		t.Errorf("entry[1].Category = %q, want pattern", imported[1].Category)
	}
}

// TestExportWisdomFiltersByConfidence tests that entries below threshold are excluded.
func TestExportWisdomFiltersByConfidence(t *testing.T) {
	entries := []WisdomEntry{
		{ID: "phil_001", Category: "philosophy", Confidence: 0.90, Domain: "test", Content: "high"},
		{ID: "phil_002", Category: "philosophy", Confidence: 0.50, Domain: "test", Content: "low"},
		{ID: "pat_001", Category: "pattern", Confidence: 0.80, Domain: "test", Content: "mid"},
	}

	data, err := ExportWisdom(entries, 0.75, "test-colony")
	if err != nil {
		t.Fatalf("ExportWisdom failed: %v", err)
	}

	imported, err := ImportWisdom(data)
	if err != nil {
		t.Fatalf("ImportWisdom failed: %v", err)
	}

	// Only 0.90 and 0.80 should pass
	if len(imported) != 2 {
		t.Fatalf("expected 2 entries (confidence >= 0.75), got %d", len(imported))
	}

	for _, e := range imported {
		if e.Confidence < 0.75 {
			t.Errorf("entry %s has confidence %f, should have been filtered", e.ID, e.Confidence)
		}
	}
}

// TestExportRegistryRoundTrip tests registry export then import round-trip.
func TestExportRegistryRoundTrip(t *testing.T) {
	entries := []ColonyEntry{
		{
			ID:        "colony_alpha",
			Name:      "Alpha Colony",
			Status:    "active",
			CreatedAt: "2026-04-01T10:00:00Z",
			ParentID:  "colony_parent",
			Ancestors: []Ancestor{
				{ID: "colony_grandparent", Depth: 2},
				{ID: "colony_parent", Depth: 1},
			},
		},
		{
			ID:        "colony_beta",
			Name:      "Beta Colony",
			Status:    "sealed",
			CreatedAt: "2026-04-01T11:00:00Z",
		},
	}

	data, err := ExportRegistry(entries)
	if err != nil {
		t.Fatalf("ExportRegistry failed: %v", err)
	}

	if !strings.HasPrefix(string(data), "<?xml") {
		t.Error("missing XML header")
	}

	imported, err := ImportRegistry(data)
	if err != nil {
		t.Fatalf("ImportRegistry failed: %v", err)
	}

	if len(imported) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(imported))
	}

	// Verify alpha colony with lineage
	if imported[0].ID != "colony_alpha" {
		t.Errorf("entry[0].ID = %q, want %q", imported[0].ID, "colony_alpha")
	}
	if imported[0].ParentID != "colony_parent" {
		t.Errorf("entry[0].ParentID = %q, want %q", imported[0].ParentID, "colony_parent")
	}
	if len(imported[0].Ancestors) != 2 {
		t.Fatalf("entry[0].Ancestors length = %d, want 2", len(imported[0].Ancestors))
	}
	if imported[0].Ancestors[0].ID != "colony_grandparent" {
		t.Errorf("ancestor[0].ID = %q, want %q", imported[0].Ancestors[0].ID, "colony_grandparent")
	}
	if imported[0].Ancestors[0].Depth != 2 {
		t.Errorf("ancestor[0].Depth = %d, want 2", imported[0].Ancestors[0].Depth)
	}

	// Verify beta colony (no lineage)
	if imported[1].ID != "colony_beta" {
		t.Errorf("entry[1].ID = %q, want %q", imported[1].ID, "colony_beta")
	}
	if len(imported[1].Ancestors) != 0 {
		t.Errorf("entry[1].Ancestors = %v, want empty", imported[1].Ancestors)
	}
}
