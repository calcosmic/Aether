# Spike Correlation Reports

This directory contains download spike detection reports for the `aether-colony` npm package.
Reports are generated automatically by the `correlation-pipeline.yml` GitHub Actions workflow.

## What Is This?

The correlation pipeline fetches daily npm download data, detects unusual spikes in activity,
and correlates those spikes against GitHub releases to determine whether a release likely
caused the download surge.

`latest.json` always contains the most recent run. The file is overwritten on each pipeline
execution.

---

## Spike Detection

A day is flagged as a spike when its download count exceeds:

```
threshold = (7-day moving average * 2) + 20
```

The absolute minimum of 20 ensures low-traffic days (e.g. 1-2 downloads/day) are not
trivially flagged by a 2x multiplier alone. The 7-day moving average is the mean of
the 7 calendar days immediately preceding the candidate day.

---

## Release Correlations

Once spikes are identified, each spike is matched against GitHub releases using a
[-2, +7] day window:

- **Before window:** 2 days before the spike date (pre-release announcement effect)
- **After window:** 7 days after the spike date (post-release adoption tail)

A release is considered correlated if its published date falls within that window.

### Proximity Scoring

Each matched release is assigned a proximity score between 0 and 1:

- A release on the same day as the spike scores **1.0**
- Releases further away score proportionally lower
- The closer the release is to the spike, the higher the score

Multiple releases may correlate to a single spike; all are included with their
individual scores.

---

## JSON Schema

```jsonc
{
  "generated_at": "ISO 8601 timestamp of when this report was generated",
  "package": "npm package name",
  "period": {
    "start": "ISO 8601 date — earliest day included in the analysis window",
    "end":   "ISO 8601 date — latest day included in the analysis window"
  },
  "spikes": [
    {
      "date":      "ISO 8601 date of the spike",
      "downloads": 1234,         // actual download count on that day
      "baseline":  456,          // 7-day moving average used as the baseline
      "ratio":     2.71          // downloads / baseline
    }
  ],
  "correlations": [
    {
      "spike_date":        "ISO 8601 date of the correlated spike",
      "release_tag":       "v1.2.3",
      "release_date":      "ISO 8601 date the release was published",
      "days_offset":       -1,   // release_date - spike_date in days (negative = before spike)
      "proximity_score":   0.88  // 0.0–1.0; higher = closer to spike
    }
  ],
  "metadata": {
    "spike_threshold":           "human-readable description of the threshold formula",
    "correlation_window_days": {
      "before_release": 2,       // days before spike to search for releases
      "after_release":  7        // days after spike to search for releases
    },
    "proximity_scoring": "closer to release = higher score"
  }
}
```

---

## Data Source

- **Downloads:** [npm download counts API](https://api.npmjs.org/downloads/range/) — daily granularity
- **Releases:** GitHub Releases API for `calcosmic/Aether`
- **Updated by:** `.github/workflows/correlation-pipeline.yml`
