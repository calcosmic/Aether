package memory

import (
	"math"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// InstinctApplicationSummary captures the usable application history for an instinct.
type InstinctApplicationSummary struct {
	Applications int
	Successes    int
	Failures     int
	SuccessRate  float64
	LastApplied  string
}

// SummarizeInstinctApplications folds legacy provenance counters and explicit
// application history into one consistent summary. Older colonies only tracked
// application_count, so missing history entries are treated as successful uses.
//
// Each ApplicationHistory entry (colony.InstinctApplicationEntry,
// SYN-204-05/06) is read in whichever of its two shapes it carries: the new
// shape's Outcome (helpful counts a success, harmful counts a failure,
// neutral/pending/empty count as neither -- an entry with no verified
// outcome is not silently treated as a success), or -- only when Outcome is
// empty -- the old shape's LegacySuccess boolean (true/false count exactly
// as they did before this change). Applications counts every entry either
// way; only the success/failure tally differs by shape.
func SummarizeInstinctApplications(entry colony.InstinctEntry) InstinctApplicationSummary {
	summary := InstinctApplicationSummary{}
	if entry.Provenance.LastApplied != nil {
		summary.LastApplied = *entry.Provenance.LastApplied
	}

	for _, app := range entry.ApplicationHistory {
		summary.Applications++
		switch {
		case app.Outcome != "":
			switch app.Outcome {
			case "helpful":
				summary.Successes++
			case "harmful":
				summary.Failures++
				// neutral, pending: a verified-but-inconclusive or
				// not-yet-verified outcome counts as neither a success
				// nor a failure.
			}
		case app.LegacySuccess != nil:
			if *app.LegacySuccess {
				summary.Successes++
			} else {
				summary.Failures++
			}
		}
		if app.Timestamp != "" {
			summary.LastApplied = newerTimestamp(summary.LastApplied, app.Timestamp)
		}
	}

	if entry.Provenance.ApplicationCount > summary.Applications {
		delta := entry.Provenance.ApplicationCount - summary.Applications
		summary.Applications = entry.Provenance.ApplicationCount
		summary.Successes += delta
	}

	if summary.Applications > 0 {
		summary.SuccessRate = float64(summary.Successes) / float64(summary.Applications)
	}

	return summary
}

// InstinctReferenceTimestamp returns the freshest timestamp available for an instinct.
func InstinctReferenceTimestamp(entry colony.InstinctEntry) string {
	summary := SummarizeInstinctApplications(entry)
	if summary.LastApplied != "" {
		return summary.LastApplied
	}
	return entry.Provenance.CreatedAt
}

// InstinctUsefulnessScore ranks how useful an instinct currently is for retrieval.
// It blends trust/confidence, freshness, and demonstrated application history.
func InstinctUsefulnessScore(entry colony.InstinctEntry, now time.Time) float64 {
	summary := SummarizeInstinctApplications(entry)
	base := entry.TrustScore
	if entry.Confidence > base {
		base = entry.Confidence
	}

	freshness := freshnessFromTimestamp(InstinctReferenceTimestamp(entry), now, 0.55)
	applications := math.Min(float64(summary.Applications), 5) / 5.0
	successRate := 0.5
	if summary.Applications > 0 {
		successRate = summary.SuccessRate
	}

	score := 0.40*clampUnit(base) +
		0.25*freshness +
		0.20*applications +
		0.15*clampUnit(successRate)

	if summary.Applications > 0 && summary.Failures > summary.Successes {
		score -= 0.10
	}

	return clampUnit(score)
}

// InstinctNeedsReview identifies instincts that are in active use but need scrutiny.
func InstinctNeedsReview(entry colony.InstinctEntry, now time.Time) bool {
	summary := SummarizeInstinctApplications(entry)
	if summary.Applications == 0 {
		return false
	}
	if summary.Failures > summary.Successes {
		return true
	}
	if summary.Applications >= 2 && summary.SuccessRate < 0.50 {
		return true
	}
	return daysSinceTimestamp(InstinctReferenceTimestamp(entry)) > 90 &&
		summary.SuccessRate < 0.70 &&
		InstinctUsefulnessScore(entry, now) < 0.60
}

// InstinctNeedsReread identifies instincts that are stale or unexercised.
func InstinctNeedsReread(entry colony.InstinctEntry, now time.Time) bool {
	summary := SummarizeInstinctApplications(entry)
	days := daysSinceTimestamp(InstinctReferenceTimestamp(entry))
	if summary.Applications == 0 {
		return days >= 45
	}
	return summary.SuccessRate >= 0.50 &&
		days >= 120 &&
		InstinctUsefulnessScore(entry, now) < 0.65
}

func clampUnit(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func freshnessFromTimestamp(ts string, now time.Time, fallback float64) float64 {
	if ts == "" {
		return clampUnit(fallback)
	}
	parsed, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return clampUnit(fallback)
	}
	if parsed.After(now) {
		return 1
	}
	days := now.Sub(parsed).Hours() / 24
	if days < 0 {
		days = 0
	}
	return clampUnit(math.Pow(0.5, days/60.0))
}

func newerTimestamp(current, candidate string) string {
	if candidate == "" {
		return current
	}
	if current == "" {
		return candidate
	}
	currentTime, err := time.Parse(time.RFC3339, current)
	if err != nil {
		return candidate
	}
	candidateTime, err := time.Parse(time.RFC3339, candidate)
	if err != nil {
		return current
	}
	if candidateTime.After(currentTime) {
		return candidate
	}
	return current
}
