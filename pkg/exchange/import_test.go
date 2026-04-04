package exchange

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// loadTestFile reads a test data file relative to the package directory.
func loadTestFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", path))
	if err != nil {
		// Try relative to package root
		data, err = os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read test file %q: %v", path, err)
		}
	}
	return data
}

// TestImportWisdomFromShellXML loads the testdata queen-wisdom.xml and verifies entries.
func TestImportWisdomFromShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "testdata/queen-wisdom.xml")

	entries, err := ImportWisdom(xmlData)
	if err != nil {
		t.Fatalf("ImportWisdom failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (1 philosophy + 1 pattern), got %d", len(entries))
	}

	// Verify philosophy
	philFound := false
	for _, e := range entries {
		if e.Category == "philosophy" {
			philFound = true
			if e.ID != "phil_001" {
				t.Errorf("philosophy.ID = %q, want %q", e.ID, "phil_001")
			}
			if e.Confidence != 0.85 {
				t.Errorf("philosophy.Confidence = %f, want 0.85", e.Confidence)
			}
			if e.Domain != "testing" {
				t.Errorf("philosophy.Domain = %q, want %q", e.Domain, "testing")
			}
			if e.Content != "Always write tests first" {
				t.Errorf("philosophy.Content = %q, want %q", e.Content, "Always write tests first")
			}
		}
	}
	if !philFound {
		t.Error("no philosophy entry found")
	}

	// Verify pattern
	patFound := false
	for _, e := range entries {
		if e.Category == "pattern" {
			patFound = true
			if e.ID != "pat_001" {
				t.Errorf("pattern.ID = %q, want %q", e.ID, "pat_001")
			}
			if e.Confidence != 0.70 {
				t.Errorf("pattern.Confidence = %f, want 0.70", e.Confidence)
			}
			if e.Domain != "architecture" {
				t.Errorf("pattern.Domain = %q, want %q", e.Domain, "architecture")
			}
			if e.Content != "Prefer composition over inheritance" {
				t.Errorf("pattern.Content = %q, want %q", e.Content, "Prefer composition over inheritance")
			}
		}
	}
	if !patFound {
		t.Error("no pattern entry found")
	}
}

// TestImportWisdomFromRealShellXML tests importing the actual shell-produced wisdom XML.
func TestImportWisdomFromRealShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "../../.aether/exchange/queen-wisdom.xml")

	entries, err := ImportWisdom(xmlData)
	if err != nil {
		t.Fatalf("ImportWisdom failed on real shell XML: %v", err)
	}

	// The real file has empty philosophies and patterns sections
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries from real shell wisdom XML, got %d", len(entries))
	}
}

// TestImportRegistryFromShellXML loads the testdata colony-registry.xml and verifies colonies.
func TestImportRegistryFromShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "testdata/colony-registry.xml")

	entries, err := ImportRegistry(xmlData)
	if err != nil {
		t.Fatalf("ImportRegistry failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 colonies, got %d", len(entries))
	}

	// Verify alpha colony with lineage
	if entries[0].ID != "colony_alpha" {
		t.Errorf("colony[0].ID = %q, want %q", entries[0].ID, "colony_alpha")
	}
	if entries[0].Name != "Alpha Colony" {
		t.Errorf("colony[0].Name = %q, want %q", entries[0].Name, "Alpha Colony")
	}
	if entries[0].Status != "active" {
		t.Errorf("colony[0].Status = %q, want %q", entries[0].Status, "active")
	}
	if entries[0].ParentID != "colony_parent" {
		t.Errorf("colony[0].ParentID = %q, want %q", entries[0].ParentID, "colony_parent")
	}
	if len(entries[0].Ancestors) != 2 {
		t.Fatalf("colony[0].Ancestors length = %d, want 2", len(entries[0].Ancestors))
	}
	if entries[0].Ancestors[0].ID != "colony_grandparent" {
		t.Errorf("ancestor[0].ID = %q, want %q", entries[0].Ancestors[0].ID, "colony_grandparent")
	}
	if entries[0].Ancestors[0].Depth != 2 {
		t.Errorf("ancestor[0].Depth = %d, want 2", entries[0].Ancestors[0].Depth)
	}
	if entries[0].Ancestors[1].ID != "colony_parent" {
		t.Errorf("ancestor[1].ID = %q, want %q", entries[0].Ancestors[1].ID, "colony_parent")
	}
	if entries[0].Ancestors[1].Depth != 1 {
		t.Errorf("ancestor[1].Depth = %d, want 1", entries[0].Ancestors[1].Depth)
	}

	// Verify beta colony (no lineage)
	if entries[1].ID != "colony_beta" {
		t.Errorf("colony[1].ID = %q, want %q", entries[1].ID, "colony_beta")
	}
	if entries[1].Name != "Beta Colony" {
		t.Errorf("colony[1].Name = %q, want %q", entries[1].Name, "Beta Colony")
	}
	if entries[1].Status != "sealed" {
		t.Errorf("colony[1].Status = %q, want %q", entries[1].Status, "sealed")
	}
}

// TestImportRegistryFromRealShellXML tests importing the actual shell-produced registry XML.
func TestImportRegistryFromRealShellXML(t *testing.T) {
	xmlData := loadTestFile(t, "../../.aether/exchange/colony-registry.xml")

	entries, err := ImportRegistry(xmlData)
	if err != nil {
		t.Fatalf("ImportRegistry failed on real shell XML: %v", err)
	}

	// Registry may be empty (no colonies registered yet) — just verify parsing succeeds.
	// If colonies exist, verify they have required fields.
	for i, e := range entries {
		if e.Name == "" {
			t.Errorf("colony[%d].Name is empty", i)
		}
	}
}

// TestImportWisdomMalformed verifies that broken XML returns an error.
func TestImportWisdomMalformed(t *testing.T) {
	_, err := ImportWisdom([]byte("not xml"))
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

// TestImportRegistryMalformed verifies that broken XML returns an error.
func TestImportRegistryMalformed(t *testing.T) {
	_, err := ImportRegistry([]byte("not xml"))
	if err == nil {
		t.Fatal("expected error for malformed XML, got nil")
	}
}

// TestImportPheromonesContentExtraction verifies content is extracted from
// the nested <content><text>...</text></content> format into json.RawMessage.
func TestImportPheromonesContentExtraction(t *testing.T) {
	xmlStr := `<?xml version="1.0" encoding="UTF-8"?>
<pheromones version="1.0" count="1">
  <signal id="sig_001" type="FOCUS" priority="normal" source="user" created_at="2026-04-01T10:00:00Z" active="true">
    <content><text>Hello world</text></content>
  </signal>
</pheromones>`

	signals, err := ImportPheromones([]byte(xmlStr))
	if err != nil {
		t.Fatalf("ImportPheromones failed: %v", err)
	}

	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}

	var content map[string]string
	if err := json.Unmarshal(signals[0].Content, &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if content["text"] != "Hello world" {
		t.Errorf("content.text = %q, want %q", content["text"], "Hello world")
	}
}
