package models

import "time"

// Keyword lookup outcome constants
const (
	OutcomeResolved = "resolved"
	OutcomeFallback = "fallback"
	OutcomeNotFound = "not_found"
)

// KeywordLookup represents a per-keyword hit count by outcome.
type KeywordLookup struct {
	Keyword    string
	Outcome    string
	Count      int64
	LastSeenAt time.Time
}
