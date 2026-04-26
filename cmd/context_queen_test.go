package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadQUEENMdExtractsPhilosophies(t *testing.T) {
	content := `# QUEEN.md
## Wisdom
- Wisdom entry here

## Patterns
- Pattern entry here

## Philosophies
- Ship fast and learn
- Test everything that matters

## Anti-Patterns
- Never skip tests
`
	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if _, ok := result["Ship fast and learn"]; !ok {
		t.Fatal("Philosophies entry 'Ship fast and learn' not extracted")
	}
	if _, ok := result["Test everything that matters"]; !ok {
		t.Fatal("Philosophies entry 'Test everything that matters' not extracted")
	}
}

func TestReadQUEENMdExtractsAntiPatterns(t *testing.T) {
	content := `# QUEEN.md
## Wisdom
- Wisdom entry here

## Anti-Patterns
- Never skip tests
- Don't hardcode secrets
`
	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if _, ok := result["Never skip tests"]; !ok {
		t.Fatal("Anti-Patterns entry 'Never skip tests' not extracted")
	}
	if _, ok := result["Don't hardcode secrets"]; !ok {
		t.Fatal("Anti-Patterns entry 'Don't hardcode secrets' not extracted")
	}
}

func TestReadQUEENMdStillExtractsWisdomAndPatterns(t *testing.T) {
	content := `# QUEEN.md
## Wisdom
- Wisdom entry here

## Patterns
- Pattern entry here
`
	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if _, ok := result["Wisdom entry here"]; !ok {
		t.Fatal("Wisdom entry not extracted (regression)")
	}
	if _, ok := result["Pattern entry here"]; !ok {
		t.Fatal("Patterns entry not extracted (regression)")
	}
}

func TestReadQUEENMdDoesNotExtractOtherSections(t *testing.T) {
	content := `# QUEEN.md
## Wisdom
- Wisdom entry

## User Preferences
- Some preference

## Colony Charter
- Charter item
`
	tmpFile := filepath.Join(t.TempDir(), "QUEEN.md")
	os.WriteFile(tmpFile, []byte(content), 0644)

	result := readQUEENMd(tmpFile)
	if _, ok := result["Some preference"]; ok {
		t.Fatal("User Preferences entry should not be extracted by readQUEENMd")
	}
	if _, ok := result["Charter item"]; ok {
		t.Fatal("Colony Charter entry should not be extracted by readQUEENMd")
	}
}
