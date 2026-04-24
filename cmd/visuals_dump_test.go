package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestVisualsDumpExportsCasteIdentityContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"visuals-dump", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("visuals-dump returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		`"builder":{"emoji":"🔨","color":"33","label":"Builder"}`,
		`"watcher":{"emoji":"👁️","color":"36","label":"Watcher"}`,
		`"oracle":{"emoji":"🔮","color":"35","label":"Oracle"}`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("visuals-dump output missing %s\n%s", want, output)
		}
	}
}
