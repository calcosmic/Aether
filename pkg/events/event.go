// Package events provides a typed event bus with in-memory pub/sub via Go
// channels, crash-recoverable JSONL persistence, TTL pruning, and topic-based
// subscription. It replaces the shell's file-based pub/sub in event-bus.sh.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DefaultTTL is the default time-to-live for events in days.
// Matches the shell _EVENT_BUS_DEFAULT_TTL=30.
const DefaultTTL = 30

// DefaultLimit is the default maximum number of events returned by queries.
// Matches the shell _EVENT_BUS_DEFAULT_LIMIT=50.
const DefaultLimit = 50

// Event represents a single event on the bus.
// JSON field names match the shell event-bus.sh output exactly.
type Event struct {
	ID        string          `json:"id"`
	Topic     string          `json:"topic"`
	Payload   json.RawMessage `json:"payload"`
	Source    string          `json:"source"`
	Timestamp string          `json:"timestamp"`
	TTLDays   int             `json:"ttl_days"`
	ExpiresAt string          `json:"expires_at"`
}

// Config holds bus configuration.
type Config struct {
	// DefaultTTL is the TTL in days for events when not specified. Defaults to 30.
	DefaultTTL int
	// DefaultLimit is the maximum number of events returned by Query/Replay. Defaults to 50.
	DefaultLimit int
	// JSONLFile is the filename for event persistence, relative to the store's base path.
	// Defaults to "event-bus.jsonl".
	JSONLFile string
}

// DefaultConfig returns a Config with sensible defaults matching the shell behavior.
func DefaultConfig() Config {
	return Config{
		DefaultTTL:   DefaultTTL,
		DefaultLimit: DefaultLimit,
		JSONLFile:    "event-bus.jsonl",
	}
}

// TopicMatch checks if a topic matches a pattern.
// Patterns ending with "*" are treated as prefix matches (e.g., "learning.*"
// matches "learning.observe", "learning.promote").
// All other patterns are exact matches.
func TopicMatch(pattern, topic string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := pattern[:len(pattern)-1]
		return strings.HasPrefix(topic, prefix)
	}
	return pattern == topic
}

// GenerateEventID creates an event ID matching the shell format:
//   evt_{unix_timestamp}_{4_hex_chars}
//
// Shell: evt_$(date +%s)_$(head -c 2 /dev/urandom | od -An -tx1 | tr -d ' \n')
func GenerateEventID(now time.Time, randomBytes []byte) string {
	unix := now.Unix()
	// Shell reads 2 bytes, converts to hex (4 chars), strips spaces
	hex := fmt.Sprintf("%x", randomBytes)
	if len(hex) > 4 {
		hex = hex[:4]
	}
	// Pad to at least 4 chars if needed
	for len(hex) < 4 {
		hex = "0" + hex
	}
	return fmt.Sprintf("evt_%d_%s", unix, hex)
}

// FormatTimestamp formats a time as ISO-8601 UTC matching the shell output:
//   date -u +%Y-%m-%dT%H:%M:%SZ
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// ComputeExpiry calculates the expiration timestamp from a creation time and TTL in days.
func ComputeExpiry(createdAt time.Time, ttlDays int) time.Time {
	return createdAt.UTC().AddDate(0, 0, ttlDays)
}
