package timeutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseRelativeOrAbsolute parses a time string that is either:
//   - A relative duration like "15m", "1h", "24h", "7d" (interpreted as that much time ago from now)
//   - An absolute ISO 8601 timestamp like "2024-01-01T00:00:00Z"
//
// The now parameter allows injecting time for testing.
func ParseRelativeOrAbsolute(s string, now time.Time) (time.Time, error) {
	if s == "" || s == "now" {
		return now, nil
	}

	// Try relative duration first
	if d, err := parseRelativeDuration(s); err == nil {
		return now.Add(-d), nil
	}

	// Try ISO 8601 formats
	for _, layout := range []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse time %q: use a relative duration (e.g. 15m, 1h, 7d) or ISO 8601 timestamp", s)
}

// parseRelativeDuration parses strings like "15m", "1h", "24h", "7d", "2w".
func parseRelativeDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration %q", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: %w", s, err)
	}

	switch unit {
	case 's':
		return time.Duration(num * float64(time.Second)), nil
	case 'm':
		return time.Duration(num * float64(time.Minute)), nil
	case 'h':
		return time.Duration(num * float64(time.Hour)), nil
	case 'd':
		return time.Duration(num * 24 * float64(time.Hour)), nil
	case 'w':
		return time.Duration(num * 7 * 24 * float64(time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown duration unit %q in %q (use s, m, h, d, or w)", string(unit), s)
	}
}
