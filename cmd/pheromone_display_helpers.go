package cmd

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func signalDecayWindowDays(signalType string) float64 {
	switch signalType {
	case "FOCUS":
		return 30
	case "REDIRECT":
		return 60
	case "FEEDBACK":
		return 90
	default:
		return 30
	}
}

func signalLifetimeSummary(sig colony.PheromoneSignal, now time.Time) string {
	parts := make([]string, 0, 2)

	if sig.ExpiresAt != nil && strings.TrimSpace(*sig.ExpiresAt) != "" {
		if expiresAt, err := time.Parse(time.RFC3339, *sig.ExpiresAt); err == nil {
			if expiresAt.After(now) {
				parts = append(parts, fmt.Sprintf("ttl %s left", humanizePheromoneDuration(expiresAt.Sub(now))))
			} else {
				parts = append(parts, "expired")
			}
		}
	} else if sig.Type == "FOCUS" {
		parts = append(parts, "phase-scoped")
	}

	remainingDecay := remainingSignalDecay(sig, now)
	parts = append(parts, fmt.Sprintf("%s decay", humanizePheromoneDuration(remainingDecay)))

	return strings.Join(parts, " | ")
}

func remainingSignalDecay(sig colony.PheromoneSignal, now time.Time) time.Duration {
	window := time.Duration(signalDecayWindowDays(sig.Type)*24) * time.Hour

	createdAt, err := time.Parse(time.RFC3339, sig.CreatedAt)
	if err != nil {
		return window
	}

	elapsed := now.Sub(createdAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed >= window {
		return 0
	}
	return window - elapsed
}

func humanizePheromoneDuration(d time.Duration) string {
	if d <= 0 {
		return "0h"
	}

	if d >= 24*time.Hour {
		days := int(math.Ceil(d.Hours() / 24))
		return fmt.Sprintf("%dd", days)
	}

	if d >= time.Hour {
		hours := int(math.Ceil(d.Hours()))
		return fmt.Sprintf("%dh", hours)
	}

	minutes := int(math.Ceil(d.Minutes()))
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("%dm", minutes)
}

// pheromoneSectionHeadings is the classic v5.4.0 pheromone display order and
// framing: each signal type gets an emoji heading with a plain-English
// parenthetical, so the display explains itself.
var pheromoneSectionHeadings = []struct {
	Type    string
	Heading string
}{
	{"FOCUS", "🎯 FOCUS (Pay attention here)"},
	{"REDIRECT", "🚫 REDIRECT (Hard constraints - DO NOT do this)"},
	{"FEEDBACK", "💬 FEEDBACK (Guidance to consider)"},
}

// renderClassicPheromoneSections renders active signals in the classic
// sectioned house style: letter-spaced banner, emoji-headed groups, one
// signal per line as [NN%] "text" with a nested lifetime line, and a decay
// footer.
func renderClassicPheromoneSections(signals []colony.PheromoneSignal, now time.Time) string {
	var b strings.Builder
	rule := strings.Repeat("━", 50)
	b.WriteString(rule + "\n")
	b.WriteString("   A C T I V E   P H E R O M O N E S\n")
	b.WriteString(rule + "\n\n")

	grouped := map[string][]colony.PheromoneSignal{}
	var otherTypes []string
	for _, sig := range signals {
		grouped[sig.Type] = append(grouped[sig.Type], sig)
	}
	known := map[string]bool{"FOCUS": true, "REDIRECT": true, "FEEDBACK": true}
	for _, sig := range signals {
		if !known[sig.Type] && !containsString(otherTypes, sig.Type) {
			otherTypes = append(otherTypes, sig.Type)
		}
	}

	writeGroup := func(heading string, group []colony.PheromoneSignal) {
		if len(group) == 0 {
			return
		}
		b.WriteString(heading + "\n\n")
		for _, sig := range group {
			percent := int(math.Round(computeEffectiveStrength(sig, now) * 100))
			text := strings.TrimSpace(extractText(sig.Content))
			if text == "" {
				text = "(no content)"
			}
			b.WriteString(fmt.Sprintf("   [%d%%] %q\n", percent, text))
			detail := signalAgeSummary(sig, now)
			if life := signalLifetimeSummary(sig, now); life != "" {
				if detail != "" {
					detail += ", "
				}
				detail += life
			}
			if detail != "" {
				b.WriteString("      └── " + detail + "\n")
			}
		}
		b.WriteString("\n")
	}

	for _, section := range pheromoneSectionHeadings {
		writeGroup(section.Heading, grouped[section.Type])
	}
	for _, other := range otherTypes {
		writeGroup("🐜 "+other, grouped[other])
	}

	b.WriteString(rule + "\n")
	b.WriteString(fmt.Sprintf("%d signal(s) active | Decay: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d\n", len(signals)))
	return b.String()
}

// signalAgeSummary renders how long ago a signal was left, in days.
func signalAgeSummary(sig colony.PheromoneSignal, now time.Time) string {
	createdAt, err := time.Parse(time.RFC3339, sig.CreatedAt)
	if err != nil {
		return ""
	}
	days := int(now.Sub(createdAt).Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf("%dd ago", days)
}
