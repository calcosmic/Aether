package cmd

import (
	"fmt"
	"math"
	"strings"
)

// Shared layout pieces for the owner's chosen screen shape (2026-09-27): a
// verdict box first, then short sections, each under a heavy spaced-letter
// header with exactly one blank line above it. The check screen
// (renderContinueVisual / renderContinueBlockedVisual) is the first screen
// built from them; the build, status, plan and seal screens can adopt the
// same helpers so every screen reads the same way.

// renderVerdictBox draws the verdict at the very top of a screen: a full
// heavy line, one glyph-led verdict line, one plain sentence, and another
// heavy line. The blank line after it comes from the first section header.
func renderVerdictBox(kind, verdict, sentence string) string {
	var b strings.Builder
	b.WriteString(visualDividerStr())
	b.WriteString("  ")
	b.WriteString(voiceLine(kind, strings.TrimSpace(verdict)))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(strings.TrimSpace(sentence))
	b.WriteString("\n")
	b.WriteString(visualDividerStr())
	return b.String()
}

// renderScreenSection opens one section: a blank line, then the heavy
// spaced-letter header renderBanner already draws for "N E X T   U P".
func renderScreenSection(emoji, title string) string {
	return "\n" + renderBanner(emoji, title)
}

// writeScreenSection writes a section header followed by its lines, each
// indented two spaces. It writes nothing at all when lines is empty, so a
// section only appears when it has something to say.
func writeScreenSection(b *strings.Builder, emoji, title string, lines []string) {
	if len(lines) == 0 {
		return
	}
	b.WriteString(renderScreenSection(emoji, title))
	for _, line := range lines {
		b.WriteString("  ")
		b.WriteString(line)
		b.WriteString("\n")
	}
}

// capScreenLines keeps at most max lines and, when some were left out, adds
// one honest "… N more" line in their place.
func capScreenLines(lines []string, max int) []string {
	if max <= 0 || len(lines) <= max {
		return lines
	}
	out := append([]string(nil), lines[:max]...)
	return append(out, fmt.Sprintf("… %d more", len(lines)-max))
}

// plainDuration renders a measured time in plain units: "14s", "5 min",
// "under 1s". A zero or unmeasured time renders as nothing.
func plainDuration(seconds float64) string {
	switch {
	case seconds <= 0:
		return ""
	case seconds < 1:
		return "under 1s"
	case seconds < 60:
		return fmt.Sprintf("%ds", int(math.Round(seconds)))
	default:
		return fmt.Sprintf("%d min", int(math.Round(seconds/60)))
	}
}

// padScreenLabel pads a short label so the detail after it lines up, the
// way the owner's mockup aligns "Build passed        (14s)".
func padScreenLabel(label string, width int) string {
	if n := len([]rune(label)); n < width {
		return label + strings.Repeat(" ", width-n)
	}
	return label + " "
}

func screenCount(n int, noun string) string {
	if n != 1 {
		noun += "s"
	}
	return fmt.Sprintf("%d %s", n, noun)
}

func screenTextLines(text string) []string {
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
