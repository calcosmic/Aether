package cmd

import (
	"bytes"
	"fmt"
	"testing"
)

func TestFilteredPrintln_VerboseTrue_WritesToStdout(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(true)
	filteredPrintln("hello world")

	got := buf.String()
	if got == "" {
		t.Fatal("expected output when verbose=true, got empty buffer")
	}
	if got != "hello world\n" {
		t.Fatalf("expected %q, got %q", "hello world\n", got)
	}
}

func TestFilteredPrintln_VerboseFalse_DiscardsOutput(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(false)
	filteredPrintln("should not appear")

	if buf.String() != "" {
		t.Fatalf("expected empty buffer when verbose=false, got %q", buf.String())
	}
}

func TestFilteredFprintf_VerboseTrue_WritesFormattedMessage(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(true)
	filteredFprintf("hello %s, count=%d", "world", 42)

	got := buf.String()
	if got == "" {
		t.Fatal("expected output when verbose=true, got empty buffer")
	}
	if got != "hello world, count=42" {
		t.Fatalf("expected %q, got %q", "hello world, count=42", got)
	}
}

func TestFilteredFprintf_VerboseFalse_DiscardsOutput(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(false)
	filteredFprintf("should not appear %d", 1)

	if buf.String() != "" {
		t.Fatalf("expected empty buffer when verbose=false, got %q", buf.String())
	}
}

func TestFilteredFprintln_VerboseTrue_WritesToWriter(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(true)
	filteredFprintln(stdout, "writer output")

	got := buf.String()
	if got == "" {
		t.Fatal("expected output when verbose=true, got empty buffer")
	}
	if got != "writer output\n" {
		t.Fatalf("expected %q, got %q", "writer output\n", got)
	}
}

func TestFilteredFprintln_VerboseFalse_DiscardsOutput(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(false)
	filteredFprintln(stdout, "should not appear")

	if buf.String() != "" {
		t.Fatalf("expected empty buffer when verbose=false, got %q", buf.String())
	}
}

func TestSetBuildVerbose_TogglesState(t *testing.T) {
	setBuildVerbose(true)
	if !buildVerbose {
		t.Fatal("expected buildVerbose=true after setBuildVerbose(true)")
	}

	setBuildVerbose(false)
	if buildVerbose {
		t.Fatal("expected buildVerbose=false after setBuildVerbose(false)")
	}
}

func TestFilteredPrintln_MultipleArgs(t *testing.T) {
	originalStdout := stdout
	defer func() { stdout = originalStdout }()

	buf := &bytes.Buffer{}
	stdout = buf

	setBuildVerbose(true)
	filteredPrintln("one", "two", 3)

	got := buf.String()
	expected := fmt.Sprintln("one", "two", 3)
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
